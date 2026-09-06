package usecase

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
)

// RegisterInput is the trusted boundary shape for API user registration.
// Phone is the primary identity (matches the WhatsApp flow); username and
// email are optional account credentials for future login.
type RegisterInput struct {
	Username string
	Password string
	Phone    string
	Email    string
	Name     string
}

// RegisterResult is the public projection of a registered user.
// PasswordHash is never returned.
type RegisterResult struct {
	ID       domain.ID `json:"id"`
	Username string    `json:"username"`
	Phone    string    `json:"phone"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
}

// RegisterUserAPI registers a user with username/password credentials for
// API clients. It validates at the boundary, rejects duplicates with
// repository.ErrConflict, stores only the bcrypt hash, and returns a
// password-free projection.
func (u *fastingUsecase) RegisterUserAPI(input RegisterInput) (*RegisterResult, error) {
	username := strings.TrimSpace(input.Username)
	password := input.Password
	phone := normalizeRegisterPhone(input.Phone)
	email := strings.TrimSpace(input.Email)
	name := strings.TrimSpace(input.Name)

	if username == "" {
		return nil, fmt.Errorf("username is required: %w", ErrValidation)
	}
	if len(username) < 3 || len(username) > 32 {
		return nil, fmt.Errorf("username must be 3-32 characters: %w", ErrValidation)
	}
	if !isValidUsername(username) {
		return nil, fmt.Errorf("username may only contain letters, numbers, dots, and underscores: %w", ErrValidation)
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters: %w", ErrValidation)
	}
	if phone == "" {
		return nil, fmt.Errorf("phone is required: %w", ErrValidation)
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return nil, fmt.Errorf("email is invalid: %w", ErrValidation)
		}
	}

	if existing, err := u.userRepo.FindByUsername(username); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf(errCheckDataFormat, err)
	} else if existing != nil {
		return nil, fmt.Errorf("username %q: %w", username, repository.ErrConflict)
	}
	if existing, err := u.userRepo.FindByPhone(phone); err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf(errCheckDataFormat, err)
	} else if existing != nil {
		return nil, fmt.Errorf("phone %q: %w", phone, repository.ErrConflict)
	}
	if email != "" {
		if existing, err := u.userRepo.FindByEmail(email); err != nil && !errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf(errCheckDataFormat, err)
		} else if existing != nil {
			return nil, fmt.Errorf("email %q: %w", email, repository.ErrConflict)
		}
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: hash,
		Phone:        phone,
		Email:        email,
		Name:         name,
	}
	if err := u.userRepo.Create(user); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, fmt.Errorf("user already registered: %w", repository.ErrConflict)
		}
		return nil, fmt.Errorf("gagal mendaftar: %w", err)
	}

	return &RegisterResult{
		ID:       user.ID,
		Username: user.Username,
		Phone:    user.Phone,
		Email:    user.Email,
		Name:     user.Name,
	}, nil
}

// normalizeRegisterPhone reuses the WhatsApp phone contract: "+"-prefixed
// digits, leading 0 converted to the 62 country prefix. Kept local so the
// HTTP layer does not import the whatsapp delivery package.
func normalizeRegisterPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}
	if at := strings.IndexByte(phone, '@'); at >= 0 {
		phone = phone[:at]
	}

	var digits strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	normalized := digits.String()
	if strings.HasPrefix(normalized, "0") {
		normalized = "62" + strings.TrimLeft(normalized, "0")
	}
	if normalized == "" {
		return ""
	}
	return "+" + normalized
}

func isValidUsername(username string) bool {
	for _, r := range username {
		isLower := r >= 'a' && r <= 'z'
		isUpper := r >= 'A' && r <= 'Z'
		isDigit := r >= '0' && r <= '9'
		if !isLower && !isUpper && !isDigit && r != '.' && r != '_' {
			return false
		}
	}
	return true
}
