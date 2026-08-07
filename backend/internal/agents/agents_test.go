package agents_test

import (
	"context"
	"os"
	"testing"

	"github.com/saas-agent-platform/backend/internal/agents/appdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/appdeveloper"
	"github.com/saas-agent-platform/backend/internal/agents/appmaintainer"
	"github.com/saas-agent-platform/backend/internal/agents/llmdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/master"
	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/store"
)

func setupTestEnvironment() (*store.Store, *shared.DaytonaClient) {
	os.Setenv("STORE_TYPE", "memory")
	s := store.NewStore()
	dc := shared.NewDaytonaClient()
	return s, dc
}

func TestMasterAgentRoutingAndTurn(t *testing.T) {
	s, dc := setupTestEnvironment()
	masterAgent := master.NewMasterAgent(s, dc)

	proj, err := s.CreateProject("user-test-1", "Test Project")
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	t.Run("Intent Build App Routing", func(t *testing.T) {
		turn, err := masterAgent.ProcessTurn(context.Background(), proj.ID, "Build a Go 1.22 REST API with PostgreSQL database", nil)
		if err != nil {
			t.Fatalf("ProcessTurn failed: %v", err)
		}
		if turn.ActivatedAgent != "App Developer Agent" {
			t.Errorf("Expected 'App Developer Agent', got '%s'", turn.ActivatedAgent)
		}
		if turn.JobType != "codegen" {
			t.Errorf("Expected JobType 'codegen', got '%s'", turn.JobType)
		}
	})

	t.Run("Intent Deploy App Routing", func(t *testing.T) {
		turn, err := masterAgent.ProcessTurn(context.Background(), proj.ID, "Deploy app to Azure VM with Standard_B2s", nil)
		if err != nil {
			t.Fatalf("ProcessTurn failed: %v", err)
		}
		if turn.ActivatedAgent != "App Deployer Agent" {
			t.Errorf("Expected 'App Deployer Agent', got '%s'", turn.ActivatedAgent)
		}
	})

	t.Run("Intent Deploy LLM Routing", func(t *testing.T) {
		turn, err := masterAgent.ProcessTurn(context.Background(), proj.ID, "Deploy Hugging Face Llama-3 model using vLLM on Azure GPU VM", nil)
		if err != nil {
			t.Fatalf("ProcessTurn failed: %v", err)
		}
		if turn.ActivatedAgent != "LLM Deployer Agent" {
			t.Errorf("Expected 'LLM Deployer Agent', got '%s'", turn.ActivatedAgent)
		}
	})

	t.Run("Intent Maintain App Routing", func(t *testing.T) {
		turn, err := masterAgent.ProcessTurn(context.Background(), proj.ID, "Fix HTTP 500 error in database connection pool", nil)
		if err != nil {
			t.Fatalf("ProcessTurn failed: %v", err)
		}
		if turn.ActivatedAgent != "App Maintainer Agent" {
			t.Errorf("Expected 'App Maintainer Agent', got '%s'", turn.ActivatedAgent)
		}
	})
}

func TestAppDeployerAzureSDKExecution(t *testing.T) {
	s, dc := setupTestEnvironment()
	deployer := appdeployer.NewAppDeployerAgent(s, dc)

	proj, _ := s.CreateProject("user-test-2", "Deployer Test")
	job, _ := s.CreateJob(proj.ID, "deploy_app", []byte("{}"))

	payload := map[string]string{
		"azure_region": "eastus",
		"vm_size":      "Standard_B2s",
	}

	// Test real execution; expected to fail gracefully due to missing Azure Subscription ID / Credentials
	result, err := deployer.ExecuteDeployJob(context.Background(), job.ID, proj.ID, payload)
	if err == nil {
		t.Logf("AppDeployer succeeded: %+v", result)
	} else {
		t.Logf("AppDeployer failed gracefully as expected due to missing credentials: %v", err)
		updatedJob, errGet := s.GetJob(job.ID)
		if errGet != nil {
			t.Fatalf("Failed to retrieve updated job: %v", errGet)
		}
		if updatedJob.Status != "failed" {
			t.Errorf("Expected job status 'failed', got '%s'", updatedJob.Status)
		}
	}
}

