package store

import (
	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRepos(pool *pgxpool.Pool) *core.Repos {
	return &core.Repos{
		Tenants:  NewTenantRepo(pool),
		Apps:     NewAppRepo(pool),
		Users:    NewUserRepo(pool),
		Memories: NewMemoryRepo(pool),
		Events:   NewEventRepo(pool),
	}
}
