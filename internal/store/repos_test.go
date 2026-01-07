package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/eann1s/codex-memory-manager/internal/config"
	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/pgvector/pgvector-go"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()

	cfg := config.Load()
	ctx := context.Background()

	db, err := NewDB(ctx, cfg.DBURL)
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	t.Cleanup(func() {
		ctx := context.Background()
		db.Pool.Exec(ctx, "TRUNCATE tenants, apps, users, memories, events CASCADE")
		db.Pool.Close()
	})

	return db
}

func TestTenantGetOrCreateIdempotency(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant1, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	tenant2, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to get tenant: %v", err)
	}

	if tenant1.ID != tenant2.ID {
		t.Errorf("expected same tenant ID, got %d and %d", tenant1.ID, tenant2.ID)
	}

	if tenant1.ExternalID != "t1" {
		t.Errorf("expected external_id 't1', got '%s'", tenant1.ExternalID)
	}
}

func TestAppGetOrCreateIdempotency(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app1, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	app2, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to get app: %v", err)
	}

	if app1.ID != app2.ID {
		t.Errorf("expected same app ID, got %d and %d", app1.ID, app2.ID)
	}

	if app1.ExternalID != "app1" {
		t.Errorf("expected external_id 'app1', got '%s'", app1.ExternalID)
	}
}

func TestUserGetOrCreateIdempotency(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	user1, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	user2, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}

	if user1.ID != user2.ID {
		t.Errorf("expected same user ID, got %d and %d", user1.ID, user2.ID)
	}

	if user1.ExternalID != "user1" {
		t.Errorf("expected external_id 'user1', got '%s'", user1.ExternalID)
	}
}

func TestUserScopingCorrectness(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant1, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant t1: %v", err)
	}

	tenant2, err := repos.Tenants.GetOrCreate(ctx, "t2")
	if err != nil {
		t.Fatalf("failed to create tenant t2: %v", err)
	}

	app1, err := repos.Apps.GetOrCreate(ctx, tenant1.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app for t1: %v", err)
	}

	app2, err := repos.Apps.GetOrCreate(ctx, tenant2.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app for t2: %v", err)
	}

	userA, err := repos.Users.GetOrCreate(ctx, tenant1.ID, app1.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user for t1/app1: %v", err)
	}

	userB, err := repos.Users.GetOrCreate(ctx, tenant2.ID, app2.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user for t2/app1: %v", err)
	}

	if userA.ID == userB.ID {
		t.Errorf("expected different user IDs for different tenants, got same ID %d", userA.ID)
	}

	if userA.TenantID == userB.TenantID {
		t.Errorf("expected different tenant IDs, got same ID %d", userA.TenantID)
	}
}

func TestUserScopingWithinSameTenant(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app1, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app1: %v", err)
	}

	app2, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app2")
	if err != nil {
		t.Fatalf("failed to create app2: %v", err)
	}

	userInApp1, err := repos.Users.GetOrCreate(ctx, tenant.ID, app1.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user in app1: %v", err)
	}

	userInApp2, err := repos.Users.GetOrCreate(ctx, tenant.ID, app2.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user in app2: %v", err)
	}

	if userInApp1.ID == userInApp2.ID {
		t.Errorf("expected different user IDs for different apps in same tenant, got same ID %d", userInApp1.ID)
	}

	if userInApp1.AppID == userInApp2.AppID {
		t.Errorf("expected different app IDs, got same ID %d", userInApp1.AppID)
	}

	if userInApp1.TenantID != userInApp2.TenantID {
		t.Errorf("expected same tenant ID, got %d and %d", userInApp1.TenantID, userInApp2.TenantID)
	}

	if userInApp1.ExternalID != "user1" || userInApp2.ExternalID != "user1" {
		t.Errorf("expected both users to have external_id 'user1', got '%s' and '%s'", userInApp1.ExternalID, userInApp2.ExternalID)
	}
}

