package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/database"
)

func TestPostgresRepositoryAppend(t *testing.T) {
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

	slug := fmt.Sprintf("audit-test-%d", time.Now().UTC().UnixNano())
	var organizationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, display_name)
		VALUES ($1, 'Audit Integration Test')
		RETURNING id::text
	`, slug).Scan(&organizationID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	repo := NewPostgresRepository(tx)
	created, err := repo.Append(ctx, Event{
		OrganizationID: &organizationID,
		ActorType:      "system",
		ActorID:        "phase2-integration-test",
		Action:         "workload.observed",
		TargetType:     "workload",
		TargetID:       "integration-test-workload",
		CorrelationID:  "phase2-audit-test",
		Metadata:       json.RawMessage(`{"source":"integration-test"}`),
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}
	if created.ID == "" || created.CreatedAt.IsZero() {
		t.Fatalf("Append() did not return persistence metadata: %#v", created)
	}
}
