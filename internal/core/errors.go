package core

import "errors"

var (
	ErrMissingTenantID     = errors.New("tenant_id is required")
	ErrMissingAppID        = errors.New("app_id is required")
	ErrMissingUserID       = errors.New("user_id is required")
	ErrTenantResolution    = errors.New("failed to resolve tenant")
	ErrAppResolution       = errors.New("failed to resolve app")
	ErrUserResolution      = errors.New("failed to resolve user")
	ErrEmbeddingGeneration = errors.New("failed to generate embedding")
	ErrEmbeddingDimension  = errors.New("embedding dimension mismatch")
	ErrMemoryInsertion     = errors.New("failed to insert memory")
)
