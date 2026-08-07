package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLLMClientCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "{\"\\/main.go\": \"package main\\nfunc main() {}\"}"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	client := &LLMClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		model:      "test-model",
		httpClient: server.Client(),
	}

	files, err := client.GenerateCodeFiles(context.Background(), "Build a Go service", "Go 1.22 REST API")
	if err != nil {
		t.Fatalf("GenerateCodeFiles failed: %v", err)
	}

	if len(files) == 0 {
		t.Fatalf("Expected generated files, got empty map")
	}

	if _, ok := files["/main.go"]; !ok {
		t.Errorf("Expected /main.go in generated files")
	}
}
