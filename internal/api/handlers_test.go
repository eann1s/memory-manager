package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/eann1s/codex-memory-manager/internal/config"
	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/eann1s/codex-memory-manager/internal/openai"
	"github.com/eann1s/codex-memory-manager/internal/store"
	"github.com/google/uuid"
)

type fakeEmbeddingProvider struct {
	dim int
}

func (f *fakeEmbeddingProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	embedding := make([]float32, f.dim)
	for i := range embedding {
		embedding[i] = 0.001
	}
	return embedding, nil
}

func skipIfNoDBIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("skipping DB integration test; set RUN_DB_INTEGRATION=1 to run")
	}
}

func setupValidationTestHandler(t *testing.T) *Handler {
	t.Helper()

	repos := &core.Repos{
		Tenants:  &panicTenantRepo{},
		Apps:     &panicAppRepo{},
		Users:    &panicUserRepo{},
		Memories: &panicMemoryRepo{},
		Events:   &panicEventRepo{},
	}
	provider := &fakeEmbeddingProvider{dim: 1536}

	writePipeline := core.NewWritePipeline(repos, provider, core.WritePipelineConfig{
		ImportanceThreshold: 0.3,
		MaxContentLen:       10000,
		ExpectedDim:         1536,
	})

	readPipeline := core.NewReadPipeline(repos, provider, core.ReadPipelineConfig{
		ImportanceThreshold: 0.3,
		MaxQueryLen:         10000,
		MaxItems:            50,
		ExpectedDim:         1536,
	})

	return NewHandler(writePipeline, readPipeline, 100)
}

type panicTenantRepo struct{}

func (p *panicTenantRepo) GetOrCreate(ctx context.Context, externalID string) (*core.Tenant, error) {
	panic("validation test should not reach tenant repo")
}

type panicAppRepo struct{}

func (p *panicAppRepo) GetOrCreate(ctx context.Context, tenantID int64, externalID string) (*core.App, error) {
	panic("validation test should not reach app repo")
}

type panicUserRepo struct{}

func (p *panicUserRepo) GetOrCreate(ctx context.Context, tenantID int64, appID int64, externalID string) (*core.User, error) {
	panic("validation test should not reach user repo")
}

type panicMemoryRepo struct{}

func (p *panicMemoryRepo) Insert(ctx context.Context, memory *core.Memory) error {
	panic("validation test should not reach memory repo")
}

func (p *panicMemoryRepo) ListByUser(ctx context.Context, tenantID int64, appID int64, userID int64, limit int) ([]*core.Memory, error) {
	panic("validation test should not reach memory repo")
}

func (p *panicMemoryRepo) SearchByEmbedding(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*core.Memory, error) {
	panic("validation test should not reach memory repo")
}

type panicEventRepo struct{}

func (p *panicEventRepo) Insert(ctx context.Context, event *core.Event) error {
	panic("validation test should not reach event repo")
}

func (p *panicEventRepo) ListByConversation(ctx context.Context, tenantID int64, appID int64, userID int64, conversationID string, limit int) ([]*core.Event, error) {
	panic("validation test should not reach event repo")
}

