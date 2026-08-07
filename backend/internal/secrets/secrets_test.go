package secrets

import (
	"testing"
)

func TestSecretEncryption(t *testing.T) {
	plaintext := "ghp_1234567890abcdefghijklmnopqrstuvwxyz"
	encrypted, err := EncryptSecret(plaintext)
	if err != nil {
		t.Fatalf("EncryptSecret failed: %v", err)
	}

	if encrypted == plaintext {
		t.Fatalf("Encrypted secret should not equal plaintext")
	}

	decrypted, err := DecryptSecret(encrypted)
	if err != nil {
		t.Fatalf("DecryptSecret failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected decrypted %s, got %s", plaintext, decrypted)
	}
}
