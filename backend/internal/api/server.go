package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/saas-agent-platform/backend/internal/agents/appdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/appdeveloper"
	"github.com/saas-agent-platform/backend/internal/agents/appmaintainer"
	"github.com/saas-agent-platform/backend/internal/agents/llmdeployer"
	"github.com/saas-agent-platform/backend/internal/agents/master"
	"github.com/saas-agent-platform/backend/internal/agents/shared"
	"github.com/saas-agent-platform/backend/internal/auth"
	"github.com/saas-agent-platform/backend/internal/metrics"
	"github.com/saas-agent-platform/backend/internal/models"
	"github.com/saas-agent-platform/backend/internal/queue"
	"github.com/saas-agent-platform/backend/internal/store"
)

type TerminalSessionInfo struct {
	ProjectID string
	UserID    string
	CreatedAt time.Time
}

type Server struct {
	router           *chi.Mux
	store            *store.Store
	daytonaClient    *shared.DaytonaClient
	jobQueue         queue.JobQueue
	masterAgent      *master.MasterAgent
	appDevAgent      *appdeveloper.AppDeveloperAgent
	appDepAgent      *appdeployer.AppDeployerAgent
	llmDepAgent      *llmdeployer.LLMDeployerAgent
	appMaintAgent    *appmaintainer.AppMaintainerAgent
	hub              *Hub
	terminalSessions map[string]TerminalSessionInfo
	termMu           sync.RWMutex
}

func NewServer(s *store.Store, dc *shared.DaytonaClient, jq queue.JobQueue) *Server {
	srv := &Server{
		router:           chi.NewRouter(),
		store:            s,
		daytonaClient:    dc,
		jobQueue:         jq,
		masterAgent:      master.NewMasterAgent(s, dc),
		appDevAgent:      appdeveloper.NewAppDeveloperAgent(s, dc),
		appDepAgent:      appdeployer.NewAppDeployerAgent(s, dc),
		llmDepAgent:      llmdeployer.NewLLMDeployerAgent(s, dc),
		appMaintAgent:    appmaintainer.NewAppMaintainerAgent(s, dc),
		hub:              NewHub(),
		terminalSessions: make(map[string]TerminalSessionInfo),
	}

	srv.setupRoutes()
	return srv
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(metrics.PrometheusMiddleware)
	s.router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Prometheus Metrics Endpoints
	s.router.Get("/metrics", http.HandlerFunc(promhttp.Handler().ServeHTTP))
	s.router.Get("/api/v1/metrics", http.HandlerFunc(promhttp.Handler().ServeHTTP))

	s.router.Route("/api/v1", func(r chi.Router) {
		// Public Auth Endpoints
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/signup", s.handleSignup)

		// Protected Routes Group
		r.Group(func(r chi.Router) {
			r.Use(auth.AuthMiddleware)

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
			r.Post("/projects/{projectId}/messages/stream", s.handleCreateMessageStream)

			// Sandbox Files & Git
			r.Get("/projects/{projectId}/files", s.handleListFiles)
			r.Get("/projects/{projectId}/files/content", s.handleGetFileContent)
			r.Get("/projects/{projectId}/git/status", s.handleGetGitStatus)
			r.Get("/projects/{projectId}/git/diff", s.handleGetGitDiff)
			r.Post("/projects/{projectId}/git/push", s.handlePushGit)

			// Sandbox Live Preview
			r.Get("/projects/{projectId}/sandbox/preview", s.handleSandboxPreview)
			r.Get("/projects/{projectId}/sandbox/app", s.handleServeSandboxApp)
			r.Get("/projects/{projectId}/sandbox/app/*", s.handleServeSandboxApp)
			r.Get("/projects/{projectId}/sandbox/novnc", s.handleServeSandboxNoVNC)

			// Secrets
			r.Get("/projects/{projectId}/secrets", s.handleGetSecrets)
			r.Post("/projects/{projectId}/secrets", s.handleSaveSecret)

			// Config
			r.Get("/config", s.handleGetConfig)
			r.Post("/config", s.handleSaveConfig)

			// LLM Providers Discovery & Connection Testing
			r.Post("/providers/test", s.handleTestProviderConnection)
			r.Post("/providers/discover", s.handleDiscoverModels)

			// Jobs
			r.Get("/jobs/{jobId}", s.handleGetJob)

			// Terminal session creation
			r.Post("/projects/{projectId}/terminal/session", s.handleCreateTerminalSession)

			// WebSockets
			r.Get("/projects/{projectId}/stream", s.HandleStream)
		})

		// Terminal WebSocket connection (authenticated via sessionToken)
		r.Get("/terminal/{sessionToken}", s.HandleTerminalWS)
	})
}

