package tools

import (
	"strings"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short password", 5},
		{"medium password", 12},
		{"long password", 32},
		{"very long password", 64},
		{"minimum length", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd := GeneratePassword(tt.length)
			if len(pwd) != tt.length {
				t.Errorf("GeneratePassword(%d) length = %d, want %d", tt.length, len(pwd), tt.length)
			}
		})
	}
}

func TestGeneratePasswordCharacters(t *testing.T) {
	const length = 50
	const iterations = 100

	validChars := charsetAlphanumeric
	hasLetter := false
	hasNumber := false

	for i := 0; i < iterations; i++ {
		pwd := GeneratePassword(length)

		if strings.ContainsAny(pwd, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			hasLetter = true
		}
		if strings.ContainsAny(pwd, "0123456789") {
			hasNumber = true
		}

		for _, char := range pwd {
			if !strings.ContainsRune(validChars, char) {
				t.Errorf("GeneratePassword generated invalid character: %c (code: %d)", char, char)
			}
		}
	}

	if !hasLetter {
		t.Error("GeneratePassword never generated letters in 100 iterations")
	}
	if !hasNumber {
		t.Error("GeneratePassword never generated numbers in 100 iterations")
	}
}

func TestGeneratePasswordSpecialChars(t *testing.T) {
	const length = 100
	const iterations = 50

	validChars := charsetAlphanumeric + charsetSpecial
	hasSpecial := false

	for i := 0; i < iterations; i++ {
		pwd := GeneratePassword(length, true)

		if len(pwd) != length {
			t.Errorf("GeneratePassword(%d, true) length = %d, want %d", length, len(pwd), length)
		}

		for _, char := range pwd {
			if !strings.ContainsRune(validChars, char) {
				t.Errorf("GeneratePassword(special=true) generated invalid character: %c (code: %d)", char, char)
			}
		}

		if strings.ContainsAny(pwd, charsetSpecial) {
			hasSpecial = true
		}
	}

	if !hasSpecial {
		t.Error("GeneratePassword(special=true) never produced a special character in 50 iterations")
	}
}

func TestGeneratePasswordNoSpecialByDefault(t *testing.T) {
	const length = 200
	const iterations = 20

	for i := 0; i < iterations; i++ {
		pwd := GeneratePassword(length)
		for _, char := range pwd {
			if strings.ContainsRune(charsetSpecial, char) {
				t.Errorf("GeneratePassword (no special) produced special character: %c", char)
			}
		}
	}
}

func TestGeneratePasswordUniqueness(t *testing.T) {
	const length = 20
	const iterations = 10

	passwords := make(map[string]bool)
	for i := 0; i < iterations; i++ {
		passwords[GeneratePassword(length)] = true
	}

	if len(passwords) < 2 {
		t.Error("GeneratePassword generated identical passwords, randomness may be broken")
	}
}

func TestGeneratePasswordZeroLength(t *testing.T) {
	pwd := GeneratePassword(0)
	if len(pwd) != 0 {
		t.Errorf("GeneratePassword(0) length = %d, want 0", len(pwd))
	}
	if pwd != "" {
		t.Errorf("GeneratePassword(0) = %q, want empty string", pwd)
	}
}
