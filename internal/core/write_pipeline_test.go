package core

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type mockTenantRepo struct {
	getOrCreateFunc func(ctx context.Context, externalID string) (*Tenant, error)
}

func (m *mockTenantRepo) GetOrCreate(ctx context.Context, externalID string) (*Tenant, error) {
	return m.getOrCreateFunc(ctx, externalID)
}

type mockAppRepo struct {
	getOrCreateFunc func(ctx context.Context, tenantID int64, externalID string) (*App, error)
}

func (m *mockAppRepo) GetOrCreate(ctx context.Context, tenantID int64, externalID string) (*App, error) {
	return m.getOrCreateFunc(ctx, tenantID, externalID)
}

type mockUserRepo struct {
	getOrCreateFunc func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error)
}

func (m *mockUserRepo) GetOrCreate(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
	return m.getOrCreateFunc(ctx, tenantID, appID, externalID)
}

type mockMemoryRepo struct {
	insertFunc    func(ctx context.Context, memory *Memory) error
	listByUserFunc func(ctx context.Context, tenantID int64, appID int64, userID int64, limit int) ([]*Memory, error)
}

func (m *mockMemoryRepo) Insert(ctx context.Context, memory *Memory) error {
	return m.insertFunc(ctx, memory)
}

func (m *mockMemoryRepo) ListByUser(ctx context.Context, tenantID int64, appID int64, userID int64, limit int) ([]*Memory, error) {
	return m.listByUserFunc(ctx, tenantID, appID, userID, limit)
}

type mockEventRepo struct {
	insertFunc      func(ctx context.Context, event *Event) error
	listByUserFunc  func(ctx context.Context, tenantID int64, appID int64, userID int64, conversationID string, limit int) ([]*Event, error)
}

func (m *mockEventRepo) Insert(ctx context.Context, event *Event) error {
	return m.insertFunc(ctx, event)
}

func (m *mockEventRepo) ListByUser(ctx context.Context, tenantID int64, appID int64, userID int64, conversationID string, limit int) ([]*Event, error) {
	return m.listByUserFunc(ctx, tenantID, appID, userID, conversationID, limit)
}

type mockEmbeddingProvider struct {
	embedFunc func(ctx context.Context, text string) ([]float32, error)
}

func (m *mockEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	return m.embedFunc(ctx, text)
}

func TestWritePipeline_HappyPath(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				memory.ID = uuid.New()
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "User prefers dark mode",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.WrittenIDs) != 1 {
		t.Fatalf("expected 1 written ID, got %d", len(result.WrittenIDs))
	}

	if len(result.Skipped) != 0 {
		t.Fatalf("expected 0 skipped items, got %d", len(result.Skipped))
	}
}

func TestWritePipeline_InvalidEnum(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called for invalid memory")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatalf("embed should not be called for invalid memory")
			return nil, nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryType("invalid"),
			Content:         "Some content",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.WrittenIDs) != 0 {
		t.Fatalf("expected 0 written IDs, got %d", len(result.WrittenIDs))
	}

	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped item, got %d", len(result.Skipped))
	}

	if result.Skipped[0].Index != 0 {
		t.Fatalf("expected skipped index 0, got %d", result.Skipped[0].Index)
	}

	if result.Skipped[0].Reason != "invalid memory type: invalid" {
		t.Fatalf("unexpected skip reason: %s", result.Skipped[0].Reason)
	}
}

func TestWritePipeline_EmptyContent(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called for invalid memory")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatalf("embed should not be called for invalid memory")
			return nil, nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped item, got %d", len(result.Skipped))
	}

	if result.Skipped[0].Reason != "content cannot be empty" {
		t.Fatalf("unexpected skip reason: %s", result.Skipped[0].Reason)
	}
}

func TestWritePipeline_WhitespaceContent(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called for invalid memory")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatalf("embed should not be called for invalid memory")
			return nil, nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "   \t\n  ",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped item, got %d", len(result.Skipped))
	}

	if result.Skipped[0].Reason != "content cannot be only whitespace" {
		t.Fatalf("unexpected skip reason: %s", result.Skipped[0].Reason)
	}
}

func TestWritePipeline_OutOfRangeImportance(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called for invalid memory")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatalf("embed should not be called for invalid memory")
			return nil, nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	testCases := []struct {
		name            string
		importanceScore float32
	}{
		{"negative", -0.1},
		{"greater than 1", 1.5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputs := []WriteInput{
				{
					Type:            MemoryTypeProfile,
					Content:         "Some content",
					ImportanceScore: tc.importanceScore,
					Stability:       MemoryStabilityLongTerm,
				},
			}

			result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(result.Skipped) != 1 {
				t.Fatalf("expected 1 skipped item, got %d", len(result.Skipped))
			}
		})
	}
}

