package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"piper_agent/gateway/internal/auth"
)

func (s *Server) registerAuth(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", s.authLogin)
	mux.HandleFunc("GET /api/v1/auth/me", s.authMe)
}

func (s *Server) authLogin(w http.ResponseWriter, r *http.Request) {
	if s.deps.Auth == nil || !s.deps.Auth.LoginRequired() {
		http.Error(w, "login not enabled", http.StatusNotFound)
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if body.Username == "" || body.Password == "" {
		http.Error(w, "username and password required", http.StatusBadRequest)
		return
	}
	token, expiresAt, err := s.deps.Auth.Login(body.Username, body.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"username":   body.Username,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"username": user})
}

func requestUserID(r *http.Request) string {
	return auth.UserFromContext(r.Context())
}
