package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type memoryRepo struct {
	pool *pgxpool.Pool
}

func NewMemoryRepo(pool *pgxpool.Pool) core.MemoryRepo {
	return &memoryRepo{pool: pool}
}

func (r *memoryRepo) Insert(ctx context.Context, memory *core.Memory) error {
	if memory.Content == "" {
		return fmt.Errorf("content cannot be empty")
	}

	metadataJSON, err := json.Marshal(memory.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO memories (
			tenant_id, app_id, user_id, memory_type, content,
			importance_score, memory_stability, embedding, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	err = r.pool.QueryRow(
		ctx,
		query,
		memory.TenantID,
		memory.AppID,
		memory.UserID,
		memory.MemoryType,
		memory.Content,
		memory.ImportanceScore,
		memory.MemoryStability,
		memory.Embedding,
		metadataJSON,
	).Scan(&memory.ID, &memory.CreatedAt, &memory.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert memory: %w", err)
	}

	return nil
}

func (r *memoryRepo) ListByUser(ctx context.Context, tenantID int64, appID int64, userID int64, limit int) ([]*core.Memory, error) {
	query := `
		SELECT
			id, tenant_id, app_id, user_id, memory_type, content,
			importance_score, memory_stability, embedding, metadata, created_at, updated_at
		FROM memories
		WHERE tenant_id = $1 AND app_id = $2 AND user_id = $3
		ORDER BY created_at DESC
		LIMIT $4
	`

	rows, err := r.pool.Query(ctx, query, tenantID, appID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query memories: %w", err)
	}
	defer rows.Close()

	var memories []*core.Memory
	for rows.Next() {
		var memory core.Memory
		var metadataJSON []byte

		err := rows.Scan(
			&memory.ID,
			&memory.TenantID,
			&memory.AppID,
			&memory.UserID,
			&memory.MemoryType,
			&memory.Content,
			&memory.ImportanceScore,
			&memory.MemoryStability,
			&memory.Embedding,
			&metadataJSON,
			&memory.CreatedAt,
			&memory.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan memory: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &memory.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		memories = append(memories, &memory)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating memories: %w", err)
	}

	return memories, nil
}
