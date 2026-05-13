package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"zxmail/backend/internal/app"
	"zxmail/backend/internal/config"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
