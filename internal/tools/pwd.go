package tools

import (
	"crypto/rand"
	"math/big"
)

const (
	charsetAlphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charsetSpecial      = "!@#$%^&*()-_=+[]{};:,.<>?"
)

// GeneratePassword returns a cryptographically random password of the given length.
// When special is true the password may also contain special characters (!@#$%^&*…).
// Defaults to alphanumeric-only when special is omitted.
func GeneratePassword(length int, special ...bool) string {
	charset := charsetAlphanumeric
	if len(special) > 0 && special[0] {
		charset += charsetSpecial
	}

	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range b {
		n, _ := rand.Int(rand.Reader, charsetLen)
		b[i] = charset[n.Int64()]
	}
	return string(b)
}
