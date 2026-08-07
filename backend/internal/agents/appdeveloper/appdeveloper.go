package appdeveloper

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
	Description    string   `json:"description"`
}

func (a *AppDeveloperAgent) ExecuteCodegenJob(ctx context.Context, jobID, projectID string, payload map[string]string) (*CodegenResult, error) {
	sb := a.daytonaClient.GetOrCreateSandbox(projectID)

	prompt := payload["prompt"]
	stack := payload["stack"]
	if stack == "" {
		stack = "Go 1.22 REST API + Docker"
	}

	files := make(map[string]string)
	description := fmt.Sprintf("Generated %s application based on prompt: '%s'", stack, prompt)

	lowerStack := strings.ToLower(stack)

	if strings.Contains(lowerStack, "react") {
		files["/package.json"] = `{
  "name": "react-sandbox-app",
  "private": true,
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "lucide-react": "^0.380.0"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.3.0",
    "vite": "^5.2.11"
  }
}`
		files["/index.html"] = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>React Sandbox Application</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.jsx"></script>
  </body>
</html>`
		files["/src/main.jsx"] = `import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.jsx'

ReactDOM.createRoot(document.getElementById('root')).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)`
		files["/src/App.jsx"] = fmt.Sprintf(`import React from 'react';

export default function App() {
  return (
    <div style={{ padding: '2rem', fontFamily: 'sans-serif', backgroundColor: '#0f172a', color: '#f8fafc', minHeight: '100vh' }}>
      <h1>React + Vite Application</h1>
      <p>Prompt: %s</p>
      <div style={{ marginTop: '1rem', padding: '1rem', background: '#1e293b', borderRadius: '8px' }}>
        Status: Application generated and running in Daytona Sandbox
      </div>
    </div>
  );
}`, prompt)
		files["/README.md"] = fmt.Sprintf("# React + Vite App\nGenerated for: %s\n\n## Run\n```bash\nnpm run dev\n```", prompt)

	} else if strings.Contains(lowerStack, "python") || strings.Contains(lowerStack, "fastapi") {
		files["/main.py"] = fmt.Sprintf(`from fastapi import FastAPI
from datetime import datetime

app = FastAPI(title="FastAPI Service", description="Generated for %s")

@app.get("/healthz")
def health_check():
    return {"status": "ok", "timestamp": datetime.utcnow().isoformat()}

@app.get("/api/v1/data")
def get_data():
    return {"prompt": "%s", "items": [{"id": 1, "name": "Python FastAPI Item"}]}
`, prompt, prompt)
		files["/requirements.txt"] = "fastapi==0.111.0\nuvicorn==0.30.0\npydantic==2.7.1"
		files["/Dockerfile"] = `FROM python:3.11-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]`
		files["/README.md"] = fmt.Sprintf("# Python FastAPI Microservice\nGenerated for: %s\n\n## Run\n```bash\nuvicorn main:app --reload\n```", prompt)

	} else if strings.Contains(lowerStack, "next") {
		files["/package.json"] = `{
  "name": "nextjs-app",
  "version": "1.0.0",
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  },
  "dependencies": {
    "next": "^14.2.3",
    "react": "^18.3.1",
    "react-dom": "^18.3.1"
  }
}`
		files["/app/page.tsx"] = fmt.Sprintf(`export default function Home() {
  return (
    <main style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <h1>Next.js Fullstack App</h1>
      <p>Requirement: %s</p>
    </main>
  );
}`, prompt)
		files["/app/api/health/route.ts"] = `import { NextResponse } from 'next/server';

export async function GET() {
  return NextResponse.json({ status: 'ok', timestamp: new Date().toISOString() });
}`
		files["/README.md"] = fmt.Sprintf("# Next.js App\nGenerated for: %s", prompt)

	} else {
		// Go REST API Default
		files["/cmd/api/main.go"] = fmt.Sprintf(`package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"prompt": "%s",
		})
	})

	http.HandleFunc("/api/v1/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"service": "Go REST Microservice",
			"prompt": "%s",
			"items": []map[string]string{
				{"id": "1", "name": "Go Microservice Item Alpha"},
				{"id": "2", "name": "Go Microservice Item Beta"},
			},
		})
	})

	log.Println("Starting service on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}`, prompt, prompt)
		files["/go.mod"] = "module github.com/user/app\n\ngo 1.22"
		files["/Dockerfile"] = `FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN go build -o server ./cmd/api/main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]`
		files["/README.md"] = fmt.Sprintf("# Generated Application\nGenerated on %s by App Developer Agent.\n\nPrompt: %s\nStack: %s\n\n## Running\n```bash\ngo run ./cmd/api/main.go\n```", time.Now().Format(time.RFC3339), prompt, stack)
	}

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
