package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/usecase"
)

// stubUsecase embeds the FastingUsecase interface so only the methods a
// test exercises need a body; the rest panic if accidentally called.
type stubUsecase struct {
	usecase.FastingUsecase
	loginResult *usecase.LoginResult
	loginErr    error
	authUser    *domain.User
	authErr     error
	logoutErr   error
}

func (s *stubUsecase) Login(input usecase.LoginInput) (*usecase.LoginResult, error) {
	return s.loginResult, s.loginErr
}

func (s *stubUsecase) Authenticate(token string) (*domain.User, error) {
	return s.authUser, s.authErr
}

func (s *stubUsecase) Logout(token string) error {
	return s.logoutErr
}

func doRequest(t *testing.T, handler http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestHandleLoginStatusMapping covers the login contract: bad JSON -> 400,
// validation -> 400, bad credentials -> one generic 401, success -> 200.
func TestHandleLoginStatusMapping(t *testing.T) {
	t.Run("invalid JSON", func(t *testing.T) {
		srv := NewServer(&stubUsecase{})
		rec := doRequest(t, srv.Handler(), http.MethodPost, "/api/v1/auth/login", "{not-json", "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("short password validation", func(t *testing.T) {
		srv := NewServer(&stubUsecase{loginErr: usecase.ErrValidation})
		rec := doRequest(t, srv.Handler(), http.MethodPost, "/api/v1/auth/login", `{"username":"kyomel","password":"short"}`, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("generic credentials failure", func(t *testing.T) {
		srv := NewServer(&stubUsecase{loginErr: usecase.ErrInvalidCredentials})
		rec := doRequest(t, srv.Handler(), http.MethodPost, "/api/v1/auth/login", `{"username":"ghost","password":"rahasia-kuat-123"}`, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
		var body errorResponse
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode error = %v", err)
		}
		if body.Error != "invalid username or password" {
			t.Fatalf("error = %q, want generic credential message", body.Error)
		}
	})

	t.Run("success", func(t *testing.T) {
		srv := NewServer(&stubUsecase{loginResult: &usecase.LoginResult{
			AccessToken: "token-rahasia",
			TokenType:   "Bearer",
			ExpiresAt:   time.Now().Add(24 * time.Hour),
			User:        usecase.RegisterResult{Username: "kyomel", Role: domain.RoleUser},
		}})
		rec := doRequest(t, srv.Handler(), http.MethodPost, "/api/v1/auth/login", `{"username":"kyomel","password":"rahasia-kuat-123"}`, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body usecase.LoginResult
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode error = %v", err)
		}
		if body.AccessToken != "token-rahasia" || body.TokenType != "Bearer" {
			t.Fatalf("body = %#v", body)
		}
	})
}

// TestProtectedEndpoints covers the middleware contract: missing token ->
// 401, dead token -> 401, valid token -> handler runs.
func TestProtectedEndpoints(t *testing.T) {
	user := &domain.User{ID: "u1", Username: "kyomel", Phone: "+628123456789", Role: domain.RoleUser}

	t.Run("me without token", func(t *testing.T) {
		srv := NewServer(&stubUsecase{})
		rec := doRequest(t, srv.Handler(), http.MethodGet, "/api/v1/users/me", "", "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("me with dead token", func(t *testing.T) {
		srv := NewServer(&stubUsecase{authErr: usecase.ErrUnauthorized})
		rec := doRequest(t, srv.Handler(), http.MethodGet, "/api/v1/users/me", "", "token-mati")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("me with valid token", func(t *testing.T) {
		srv := NewServer(&stubUsecase{authUser: user})
		rec := doRequest(t, srv.Handler(), http.MethodGet, "/api/v1/users/me", "", "token-hidup")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var body usecase.RegisterResult
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode error = %v", err)
		}
		if body.Username != "kyomel" || body.Role != domain.RoleUser {
			t.Fatalf("body = %#v", body)
		}
	})

	t.Run("logout then token dead", func(t *testing.T) {
		uc := &stubUsecase{authUser: user}
		srv := NewServer(uc)
		rec := doRequest(t, srv.Handler(), http.MethodPost, "/api/v1/auth/logout", "", "token-hidup")
		if rec.Code != http.StatusOK {
			t.Fatalf("logout status = %d, want 200", rec.Code)
		}

		uc.authUser, uc.authErr = nil, usecase.ErrUnauthorized
		rec = doRequest(t, srv.Handler(), http.MethodGet, "/api/v1/users/me", "", "token-hidup")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("me after logout status = %d, want 401", rec.Code)
		}
	})
}

// TestRequireRoleDeniesNonAdmin wires an admin-gated route the same way a
// future admin endpoint would and asserts non-admin tokens get 403.
func TestRequireRoleDeniesNonAdmin(t *testing.T) {
	ok := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
	srv := NewServer(&stubUsecase{authUser: &domain.User{ID: "u1", Username: "kyomel", Role: domain.RoleUser}})
	srv.mux.HandleFunc("GET /api/v1/admin/ping", srv.requireAuth(srv.requireRole(domain.RoleAdmin, ok)))

	rec := doRequest(t, srv.Handler(), http.MethodGet, "/api/v1/admin/ping", "", "token-user-biasa")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}

	srv = NewServer(&stubUsecase{authUser: &domain.User{ID: "u2", Username: "bos", Role: domain.RoleAdmin}})
	srv.mux.HandleFunc("GET /api/v1/admin/ping", srv.requireAuth(srv.requireRole(domain.RoleAdmin, ok)))
	rec = doRequest(t, srv.Handler(), http.MethodGet, "/api/v1/admin/ping", "", "token-admin")
	if rec.Code != http.StatusOK {
		t.Fatalf("admin status = %d, want 200", rec.Code)
	}
}
