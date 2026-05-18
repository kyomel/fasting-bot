package app

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/sync/errgroup"

	"github.com/kyomel/fasting-bot/internal/bot"
	"github.com/kyomel/fasting-bot/internal/config"
	"github.com/kyomel/fasting-bot/internal/database"
	httpserver "github.com/kyomel/fasting-bot/internal/http"
)

type App struct {
	cfg          config.Config
	db           *database.DB
	http         *fiber.App
	bot          *bot.Runner
	shuttingDown atomic.Bool
	logger       *slog.Logger
}

func New(ctx context.Context, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = slog.Default()
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	db, err := database.Open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	runner := bot.NewRunner(cfg, logger)
	application := &App{
		cfg:    cfg,
		db:     db,
		bot:    runner,
		logger: logger,
	}

	application.http = httpserver.New(cfg, db, &application.shuttingDown, logger)

	return application, nil
}

func (a *App) Run(ctx context.Context) error {
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		a.logger.Info("starting bot runner", "interval", a.cfg.BotTickInterval.String())
		return a.bot.Run(groupCtx)
	})

	group.Go(func() error {
		a.logger.Info("starting http server", "addr", a.cfg.HTTPAddr)
		if err := a.http.Listen(a.cfg.HTTPAddr); err != nil {
			if groupCtx.Err() != nil {
				return nil
			}
			return err
		}

		return nil
	})

	group.Go(func() error {
		<-groupCtx.Done()
		a.shuttingDown.Store(true)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()

		a.logger.Info("shutting down http server")
		if err := a.http.ShutdownWithContext(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}

		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	return nil
}

func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Warn("database close failed", "error", err)
		}
	}
}
