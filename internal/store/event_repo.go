package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/eann1s/codex-memory-manager/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventRepo struct {
	pool *pgxpool.Pool
}

func NewEventRepo(pool *pgxpool.Pool) core.EventRepo {
	return &eventRepo{pool: pool}
}

func (r *eventRepo) Insert(ctx context.Context, event *core.Event) error {
	if event.Content == "" {
		return fmt.Errorf("content cannot be empty")
	}
	if event.ConversationID == "" {
		return fmt.Errorf("conversation_id cannot be empty")
	}

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO events (
			tenant_id, app_id, user_id, conversation_id,
			role, content, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, timestamp
	`

	err = r.pool.QueryRow(
		ctx,
		query,
		event.TenantID,
		event.AppID,
		event.UserID,
		event.ConversationID,
		event.Role,
		event.Content,
		metadataJSON,
	).Scan(&event.ID, &event.Timestamp)

	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	return nil
}

func (r *eventRepo) ListByConversation(ctx context.Context, tenantID int64, appID int64, userID int64, conversationID string, limit int) ([]*core.Event, error) {
	query := `
		SELECT
			id, tenant_id, app_id, user_id, conversation_id,
			role, content, metadata, timestamp
		FROM events
		WHERE tenant_id = $1 AND app_id = $2 AND user_id = $3 AND conversation_id = $4
		ORDER BY timestamp DESC
		LIMIT $5
	`

	rows, err := r.pool.Query(ctx, query, tenantID, appID, userID, conversationID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*core.Event
	for rows.Next() {
		var event core.Event
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.TenantID,
			&event.AppID,
			&event.UserID,
			&event.ConversationID,
			&event.Role,
			&event.Content,
			&metadataJSON,
			&event.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}