func TestWritePipeline_BelowThreshold(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called for memory below threshold")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatalf("embed should not be called for memory below threshold")
			return nil, nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.5,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "Low importance memory",
			ImportanceScore: 0.4,
			Stability:       MemoryStabilityShortTerm,
		},
	}

	result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.WrittenIDs) != 0 {
		t.Fatalf("expected 0 written IDs, got %d", len(result.WrittenIDs))
	}

	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped item, got %d", len(result.Skipped))
	}

	if result.Skipped[0].Reason != "importance_score 0.40 below threshold 0.50" {
		t.Fatalf("unexpected skip reason: %s", result.Skipped[0].Reason)
	}
}

func TestWritePipeline_ContentTooLong(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called for content too long")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			t.Fatalf("embed should not be called for content too long")
			return nil, nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       100,
		ExpectedDim:         1536,
	})

	longContent := make([]byte, 101)
	for i := range longContent {
		longContent[i] = 'a'
	}

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         string(longContent),
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped item, got %d", len(result.Skipped))
	}

	if result.Skipped[0].Reason != "content length 101 bytes exceeds maximum 100 bytes" {
		t.Fatalf("unexpected skip reason: %s", result.Skipped[0].Reason)
	}
}

func TestWritePipeline_ProviderError(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called when provider fails")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return nil, errors.New("provider unavailable")
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "Valid content",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	_, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestWritePipeline_DimensionMismatch(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				t.Fatalf("insert should not be called with wrong dimension")
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 512), nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "Valid content",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	_, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err == nil {
		t.Fatalf("expected error for dimension mismatch, got nil")
	}
}

func TestWritePipeline_MixedValidInvalid(t *testing.T) {
	ctx := context.Background()

	insertCount := 0
	embedCount := 0

	repos := &Repos{
		Tenants: &mockTenantRepo{
			getOrCreateFunc: func(ctx context.Context, externalID string) (*Tenant, error) {
				return &Tenant{ID: 1, ExternalID: externalID}, nil
			},
		},
		Apps: &mockAppRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, externalID string) (*App, error) {
				return &App{ID: 2, TenantID: tenantID, ExternalID: externalID}, nil
			},
		},
		Users: &mockUserRepo{
			getOrCreateFunc: func(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error) {
				return &User{ID: 3, TenantID: tenantID, AppID: appID, ExternalID: externalID}, nil
			},
		},
		Memories: &mockMemoryRepo{
			insertFunc: func(ctx context.Context, memory *Memory) error {
				insertCount++
				memory.ID = uuid.New()
				return nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			embedCount++
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "Valid memory 1",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
		{
			Type:            MemoryType("invalid"),
			Content:         "Invalid type",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
		{
			Type:            MemoryTypePreference,
			Content:         "Valid memory 2",
			ImportanceScore: 0.7,
			Stability:       MemoryStabilityShortTerm,
		},
		{
			Type:            MemoryTypeKnowledge,
			Content:         "Below threshold",
			ImportanceScore: 0.2,
			Stability:       MemoryStabilityShortTerm,
		},
	}

	result, err := pipeline.Write(ctx, "tenant-1", "app-1", "user-1", inputs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.WrittenIDs) != 2 {
		t.Fatalf("expected 2 written IDs, got %d", len(result.WrittenIDs))
	}

	if len(result.Skipped) != 2 {
		t.Fatalf("expected 2 skipped items, got %d", len(result.Skipped))
	}

	if result.Skipped[0].Index != 1 {
		t.Fatalf("expected first skip at index 1, got %d", result.Skipped[0].Index)
	}

	if result.Skipped[1].Index != 3 {
		t.Fatalf("expected second skip at index 3, got %d", result.Skipped[1].Index)
	}

	if insertCount != 2 {
		t.Fatalf("expected 2 inserts, got %d", insertCount)
	}

	if embedCount != 2 {
		t.Fatalf("expected 2 embed calls, got %d", embedCount)
	}
}

func TestWritePipeline_MissingScope(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{}
	provider := &mockEmbeddingProvider{}

	pipeline := NewWritePipeline(repos, provider, WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	inputs := []WriteInput{
		{
			Type:            MemoryTypeProfile,
			Content:         "Valid content",
			ImportanceScore: 0.8,
			Stability:       MemoryStabilityLongTerm,
		},
	}

	testCases := []struct {
		name     string
		tenantID string
		appID    string
		userID   string
	}{
		{"missing tenant", "", "app-1", "user-1"},
		{"missing app", "tenant-1", "", "user-1"},
		{"missing user", "tenant-1", "app-1", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pipeline.Write(ctx, tc.tenantID, tc.appID, tc.userID, inputs)
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
		})
	}
}
