package usecase

import (
	"errors"
	"sync"
	"testing"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

// TestLoginValidation covers the boundary checks: a short password fails
// validation before any credential lookup happens.
func TestLoginValidation(t *testing.T) {
	uc := NewFastingUsecase(newRegisterFakeRepo(), &motivationScheduleRepo{}, &motivationNotificationRepo{}, &motivationBadgeRepo{}, newFakeAuthSessionRepo())

	if _, err := uc.Login(LoginInput{Username: "kyomel", Password: "short"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("short password err = %v, want ErrValidation", err)
	}
	if _, err := uc.Login(LoginInput{Username: "", Password: "password-panjang"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty username err = %v, want ErrValidation", err)
	}
}

// TestLoginSuccess verifies the happy path: opaque token returned, only its
// SHA-256 hash persisted, and the token authenticates back to the user.
func TestLoginSuccess(t *testing.T) {
	userRepo := newRegisterFakeRepo()
	sessionRepo := newFakeAuthSessionRepo()
	uc := NewFastingUsecase(userRepo, &motivationScheduleRepo{}, &motivationNotificationRepo{}, &motivationBadgeRepo{}, sessionRepo)

	if _, err := uc.RegisterUserAPI(RegisterInput{Username: "kyomel", Password: "rahasia-kuat-123", Phone: "08123456789"}); err != nil {
		t.Fatalf("RegisterUserAPI() error = %v", err)
	}

	got, err := uc.Login(LoginInput{Username: "kyomel", Password: "rahasia-kuat-123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if got.AccessToken == "" || got.TokenType != "Bearer" {
		t.Fatalf("Login() = %#v, want bearer access token", got)
	}
	if time.Until(got.ExpiresAt) <= 23*time.Hour || time.Until(got.ExpiresAt) > 24*time.Hour {
		t.Fatalf("ExpiresAt = %v, want ~24h TTL", got.ExpiresAt)
	}
	if got.User.Username != "kyomel" || got.User.Role != domain.RoleUser {
		t.Fatalf("Login() user = %#v, want username + default role", got.User)
	}

	stored := sessionRepo.findByHash(hashAccessToken(got.AccessToken))
	if stored == nil {
		t.Fatal("session not persisted")
	}
	if stored.TokenHash == got.AccessToken {
		t.Fatal("raw token must never be persisted; only its SHA-256 hash")
	}

	user, err := uc.Authenticate(got.AccessToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if user.Username != "kyomel" {
		t.Fatalf("Authenticate() user = %#v", user)
	}
}

// TestLoginGenericFailure asserts unknown username and wrong password are
// indistinguishable: same sentinel error, no account enumeration.
func TestLoginGenericFailure(t *testing.T) {
	userRepo := newRegisterFakeRepo()
	uc := NewFastingUsecase(userRepo, &motivationScheduleRepo{}, &motivationNotificationRepo{}, &motivationBadgeRepo{}, newFakeAuthSessionRepo())

	if _, err := uc.RegisterUserAPI(RegisterInput{Username: "kyomel", Password: "rahasia-kuat-123", Phone: "08123456789"}); err != nil {
		t.Fatalf("RegisterUserAPI() error = %v", err)
	}

	_, unknownErr := uc.Login(LoginInput{Username: "ghost", Password: "rahasia-kuat-123"})
	_, wrongPassErr := uc.Login(LoginInput{Username: "kyomel", Password: "salah-banget-1"})

	if !errors.Is(unknownErr, ErrInvalidCredentials) || !errors.Is(wrongPassErr, ErrInvalidCredentials) {
		t.Fatalf("unknown=%v wrong=%v, both must be ErrInvalidCredentials", unknownErr, wrongPassErr)
	}
	if unknownErr.Error() != wrongPassErr.Error() {
		t.Fatalf("error text differs: %q vs %q - leaks account existence", unknownErr, wrongPassErr)
	}
}

// TestLogoutRevokesSession covers revocation: after logout the same token
// is denied, and logout stays idempotent.
func TestLogoutRevokesSession(t *testing.T) {
	userRepo := newRegisterFakeRepo()
	sessionRepo := newFakeAuthSessionRepo()
	uc := NewFastingUsecase(userRepo, &motivationScheduleRepo{}, &motivationNotificationRepo{}, &motivationBadgeRepo{}, sessionRepo)

	if _, err := uc.RegisterUserAPI(RegisterInput{Username: "kyomel", Password: "rahasia-kuat-123", Phone: "08123456789"}); err != nil {
		t.Fatalf("RegisterUserAPI() error = %v", err)
	}
	login, err := uc.Login(LoginInput{Username: "kyomel", Password: "rahasia-kuat-123"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if err := uc.Logout(login.AccessToken); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := uc.Authenticate(login.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authenticate() after logout err = %v, want ErrUnauthorized", err)
	}
	if err := uc.Logout(login.AccessToken); err != nil {
		t.Fatalf("second Logout() = %v, want idempotent nil", err)
	}
}

// TestAuthenticateDeniesExpiredAndUnknown covers the denial paths:
// expired sessions, unknown tokens, and empty tokens all map to
// ErrUnauthorized.
func TestAuthenticateDeniesExpiredAndUnknown(t *testing.T) {
	userRepo := newRegisterFakeRepo()
	sessionRepo := newFakeAuthSessionRepo()
	uc := NewFastingUsecase(userRepo, &motivationScheduleRepo{}, &motivationNotificationRepo{}, &motivationBadgeRepo{}, sessionRepo)

	if _, err := uc.RegisterUserAPI(RegisterInput{Username: "kyomel", Password: "rahasia-kuat-123", Phone: "08123456789"}); err != nil {
		t.Fatalf("RegisterUserAPI() error = %v", err)
	}
	user, err := userRepo.FindByUsername("kyomel")
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}

	// Expired session: inserted directly with a past expiry.
	expired := &domain.AuthSession{
		UserID:    user.ID,
		TokenHash: hashAccessToken("expired-token"),
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	if err := sessionRepo.Create(expired); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	for name, token := range map[string]string{
		"expired": "expired-token",
		"unknown": "token-yang-tidak-ada",
		"empty":   "",
	} {
		if _, err := uc.Authenticate(token); !errors.Is(err, ErrUnauthorized) {
			t.Fatalf("%s: Authenticate() err = %v, want ErrUnauthorized", name, err)
		}
	}
}

type fakeAuthSessionRepo struct {
	mu       sync.Mutex
	sessions map[string]*domain.AuthSession
	seq      int
}

func newFakeAuthSessionRepo() *fakeAuthSessionRepo {
	return &fakeAuthSessionRepo{sessions: map[string]*domain.AuthSession{}}
}

func (r *fakeAuthSessionRepo) Create(session *domain.AuthSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	session.ID = domain.ID("session-" + string(rune('0'+r.seq)))
	session.CreatedAt = time.Now()
	r.sessions[session.TokenHash] = session
	return nil
}

func (r *fakeAuthSessionRepo) FindActiveByTokenHash(tokenHash string, now time.Time) (*domain.AuthSession, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[tokenHash]
	if !ok || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return nil, repository.ErrNotFound
	}
	return session, nil
}

func (r *fakeAuthSessionRepo) RevokeByTokenHash(tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session, ok := r.sessions[tokenHash]; ok && session.RevokedAt == nil {
		now := time.Now()
		session.RevokedAt = &now
	}
	return nil
}

func (r *fakeAuthSessionRepo) findByHash(tokenHash string) *domain.AuthSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sessions[tokenHash]
}
