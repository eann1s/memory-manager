package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type EmbeddingProvider interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type WritePipelineConfig struct {
	ImportanceThreshold float32
	MaxContentLen       int
	ExpectedDim         int
}

type WritePipeline struct {
	repos             *Repos
	embeddingProvider EmbeddingProvider
	config            WritePipelineConfig
}

func NewWritePipeline(repos *Repos, embeddingProvider EmbeddingProvider, config WritePipelineConfig) *WritePipeline {
	if repos == nil {
		panic("repos cannot be nil")
	}
	if embeddingProvider == nil {
		panic("embeddingProvider cannot be nil")
	}
	if config.ExpectedDim <= 0 {
		panic("config.ExpectedDim must be > 0")
	}
	if config.MaxContentLen <= 0 {
		panic("config.MaxContentLen must be > 0")
	}
	if config.ImportanceThreshold < 0 || config.ImportanceThreshold > 1 {
		panic("config.ImportanceThreshold must be in range [0, 1]")
	}
	return &WritePipeline{
		repos:             repos,
		embeddingProvider: embeddingProvider,
		config:            config,
	}
}

type WriteInput struct {
	Type            MemoryType
	Content         string
	ImportanceScore float32
	Stability       MemoryStability
	Metadata        map[string]interface{}
}

type WriteResult struct {
	WrittenIDs []uuid.UUID
	Skipped    []SkippedMemory
}

type SkippedMemory struct {
	Index  int
	Reason string
}

func (p *WritePipeline) Write(ctx context.Context, tenantID, appID, userID string, inputs []WriteInput) (*WriteResult, error) {
	if tenantID == "" {
		return nil, ErrMissingTenantID
	}
	if appID == "" {
		return nil, ErrMissingAppID
	}
	if userID == "" {
		return nil, ErrMissingUserID
	}

	result := &WriteResult{
		WrittenIDs: make([]uuid.UUID, 0),
		Skipped:    make([]SkippedMemory, 0),
	}

	eligibleIndices := make([]int, 0, len(inputs))
	for i, input := range inputs {
		if reason := p.validateInput(input); reason != "" {
			result.Skipped = append(result.Skipped, SkippedMemory{Index: i, Reason: reason})
			continue
		}

		if input.ImportanceScore < p.config.ImportanceThreshold {
			result.Skipped = append(result.Skipped, SkippedMemory{
				Index:  i,
				Reason: fmt.Sprintf("importance_score %.2f below threshold %.2f", input.ImportanceScore, p.config.ImportanceThreshold),
			})
			continue
		}

		eligibleIndices = append(eligibleIndices, i)
	}

	if len(eligibleIndices) == 0 {
		return result, nil
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

	for _, i := range eligibleIndices {
		input := inputs[i]

		embedding, err := p.embeddingProvider.Embed(ctx, input.Content)
		if err != nil {
			return nil, fmt.Errorf("%w for memory %d: %v", ErrEmbeddingGeneration, i, err)
		}

		if len(embedding) != p.config.ExpectedDim {
			return nil, fmt.Errorf("%w for memory %d: got %d, expected %d", ErrEmbeddingDimension, i, len(embedding), p.config.ExpectedDim)
		}

		memory := &Memory{
			TenantID:        tenant.ID,
			AppID:           app.ID,
			UserID:          user.ID,
			MemoryType:      input.Type,
			Content:         input.Content,
			ImportanceScore: input.ImportanceScore,
			MemoryStability: input.Stability,
			Embedding:       pgvector.NewVector(embedding),
			Metadata:        input.Metadata,
		}

		if memory.Metadata == nil {
			memory.Metadata = make(map[string]interface{})
		}

		if err := p.repos.Memories.Insert(ctx, memory); err != nil {
			return nil, fmt.Errorf("%w %d: %v", ErrMemoryInsertion, i, err)
		}

		result.WrittenIDs = append(result.WrittenIDs, memory.ID)
	}

	return result, nil
}

func (p *WritePipeline) validateInput(input WriteInput) string {
	if input.Content == "" {
		return "content cannot be empty"
	}

	if strings.TrimSpace(input.Content) == "" {
		return "content cannot be only whitespace"
	}

	if len(input.Content) > p.config.MaxContentLen {
		return fmt.Sprintf("content length %d bytes exceeds maximum %d bytes", len(input.Content), p.config.MaxContentLen)
	}

	if !isValidMemoryType(input.Type) {
		return fmt.Sprintf("invalid memory type: %s", input.Type)
	}

	if !isValidMemoryStability(input.Stability) {
		return fmt.Sprintf("invalid memory stability: %s", input.Stability)
	}

	if input.ImportanceScore < 0 || input.ImportanceScore > 1 {
		return fmt.Sprintf("importance_score %.2f out of range [0, 1]", input.ImportanceScore)
	}

	return ""
}

func isValidMemoryType(mt MemoryType) bool {
	switch mt {
	case MemoryTypeProfile, MemoryTypePreference, MemoryTypeProject, MemoryTypeEpisodic, MemoryTypeKnowledge, MemoryTypeOther:
		return true
	default:
		return false
	}
}

func isValidMemoryStability(ms MemoryStability) bool {
	switch ms {
	case MemoryStabilityShortTerm, MemoryStabilityLongTerm:
		return true
	default:
		return false
	}
}
