package domain

import "time"

// Role is a branded access-control variant on User. The database enforces
// the same allow-list via a CHECK constraint (migration 00010).
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// AuthSession is one opaque bearer-token session. Only the SHA-256 hash of
// the token is ever persisted; the raw token exists only in flight.
type AuthSession struct {
	ID        ID
	UserID    ID
	TokenHash string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}
