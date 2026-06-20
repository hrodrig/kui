package auth

import (
	"strings"
	"unicode"
)

// ValidEmail checks local@domain.tld (domain must include a dot before the TLD).
func ValidEmail(email string) bool {
	email = strings.TrimSpace(email)
	i := strings.LastIndex(email, "@")
	if i <= 0 || i >= len(email)-1 {
		return false
	}
	local, domain := email[:i], email[i+1:]
	if local == "" || domain == "" {
		return false
	}
	if domain == "localhost" {
		return true
	}
	dot := strings.LastIndex(domain, ".")
	if dot <= 0 || dot >= len(domain)-1 {
		return false
	}
	return len(domain[dot+1:]) >= 1
}

// TrimPassword treats whitespace-only passwords as empty.
func TrimPassword(pw string) string {
	return strings.TrimSpace(pw)
}

// PasswordPolicy describes password rules for UI copy.
const PasswordPolicy = "at least 8 characters with letters and numbers"

// ValidPassword requires length, letter+digit mix, and rejects trivial repeats.
func ValidPassword(pw string) bool {
	if len(pw) < 8 {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range pw {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return false
	}
	return !allSameRunes([]rune(pw))
}

func allSameRunes(runes []rune) bool {
	if len(runes) <= 1 {
		return true
	}
	for _, r := range runes[1:] {
		if r != runes[0] {
			return false
		}
	}
	return true
}
