package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GoogleADKClient struct {
	apiKey string
	model  string
}

func NewGoogleADKClient() *GoogleADKClient {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
	}
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-1.5-pro"
	}
	return &GoogleADKClient{
		apiKey: apiKey,
		model:  model,
	}
}

func (g *GoogleADKClient) HasCredentials() bool {
	return g.apiKey != ""
}

func (g *GoogleADKClient) CompleteWithADK(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if !g.HasCredentials() {
		return "", fmt.Errorf("GEMINI_API_KEY / GOOGLE_API_KEY not configured")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(g.apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create Google GenAI ADK client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(g.model)
	model.SystemInstruction = genai.NewUserContent(genai.Text(systemPrompt))

	resp, err := model.GenerateContent(ctx, genai.Text(userPrompt))
	if err != nil {
		return "", fmt.Errorf("Google GenAI ADK Content Generation error: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("Google GenAI ADK returned empty response candidates")
	}

	var textResult string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			textResult += string(txt)
		}
	}

	return textResult, nil
}

func (g *GoogleADKClient) ExecuteADKToolCalling(ctx context.Context, prompt string) (*genai.FunctionCall, string, error) {
	if !g.HasCredentials() {
		return nil, "", fmt.Errorf("GEMINI_API_KEY not configured")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(g.apiKey))
	if err != nil {
		return nil, "", err
	}
	defer client.Close()

	model := client.GenerativeModel(g.model)

	// Official Google ADK Function Declarations & Tools
	model.Tools = []*genai.Tool{
		{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        "build_app",
					Description: "Generates a full-stack codebase in the Daytona sandbox based on prompt requirements and stack",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"prompt": {Type: genai.TypeString, Description: "Application requirements and description"},
							"stack":  {Type: genai.TypeString, Description: "Tech stack e.g. Go 1.22 REST API, React, Python FastAPI, Next.js"},
						},
						Required: []string{"prompt", "stack"},
					},
				},
				{
					Name:        "deploy_azure_app",
					Description: "Provisions Azure cloud infrastructure and deploys application to Azure VM",
					Parameters: &genai.Schema{
						Type: genai.TypeObject,
						Properties: map[string]*genai.Schema{
							"vm_size":      {Type: genai.TypeString, Description: "Azure VM size e.g. Standard_B2s"},
							"azure_region": {Type: genai.TypeString, Description: "Azure region e.g. eastus"},
						},
						Required: []string{"vm_size", "azure_region"},
					},
				},
			},
		},
	}

	session := model.StartChat()
	resp, err := session.SendMessage(ctx, genai.Text(prompt))
	if err != nil {
		return nil, "", err
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, "", fmt.Errorf("empty candidate response")
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if fnCall, ok := part.(genai.FunctionCall); ok {
			log.Printf("⚡ Official Google ADK Function Call Triggered: %s", fnCall.Name)
			return &fnCall, "", nil
		}
		if txt, ok := part.(genai.Text); ok {
			return nil, string(txt), nil
		}
	}

	return nil, "", nil
}

func (g *GoogleADKClient) GenerateADKStructuredFiles(ctx context.Context, prompt, stack string) (map[string]string, error) {
	if !g.HasCredentials() {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(g.apiKey))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	model := client.GenerativeModel(g.model)
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = &genai.Schema{
		Type:        genai.TypeObject,
		Description: "Map of file path to file content",
	}

	systemInstruction := fmt.Sprintf("You are an expert AI developer using Google ADK. Generate complete code files for stack '%s' and prompt: '%s'. Return a JSON object with file paths as keys and file contents as string values.", stack, prompt)
	model.SystemInstruction = genai.NewUserContent(genai.Text(systemInstruction))

	resp, err := model.GenerateContent(ctx, genai.Text("Generate files JSON"))
	if err != nil {
		return nil, err
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("empty candidates")
	}

	var jsonText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			jsonText += string(txt)
		}
	}

	cleaned := strings.TrimSpace(jsonText)
	var files map[string]string
	if err := json.Unmarshal([]byte(cleaned), &files); err != nil {
		return nil, fmt.Errorf("failed to parse Google ADK structured JSON response: %w", err)
	}

	return files, nil
}
