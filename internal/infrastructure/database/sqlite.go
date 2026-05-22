package database

import (
	"database/sql"
	"fmt"

	"fasting-bot/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	Conn *sql.DB
}

func New() (*DB, error) {
	conn, err := sql.Open("sqlite3", config.DatabasePath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pool — SQLite supports max 1 writer but multiple readers in WAL mode
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := migrate(conn); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return &DB{Conn: conn}, nil
}

func (d *DB) Close() error {
	return d.Conn.Close()
}

func migrate(conn *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			phone TEXT NOT NULL UNIQUE,
			name TEXT,
			jid TEXT UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS fasting_schedules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			fast_start TEXT NOT NULL,
			fast_end TEXT NOT NULL,
			fasting_type_name TEXT DEFAULT '',
			is_active BOOLEAN DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS notification_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			notification_type TEXT NOT NULL,
			sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS fasting_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			schedule_id INTEGER NOT NULL,
			fasting_type_name TEXT DEFAULT '',
			fast_start TEXT NOT NULL,
			planned_fast_end TEXT NOT NULL,
			opened_at TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			completed_date TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS user_fasting_stats (
			user_id INTEGER PRIMARY KEY,
			total_sessions INTEGER NOT NULL DEFAULT 0,
			total_minutes INTEGER NOT NULL DEFAULT 0,
			current_streak_days INTEGER NOT NULL DEFAULT 0,
			longest_streak_days INTEGER NOT NULL DEFAULT 0,
			last_completed_date TEXT NOT NULL DEFAULT '',
			last_opened_at TEXT NOT NULL DEFAULT '',
			last_duration_minutes INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS groups_ (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			jid TEXT NOT NULL UNIQUE,
			name TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_users_phone ON users(phone);`,
		`CREATE INDEX IF NOT EXISTS idx_fasting_schedules_user_active ON fasting_schedules(user_id, is_active);`,
		`CREATE INDEX IF NOT EXISTS idx_fasting_schedules_active_start ON fasting_schedules(is_active, fast_start);`,
		`CREATE INDEX IF NOT EXISTS idx_fasting_schedules_active_end ON fasting_schedules(is_active, fast_end);`,
		`CREATE INDEX IF NOT EXISTS idx_fasting_schedules_inactive_created ON fasting_schedules(is_active, created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_notification_logs_user ON notification_logs(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_notification_logs_user_type_sent ON notification_logs(user_id, notification_type, sent_at);`,
		`CREATE INDEX IF NOT EXISTS idx_fasting_records_user_date ON fasting_records(user_id, completed_date);`,
		`CREATE INDEX IF NOT EXISTS idx_fasting_records_total ON fasting_records(duration_minutes);`,
		`CREATE INDEX IF NOT EXISTS idx_fasting_records_created_at ON fasting_records(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_user_fasting_stats_total ON user_fasting_stats(total_minutes DESC);`,
	}

	for _, query := range queries {
		if _, err := conn.Exec(query); err != nil {
			return err
		}
	}

	if err := addColumnIfMissing(conn, "fasting_schedules", "fasting_type_name", "TEXT DEFAULT ''"); err != nil {
		return err
	}

	return nil
}

func addColumnIfMissing(conn *sql.DB, table, column, definition string) error {
	rows, err := conn.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue interface{}
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = conn.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}
