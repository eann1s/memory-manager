package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type tenantRepo struct {
	pool *pgxpool.Pool
}

func NewTenantRepo(pool *pgxpool.Pool) core.TenantRepo {
	return &tenantRepo{pool: pool}
}

func (r *tenantRepo) GetOrCreate(ctx context.Context, externalID string) (*core.Tenant, error) {
	if externalID == "" {
		return nil, fmt.Errorf("external_id cannot be empty")
	}

	query := `
		INSERT INTO tenants (external_id, metadata)
		VALUES ($1, '{}'::jsonb)
		ON CONFLICT (external_id) DO UPDATE SET external_id = EXCLUDED.external_id
		RETURNING id, external_id, metadata, created_at
	`

	var tenant core.Tenant
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, externalID).Scan(
		&tenant.ID,
		&tenant.ExternalID,
		&metadataJSON,
		&tenant.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create tenant: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &tenant.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &tenant, nil
}
