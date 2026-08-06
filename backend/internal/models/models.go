package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	PasswordHash string    `json:"-"`
	PlanTier     string    `json:"plan_tier"`
	CreatedAt    time.Time `json:"created_at"`
}

type Project struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	UserID       string    `gorm:"index" json:"user_id"`
	Name         string    `json:"name"`
	Status       string    `json:"status"` // draft, building, deployed, error
	GitRemoteURL string    `json:"git_remote_url"`
	SandboxID    *string   `json:"sandbox_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Message struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	ProjectID string    `gorm:"index" json:"project_id"`
	Role      string    `json:"role"` // user, assistant, system-status
	Content   string    `gorm:"type:text" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type Skill struct {
	ID          string    `gorm:"primaryKey" json:"id"`
	UserID      string    `gorm:"index" json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `gorm:"type:text" json:"content"`
	Source      string    `json:"source"` // uploaded, manual
	CreatedAt   time.Time `json:"created_at"`
}

type SecretRef struct {
	ID                    string    `gorm:"primaryKey" json:"id"`
	ProjectID             string    `gorm:"index" json:"project_id"`
	Type                  string    `json:"type"` // github_pat, azure_credentials, huggingface_token, nvidia_nim_token
	KeyVaultSecretName    string    `json:"keyvault_secret_name"`
	KeyVaultSecretVersion string    `json:"keyvault_secret_version"`
	SecretValue           string    `gorm:"type:text" json:"-"`
	CreatedAt             time.Time `json:"created_at"`
}

type Job struct {
	ID        string          `gorm:"primaryKey" json:"id"`
	ProjectID string          `gorm:"index" json:"project_id"`
	Type      string          `json:"type"` // codegen, build_push, deploy_app, deploy_llm, maintain_app, push
	Status    string          `json:"status"` // queued, running, succeeded, failed, cancelled
	Payload   json.RawMessage `gorm:"type:text" json:"payload,omitempty"`
	Result    json.RawMessage `gorm:"type:text" json:"result,omitempty"`
	Error     *string         `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Deployment struct {
	ID          string          `gorm:"primaryKey" json:"id"`
	ProjectID   string          `gorm:"index" json:"project_id"`
	JobID       string          `json:"job_id"`
	Kind        string          `json:"kind"` // app, llm
	EndpointURL string          `json:"endpoint_url"`
	Details     json.RawMessage `gorm:"type:text" json:"details,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type FileItem struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

type GitStatusItem struct {
	Path   string `json:"path"`
	Status string `json:"status"` // A, M, D, ??
}

type SecretsPresence struct {
	GitHubPAT        bool `json:"github_pat"`
	AzureCredentials bool `json:"azure_credentials"`
	HuggingFaceToken bool `json:"huggingface_token"`
	NvidiaNimToken   bool `json:"nvidia_nim_token"`
}

type WSEvent struct {
	Type        string          `json:"type"`
	MessageID   string          `json:"message_id,omitempty"`
	Delta       string          `json:"delta,omitempty"`
	JobID       string          `json:"job_id,omitempty"`
	Agent       string          `json:"agent,omitempty"`
	Text        string          `json:"text,omitempty"`
	Level       string          `json:"level,omitempty"` // info, success, error
	Uncommitted []GitStatusItem `json:"uncommitted,omitempty"`
	Status      string          `json:"status,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
}
