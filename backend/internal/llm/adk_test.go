package llm

import (
	"testing"

	"github.com/google/generative-ai-go/genai"
)

func TestGoogleADKClientCredentialsCheck(t *testing.T) {
	client := NewGoogleADKClient()
	if client.HasCredentials() != (client.apiKey != "") {
		t.Errorf("Mismatch in HasCredentials check")
	}
}

func TestADKFunctionCallStructure(t *testing.T) {
	fnCall := &genai.FunctionCall{
		Name: "build_app",
		Args: map[string]interface{}{
			"prompt": "Build a Go API",
			"stack":  "Go 1.22 REST API",
		},
	}

	if fnCall.Name != "build_app" {
		t.Errorf("Expected function name build_app, got %s", fnCall.Name)
	}

	if fnCall.Args["prompt"] != "Build a Go API" {
		t.Errorf("Expected prompt 'Build a Go API', got %v", fnCall.Args["prompt"])
	}
}
