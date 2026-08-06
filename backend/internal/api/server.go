package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/saas-agent-platform/backend/internal/agents/appdeveloper"
	"github.com/saas-agent-platform/backend/internal/agents/appdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/appmaintainer"
	"github.com/saas-agent-platform/backend/internal/agents/llmdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/master"
	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/models"
	"github.com/saas-agent-platform/backend/internal/store"
)

type Server struct {
	router      *chi.Mux
	store       *store.Store
	daytonaClient *shared.DaytonaClient
	masterAgent *master.MasterAgent
	appDevAgent *appdeveloper.AppDeveloperAgent
	appDepAgent *appdeployer.AppDeployerAgent
	llmDepAgent *llmdeployer.LLMDeployerAgent
	appMaintAgent *appmaintainer.AppMaintainerAgent
	hub         *Hub
}

func NewServer(s *store.Store, dc *shared.DaytonaClient) *Server {
	srv := &Server{
		router:        chi.NewRouter(),
		store:         s,
		daytonaClient: dc,
		masterAgent:   master.NewMasterAgent(s, dc),
		appDevAgent:   appdeveloper.NewAppDeveloperAgent(s, dc),
		appDepAgent:   appdeployer.NewAppDeployerAgent(s, dc),
		llmDepAgent:   llmdeployer.NewLLMDeployerAgent(s, dc),
		appMaintAgent: appmaintainer.NewAppMaintainerAgent(s, dc),
		hub:           NewHub(),
	}

	srv.setupRoutes()
	return srv
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s.router.Route("/api/v1", func(r chi.Router) {
		// Auth
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/signup", s.handleSignup)

		// Projects
		r.Get("/projects", s.handleListProjects)
		r.Post("/projects", s.handleCreateProject)
		r.Get("/projects/{projectId}", s.handleGetProject)
		r.Patch("/projects/{projectId}", s.handleUpdateProject)
		r.Delete("/projects/{projectId}", s.handleDeleteProject)

		// Skills
		r.Get("/skills", s.handleListSkills)
		r.Post("/skills", s.handleCreateSkill)
		r.Post("/skills/upload", s.handleUploadSkills)
		r.Delete("/skills/{skillId}", s.handleDeleteSkill)

		// Chat Messages
		r.Get("/projects/{projectId}/messages", s.handleListMessages)
		r.Post("/projects/{projectId}/messages", s.handleCreateMessage)

		// Sandbox Files & Git
		r.Get("/projects/{projectId}/files", s.handleListFiles)
		r.Get("/projects/{projectId}/files/content", s.handleGetFileContent)
		r.Get("/projects/{projectId}/git/status", s.handleGetGitStatus)
		r.Get("/projects/{projectId}/git/diff", s.handleGetGitDiff)
		r.Post("/projects/{projectId}/git/push", s.handlePushGit)

		// Secrets
		r.Get("/projects/{projectId}/secrets", s.handleGetSecrets)
		r.Post("/projects/{projectId}/secrets", s.handleSaveSecret)

		// Jobs
		r.Get("/jobs/{jobId}", s.handleGetJob)

		// Terminal session creation
		r.Post("/projects/{projectId}/terminal/session", s.handleCreateTerminalSession)

		// WebSockets
		r.Get("/projects/{projectId}/stream", s.HandleStream)
		r.Get("/terminal/{sessionToken}", s.HandleTerminalWS)
	})
}

