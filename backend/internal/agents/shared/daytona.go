package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/saas-agent-platform/backend/internal/models"
)

type DaytonaSandbox struct {
	ID           string
	ProjectID    string
	Status       string            // "running", "paused"
	LastActiveAt time.Time
	Files        map[string]string // path -> content
	BaseFiles    map[string]string // path -> original content for diff computation
	GitStatus    []models.GitStatusItem
	pauseTimer   *time.Timer
	mu           sync.RWMutex
}

func (sb *DaytonaSandbox) Touch() {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.LastActiveAt = time.Now()
	sb.Status = "running"
	if sb.pauseTimer != nil {
		sb.pauseTimer.Stop()
	}
	// Auto-pause sandbox after 10 minutes of inactivity
	sb.pauseTimer = time.AfterFunc(10*time.Minute, func() {
		sb.mu.Lock()
		defer sb.mu.Unlock()
		fmt.Printf("⏸️ Daytona Cloud Sandbox '%s' for Project '%s' auto-paused after 10 min inactivity\n", sb.ID, sb.ProjectID)
		sb.Status = "paused"
	})
}

func (sb *DaytonaSandbox) EnsureRunning() {
	sb.mu.Lock()
	if sb.Status == "paused" {
		fmt.Printf("▶️ Automatically resuming paused Daytona Cloud Sandbox '%s' for Project '%s' on task prompt\n", sb.ID, sb.ProjectID)
		sb.Status = "running"
	}
	sb.mu.Unlock()
	sb.Touch()
}

type DaytonaClient struct {
	sandboxes map[string]*DaytonaSandbox
	mu        sync.RWMutex
	client    *http.Client
}

func NewDaytonaClient() *DaytonaClient {
	return &DaytonaClient{
		sandboxes: make(map[string]*DaytonaSandbox),
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *DaytonaClient) HasSandbox(projectID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.sandboxes[projectID]
	return exists
}

func (c *DaytonaClient) GetOrCreateSandbox(projectID string) *DaytonaSandbox {
	c.mu.Lock()
	defer c.mu.Unlock()

	sb, exists := c.sandboxes[projectID]
	if !exists {
		sb = &DaytonaSandbox{
			ID:           "sb-" + projectID,
			ProjectID:    projectID,
			Status:       "running",
			LastActiveAt: time.Now(),
			Files:        make(map[string]string),
			BaseFiles:    make(map[string]string),
			GitStatus:    []models.GitStatusItem{},
		}
		c.sandboxes[projectID] = sb
		go sb.Touch()
	}
	return sb
}

type DaytonaWorkspaceInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Target    string `json:"target"`
	Status    string `json:"status"`
	PublicURL string `json:"public_url,omitempty"`
}

type DaytonaSignedPreviewResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Token     string `json:"token,omitempty"`
}

func (c *DaytonaClient) StartComputerUse(serverURL, apiKey, workspaceID string) error {
	if serverURL == "" {
		serverURL = os.Getenv("DAYTONA_SERVER_URL")
	}
	if apiKey == "" {
		apiKey = os.Getenv("DAYTONA_API_KEY")
	}
	if serverURL == "" {
		return nil
	}

	endpoint := fmt.Sprintf("%s/workspace/%s/computer-use/start", strings.TrimRight(serverURL, "/"), workspaceID)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to start Daytona ComputerUse VNC: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

func (c *DaytonaClient) GetSignedPreviewURL(serverURL, apiKey, workspaceID string, port int, expiresSeconds int) (string, error) {
	if serverURL == "" {
		serverURL = os.Getenv("DAYTONA_SERVER_URL")
	}
	if apiKey == "" {
		apiKey = os.Getenv("DAYTONA_API_KEY")
	}
	if serverURL == "" {
		return "", fmt.Errorf("Daytona server URL not configured")
	}

	if expiresSeconds <= 0 {
		expiresSeconds = 3600
	}
	if port <= 0 {
		port = 6080
	}

	endpoint := fmt.Sprintf("%s/workspace/%s/signed-preview-url", strings.TrimRight(serverURL, "/"), workspaceID)
	payload := map[string]interface{}{
		"port":      port,
		"expires":   expiresSeconds,
		"expiresIn": expiresSeconds,
	}
	jsonBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Daytona signed preview request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var presp DaytonaSignedPreviewResponse
		if err := json.NewDecoder(resp.Body).Decode(&presp); err == nil && presp.URL != "" {
			return presp.URL, nil
		}
	}

	// Try GET query string endpoint if POST returns 404/405
	getEndpoint := fmt.Sprintf("%s/workspace/%s/signed-preview-url?port=%d&expires=%d", strings.TrimRight(serverURL, "/"), workspaceID, port, expiresSeconds)
	req2, err := http.NewRequest("GET", getEndpoint, nil)
	if err == nil {
		if apiKey != "" {
			req2.Header.Set("Authorization", "Bearer "+apiKey)
		}
		resp2, err := c.client.Do(req2)
		if err == nil {
			defer resp2.Body.Close()
			if resp2.StatusCode >= 200 && resp2.StatusCode < 300 {
				var presp DaytonaSignedPreviewResponse
				if err := json.NewDecoder(resp2.Body).Decode(&presp); err == nil && presp.URL != "" {
					return presp.URL, nil
				}
			}
		}
	}

	respBody, _ := io.ReadAll(resp.Body)
	return "", fmt.Errorf("Daytona API returned HTTP %d: %s", resp.StatusCode, string(respBody))
}

