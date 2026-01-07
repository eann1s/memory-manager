package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type appRepo struct {
	pool *pgxpool.Pool
}

func NewAppRepo(pool *pgxpool.Pool) core.AppRepo {
	return &appRepo{pool: pool}
}

func (r *appRepo) GetOrCreate(ctx context.Context, tenantID int64, externalID string) (*core.App, error) {
	if externalID == "" {
		return nil, fmt.Errorf("external_id cannot be empty")
	}

	query := `
		INSERT INTO apps (tenant_id, external_id, metadata)
		VALUES ($1, $2, '{}'::jsonb)
		ON CONFLICT (tenant_id, external_id) DO UPDATE SET external_id = EXCLUDED.external_id
		RETURNING id, tenant_id, external_id, metadata, created_at
	`

	var app core.App
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, tenantID, externalID).Scan(
		&app.ID,
		&app.TenantID,
		&app.ExternalID,
		&metadataJSON,
		&app.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create app: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &app.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &app, nil
}
