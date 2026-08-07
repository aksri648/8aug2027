package store

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/saas-agent-platform/backend/internal/auth"
	"github.com/saas-agent-platform/backend/internal/models"
	"github.com/saas-agent-platform/backend/internal/secrets"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrUnauthorized  = errors.New("unauthorized: project does not belong to user")
	ErrAlreadyExists = errors.New("user already exists")
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

	// Configure DB Connection Pooling
	sqlDB, err := db.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)
		log.Println("⚡ Configured Database Connection Pool (MaxOpen: 25, MaxIdle: 10, Lifetime: 5m)")
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
	email := os.Getenv("DEFAULT_USER_EMAIL")
	password := os.Getenv("DEFAULT_USER_PASSWORD")
	if email == "" || password == "" {
		return
	}

	var userCount int64
	s.db.Model(&models.User{}).Where("email = ?", email).Count(&userCount)
	if userCount == 0 {
		hashed, _ := auth.HashPassword(password)
		defaultUser := &models.User{
			ID:           "user-env-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			Email:        email,
			PasswordHash: hashed,
			PlanTier:     "Pro Plan",
			CreatedAt:    time.Now(),
		}
		s.db.Create(defaultUser)
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

func (s *Store) AuthenticateUser(email, password string) (*models.User, error) {
	u, err := s.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("invalid email or password")
	}

	if !auth.CheckPasswordHash(password, u.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}

	return u, nil
}

func (s *Store) CreateUser(email, plainPassword string) (*models.User, error) {
	if email == "" || plainPassword == "" {
		return nil, errors.New("email and password are required")
	}

	var existing int64
	s.db.Model(&models.User{}).Where("email = ?", email).Count(&existing)
	if existing > 0 {
		return nil, ErrAlreadyExists
	}

	hashedPassword, err := auth.HashPassword(plainPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	u := &models.User{
		ID:           "user-" + uuid.New().String()[:8],
		Email:        email,
		PasswordHash: hashedPassword,
		PlanTier:     "Pro Plan",
		CreatedAt:    time.Now(),
	}
	if err := s.db.Create(u).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
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

func (s *Store) GetProjectForUser(id, userID string) (*models.Project, error) {
	var p models.Project
	if err := s.db.First(&p, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (s *Store) CreateProject(userID, name string) (*models.Project, error) {
	if userID == "" {
		return nil, errors.New("user_id is required to create a project")
	}
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

func (s *Store) UpdateProjectForUser(id, userID, gitRemoteURL, name string) (*models.Project, error) {
	var p models.Project
	if err := s.db.First(&p, "id = ? AND user_id = ?", id, userID).Error; err != nil {
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
	if err := s.db.Model(&p).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) DeleteProjectForUser(id, userID string) error {
	var p models.Project
	if err := s.db.First(&p, "id = ? AND user_id = ?", id, userID).Error; err != nil {
		return ErrNotFound
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.Project{}, "id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Message{}, "project_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.SecretRef{}, "project_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Job{}, "project_id = ?", id).Error; err != nil {
			return err
		}
		return nil
	})
}

// Message methods
func (s *Store) ListMessages(projectID string) ([]*models.Message, error) {
	var msgs []*models.Message
	if err := s.db.Where("project_id = ?", projectID).Order("created_at asc").Find(&msgs).Error; err != nil {
		return nil, err
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
		return nil, fmt.Errorf("failed to save message: %w", err)
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
		return nil, fmt.Errorf("failed to create skill: %w", err)
	}
	return sk, nil
}

func (s *Store) DeleteSkill(id, userID string) error {
	res := s.db.Delete(&models.Skill{}, "id = ? AND user_id = ?", id, userID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// Secrets methods
func (s *Store) SaveSecret(projectID, secretType, value string) error {
	encryptedVal, err := secrets.EncryptSecret(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	var sec models.SecretRef
	err = s.db.Where("project_id = ? AND type = ?", projectID, secretType).First(&sec).Error
	if err != nil {
		sec = models.SecretRef{
			ID:          "sec-" + uuid.New().String()[:8],
			ProjectID:   projectID,
			Type:        secretType,
			SecretValue: encryptedVal,
			CreatedAt:   time.Now(),
		}
		if err := s.db.Create(&sec).Error; err != nil {
			return fmt.Errorf("failed to save secret: %w", err)
		}
	} else {
		if err := s.db.Model(&sec).Update("secret_value", encryptedVal).Error; err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}
	}
	return nil
}

func (s *Store) GetSecretsPresence(projectID string) models.SecretsPresence {
	var secretRefs []models.SecretRef
	s.db.Where("project_id = ?", projectID).Find(&secretRefs)

	presence := models.SecretsPresence{}
	for _, sec := range secretRefs {
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
	decrypted, err := secrets.DecryptSecret(sec.SecretValue)
	if err != nil {
		return "", err
	}
	return decrypted, nil
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
		return nil, fmt.Errorf("failed to create job: %w", err)
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

func (s *Store) GetJobForUser(id, userID string) (*models.Job, error) {
	var job models.Job
	err := s.db.Joins("JOIN projects ON projects.id = jobs.project_id").
		Where("jobs.id = ? AND projects.user_id = ?", id, userID).
		First(&job).Error
	if err != nil {
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
	if err := s.db.Model(&job).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}
	return &job, nil
}
