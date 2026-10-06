package middleware

import (
	"context"
	"net/http"
	"strings"

	"piper_go/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

type ctxKey int

const (
	ctxKeyUID ctxKey = iota + 1
)

// UIDFromContext returns authenticated user id set by auth middleware.
func UIDFromContext(r *http.Request) string {
	uid, _ := r.Context().Value(ctxKeyUID).(string)
	return uid
}

// Register attaches Piper global middleware to the go-zero server.
func Register(server *rest.Server, s *svc.ServiceContext) {
	server.Use(corsMiddleware())
	server.Use(readyMiddleware(s))
	server.Use(authMiddleware(s))
	server.Use(jsonHeadersMiddleware())
}

func corsMiddleware() func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Add("Vary", "Origin")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE, PATCH, HEAD")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Authorization, AccessToken, Token, X-CSRF-Token, Range")
			w.Header().Set("Access-Control-Expose-Headers", "Authorization, Content-Length")

			if accessHeaders := r.Header.Get("Access-Control-Request-Headers"); accessHeaders != "" {
				w.Header().Set("Access-Control-Allow-Headers", accessHeaders)
			}
			if accessMethod := r.Header.Get("Access-Control-Request-Method"); accessMethod != "" {
				w.Header().Set("Access-Control-Allow-Methods", accessMethod)
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
				return
			}

			next(w, r)
		}
	}
}

func readyMiddleware(s *svc.ServiceContext) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if !pathEqualsAny(path, initRoutes) && !s.Ready() {
				http.Error(w, "Server not ready", http.StatusInternalServerError)
				return
			}
			next(w, r)
		}
	}
}

func authMiddleware(s *svc.ServiceContext) func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if s.Config.WebAPI.NoAuth {
				next(w, r)
				return
			}
			if r.Method == http.MethodOptions || pathEqualsAny(r.URL.Path, noAuthRoutes) {
				next(w, r)
				return
			}

			token, err := bearerToken(r)
			if err != nil {
				logx.WithContext(r.Context()).Infof("token verify error, path=%s: %v", r.URL.Path, err)
				http.Error(w, "Token error", http.StatusUnauthorized)
				return
			}

			user, err := s.Auth.VerifyAccessToken(token)
			if err != nil {
				logx.WithContext(r.Context()).Infof("token verify error, path=%s: %v", r.URL.Path, err)
				http.Error(w, "Token error", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUID, user.ID)
			next(w, r.WithContext(ctx))
		}
	}
}

func jsonHeadersMiddleware() func(next http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !isMetricsPath(r.URL.Path) {
				w.Header().Set("Cache-Control", "no-store")
				// Java sets Content-Encoding: gzip; defer body compression to a later phase.
			}
			next(w, r)
		}
	}
}

func bearerToken(r *http.Request) (string, error) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errMissingAuth
	}
	if !strings.Contains(h, prefix) {
		return "", errMissingAuth
	}
	parts := strings.SplitN(h, prefix, 2)
	if len(parts) != 2 || parts[1] == "" {
		return "", errMissingAuth
	}
	return strings.TrimSpace(parts[1]), nil
}

var errMissingAuth = &authError{"missing authorization"}

type authError struct{ msg string }

func (e *authError) Error() string { return e.msg }