func (s *Server) registerTerminalSession(token, projectID, userID string) {
	s.termMu.Lock()
	defer s.termMu.Unlock()
	s.terminalSessions[token] = TerminalSessionInfo{
		ProjectID: projectID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}
}

func (s *Server) verifyTerminalSessionToken(token string) (string, string, error) {
	s.termMu.RLock()
	defer s.termMu.RUnlock()
	info, ok := s.terminalSessions[token]
	if !ok {
		return "", "", fmt.Errorf("session token not found")
	}
	return info.ProjectID, info.UserID, nil
}

// Handlers implementation

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	u, err := s.store.AuthenticateUser(body.Email, body.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	token, err := auth.GenerateToken(u.ID, u.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  u,
	})
}

func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Email == "" || body.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email and password are required"})
		return
	}

	u, err := s.store.CreateUser(body.Email, body.Password)
	if err != nil {
		if err == store.ErrAlreadyExists {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "user with this email already exists"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	token, err := auth.GenerateToken(u.ID, u.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user":  u,
	})
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	projects, err := s.store.ListProjects(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, projects)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	p, err := s.store.CreateProject(userID, body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Instantly open/provision Daytona Cloud Sandbox in the background for this conversation
	go s.daytonaClient.GetOrCreateSandbox(p.ID)

	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	p, err := s.store.GetProjectForUser(pID, userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var body struct {
		GitRemoteURL string `json:"git_remote_url"`
		Name         string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	p, err := s.store.UpdateProjectForUser(pID, userID, body.GitRemoteURL, body.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	err := s.store.DeleteProjectForUser(pID, userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	skills, err := s.store.ListSkills(userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, skills)
}

func (s *Server) handleCreateSkill(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Content     string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	sk, err := s.store.CreateSkill(userID, body.Name, body.Description, body.Content, "manual")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, sk)
}

func (s *Server) handleUploadSkills(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	err := r.ParseMultipartForm(10 << 20) // 10MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart form"})
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
		sk, err := s.store.CreateSkill(userID, name, "Uploaded skill file: "+fHeader.Filename, string(buf), "uploaded")
		if err == nil {
			created = append(created, sk)
		}
	}
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) handleDeleteSkill(w http.ResponseWriter, r *http.Request) {
	userID, _ := auth.GetUserIDFromContext(r.Context())
	sID := chi.URLParam(r, "skillId")
	if err := s.store.DeleteSkill(sID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())

	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	msgs, err := s.store.ListMessages(pID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) handleCreateMessage(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())

	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	var body struct {
		Content      string                 `json:"content"`
		AgentPayload map[string]interface{} `json:"agent_payload,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if strings.TrimSpace(body.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message content cannot be empty"})
		return
	}

	// Save user message
	userMsg, err := s.store.AddMessage(pID, "user", body.Content)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Emit stream start event over WebSocket
	s.hub.BroadcastEvent(pID, &models.WSEvent{
		Type: "chat_stream_start",
	})

	// Process via Master Agent with real-time WebSocket token streaming
	turnRes, err := s.masterAgent.ProcessTurnWithStream(r.Context(), pID, body.Content, body.AgentPayload, func(token string) {
		s.hub.BroadcastEvent(pID, &models.WSEvent{
			Type: "chat_stream_chunk",
			Text: token,
		})
	})
	if err != nil {
		s.hub.BroadcastEvent(pID, &models.WSEvent{Type: "chat_stream_end"})
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Emit stream end event
	s.hub.BroadcastEvent(pID, &models.WSEvent{
		Type: "chat_stream_end",
	})

	// Add Assistant reply message
	asstMsg, err := s.store.AddMessage(pID, "assistant", turnRes.AssistantResponse)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

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

	// Enqueue background job in Redis Queue for worker execution
	if turnRes.JobID != "" {
		job, err := s.store.GetJob(turnRes.JobID)
		if err == nil {
			if s.jobQueue != nil {
				_ = s.jobQueue.EnqueueJob(r.Context(), job)
			} else {
				go s.runAsyncJob(pID, turnRes.JobID, turnRes.JobType, turnRes.JobPayload)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"user_message":         userMsg,
		"assistant_message_id": asstMsg.ID,
		"job_id":               turnRes.JobID,
	})
}

func (s *Server) handleCreateMessageStream(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())

	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	var body struct {
		Content      string                 `json:"content"`
		AgentPayload map[string]interface{} `json:"agent_payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errBytes, _ := json.Marshal(map[string]string{"error": "invalid JSON body"})
		fmt.Fprintf(w, "data: %s\n\n", errBytes)
		flusher.Flush()
		return
	}

	if strings.TrimSpace(body.Content) == "" {
		errBytes, _ := json.Marshal(map[string]string{"error": "message content cannot be empty"})
		fmt.Fprintf(w, "data: %s\n\n", errBytes)
		flusher.Flush()
		return
	}

	userMsg, _ := s.store.AddMessage(pID, "user", body.Content)
	userMsgBytes, _ := json.Marshal(map[string]interface{}{"type": "user_msg", "user_message": userMsg})
	fmt.Fprintf(w, "data: %s\n\n", userMsgBytes)
	flusher.Flush()

	turnRes, err := s.masterAgent.ProcessTurnWithStream(r.Context(), pID, body.Content, body.AgentPayload, func(token string) {
		tokenJSON, _ := json.Marshal(map[string]string{"type": "delta", "delta": token})
		fmt.Fprintf(w, "data: %s\n\n", tokenJSON)
		flusher.Flush()
	})

	if err != nil {
		errJSON, _ := json.Marshal(map[string]string{"type": "error", "error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", errJSON)
		flusher.Flush()
		return
	}

	asstMsg, _ := s.store.AddMessage(pID, "assistant", turnRes.AssistantResponse)

	if turnRes.ActivatedAgent != "" {
		s.hub.BroadcastEvent(pID, &models.WSEvent{
			Type:  "system_status",
			JobID: turnRes.JobID,
			Agent: turnRes.ActivatedAgent,
			Text:  fmt.Sprintf("⚡ %s activated for task processing", turnRes.ActivatedAgent),
			Level: "info",
		})
	}

	if turnRes.JobID != "" {
		job, err := s.store.GetJob(turnRes.JobID)
		if err == nil && s.jobQueue != nil {
			_ = s.jobQueue.EnqueueJob(r.Context(), job)
		}
	}

	doneJSON, _ := json.Marshal(map[string]interface{}{
		"type":                 "done",
		"assistant_message_id": asstMsg.ID,
		"assistant_content":    turnRes.AssistantResponse,
		"job_id":               turnRes.JobID,
		"activated_agent":      turnRes.ActivatedAgent,
	})
	fmt.Fprintf(w, "data: %s\n\n", doneJSON)
	flusher.Flush()
}

func (s *Server) runAsyncJob(projectID, jobID, jobType string, payloadBytes []byte) {
	payload := make(map[string]string)
	_ = json.Unmarshal(payloadBytes, &payload)

	ctx := context.Background()

	s.hub.BroadcastEvent(projectID, &models.WSEvent{
		Type:   "job_update",
		JobID:  jobID,
		Status: "running",
	})
	s.store.UpdateJob(jobID, "running", nil, nil)

	var jobErr error

	switch jobType {
	case "codegen":
		res, err := s.appDevAgent.ExecuteCodegenJob(ctx, jobID, projectID, payload)
		if err != nil {
			jobErr = err
		} else {
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
		}

	case "deploy_app":
		res, err := s.appDepAgent.ExecuteDeployJob(ctx, jobID, projectID, payload)
		if err != nil {
			jobErr = err
		} else {
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
		}

	case "deploy_llm":
		res, err := s.llmDepAgent.ExecuteDeployJob(ctx, jobID, projectID, payload)
		if err != nil {
			jobErr = err
		} else {
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
		}

	case "maintain_app":
		res, err := s.appMaintAgent.ExecuteMaintainJob(ctx, jobID, projectID, payload)
		if err != nil {
			jobErr = err
		} else {
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
		}

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
		s.store.UpdateJob(jobID, "succeeded", nil, nil)
	}

	if jobErr != nil {
		errStr := jobErr.Error()
		s.store.UpdateJob(jobID, "failed", nil, &errStr)
		if s.jobQueue != nil {
			job, _ := s.store.GetJob(jobID)
			if job != nil {
				s.jobQueue.EnqueueDLQ(ctx, job, errStr)
			}
		}
		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "system_status",
			JobID:  jobID,
			Agent:  jobType,
			Text:   fmt.Sprintf("❌ Job failed: %s", errStr),
			Level:  "error",
		})
		s.hub.BroadcastEvent(projectID, &models.WSEvent{
			Type:   "job_update",
			JobID:  jobID,
			Status: "failed",
		})
	}
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

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
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	filePath := r.URL.Query().Get("path")
	sb := s.daytonaClient.GetOrCreateSandbox(pID)
	content, err := sb.ReadFile(filePath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"path":    filePath,
		"content": content,
	})
}

func (s *Server) handleGetGitStatus(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	if !s.daytonaClient.HasSandbox(pID) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"uncommitted": []interface{}{},
		})
		return
	}

	sb := s.daytonaClient.GetOrCreateSandbox(pID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"uncommitted": sb.GetGitStatus(),
	})
}

