package auth

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"piper_go/internal/config"
)

type User struct {
	ID       string
	Username string
	Email    string
	Enabled  bool
	Roles    []string
}

type Service struct {
	serviceToken string
	tokenTTL     time.Duration
	accounts     map[string]account
	tokens       map[string]tokenRecord
	mu           sync.RWMutex
}

type account struct {
	password string
	email    string
	roles    []string
}

type tokenRecord struct {
	user      User
	expiresAt time.Time
}

func NewService(c config.AuthConf) *Service {
	s := &Service{
		serviceToken: c.ServiceToken,
		tokenTTL:     time.Duration(c.TokenTTLSeconds) * time.Second,
		accounts:     make(map[string]account),
		tokens:       make(map[string]tokenRecord),
	}
	for _, u := range c.Users {
		email := u.Email
		if email == "" {
			email = u.Username + "@local"
		}
		s.accounts[u.Username] = account{
			password: u.Password,
			email:    email,
			roles:    append([]string(nil), u.Roles...),
		}
	}
	return s
}

func (s *Service) VerifyAccessToken(token string) (User, error) {
	if token == "" {
		return User{}, errors.New("empty token")
	}

	if token == s.serviceToken {
		return s.buildServiceUser(), nil
	}

	s.mu.RLock()
	rec, ok := s.tokens[token]
	s.mu.RUnlock()
	if !ok {
		return User{}, errors.New("unknown token")
	}
	if time.Now().After(rec.expiresAt) {
		s.mu.Lock()
		delete(s.tokens, token)
		s.mu.Unlock()
		return User{}, errors.New("token expired")
	}

	user := rec.user
	user.Roles = append([]string(nil), rec.user.Roles...)
	return user, nil
}

func (s *Service) buildServiceUser() User {
	return User{
		ID:       "piper-service",
		Username: "service",
		Email:    "service@local",
		Enabled:  true,
		Roles:    []string{"admin", "user"},
	}
}

func userID(username string) string {
	sum := md5.Sum([]byte("piper-user:" + username))
	return hex.EncodeToString(sum[:])
}

func (s *Service) buildUser(username string, acc account) User {
	return User{
		ID:       userID(username),
		Username: username,
		Email:    acc.email,
		Enabled:  true,
		Roles:    append([]string(nil), acc.roles...),
	}
}

// Login issues a bearer token (Phase 1 /auth/login will call this).
func (s *Service) Login(username, password string) (token string, err error) {
	acc, ok := s.accounts[username]
	if !ok || acc.password != password {
		return "", fmt.Errorf("invalid credentials")
	}
	user := s.buildUser(username, acc)
	token = newToken()
	exp := time.Now().Add(s.tokenTTL)
	s.mu.Lock()
	s.tokens[token] = tokenRecord{user: user, expiresAt: exp}
	s.mu.Unlock()
	return token, nil
}

func newToken() string {
	sum := md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(sum[:])
}

func (s *Service) RevokeToken(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	delete(s.tokens, token)
	s.mu.Unlock()
}

func (s *Service) UserProfile(token string) (map[string]any, error) {
	user, err := s.VerifyAccessToken(token)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"sub":                user.ID,
		"preferred_username": user.Username,
		"name":               user.Username,
		"email":              user.Email,
		"roles":              user.Roles,
	}, nil
}

type LoginCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func ParseLoginBody(body []byte) (LoginCredentials, error) {
	var cred LoginCredentials
	if err := json.Unmarshal(body, &cred); err != nil {
		return cred, err
	}
	if cred.Username == "" || cred.Password == "" {
		return cred, fmt.Errorf("username and password required")
	}
	return cred, nil
}
