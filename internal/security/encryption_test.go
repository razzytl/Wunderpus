package security

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("1234567890123456") // 16 bytes for AES-128
	plaintext := "Hello, World!"

	// Encrypt
	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Decrypt
	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptEmpty(t *testing.T) {
	key := []byte("1234567890123456") // 16 bytes for AES-128

	ciphertext, err := Encrypt("", key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != "" {
		t.Errorf("expected empty string, got %q", decrypted)
	}
}

func TestEncryptLongText(t *testing.T) {
	key := []byte("1234567890123456") // 16 bytes for AES-128
	plaintext := `This is a very long text that spans multiple lines 
	and contains various characters including numbers 12345 
	and symbols !@#$%^&*() for testing encryption.`

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptWrongKey(t *testing.T) {
	plaintext := "Hello, World!"

	key1 := []byte("1234567890123456") // 16 bytes
	key2 := []byte("abcdefghijklmnop") // 16 bytes different

	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Error("expected error with wrong key, got nil")
	}
}
