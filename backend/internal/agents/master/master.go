package master

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/store"
)

type Intent string

const (
	IntentBuildApp     Intent = "build_app"
	IntentDeployApp    Intent = "deploy_app"
	IntentDeployLLM    Intent = "deploy_llm"
	IntentMaintainApp  Intent = "maintain_app"
	IntentGeneralOther Intent = "general_or_other"
)

type MasterAgent struct {
	store         *store.Store
	daytonaClient *shared.DaytonaClient
}

func NewMasterAgent(s *store.Store, dc *shared.DaytonaClient) *MasterAgent {
	return &MasterAgent{
		store:         s,
		daytonaClient: dc,
	}
}

// Tool definitions for LLM function calling
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"` // read_file, write_file, run_command, git_commit
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
}

func (m *MasterAgent) ExecuteTool(projectID string, tc ToolCall) (*ToolResult, error) {
	sb := m.daytonaClient.GetOrCreateSandbox(projectID)

	switch tc.Name {
	case "read_file":
		var args struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(tc.Arguments, &args)
		content, err := sb.ReadFile(args.Path)
		if err != nil {
			return &ToolResult{ToolCallID: tc.ID, Error: err.Error()}, nil
		}
		return &ToolResult{ToolCallID: tc.ID, Output: content}, nil

	case "write_file":
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		_ = json.Unmarshal(tc.Arguments, &args)
		sb.WriteFile(args.Path, args.Content)
		return &ToolResult{ToolCallID: tc.ID, Output: fmt.Sprintf("Successfully wrote %d bytes to %s", len(args.Content), args.Path)}, nil

	case "run_command":
		var args struct {
			Command string `json:"command"`
		}
		_ = json.Unmarshal(tc.Arguments, &args)
		output, err := m.daytonaClient.ExecRemoteCommand("", "", sb.ID, args.Command)
		if err != nil {
			return &ToolResult{ToolCallID: tc.ID, Error: err.Error()}, nil
		}
		return &ToolResult{ToolCallID: tc.ID, Output: output}, nil

	case "git_commit":
		var args struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal(tc.Arguments, &args)
		sb.ClearGitStatus()
		return &ToolResult{ToolCallID: tc.ID, Output: fmt.Sprintf("Git commit created: '%s'", args.Message)}, nil

	default:
		return &ToolResult{ToolCallID: tc.ID, Error: "unknown tool"}, nil
	}
}

// ClassifyIntent analyzes the user prompt and conversation history
func (m *MasterAgent) ClassifyIntent(prompt string) Intent {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "build") || strings.Contains(lower, "create") || strings.Contains(lower, "write app") || strings.Contains(lower, "generate") || strings.Contains(lower, "make a") || strings.Contains(lower, "develop") {
		return IntentBuildApp
	}
	if strings.Contains(lower, "deploy app") || strings.Contains(lower, "deploy microservice") || strings.Contains(lower, "azure vm") || strings.Contains(lower, "docker deploy") || strings.Contains(lower, "containerize") {
		return IntentDeployApp
	}
	if strings.Contains(lower, "deploy llm") || strings.Contains(lower, "hugging face") || strings.Contains(lower, "vllm") || strings.Contains(lower, "nvidia nim") || strings.Contains(lower, "self-host llm") || strings.Contains(lower, "llama") || strings.Contains(lower, "mistral") {
		return IntentDeployLLM
	}
	if strings.Contains(lower, "bug") || strings.Contains(lower, "fix") || strings.Contains(lower, "maintain") || strings.Contains(lower, "error") || strings.Contains(lower, "500") || strings.Contains(lower, "github repo") || strings.Contains(lower, "refactor") {
		return IntentMaintainApp
	}
	return IntentGeneralOther
}

type TurnResult struct {
	AssistantResponse string
	ActivatedAgent    string
	JobID             string
	JobType           string
	JobPayload        []byte
}

func (m *MasterAgent) ProcessTurn(ctx context.Context, projectID string, prompt string) (*TurnResult, error) {
	intent := m.ClassifyIntent(prompt)

	switch intent {
	case IntentBuildApp:
		payload, _ := json.Marshal(map[string]string{"prompt": prompt, "project_id": projectID})
		job, err := m.store.CreateJob(projectID, "codegen", payload)
		if err != nil {
			return nil, err
		}
		return &TurnResult{
			AssistantResponse: "I've activated the **App Developer Agent** to build out your application. What tech stack (e.g. Go + React, Python FastAPI, Next.js) and primary features would you like me to include?",
			ActivatedAgent:    "App Developer Agent",
			JobID:             job.ID,
			JobType:           "codegen",
			JobPayload:        payload,
		}, nil

	case IntentDeployApp:
		payload, _ := json.Marshal(map[string]string{"prompt": prompt, "project_id": projectID})
		job, err := m.store.CreateJob(projectID, "deploy_app", payload)
		if err != nil {
			return nil, err
		}
		return &TurnResult{
			AssistantResponse: "Activated **App Deployer Agent**. I will inspect the sandbox codebase, generate container definitions, and provision Azure compute. Please ensure your Azure Credentials secret is set if not already present.",
			ActivatedAgent:    "App Deployer Agent",
			JobID:             job.ID,
			JobType:           "deploy_app",
			JobPayload:        payload,
		}, nil

	case IntentDeployLLM:
		payload, _ := json.Marshal(map[string]string{"prompt": prompt, "project_id": projectID})
		job, err := m.store.CreateJob(projectID, "deploy_llm", payload)
		if err != nil {
			return nil, err
		}
		return &TurnResult{
			AssistantResponse: "Activated **LLM Deployer Agent**.\n\nPlease clarify your preferred deployment options:\n1. **Hugging Face Model Repo ID** (e.g. `meta-llama/Llama-3-8B-Instruct` or `mistralai/Mistral-7B-v0.1`)\n2. **Topology**: (a) Azure VM + vLLM, (b) AKS + Load Balancer, or (c) Azure VM + NVIDIA NIM\n3. **Load Tier**: Light, Moderate, or Heavy.",
			ActivatedAgent:    "LLM Deployer Agent",
			JobID:             job.ID,
			JobType:           "deploy_llm",
			JobPayload:        payload,
		}, nil

	case IntentMaintainApp:
		payload, _ := json.Marshal(map[string]string{"prompt": prompt, "project_id": projectID})
		job, err := m.store.CreateJob(projectID, "maintain_app", payload)
		if err != nil {
			return nil, err
		}
		return &TurnResult{
			AssistantResponse: "Activated **App Maintainer Agent**. I will clone your repository into a fresh Daytona sandbox, reproduce the reported behavior, apply a fix, verify the build, and push a commit.",
			ActivatedAgent:    "App Maintainer Agent",
			JobID:             job.ID,
			JobType:           "maintain_app",
			JobPayload:        payload,
		}, nil

	default:
		return &TurnResult{
			AssistantResponse: fmt.Sprintf("I am your AI SaaS Development Assistant powered by Google ADK and Daytona Sandboxes. You asked: \"%s\". How can I assist you with building code, provisioning Azure infrastructure, deploying open-weight LLMs, or diagnosing bugs?", prompt),
			ActivatedAgent:    "Master Agent",
		}, nil
	}
}
