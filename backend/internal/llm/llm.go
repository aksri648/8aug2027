package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type LLMClient struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type ChatMessage struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type ChatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewLLMClient() *LLMClient {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = os.Getenv("CUSTOM_OPENAI_BASE_URL")
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("CUSTOM_OPENAI_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	model := os.Getenv("LLM_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}

	return &LLMClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *LLMClient) HasCredentials() bool {
	return c.apiKey != "" || strings.Contains(c.baseURL, "localhost") || strings.Contains(c.baseURL, "127.0.0.1")
}

func (c *LLMClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !c.HasCredentials() {
		return "", fmt.Errorf("no LLM API key or base URL configured")
	}

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	reqBody := ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.2,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	endpoint := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("LLM API returned HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if chatResp.Error != nil && chatResp.Error.Message != "" {
		return "", fmt.Errorf("LLM Error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response choices returned from LLM")
}

func (c *LLMClient) GenerateCodeFiles(ctx context.Context, prompt, stack string) (map[string]string, error) {
	systemPrompt := `You are an expert AI software architect and full-stack developer.
Given a user prompt and target tech stack, generate a complete, working codebase.
Your response MUST be a valid JSON object mapping file relative paths to file contents.
Do NOT surround with backticks or extra text. Output strictly raw JSON object format:
{
  "/path/to/file1.ext": "file content here",
  "/path/to/file2.ext": "file content here"
}`

	userPrompt := fmt.Sprintf("Target Tech Stack: %s\nUser Requirements:\n%s", stack, prompt)

	output, err := c.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	// Clean JSON string if LLM wrapped in ```json ... ```
	cleaned := strings.TrimSpace(output)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)

	var files map[string]string
	if err := json.Unmarshal([]byte(cleaned), &files); err != nil {
		return nil, fmt.Errorf("LLM generated invalid JSON file mapping: %w (raw response: %s)", err, output)
	}

	return files, nil
}

func (c *LLMClient) GenerateBugFix(ctx context.Context, prompt string, existingFiles map[string]string) (map[string]string, string, error) {
	systemPrompt := `You are an expert AI software engineer diagnosing and fixing codebase bugs.
Inspect the provided existing files and bug report. Apply targeted bug fixes and refactoring.
Your response MUST be a JSON object with two fields:
{
  "diagnosis": "Detailed explanation of the root cause and applied fix",
  "files": {
    "/file/path.ext": "updated complete file content"
  }
}`

	filesJSON, _ := json.Marshal(existingFiles)
	userPrompt := fmt.Sprintf("Bug Report / Maintenance Prompt: %s\n\nExisting Codebase Files:\n%s", prompt, string(filesJSON))

	output, err := c.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, "", err
	}

	cleaned := strings.TrimSpace(output)
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)

	var result struct {
		Diagnosis string            `json:"diagnosis"`
		Files     map[string]string `json:"files"`
	}

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, "", fmt.Errorf("LLM generated invalid JSON bug fix format: %w", err)
	}

	return result.Files, result.Diagnosis, nil
}