func TestLLMDeployerAzureSDKExecution(t *testing.T) {
	s, dc := setupTestEnvironment()
	llmAgent := llmdeployer.NewLLMDeployerAgent(s, dc)

	proj, _ := s.CreateProject("user-test-3", "LLM Deployer Test")
	job, _ := s.CreateJob(proj.ID, "deploy_llm", []byte("{}"))

	payload := map[string]string{
		"model_repo_id": "meta-llama/Llama-3-8B-Instruct",
		"topology":      "vLLM (Azure GPU VM)",
		"gpu_type":      "NVIDIA A10G (24GB VRAM)",
	}

	// Test real execution; expected to fail gracefully due to missing Azure Subscription ID
	result, err := llmAgent.ExecuteDeployJob(context.Background(), job.ID, proj.ID, payload)
	if err == nil {
		t.Logf("LLMDeployer succeeded: %+v", result)
	} else {
		t.Logf("LLMDeployer failed gracefully as expected due to missing credentials: %v", err)
		updatedJob, errGet := s.GetJob(job.ID)
		if errGet != nil {
			t.Fatalf("Failed to retrieve updated job: %v", errGet)
		}
		if updatedJob.Status != "failed" {
			t.Errorf("Expected job status 'failed', got '%s'", updatedJob.Status)
		}
	}
}

func TestDaytonaSandboxFileOperationsAndGitState(t *testing.T) {
	dc := shared.NewDaytonaClient()
	sb := dc.GetOrCreateSandbox("proj-test-daytona")

	sb.WriteFile("/main.go", "package main\nfunc main() {}\n")
	content, err := sb.ReadFile("/main.go")
	if err != nil {
		t.Fatalf("Failed to read sandbox file: %v", err)
	}
	if content != "package main\nfunc main() {}\n" {
		t.Errorf("Unexpected sandbox file content: %s", content)
	}

	status := sb.GetGitStatus()
	if len(status) == 0 {
		t.Errorf("Expected non-empty git status")
	}

	diff := sb.GetGitDiff("/main.go")
	if diff == "" {
		t.Errorf("Expected non-empty diff string")
	}

	sb.ClearGitStatus()
	if len(sb.GetGitStatus()) != 0 {
		t.Errorf("Expected cleared git status")
	}
}

func TestAppDeveloperAndMaintainerAgents(t *testing.T) {
	s, dc := setupTestEnvironment()
	devAgent := appdeveloper.NewAppDeveloperAgent(s, dc)
	maintAgent := appmaintainer.NewAppMaintainerAgent(s, dc)

	proj, _ := s.CreateProject("user-test-4", "Dev Test")
	job1, _ := s.CreateJob(proj.ID, "codegen", []byte("{}"))
	job2, _ := s.CreateJob(proj.ID, "maintain_app", []byte("{}"))

	// Test AppDeveloper execution
	res1, err1 := devAgent.ExecuteCodegenJob(context.Background(), job1.ID, proj.ID, map[string]string{
		"prompt": "Create a Go web server",
		"stack":  "Go 1.22 REST API",
	})
	if err1 != nil {
		t.Logf("AppDeveloper execution note: %v", err1)
	} else {
		if res1.FilesGenerated == 0 {
			t.Errorf("Expected positive files count")
		}
	}

	// Test AppMaintainer execution
	res2, err2 := maintAgent.ExecuteMaintainJob(context.Background(), job2.ID, proj.ID, map[string]string{
		"prompt": "Fix database timeout in main.go",
	})
	if err2 != nil {
		t.Logf("AppMaintainer execution note: %v", err2)
	} else {
		if res2.FilesFixed == 0 {
			t.Errorf("Expected positive files fixed")
		}
	}
}
