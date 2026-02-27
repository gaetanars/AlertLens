package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const tokenTTL = 24 * time.Hour

// Service handles password verification and JWT lifecycle.
// The JWT secret is derived from the admin password, so tokens are
// automatically invalidated if the password changes or the server restarts
// (intentional stateless behaviour).
type Service struct {
	enabled bool
	secret  []byte

	mu         sync.RWMutex
	revokedSet map[string]time.Time // jti → token expiry (TTL-based cleanup)
}

// NewService creates an auth Service. If password is empty, admin mode is disabled.
func NewService(password string) *Service {
	svc := &Service{revokedSet: make(map[string]time.Time)}
	if password == "" {
		return svc
	}
	svc.enabled = true
	h := sha256.Sum256([]byte(password))
	svc.secret = h[:]
	return svc
}

// AdminEnabled returns true if a password has been configured.
func (s *Service) AdminEnabled() bool { return s.enabled }

// Login verifies the password and returns a signed JWT if correct.
func (s *Service) Login(password string) (string, time.Time, error) {
	if !s.enabled {
		return "", time.Time{}, errors.New("admin mode is not enabled")
	}
	h := sha256.Sum256([]byte(password))
	// SEC-01: constant-time comparison to prevent timing attacks.
	if !hmac.Equal(h[:], s.secret) {
		return "", time.Time{}, errors.New("invalid password")
	}

	now := time.Now()
	exp := now.Add(tokenTTL)

	// SEC-05: cryptographically random JTI prevents collisions and guessing.
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", time.Time{}, fmt.Errorf("generating JTI: %w", err)
	}
	jti := fmt.Sprintf("%x", jtiBytes)

	claims := jwt.MapClaims{
		"sub": "admin",
		"iat": now.Unix(),
		"exp": exp.Unix(),
		"jti": jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing token: %w", err)
	}
	return signed, exp, nil
}

// Validate parses and validates a JWT. Returns the jti on success.
func (s *Service) Validate(tokenStr string) (string, error) {
	if !s.enabled {
		return "", errors.New("admin mode is not enabled")
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}
	jti, _ := claims["jti"].(string)

	s.mu.RLock()
	_, revoked := s.revokedSet[jti]
	s.mu.RUnlock()
	if revoked {
		return "", errors.New("token has been revoked")
	}

	return jti, nil
}

// Revoke adds a token's jti to the revocation set (logout).
func (s *Service) Revoke(tokenStr string) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		return s.secret, nil
	})
	if err != nil || !token.Valid {
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return
	}

	// Store with expiry for SEC-06 TTL-based cleanup.
	var exp time.Time
	if expClaim, ok := claims["exp"].(float64); ok {
		exp = time.Unix(int64(expClaim), 0)
	} else {
		exp = time.Now().Add(tokenTTL)
	}

	s.mu.Lock()
	s.revokedSet[jti] = exp
	s.purgeExpiredLocked() // opportunistic cleanup on each revocation
	s.mu.Unlock()
}

// purgeExpiredLocked removes JTIs whose tokens have already expired.
// Must be called with s.mu held.
func (s *Service) purgeExpiredLocked() {
	now := time.Now()
	for jti, exp := range s.revokedSet {
		if now.After(exp) {
			delete(s.revokedSet, jti)
		}
	}
}
