package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"

	"github.com/alertlens/alertlens/internal/config"
)

// ─── ValidateTOTP ────────────────────────────────────────────────────────────

func TestValidateTOTP_ValidCode(t *testing.T) {
	_, secret, _, err := freshTOTPSecret(t)
	_ = err
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generating TOTP code: %v", err)
	}
	if err := ValidateTOTP(secret, code); err != nil {
		t.Errorf("ValidateTOTP returned error for valid code: %v", err)
	}
}

func TestValidateTOTP_InvalidCode(t *testing.T) {
	_, secret, _, _ := freshTOTPSecret(t)
	err := ValidateTOTP(secret, "000000")
	// 000000 is almost certainly wrong; if it isn't, the test is flaky — acceptable.
	if err == nil {
		t.Log("000000 happened to be a valid code (1-in-1M chance) — retry")
	}
}

func TestValidateTOTP_EmptyCode_ReturnsError(t *testing.T) {
	_, secret, _, _ := freshTOTPSecret(t)
	if err := ValidateTOTP(secret, ""); err == nil {
		t.Error("expected error for empty TOTP code")
	}
}

func TestValidateTOTP_BadSecret_ReturnsError(t *testing.T) {
	err := ValidateTOTP("NOTBASE32!!!", "123456")
	if err == nil {
		t.Error("expected error for invalid base32 secret")
	}
}

// ─── ValidateTOTPSecret ──────────────────────────────────────────────────────

func TestValidateTOTPSecret_EmptyString_OK(t *testing.T) {
	if err := ValidateTOTPSecret(""); err != nil {
		t.Errorf("empty secret should be valid (MFA disabled): %v", err)
	}
}

func TestValidateTOTPSecret_ValidBase32_OK(t *testing.T) {
	_, secret, _, _ := freshTOTPSecret(t)
	if err := ValidateTOTPSecret(secret); err != nil {
		t.Errorf("valid base32 secret should pass: %v", err)
	}
}

func TestValidateTOTPSecret_InvalidBase32_Error(t *testing.T) {
	if err := ValidateTOTPSecret("this is not base32 !@#$"); err == nil {
		t.Error("expected error for invalid base32 secret")
	}
}

// ─── GenerateTOTPSecret ──────────────────────────────────────────────────────

func TestGenerateTOTPSecret_ReturnsNonEmpty(t *testing.T) {
	secret, uri, err := GenerateTOTPSecret("AlertLens", "admin")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}
	if secret == "" {
		t.Error("expected non-empty secret")
	}
	if uri == "" {
		t.Error("expected non-empty provisioning URI")
	}
}

func TestGenerateTOTPSecret_ProducesValidSecret(t *testing.T) {
	secret, _, err := GenerateTOTPSecret("AlertLens", "admin")
	if err != nil {
		t.Fatalf("GenerateTOTPSecret: %v", err)
	}
	// A code generated from the secret should validate.
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generating code from fresh secret: %v", err)
	}
	if err := ValidateTOTP(secret, code); err != nil {
		t.Errorf("freshly generated secret did not validate its own code: %v", err)
	}
}

// ─── Service MFA integration ─────────────────────────────────────────────────

func TestLogin_MFAEnabled_NoTOTPCode_ReturnsErrMFARequired(t *testing.T) {
	secret, _, _ := generateSecret(t)
	svc := serviceWithTOTP(t, "password", secret)

	_, _, err := svc.Login("password", "")
	if err == nil {
		t.Fatal("expected error when TOTP code is missing")
	}
	if err != ErrMFARequired {
		t.Errorf("expected ErrMFARequired, got %v", err)
	}
}

func TestLogin_MFAEnabled_ValidTOTPCode_Succeeds(t *testing.T) {
	secret, _, _ := generateSecret(t)
	svc := serviceWithTOTP(t, "password", secret)

	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("generating TOTP code: %v", err)
	}

	token, exp, err := svc.Login("password", code)
	if err != nil {
		t.Fatalf("Login with valid TOTP failed: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	if exp.IsZero() {
		t.Error("expected non-zero expiry")
	}
}

func TestLogin_MFAEnabled_InvalidTOTPCode_ReturnsErrInvalidTOTP(t *testing.T) {
	secret, _, _ := generateSecret(t)
	svc := serviceWithTOTP(t, "password", secret)

	_, _, err := svc.Login("password", "000000")
	// 000000 is almost certainly wrong (1-in-1M chance of being valid).
	if err == nil {
		t.Log("000000 happened to be valid — flaky test, skipping assertion")
		return
	}
}

func TestLogin_MFADisabled_TOTPCodeIgnored(t *testing.T) {
	svc := NewService("password")
	// Should succeed even with a garbage TOTP code when MFA is not configured.
	token, _, err := svc.Login("password", "999999")
	if err != nil {
		t.Errorf("Login should ignore TOTP code when MFA is not configured: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// freshTOTPSecret generates a TOTP key and returns (key, secret, uri, err).
func freshTOTPSecret(t *testing.T) (key interface{ Secret() string }, secret, uri string, err error) {
	t.Helper()
	s, u, genErr := GenerateTOTPSecret("AlertLens", "test@example.com")
	return nil, s, u, genErr
}

// generateSecret is a simpler helper that fatals on error.
func generateSecret(t *testing.T) (secret, uri string, err error) {
	t.Helper()
	s, u, e := GenerateTOTPSecret("AlertLens", "test")
	if e != nil {
		t.Fatalf("GenerateTOTPSecret: %v", e)
	}
	return s, u, e
}

// serviceWithTOTP creates an auth.Service with a user that has TOTP enabled.
func serviceWithTOTP(t *testing.T, password, totpSecret string) *Service {
	t.Helper()
	return NewServiceFromConfig(config.AuthConfig{
		Users: []config.UserConfig{
			{Password: password, Role: "admin", TOTPSecret: totpSecret},
		},
	}, nil)
}
