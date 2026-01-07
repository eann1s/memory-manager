package core

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type MemoryType string

const (
	MemoryTypeProfile   MemoryType = "profile"
	MemoryTypePreference MemoryType = "preference"
	MemoryTypeProject   MemoryType = "project"
	MemoryTypeEpisodic  MemoryType = "episodic"
	MemoryTypeKnowledge MemoryType = "knowledge"
	MemoryTypeOther     MemoryType = "other"
)

type MemoryStability string

const (
	MemoryStabilityShortTerm MemoryStability = "short_term"
	MemoryStabilityLongTerm  MemoryStability = "long_term"
)

type Tenant struct {
	ID         int64                  `json:"id"`
	ExternalID string                 `json:"external_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

type App struct {
	ID         int64                  `json:"id"`
	TenantID   int64                  `json:"tenant_id"`
	ExternalID string                 `json:"external_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

type User struct {
	ID         int64                  `json:"id"`
	TenantID   int64                  `json:"tenant_id"`
	AppID      int64                  `json:"app_id"`
	ExternalID string                 `json:"external_id"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

type Memory struct {
	ID               uuid.UUID              `json:"id"`
	TenantID         int64                  `json:"tenant_id"`
	AppID            int64                  `json:"app_id"`
	UserID           int64                  `json:"user_id"`
	MemoryType       MemoryType             `json:"memory_type"`
	Content          string                 `json:"content"`
	ImportanceScore  float32                `json:"importance_score"`
	MemoryStability  MemoryStability        `json:"memory_stability"`
	Embedding        pgvector.Vector        `json:"embedding,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type Event struct {
	ID             uuid.UUID              `json:"id"`
	TenantID       int64                  `json:"tenant_id"`
	AppID          int64                  `json:"app_id"`
	UserID         int64                  `json:"user_id"`
	ConversationID string                 `json:"conversation_id"`
	Role           string                 `json:"role"`
	Content        string                 `json:"content"`
	Metadata       map[string]interface{} `json:"metadata"`
	Timestamp      time.Time              `json:"timestamp"`
}
