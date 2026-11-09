package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"piper_agent/gateway/internal/auth"
	"piper_agent/gateway/internal/config"
	"piper_agent/gateway/internal/run"
	"piper_agent/gateway/internal/session"
)

func TestLoginAndProtectedAPI(t *testing.T) {
	authSvc, err := auth.New(config.Auth{
		Mode: "login",
		Users: []config.AuthUser{
			{Username: "admin", Password: "password"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	deps := &Deps{
		Store: session.NewMemoryStore(),
		Config: &config.Config{
			Classifier: config.ClassifierConfig{Mode: "rule"},
			Agents:     []config.AgentSpec{{ID: "general_chat", Enabled: true, Default: true}},
		},
		Runs: run.NewRegistry(),
		Auth: authSvc,
	}
	h := NewHandler(deps)

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "password"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &loginResp); err != nil || loginResp.Token == "" {
		t.Fatalf("login response: %v", err)
	}

	unauth := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	unauthRec := httptest.NewRecorder()
	h.ServeHTTP(unauthRec, unauth)
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth list status=%d", unauthRec.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	authReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	authRec := httptest.NewRecorder()
	h.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusOK {
		t.Fatalf("auth list status=%d body=%s", authRec.Code, authRec.Body.String())
	}
}
