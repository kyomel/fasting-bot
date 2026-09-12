package persistence

import (
	"database/sql"
	"errors"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

type AuthSessionRepositoryPostgres struct {
	db             *sql.DB
	createStmt     *sql.Stmt
	findActiveStmt *sql.Stmt
	revokeStmt     *sql.Stmt
}

func NewAuthSessionRepositoryPostgres(db *sql.DB) repository.AuthSessionRepository {
	r := &AuthSessionRepositoryPostgres{db: db}

	r.createStmt, _ = db.Prepare("INSERT INTO auth_sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3) RETURNING id, created_at")
	r.findActiveStmt, _ = db.Prepare("SELECT id, user_id, token_hash, created_at, expires_at, revoked_at FROM auth_sessions WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2")
	r.revokeStmt, _ = db.Prepare("UPDATE auth_sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL")

	return r
}

func (r *AuthSessionRepositoryPostgres) Create(session *domain.AuthSession) error {
	return r.createStmt.QueryRow(string(session.UserID), session.TokenHash, session.ExpiresAt).Scan(&session.ID, &session.CreatedAt)
}

func (r *AuthSessionRepositoryPostgres) FindActiveByTokenHash(tokenHash string, now time.Time) (*domain.AuthSession, error) {
	var session domain.AuthSession
	var revokedAt sql.NullTime
	err := r.findActiveStmt.QueryRow(tokenHash, now).Scan(&session.ID, &session.UserID, &session.TokenHash, &session.CreatedAt, &session.ExpiresAt, &revokedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	if revokedAt.Valid {
		session.RevokedAt = &revokedAt.Time
	}
	return &session, nil
}

// RevokeByTokenHash is idempotent: revoking an unknown or already-revoked
// token is a no-op so logout never fails on a stale token.
func (r *AuthSessionRepositoryPostgres) RevokeByTokenHash(tokenHash string) error {
	_, err := r.revokeStmt.Exec(tokenHash)
	return err
}