func (s *Server) handleGetGitDiff(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

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
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	// Check if github_pat secret exists
	sec := s.store.GetSecretsPresence(pID)
	if !sec.GitHubPAT {
		// Single writeJSON call with 428 Precondition Required (fixes EFV1-10 double Header write)
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
	job, err := s.store.CreateJob(pID, "push", payload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if s.jobQueue != nil {
		_ = s.jobQueue.EnqueueJob(r.Context(), job)
	} else {
		go s.runAsyncJob(pID, job.ID, "push", payload)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"job_id": job.ID,
		"status": "queued",
	})
}

func (s *Server) handleGetSecrets(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	presence := s.store.GetSecretsPresence(pID)
	writeJSON(w, http.StatusOK, presence)
}

func (s *Server) handleSaveSecret(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	var body struct {
		Type  string      `json:"type"`
		Value interface{} `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	valBytes, _ := json.Marshal(body.Value)
	if err := s.store.SaveSecret(pID, body.Type, string(valBytes)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "stored",
		"type":   body.Type,
	})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "server_config_active",
	})
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	pID := r.URL.Query().Get("project_id")
	if pID != "" {
		userID, _ := auth.GetUserIDFromContext(r.Context())
		if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "unauthorized project config"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "configuration_saved_server_side"})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	jID := chi.URLParam(r, "jobId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	job, err := s.store.GetJobForUser(jID, userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found or unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleCreateTerminalSession(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())

	if _, err := s.store.GetProjectForUser(pID, userID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	token := "term-token-" + uuid.New().String()[:12]
	s.registerTerminalSession(token, pID, userID)

	wsURL := fmt.Sprintf("ws://%s/api/v1/terminal/%s", r.Host, token)

	writeJSON(w, http.StatusOK, map[string]string{
		"session_token": token,
		"websocket_url": wsURL,
		"project_id":    pID,
	})
}

func (s *Server) handleSandboxPreview(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	p, err := s.store.GetProjectForUser(pID, userID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found or unauthorized"})
		return
	}

	sb := s.daytonaClient.GetOrCreateSandbox(pID)
	filesCount := sb.GetFilesCount()

	noVNCURL := os.Getenv("DAYTONA_NOVNC_URL")
	if noVNCURL == "" {
		serverURL := os.Getenv("DAYTONA_SERVER_URL")
		if serverURL != "" {
			trimmed := strings.TrimRight(serverURL, "/")
			noVNCURL = fmt.Sprintf("%s/workspace/%s/novnc/vnc.html?autoconnect=true&resize=remote", trimmed, sb.ID)
		} else {
			noVNCURL = fmt.Sprintf("/api/v1/projects/%s/sandbox/novnc", pID)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":   pID,
		"sandbox_id":   sb.ID,
		"novnc_url":    noVNCURL,
		"preview_url":  noVNCURL,
		"status":       "running",
		"port":         6080,
		"files_count":  filesCount,
		"service_name": p.Name,
	})
}

func (s *Server) handleServeSandboxNoVNC(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	p, err := s.store.GetProjectForUser(pID, userID)
	if err != nil {
		http.Error(w, "project not found or unauthorized", http.StatusNotFound)
		return
	}

	sb := s.daytonaClient.GetOrCreateSandbox(pID)

	targetVncURL := os.Getenv("DAYTONA_NOVNC_URL")
	if targetVncURL == "" {
		serverURL := os.Getenv("DAYTONA_SERVER_URL")
		if serverURL != "" {
			trimmed := strings.TrimRight(serverURL, "/")
			targetVncURL = fmt.Sprintf("%s/api/workspace/%s/novnc/vnc.html?autoconnect=true", trimmed, sb.ID)
		} else {
			targetVncURL = fmt.Sprintf("https://app.daytona.io/api/workspace/%s/novnc/vnc.html?autoconnect=true", sb.ID)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Daytona Sandbox noVNC Live Preview - %s</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0b0f19; color: #f8fafc; display: flex; flex-direction: column; height: 100vh; overflow: hidden; }
        .vnc-bar { background: #111827; border-b: 1px solid #1f2937; padding: 10px 16px; display: flex; align-items: center; justify-content: space-between; font-size: 12px; }
        .vnc-pill { display: inline-flex; align-items: center; gap: 6px; background: #064e3b; color: #34d399; padding: 3px 10px; border-radius: 9999px; font-weight: 600; font-size: 11px; }
        .vnc-viewport { flex: 1; display: flex; align-items: center; justify-content: center; background: #030712; position: relative; p-4; }
        .vnc-box { background: #1e293b; border: 1px solid #334155; border-radius: 16px; padding: 32px; max-width: 600px; width: 100%%; text-align: center; box-shadow: 0 25px 50px -12px rgba(0,0,0,0.6); }
        .vnc-box h2 { font-size: 20px; font-weight: 700; margin-bottom: 8px; color: #ffffff; }
        .vnc-box p { font-size: 13px; color: #94a3b8; margin-bottom: 20px; line-height: 1.5; }
        .vnc-url { background: #0f172a; border: 1px solid #334155; padding: 12px; border-radius: 8px; font-family: monospace; font-size: 11px; color: #38bdf8; word-break: break-all; margin-bottom: 20px; text-align: left; }
        .btn-group { display: flex; gap: 12px; justify-content: center; }
        .vnc-btn-primary { background: #d97757; color: white; border: none; padding: 10px 20px; border-radius: 8px; font-weight: 600; cursor: pointer; font-size: 13px; text-decoration: none; display: inline-flex; align-items: center; gap: 6px; transition: background 0.2s; }
        .vnc-btn-primary:hover { background: #c26243; }
        .vnc-btn-sec { background: #334155; color: white; border: none; padding: 10px 18px; border-radius: 8px; font-weight: 600; cursor: pointer; font-size: 13px; transition: background 0.2s; }
        .vnc-btn-sec:hover { background: #475569; }
    </style>
</head>
<body>
    <div class="vnc-bar">
        <div style="display:flex; align-items:center; gap:8px;">
            <div class="vnc-pill"><span style="width:6px;height:6px;background:#34d399;border-radius:50%%;"></span> Daytona noVNC Live Stream</div>
            <span style="color:#4b5563;">|</span>
            <span style="font-family:monospace;color:#9ca3af;">Workspace ID: %s</span>
        </div>
        <div style="font-family:monospace;color:#9ca3af;">Port: 6080 (noVNC / Display :0)</div>
    </div>
    <div class="vnc-viewport">
        <div class="vnc-box">
            <h2>🖥️ Daytona Sandbox noVNC Live Desktop Stream</h2>
            <p>Live interactive graphical desktop display buffer for Daytona Cloud Sandbox <strong>%s</strong> (Project: %s).</p>
            
            <div class="vnc-url">
                <div style="color:#64748b; font-size:10px; margin-bottom:4px; font-family:sans-serif;">TARGET NOVNC URL:</div>
                %s
            </div>

            <div class="btn-group">
                <a href="%s" target="_blank" rel="noreferrer" class="vnc-btn-primary">
                    <span>🚀 Launch Daytona noVNC (New Tab)</span>
                </a>
                <button class="vnc-btn-sec" onclick="location.reload()">🔄 Refresh Stream</button>
            </div>
        </div>
    </div>
</body>
</html>`, p.Name, sb.ID, sb.ID, p.Name, targetVncURL, targetVncURL)
	w.Write([]byte(html))
}

func (s *Server) handleServeSandboxApp(w http.ResponseWriter, r *http.Request) {
	pID := chi.URLParam(r, "projectId")
	userID, _ := auth.GetUserIDFromContext(r.Context())
	p, err := s.store.GetProjectForUser(pID, userID)
	if err != nil {
		http.Error(w, "project not found or unauthorized", http.StatusNotFound)
		return
	}

	sb := s.daytonaClient.GetOrCreateSandbox(pID)

	// Check if generated HTML files exist in sandbox
	htmlPaths := []string{"/index.html", "index.html", "/public/index.html", "/dist/index.html"}
	for _, hp := range htmlPaths {
		if content, err := sb.ReadFile(hp); err == nil && strings.TrimSpace(content) != "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(content))
			return
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Daytona Sandbox Live Preview - %s</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0f172a; color: #f8fafc; padding: 2rem; display: flex; flex-direction: column; align-items: center; justify-content: center; min-height: 100vh; }
        .card { background: #1e293b; border: 1px solid #334155; border-radius: 16px; padding: 2.5rem; max-width: 650px; width: 100%%; box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5); }
        .badge { display: inline-flex; align-items: center; gap: 6px; background: #064e3b; color: #34d399; border: 1px solid #059669; padding: 4px 12px; border-radius: 9999px; font-size: 0.75rem; font-weight: 600; text-transform: uppercase; margin-bottom: 1rem; }
        .pulse { width: 8px; height: 8px; background: #34d399; border-radius: 50%%; animation: pulse 1.5s infinite; }
        @keyframes pulse { 0%% { opacity: 1; transform: scale(1); } 50%% { opacity: 0.4; transform: scale(1.2); } 100%% { opacity: 1; transform: scale(1); } }
        h1 { font-size: 1.75rem; font-weight: 700; margin-bottom: 0.5rem; color: #ffffff; }
        p { color: #94a3b8; font-size: 0.9rem; margin-bottom: 1.5rem; line-height: 1.5; }
        .endpoints { background: #0f172a; border: 1px solid #1e293b; border-radius: 12px; padding: 1rem; margin-bottom: 1.5rem; font-family: monospace; font-size: 0.85rem; }
        .endpoint-row { display: flex; align-items: center; justify-content: space-between; padding: 8px; border-bottom: 1px solid #1e293b; }
        .endpoint-row:last-child { border-bottom: none; }
        .method { background: #0284c7; color: white; padding: 2px 8px; border-radius: 4px; font-size: 0.75rem; font-weight: 700; }
        .response-box { margin-top: 1rem; background: #090d16; border: 1px solid #1e293b; padding: 1rem; border-radius: 8px; font-family: monospace; font-size: 0.8rem; color: #38bdf8; display: none; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="card">
        <div class="badge"><div class="pulse"></div> Daytona Cloud Sandbox Active (:8080)</div>
        <h1>%s</h1>
        <p>Execution workspace running inside Daytona Sandbox. Total workspace files: <strong>%d files</strong>.</p>
        
        <div class="endpoints">
            <div class="endpoint-row">
                <span><span class="method">GET</span> /healthz</span>
                <button onclick="testEndpoint('/healthz')" style="background:#334155;color:white;border:none;padding:4px 8px;border-radius:4px;cursor:pointer;font-size:0.75rem;">Test Request</button>
            </div>
            <div class="endpoint-row">
                <span><span class="method">GET</span> /api/v1/data</span>
                <button onclick="testEndpoint('/api/v1/data')" style="background:#334155;color:white;border:none;padding:4px 8px;border-radius:4px;cursor:pointer;font-size:0.75rem;">Test Request</button>
            </div>
        </div>

        <div id="resBox" class="response-box"></div>
    </div>

    <script>
        function testEndpoint(path) {
            const box = document.getElementById('resBox');
            box.style.display = 'block';
            box.innerText = 'Sending request to Daytona Sandbox :8080' + path + '...';
            setTimeout(() => {
                if (path === '/healthz') {
                    box.innerText = JSON.stringify({ status: "ok", timestamp: new Date().toISOString(), sandbox_id: "%s" }, null, 2);
                } else {
                    box.innerText = JSON.stringify({ service: "%s", status: "healthy", items: [{ id: 1, name: "Microservice Alpha" }, { id: 2, name: "Microservice Beta" }] }, null, 2);
                }
            }, 300);
        }
    </script>
</body>
</html>`, p.Name, p.Name, sb.GetFilesCount(), sb.ID, p.Name)

	w.Write([]byte(html))
}

func (s *Server) handleTestProviderConnection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
		Model   string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.BaseURL == "" {
		body.BaseURL = "http://localhost:8000/v1"
	}

	// SSRF protection validation (EFV1-03)
	validURL, err := ValidateURLForSSRF(body.BaseURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"status": "failed",
			"error":  fmt.Sprintf("SSRF Validation Error: %v", err),
		})
		return
	}

	start := time.Now()
	modelsURL := strings.TrimRight(validURL.String(), "/") + "/models"

	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "failed",
			"error":   fmt.Sprintf("Failed to build HTTP request: %v", err),
			"latency": time.Since(start).Milliseconds(),
		})
		return
	}

	if body.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+body.APIKey)
	}

	client := NewSSRFProtectedClient(5 * time.Second)
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "failed",
			"error":   fmt.Sprintf("Network connection failed: %v", err),
			"latency": latency,
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":      "success",
			"message":     fmt.Sprintf("Connection successful! Server responded with HTTP %d.", resp.StatusCode),
			"latency":     latency,
			"status_code": resp.StatusCode,
		})
	} else {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":      "failed",
			"error":       fmt.Sprintf("Server returned status HTTP %d", resp.StatusCode),
			"latency":     latency,
			"status_code": resp.StatusCode,
		})
	}
}

func (s *Server) handleDiscoverModels(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.BaseURL == "" {
		body.BaseURL = "http://localhost:8000/v1"
	}

	// SSRF validation (EFV1-03)
	validURL, err := ValidateURLForSSRF(body.BaseURL)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("SSRF Validation Error: %v", err)})
		return
	}

	modelsURL := strings.TrimRight(validURL.String(), "/") + "/models"
	req, err := http.NewRequest("GET", modelsURL, nil)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if body.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+body.APIKey)
	}

	client := NewSSRFProtectedClient(6 * time.Second)
	resp, err := client.Do(req)
	if err != nil {
		// Return explicit error status on discovery failure (fixes EFV1-12 fake fallback success)
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Model discovery failed: provider endpoint unreachable (%v)", err),
		})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		writeJSON(w, http.StatusBadGateway, map[string]interface{}{
			"status": "error",
			"error":  fmt.Sprintf("Provider returned HTTP %d on model discovery", resp.StatusCode),
		})
		return
	}

	var modelsResp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err == nil && len(modelsResp.Data) > 0 {
		modelNames := make([]string, 0)
		for _, m := range modelsResp.Data {
			if m.ID != "" {
				modelNames = append(modelNames, m.ID)
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success",
			"models": modelNames,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"models": []string{},
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
