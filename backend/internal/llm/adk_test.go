package llm

import (
	"testing"

	"github.com/google/generative-ai-go/genai"
)

func TestADKFunctionArgsToMap(t *testing.T) {
	fnCall := &genai.FunctionCall{
		Name: "build_app",
		Args: map[string]interface{}{
			"prompt": "Build a Go API",
			"stack":  "Go 1.22 REST API",
		},
	}

	m := ADKFunctionArgsToMap(fnCall)
	if m["prompt"] != "Build a Go API" {
		t.Errorf("Expected prompt 'Build a Go API', got %v", m["prompt"])
	}
	if m["stack"] != "Go 1.22 REST API" {
		t.Errorf("Expected stack 'Go 1.22 REST API', got %v", m["stack"])
	}
}

func TestGoogleADKClientCredentialsCheck(t *testing.T) {
	client := NewGoogleADKClient()
	// HasCredentials returns false when GEMINI_API_KEY / GOOGLE_API_KEY is not set
	if client.HasCredentials() != (client.apiKey != "") {
		t.Errorf("Mismatch in HasCredentials check")
	}
}
