package auth

import (
	"net"
	"net/http"
	"sync"
	"time"

	"routex/i18n"
)

const (
	maxAttempts = 5
	windowSize  = 15 * time.Minute
)

type loginRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func newLoginRateLimiter() *loginRateLimiter {
	return &loginRateLimiter{
		attempts: make(map[string][]time.Time),
	}
}

func (l *loginRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-windowSize)

	times := l.attempts[ip]
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	l.attempts[ip] = valid

	return len(valid) < maxAttempts
}

func (l *loginRateLimiter) record(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attempts[ip] = append(l.attempts[ip], time.Now())
}

func extractIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type statusCapture struct {
	http.ResponseWriter
	code int
}

func (s *statusCapture) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func LoginRateLimitMiddleware() func(http.Handler) http.Handler {
	limiter := newLoginRateLimiter()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)
			if !limiter.allow(ip) {
				loc := i18n.FromContext(r.Context())
				http.Error(w, loc.T("error.rate_limited"), http.StatusTooManyRequests)
				return
			}

			capture := &statusCapture{ResponseWriter: w, code: http.StatusOK}
			next.ServeHTTP(capture, r)

			if capture.code >= 400 {
				limiter.record(ip)
			}
		})
	}
}