func TestMemoryInsertAndList(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	user, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	embedding := pgvector.NewVector(make([]float32, 1536))

	memory1 := &core.Memory{
		TenantID:        tenant.ID,
		AppID:           app.ID,
		UserID:          user.ID,
		MemoryType:      core.MemoryTypeProfile,
		Content:         "User prefers dark mode",
		ImportanceScore: 0.8,
		MemoryStability: core.MemoryStabilityLongTerm,
		Embedding:       embedding,
		Metadata:        map[string]interface{}{"source": "test"},
	}

	if err := repos.Memories.Insert(ctx, memory1); err != nil {
		t.Fatalf("failed to insert memory1: %v", err)
	}

	memory2 := &core.Memory{
		TenantID:        tenant.ID,
		AppID:           app.ID,
		UserID:          user.ID,
		MemoryType:      core.MemoryTypePreference,
		Content:         "User likes coffee",
		ImportanceScore: 0.6,
		MemoryStability: core.MemoryStabilityShortTerm,
		Embedding:       embedding,
		Metadata:        map[string]interface{}{"source": "test"},
	}

	if err := repos.Memories.Insert(ctx, memory2); err != nil {
		t.Fatalf("failed to insert memory2: %v", err)
	}

	memories, err := repos.Memories.ListByUser(ctx, tenant.ID, app.ID, user.ID, 10)
	if err != nil {
		t.Fatalf("failed to list memories: %v", err)
	}

	if len(memories) != 2 {
		t.Errorf("expected 2 memories, got %d", len(memories))
	}

	if memories[0].Content != "User likes coffee" {
		t.Errorf("expected newest memory first, got '%s'", memories[0].Content)
	}

	if memories[0].TenantID != tenant.ID {
		t.Errorf("expected tenant_id %d, got %d", tenant.ID, memories[0].TenantID)
	}

	if memories[0].AppID != app.ID {
		t.Errorf("expected app_id %d, got %d", app.ID, memories[0].AppID)
	}

	if memories[0].UserID != user.ID {
		t.Errorf("expected user_id %d, got %d", user.ID, memories[0].UserID)
	}
}

func TestEventInsertAndListByConversation(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	user, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	event1 := &core.Event{
		TenantID:       tenant.ID,
		AppID:          app.ID,
		UserID:         user.ID,
		ConversationID: "conv1",
		Role:           "user",
		Content:        "Hello",
		Metadata:       map[string]interface{}{"source": "test"},
	}

	if err := repos.Events.Insert(ctx, event1); err != nil {
		t.Fatalf("failed to insert event1: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	event2 := &core.Event{
		TenantID:       tenant.ID,
		AppID:          app.ID,
		UserID:         user.ID,
		ConversationID: "conv1",
		Role:           "assistant",
		Content:        "Hi there!",
		Metadata:       map[string]interface{}{"source": "test"},
	}

	if err := repos.Events.Insert(ctx, event2); err != nil {
		t.Fatalf("failed to insert event2: %v", err)
	}

	event3 := &core.Event{
		TenantID:       tenant.ID,
		AppID:          app.ID,
		UserID:         user.ID,
		ConversationID: "conv2",
		Role:           "user",
		Content:        "Different conversation",
		Metadata:       map[string]interface{}{"source": "test"},
	}

	if err := repos.Events.Insert(ctx, event3); err != nil {
		t.Fatalf("failed to insert event3: %v", err)
	}

	events, err := repos.Events.ListByConversation(ctx, tenant.ID, app.ID, user.ID, "conv1", 10)
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	if events[0].ConversationID != "conv1" {
		t.Errorf("expected conversation_id 'conv1', got '%s'", events[0].ConversationID)
	}

	if events[0].Content != "Hi there!" {
		t.Errorf("expected newest event first, got '%s'", events[0].Content)
	}
}

func TestMemoryCrossScopeIsolation(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	user1, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	user2, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user2")
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	embedding := pgvector.NewVector(make([]float32, 1536))

	memory := &core.Memory{
		TenantID:        tenant.ID,
		AppID:           app.ID,
		UserID:          user1.ID,
		MemoryType:      core.MemoryTypeProfile,
		Content:         "User1 memory",
		ImportanceScore: 0.8,
		MemoryStability: core.MemoryStabilityLongTerm,
		Embedding:       embedding,
		Metadata:        map[string]interface{}{"source": "test"},
	}

	if err := repos.Memories.Insert(ctx, memory); err != nil {
		t.Fatalf("failed to insert memory: %v", err)
	}

	user2Memories, err := repos.Memories.ListByUser(ctx, tenant.ID, app.ID, user2.ID, 10)
	if err != nil {
		t.Fatalf("failed to list user2 memories: %v", err)
	}

	if len(user2Memories) != 0 {
		t.Errorf("expected 0 memories for user2, got %d", len(user2Memories))
	}
}

func TestEventCrossScopeIsolation(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	user1, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user1: %v", err)
	}

	user2, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user2")
	if err != nil {
		t.Fatalf("failed to create user2: %v", err)
	}

	event := &core.Event{
		TenantID:       tenant.ID,
		AppID:          app.ID,
		UserID:         user1.ID,
		ConversationID: "conv1",
		Role:           "user",
		Content:        "User1 event",
		Metadata:       map[string]interface{}{"source": "test"},
	}

	if err := repos.Events.Insert(ctx, event); err != nil {
		t.Fatalf("failed to insert event: %v", err)
	}

	user2Events, err := repos.Events.ListByConversation(ctx, tenant.ID, app.ID, user2.ID, "conv1", 10)
	if err != nil {
		t.Fatalf("failed to list user2 events: %v", err)
	}

	if len(user2Events) != 0 {
		t.Errorf("expected 0 events for user2, got %d", len(user2Events))
	}
}

