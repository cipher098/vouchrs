package cipher_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gothi/vouchrs/src/external/cipher"
)

// 32-byte test key as 64 hex chars.
const testKey = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func newCipher(t *testing.T) interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
	Hash(string) string
} {
	t.Helper()
	svc, err := cipher.NewAESCipher(testKey)
	require.NoError(t, err)
	return svc
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	c := newCipher(t)
	plain := "AMZN-1234-5678-9012"

	enc, err := c.Encrypt(plain)
	require.NoError(t, err)
	assert.NotEmpty(t, enc)
	assert.NotEqual(t, plain, enc)

	dec, err := c.Decrypt(enc)
	require.NoError(t, err)
	assert.Equal(t, plain, dec)
}

func TestEncrypt_ProducesUniqueCiphertexts(t *testing.T) {
	c := newCipher(t)
	plain := "SAME-CODE"

	enc1, _ := c.Encrypt(plain)
	enc2, _ := c.Encrypt(plain)
	// GCM uses random nonce, so two encryptions of the same plaintext must differ
	assert.NotEqual(t, enc1, enc2)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	c := newCipher(t)
	enc, _ := c.Encrypt("valid-code")

	// Flip a character in the middle of the base64 to corrupt the ciphertext
	b := []byte(enc)
	b[len(b)/2] ^= 0xFF
	_, err := c.Decrypt(string(b))
	assert.Error(t, err)
}

func TestHash_Deterministic(t *testing.T) {
	c := newCipher(t)
	h1 := c.Hash("CARD-CODE")
	h2 := c.Hash("CARD-CODE")
	assert.Equal(t, h1, h2)
	assert.Len(t, h1, 64) // SHA-256 hex = 64 chars
}

func TestHash_DifferentInputs(t *testing.T) {
	c := newCipher(t)
	assert.NotEqual(t, c.Hash("CODE-A"), c.Hash("CODE-B"))
}

func TestNewAESCipher_BadKey(t *testing.T) {
	_, err := cipher.NewAESCipher("not-hex!")
	assert.Error(t, err)

	_, err = cipher.NewAESCipher(strings.Repeat("ab", 16)) // 16 bytes — too short
	assert.Error(t, err)
}
