package store_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/saas-agent-platform/backend/internal/store"
)

func TestStoreProjectAndJobOperations(t *testing.T) {
	os.Setenv("STORE_TYPE", "memory")
	s := store.NewStore()

	uniqueEmail := fmt.Sprintf("test_store_%d@example.com", time.Now().UnixNano())

	// 1. User Creation & Retrieval
	user, err := s.CreateUser(uniqueEmail, "hashed_password")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	fetchedUser, err := s.GetUserByEmail(uniqueEmail)
	if err != nil || fetchedUser.ID != user.ID {
		t.Fatalf("Failed to fetch user by email: %v", err)
	}

	// 2. Project Creation & Retrieval
	proj, err := s.CreateProject(user.ID, "Store Test Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	projects, err := s.ListProjects(user.ID)
	if err != nil || len(projects) == 0 {
		t.Fatalf("Failed to list projects for user: %v", err)
	}

	// 3. Job Creation & Status Updates
	job, err := s.CreateJob(proj.ID, "codegen", []byte(`{"prompt":"build api"}`))
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}

	if job.Status != "queued" {
		t.Errorf("Expected initial status 'queued', got '%s'", job.Status)
	}

	resData := []byte(`{"status":"success"}`)
	updatedJob, err := s.UpdateJob(job.ID, "succeeded", resData, nil)
	if err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}
	if updatedJob.Status != "succeeded" {
		t.Errorf("Expected updated status 'succeeded', got '%s'", updatedJob.Status)
	}

	// 4. Secrets Management & Presence
	err = s.SaveSecret(proj.ID, "github_pat", "ghp_mocktoken12345")
	if err != nil {
		t.Fatalf("Failed to save secret: %v", err)
	}

	presence := s.GetSecretsPresence(proj.ID)
	if !presence.GitHubPAT {
		t.Errorf("Expected GitHubPAT presence to be true")
	}

	secVal, err := s.GetSecretValue(proj.ID, "github_pat")
	if err != nil || secVal != "ghp_mocktoken12345" {
		t.Fatalf("Failed to retrieve decrypted secret value: %v", err)
	}
}
