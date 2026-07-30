package ai

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
)

var errEncryptionKey = errors.New("encryption key error")

const keyFile = ".ai_key_encryption"

// encryptionKey loads or generates a 256-bit AES key stored alongside the config.
// The key file is stored in the same directory as the config YAML for consistency.
func encryptionKey(configDir string) ([]byte, error) {
	if configDir == "" {
		return nil, errEncryptionKey
	}
	kp := filepath.Join(configDir, keyFile)
	data, err := os.ReadFile(kp)
	if err == nil && len(data) == 32 {
		return data, nil
	}

	// Generate a new key
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(kp, key, 0600); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptAPIKey encrypts a plaintext API key using AES-256-GCM.
// Returns a base64-encoded ciphertext string.
func EncryptAPIKey(plaintext, configDir string) (string, error) {
	key, err := encryptionKey(configDir)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAPIKey decrypts a base64-encoded ciphertext back to the plaintext API key.
func DecryptAPIKey(cipherB64, configDir string) (string, error) {
	key, err := encryptionKey(configDir)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(cipherB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