func setupTestHandler(t *testing.T) (*Handler, *store.DB) {
	t.Helper()
	skipIfNoDBIntegration(t)

	cfg := config.Load()
	ctx := context.Background()

	db, err := store.NewDB(ctx, cfg.DBURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		db.Pool.Exec(ctx, "TRUNCATE tenants, apps, users, memories, events CASCADE")
		db.Pool.Close()
	})

	repos := store.NewRepos(db.Pool)
	provider := &fakeEmbeddingProvider{dim: cfg.EmbeddingDim}

	writePipeline := core.NewWritePipeline(repos, provider, core.WritePipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxContentLen:       cfg.MemoryMaxContentLen,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	readPipeline := core.NewReadPipeline(repos, provider, core.ReadPipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxQueryLen:         cfg.ReadMaxQueryLen,
		MaxItems:            cfg.ReadMaxItems,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	handler := NewHandler(writePipeline, readPipeline, cfg.WriteMaxItems)
	return handler, db
}

func TestHandler_Write_EndToEnd(t *testing.T) {
	handler, db := setupTestHandler(t)
	cfg := config.Load()

	reqBody := WriteRequest{
		TenantID: "tenant-1",
		AppID:    "app-1",
		UserID:   "user-1",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "User prefers dark mode",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
				Metadata:        map[string]interface{}{"source": "settings"},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.Write(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WriteResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.WrittenIDs) != 1 {
		t.Fatalf("expected 1 written ID, got %d", len(resp.WrittenIDs))
	}

	for i, idStr := range resp.WrittenIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			t.Fatalf("written_ids[%d] is not a valid UUID: %v", i, err)
		}
		if id == uuid.Nil {
			t.Fatalf("written_ids[%d] is nil UUID", i)
		}
	}

	if len(resp.Skipped) != 0 {
		t.Fatalf("expected 0 skipped items, got %d", len(resp.Skipped))
	}

	ctx := context.Background()
	repos := store.NewRepos(db.Pool)

	tenant, err := repos.Tenants.GetOrCreate(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("failed to get tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app-1")
	if err != nil {
		t.Fatalf("failed to get app: %v", err)
	}

	user, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user-1")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	memories, err := repos.Memories.ListByUser(ctx, tenant.ID, app.ID, user.ID, 10)
	if err != nil {
		t.Fatalf("failed to list memories: %v", err)
	}

	if len(memories) != 1 {
		t.Fatalf("expected 1 memory in DB, got %d", len(memories))
	}

	if memories[0].Content != "User prefers dark mode" {
		t.Fatalf("unexpected content: %s", memories[0].Content)
	}

	if memories[0].MemoryType != core.MemoryTypeProfile {
		t.Fatalf("unexpected type: %s", memories[0].MemoryType)
	}

	if len(memories[0].Embedding.Slice()) != cfg.EmbeddingDim {
		t.Fatalf("expected embedding dimension %d, got %d", cfg.EmbeddingDim, len(memories[0].Embedding.Slice()))
	}
}

func TestHandler_Write_Isolation(t *testing.T) {
	handler, db := setupTestHandler(t)

	reqBody1 := WriteRequest{
		TenantID: "tenant-1",
		AppID:    "app-1",
		UserID:   "user-1",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "Memory for tenant-1/app-1/user-1",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
		},
	}

	body1, _ := json.Marshal(reqBody1)
	req1 := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body1))
	rec1 := httptest.NewRecorder()
	handler.Write(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected status 200 for request 1, got %d", rec1.Code)
	}

	reqBody2 := WriteRequest{
		TenantID: "tenant-1",
		AppID:    "app-2",
		UserID:   "user-1",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "Memory for tenant-1/app-2/user-1",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
		},
	}

	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	handler.Write(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected status 200 for request 2, got %d", rec2.Code)
	}

	ctx := context.Background()
	repos := store.NewRepos(db.Pool)

	tenant, _ := repos.Tenants.GetOrCreate(ctx, "tenant-1")
	app1, _ := repos.Apps.GetOrCreate(ctx, tenant.ID, "app-1")
	user1, _ := repos.Users.GetOrCreate(ctx, tenant.ID, app1.ID, "user-1")

	memories1, err := repos.Memories.ListByUser(ctx, tenant.ID, app1.ID, user1.ID, 10)
	if err != nil {
		t.Fatalf("failed to list memories for user 1: %v", err)
	}

	if len(memories1) != 1 {
		t.Fatalf("expected 1 memory for user 1, got %d", len(memories1))
	}

	if memories1[0].Content != "Memory for tenant-1/app-1/user-1" {
		t.Fatalf("unexpected content for user 1: %s", memories1[0].Content)
	}

	app2, _ := repos.Apps.GetOrCreate(ctx, tenant.ID, "app-2")
	user2, _ := repos.Users.GetOrCreate(ctx, tenant.ID, app2.ID, "user-1")

	memories2, err := repos.Memories.ListByUser(ctx, tenant.ID, app2.ID, user2.ID, 10)
	if err != nil {
		t.Fatalf("failed to list memories for user 2: %v", err)
	}

	if len(memories2) != 1 {
		t.Fatalf("expected 1 memory for user 2, got %d", len(memories2))
	}

	if memories2[0].Content != "Memory for tenant-1/app-2/user-1" {
		t.Fatalf("unexpected content for user 2: %s", memories2[0].Content)
	}

	if user1.ID == user2.ID {
		t.Fatalf("user IDs should be different across apps")
	}
}

