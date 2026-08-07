package appdeveloper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/llm"
	"github.com/saas-agent-platform/backend/internal/store"
)

type AppDeveloperAgent struct {
	store         *store.Store
	daytonaClient *shared.DaytonaClient
	llmClient     *llm.LLMClient
}

func NewAppDeveloperAgent(s *store.Store, dc *shared.DaytonaClient) *AppDeveloperAgent {
	return &AppDeveloperAgent{
		store:         s,
		daytonaClient: dc,
		llmClient:     llm.NewLLMClient(),
	}
}

type CodegenResult struct {
	FilesGenerated int      `json:"files_generated"`
	FilePaths      []string `json:"file_paths"`
	Stack          string   `json:"stack"`
	Description    string   `json:"description"`
}

func (a *AppDeveloperAgent) ExecuteCodegenJob(ctx context.Context, jobID, projectID string, payload map[string]string) (*CodegenResult, error) {
	sb := a.daytonaClient.GetOrCreateSandbox(projectID)

	prompt := payload["prompt"]
	stack := payload["stack"]
	if stack == "" {
		stack = "Go 1.22 REST API"
	}

	// Strict Production Check: Fail job if LLM credentials are missing
	if !a.llmClient.HasCredentials() {
		errStr := "App Developer Agent failed: No LLM API credentials configured. Set OPENAI_API_KEY, GEMINI_API_KEY, or CUSTOM_OPENAI_BASE_URL to generate code."
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, fmt.Errorf("%s", errStr)
	}

	log.Printf("🤖 Invoking LLM API for dynamic code generation (Stack: %s, Prompt: %s)...", stack, prompt)
	files, err := a.llmClient.GenerateCodeFiles(ctx, prompt, stack)
	if err != nil || len(files) == 0 {
		errStr := fmt.Sprintf("App Developer Agent failed during LLM code synthesis: %v", err)
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, fmt.Errorf("%s", errStr)
	}

	description := fmt.Sprintf("Dynamically generated %d files via LLM API for stack '%s'", len(files), stack)

	paths := make([]string, 0, len(files))
	for p, content := range files {
		sb.WriteFile(p, content)
		paths = append(paths, p)
	}

	res := &CodegenResult{
		FilesGenerated: len(files),
		FilePaths:      paths,
		Stack:          stack,
		Description:    description,
	}

	resBytes, err := json.Marshal(res)
	if err != nil {
		errStr := err.Error()
		a.store.UpdateJob(jobID, "failed", nil, &errStr)
		return nil, err
	}

	_, err = a.store.UpdateJob(jobID, "succeeded", resBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to update job status: %w", err)
	}

	return res, nil
}
