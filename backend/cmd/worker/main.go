package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/saas-agent-platform/backend/internal/agents/appdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/appdeveloper"
	"github.com/saas-agent-platform/backend/internal/agents/appmaintainer"
	"github.com/saas-agent-platform/backend/internal/agents/llmdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/queue"
	"github.com/saas-agent-platform/backend/internal/store"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	appStore := store.NewStore()
	daytonaClient := shared.NewDaytonaClient()
	jobQueue := queue.NewRedisQueue(redisAddr)

	appDevAgent := appdeveloper.NewAppDeveloperAgent(appStore, daytonaClient)
	appDepAgent := appdeployer.NewAppDeployerAgent(appStore, daytonaClient)
	llmDepAgent := llmdeployer.NewLLMDeployerAgent(appStore, daytonaClient)
	appMaintAgent := appmaintainer.NewAppMaintainerAgent(appStore, daytonaClient)

	log.Printf("⚡ SaaS Agent Platform Async Worker listening to Redis Queue at %s...", redisAddr)

	jobsChan := jobQueue.DequeueJobs()
	ctx := context.Background()

	for job := range jobsChan {
		log.Printf("⚙️ [Redis Worker] Processing job %s (Type: %s, Project: %s)", job.ID, job.Type, job.ProjectID)

		// Set job status to running
		appStore.UpdateJob(job.ID, "running", nil, nil)

		payload := make(map[string]string)
		if len(job.Payload) > 0 {
			_ = json.Unmarshal(job.Payload, &payload)
		}

		var jobErr error

		switch job.Type {
		case "codegen":
			_, err := appDevAgent.ExecuteCodegenJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				jobErr = err
			}
		case "deploy_app":
			_, err := appDepAgent.ExecuteDeployJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				jobErr = err
			}
		case "deploy_llm":
			_, err := llmDepAgent.ExecuteDeployJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				jobErr = err
			}
		case "maintain_app":
			_, err := appMaintAgent.ExecuteMaintainJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				jobErr = err
			}
		case "push":
			sb := daytonaClient.GetOrCreateSandbox(job.ProjectID)
			sb.ClearGitStatus()
			appStore.UpdateJob(job.ID, "succeeded", nil, nil)
		default:
			jobErr = fmt.Errorf("unknown job type: %s", job.Type)
		}

		if jobErr != nil {
			errStr := jobErr.Error()
			log.Printf("❌ [Worker] Job %s failed: %v", job.ID, jobErr)
			appStore.UpdateJob(job.ID, "failed", nil, &errStr)
			jobQueue.EnqueueDLQ(ctx, job, errStr)
		} else {
			log.Printf("✅ [Worker] Job %s completed cleanly for project %s", job.ID, job.ProjectID)
		}
	}
}
