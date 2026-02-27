package auth

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// loginRate allows 5 login attempts per minute per IP.
	loginRate  = rate.Limit(5.0 / 60.0)
	loginBurst = 5

	// cleanupTTL is how long an idle entry stays in the limiter map.
	cleanupTTL      = 10 * time.Minute
	cleanupInterval = 5 * time.Minute
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// LoginRateLimiter is a per-IP token-bucket rate limiter for the login endpoint.
// It keeps one Limiter per remote IP and periodically evicts stale entries.
type LoginRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
}

// NewLoginRateLimiter creates a LoginRateLimiter and starts the background
// cleanup goroutine. The goroutine exits when the process does.
func NewLoginRateLimiter() *LoginRateLimiter {
	rl := &LoginRateLimiter{entries: make(map[string]*ipEntry)}
	go rl.cleanupLoop()
	return rl
}

func (rl *LoginRateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	e, ok := rl.entries[ip]
	if !ok {
		e = &ipEntry{limiter: rate.NewLimiter(loginRate, loginBurst)}
		rl.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter
}

func (rl *LoginRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		for ip, e := range rl.entries {
			if time.Since(e.lastSeen) > cleanupTTL {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// Middleware returns an HTTP middleware that rate-limits by remote IP.
// Rejected requests receive HTTP 429 with a Retry-After header.
// It relies on chi's RealIP middleware having already normalised r.RemoteAddr.
func (rl *LoginRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractRemoteIP(r.RemoteAddr)
		if !rl.getLimiter(ip).Allow() {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error":"too many login attempts, please try again later"}`)) //nolint:errcheck
			return
		}
		next.ServeHTTP(w, r)
	})
}

// extractRemoteIP extracts the host portion from a "host:port" addr string.
// If addr is already a bare IP (as set by chi's RealIP middleware), it is
// returned unchanged.
func extractRemoteIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
