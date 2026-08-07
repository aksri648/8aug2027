package auth

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	password := "SecretPass123!"
	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hashed == password {
		t.Fatalf("Hashed password should not equal raw password")
	}

	if !CheckPasswordHash(password, hashed) {
		t.Fatalf("CheckPasswordHash should return true for correct password")
	}

	if CheckPasswordHash("WrongPass123!", hashed) {
		t.Fatalf("CheckPasswordHash should return false for incorrect password")
	}
}

func TestJWTTokenValidation(t *testing.T) {
	userID := "user-12345"
	email := "test@example.com"

	token, err := GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}

	// Test invalid token
	_, err = ValidateToken("invalid-token-string")
	if err == nil {
		t.Errorf("ValidateToken should fail on invalid token string")
	}
}
