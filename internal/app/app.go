package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
	"cloudpan-sync-go/internal/task"
	webui "cloudpan-sync-go/web"
)

type App struct {
	cfg       Config
	logger    *slog.Logger
	store     *sqlitestore.Store
	providers *provider.Registry
	auth      *auth.Service
	tasks     *task.Service
	server    *http.Server
	webIndex  []byte
	webStatic http.Handler

	recoverBlockedTasksFunc func(context.Context, task.RecoverOptions) (task.RecoverResult, error)
}

func New(ctx context.Context, cfg Config) (*App, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	store, err := sqlitestore.New(ctx, cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite store: %w", err)
	}

	registry := provider.NewRegistry(provider.DefaultCatalog()...)
	authService := auth.NewService(store, registry)
	webIndex, err := webui.IndexHTML()
	if err != nil {
		return nil, fmt.Errorf("load web index: %w", err)
	}
	staticFS, err := webui.StaticFS()
	if err != nil {
		return nil, fmt.Errorf("load web assets: %w", err)
	}
	app := &App{
		cfg:       cfg,
		logger:    logger,
		store:     store,
		providers: registry,
		auth:      authService,
		tasks:     task.NewService(store, registry, authService),
		webIndex:  webIndex,
		webStatic: http.StripPrefix("/assets/", http.FileServer(http.FS(staticFS))),
	}

	app.server = &http.Server{
		Addr:              cfg.Addr,
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	if a.cfg.AutoRetryTick > 0 {
		go a.runAutoRetryScheduler(runCtx)
	}

	go func() {
		a.logger.Info("http server starting", "addr", a.cfg.Addr, "db_path", a.cfg.DBPath)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-runCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a.server.Shutdown(shutdownCtx)
		return a.store.Close()
	case err := <-errCh:
		closeErr := a.store.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
}

func (a *App) runAutoRetryScheduler(ctx context.Context) {
	a.runAutoRetryOnce(ctx, "startup")

	ticker := time.NewTicker(a.cfg.AutoRetryTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runAutoRetryOnce(ctx, "tick")
		}
	}
}

func (a *App) runAutoRetryOnce(ctx context.Context, reason string) {
	result, err := a.recoverBlockedTasks(ctx, task.RecoverOptions{Limit: a.cfg.AutoRetryBatchLimit})
	if err != nil {
		a.logger.Warn("auto retry recovery failed", "reason", reason, "error", err)
		return
	}
	if result.RecoveredCount > 0 {
		a.logger.Info(
			"auto retry recovered blocked tasks",
			"reason", reason,
			"count", result.RecoveredCount,
			"matched", result.MatchedCount,
			"skipped_by_limit", result.SkippedByLimit,
			"limit", result.Limit,
		)
	}
}

func (a *App) recoverBlockedTasks(ctx context.Context, opts task.RecoverOptions) (task.RecoverResult, error) {
	if a.recoverBlockedTasksFunc != nil {
		return a.recoverBlockedTasksFunc(ctx, opts)
	}
	return a.tasks.RecoverBlockedTasksWithOptions(ctx, opts)
}

func (a *App) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		a.logger.Info("request completed", "method", r.Method, "path", r.URL.Path, "elapsed_ms", time.Since(start).Milliseconds())
	})
}
