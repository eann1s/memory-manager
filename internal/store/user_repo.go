package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) core.UserRepo {
	return &userRepo{pool: pool}
}

func (r *userRepo) GetOrCreate(ctx context.Context, tenantID int64, appID int64, externalID string) (*core.User, error) {
	if externalID == "" {
		return nil, fmt.Errorf("external_id cannot be empty")
	}

	query := `
		INSERT INTO users (tenant_id, app_id, external_id, metadata)
		VALUES ($1, $2, $3, '{}'::jsonb)
		ON CONFLICT (tenant_id, app_id, external_id) DO UPDATE SET external_id = EXCLUDED.external_id
		RETURNING id, tenant_id, app_id, external_id, metadata, created_at
	`

	var user core.User
	var metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, tenantID, appID, externalID).Scan(
		&user.ID,
		&user.TenantID,
		&user.AppID,
		&user.ExternalID,
		&metadataJSON,
		&user.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get or create user: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &user.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &user, nil
}
