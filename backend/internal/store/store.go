package store

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/saas-agent-platform/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ErrNotFound = errors.New("record not found")
)

type Store struct {
	db *gorm.DB
	mu sync.RWMutex
}

func NewStore() *Store {
	var db *gorm.DB
	var err error

	pgDSN := os.Getenv("DATABASE_URL")
	if pgDSN == "" {
		pgDSN = os.Getenv("POSTGRES_ADDR")
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	if pgDSN != "" {
		log.Printf("🐘 Connecting to PostgreSQL Database: %s", pgDSN)
		db, err = gorm.Open(postgres.Open(pgDSN), gormConfig)
		if err != nil {
			log.Printf("⚠️ PostgreSQL connection failed: %v. Falling back to SQLite persistent engine.", err)
			db, err = gorm.Open(sqlite.Open("saas_platform.db"), gormConfig)
		}
	} else {
		log.Println("🗄️ Using SQLite Persistent Database Engine (saas_platform.db)")
		db, err = gorm.Open(sqlite.Open("saas_platform.db"), gormConfig)
	}

	if err != nil {
		log.Fatalf("Fatal: failed to initialize database store: %v", err)
	}

	// Run GORM AutoMigrations
	err = db.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Message{},
		&models.Skill{},
		&models.SecretRef{},
		&models.Job{},
		&models.Deployment{},
	)
	if err != nil {
		log.Printf("⚠️ Database auto-migration warning: %v", err)
	}

	s := &Store{db: db}
	s.seedDefaultData()
	return s
}

