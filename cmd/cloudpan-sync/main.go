package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"cloudpan-sync-go/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := app.MustLoadConfig()
	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("run app: %v", err)
	}
}
