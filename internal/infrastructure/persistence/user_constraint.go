package persistence

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"fasting-bot/internal/repository"
)

// mapUserConstraintError converts PostgreSQL unique-violation errors into
// repository.ErrConflict so callers stay driver-agnostic.
func mapUserConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return repository.ErrConflict
	}
	return err
}
