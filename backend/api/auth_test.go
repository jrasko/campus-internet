package api

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHMACSecretKey(t *testing.T) {
	t.Run("valid base64 string with 64 or more bytes", func(t *testing.T) {
		// Create a secret of 64 raw bytes and base64-encode it
		rawSecret := []byte(strings.Repeat("a", 64))
		base64Secret := base64.StdEncoding.EncodeToString(rawSecret)

		result := getHMACSecretKey(base64Secret)
		assert.Equal(t, rawSecret, result)
	})

	t.Run("valid base64 string with less than 64 bytes", func(t *testing.T) {
		// Create a secret of 32 raw bytes and base64-encode it
		rawSecret := []byte(strings.Repeat("b", 32))
		base64Secret := base64.StdEncoding.EncodeToString(rawSecret)

		result := getHMACSecretKey(base64Secret)
		// Should fall back to raw string bytes because decoded length < 64
		assert.Equal(t, []byte(base64Secret), result)
	})

	t.Run("invalid base64 string", func(t *testing.T) {
		plainSecret := "my-plain-text-secret-key-that-is-not-base64!"
		result := getHMACSecretKey(plainSecret)
		assert.Equal(t, []byte(plainSecret), result)
	})

	t.Run("empty string", func(t *testing.T) {
		result := getHMACSecretKey("")
		assert.Equal(t, []byte(""), result)
	})
}
