package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/saas-agent-platform/backend/internal/models"
)

var (
	ErrNotFound = errors.New("record not found")
)

type Store struct {
	mu          sync.RWMutex
	users       map[string]*models.User
	projects    map[string]*models.Project
	messages    map[string][]*models.Message
	skills      map[string]*models.Skill
	secrets     map[string]map[string]string // projectId -> secretType -> secretValue
	jobs        map[string]*models.Job
	deployments map[string]*models.Deployment
}

func NewStore() *Store {
	s := &Store{
		users:       make(map[string]*models.User),
		projects:    make(map[string]*models.Project),
		messages:    make(map[string][]*models.Message),
		skills:      make(map[string]*models.Skill),
		secrets:     make(map[string]map[string]string),
		jobs:        make(map[string]*models.Job),
		deployments: make(map[string]*models.Deployment),
	}
	// Create default dev user
	defaultUser := &models.User{
		ID:        "user-default",
		Email:     "developer@example.com",
		PlanTier:  "Pro Plan",
		CreatedAt: time.Now(),
	}
	s.users[defaultUser.ID] = defaultUser

	// Create default demo project
	demoProject := &models.Project{
		ID:           "proj-default",
		UserID:       defaultUser.ID,
		Name:         "E-Commerce Microservices Platform",
		Status:       "building",
		GitRemoteURL: "https://github.com/example/ecommerce-app.git",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	s.projects[demoProject.ID] = demoProject

	// Create a default skill
	skill1 := &models.Skill{
		ID:          "skill-1",
		UserID:      defaultUser.ID,
		Name:        "Go Microservice Conventions",
		Description: "Enforces Chi router, structured JSON logs, and clean architecture",
		Content:     "# Go Microservice Guidelines\n- Use `chi` router\n- Implement `/healthz`\n- Return standard JSON error responses",
		Source:      "manual",
		CreatedAt:   time.Now(),
	}
	s.skills[skill1.ID] = skill1

	return s
}

// User methods
func (s *Store) GetUserByID(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (s *Store) CreateUser(email, passwordHash string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := &models.User{
		ID:           "user-" + uuid.New().String()[:8],
		Email:        email,
		PasswordHash: passwordHash,
		PlanTier:     "Pro Plan",
		CreatedAt:    time.Now(),
	}
	s.users[u.ID] = u
	return u, nil
}

// Project methods
func (s *Store) ListProjects(userID string) ([]*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*models.Project, 0)
	for _, p := range s.projects {
		if p.UserID == userID || userID == "" {
			list = append(list, p)
		}
	}
	return list, nil
}

func (s *Store) GetProject(id string) (*models.Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	return p, nil
}

func (s *Store) CreateProject(userID, name string) (*models.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if name == "" {
		name = fmt.Sprintf("Project-%s", uuid.New().String()[:6])
	}
	p := &models.Project{
		ID:        "proj-" + uuid.New().String()[:8],
		UserID:    userID,
		Name:      name,
		Status:    "draft",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.projects[p.ID] = p
	return p, nil
}

func (s *Store) UpdateProject(id string, gitRemoteURL string, name string) (*models.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[id]
	if !ok {
		return nil, ErrNotFound
	}
	if gitRemoteURL != "" {
		p.GitRemoteURL = gitRemoteURL
	}
	if name != "" {
		p.Name = name
	}
	p.UpdatedAt = time.Now()
	return p, nil
}

func (s *Store) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.projects, id)
	delete(s.messages, id)
	delete(s.secrets, id)
	return nil
}

// Message methods
func (s *Store) ListMessages(projectID string) ([]*models.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs, ok := s.messages[projectID]
	if !ok {
		return []*models.Message{}, nil
	}
	return msgs, nil
}

func (s *Store) AddMessage(projectID, role, content string) (*models.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	msg := &models.Message{
		ID:        "msg-" + uuid.New().String()[:8],
		ProjectID: projectID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}
	s.messages[projectID] = append(s.messages[projectID], msg)
	return msg, nil
}

// Skill methods
func (s *Store) ListSkills(userID string) ([]*models.Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*models.Skill, 0)
	for _, sk := range s.skills {
		if sk.UserID == userID || userID == "" {
			list = append(list, sk)
		}
	}
	return list, nil
}

func (s *Store) CreateSkill(userID, name, description, content, source string) (*models.Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sk := &models.Skill{
		ID:          "skill-" + uuid.New().String()[:8],
		UserID:      userID,
		Name:        name,
		Description: description,
		Content:     content,
		Source:      source,
		CreatedAt:   time.Now(),
	}
	s.skills[sk.ID] = sk
	return sk, nil
}

func (s *Store) DeleteSkill(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.skills, id)
	return nil
}

// Secrets methods
func (s *Store) SaveSecret(projectID, secretType, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.secrets[projectID]; !ok {
		s.secrets[projectID] = make(map[string]string)
	}
	s.secrets[projectID][secretType] = value
	return nil
}

func (s *Store) GetSecretsPresence(projectID string) models.SecretsPresence {
	s.mu.RLock()
	defer s.mu.RUnlock()
	secMap, ok := s.secrets[projectID]
	if !ok {
		return models.SecretsPresence{}
	}
	_, hasGH := secMap["github_pat"]
	_, hasAz := secMap["azure_credentials"]
	_, hasHF := secMap["huggingface_token"]
	_, hasNIM := secMap["nvidia_nim_token"]
	return models.SecretsPresence{
		GitHubPAT:        hasGH,
		AzureCredentials: hasAz,
		HuggingFaceToken: hasHF,
		NvidiaNimToken:   hasNIM,
	}
}

func (s *Store) GetSecretValue(projectID, secretType string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	secMap, ok := s.secrets[projectID]
	if !ok {
		return "", ErrNotFound
	}
	val, ok := secMap[secretType]
	if !ok {
		return "", ErrNotFound
	}
	return val, nil
}

// Job methods
func (s *Store) CreateJob(projectID, jobType string, payload []byte) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job := &models.Job{
		ID:        "job-" + uuid.New().String()[:8],
		ProjectID: projectID,
		Type:      jobType,
		Status:    "queued",
		Payload:   payload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.jobs[job.ID] = job
	return job, nil
}

func (s *Store) GetJob(id string) (*models.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	return job, nil
}

func (s *Store) UpdateJob(id string, status string, result []byte, errStr *string) (*models.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[id]
	if !ok {
		return nil, ErrNotFound
	}
	job.Status = status
	if result != nil {
		job.Result = result
	}
	if errStr != nil {
		job.Error = errStr
	}
	job.UpdatedAt = time.Now()
	return job, nil
}
