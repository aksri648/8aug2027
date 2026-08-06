package shared

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/saas-agent-platform/backend/internal/models"
)

// DaytonaSandbox represents a project's Daytona cloud sandbox workspace
type DaytonaSandbox struct {
	ID        string
	ProjectID string
	Files     map[string]string // path -> content
	GitStatus []models.GitStatusItem
	mu        sync.RWMutex
}

type DaytonaClient struct {
	sandboxes map[string]*DaytonaSandbox
	mu        sync.RWMutex
}

func NewDaytonaClient() *DaytonaClient {
	c := &DaytonaClient{
		sandboxes: make(map[string]*DaytonaSandbox),
	}
	// Seed demo project sandbox
	demoSb := c.GetOrCreateSandbox("proj-default")
	demoSb.Files["/main.go"] = `package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from Daytona Cloud Sandbox!")
	})
	http.ListenAndServe(":8080", nil)
}`
	demoSb.Files["/Dockerfile"] = "FROM golang:1.22-alpine\nWORKDIR /app\nCOPY . .\nRUN go build -o server main.go\nEXPOSE 8080\nCMD [\"./server\"]"
	demoSb.Files["/go.mod"] = "module example.com/ecommerce\n\ngo 1.22"
	demoSb.Files["/README.md"] = "# E-Commerce Microservices Platform\nBuilt via SaaS Agentic Platform."

	demoSb.GitStatus = []models.GitStatusItem{
		{Path: "main.go", Status: "M"},
		{Path: "Dockerfile", Status: "A"},
		{Path: "README.md", Status: "M"},
	}

	return c
}

func (c *DaytonaClient) GetOrCreateSandbox(projectID string) *DaytonaSandbox {
	c.mu.Lock()
	defer c.mu.Unlock()

	sb, exists := c.sandboxes[projectID]
	if !exists {
		sb = &DaytonaSandbox{
			ID:        "sb-" + projectID,
			ProjectID: projectID,
			Files:     make(map[string]string),
			GitStatus: []models.GitStatusItem{},
		}
		// Default files for new sandboxes
		sb.Files["/README.md"] = fmt.Sprintf("# Project %s\nCreated in Daytona Cloud Sandbox on %s", projectID, time.Now().Format(time.RFC3339))
		c.sandboxes[projectID] = sb
	}
	return sb
}

func (sb *DaytonaSandbox) ListFiles(dirPath string) []models.FileItem {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	dirPath = filepath.Clean(dirPath)
	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}
	if dirPath != "/" && !strings.HasSuffix(dirPath, "/") {
		dirPath = dirPath + "/"
	}

	items := make([]models.FileItem, 0)
	seen := make(map[string]bool)

	for p, content := range sb.Files {
		cleanP := filepath.Clean(p)
		if !strings.HasPrefix(cleanP, "/") {
			cleanP = "/" + cleanP
		}

		if dirPath == "/" {
			rel := strings.TrimPrefix(cleanP, "/")
			parts := strings.Split(rel, "/")
			if len(parts) > 0 && parts[0] != "" {
				name := parts[0]
				if !seen[name] {
					seen[name] = true
					isDir := len(parts) > 1
					items = append(items, models.FileItem{
						Name:  name,
						Path:  "/" + name,
						IsDir: isDir,
						Size:  int64(len(content)),
					})
				}
			}
		} else {
			if strings.HasPrefix(cleanP, dirPath) {
				rel := strings.TrimPrefix(cleanP, dirPath)
				parts := strings.Split(rel, "/")
				if len(parts) > 0 && parts[0] != "" {
					name := parts[0]
					if !seen[name] {
						seen[name] = true
						isDir := len(parts) > 1
						items = append(items, models.FileItem{
							Name:  name,
							Path:  dirPath + name,
							IsDir: isDir,
							Size:  int64(len(content)),
						})
					}
				}
			}
		}
	}
	return items
}

func (sb *DaytonaSandbox) ReadFile(filePath string) (string, error) {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	filePath = filepath.Clean(filePath)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	content, ok := sb.Files[filePath]
	if !ok {
		return "", fmt.Errorf("file not found: %s", filePath)
	}
	return content, nil
}

func (sb *DaytonaSandbox) WriteFile(filePath, content string) {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	filePath = filepath.Clean(filePath)
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	sb.Files[filePath] = content

	// Update git status
	relPath := strings.TrimPrefix(filePath, "/")
	found := false
	for i, st := range sb.GitStatus {
		if st.Path == relPath {
			sb.GitStatus[i].Status = "M"
			found = true
			break
		}
	}
	if !found {
		sb.GitStatus = append(sb.GitStatus, models.GitStatusItem{
			Path:   relPath,
			Status: "A",
		})
	}
}

func (sb *DaytonaSandbox) GetGitStatus() []models.GitStatusItem {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.GitStatus
}

func (sb *DaytonaSandbox) GetGitDiff(filePath string) string {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	relPath := strings.TrimPrefix(filePath, "/")
	content, ok := sb.Files["/"+relPath]
	if !ok {
		return fmt.Sprintf("--- /dev/null\n+++ b/%s\n@@ -0,0 +1 @@\n+ (deleted)", relPath)
	}

	lines := strings.Split(content, "\n")
	diff := fmt.Sprintf("--- a/%s\n+++ b/%s\n@@ -0,0 +1,%d @@\n", relPath, relPath, len(lines))
	for _, l := range lines {
		diff += "+" + l + "\n"
	}
	return diff
}

func (sb *DaytonaSandbox) ClearGitStatus() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.GitStatus = []models.GitStatusItem{}
}
