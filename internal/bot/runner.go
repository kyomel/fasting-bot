package bot

import (
	"context"
	"log/slog"
	"time"

	"github.com/kyomel/fasting-bot/internal/config"
)

type Runner struct {
	cfg    config.Config
	logger *slog.Logger
}

func NewRunner(cfg config.Config, logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.Default()
	}

	return &Runner{cfg: cfg, logger: logger}
}

func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.cfg.BotTickInterval)
	defer ticker.Stop()

	r.logger.Info("bot runner ready")

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("bot runner stopped")
			return nil
		case tickedAt := <-ticker.C:
			r.handleTick(ctx, tickedAt)
		}
	}
}

func (r *Runner) handleTick(ctx context.Context, tickedAt time.Time) {
	if err := ctx.Err(); err != nil {
		return
	}

	// Placeholder for the fasting reminder/check loop. Keep it small and
	// context-aware so later bot integrations can stop cleanly on shutdown.
	r.logger.Debug("bot tick", "at", tickedAt.UTC().Format(time.RFC3339))
}
