package auth

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
)

const totpIssuer = "kui"

// NewTOTPSecret generates a base32 secret for the given account email.
func NewTOTPSecret(email string) (secret string, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: email,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// ValidTOTPCode checks a 6-digit authenticator code against the stored secret.
func ValidTOTPCode(secret, code string) bool {
	secret = strings.TrimSpace(secret)
	code = strings.TrimSpace(code)
	if secret == "" || code == "" {
		return false
	}
	return totp.Validate(code, secret)
}

// QRCodeDataURI returns a PNG data URI for an otpauth URL.
func QRCodeDataURI(otpauthURL string) (string, error) {
	png, err := qrcode.Encode(otpauthURL, qrcode.Medium, 200)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}

// NormalizeTOTPCode strips spaces from user input.
func NormalizeTOTPCode(code string) string {
	return strings.ReplaceAll(strings.TrimSpace(code), " ", "")
}

// TOTPURL builds the otpauth provisioning URL for an existing secret.
func TOTPURL(email, secret string) string {
	label := url.PathEscape(totpIssuer + ":" + email)
	return fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s", label, secret, url.QueryEscape(totpIssuer))
}

// ProvisioningLabel formats the manual-entry label shown during setup.
func ProvisioningLabel(email string) string {
	return fmt.Sprintf("%s:%s", totpIssuer, email)
}
