package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

func Encrypt(plaintext, key []byte) ([]byte, error) {
	normalized, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(normalized)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := gcm.Seal(nonce, nonce, plaintext, nil)
	return sealed, nil
}

func Decrypt(ciphertext, key []byte) ([]byte, error) {
	normalized, err := NormalizeKey(key)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(normalized)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, payload := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	return gcm.Open(nil, nonce, payload, nil)
}

func NormalizeKey(key []byte) ([]byte, error) {
	switch len(key) {
	case 16, 24, 32:
		copied := append([]byte(nil), key...)
		return copied, nil
	default:
		return nil, fmt.Errorf("key length must be 16, 24, or 32 bytes")
	}
}
