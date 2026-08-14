package registration

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Imad-cpp/workload-trust-platform/internal/database"
	"github.com/jackc/pgx/v5"
)

func TestPostgresMutationServiceIsAuditedAndRevisionSafe(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()

	outer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin outer test transaction: %v", err)
	}
	defer func() { _ = outer.Rollback(context.Background()) }()

	organizationID, _, workloadID := seedMutationWorkload(t, ctx, outer, "audited")
	service := NewPostgresMutationService(outer)
	actor := MutationActor{Type: "operator", ID: "operator-ci", Role: "operator"}
	parentID := "spiffe://mutation-test.example/spire/agent/test"

	created, err := service.CreateDesired(ctx, actor, "create-correlation", CreateDesiredInput{
		WorkloadID: workloadID,
		Selectors: []string{
			"docker:label:com.workload-trust.name:api",
			"docker:label:com.workload-trust.environment:test",
		},
		ParentSPIFFEID:     parentID,
		X509SVIDTTLSeconds: 300,
	})
	if err != nil {
		t.Fatalf("CreateDesired() error = %v", err)
	}
	if created.OrganizationID != organizationID || created.Revision != 1 || created.ReconcileStatus != "pending" {
		t.Fatalf("unexpected create result: %#v", created)
	}
	assertMutationAudit(t, ctx, outer, created.ID, "registration_rule.create_desired_state", "operator-ci", 1)

	replaced, err := service.ReplaceDesired(ctx, actor, "replace-correlation", created.ID, ReplaceDesiredInput{
		ExpectedRevision:   1,
		DesiredState:       "absent",
		Selectors:          []string{"docker:label:com.workload-trust.name:api", "docker:label:com.workload-trust.environment:test"},
		ParentSPIFFEID:     parentID,
		X509SVIDTTLSeconds: 600,
	})
	if err != nil {
		t.Fatalf("ReplaceDesired() error = %v", err)
	}
	if replaced.Revision != 2 || replaced.DesiredState != "absent" || replaced.ReconcileStatus != "pending" {
		t.Fatalf("unexpected replace result: %#v", replaced)
	}
	assertMutationAudit(t, ctx, outer, created.ID, "registration_rule.replace_desired_state", "operator-ci", 1)

	_, err = service.ReplaceDesired(ctx, actor, "stale-correlation", created.ID, ReplaceDesiredInput{
		ExpectedRevision:   1,
		DesiredState:       "present",
		Selectors:          []string{"docker:label:com.workload-trust.name:api", "docker:label:com.workload-trust.environment:test"},
		ParentSPIFFEID:     parentID,
		X509SVIDTTLSeconds: 300,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale ReplaceDesired() error = %v, want ErrConflict", err)
	}
	var revision int64
	if err := outer.QueryRow(ctx, `SELECT revision FROM registration_rules WHERE id = $1::uuid`, created.ID).Scan(&revision); err != nil {
		t.Fatalf("read revision after conflict: %v", err)
	}
	if revision != 2 {
		t.Fatalf("revision = %d after stale write, want 2", revision)
	}
}

func TestPostgresMutationRollsBackWhenAuditFails(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer pool.Close()

	outer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin outer test transaction: %v", err)
	}
	defer func() { _ = outer.Rollback(context.Background()) }()

	_, _, workloadID := seedMutationWorkload(t, ctx, outer, "rollback")
	if _, err := outer.Exec(ctx, `
		CREATE FUNCTION reject_mutation_audit_for_test() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.action = 'registration_rule.create_desired_state' THEN
				RAISE EXCEPTION 'intentional audit failure';
			END IF;
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER reject_mutation_audit_for_test
		BEFORE INSERT ON audit_events
		FOR EACH ROW EXECUTE FUNCTION reject_mutation_audit_for_test();
	`); err != nil {
		t.Fatalf("install test audit failure trigger: %v", err)
	}

	service := NewPostgresMutationService(outer)
	_, err = service.CreateDesired(ctx, MutationActor{Type: "operator", ID: "operator-ci", Role: "operator"}, "rollback-correlation", CreateDesiredInput{
		WorkloadID:         workloadID,
		Selectors:          []string{"docker:label:com.workload-trust.name:rollback"},
		ParentSPIFFEID:     "spiffe://mutation-test.example/spire/agent/test",
		X509SVIDTTLSeconds: 300,
	})
	if err == nil {
		t.Fatal("expected audit failure to abort mutation")
	}

	var count int
	if err := outer.QueryRow(ctx, `SELECT count(*) FROM registration_rules WHERE workload_id = $1::uuid`, workloadID).Scan(&count); err != nil {
		t.Fatalf("count registration rules after rollback: %v", err)
	}
	if count != 0 {
		t.Fatalf("registration rule count = %d after audit failure, want 0", count)
	}
}

func seedMutationWorkload(t *testing.T, ctx context.Context, tx pgx.Tx, suffix string) (string, string, string) {
	t.Helper()
	unique := time.Now().UTC().UnixNano()
	var organizationID, trustDomainID, workloadID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, display_name)
		VALUES ($1, 'Mutation Test')
		RETURNING id::text
	`, fmt.Sprintf("mutation-%s-%d", suffix, unique)).Scan(&organizationID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO trust_domains (organization_id, name)
		VALUES ($1::uuid, $2)
		RETURNING id::text
	`, organizationID, fmt.Sprintf("mutation-%s-%d.example", suffix, unique)).Scan(&trustDomainID); err != nil {
		t.Fatalf("insert trust domain: %v", err)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id)
		VALUES ($1::uuid, $2::uuid, 'api', 'test', $3)
		RETURNING id::text
	`, organizationID, trustDomainID, fmt.Sprintf("spiffe://mutation-%s-%d.example/test/api", suffix, unique)).Scan(&workloadID); err != nil {
		t.Fatalf("insert workload: %v", err)
	}
	return organizationID, trustDomainID, workloadID
}

func assertMutationAudit(t *testing.T, ctx context.Context, tx pgx.Tx, targetID, action, actorID string, want int) {
	t.Helper()
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM audit_events
		WHERE target_id = $1 AND action = $2 AND actor_id = $3
	`, targetID, action, actorID).Scan(&count); err != nil {
		t.Fatalf("query mutation audit: %v", err)
	}
	if count != want {
		t.Fatalf("audit count for %s = %d, want %d", action, count, want)
	}
}
