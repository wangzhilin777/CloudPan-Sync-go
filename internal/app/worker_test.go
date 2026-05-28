package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"cloudpan-sync-go/internal/task"
)

func TestAutoRetrySchedulerRunsRecoveryOnStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := make(chan struct{}, 1)
	app := &App{
		cfg: Config{
			AutoRetryTick:       time.Hour,
			AutoRetryBatchLimit: 2,
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		recoverBlockedTasksFunc: func(_ context.Context, opts task.RecoverOptions) (task.RecoverResult, error) {
			if opts.Limit != 2 {
				t.Fatalf("expected auto retry batch limit 2, got %d", opts.Limit)
			}
			called <- struct{}{}
			return task.RecoverResult{RecoveredCount: 1, Limit: opts.Limit}, nil
		},
	}

	go app.runAutoRetryScheduler(ctx)

	select {
	case <-called:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected auto retry scheduler to recover once on startup")
	}
}
