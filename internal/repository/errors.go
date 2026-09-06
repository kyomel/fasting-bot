package repository

import "errors"

// ErrNotFound is returned by repositories when a requested record does not exist.
// Inner layers can check for this error instead of leaking database/sql errors.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a create violates a uniqueness constraint
// (username, email, or phone already registered).
var ErrConflict = errors.New("already exists")
