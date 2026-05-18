package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/kyomel/fasting-bot/internal/config"
	"github.com/kyomel/fasting-bot/internal/database"
)

type shutdownState interface {
	Load() bool
}

type server struct {
	cfg          config.Config
	db           *database.DB
	shuttingDown shutdownState
	logger       *slog.Logger
}

func New(cfg config.Config, db *database.DB, shuttingDown shutdownState, appLogger *slog.Logger) *fiber.App {
	if appLogger == nil {
		appLogger = slog.Default()
	}

	server := &server{cfg: cfg, db: db, shuttingDown: shuttingDown, logger: appLogger}
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: errorHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	})

	app.Use(recover.New())
	app.Use(logger.New())

	app.Get("/healthz", server.health)
	app.Get("/readyz", server.ready)

	api := app.Group("/api/v1")
	api.Get("/", server.apiIndex)

	return app
}

func (s *server) health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"service": s.cfg.AppName,
		"env":     s.cfg.AppEnv,
		"status":  "ok",
	})
}

func (s *server) ready(c fiber.Ctx) error {
	if s.shuttingDown != nil && s.shuttingDown.Load() {
		return fiber.NewError(fiber.StatusServiceUnavailable, "application is shutting down")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		s.logger.Warn("readiness check failed", "error", err)
		return fiber.NewError(fiber.StatusServiceUnavailable, "database is not ready")
	}

	return c.JSON(fiber.Map{
		"service":  s.cfg.AppName,
		"database": "ok",
		"status":   "ready",
	})
}

func (s *server) apiIndex(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "fasting API routes will be added in the next plan",
		"status":  "ok",
	})
}

func errorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "internal server error"

	if fiberErr, ok := err.(*fiber.Error); ok {
		code = fiberErr.Code
		message = fiberErr.Message
	}

	return c.Status(code).JSON(fiber.Map{"error": message})
}