func TestConcurrencyGetOrCreateUser(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	const numGoroutines = 10
	var wg sync.WaitGroup
	userIDs := make([]int64, numGoroutines)
	errors := make([]error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			user, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user_concurrent")
			if err != nil {
				errors[idx] = err
				return
			}
			userIDs[idx] = user.ID
		}(i)
	}

	wg.Wait()

	for i, err := range errors {
		if err != nil {
			t.Errorf("goroutine %d failed: %v", i, err)
		}
	}

	firstID := userIDs[0]
	for i, id := range userIDs {
		if id != firstID {
			t.Errorf("goroutine %d got different ID %d, expected %d", i, id, firstID)
		}
	}

	var count int
	err = db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE tenant_id = $1 AND app_id = $2 AND external_id = $3", tenant.ID, app.ID, "user_concurrent").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count users: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 user row, got %d", count)
	}
}

func TestEmptyExternalIDValidation(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	_, err := repos.Tenants.GetOrCreate(ctx, "")
	if err == nil {
		t.Error("expected error for empty tenant external_id, got nil")
	}

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	_, err = repos.Apps.GetOrCreate(ctx, tenant.ID, "")
	if err == nil {
		t.Error("expected error for empty app external_id, got nil")
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	_, err = repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "")
	if err == nil {
		t.Error("expected error for empty user external_id, got nil")
	}
}

func TestMemoryContentValidation(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	user, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	embedding := pgvector.NewVector(make([]float32, 1536))

	memory := &core.Memory{
		TenantID:        tenant.ID,
		AppID:           app.ID,
		UserID:          user.ID,
		MemoryType:      core.MemoryTypeProfile,
		Content:         "",
		ImportanceScore: 0.8,
		MemoryStability: core.MemoryStabilityLongTerm,
		Embedding:       embedding,
		Metadata:        map[string]interface{}{},
	}

	err = repos.Memories.Insert(ctx, memory)
	if err == nil {
		t.Error("expected error for empty memory content, got nil")
	}
}

func TestEventContentValidation(t *testing.T) {
	db := setupTestDB(t)
	repos := NewRepos(db.Pool)
	ctx := context.Background()

	tenant, err := repos.Tenants.GetOrCreate(ctx, "t1")
	if err != nil {
		t.Fatalf("failed to create tenant: %v", err)
	}

	app, err := repos.Apps.GetOrCreate(ctx, tenant.ID, "app1")
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	user, err := repos.Users.GetOrCreate(ctx, tenant.ID, app.ID, "user1")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	event := &core.Event{
		TenantID:       tenant.ID,
		AppID:          app.ID,
		UserID:         user.ID,
		ConversationID: "conv1",
		Role:           "user",
		Content:        "",
		Metadata:       map[string]interface{}{},
	}

	err = repos.Events.Insert(ctx, event)
	if err == nil {
		t.Error("expected error for empty event content, got nil")
	}

	event.Content = "Hello"
	event.ConversationID = ""

	err = repos.Events.Insert(ctx, event)
	if err == nil {
		t.Error("expected error for empty conversation_id, got nil")
	}
}
