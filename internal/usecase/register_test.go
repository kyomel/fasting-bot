package usecase

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

// TestRegisterUserAPIHashesPasswordAndRejectsDuplicates verifies the API
// registration path: bcrypt hash stored (never plaintext), duplicates map to
// ErrConflict, and short passwords fail validation.
func TestRegisterUserAPIHashesPasswordAndRejectsDuplicates(t *testing.T) {
	repo := newRegisterFakeRepo()
	uc := NewFastingUsecase(repo, &motivationScheduleRepo{}, &motivationNotificationRepo{}, &motivationBadgeRepo{})

	got, err := uc.RegisterUserAPI(RegisterInput{
		Username: "kyomel",
		Password: "rahasia-kuat-123",
		Phone:    "08123456789",
		Email:    "kyomel@example.com",
		Name:     "Kyo",
	})
	if err != nil {
		t.Fatalf("RegisterUserAPI() error = %v", err)
	}
	if got.ID == "" || got.Phone != "+628123456789" {
		t.Fatalf("RegisterUserAPI() = %#v, want id set and normalized phone", got)
	}

	stored, err := repo.FindByUsername("kyomel")
	if err != nil {
		t.Fatalf("FindByUsername() error = %v", err)
	}
	if stored.PasswordHash == "rahasia-kuat-123" || stored.PasswordHash == "" {
		t.Fatal("PasswordHash must be a bcrypt hash, never plaintext or empty")
	}
	if !strings.HasPrefix(stored.PasswordHash, "$2a$") {
		t.Fatalf("PasswordHash = %q, want bcrypt hash", stored.PasswordHash)
	}
	if stored.CreatedAt.IsZero() || stored.UpdatedAt.IsZero() {
		t.Fatalf("timestamps not populated: %#v", stored)
	}
	if err := CheckPassword(stored.PasswordHash, "rahasia-kuat-123"); err != nil {
		t.Fatalf("CheckPassword() error = %v", err)
	}

	if _, err := uc.RegisterUserAPI(RegisterInput{Username: "kyomel", Password: "lain-lain-123", Phone: "+628999888777"}); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("duplicate username err = %v, want ErrConflict", err)
	}
	if _, err := uc.RegisterUserAPI(RegisterInput{Username: "kyomel2", Password: "lain-lain-123", Phone: "+628123456789"}); !errors.Is(err, repository.ErrConflict) {
		t.Fatalf("duplicate phone err = %v, want ErrConflict", err)
	}
	if _, err := uc.RegisterUserAPI(RegisterInput{Username: "pendek", Password: "short", Phone: "+628111222333"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("short password err = %v, want ErrValidation", err)
	}
}

type registerFakeRepo struct {
	mu    sync.Mutex
	users map[string]*domain.User
	seq   int
}

func newRegisterFakeRepo() *registerFakeRepo {
	return &registerFakeRepo{users: map[string]*domain.User{}}
}

func (r *registerFakeRepo) Create(user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if existing.Username == user.Username || existing.Phone == user.Phone {
			return repository.ErrConflict
		}
		if user.Email != "" && existing.Email == user.Email {
			return repository.ErrConflict
		}
	}
	r.seq++
	user.ID = domain.ID("fake-" + string(rune('0'+r.seq)))
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	r.users[string(user.ID)] = user
	return nil
}

func (r *registerFakeRepo) UpdateName(userID domain.ID, name string) error { return nil }

func (r *registerFakeRepo) FindByPhone(phone string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Phone == phone {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *registerFakeRepo) FindByUsername(username string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *registerFakeRepo) FindByEmail(email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Email != "" && u.Email == email {
			return u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (r *registerFakeRepo) FindByID(id domain.ID) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.users[string(id)]; ok {
		return u, nil
	}
	return nil, repository.ErrNotFound
}
