package core

import (
	"context"
)

type TenantRepo interface {
	GetOrCreate(ctx context.Context, externalID string) (*Tenant, error)
}

type AppRepo interface {
	GetOrCreate(ctx context.Context, tenantID int64, externalID string) (*App, error)
}

type UserRepo interface {
	GetOrCreate(ctx context.Context, tenantID int64, appID int64, externalID string) (*User, error)
}

type MemoryRepo interface {
	Insert(ctx context.Context, memory *Memory) error
	ListByUser(ctx context.Context, tenantID int64, appID int64, userID int64, limit int) ([]*Memory, error)
	SearchByEmbedding(ctx context.Context, tenantID int64, appID int64, userID int64, embedding []float32, limit int, types []string) ([]*Memory, error)
}

type EventRepo interface {
	Insert(ctx context.Context, event *Event) error
	ListByConversation(ctx context.Context, tenantID int64, appID int64, userID int64, conversationID string, limit int) ([]*Event, error)
}

type Repos struct {
	Tenants  TenantRepo
	Apps     AppRepo
	Users    UserRepo
	Memories MemoryRepo
	Events   EventRepo
}
