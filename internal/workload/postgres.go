package workload

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type PostgresRepository struct {
	db queryer
}

func NewPostgresRepository(db queryer) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListByOrganization(ctx context.Context, organizationID string) ([]Workload, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			id::text,
			organization_id::text,
			trust_domain_id::text,
			name,
			environment,
			spiffe_id,
			status,
			last_seen_at,
			created_at,
			updated_at
		FROM workloads
		WHERE organization_id = $1::uuid
		ORDER BY environment ASC, name ASC, id ASC
	`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("query workloads: %w", err)
	}
	defer rows.Close()

	workloads := make([]Workload, 0)
	for rows.Next() {
		var item Workload
		var lastSeen sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.OrganizationID,
			&item.TrustDomainID,
			&item.Name,
			&item.Environment,
			&item.SPIFFEID,
			&item.Status,
			&lastSeen,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan workload: %w", err)
		}
		if lastSeen.Valid {
			seenAt := lastSeen.Time
			item.LastSeenAt = &seenAt
		}
		workloads = append(workloads, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workloads: %w", err)
	}
	return workloads, nil
}