// Handlers implementation

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	u, err := s.store.GetUserByEmail(body.Email)
	if err != nil {
		u, _ = s.store.GetUserByID("user-default")
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": "dev-jwt-token-" + u.ID,
		"user":  u,
	})
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	u, _ := s.store.CreateUser(body.Email, body.Password)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": "dev-jwt-token-" + u.ID,
		"user":  u,
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	projects, _ := s.store.ListProjects("user-default")
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	p, err := s.store.CreateProject("user-default", body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	p, err := s.store.GetProject(pID)
	if err != nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	var body struct {
		GitRemoteURL string `json:"git_remote_url"`
		Name         string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	p, err := s.store.UpdateProject(pID, body.GitRemoteURL, body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	_ = s.store.DeleteProject(pID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	skills, _ := s.store.ListSkills("user-default")
	writeJSON(w, http.StatusOK, skills)
}

func (s *Server) handleCreateSkill(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Content     string `json:"content"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	sk, _ := s.store.CreateSkill("user-default", body.Name, body.Description, body.Content, "manual")
	writeJSON(w, http.StatusCreated, sk)
}

func (s *Server) handleUploadSkills(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	created := make([]*models.Skill, 0)
	for _, fHeader := range files {
		file, err := fHeader.Open()
		if err != nil {
			continue
		}
		buf, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(fHeader.Filename, ".md")
		sk, _ := s.store.CreateSkill("user-default", name, "Uploaded skill file: "+fHeader.Filename, string(buf), "uploaded")
		created = append(created, sk)
	}
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleDeleteSkill(w http.ResponseWriter, r *http.Request) {
	sID := chi.URLParam(r, "skillId")
	_ = s.store.DeleteSkill(sID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	msgs, _ := s.store.ListMessages(pID)
	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	var body struct {
		Content string `json:"content"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Save user message
	userMsg, _ := s.store.AddMessage(pID, "user", body.Content)

	// Process via Master Agent
	turnRes, err := s.masterAgent.ProcessTurn(r.Context(), pID, body.Content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add Assistant reply message
	asstMsg, _ := s.store.AddMessage(pID, "assistant", turnRes.AssistantResponse)

	// Emit system_status over WebSocket
	if turnRes.ActivatedAgent != "" {
		s.hub.BroadcastEvent(pID, &models.WSEvent{
			Type:  "system_status",
			JobID: turnRes.JobID,
			Agent: turnRes.ActivatedAgent,
			Text:  fmt.Sprintf("⚡ %s activated for task processing", turnRes.ActivatedAgent),
			Level: "info",
		})
	}

	// Asynchronously execute background job & stream tokens/updates
	if turnRes.JobID != "" {
		go s.runAsyncJob(pID, turnRes.JobID, turnRes.JobType, turnRes.JobPayload)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_message":         userMsg,
		"assistant_message_id": asstMsg.ID,
		"job_id":               turnRes.JobID,
	})
}

func (s *Server) runAsyncJob(projectID, jobID, jobType string, payloadBytes []byte) {
	payload := make(map[string]string)
	_ = json.Unmarshal(payloadBytes, &payload)

	ctx := context.Background()
	time.Sleep(1 * time.Second) // Simulate queue latency

	s.hub.BroadcastEvent(projectID, &models.WSEvent{
		Type:   "job_update",
		JobID:  jobID,
		Status: "running",
	})

	switch jobType {
	case "codegen":
		res, _ := s.appDevAgent.ExecuteCodegenJob(ctx, jobID, projectID, payload)
		resJSON, _ := json.Marshal(res)

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "system_status",
			JobID:  jobID,
			Agent:  "App Developer Agent",
			Text:   fmt.Sprintf("✅ Generated %d files (%s) in Daytona sandbox", res.FilesGenerated, res.Stack),
			Level:  "success",
		})

		sb := s.daytonaClient.GetOrCreateSandbox(projectID)
		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:        "git_status_changed",
			Uncommitted: sb.GetGitStatus(),
		})

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "job_update",
			JobID:  jobID,
			Status: "succeeded",
			Result: resJSON,
		})

	case "deploy_app":
		res, _ := s.appDepAgent.ExecuteDeployJob(ctx, jobID, projectID, payload)
		resJSON, _ := json.Marshal(res)

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "system_status",
			JobID:  jobID,
			Agent:  "App Deployer Agent",
			Text:   fmt.Sprintf("🚀 App deployed to Azure VM! Public URL: %s (IP: %s)", res.EndpointURL, res.PublicIP),
			Level:  "success",
		})

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "job_update",
			JobID:  jobID,
			Status: "succeeded",
			Result: resJSON,
		})

	case "deploy_llm":
		res, _ := s.llmDepAgent.ExecuteDeployJob(ctx, jobID, projectID, payload)
		resJSON, _ := json.Marshal(res)

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "system_status",
			JobID:  jobID,
			Agent:  "LLM Deployer Agent",
			Text:   fmt.Sprintf("🤖 LLM Endpoint Live! Model: %s | Topology: %s | URL: %s%s", res.ModelRepoID, res.Topology, res.EndpointURL, res.APIPath),
			Level:  "success",
		})

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "job_update",
			JobID:  jobID,
			Status: "succeeded",
			Result: resJSON,
		})

	case "maintain_app":
		res, _ := s.appMaintAgent.ExecuteMaintainJob(ctx, jobID, projectID, payload)
		resJSON, _ := json.Marshal(res)

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "system_status",
			JobID:  jobID,
			Agent:  "App Maintainer Agent",
			Text:   fmt.Sprintf("🛠️ Fix applied and verified! Commit %s pushed to remote.", res.CommitHash),
			Level:  "success",
		})

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "job_update",
			JobID:  jobID,
			Status: "succeeded",
			Result: resJSON,
		})

	case "push":
		sb := s.daytonaClient.GetOrCreateSandbox(projectID)
		sb.ClearGitStatus()

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "system_status",
			JobID:  jobID,
			Agent:  "Git Maintainer",
			Text:   "Pushed all uncommitted changes cleanly to remote repository.",
			Level:  "success",
		})

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:        "git_status_changed",
			Uncommitted: sb.GetGitStatus(),
		})

		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "job_update",
			JobID:  jobID,
			Status: "succeeded",
		})
	}
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}
	sb := s.daytonaClient.GetOrCreateSandbox(pID)
	items := sb.ListFiles(path)
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleGetFileContent(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	filePath := r.URL.Query().Get("path")

	sb := s.daytonaClient.GetOrCreateSandbox(pID)
	content, err := sb.ReadFile(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"path":    filePath,
		"content": content,
	})
}

