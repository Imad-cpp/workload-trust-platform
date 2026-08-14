package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type rowQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresRepository struct {
	db rowQueryer
}

func NewPostgresRepository(db rowQueryer) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Append(ctx context.Context, event Event) (Event, error) {
	if event.ActorType == "" || event.ActorID == "" || event.Action == "" || event.TargetType == "" || event.TargetID == "" || event.CorrelationID == "" {
		return Event{}, errors.New("audit event is missing required attribution fields")
	}
	if len(event.Metadata) == 0 {
		event.Metadata = json.RawMessage(`{}`)
	}
	if !json.Valid(event.Metadata) {
		return Event{}, errors.New("audit metadata must be valid JSON")
	}

	var organizationID any
	if event.OrganizationID != nil {
		organizationID = *event.OrganizationID
	}

	err := r.db.QueryRow(ctx, `
		INSERT INTO audit_events (
			organization_id,
			actor_type,
			actor_id,
			action,
			target_type,
			target_id,
			correlation_id,
			metadata
		) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8::jsonb)
		RETURNING id::text, created_at
	`, organizationID, event.ActorType, event.ActorID, event.Action, event.TargetType, event.TargetID, event.CorrelationID, string(event.Metadata)).Scan(&event.ID, &event.CreatedAt)
	if err != nil {
		return Event{}, fmt.Errorf("append audit event: %w", err)
	}
	return event, nil
}
