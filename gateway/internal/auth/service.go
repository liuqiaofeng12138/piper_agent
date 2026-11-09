package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"piper_agent/gateway/internal/config"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type Service struct {
	mode      string
	bearer    string
	secret    []byte
	ttl       time.Duration
	users     map[string][]byte // username -> bcrypt hash
}

func New(cfg config.Auth) (*Service, error) {
	s := &Service{
		mode:   strings.TrimSpace(cfg.Mode),
		bearer: strings.TrimSpace(cfg.Token),
		users:  make(map[string][]byte),
	}
	if s.mode == "" {
		s.mode = "none"
	}
	secret := strings.TrimSpace(cfg.JWTSecret)
	if secret == "" {
		secret = "piper-dev-jwt-secret"
	}
	s.secret = []byte(secret)

	ttlH := cfg.TokenTTLHours
	if ttlH <= 0 {
		ttlH = 72
	}
	s.ttl = time.Duration(ttlH) * time.Hour

	for _, u := range cfg.Users {
		name := strings.TrimSpace(u.Username)
		if name == "" || u.Password == "" {
			continue
		}
		hash, err := passwordHash(u.Password)
		if err != nil {
			return nil, fmt.Errorf("auth user %q: %w", name, err)
		}
		s.users[name] = hash
	}
	if s.mode == "login" && len(s.users) == 0 {
		return nil, fmt.Errorf("auth.mode=login requires at least one gateway.auth.users entry")
	}
	return s, nil
}

func passwordHash(plain string) ([]byte, error) {
	if strings.HasPrefix(plain, "$2a$") || strings.HasPrefix(plain, "$2b$") {
		return []byte(plain), nil
	}
	return bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
}

func (s *Service) Mode() string { return s.mode }

func (s *Service) BearerToken() string { return s.bearer }

func (s *Service) LoginRequired() bool { return s.mode == "login" }

func (s *Service) Login(username, password string) (token string, expiresAt time.Time, err error) {
	if !s.LoginRequired() {
		return "", time.Time{}, errors.New("login not enabled")
	}
	hash, ok := s.users[strings.TrimSpace(username)]
	if !ok || bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil {
		return "", time.Time{}, ErrInvalidCredentials
	}
	expiresAt = time.Now().UTC().Add(s.ttl)
	claims := jwt.MapClaims{
		"sub": strings.TrimSpace(username),
		"exp": expiresAt.Unix(),
		"iat": time.Now().UTC().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = t.SignedString(s.secret)
	return token, expiresAt, err
}

func (s *Service) ValidateBearer(header string) (username string, ok bool) {
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" {
		return "", false
	}
	if s.LoginRequired() {
		return s.parseJWT(raw)
	}
	if s.mode == "bearer" && s.bearer != "" && raw == s.bearer {
		return "", true
	}
	return "", false
}

func (s *Service) parseJWT(raw string) (string, bool) {
	tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil || !tok.Valid {
		return "", false
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", false
	}
	sub, _ := claims["sub"].(string)
	sub = strings.TrimSpace(sub)
	if sub == "" {
		return "", false
	}
	if _, ok := s.users[sub]; !ok {
		return "", false
	}
	return sub, true
}
