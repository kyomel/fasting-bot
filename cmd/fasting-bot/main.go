package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kyomel/fasting-bot/internal/app"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, logger)
	if err != nil {
		logger.Error("startup failed", "error", err)
		return 1
	}
	defer application.Close()

	if err := application.Run(ctx); err != nil {
		logger.Error("application stopped with error", "error", err)
		return 1
	}

	return 0
}
