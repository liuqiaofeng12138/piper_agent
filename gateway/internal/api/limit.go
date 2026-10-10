package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu       sync.Mutex
	perMin   int
	requests map[string][]time.Time
}

func newRateLimiter(perMinute int) *rateLimiter {
	if perMinute <= 0 {
		perMinute = 60
	}
	return &rateLimiter{
		perMin:   perMinute,
		requests: make(map[string][]time.Time),
	}
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.Index(xff, ","); i > 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (l *rateLimiter) allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	l.mu.Lock()
	defer l.mu.Unlock()
	times := l.requests[ip]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.perMin {
		l.requests[ip] = kept
		return false
	}
	kept = append(kept, now)
	l.requests[ip] = kept
	return true
}

func withRateLimit(perMinute int, next http.Handler) http.Handler {
	lim := newRateLimiter(perMinute)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !lim.allow(clientIP(r)) {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withMaxBody(defaultMax int64, uploadMax int64, next http.Handler) http.Handler {
	if defaultMax <= 0 {
		defaultMax = 1 << 20
	}
	if uploadMax <= 0 {
		uploadMax = 32 << 20
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut {
			limit := defaultMax
			if r.URL.Path == "/api/v1/chat/completions" &&
				strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				limit = uploadMax
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}