func (s *Store) seedDefaultData() {
	// Seed Default User
	var userCount int64
	s.db.Model(&models.User{}).Where("id = ?", "user-default").Count(&userCount)
	if userCount == 0 {
		defaultUser := &models.User{
			ID:        "user-default",
			Email:     "developer@example.com",
			PlanTier:  "Pro Plan",
			CreatedAt: time.Now(),
		}
		s.db.Create(defaultUser)
	}

	// Seed Demo Project
	var projCount int64
	s.db.Model(&models.Project{}).Where("id = ?", "proj-default").Count(&projCount)
	if projCount == 0 {
		demoProject := &models.Project{
			ID:           "proj-default",
			UserID:       "user-default",
			Name:         "E-Commerce Microservices Platform",
			Status:       "building",
			GitRemoteURL: "https://github.com/example/ecommerce-app.git",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		s.db.Create(demoProject)
	}

	// Seed Default Skill
	var skillCount int64
	s.db.Model(&models.Skill{}).Where("id = ?", "skill-1").Count(&skillCount)
	if skillCount == 0 {
		skill1 := &models.Skill{
			ID:          "skill-1",
			UserID:      "user-default",
			Name:        "Go Microservice Conventions",
			Description: "Enforces Chi router, structured JSON logs, and clean architecture",
			Content:     "# Go Microservice Guidelines\n- Use `chi` router\n- Implement `/healthz`\n- Return standard JSON error responses",
			Source:      "manual",
			CreatedAt:   time.Now(),
		}
		s.db.Create(skill1)
	}
}

// User methods
func (s *Store) GetUserByID(id string) (*models.User, error) {
	var u models.User
	if err := s.db.First(&u, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &u, nil
}

func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	var u models.User
	if err := s.db.First(&u, "email = ?", email).Error; err != nil {
		return nil, ErrNotFound
	}
	return &u, nil
}

func (s *Store) CreateUser(email, passwordHash string) (*models.User, error) {
	u := &models.User{
		ID:           "user-" + uuid.New().String()[:8],
		Email:        email,
		PasswordHash: passwordHash,
		PlanTier:     "Pro Plan",
		CreatedAt:    time.Now(),
	}
	if err := s.db.Create(u).Error; err != nil {
		return nil, err
	}
	return u, nil
}

// Project methods
func (s *Store) ListProjects(userID string) ([]*models.Project, error) {
	var projects []*models.Project
	query := s.db.Order("created_at desc")
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *Store) GetProject(id string) (*models.Project, error) {
	var p models.Project
	if err := s.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *Store) CreateProject(userID, name string) (*models.Project, error) {
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
	if err := s.db.Create(p).Error; err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) UpdateProject(id string, gitRemoteURL string, name string) (*models.Project, error) {
	var p models.Project
	if err := s.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	updates := map[string]interface{}{"updated_at": time.Now()}
	if gitRemoteURL != "" {
		updates["git_remote_url"] = gitRemoteURL
		p.GitRemoteURL = gitRemoteURL
	}
	if name != "" {
		updates["name"] = name
		p.Name = name
	}
	s.db.Model(&p).Updates(updates)
	return &p, nil
}

func (s *Store) DeleteProject(id string) error {
	s.db.Delete(&models.Project{}, "id = ?", id)
	s.db.Delete(&models.Message{}, "project_id = ?", id)
	s.db.Delete(&models.SecretRef{}, "project_id = ?", id)
	return nil
}

// Message methods
func (s *Store) ListMessages(projectID string) ([]*models.Message, error) {
	var msgs []*models.Message
	if err := s.db.Where("project_id = ?", projectID).Order("created_at asc").Find(&msgs).Error; err != nil {
		return []*models.Message{}, nil
	}
	return msgs, nil
}

func (s *Store) AddMessage(projectID, role, content string) (*models.Message, error) {
	msg := &models.Message{
		ID:        "msg-" + uuid.New().String()[:8],
		ProjectID: projectID,
		Role:      role,
		Content:   content,
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(msg).Error; err != nil {
		return nil, err
	}
	return msg, nil
}

// Skill methods
func (s *Store) ListSkills(userID string) ([]*models.Skill, error) {
	var skills []*models.Skill
	query := s.db.Order("created_at desc")
	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

func (s *Store) CreateSkill(userID, name, description, content, source string) (*models.Skill, error) {
	sk := &models.Skill{
		ID:          "skill-" + uuid.New().String()[:8],
		UserID:      userID,
		Name:        name,
		Description: description,
		Content:     content,
		Source:      source,
		CreatedAt:   time.Now(),
	}
	if err := s.db.Create(sk).Error; err != nil {
		return nil, err
	}
	return sk, nil
}

func (s *Store) DeleteSkill(id string) error {
	s.db.Delete(&models.Skill{}, "id = ?", id)
	return nil
}

// Secrets methods
func (s *Store) SaveSecret(projectID, secretType, value string) error {
	var sec models.SecretRef
	err := s.db.Where("project_id = ? AND type = ?", projectID, secretType).First(&sec).Error
	if err != nil {
		sec = models.SecretRef{
			ID:          "sec-" + uuid.New().String()[:8],
			ProjectID:   projectID,
			Type:        secretType,
			SecretValue: value,
			CreatedAt:   time.Now(),
		}
		s.db.Create(&sec)
	} else {
		s.db.Model(&sec).Update("secret_value", value)
	}
	return nil
}

func (s *Store) GetSecretsPresence(projectID string) models.SecretsPresence {
	var secrets []models.SecretRef
	s.db.Where("project_id = ?", projectID).Find(&secrets)

	presence := models.SecretsPresence{}
	for _, sec := range secrets {
		switch sec.Type {
		case "github_pat":
			presence.GitHubPAT = true
		case "azure_credentials":
			presence.AzureCredentials = true
		case "huggingface_token":
			presence.HuggingFaceToken = true
		case "nvidia_nim_token":
			presence.NvidiaNimToken = true
		}
	}
	return presence
}

func (s *Store) GetSecretValue(projectID, secretType string) (string, error) {
	var sec models.SecretRef
	if err := s.db.Where("project_id = ? AND type = ?", projectID, secretType).First(&sec).Error; err != nil {
		return "", ErrNotFound
	}
	return sec.SecretValue, nil
}

// Job methods
func (s *Store) CreateJob(projectID, jobType string, payload []byte) (*models.Job, error) {
	job := &models.Job{
		ID:        "job-" + uuid.New().String()[:8],
		ProjectID: projectID,
		Type:      jobType,
		Status:    "queued",
		Payload:   payload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.db.Create(job).Error; err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Store) GetJob(id string) (*models.Job, error) {
	var job models.Job
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	return &job, nil
}

func (s *Store) UpdateJob(id string, status string, result []byte, errStr *string) (*models.Job, error) {
	var job models.Job
	if err := s.db.First(&job, "id = ?", id).Error; err != nil {
		return nil, ErrNotFound
	}
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if result != nil {
		updates["result"] = result
		job.Result = result
	}
	if errStr != nil {
		updates["error"] = errStr
		job.Error = errStr
	}
	job.Status = status
	s.db.Model(&job).Updates(updates)
	return &job, nil
}
