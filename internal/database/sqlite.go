package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kyomel/fasting-bot/internal/config"
)

type DB struct {
	conn *sql.DB
}

func Open(ctx context.Context, cfg config.Config) (*DB, error) {
	conn, err := sql.Open("sqlite3", cfg.SQLiteDSN)
	if err != nil {
		return nil, fmt.Errorf("open sqlite connection: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	db := &DB{conn: conn}
	if err := db.ping(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	if err := db.migrate(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.ping(ctx)
}

func (db *DB) ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := db.conn.PingContext(pingCtx); err != nil {
		return fmt.Errorf("ping sqlite: %w", err)
	}

	return nil
}

func (db *DB) migrate(ctx context.Context) error {
	_, err := db.conn.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS app_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT INTO app_metadata (key, value)
VALUES ('schema_version', '1')
ON CONFLICT(key) DO NOTHING;
`)
	if err != nil {
		return fmt.Errorf("run sqlite migrations: %w", err)
	}

	return nil
}
