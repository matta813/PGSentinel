package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

type Cipher struct{ aead cipher.AEAD }

func NewCipher(secret string) (*Cipher, error) {
	if len(secret) < 16 {
		return nil, fmt.Errorf("encryption key must contain at least 16 characters")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}
func (c *Cipher) Encrypt(plaintext string) ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, []byte(plaintext), nil), nil
}
func (c *Cipher) Decrypt(ciphertext []byte) (string, error) {
	n := c.aead.NonceSize()
	if len(ciphertext) < n {
		return "", fmt.Errorf("invalid encrypted value")
	}
	plain, err := c.aead.Open(nil, ciphertext[:n], ciphertext[n:], nil)
	if err != nil {
		return "", fmt.Errorf("decrypt value: %w", err)
	}
	return string(plain), nil
}
