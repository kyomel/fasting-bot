package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"fasting-bot/internal/repository"
	"fasting-bot/internal/usecase"
)

// Server exposes the public REST API. It depends only on the usecase layer,
// keeping delivery -> usecase -> repository direction intact.
type Server struct {
	usecase usecase.FastingUsecase
	mux     *http.ServeMux
}

// NewServer wires the route table. POST /api/v1/users/register is the only
// route for now; the mux leaves room for future login/profile endpoints.
func NewServer(uc usecase.FastingUsecase) *Server {
	s := &Server{usecase: uc, mux: http.NewServeMux()}
	s.mux.HandleFunc("POST /api/v1/users/register", s.handleRegister)
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

type errorResponse struct {
	Error string `json:"error"`
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
