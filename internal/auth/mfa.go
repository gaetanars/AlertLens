package auth

import (
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// ValidateTOTP checks whether code is a valid current TOTP code for the
// given base32-encoded secret.  It accepts a ±30 s skew window (1 step)
// to handle small clock drift between client and server.
//
// TOTP follows RFC 6238. The secret must be stored as a base32 string
// (compatible with Google Authenticator, Authy, 1Password, etc.).
func ValidateTOTP(secret, code string) error {
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1, // ±1 step (±30 s) to tolerate clock drift
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return fmt.Errorf("totp validation: %w", err)
	}
	if !valid {
		return ErrInvalidTOTP
	}
	return nil
}

// GenerateTOTPSecret creates a new TOTP secret for the given issuer and
// account label.  Returns the base32 secret and the otpauth:// provisioning
// URI suitable for encoding as a QR code.
//
// Intended for admin tooling / initial setup only.
func GenerateTOTPSecret(issuer, account string) (secret, otpauthURI string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: account,
		Period:      30,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", fmt.Errorf("generating totp key: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// ValidateTOTPSecret returns an error if s is not a valid base32-encoded
// TOTP secret string (catches common config typos before startup).
func ValidateTOTPSecret(s string) error {
	if s == "" {
		return nil // empty = MFA disabled, which is valid
	}
	// base32 is case-insensitive; pquerna/otp uppercases internally.
	_, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(s))
	if err != nil {
		return fmt.Errorf("totp_secret is not valid base32: %w", err)
	}
	return nil
}
