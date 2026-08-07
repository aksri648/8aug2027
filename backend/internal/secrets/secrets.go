package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"
)

var (
	ErrDecryptionFailed = errors.New("failed to decrypt secret")
	masterKey           []byte
	once                sync.Once
)

func getMasterKey() []byte {
	once.Do(func() {
		keyStr := os.Getenv("SECRET_ENCRYPTION_KEY")
		if len(keyStr) < 32 {
			// Pad or generate deterministic 32-byte key for demo/dev
			padded := make([]byte, 32)
			copy(padded, []byte("saas-platform-encryption-key-32b!!"+keyStr))
			masterKey = padded
		} else {
			masterKey = []byte(keyStr[:32])
		}
	})
	return masterKey
}

func EncryptSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := getMasterKey()
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

func DecryptSecret(ciphertextStr string) (string, error) {
	if ciphertextStr == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		// If decoding fails, it might be stored plaintext (migration backward compatibility)
		return ciphertextStr, nil
	}

	key := getMasterKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return ciphertextStr, nil
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ciphertextStr, nil
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return ciphertextStr, nil
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// Fallback to raw value if not encrypted
		return ciphertextStr, nil
	}

	return string(plaintext), nil
}
