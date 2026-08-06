package main

import (
	"context"
	"log"
	"os"

	"github.com/saas-agent-platform/backend/internal/agents/appdeveloper"
	"github.com/saas-agent-platform/backend/internal/agents/appdeployer"
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

		payload := make(map[string]string)
		// Process job per type
		switch job.Type {
		case "codegen":
			_, err := appDevAgent.ExecuteCodegenJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				log.Printf("❌ Codegen job failed: %v", err)
			} else {
				log.Printf("✅ Codegen job completed for %s", job.ProjectID)
			}
		case "deploy_app":
			_, err := appDepAgent.ExecuteDeployJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				log.Printf("❌ Deploy app job failed: %v", err)
			} else {
				log.Printf("✅ Deploy app job completed for %s", job.ProjectID)
			}
		case "deploy_llm":
			_, err := llmDepAgent.ExecuteDeployJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				log.Printf("❌ Deploy LLM job failed: %v", err)
			} else {
				log.Printf("✅ Deploy LLM job completed for %s", job.ProjectID)
			}
		case "maintain_app":
			_, err := appMaintAgent.ExecuteMaintainJob(ctx, job.ID, job.ProjectID, payload)
			if err != nil {
				log.Printf("❌ Maintain app job failed: %v", err)
			} else {
				log.Printf("✅ Maintain app job completed for %s", job.ProjectID)
			}
		}
	}
}
