package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"fasting-bot/internal/domain"
	"fasting-bot/internal/repository"
	"fasting-bot/internal/usecase"
)

// Server exposes the public REST API. It depends only on the usecase layer,
// keeping delivery -> usecase -> repository direction intact.
type Server struct {
	usecase usecase.FastingUsecase
	mux     *http.ServeMux
}

// NewServer wires the route table. Registration and /healthz stay public;
// everything else sits behind bearer auth - deny by default.
func NewServer(uc usecase.FastingUsecase) *Server {
	s := &Server{usecase: uc, mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /api/v1/users/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/v1/auth/logout", s.requireAuth(s.handleLogout))
	s.mux.HandleFunc("GET /api/v1/users/me", s.requireAuth(s.handleMe))
	s.mux.HandleFunc("GET /healthz", s.handleHealth)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// contextKey scopes request-context values to this package.
type contextKey string

const userContextKey contextKey = "auth-user"

// bearerToken extracts the opaque token from "Authorization: Bearer <token>".
func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

// requireAuth authenticates the bearer token and injects the user into the
// request context. Missing or dead tokens get one generic 401.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := bearerToken(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "missing bearer token"})
			return
		}

		user, err := s.usecase.Authenticate(token)
		if err != nil {
			if !errors.Is(err, usecase.ErrUnauthorized) {
				log.Printf("[ERROR] authenticate failed: %v", err)
				writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "gagal memeriksa sesi"})
				return
			}
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid or expired token"})
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	}
}

// requireRole gates a route on the authenticated user's role. It composes
// after requireAuth; no admin route exists yet, but future protected routes
// get admin-only access for free:
// mux.HandleFunc("GET /api/v1/admin/x", s.requireAuth(s.requireRole(domain.RoleAdmin, h)))
func (s *Server) requireRole(role domain.Role, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value(userContextKey).(*domain.User)
		if !ok || user == nil {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid or expired token"})
			return
		}
		if user.Role != role {
			writeJSON(w, http.StatusForbidden, errorResponse{Error: "insufficient permissions"})
			return
		}
		next(w, r)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleRegister decodes the JSON body, delegates validation + hashing to the
// usecase, and maps domain errors to status codes: 400 validation, 409
// duplicate, 500 anything else. The password hash never leaves the server.
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	result, err := s.usecase.RegisterUserAPI(usecase.RegisterInput{
		Username: req.Username,
		Password: req.Password,
		Phone:    req.Phone,
		Email:    req.Email,
		Name:     req.Name,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrValidation):
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: trimValidationSuffix(err)})
		case errors.Is(err, repository.ErrConflict):
			writeJSON(w, http.StatusConflict, errorResponse{Error: "username, email, atau nomor sudah terdaftar"})
		default:
			log.Printf("[ERROR] register failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "gagal mendaftar, coba lagi nanti"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// handleLogin decodes credentials and delegates to usecase.Login. Failures
// are deliberately generic: validation -> 400, bad credentials -> 401 with
// no hint about which part failed.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	result, err := s.usecase.Login(usecase.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrValidation):
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: trimValidationSuffix(err)})
		case errors.Is(err, usecase.ErrInvalidCredentials):
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid username or password"})
		default:
			log.Printf("[ERROR] login failed: %v", err)
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "gagal login, coba lagi nanti"})
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleLogout runs behind requireAuth, so the token is already known-good;
// revoking it is idempotent.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token, _ := bearerToken(r)
	if err := s.usecase.Logout(token); err != nil {
		if errors.Is(err, usecase.ErrUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid or expired token"})
			return
		}
		log.Printf("[ERROR] logout failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "gagal logout, coba lagi nanti"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

// handleMe returns the authenticated user's public projection.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(userContextKey).(*domain.User)
	if !ok || user == nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "invalid or expired token"})
		return
	}
	writeJSON(w, http.StatusOK, usecase.RegisterResult{
		ID:       user.ID,
		Username: user.Username,
		Phone:    user.Phone,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// trimValidationSuffix strips the ": validation failed" sentinel suffix so
// clients see only the human-readable reason.
func trimValidationSuffix(err error) string {
	msg := err.Error()
	if idx := strings.LastIndex(msg, ": "); idx >= 0 {
		return msg[:idx]
	}
	return msg
}
