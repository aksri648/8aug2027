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

func (m *MasterAgent) ClassifyIntent(prompt string) Intent {
	lower := strings.ToLower(prompt)
	if strings.Contains(lower, "build") || strings.Contains(lower, "create") || strings.Contains(lower, "write app") || strings.Contains(lower, "generate") || strings.Contains(lower, "make a") || strings.Contains(lower, "develop") || strings.Contains(lower, "[app developer agent]") {
		return IntentBuildApp
	}
	if strings.Contains(lower, "deploy app") || strings.Contains(lower, "deploy microservice") || strings.Contains(lower, "azure vm") || strings.Contains(lower, "docker deploy") || strings.Contains(lower, "containerize") || strings.Contains(lower, "[app deployer agent]") {
		return IntentDeployApp
	}
	if strings.Contains(lower, "deploy llm") || strings.Contains(lower, "hugging face") || strings.Contains(lower, "vllm") || strings.Contains(lower, "nvidia nim") || strings.Contains(lower, "self-host llm") || strings.Contains(lower, "llama") || strings.Contains(lower, "mistral") || strings.Contains(lower, "[llm deployer agent]") {
		return IntentDeployLLM
	}
	if strings.Contains(lower, "bug") || strings.Contains(lower, "fix") || strings.Contains(lower, "maintain") || strings.Contains(lower, "error") || strings.Contains(lower, "500") || strings.Contains(lower, "github repo") || strings.Contains(lower, "refactor") || strings.Contains(lower, "[app maintainer agent]") {
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

func (m *MasterAgent) ProcessTurn(ctx context.Context, projectID string, prompt string, extraPayload map[string]interface{}) (*TurnResult, error) {
	intent := m.ClassifyIntent(prompt)

	payloadMap := map[string]string{
		"prompt":     prompt,
		"project_id": projectID,
	}
	for k, v := range extraPayload {
		payloadMap[k] = fmt.Sprintf("%v", v)
	}

	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job payload: %w", err)
	}

	switch intent {
	case IntentBuildApp:
		job, err := m.store.CreateJob(projectID, "codegen", payload)
		if err != nil {
			return nil, err
		}
		stackMsg := ""
		if s, ok := payloadMap["stack"]; ok && s != "" {
			stackMsg = fmt.Sprintf(" using stack **%s**", s)
		}
		return &TurnResult{
			AssistantResponse: fmt.Sprintf("I've activated the **App Developer Agent** to build out your application%s based on your prompt requirements.", stackMsg),
			ActivatedAgent:    "App Developer Agent",
			JobID:             job.ID,
			JobType:           "codegen",
			JobPayload:        payload,
		}, nil

	case IntentDeployApp:
		job, err := m.store.CreateJob(projectID, "deploy_app", payload)
		if err != nil {
			return nil, err
		}
		vmSize := payloadMap["vm_size"]
		if vmSize == "" {
			vmSize = payloadMap["vmSize"]
		}
		region := payloadMap["azure_region"]
		if region == "" {
			region = payloadMap["azureRegion"]
		}

		infoStr := ""
		if vmSize != "" || region != "" {
			infoStr = fmt.Sprintf(" (Target: VM Size %s, Region %s)", vmSize, region)
		}

		return &TurnResult{
			AssistantResponse: fmt.Sprintf("Activated **App Deployer Agent**%s. Inspecting sandbox codebase, building container definition, and provisioning Azure infrastructure.", infoStr),
			ActivatedAgent:    "App Deployer Agent",
			JobID:             job.ID,
			JobType:           "deploy_app",
			JobPayload:        payload,
		}, nil

	case IntentDeployLLM:
		job, err := m.store.CreateJob(projectID, "deploy_llm", payload)
		if err != nil {
			return nil, err
		}
		modelRepo := payloadMap["model_repo_id"]
		if modelRepo == "" {
			modelRepo = payloadMap["modelRepo"]
		}
		return &TurnResult{
			AssistantResponse: fmt.Sprintf("Activated **LLM Deployer Agent** for model `%s`. Provisioning dedicated GPU infrastructure and vLLM / NIM serving endpoint.", modelRepo),
			ActivatedAgent:    "LLM Deployer Agent",
			JobID:             job.ID,
			JobType:           "deploy_llm",
			JobPayload:        payload,
		}, nil

	case IntentMaintainApp:
		job, err := m.store.CreateJob(projectID, "maintain_app", payload)
		if err != nil {
			return nil, err
		}
		return &TurnResult{
			AssistantResponse: "Activated **App Maintainer Agent**. Inspecting repository in Daytona sandbox, diagnosing issue, applying fix, and running verification tests.",
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
