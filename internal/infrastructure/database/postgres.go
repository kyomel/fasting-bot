package database

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"fasting-bot/internal/config"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

type DB struct {
	Conn *sql.DB
}

func (d *DB) Close() error {
	return d.Conn.Close()
}

// openPoolerSafe opens a database/sql handle over pgx with
// QueryExecModeSimpleProtocol. Production PostgreSQL sits behind a
// pooler (pgbouncer-style, port 6432): server-side named prepared statements
// do not survive across pooled sessions, so the default extended-protocol
// mode fails with 42P05 "prepared statement already exists" (pgx names them
// stmtcache_*). The simple protocol sends the query text inline on every
// execution with no server-side named statements at all, so it is
// pooler-safe; it is also one of the modes pgx itself exercises against
// pgbouncer (TestPgbouncerSimpleProtocol).
func openPoolerSafe(dsn string) (*sql.DB, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return stdlib.OpenDB(*cfg), nil
}

// New opens the PostgreSQL database using DB_CONNECTION and applies all
// goose migrations on startup. DB_CONNECTION is required: SQLite support
// was removed when PostgreSQL became the primary database.
func New() (*DB, error) {
	if strings.TrimSpace(config.DBConnection) == "" {
		return nil, fmt.Errorf("DB_CONNECTION is required (postgresql://...); SQLite support was removed")
	}
	return NewPostgres()
}

// NewPostgres opens a PostgreSQL connection using DB_CONNECTION and applies
// all goose migrations on startup.
func NewPostgres() (*DB, error) {
	conn, err := openPoolerSafe(config.DBConnection)
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
