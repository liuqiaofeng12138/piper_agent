package auth

import (
	"testing"

	"piper_agent/gateway/internal/config"
)

func TestLoginAndValidate(t *testing.T) {
	svc, err := New(config.Auth{
		Mode: "login",
		Users: []config.AuthUser{
			{Username: "admin", Password: "password"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tok, _, err := svc.Login("admin", "password")
	if err != nil {
		t.Fatal(err)
	}
	user, ok := svc.ValidateBearer("Bearer " + tok)
	if !ok || user != "admin" {
		t.Fatalf("validate: user=%q ok=%v", user, ok)
	}
	_, _, err = svc.Login("admin", "wrong")
	if err != ErrInvalidCredentials {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}
