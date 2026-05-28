package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestAutoRetrySchedulerRunsRecoveryOnStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := make(chan struct{}, 1)
	app := &App{
		cfg: Config{
			AutoRetryTick: time.Hour,
		},
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		recoverBlockedTasksFunc: func(context.Context) (int, error) {
			called <- struct{}{}
			return 1, nil
		},
	}

	go app.runAutoRetryScheduler(ctx)

	select {
	case <-called:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected auto retry scheduler to recover once on startup")
	}
}
