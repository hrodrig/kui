package auth

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func TestTOTPRoundTrip(t *testing.T) {
	secret, url, err := NewTOTPSecret("user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" || url == "" {
		t.Fatal("expected secret and url")
	}
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !ValidTOTPCode(secret, code) {
		t.Fatal("expected valid code")
	}
	if ValidTOTPCode(secret, "000000") {
		t.Fatal("expected invalid code")
	}
}

func TestQRCodeDataURI(t *testing.T) {
	_, url, err := NewTOTPSecret("a@b.c")
	if err != nil {
		t.Fatal(err)
	}
	uri, err := QRCodeDataURI(url)
	if err != nil {
		t.Fatal(err)
	}
	if len(uri) < 50 || uri[:22] != "data:image/png;base64," {
		t.Fatalf("unexpected data uri: %q", uri[:30])
	}
}