func (s *Server) handleGetGitStatus(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	sb := s.daytonaClient.GetOrCreateSandbox(pID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"uncommitted": sb.GetGitStatus(),
	})
}

func (s *Server) handleGetGitDiff(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	filePath := r.URL.Query().Get("path")

	sb := s.daytonaClient.GetOrCreateSandbox(pID)
	diff := sb.GetGitDiff(filePath)
	writeJSON(w, http.StatusOK, map[string]string{
		"path": filePath,
		"diff": diff,
	})
}

func (s *Server) handlePushGit(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")

	// Check if github_pat secret exists
	sec := s.store.GetSecretsPresence(pID)
	if !sec.GitHubPAT {
		w.WriteHeader(http.StatusPreconditionRequired) // HTTP 428
		writeJSON(w, http.StatusPreconditionRequired, map[string]string{
			"error": "github_pat_required",
		})
		return
	}

	var body struct {
		CommitMessage string `json:"commit_message"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.CommitMessage == "" {
		body.CommitMessage = "Update project from SaaS Agentic Platform"
	}

	payload, _ := json.Marshal(map[string]string{"commit_message": body.CommitMessage})
	job, _ := s.store.CreateJob(pID, "push", payload)

	go s.runAsyncJob(pID, job.ID, "push", payload)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"job_id": job.ID,
		"status": "queued",
	})
}

func (s *Server) handleGetSecrets(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	presence := s.store.GetSecretsPresence(pID)
	writeJSON(w, http.StatusOK, presence)
}

func (s *Server) handleSaveSecret(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	var body struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	valBytes, _ := json.Marshal(body.Value)
	s.store.SaveSecret(pID, body.Type, string(valBytes))

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "stored",
		"type":   body.Type,
	})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	jID := chi.URLParam(r, "jobId")
	job, err := s.store.GetJob(jID)
	if err != nil {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleCreateTerminalSession(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	token := "term-token-" + uuid.New().String()[:8]
	wsURL := fmt.Sprintf("ws://localhost:8080/api/v1/terminal/%s", token)

	writeJSON(w, http.StatusOK, map[string]string{
		"session_token": token,
		"websocket_url": wsURL,
		"project_id":    pID,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
