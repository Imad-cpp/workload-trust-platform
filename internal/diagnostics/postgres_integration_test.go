package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/database"
)

func TestPostgresReaderSummarizesOrganization(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin test transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	unique := time.Now().UTC().UnixNano()
	var organizationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, display_name)
		VALUES ($1, 'Diagnostics Test')
		RETURNING id::text
	`, fmt.Sprintf("diagnostics-%d", unique)).Scan(&organizationID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	var trustDomainID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO trust_domains (organization_id, name)
		VALUES ($1::uuid, 'diagnostics.test')
		RETURNING id::text
	`, organizationID).Scan(&trustDomainID); err != nil {
		t.Fatalf("insert trust domain: %v", err)
	}

	var workloadID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id, status)
		VALUES ($1::uuid, $2::uuid, 'frontend', 'test', 'spiffe://diagnostics.test/test/frontend', 'healthy')
		RETURNING id::text
	`, organizationID, trustDomainID).Scan(&workloadID); err != nil {
		t.Fatalf("insert workload: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO registration_rules (
			organization_id, workload_id, selectors, parent_spiffe_id,
			x509_svid_ttl_seconds, reconcile_status
		) VALUES (
			$1::uuid, $2::uuid, '["docker:label:service:frontend"]'::jsonb,
			'spiffe://diagnostics.test/spire/agent/test', 300, 'pending'
		)
	`, organizationID, workloadID); err != nil {
		t.Fatalf("insert registration rule: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO access_policies (organization_id, name, status)
		VALUES ($1::uuid, 'frontend-to-orders', 'active')
	`, organizationID); err != nil {
		t.Fatalf("insert access policy: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			organization_id, actor_type, actor_id, action, target_type, target_id, correlation_id, metadata
		) VALUES (
			$1::uuid, 'operator', 'diagnostics-test', 'diagnostics.seeded', 'organization', $1, 'diag-1', '{}'::jsonb
		)
	`, organizationID); err != nil {
		t.Fatalf("insert audit event: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO security_events (
			organization_id, event_type, severity, reason_code, correlation_id, metadata
		) VALUES ($1::uuid, 'diagnostics-test', 'info', 'seeded', 'diag-1', '{}'::jsonb)
	`, organizationID); err != nil {
		t.Fatalf("insert security event: %v", err)
	}

	summary, err := NewPostgresReader(tx).SummaryByOrganization(ctx, organizationID)
	if err != nil {
		t.Fatalf("SummaryByOrganization() error = %v", err)
	}
	if summary.OrganizationID != organizationID || summary.WorkloadsTotal != 1 || summary.WorkloadsHealthy != 1 {
		t.Fatalf("unexpected workload diagnostics: %#v", summary)
	}
	if summary.WorkloadsUnknown != 0 || summary.WorkloadsDegraded != 0 || summary.WorkloadsOffline != 0 || summary.WorkloadsDisabled != 0 {
		t.Fatalf("unexpected non-healthy workload counts: %#v", summary)
	}
	if summary.RegistrationsPending != 1 || summary.PoliciesActive != 1 || summary.AuditEvents != 1 || summary.SecurityEvents != 1 {
		t.Fatalf("unexpected state diagnostics: %#v", summary)
	}
	if summary.LatestAuditAt == nil {
		t.Fatal("expected latest audit timestamp")
	}

	_, err = NewPostgresReader(tx).SummaryByOrganization(ctx, "123e4567-e89b-12d3-a456-426614174000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing organization error = %v, want ErrNotFound", err)
	}
}
