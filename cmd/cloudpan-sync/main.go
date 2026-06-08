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
		log.Fatalf("启动应用失败: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("运行服务失败: %v", err)
	}
}
