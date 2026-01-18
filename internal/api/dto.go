package api

import "github.com/eann1s/codex-memory-manager/internal/core"

type WriteRequest struct {
	TenantID string         `json:"tenant_id"`
	AppID    string         `json:"app_id"`
	UserID   string         `json:"user_id"`
	Memories []MemoryInput  `json:"memories"`
}

type MemoryInput struct {
	Type            core.MemoryType       `json:"type"`
	Content         string                `json:"content"`
	ImportanceScore float32               `json:"importance_score"`
	Stability       core.MemoryStability  `json:"stability"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type WriteResponse struct {
	WrittenIDs []string       `json:"written_ids"`
	Skipped    []SkippedItem  `json:"skipped"`
}

type SkippedItem struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}
