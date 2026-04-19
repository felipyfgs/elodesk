package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidKEK       = errors.New("invalid KEK")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

type Cipher struct {
	aead cipher.AEAD
}

func NewCipher(kekBase64 string) (*Cipher, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(kekBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: must be valid base64: %w", ErrInvalidKEK, err)
	}
	if len(keyBytes) < 32 {
		return nil, fmt.Errorf("%w: must decode to at least 32 bytes (got %d)", ErrInvalidKEK, len(keyBytes))
	}
	block, err := aes.NewCipher(keyBytes[:32])
	if err != nil {
		return nil, fmt.Errorf("init aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}
	return &Cipher{aead: gcm}, nil
}

// Encrypt encrypts plaintext with AES-256-GCM and returns base64(nonce || ciphertext).
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read nonce: %w", err)
	}
	ct := c.aead.Seal(nil, nonce, []byte(plaintext), nil)
	out := make([]byte, 0, len(nonce)+len(ct))
	out = append(out, nonce...)
	out = append(out, ct...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt reverses Encrypt.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: base64 decode: %w", ErrInvalidCiphertext, err)
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("%w: ciphertext shorter than nonce size", ErrInvalidCiphertext)
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("%w: open: %w", ErrInvalidCiphertext, err)
	}
	return string(pt), nil
}

// HashLookup produces a deterministic SHA-256 hex digest used as a lookup key
// for values whose plaintext we never want to store (e.g. api_access_token).
func HashLookup(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
