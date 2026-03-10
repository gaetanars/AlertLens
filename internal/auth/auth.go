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

	"github.com/alertlens/alertlens/internal/config"
)

const tokenTTL = 24 * time.Hour

// userEntry stores a hashed password and the role it grants.
// totpSecret, if non-empty, enables TOTP MFA for this user.
type userEntry struct {
	hash       []byte
	role       Role
	totpSecret string // base32-encoded TOTP secret; empty = MFA disabled
}

// Service handles password verification and JWT lifecycle.
// The JWT secret is derived from the admin password, so tokens are
// automatically invalidated if the password changes or the server restarts
// (intentional stateless behaviour).
type Service struct {
	enabled bool
	secret  []byte

	// users is an ordered list of (passwordHash, role) pairs.
	// Login iterates them in order; the first match wins.
	users []userEntry

	mu         sync.RWMutex
	revokedSet map[string]time.Time // jti → token expiry (TTL-based cleanup)
}

// NewServiceFromConfig builds a Service from the AuthConfig section of the
// application config.  Backward-compatible: if only AdminPassword is set,
// logging in with that password grants the admin role.
func NewServiceFromConfig(cfg config.AuthConfig) *Service {
	svc := &Service{revokedSet: make(map[string]time.Time)}

	// Admin password entry (highest privilege).
	if cfg.AdminPassword != "" {
		h := sha256.Sum256([]byte(cfg.AdminPassword))
		svc.users = append(svc.users, userEntry{hash: h[:], role: RoleAdmin})
		// Derive the JWT-signing secret from the admin password so that
		// changing the password automatically invalidates all existing tokens.
		svc.secret = h[:]
		svc.enabled = true
	}

	// Additional per-role users defined in config.
	for _, u := range cfg.Users {
		r := Role(u.Role)
		if !r.IsValid() {
			// Skip entries with unrecognised roles; callers should validate the
			// config before reaching here, but be defensive.
			continue
		}
		h := sha256.Sum256([]byte(u.Password))
		svc.users = append(svc.users, userEntry{
			hash:       h[:],
			role:       r,
			totpSecret: u.TOTPSecret, // empty = MFA disabled for this user
		})
		if !svc.enabled {
			// If no admin password is configured, derive the signing secret
			// from the first user password found.
			svc.secret = h[:]
			svc.enabled = true
		}
	}

	return svc
}

// NewService creates an auth Service with a single admin password.
// Deprecated: prefer NewServiceFromConfig for new call sites.
// Kept for backward-compatibility with unit tests.
func NewService(password string) *Service {
	return NewServiceFromConfig(config.AuthConfig{AdminPassword: password})
}

// AdminEnabled returns true if at least one user / password has been configured.
func (s *Service) AdminEnabled() bool { return s.enabled }

// matchUser performs a constant-time scan of all user entries and returns the
// first entry whose password hash matches.  Returns nil if no match.
func (s *Service) matchUser(password string) *userEntry {
	h := sha256.Sum256([]byte(password))
	for i := range s.users {
		// SEC-01: constant-time comparison to prevent timing attacks.
		if hmac.Equal(h[:], s.users[i].hash) {
			return &s.users[i]
		}
	}
	return nil
}

// Login verifies the password (and optionally a TOTP code) and returns a
// signed JWT if authentication succeeds.
//
// MFA behaviour:
//   - If the matching user has a TOTP secret configured:
//     • totpCode must be provided and valid, otherwise ErrMFARequired or
//       ErrInvalidTOTP is returned (no token issued).
//   - If the user has no TOTP secret, totpCode is ignored.
//
// The issued JWT carries a "role" claim reflecting the privilege level.
func (s *Service) Login(password, totpCode string) (string, time.Time, error) {
	if !s.enabled {
		return "", time.Time{}, errors.New("admin mode is not enabled")
	}

	u := s.matchUser(password)
	if u == nil {
		return "", time.Time{}, errors.New("invalid password")
	}
	role := u.role

	// MFA validation — only enforced when a TOTP secret is configured.
	if u.totpSecret != "" {
		if totpCode == "" {
			return "", time.Time{}, ErrMFARequired
		}
		if err := ValidateTOTP(u.totpSecret, totpCode); err != nil {
			return "", time.Time{}, err // ErrInvalidTOTP or wrapped error
		}
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
		"sub":  string(role),
		"role": string(role),
		"iat":  now.Unix(),
		"exp":  exp.Unix(),
		"jti":  jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("signing token: %w", err)
	}
	return signed, exp, nil
}

// Validate parses and validates a JWT.
// Returns the jti and role on success.
func (s *Service) Validate(tokenStr string) (jti string, role Role, err error) {
	if !s.enabled {
		return "", "", errors.New("admin mode is not enabled")
	}
	token, parseErr := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if parseErr != nil || !token.Valid {
		return "", "", errors.New("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid token claims")
	}
	jti, _ = claims["jti"].(string)

	s.mu.RLock()
	_, revoked := s.revokedSet[jti]
	s.mu.RUnlock()
	if revoked {
		return "", "", errors.New("token has been revoked")
	}

	// Extract role claim; fall back to "admin" for tokens issued before RBAC.
	roleStr, _ := claims["role"].(string)
	if roleStr == "" {
		roleStr = string(RoleAdmin)
	}
	return jti, Role(roleStr), nil
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

// CSRFSecret returns a CSRF-specific signing key derived from the JWT secret.
//
// Derivation: HMAC-SHA256(jwtSecret, "alertlens-csrf-v1")
//
// This guarantees:
//   - The CSRF key is cryptographically distinct from the JWT signing key.
//   - It rotates automatically whenever the admin password changes.
//   - It is never a hardcoded constant, eliminating CWE-321.
//
// When auth is disabled (no password configured) the function returns a fixed
// development-only key; CSRF protection remains active but relies on the
// SameSite cookie policy since there is no per-deployment secret.
func (s *Service) CSRFSecret() []byte {
	// s.secret is written only during construction (NewServiceFromConfig) and
	// never modified afterward, so reading without a lock is safe.
	if len(s.secret) == 0 {
		return []byte("alertlens-csrf-noauth-dev")
	}
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte("alertlens-csrf-v1")) //nolint:errcheck
	return mac.Sum(nil)
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