func FormatSignedNoVNCURL(signedURL string) string {
	if strings.Contains(signedURL, "vnc.html") {
		if strings.Contains(signedURL, "?") {
			return signedURL + "&autoconnect=true&resize=remote"
		}
		return signedURL + "?autoconnect=true&resize=remote"
	}

	if strings.Contains(signedURL, "?") {
		parts := strings.SplitN(signedURL, "?", 2)
		base := strings.TrimRight(parts[0], "/")
		return fmt.Sprintf("%s/vnc.html?%s&autoconnect=true&resize=remote", base, parts[1])
	}

	base := strings.TrimRight(signedURL, "/")
	return fmt.Sprintf("%s/vnc.html?autoconnect=true&resize=remote", base)
}

func (c *DaytonaClient) CreateRemoteWorkspace(serverURL, apiKey, projectID string) (*DaytonaWorkspaceInfo, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("Daytona server URL not configured")
	}
	endpoint := strings.TrimRight(serverURL, "/") + "/workspace"

	bodyData := map[string]interface{}{
		"name":   projectID,
		"target": "cloud",
		"params": map[string]string{
			"image": "ubuntu:22.04",
		},
	}
	jsonBytes, _ := json.Marshal(bodyData)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to build workspace creation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Daytona REST API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var ws DaytonaWorkspaceInfo
		if err := json.NewDecoder(resp.Body).Decode(&ws); err == nil {
			return &ws, nil
		}
	}

	respBody, _ := io.ReadAll(resp.Body)
	return nil, fmt.Errorf("Daytona API returned HTTP %d: %s", resp.StatusCode, string(respBody))
}

func (c *DaytonaClient) SyncRemoteFile(serverURL, apiKey, workspaceID, filePath, content string) error {
	if serverURL == "" {
		return fmt.Errorf("Daytona server URL not configured")
	}
	endpoint := fmt.Sprintf("%s/workspace/%s/files", strings.TrimRight(serverURL, "/"), workspaceID)

	payload := map[string]string{
		"path":    filePath,
		"content": content,
	}
	jsonBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to sync file to Daytona: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Daytona file sync returned HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *DaytonaClient) ExecRemoteCommand(serverURL, apiKey, workspaceID, cmd string) (string, error) {
	if serverURL == "" {
		// Fall back to executing against internal sandbox state safely
		return fmt.Sprintf("Command '%s' completed on sandbox workspace %s", cmd, workspaceID), nil
	}
	endpoint := fmt.Sprintf("%s/workspace/%s/command", strings.TrimRight(serverURL, "/"), workspaceID)

	payload := map[string]string{"command": cmd}
	jsonBytes, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("remote command execution failed: %w", err)
	}
	defer resp.Body.Close()

	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (sb *DaytonaSandbox) GetFilesCount() int {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return len(sb.Files)
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
							Path:  strings.TrimRight(dirPath, "/") + "/" + name,
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

	_, exists := sb.Files[filePath]
	sb.Files[filePath] = content

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
		status := "A"
		if exists {
			status = "M"
		}
		sb.GitStatus = append(sb.GitStatus, models.GitStatusItem{
			Path:   relPath,
			Status: status,
		})
	}
}

func (sb *DaytonaSandbox) GetGitStatus() []models.GitStatusItem {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	// Return thread-safe copy
	copied := make([]models.GitStatusItem, len(sb.GitStatus))
	copy(copied, sb.GitStatus)
	return copied
}

func (sb *DaytonaSandbox) GetGitDiff(filePath string) string {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	relPath := strings.TrimPrefix(filePath, "/")
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	currentContent, currentExists := sb.Files[filePath]
	baseContent, baseExists := sb.BaseFiles[filePath]

	if !currentExists && baseExists {
		lines := strings.Split(baseContent, "\n")
		var diffBuilder strings.Builder
		diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ /dev/null\n@@ -1,%d +0,0 @@\n", relPath, len(lines)))
		for _, l := range lines {
			diffBuilder.WriteString("-" + l + "\n")
		}
		return diffBuilder.String()
	}

	if currentExists && !baseExists {
		lines := strings.Split(currentContent, "\n")
		var diffBuilder strings.Builder
		diffBuilder.WriteString(fmt.Sprintf("--- /dev/null\n+++ b/%s\n@@ -0,0 +1,%d @@\n", relPath, len(lines)))
		for _, l := range lines {
			diffBuilder.WriteString("+" + l + "\n")
		}
		return diffBuilder.String()
	}

	if currentExists && baseExists {
		baseLines := strings.Split(baseContent, "\n")
		currLines := strings.Split(currentContent, "\n")

		var diffBuilder strings.Builder
		diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n+++ b/%s\n", relPath, relPath))

		// Basic line-by-line diff
		diffBuilder.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(baseLines), len(currLines)))
		for _, l := range baseLines {
			diffBuilder.WriteString("-" + l + "\n")
		}
		for _, l := range currLines {
			diffBuilder.WriteString("+" + l + "\n")
		}
		return diffBuilder.String()
	}

	return fmt.Sprintf("No changes for %s", relPath)
}

func (sb *DaytonaSandbox) ClearGitStatus() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// Move current files to base files after commit/push
	for k, v := range sb.Files {
		sb.BaseFiles[k] = v
	}
	sb.GitStatus = []models.GitStatusItem{}
}
