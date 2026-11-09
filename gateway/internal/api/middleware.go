package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"piper_agent/gateway/internal/auth"
)

func isPublicAPIPath(path string) bool {
	switch path {
	case "/api/v1/health", "/api/v1/auth/login":
		return true
	default:
		return false
	}
}

func withAccessControl(svc *auth.Service, next http.Handler) http.Handler {
	if svc == nil || (svc.Mode() != "login" && svc.Mode() != "bearer") {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions || isPublicAPIPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		h := r.Header.Get("Authorization")
		username, ok := svc.ValidateBearer(h)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if svc.LoginRequired() {
			r = r.WithContext(auth.WithUser(r.Context(), username))
		}
		next.ServeHTTP(w, r)
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Trace-Id")
		w.Header().Set("Access-Control-Expose-Headers", "X-Trace-Id")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withTraceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid := r.Header.Get("X-Trace-Id")
		if tid == "" {
			tid = uuid.NewString()
		}
		w.Header().Set("X-Trace-Id", tid)
		r = r.WithContext(withTraceContext(r.Context(), tid))
		next.ServeHTTP(w, r)
	})
}

