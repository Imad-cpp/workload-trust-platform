package registration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresRepository struct {
	db queryer
}

func NewPostgresRepository(db queryer) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) List(ctx context.Context) ([]Rule, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			rr.id::text,
			rr.organization_id::text,
			rr.workload_id::text,
			w.spiffe_id,
			rr.selectors,
			rr.desired_state,
			rr.revision,
			rr.parent_spiffe_id,
			rr.x509_svid_ttl_seconds,
			rr.spire_entry_id,
			rr.reconcile_status,
			rr.last_reconciled_at,
			rr.last_error_code
		FROM registration_rules rr
		JOIN workloads w ON w.id = rr.workload_id
		ORDER BY rr.organization_id ASC, rr.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query registration rules: %w", err)
	}
	defer rows.Close()

	items := make([]Rule, 0)
	for rows.Next() {
		var item Rule
		var selectorsRaw []byte
		var parent sql.NullString
		var entryID sql.NullString
		var reconciledAt sql.NullTime
		var lastError sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.OrganizationID,
			&item.WorkloadID,
			&item.WorkloadSPIFFEID,
			&selectorsRaw,
			&item.DesiredState,
			&item.Revision,
			&parent,
			&item.X509SVIDTTLSeconds,
			&entryID,
			&item.ReconcileStatus,
			&reconciledAt,
			&lastError,
		); err != nil {
			return nil, fmt.Errorf("scan registration rule: %w", err)
		}
		if err := json.Unmarshal(selectorsRaw, &item.Selectors); err != nil {
			return nil, fmt.Errorf("decode registration selectors: %w", err)
		}
		if parent.Valid {
			value := parent.String
			item.ParentSPIFFEID = &value
		}
		if entryID.Valid {
			value := entryID.String
			item.SPIREEntryID = &value
		}
		if reconciledAt.Valid {
			value := reconciledAt.Time
			item.LastReconciledAt = &value
		}
		if lastError.Valid {
			value := lastError.String
			item.LastErrorCode = &value
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate registration rules: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) MarkConverged(ctx context.Context, ruleID string, spireEntryID *string) error {
	var entry any
	if spireEntryID != nil {
		entry = *spireEntryID
	}
	var updatedID string
	if err := r.db.QueryRow(ctx, `
		UPDATE registration_rules
		SET
			spire_entry_id = $2,
			reconcile_status = 'converged',
			last_reconciled_at = now(),
			last_error_code = NULL,
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text
	`, ruleID, entry).Scan(&updatedID); err != nil {
		return fmt.Errorf("mark registration rule converged: %w", err)
	}
	return nil
}

func (r *PostgresRepository) MarkError(ctx context.Context, ruleID, errorCode string) error {
	var updatedID string
	if err := r.db.QueryRow(ctx, `
		UPDATE registration_rules
		SET
			reconcile_status = 'error',
			last_error_code = $2,
			updated_at = now()
		WHERE id = $1::uuid
		RETURNING id::text
	`, ruleID, errorCode).Scan(&updatedID); err != nil {
		return fmt.Errorf("mark registration rule error: %w", err)
	}
	return nil
}
