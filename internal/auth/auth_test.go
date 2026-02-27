package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ─── NewService ───────────────────────────────────────────────────────────────

func TestNewService_NoPassword_AdminDisabled(t *testing.T) {
	svc := NewService("")
	if svc.AdminEnabled() {
		t.Error("expected admin to be disabled when no password is set")
	}
}

func TestNewService_WithPassword_AdminEnabled(t *testing.T) {
	svc := NewService("secret")
	if !svc.AdminEnabled() {
		t.Error("expected admin to be enabled when a password is set")
	}
}

// ─── Login ────────────────────────────────────────────────────────────────────

func TestLogin_ValidPassword(t *testing.T) {
	svc := NewService("hunter2")
	token, exp, err := svc.Login("hunter2")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if token == "" {
		t.Error("expected a non-empty token")
	}
	// Expiry should be ~24 hours from now.
	if time.Until(exp) < 23*time.Hour {
		t.Errorf("token expiry too soon: %v", exp)
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	svc := NewService("hunter2")
	_, _, err := svc.Login("wrongpassword")
	if err == nil {
		t.Error("expected error for wrong password")
	}
}

func TestLogin_AdminDisabled(t *testing.T) {
	svc := NewService("")
	_, _, err := svc.Login("anypassword")
	if err == nil {
		t.Error("expected error when admin mode is disabled")
	}
}

func TestLogin_TokenIsJWT(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, err := svc.Login("secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	// A JWT has exactly 3 dot-separated segments.
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 JWT segments, got %d", len(parts))
	}
}

// ─── Validate ─────────────────────────────────────────────────────────────────

func TestValidate_ValidToken(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, err := svc.Login("secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	jti, err := svc.Validate(tokenStr)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if jti == "" {
		t.Error("expected non-empty jti")
	}
}

func TestValidate_InvalidToken(t *testing.T) {
	svc := NewService("secret")
	_, err := svc.Validate("not.a.token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestValidate_EmptyToken(t *testing.T) {
	svc := NewService("secret")
	_, err := svc.Validate("")
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestValidate_AdminDisabled(t *testing.T) {
	svc := NewService("")
	_, err := svc.Validate("any.token.here")
	if err == nil {
		t.Error("expected error when admin mode is disabled")
	}
}

func TestValidate_TokenSignedWithDifferentSecret(t *testing.T) {
	other := NewService("otherpassword")
	tokenStr, _, err := other.Login("otherpassword")
	if err != nil {
		t.Fatalf("Login with other svc: %v", err)
	}

	svc := NewService("secret")
	_, err = svc.Validate(tokenStr)
	if err == nil {
		t.Error("expected error for token signed with wrong secret")
	}
}

func TestValidate_ExpiredToken(t *testing.T) {
	svc := NewService("secret")

	// Craft an already-expired token using the same secret.
	claims := jwt.MapClaims{
		"sub": "admin",
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
		"exp": time.Now().Add(-1 * time.Hour).Unix(), // already expired
		"jti": "test-jti-expired",
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := tok.SignedString(svc.secret)
	if err != nil {
		t.Fatalf("signing expired token: %v", err)
	}

	_, err = svc.Validate(tokenStr)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestValidate_WrongSigningAlgorithm(t *testing.T) {
	svc := NewService("secret")

	// Create token with a none algorithm (should be rejected).
	claims := jwt.MapClaims{
		"sub": "admin",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"jti": "test-jti-alg",
	}
	// jwt.SigningMethodNone requires jwt.UnsafeAllowNoneSignatureType
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing none-alg token: %v", err)
	}

	_, err = svc.Validate(tokenStr)
	if err == nil {
		t.Error("expected error for token with 'none' signing algorithm")
	}
}

// ─── Revoke ───────────────────────────────────────────────────────────────────

func TestRevoke_ValidToken_SubsequentValidateFails(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, err := svc.Login("secret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// Token should be valid before revocation.
	if _, err := svc.Validate(tokenStr); err != nil {
		t.Fatalf("Validate before revoke: %v", err)
	}

	svc.Revoke(tokenStr)

	// Token should be invalid after revocation.
	if _, err := svc.Validate(tokenStr); err == nil {
		t.Error("expected error after token revocation")
	}
}

func TestRevoke_InvalidToken_NoPanic(t *testing.T) {
	svc := NewService("secret")
	// Should not panic on invalid input.
	svc.Revoke("not.a.valid.token")
	svc.Revoke("")
}

func TestRevoke_TokenFromDifferentSecret_NoPanic(t *testing.T) {
	other := NewService("other")
	tokenStr, _, _ := other.Login("other")

	svc := NewService("secret")
	svc.Revoke(tokenStr) // should not panic
}

func TestRevoke_Idempotent(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, _ := svc.Login("secret")

	svc.Revoke(tokenStr)
	svc.Revoke(tokenStr) // second revocation must not panic or change behaviour

	_, err := svc.Validate(tokenStr)
	if err == nil {
		t.Error("expected error after double revocation")
	}
}

// ─── purgeExpiredLocked ───────────────────────────────────────────────────────

func TestPurgeExpiredLocked_RemovesExpiredEntries(t *testing.T) {
	svc := NewService("secret")

	// Inject an already-expired entry directly.
	svc.mu.Lock()
	svc.revokedSet["expired-jti"] = time.Now().Add(-1 * time.Hour)
	svc.mu.Unlock()

	// Trigger a revocation which calls purgeExpiredLocked internally.
	tokenStr, _, _ := svc.Login("secret")
	svc.Revoke(tokenStr)

	svc.mu.RLock()
	_, stillPresent := svc.revokedSet["expired-jti"]
	svc.mu.RUnlock()

	if stillPresent {
		t.Error("expected expired JTI to be purged from revocation set")
	}
}

func TestPurgeExpiredLocked_KeepsActiveEntries(t *testing.T) {
	svc := NewService("secret")
	tokenStr, _, _ := svc.Login("secret")
	svc.Revoke(tokenStr)

	// The just-revoked token's jti should still be in the set (not yet expired).
	svc.mu.RLock()
	count := len(svc.revokedSet)
	svc.mu.RUnlock()

	if count == 0 {
		t.Error("expected revoked JTI to remain in set until token TTL expires")
	}
}