func TestHandler_Write_PartialSkip(t *testing.T) {
	handler, _ := setupTestHandler(t)

	reqBody := WriteRequest{
		TenantID: "tenant-1",
		AppID:    "app-1",
		UserID:   "user-1",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "Valid memory",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
			{
				Type:            core.MemoryType("invalid"),
				Content:         "Invalid type",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
			{
				Type:            core.MemoryTypePreference,
				Content:         "Another valid memory",
				ImportanceScore: 0.7,
				Stability:       core.MemoryStabilityShortTerm,
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Write(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp WriteResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.WrittenIDs) != 2 {
		t.Fatalf("expected 2 written IDs, got %d", len(resp.WrittenIDs))
	}

	if len(resp.Skipped) != 1 {
		t.Fatalf("expected 1 skipped item, got %d", len(resp.Skipped))
	}

	if resp.Skipped[0].Index != 1 {
		t.Fatalf("expected skipped index 1, got %d", resp.Skipped[0].Index)
	}
}

func TestHandler_Write_InvalidRequest(t *testing.T) {
	handler := setupValidationTestHandler(t)

	testCases := []struct {
		name       string
		reqBody    interface{}
		expectCode int
	}{
		{
			name:       "invalid JSON",
			reqBody:    "not json",
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing tenant_id",
			reqBody: WriteRequest{
				AppID:  "app-1",
				UserID: "user-1",
				Memories: []MemoryInput{
					{
						Type:            core.MemoryTypeProfile,
						Content:         "Test",
						ImportanceScore: 0.8,
						Stability:       core.MemoryStabilityLongTerm,
					},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing app_id",
			reqBody: WriteRequest{
				TenantID: "tenant-1",
				UserID:   "user-1",
				Memories: []MemoryInput{
					{
						Type:            core.MemoryTypeProfile,
						Content:         "Test",
						ImportanceScore: 0.8,
						Stability:       core.MemoryStabilityLongTerm,
					},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing user_id",
			reqBody: WriteRequest{
				TenantID: "tenant-1",
				AppID:    "app-1",
				Memories: []MemoryInput{
					{
						Type:            core.MemoryTypeProfile,
						Content:         "Test",
						ImportanceScore: 0.8,
						Stability:       core.MemoryStabilityLongTerm,
					},
				},
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "empty memories array",
			reqBody: WriteRequest{
				TenantID: "tenant-1",
				AppID:    "app-1",
				UserID:   "user-1",
				Memories: []MemoryInput{},
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if str, ok := tc.reqBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Write(rec, req)

			if rec.Code != tc.expectCode {
				t.Fatalf("expected status %d, got %d: %s", tc.expectCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandler_Write_MaxItemsRejection(t *testing.T) {
	handler := setupValidationTestHandler(t)

	testCases := []struct {
		name        string
		memoriesLen int
	}{
		{
			name:        "exceeds max by one",
			memoriesLen: 101,
		},
		{
			name:        "exceeds max by many",
			memoriesLen: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memories := make([]MemoryInput, tc.memoriesLen)
			for i := range memories {
				memories[i] = MemoryInput{
					Type:            core.MemoryTypeProfile,
					Content:         "test content",
					ImportanceScore: 0.8,
					Stability:       core.MemoryStabilityLongTerm,
				}
			}

			reqBody := WriteRequest{
				TenantID: "test-tenant",
				AppID:    "test-app",
				UserID:   "test-user",
				Memories: memories,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Write(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
			}

			var errResp map[string]string
			json.NewDecoder(rec.Body).Decode(&errResp)
			if errResp["error"] == "" {
				t.Fatal("expected error message in response")
			}
		})
	}
}

func TestHandler_Write_MaxItemsAllowed(t *testing.T) {
	handler, _ := setupTestHandler(t)
	cfg := config.Load()

	memories := make([]MemoryInput, cfg.WriteMaxItems)
	for i := range memories {
		memories[i] = MemoryInput{
			Type:            core.MemoryTypeProfile,
			Content:         "test content",
			ImportanceScore: 0.8,
			Stability:       core.MemoryStabilityLongTerm,
		}
	}

	reqBody := WriteRequest{
		TenantID: "test-tenant",
		AppID:    "test-app",
		UserID:   "test-user",
		Memories: memories,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Write(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WriteResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.WrittenIDs) != cfg.WriteMaxItems {
		t.Fatalf("expected %d written IDs, got %d", cfg.WriteMaxItems, len(resp.WrittenIDs))
	}

	for i, idStr := range resp.WrittenIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			t.Fatalf("written_ids[%d] is not a valid UUID: %v", i, err)
		}
		if id == uuid.Nil {
			t.Fatalf("written_ids[%d] is nil UUID", i)
		}
	}
}

func TestHandler_Write_WithRealOpenAI(t *testing.T) {
	skipIfNoDBIntegration(t)

	if os.Getenv("RUN_OPENAI_INTEGRATION") != "1" {
		t.Skip("skipping OpenAI integration test; set RUN_OPENAI_INTEGRATION=1 to run")
	}

	cfg := config.Load()
	ctx := context.Background()

	db, err := store.NewDB(ctx, cfg.DBURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		db.Pool.Exec(ctx, "TRUNCATE tenants, apps, users, memories, events CASCADE")
		db.Pool.Close()
	})

	repos := store.NewRepos(db.Pool)

	provider, err := openai.NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create OpenAI provider: %v", err)
	}

	writePipeline := core.NewWritePipeline(repos, provider, core.WritePipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxContentLen:       cfg.MemoryMaxContentLen,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	readPipeline := core.NewReadPipeline(repos, provider, core.ReadPipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxQueryLen:         cfg.ReadMaxQueryLen,
		MaxItems:            cfg.ReadMaxItems,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	handler := NewHandler(writePipeline, readPipeline, cfg.WriteMaxItems)

	reqBody := WriteRequest{
		TenantID: "tenant-openai",
		AppID:    "app-openai",
		UserID:   "user-openai",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "User prefers Python for backend development",
				ImportanceScore: 0.9,
				Stability:       core.MemoryStabilityLongTerm,
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.Write(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WriteResponse
	json.NewDecoder(rec.Body).Decode(&resp)

	if len(resp.WrittenIDs) != 1 {
		t.Fatalf("expected 1 written ID, got %d", len(resp.WrittenIDs))
	}

	tenant, _ := repos.Tenants.GetOrCreate(ctx, "tenant-openai")
	app, _ := repos.Apps.GetOrCreate(ctx, tenant.ID, "app-openai")
	user, _ := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user-openai")

	memories, err := repos.Memories.ListByUser(ctx, tenant.ID, app.ID, user.ID, 10)
	if err != nil {
		t.Fatalf("failed to list memories: %v", err)
	}

	if len(memories) != 1 {
		t.Fatalf("expected 1 memory in DB, got %d", len(memories))
	}

	if len(memories[0].Embedding.Slice()) != cfg.EmbeddingDim {
		t.Fatalf("expected embedding dimension %d, got %d", cfg.EmbeddingDim, len(memories[0].Embedding.Slice()))
	}
}

func TestHandler_Read_EndToEnd(t *testing.T) {
	handler, _ := setupTestHandler(t)

	writeReqBody := WriteRequest{
		TenantID: "tenant-read-1",
		AppID:    "app-read-1",
		UserID:   "user-read-1",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "User prefers dark mode for all applications",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
			{
				Type:            core.MemoryTypePreference,
				Content:         "User likes coffee in the morning",
				ImportanceScore: 0.7,
				Stability:       core.MemoryStabilityShortTerm,
			},
		},
	}

	writeBody, _ := json.Marshal(writeReqBody)
	writeReq := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(writeBody))
	writeRec := httptest.NewRecorder()
	handler.Write(writeRec, writeReq)

	if writeRec.Code != http.StatusOK {
		t.Fatalf("write failed: %d: %s", writeRec.Code, writeRec.Body.String())
	}

	readReqBody := ReadRequest{
		TenantID: "tenant-read-1",
		AppID:    "app-read-1",
		UserID:   "user-read-1",
		Query:    "dark mode settings",
		Limit:    10,
	}

	readBody, _ := json.Marshal(readReqBody)
	readReq := httptest.NewRequest(http.MethodPost, "/v1/read", bytes.NewReader(readBody))
	readRec := httptest.NewRecorder()
	handler.Read(readRec, readReq)

	if readRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", readRec.Code, readRec.Body.String())
	}

	var readResp ReadResponse
	if err := json.NewDecoder(readRec.Body).Decode(&readResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(readResp.Memories) == 0 {
		t.Fatalf("expected at least 1 memory, got 0")
	}

	for _, mem := range readResp.Memories {
		if mem.ID == "" {
			t.Fatal("memory ID should not be empty")
		}
		if mem.Content == "" {
			t.Fatal("memory content should not be empty")
		}
	}
}

func TestHandler_Read_Isolation(t *testing.T) {
	handler, _ := setupTestHandler(t)

	writeReq1 := WriteRequest{
		TenantID: "tenant-iso-1",
		AppID:    "app-iso-1",
		UserID:   "user-iso-1",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "Memory for tenant-iso-1",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
		},
	}
	body1, _ := json.Marshal(writeReq1)
	req1 := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body1))
	rec1 := httptest.NewRecorder()
	handler.Write(rec1, req1)

	writeReq2 := WriteRequest{
		TenantID: "tenant-iso-2",
		AppID:    "app-iso-2",
		UserID:   "user-iso-2",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "Memory for tenant-iso-2",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
		},
	}
	body2, _ := json.Marshal(writeReq2)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(body2))
	rec2 := httptest.NewRecorder()
	handler.Write(rec2, req2)

	readReq := ReadRequest{
		TenantID: "tenant-iso-1",
		AppID:    "app-iso-1",
		UserID:   "user-iso-1",
		Query:    "Memory",
		Limit:    10,
	}
	readBody, _ := json.Marshal(readReq)
	readReqHTTP := httptest.NewRequest(http.MethodPost, "/v1/read", bytes.NewReader(readBody))
	readRec := httptest.NewRecorder()
	handler.Read(readRec, readReqHTTP)

	if readRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", readRec.Code)
	}

	var readResp ReadResponse
	json.NewDecoder(readRec.Body).Decode(&readResp)

	for _, mem := range readResp.Memories {
		if mem.Content == "Memory for tenant-iso-2" {
			t.Fatal("read returned memory from different tenant/app/user")
		}
	}
}

func TestHandler_Read_TypesFilter(t *testing.T) {
	handler, _ := setupTestHandler(t)

	writeReqBody := WriteRequest{
		TenantID: "tenant-types-1",
		AppID:    "app-types-1",
		UserID:   "user-types-1",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "Profile memory content",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
			{
				Type:            core.MemoryTypePreference,
				Content:         "Preference memory content",
				ImportanceScore: 0.8,
				Stability:       core.MemoryStabilityLongTerm,
			},
		},
	}
	writeBody, _ := json.Marshal(writeReqBody)
	writeReq := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(writeBody))
	writeRec := httptest.NewRecorder()
	handler.Write(writeRec, writeReq)

	readReqBody := ReadRequest{
		TenantID: "tenant-types-1",
		AppID:    "app-types-1",
		UserID:   "user-types-1",
		Query:    "memory content",
		Types:    []string{"profile"},
		Limit:    10,
	}
	readBody, _ := json.Marshal(readReqBody)
	readReq := httptest.NewRequest(http.MethodPost, "/v1/read", bytes.NewReader(readBody))
	readRec := httptest.NewRecorder()
	handler.Read(readRec, readReq)

	if readRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", readRec.Code, readRec.Body.String())
	}

	var readResp ReadResponse
	json.NewDecoder(readRec.Body).Decode(&readResp)

	for _, mem := range readResp.Memories {
		if mem.Type != core.MemoryTypeProfile {
			t.Fatalf("expected only profile memories, got %s", mem.Type)
		}
	}
}

func TestHandler_Read_InvalidRequest(t *testing.T) {
	handler := setupValidationTestHandler(t)

	testCases := []struct {
		name       string
		reqBody    interface{}
		expectCode int
	}{
		{
			name:       "invalid JSON",
			reqBody:    "not json",
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing tenant_id",
			reqBody: ReadRequest{
				AppID:  "app-1",
				UserID: "user-1",
				Query:  "test query",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing app_id",
			reqBody: ReadRequest{
				TenantID: "tenant-1",
				UserID:   "user-1",
				Query:    "test query",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing user_id",
			reqBody: ReadRequest{
				TenantID: "tenant-1",
				AppID:    "app-1",
				Query:    "test query",
			},
			expectCode: http.StatusBadRequest,
		},
		{
			name: "missing query",
			reqBody: ReadRequest{
				TenantID: "tenant-1",
				AppID:    "app-1",
				UserID:   "user-1",
			},
			expectCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var body []byte
			if str, ok := tc.reqBody.(string); ok {
				body = []byte(str)
			} else {
				body, _ = json.Marshal(tc.reqBody)
			}

			req := httptest.NewRequest(http.MethodPost, "/v1/read", bytes.NewReader(body))
			rec := httptest.NewRecorder()

			handler.Read(rec, req)

			if rec.Code != tc.expectCode {
				t.Fatalf("expected status %d, got %d: %s", tc.expectCode, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestHandler_Read_LimitRespected(t *testing.T) {
	handler, _ := setupTestHandler(t)

	memories := make([]MemoryInput, 10)
	for i := range memories {
		memories[i] = MemoryInput{
			Type:            core.MemoryTypeProfile,
			Content:         "Test memory content for limit test",
			ImportanceScore: 0.8,
			Stability:       core.MemoryStabilityLongTerm,
		}
	}

	writeReqBody := WriteRequest{
		TenantID: "tenant-limit-1",
		AppID:    "app-limit-1",
		UserID:   "user-limit-1",
		Memories: memories,
	}
	writeBody, _ := json.Marshal(writeReqBody)
	writeReq := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(writeBody))
	writeRec := httptest.NewRecorder()
	handler.Write(writeRec, writeReq)

	readReqBody := ReadRequest{
		TenantID: "tenant-limit-1",
		AppID:    "app-limit-1",
		UserID:   "user-limit-1",
		Query:    "memory content",
		Limit:    3,
	}
	readBody, _ := json.Marshal(readReqBody)
	readReq := httptest.NewRequest(http.MethodPost, "/v1/read", bytes.NewReader(readBody))
	readRec := httptest.NewRecorder()
	handler.Read(readRec, readReq)

	if readRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", readRec.Code)
	}

	var readResp ReadResponse
	json.NewDecoder(readRec.Body).Decode(&readResp)

	if len(readResp.Memories) > 3 {
		t.Fatalf("expected at most 3 memories, got %d", len(readResp.Memories))
	}
}

func TestHandler_Read_WithRealOpenAI(t *testing.T) {
	skipIfNoDBIntegration(t)

	if os.Getenv("RUN_OPENAI_INTEGRATION") != "1" {
		t.Skip("skipping OpenAI integration test; set RUN_OPENAI_INTEGRATION=1 to run")
	}

	cfg := config.Load()
	ctx := context.Background()

	db, err := store.NewDB(ctx, cfg.DBURL)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		db.Pool.Exec(ctx, "TRUNCATE tenants, apps, users, memories, events CASCADE")
		db.Pool.Close()
	})

	repos := store.NewRepos(db.Pool)

	provider, err := openai.NewEmbeddingProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create OpenAI provider: %v", err)
	}

	writePipeline := core.NewWritePipeline(repos, provider, core.WritePipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxContentLen:       cfg.MemoryMaxContentLen,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	readPipeline := core.NewReadPipeline(repos, provider, core.ReadPipelineConfig{
		ImportanceThreshold: cfg.MemoryImportanceThreshold,
		MaxQueryLen:         cfg.ReadMaxQueryLen,
		MaxItems:            cfg.ReadMaxItems,
		ExpectedDim:         cfg.EmbeddingDim,
	})

	handler := NewHandler(writePipeline, readPipeline, cfg.WriteMaxItems)

	writeReqBody := WriteRequest{
		TenantID: "tenant-openai-read",
		AppID:    "app-openai-read",
		UserID:   "user-openai-read",
		Memories: []MemoryInput{
			{
				Type:            core.MemoryTypeProfile,
				Content:         "User prefers TypeScript for frontend development",
				ImportanceScore: 0.9,
				Stability:       core.MemoryStabilityLongTerm,
			},
			{
				Type:            core.MemoryTypePreference,
				Content:         "User enjoys hiking on weekends",
				ImportanceScore: 0.7,
				Stability:       core.MemoryStabilityShortTerm,
			},
		},
	}

	writeBody, _ := json.Marshal(writeReqBody)
	writeReq := httptest.NewRequest(http.MethodPost, "/v1/write", bytes.NewReader(writeBody))
	writeRec := httptest.NewRecorder()
	handler.Write(writeRec, writeReq)

	if writeRec.Code != http.StatusOK {
		t.Fatalf("write failed: %d: %s", writeRec.Code, writeRec.Body.String())
	}

	readReqBody := ReadRequest{
		TenantID: "tenant-openai-read",
		AppID:    "app-openai-read",
		UserID:   "user-openai-read",
		Query:    "programming language preference",
		Limit:    10,
	}

	readBody, _ := json.Marshal(readReqBody)
	readReq := httptest.NewRequest(http.MethodPost, "/v1/read", bytes.NewReader(readBody))
	readRec := httptest.NewRecorder()
	handler.Read(readRec, readReq)

	if readRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", readRec.Code, readRec.Body.String())
	}

	var readResp ReadResponse
	json.NewDecoder(readRec.Body).Decode(&readResp)

	if len(readResp.Memories) == 0 {
		t.Fatal("expected at least 1 memory from OpenAI search")
	}
}
