package master

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/llm"
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
	llmClient     *llm.LLMClient
}

func NewMasterAgent(s *store.Store, dc *shared.DaytonaClient) *MasterAgent {
	return &MasterAgent{
		store:         s,
		daytonaClient: dc,
		llmClient:     llm.NewLLMClient(),
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

func (m *MasterAgent) ClassifyIntentFallback(prompt string) Intent {
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

type LLMPlanAndRouting struct {
	AssistantResponse string   `json:"assistant_response"`
	Action            string   `json:"action"` // build_app, deploy_app, deploy_llm, maintain_app, general
	PlanSteps         []string `json:"plan_steps,omitempty"`
}

func (m *MasterAgent) ProcessTurn(ctx context.Context, projectID string, prompt string, extraPayload map[string]interface{}) (*TurnResult, error) {
	intent := m.ClassifyIntentFallback(prompt)
	assistantMsg := ""

	// Dynamic LLM Planning & Routing
	if m.llmClient.HasCredentials() {
		log.Printf("🧠 Invoking Master Agent LLM Planning & Routing for prompt: '%s'...", prompt)
		systemPrompt := `You are the Lead Master AI Architect & Orchestrator of an Autonomous SaaS Development Platform.
Analyze the user request and generate a structured developer plan and routing action.
Your response MUST be a JSON object in this exact schema:
{
  "assistant_response": "Your natural language response to the user explaining the architecture, plan, and agent activation",
  "action": "build_app" | "deploy_app" | "deploy_llm" | "maintain_app" | "general",
  "plan_steps": ["Step 1: ...", "Step 2: ..."]
}`
		userPrompt := fmt.Sprintf("User Request: %s\nProject ID: %s", prompt, projectID)

		llmOut, err := m.llmClient.Complete(ctx, systemPrompt, userPrompt)
		if err == nil {
			cleaned := strings.TrimSpace(llmOut)
			if strings.HasPrefix(cleaned, "```json") {
				cleaned = strings.TrimPrefix(cleaned, "```json")
				cleaned = strings.TrimSuffix(cleaned, "```")
			} else if strings.HasPrefix(cleaned, "```") {
				cleaned = strings.TrimPrefix(cleaned, "```")
				cleaned = strings.TrimSuffix(cleaned, "```")
			}
			cleaned = strings.TrimSpace(cleaned)

			var plan LLMPlanAndRouting
			if json.Unmarshal([]byte(cleaned), &plan) == nil && plan.Action != "" {
				assistantMsg = plan.AssistantResponse
				switch plan.Action {
				case "build_app":
					intent = IntentBuildApp
				case "deploy_app":
					intent = IntentDeployApp
				case "deploy_llm":
					intent = IntentDeployLLM
				case "maintain_app":
					intent = IntentMaintainApp
				case "general":
					intent = IntentGeneralOther
				}
			}
		} else {
			log.Printf("⚠️ Master Agent LLM planning fallback (%v). Using rule-based intent router.", err)
		}
	}

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
		if assistantMsg == "" {
			stackMsg := ""
			if s, ok := payloadMap["stack"]; ok && s != "" {
				stackMsg = fmt.Sprintf(" using stack **%s**", s)
			}
			assistantMsg = fmt.Sprintf("I've activated the **App Developer Agent** to build out your application%s based on your prompt requirements.", stackMsg)
		}
		return &TurnResult{
			AssistantResponse: assistantMsg,
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
		if assistantMsg == "" {
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
			assistantMsg = fmt.Sprintf("Activated **App Deployer Agent**%s. Inspecting sandbox codebase, building container definition, and provisioning Azure infrastructure.", infoStr)
		}
		return &TurnResult{
			AssistantResponse: assistantMsg,
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
		if assistantMsg == "" {
			modelRepo := payloadMap["model_repo_id"]
			if modelRepo == "" {
				modelRepo = payloadMap["modelRepo"]
			}
			assistantMsg = fmt.Sprintf("Activated **LLM Deployer Agent** for model `%s`. Provisioning dedicated GPU infrastructure and vLLM / NIM serving endpoint.", modelRepo)
		}
		return &TurnResult{
			AssistantResponse: assistantMsg,
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
		if assistantMsg == "" {
			assistantMsg = "Activated **App Maintainer Agent**. Inspecting repository in Daytona sandbox, diagnosing issue, applying fix, and running verification tests."
		}
		return &TurnResult{
			AssistantResponse: assistantMsg,
			ActivatedAgent:    "App Maintainer Agent",
			JobID:             job.ID,
			JobType:           "maintain_app",
			JobPayload:        payload,
		}, nil

	default:
		if assistantMsg == "" {
			assistantMsg = fmt.Sprintf("I am your AI SaaS Development Assistant powered by Google ADK and Daytona Sandboxes. You asked: \"%s\". How can I assist you with building code, provisioning Azure infrastructure, deploying open-weight LLMs, or diagnosing bugs?", prompt)
		}
		return &TurnResult{
			AssistantResponse: assistantMsg,
			ActivatedAgent:    "Master Agent",
		}, nil
	}
}
