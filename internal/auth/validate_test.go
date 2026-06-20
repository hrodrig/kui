package auth

import "testing"

func TestValidEmail(t *testing.T) {
	ok := []string{"user@example.com", "a@b.co", "a@b.c", "admin@mail.localhost.dev", "admin@localhost"}
	bad := []string{"", "no-at", "@nodomain", "user@", "user@nodot", "user@domain.", "nbnbvnbvbn@dsfsddsfds"}

	for _, e := range ok {
		if !ValidEmail(e) {
			t.Errorf("expected valid: %q", e)
		}
	}
	for _, e := range bad {
		if ValidEmail(e) {
			t.Errorf("expected invalid: %q", e)
		}
	}
}

func TestTrimPassword(t *testing.T) {
	if TrimPassword("   ") != "" {
		t.Fatal("whitespace-only should be empty")
	}
	if TrimPassword("  secret  ") != "secret" {
		t.Fatal("expected trim")
	}
}

func TestValidPassword(t *testing.T) {
	ok := []string{"abc12345", "Passw0rd", "a1aaaaaa"}
	bad := []string{"", "short1", "11111111", "abcdefgh", "no-digits-here", "12345678"}

	for _, p := range ok {
		if !ValidPassword(p) {
			t.Errorf("expected valid: %q", p)
		}
	}
	for _, p := range bad {
		if ValidPassword(p) {
			t.Errorf("expected invalid: %q", p)
		}
	}
}
