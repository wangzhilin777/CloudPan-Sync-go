package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"cloudpan-sync-go/internal/app"
	"cloudpan-sync-go/internal/desktop"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := app.MustLoadConfig()
	if err := desktop.Run(ctx, cfg); err != nil {
		log.Fatalf("启动桌面模式失败: %v", err)
	}
}
