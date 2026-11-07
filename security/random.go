package security

import (
	"crypto/rand"
	"fmt"
)

const lowercaseAlphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomSecret returns a cryptographically random lowercase string of
// the requested length. Returns an empty string when length <= 0.
func RandomSecret(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("security: rand read: %w", err)
	}
	out := make([]byte, length)
	for i, b := range buf {
		out[i] = lowercaseAlphabet[int(b)%len(lowercaseAlphabet)]
	}
	return string(out), nil
}

// RandomBytes returns a cryptographically random byte slice of the
// requested length.
func RandomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, nil
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("security: rand read: %w", err)
	}
	return buf, nil
}
