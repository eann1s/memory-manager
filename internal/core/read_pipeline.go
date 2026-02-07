package core

import (
	"context"
	"fmt"
	"strings"
)

type ReadPipelineConfig struct {
	ImportanceThreshold float32
	MaxQueryLen         int
	MaxItems            int
	ExpectedDim         int
}

type ReadPipeline struct {
	repos             *Repos
	embeddingProvider EmbeddingProvider
	config            ReadPipelineConfig
}

func NewReadPipeline(repos *Repos, embeddingProvider EmbeddingProvider, config ReadPipelineConfig) *ReadPipeline {
	if repos == nil {
		panic("repos cannot be nil")
	}
	if embeddingProvider == nil {
		panic("embeddingProvider cannot be nil")
	}
	if config.ExpectedDim <= 0 {
		panic("config.ExpectedDim must be > 0")
	}
	if config.MaxQueryLen <= 0 {
		panic("config.MaxQueryLen must be > 0")
	}
	if config.MaxItems <= 0 {
		panic("config.MaxItems must be > 0")
	}
	if config.ImportanceThreshold < 0 || config.ImportanceThreshold > 1 {
		panic("config.ImportanceThreshold must be in range [0, 1]")
	}
	return &ReadPipeline{
		repos:             repos,
		embeddingProvider: embeddingProvider,
		config:            config,
	}
}

type ReadInput struct {
	Query string
	Types []string
	Limit int
}

type ReadResult struct {
	Memories []*Memory
}

func (p *ReadPipeline) Read(ctx context.Context, tenantID, appID, userID string, input ReadInput) (*ReadResult, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if appID == "" {
		return nil, ErrMissingAppID
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	if input.Query == "" {
		return nil, ErrMissingQuery
	}

	if strings.TrimSpace(input.Query) == "" {
		return nil, ErrEmptyQuery
	}

	if len(input.Query) > p.config.MaxQueryLen {
		return nil, fmt.Errorf("%w: got %d bytes, max %d bytes", ErrQueryTooLong, len(input.Query), p.config.MaxQueryLen)
	}

	limit := input.Limit
	if limit <= 0 || limit > p.config.MaxItems {
		limit = p.config.MaxItems
	}

	for _, t := range input.Types {
		if !isValidMemoryType(MemoryType(t)) {
			return nil, fmt.Errorf("%w: %s", ErrInvalidMemoryType, t)
		}
	}

	tenant, err := p.repos.Tenants.GetOrCreate(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTenantResolution, err)
	}

	app, err := p.repos.Apps.GetOrCreate(ctx, tenant.ID, appID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAppResolution, err)
	}

	user, err := p.repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUserResolution, err)
	}

	embedding, err := p.embeddingProvider.Embed(ctx, input.Query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEmbeddingGeneration, err)
	}

	if len(embedding) != p.config.ExpectedDim {
		return nil, fmt.Errorf("%w: got %d, expected %d", ErrEmbeddingDimension, len(embedding), p.config.ExpectedDim)
	}

	memories, err := p.repos.Memories.SearchByEmbedding(ctx, tenant.ID, app.ID, user.ID, embedding, limit, input.Types)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMemorySearch, err)
	}

	filtered := make([]*Memory, 0, len(memories))
	for _, m := range memories {
		if m.ImportanceScore >= p.config.ImportanceThreshold {
			filtered = append(filtered, m)
		}
	}

	return &ReadResult{Memories: filtered}, nil
}
