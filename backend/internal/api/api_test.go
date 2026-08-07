package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/queue"
	"github.com/saas-agent-platform/backend/internal/store"
)

func setupTestServer() *Server {
	s := store.NewStore()
	dc := shared.NewDaytonaClient()
	jq := queue.NewRedisQueue("localhost:6379")
	return NewServer(s, dc, jq)
}

func TestSSRFValidation(t *testing.T) {
	// Forbidden internal IPs
	forbidden := []string{
		"http://127.0.0.1:8000/v1",
		"http://localhost:8000/v1",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/api",
		"http://192.168.1.1/api",
	}

	for _, u := range forbidden {
		_, err := ValidateURLForSSRF(u)
		if err == nil {
			t.Errorf("ValidateURLForSSRF should block forbidden internal URL: %s", u)
		}
	}

	// Allowed public domain format
	publicURL := "https://api.openai.com/v1"
	_, err := ValidateURLForSSRF(publicURL)
	if err != nil {
		t.Errorf("ValidateURLForSSRF should allow public URL %s, got error: %v", publicURL, err)
	}
}

func TestSignupAndLoginFlow(t *testing.T) {
	srv := setupTestServer()
	uniqueEmail := fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())

	// 1. Signup
	signupBody := map[string]string{
		"email":    uniqueEmail,
		"password": "Password123!",
	}
	b, _ := json.Marshal(signupBody)
	req := httptest.NewRequest("POST", "/api/v1/auth/signup", bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created on signup, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var res struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil || res.Token == "" {
		t.Fatalf("Expected valid token on signup")
	}

	// 2. Login with wrong password
	loginWrong := map[string]string{
		"email":    uniqueEmail,
		"password": "WrongPassword!",
	}
	bWrong, _ := json.Marshal(loginWrong)
	reqWrong := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(bWrong))
	reqWrong.Header.Set("Content-Type", "application/json")
	recWrong := httptest.NewRecorder()

	srv.Router().ServeHTTP(recWrong, reqWrong)
	if recWrong.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for incorrect password, got %d", recWrong.Code)
	}

	// 3. Login with correct password
	loginCorrect := map[string]string{
		"email":    uniqueEmail,
		"password": "Password123!",
	}
	bCorrect, _ := json.Marshal(loginCorrect)
	reqCorrect := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(bCorrect))
	reqCorrect.Header.Set("Content-Type", "application/json")
	recCorrect := httptest.NewRecorder()

	srv.Router().ServeHTTP(recCorrect, reqCorrect)
	if recCorrect.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for correct credentials, got %d", recCorrect.Code)
	}
}

func TestUnauthenticatedAccessBlocked(t *testing.T) {
	srv := setupTestServer()

	// Protected endpoint without token
	req := httptest.NewRequest("GET", "/api/v1/projects", nil)
	rec := httptest.NewRecorder()

	srv.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for unauthenticated request, got %d", rec.Code)
	}
}
