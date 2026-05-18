package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName         string
	AppEnv          string
	HTTPAddr        string
	SQLiteDSN       string
	BotTickInterval time.Duration
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	_ = godotenv.Load()

	httpAddr := stringFromEnv("HTTP_ADDR", ":3000")
	if httpAddr == "" {
		return Config{}, fmt.Errorf("HTTP_ADDR must not be empty")
	}

	botTickInterval, err := durationFromEnv("BOT_TICK_INTERVAL", time.Minute)
	if err != nil {
		return Config{}, err
	}
	if botTickInterval <= 0 {
		return Config{}, fmt.Errorf("BOT_TICK_INTERVAL must be greater than 0")
	}

	shutdownTimeout, err := durationFromEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	if shutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("SHUTDOWN_TIMEOUT must be greater than 0")
	}

	return Config{
		AppName:         stringFromEnv("APP_NAME", "fasting-bot"),
		AppEnv:          stringFromEnv("APP_ENV", "development"),
		HTTPAddr:        httpAddr,
		SQLiteDSN:       stringFromEnv("SQLITE_DSN", "file:fasting.db?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on"),
		BotTickInterval: botTickInterval,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func stringFromEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func durationFromEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return duration, nil
}
