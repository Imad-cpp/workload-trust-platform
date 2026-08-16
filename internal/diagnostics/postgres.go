package diagnostics

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type rowQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresReader struct {
	db rowQueryer
}

func NewPostgresReader(db rowQueryer) *PostgresReader {
	return &PostgresReader{db: db}
}

func (r *PostgresReader) SummaryByOrganization(ctx context.Context, organizationID string) (Summary, error) {
	var summary Summary
	err := r.db.QueryRow(ctx, `
		SELECT
			o.id::text,
			(SELECT count(*) FROM workloads w WHERE w.organization_id = o.id),
			(SELECT count(*) FROM workloads w WHERE w.organization_id = o.id AND w.status = 'healthy'),
			(SELECT count(*) FROM workloads w WHERE w.organization_id = o.id AND w.status = 'degraded'),
			(SELECT count(*) FROM workloads w WHERE w.organization_id = o.id AND w.status = 'offline'),
			(SELECT count(*) FROM registration_rules rr WHERE rr.organization_id = o.id AND rr.reconcile_status = 'pending'),
			(SELECT count(*) FROM registration_rules rr WHERE rr.organization_id = o.id AND rr.reconcile_status = 'converged'),
			(SELECT count(*) FROM registration_rules rr WHERE rr.organization_id = o.id AND rr.reconcile_status = 'error'),
			(SELECT count(*) FROM access_policies p WHERE p.organization_id = o.id AND p.status = 'draft'),
			(SELECT count(*) FROM access_policies p WHERE p.organization_id = o.id AND p.status = 'active'),
			(SELECT count(*) FROM access_policies p WHERE p.organization_id = o.id AND p.status = 'disabled'),
			(SELECT count(*) FROM audit_events a WHERE a.organization_id = o.id),
			(SELECT count(*) FROM security_events s WHERE s.organization_id = o.id),
			(SELECT max(a.created_at) FROM audit_events a WHERE a.organization_id = o.id)
		FROM organizations o
		WHERE o.id = $1::uuid
	`, organizationID).Scan(
		&summary.OrganizationID,
		&summary.WorkloadsTotal,
		&summary.WorkloadsHealthy,
		&summary.WorkloadsDegraded,
		&summary.WorkloadsOffline,
		&summary.RegistrationsPending,
		&summary.RegistrationsConverged,
		&summary.RegistrationsError,
		&summary.PoliciesDraft,
		&summary.PoliciesActive,
		&summary.PoliciesDisabled,
		&summary.AuditEvents,
		&summary.SecurityEvents,
		&summary.LatestAuditAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Summary{}, ErrNotFound
	}
	if err != nil {
		return Summary{}, fmt.Errorf("read organization diagnostics: %w", err)
	}
	return summary, nil
}
