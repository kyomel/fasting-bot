package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"

	"fasting-bot/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

// NewPostgres opens a PostgreSQL connection using DB_CONNECTION and applies
// all goose migrations on startup.
func NewPostgres() (*DB, error) {
	conn, err := sql.Open("pgx", config.DBConnection)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	if err := MigratePostgres(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate postgres database: %w", err)
	}

	return &DB{Conn: conn}, nil
}

// MigratePostgres applies all embedded goose migrations to conn. Exported so
// integration tests can provision a schema identical to production.
func MigratePostgres(conn *sql.DB) error {
	goose.SetBaseFS(postgresMigrations)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(conn, "migrations/postgres"); err != nil {
		return fmt.Errorf("run goose migrations: %w", err)
	}

	version, err := goose.GetDBVersion(conn)
	if err != nil {
		return fmt.Errorf("read goose version: %w", err)
	}
	log.Printf("✅ PostgreSQL migrations applied (schema version %d)", version)
	return nil
}
