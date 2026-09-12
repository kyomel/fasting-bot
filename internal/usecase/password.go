package usecase

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost balances login latency against offline brute-force resistance.
// DefaultCost (10) is the stdlib recommendation; raise only with benchmarks.
const bcryptCost = bcrypt.DefaultCost

// HashPassword hashes a plaintext password with bcrypt, the standard
// adaptive password-hashing algorithm for Go (golang.org/x/crypto/bcrypt).
// Adaptive cost means verification stays slow for attackers while remaining
// fast enough for logins; Argon2id is stronger per second but needs careful
// memory tuning, so bcrypt is the boring, correct default here.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a bcrypt hash against a plaintext candidate.
func CheckPassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	return nil
}
