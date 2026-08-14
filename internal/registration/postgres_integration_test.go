package registration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/database"
)

func TestPostgresRepositoryReconciliationState(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	suffix := time.Now().UTC().UnixNano()
	var organizationID, trustDomainID, workloadID, ruleID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, display_name)
		VALUES ($1, 'Registration Repository Test')
		RETURNING id::text
	`, fmt.Sprintf("registration-test-%d", suffix)).Scan(&organizationID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO trust_domains (organization_id, name)
		VALUES ($1::uuid, 'registration-test.example')
		RETURNING id::text
	`, organizationID).Scan(&trustDomainID); err != nil {
		t.Fatalf("insert trust domain: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id)
		VALUES ($1::uuid, $2::uuid, 'api', 'test', 'spiffe://registration-test.example/test/api')
		RETURNING id::text
	`, organizationID, trustDomainID).Scan(&workloadID); err != nil {
		t.Fatalf("insert workload: %v", err)
	}
	parentID := "spiffe://registration-test.example/spire/agent/test"
	if err := tx.QueryRow(ctx, `
		INSERT INTO registration_rules (
			organization_id, workload_id, selectors, parent_spiffe_id, x509_svid_ttl_seconds
		) VALUES ($1::uuid, $2::uuid, $3::jsonb, $4, 300)
		RETURNING id::text
	`, organizationID, workloadID, `["docker:label:service:api","docker:label:environment:test"]`, parentID).Scan(&ruleID); err != nil {
		t.Fatalf("insert registration rule: %v", err)
	}

	repo := NewPostgresRepository(tx)
	items, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	var rule *Rule
	for i := range items {
		if items[i].ID == ruleID {
			rule = &items[i]
			break
		}
	}
	if rule == nil {
		t.Fatal("inserted registration rule was not listed")
	}
	if rule.ParentSPIFFEID == nil || *rule.ParentSPIFFEID != parentID || rule.X509SVIDTTLSeconds != 300 {
		t.Fatalf("unexpected desired state: %#v", rule)
	}

	entryID := "spire-entry-test"
	if err := repo.MarkConverged(ctx, ruleID, &entryID); err != nil {
		t.Fatalf("MarkConverged() error = %v", err)
	}
	if err := repo.MarkError(ctx, ruleID, "ownership_mismatch"); err != nil {
		t.Fatalf("MarkError() error = %v", err)
	}

	var storedEntryID, status, errorCode string
	if err := tx.QueryRow(ctx, `
		SELECT spire_entry_id, reconcile_status, last_error_code
		FROM registration_rules
		WHERE id = $1::uuid
	`, ruleID).Scan(&storedEntryID, &status, &errorCode); err != nil {
		t.Fatalf("read reconciliation state: %v", err)
	}
	if storedEntryID != entryID || status != "error" || errorCode != "ownership_mismatch" {
		t.Fatalf("unexpected stored state: entry=%q status=%q error=%q", storedEntryID, status, errorCode)
	}
}
