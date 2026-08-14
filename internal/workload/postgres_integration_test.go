package workload

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/database"
)

func TestPostgresRepositoryListByOrganization(t *testing.T) {
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
	slug := fmt.Sprintf("repo-test-%d", suffix)
	var organizationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, display_name)
		VALUES ($1, 'Repository Integration Test')
		RETURNING id::text
	`, slug).Scan(&organizationID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	var trustDomainID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO trust_domains (organization_id, name)
		VALUES ($1::uuid, 'repo-test.example')
		RETURNING id::text
	`, organizationID).Scan(&trustDomainID); err != nil {
		t.Fatalf("insert trust domain: %v", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO workloads (
			organization_id, trust_domain_id, name, environment, spiffe_id, status
		) VALUES
			($1::uuid, $2::uuid, 'zeta', 'prod', 'spiffe://repo-test.example/prod/zeta', 'healthy'),
			($1::uuid, $2::uuid, 'alpha', 'prod', 'spiffe://repo-test.example/prod/alpha', 'offline')
	`, organizationID, trustDomainID)
	if err != nil {
		t.Fatalf("insert workloads: %v", err)
	}

	repo := NewPostgresRepository(tx)
	items, err := repo.ListByOrganization(ctx, organizationID)
	if err != nil {
		t.Fatalf("ListByOrganization() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].Name != "alpha" || items[1].Name != "zeta" {
		t.Fatalf("unexpected deterministic order: %q, %q", items[0].Name, items[1].Name)
	}
	if items[0].OrganizationID != organizationID || items[0].TrustDomainID != trustDomainID {
		t.Fatalf("unexpected ownership fields: %#v", items[0])
	}
}
