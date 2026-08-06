package appdeveloper

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/store"
)

type AppDeveloperAgent struct {
	store         *store.Store
	daytonaClient *shared.DaytonaClient
}

func NewAppDeveloperAgent(s *store.Store, dc *shared.DaytonaClient) *AppDeveloperAgent {
	return &AppDeveloperAgent{
		store:         s,
		daytonaClient: dc,
	}
}

type CodegenResult struct {
	FilesGenerated int      `json:"files_generated"`
	FilePaths      []string `json:"file_paths"`
	Stack          string   `json:"stack"`
}

func (a *AppDeveloperAgent) ExecuteCodegenJob(ctx context.Context, jobID, projectID string, payload map[string]string) (*CodegenResult, error) {
	sb := a.daytonaClient.GetOrCreateSandbox(projectID)

	prompt := payload["prompt"]
	_ = prompt

	// Generate files into sandbox
	files := map[string]string{
		"/cmd/api/main.go": `package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, ` + "`" + `{"status":"ok","timestamp":"%s"}` + "`" + `, time.Now().Format(time.RFC3339))
	})

	http.HandleFunc("/api/v1/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, ` + "`" + `{"items":[{"id":"1","name":"Microservice Alpha"},{"id":"2","name":"Microservice Beta"}]}` + "`" + `)
	})

	log.Println("Starting service on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}`,
		"/go.mod": "module github.com/user/app\n\ngo 1.22",
		"/Dockerfile": `FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go build -o server ./cmd/api/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]`,
		"/README.md": fmt.Sprintf("# Generated Application\nGenerated on %s by App Developer Agent.\n\n## Running\n```bash\ngo run ./cmd/api/main.go\n```", time.Now().Format(time.RFC3339)),
		"/config.json": `{"env": "production", "port": 8080, "log_level": "info"}`,
	}

	paths := make([]string, 0, len(files))
	for p, content := range files {
		sb.WriteFile(p, content)
		paths = append(paths, p)
	}

	res := &CodegenResult{
		FilesGenerated: len(files),
		FilePaths:      paths,
		Stack:          "Go 1.22 REST API + Docker",
	}

	resBytes, _ := json.Marshal(res)
	a.store.UpdateJob(jobID, "succeeded", resBytes, nil)

	return res, nil
}
