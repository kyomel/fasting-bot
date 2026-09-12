package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

const (
	// accessTokenBytes is the entropy of the opaque bearer token: 256 bits
	// from crypto/rand, base64url-encoded for the Authorization header.
	accessTokenBytes = 32
	// sessionTTL is the fixed session lifetime. A constant is deliberate:
	// the deployment is a single service, so env-based TTL adds config
	// surface without an operational need.
	sessionTTL = 24 * time.Hour
)

// ErrInvalidCredentials is the single generic login failure. Unknown
// username and wrong password produce the same error so the API cannot be
// used to enumerate accounts.
var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrUnauthorized marks missing, unknown, expired, or revoked bearer tokens.
var ErrUnauthorized = errors.New("invalid or expired token")

// dummyPasswordHash runs one bcrypt verification on the unknown-username
// path so login latency does not reveal whether the account exists.
var dummyPasswordHash = sync.OnceValue(func() string {
	hash, _ := HashPassword("timing-equalizer")
	return hash
})

type LoginInput struct {
	Username string
	Password string
}

// LoginResult is the public login projection. AccessToken is the only place
// the raw token ever appears; the session store keeps its SHA-256 hash.
type LoginResult struct {
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type"`
	ExpiresAt   time.Time      `json:"expires_at"`
	User        RegisterResult `json:"user"`
}

func (u *fastingUsecase) Login(input LoginInput) (*LoginResult, error) {
	username := strings.TrimSpace(input.Username)
	password := input.Password

	if username == "" {
		return nil, fmt.Errorf("username is required: %w", ErrValidation)
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters: %w", ErrValidation)
	}

	user, err := u.userRepo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = CheckPassword(dummyPasswordHash(), password)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf(errCheckDataFormat, err)
	}
	if err := CheckPassword(user.PasswordHash, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := newAccessToken()
	if err != nil {
		return nil, err
	}
	session := &domain.AuthSession{
		UserID:    user.ID,
		TokenHash: hashAccessToken(token),
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	if err := u.authSessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("gagal membuat sesi: %w", err)
	}

	return &LoginResult{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   session.ExpiresAt,
		User:        publicUser(user),
	}, nil
}

// Authenticate resolves a raw bearer token to its user. Expired and revoked
// sessions are indistinguishable from unknown tokens: ErrUnauthorized.
func (u *fastingUsecase) Authenticate(token string) (*domain.User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}

	session, err := u.authSessionRepo.FindActiveByTokenHash(hashAccessToken(token), time.Now())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf(errCheckDataFormat, err)
	}

	user, err := u.userRepo.FindByID(session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUnauthorized
		}
		return nil, fmt.Errorf(errCheckDataFormat, err)
	}
	return user, nil
}

// Logout revokes the session behind the bearer token. It is idempotent:
// revoking an already-dead session succeeds silently.
func (u *fastingUsecase) Logout(token string) error {
	if token == "" {
		return ErrUnauthorized
	}
	if err := u.authSessionRepo.RevokeByTokenHash(hashAccessToken(token)); err != nil {
		return fmt.Errorf("gagal menutup sesi: %w", err)
	}
	return nil
}

func newAccessToken() (string, error) {
	buf := make([]byte, accessTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate access token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashAccessToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
