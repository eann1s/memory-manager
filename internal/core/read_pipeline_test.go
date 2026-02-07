package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestReadPipeline_HappyPath(t *testing.T) {
	ctx := context.Background()

	memoryID := uuid.New()
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
			searchByEmbeddingFunc: func(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*Memory, error) {
				return []*Memory{
					{
						ID:              memoryID,
						TenantID:        tenantID,
						AppID:           appID,
						UserID:          userID,
						MemoryType:      MemoryTypeProfile,
						Content:         "User prefers dark mode",
						ImportanceScore: 0.8,
						MemoryStability: MemoryStabilityLongTerm,
						Metadata:        map[string]interface{}{},
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					},
				}, nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "dark mode preference",
		Limit: 10,
	}

	result, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(result.Memories))
	}

	if result.Memories[0].ID != memoryID {
		t.Fatalf("unexpected memory ID")
	}
}

func TestReadPipeline_MissingScope(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{}
	provider := &mockEmbeddingProvider{}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Limit: 10,
	}

	testCases := []struct {
		name     string
		tenantID string
		appID    string
		userID   string
		wantErr  error
	}{
		{"missing tenant", "", "app-1", "user-1", ErrMissingTenantID},
		{"missing app", "tenant-1", "", "user-1", ErrMissingAppID},
		{"missing user", "tenant-1", "app-1", "", ErrMissingUserID},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := pipeline.Read(ctx, tc.tenantID, tc.appID, tc.userID, input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected error %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestReadPipeline_MissingQuery(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{}
	provider := &mockEmbeddingProvider{}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "",
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if !errors.Is(err, ErrMissingQuery) {
		t.Fatalf("expected ErrMissingQuery, got %v", err)
	}
}

func TestReadPipeline_WhitespaceQuery(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{}
	provider := &mockEmbeddingProvider{}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "   \t\n  ",
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("expected ErrEmptyQuery, got %v", err)
	}
}

func TestReadPipeline_QueryTooLong(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{}
	provider := &mockEmbeddingProvider{}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         100,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	longQuery := make([]byte, 101)
	for i := range longQuery {
		longQuery[i] = 'a'
	}

	input := ReadInput{
		Query: string(longQuery),
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if !errors.Is(err, ErrQueryTooLong) {
		t.Fatalf("expected ErrQueryTooLong, got %v", err)
	}
}

func TestReadPipeline_InvalidMemoryType(t *testing.T) {
	ctx := context.Background()

	repos := &Repos{}
	provider := &mockEmbeddingProvider{}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Types: []string{"invalid_type"},
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if !errors.Is(err, ErrInvalidMemoryType) {
		t.Fatalf("expected ErrInvalidMemoryType, got %v", err)
	}
}

func TestReadPipeline_ValidTypes(t *testing.T) {
	ctx := context.Background()

	var capturedTypes []string
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
			searchByEmbeddingFunc: func(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*Memory, error) {
				capturedTypes = types
				return []*Memory{}, nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Types: []string{"profile", "preference"},
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedTypes) != 2 {
		t.Fatalf("expected 2 types, got %d", len(capturedTypes))
	}
}

func TestReadPipeline_LimitClamp(t *testing.T) {
	ctx := context.Background()

	var capturedLimit int
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
			searchByEmbeddingFunc: func(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*Memory, error) {
				capturedLimit = limit
				return []*Memory{}, nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	testCases := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"zero limit uses max", 0, 50},
		{"negative limit uses max", -1, 50},
		{"exceeds max uses max", 100, 50},
		{"valid limit preserved", 10, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := ReadInput{
				Query: "test query",
				Limit: tc.inputLimit,
			}

			_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if capturedLimit != tc.expectedLimit {
				t.Fatalf("expected limit %d, got %d", tc.expectedLimit, capturedLimit)
			}
		})
	}
}

func TestReadPipeline_EmbeddingProviderError(t *testing.T) {
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
		Memories: &mockMemoryRepo{},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return nil, errors.New("provider unavailable")
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if !errors.Is(err, ErrEmbeddingGeneration) {
		t.Fatalf("expected ErrEmbeddingGeneration, got %v", err)
	}
}

func TestReadPipeline_DimensionMismatch(t *testing.T) {
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
		Memories: &mockMemoryRepo{},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 512), nil
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if !errors.Is(err, ErrEmbeddingDimension) {
		t.Fatalf("expected ErrEmbeddingDimension, got %v", err)
	}
}

func TestReadPipeline_RepoSearchError(t *testing.T) {
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
			searchByEmbeddingFunc: func(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*Memory, error) {
				return nil, errors.New("database error")
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Limit: 10,
	}

	_, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if !errors.Is(err, ErrMemorySearch) {
		t.Fatalf("expected ErrMemorySearch, got %v", err)
	}
}

func TestReadPipeline_ImportanceThresholdFiltering(t *testing.T) {
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
			searchByEmbeddingFunc: func(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*Memory, error) {
				return []*Memory{
					{
						ID:              uuid.New(),
						ImportanceScore: 0.8,
						Content:         "High importance",
						MemoryType:      MemoryTypeProfile,
						MemoryStability: MemoryStabilityLongTerm,
						Metadata:        map[string]interface{}{},
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					},
					{
						ID:              uuid.New(),
						ImportanceScore: 0.2,
						Content:         "Low importance",
						MemoryType:      MemoryTypeProfile,
						MemoryStability: MemoryStabilityLongTerm,
						Metadata:        map[string]interface{}{},
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					},
					{
						ID:              uuid.New(),
						ImportanceScore: 0.5,
						Content:         "Medium importance",
						MemoryType:      MemoryTypeProfile,
						MemoryStability: MemoryStabilityLongTerm,
						Metadata:        map[string]interface{}{},
						CreatedAt:       time.Now(),
						UpdatedAt:       time.Now(),
					},
				}, nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.5,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Limit: 10,
	}

	result, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Memories) != 2 {
		t.Fatalf("expected 2 memories after filtering, got %d", len(result.Memories))
	}

	for _, mem := range result.Memories {
		if mem.ImportanceScore < 0.5 {
			t.Fatalf("memory with importance %f should have been filtered", mem.ImportanceScore)
		}
	}
}

func TestReadPipeline_EmptyResult(t *testing.T) {
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
			searchByEmbeddingFunc: func(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*Memory, error) {
				return []*Memory{}, nil
			},
		},
	}

	provider := &mockEmbeddingProvider{
		embedFunc: func(ctx context.Context, text string) ([]float32, error) {
			return make([]float32, 1536), nil
		},
	}

	pipeline := NewReadPipeline(repos, provider, ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	input := ReadInput{
		Query: "test query",
		Limit: 10,
	}

	result, err := pipeline.Read(ctx, "tenant-1", "app-1", "user-1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Memories == nil {
		t.Fatalf("expected non-nil memories slice")
	}

	if len(result.Memories) != 0 {
		t.Fatalf("expected 0 memories, got %d", len(result.Memories))
	}
}
