package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

// MinSaltSize is the minimum recommended salt size in bytes (16 bytes = 128 bits)
const MinSaltSize = 16

// GenerateSalt generates a cryptographically secure random salt of the specified size.
func GenerateSalt(size int) ([]byte, error) {
	if size < MinSaltSize {
		size = MinSaltSize
	}
	salt := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	return salt, nil
}

// GenerateSaltString generates a cryptographically secure random salt and returns it as a base64 string.
func GenerateSaltString() (string, error) {
	salt, err := GenerateSalt(MinSaltSize)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(salt), nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
func Encrypt(plaintext string, key []byte) (string, error) {
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

// Decrypt decrypts a base64 encoded ciphertext using AES-256-GCM.
func Decrypt(cryptoText string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(cryptoText)
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

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// DeriveKey derives a 32-byte key from a passphrase and salt using Argon2id.
// The salt parameter must be at least MinSaltSize (16) bytes.
// Returns an error if the salt is too small or invalid.
func DeriveKey(passphrase string, salt []byte) ([]byte, error) {
	if len(salt) < MinSaltSize {
		return nil, errors.New("salt must be at least 16 bytes for secure key derivation")
	}
	// Recommended parameters: time=3, memory=64MB, threads=4, keyLen=32
	return argon2.IDKey([]byte(passphrase), salt, 3, 64*1024, 4, 32), nil
}

// DeriveKeyFromBase64 derives a key from a passphrase and a base64-encoded salt string.
// This is a convenience wrapper around DeriveKey for use with config values.
func DeriveKeyFromBase64(passphrase string, saltBase64 string) ([]byte, error) {
	if saltBase64 == "" {
		return nil, errors.New("salt is required for key derivation (found empty string)")
	}

	salt, err := base64.StdEncoding.DecodeString(saltBase64)
	if err != nil {
		return nil, errors.New("invalid salt: must be valid base64")
	}

	return DeriveKey(passphrase, salt)
}
