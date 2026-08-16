package policy

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

func TestPostgresManagerVersionsAndActivatesPolicy(t *testing.T) {
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

	outer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin outer test transaction: %v", err)
	}
	defer func() { _ = outer.Rollback(context.Background()) }()

	organizationID := seedPolicyOrganization(t, ctx, outer, "lifecycle")
	manager := NewPostgresManager(outer)
	actor := MutationActor{Type: "operator", ID: "operator-ci", Role: "operator"}

	created, err := manager.Create(ctx, actor, "policy-create-correlation", CreateInput{
		OrganizationID: organizationID,
		Name:           "frontend to orders",
		InitialVersion: RuleInput{
			SourceSPIFFEID:      "spiffe://policy-test.example/prod/frontend",
			DestinationSPIFFEID: "spiffe://policy-test.example/prod/orders-api",
			Action:              " CONNECT ",
			Effect:              " ALLOW ",
			ChangeReason:        " initial reviewed route ",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Policy.Status != "draft" || created.Policy.Revision != 1 || created.Policy.ActiveVersionID != nil {
		t.Fatalf("unexpected created policy: %#v", created.Policy)
	}
	if created.Version.VersionNumber != 1 || created.Version.PolicyRevision != 1 || created.Version.Effect != EffectAllow {
		t.Fatalf("unexpected initial version: %#v", created.Version)
	}
	assertPolicyAudit(t, ctx, outer, created.Policy.ID, "access_policy.create_with_version", 1)

	appended, err := manager.AppendVersion(ctx, actor, "policy-append-correlation", created.Policy.ID, AppendVersionInput{
		ExpectedRevision: 1,
		Version: RuleInput{
			SourceSPIFFEID:      "spiffe://policy-test.example/prod/frontend",
			DestinationSPIFFEID: "spiffe://policy-test.example/prod/orders-api",
			Action:              ActionConnect,
			Effect:              EffectDeny,
			ChangeReason:        "temporarily deny during review",
		},
	})
	if err != nil {
		t.Fatalf("AppendVersion() error = %v", err)
	}
	if appended.VersionNumber != 2 || appended.PolicyRevision != 2 || appended.Effect != EffectDeny {
		t.Fatalf("unexpected appended version: %#v", appended)
	}
	assertPolicyAudit(t, ctx, outer, created.Policy.ID, "access_policy.append_version", 1)

	_, err = manager.AppendVersion(ctx, actor, "policy-stale-correlation", created.Policy.ID, AppendVersionInput{
		ExpectedRevision: 1,
		Version: RuleInput{
			SourceSPIFFEID:      "spiffe://policy-test.example/prod/frontend",
			DestinationSPIFFEID: "spiffe://policy-test.example/prod/orders-api",
			Action:              ActionConnect,
			Effect:              EffectAllow,
			ChangeReason:        "stale write must fail",
		},
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("stale AppendVersion() error = %v, want ErrConflict", err)
	}

	activated, err := manager.Activate(ctx, actor, "policy-activate-correlation", created.Policy.ID, ActivateInput{
		ExpectedRevision: appended.PolicyRevision,
		VersionID:        appended.ID,
	})
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}
	if activated.Status != "active" || activated.Revision != 3 || activated.ActiveVersionID == nil || *activated.ActiveVersionID != appended.ID {
		t.Fatalf("unexpected activated policy: %#v", activated)
	}
	assertPolicyAudit(t, ctx, outer, created.Policy.ID, "access_policy.activate_version", 1)

	var (
		versionCount int
		storedAction string
		storedEffect string
	)
	if err := outer.QueryRow(ctx, `
		SELECT count(*), min(action), min(effect)
		FROM access_policy_versions
		WHERE policy_id = $1::uuid
	`, created.Policy.ID).Scan(&versionCount, &storedAction, &storedEffect); err != nil {
		t.Fatalf("inspect policy versions: %v", err)
	}
	if versionCount != 2 || storedAction != ActionConnect {
		t.Fatalf("unexpected version storage: count=%d action=%q", versionCount, storedAction)
	}
	if storedEffect != EffectAllow && storedEffect != EffectDeny {
		t.Fatalf("unexpected stored effect: %q", storedEffect)
	}
}

func TestPostgresManagerRejectsForeignAndMalformedActivation(t *testing.T) {
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
	outer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin outer test transaction: %v", err)
	}
	defer func() { _ = outer.Rollback(context.Background()) }()

	organizationID := seedPolicyOrganization(t, ctx, outer, "negative")
	manager := NewPostgresManager(outer)
	actor := MutationActor{Type: "operator", ID: "operator-ci", Role: "operator"}
	create := func(name string) CreateResult {
		t.Helper()
		result, err := manager.Create(ctx, actor, "create-"+name, CreateInput{
			OrganizationID: organizationID,
			Name:           name,
			InitialVersion: RuleInput{
				SourceSPIFFEID:      "spiffe://policy-test.example/prod/frontend",
				DestinationSPIFFEID: "spiffe://policy-test.example/prod/orders-api",
				Action:              ActionConnect,
				Effect:              EffectAllow,
				ChangeReason:        "seed policy for negative activation",
			},
		})
		if err != nil {
			t.Fatalf("Create(%q) error = %v", name, err)
		}
		return result
	}

	first := create("negative first")
	second := create("negative second")
	_, err = manager.Activate(ctx, actor, "foreign-version", first.Policy.ID, ActivateInput{
		ExpectedRevision: 1,
		VersionID:        second.Version.ID,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign activation error = %v, want ErrNotFound", err)
	}

	var malformedVersionID string
	if err := outer.QueryRow(ctx, `
		INSERT INTO access_policy_versions (
			policy_id, version, source_spiffe_id, destination_spiffe_id,
			action, effect, created_by, change_reason
		) VALUES ($1::uuid, 2, $2, $3, 'invoke', 'allow', 'fixture', 'malformed direct fixture')
		RETURNING id::text
	`, first.Policy.ID,
		"spiffe://policy-test.example/prod/frontend",
		"spiffe://policy-test.example/prod/orders-api",
	).Scan(&malformedVersionID); err != nil {
		t.Fatalf("insert malformed fixture: %v", err)
	}

	_, err = manager.Activate(ctx, actor, "malformed-version", first.Policy.ID, ActivateInput{
		ExpectedRevision: 1,
		VersionID:        malformedVersionID,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("malformed activation error = %v, want ErrInvalidInput", err)
	}

	var activeVersionID *string
	var revision int64
	if err := outer.QueryRow(ctx, `
		SELECT active_version_id::text, revision
		FROM access_policies
		WHERE id = $1::uuid
	`, first.Policy.ID).Scan(&activeVersionID, &revision); err != nil {
		t.Fatalf("inspect unchanged policy: %v", err)
	}
	if activeVersionID != nil || revision != 1 {
		t.Fatalf("policy changed after rejected activation: active=%v revision=%d", activeVersionID, revision)
	}
}

func TestPostgresManagerRollsBackWhenAuditFails(t *testing.T) {
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
	outer, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin outer test transaction: %v", err)
	}
	defer func() { _ = outer.Rollback(context.Background()) }()

	organizationID := seedPolicyOrganization(t, ctx, outer, "rollback")
	manager := NewPostgresManager(outer)
	actor := MutationActor{Type: "operator", ID: "operator-ci", Role: "operator"}
	created, err := manager.Create(ctx, actor, "rollback-seed", CreateInput{
		OrganizationID: organizationID,
		Name:           "rollback policy",
		InitialVersion: RuleInput{
			SourceSPIFFEID:      "spiffe://policy-test.example/prod/frontend",
			DestinationSPIFFEID: "spiffe://policy-test.example/prod/orders-api",
			Action:              ActionConnect,
			Effect:              EffectAllow,
			ChangeReason:        "seed rollback policy",
		},
	})
	if err != nil {
		t.Fatalf("seed Create() error = %v", err)
	}

	if _, err := outer.Exec(ctx, `
		CREATE FUNCTION reject_policy_audit_for_test() RETURNS trigger
		LANGUAGE plpgsql AS $$
		BEGIN
			IF NEW.action IN ('access_policy.append_version', 'access_policy.activate_version') THEN
				RAISE EXCEPTION 'intentional policy audit failure';
			END IF;
			RETURN NEW;
		END;
		$$;
		CREATE TRIGGER reject_policy_audit_for_test
		BEFORE INSERT ON audit_events
		FOR EACH ROW EXECUTE FUNCTION reject_policy_audit_for_test();
	`); err != nil {
		t.Fatalf("install policy audit failure trigger: %v", err)
	}

	_, err = manager.AppendVersion(ctx, actor, "append-audit-failure", created.Policy.ID, AppendVersionInput{
		ExpectedRevision: 1,
		Version: RuleInput{
			SourceSPIFFEID:      "spiffe://policy-test.example/prod/frontend",
			DestinationSPIFFEID: "spiffe://policy-test.example/prod/orders-api",
			Action:              ActionConnect,
			Effect:              EffectDeny,
			ChangeReason:        "this write must roll back",
		},
	})
	if err == nil {
		t.Fatal("expected append audit failure")
	}
	var versionCount int
	var revision int64
	if err := outer.QueryRow(ctx, `SELECT count(*) FROM access_policy_versions WHERE policy_id = $1::uuid`, created.Policy.ID).Scan(&versionCount); err != nil {
		t.Fatalf("count versions after rollback: %v", err)
	}
	if err := outer.QueryRow(ctx, `SELECT revision FROM access_policies WHERE id = $1::uuid`, created.Policy.ID).Scan(&revision); err != nil {
		t.Fatalf("read revision after rollback: %v", err)
	}
	if versionCount != 1 || revision != 1 {
		t.Fatalf("append audit rollback failed: versions=%d revision=%d", versionCount, revision)
	}

	_, err = manager.Activate(ctx, actor, "activate-audit-failure", created.Policy.ID, ActivateInput{
		ExpectedRevision: 1,
		VersionID:        created.Version.ID,
	})
	if err == nil {
		t.Fatal("expected activation audit failure")
	}
	var status string
	var activeVersionID *string
	if err := outer.QueryRow(ctx, `
		SELECT status, active_version_id::text, revision
		FROM access_policies
		WHERE id = $1::uuid
	`, created.Policy.ID).Scan(&status, &activeVersionID, &revision); err != nil {
		t.Fatalf("inspect activation rollback: %v", err)
	}
	if status != "draft" || activeVersionID != nil || revision != 1 {
		t.Fatalf("activation audit rollback failed: status=%q active=%v revision=%d", status, activeVersionID, revision)
	}
}

func seedPolicyOrganization(t *testing.T, ctx context.Context, tx pgx.Tx, suffix string) string {
	t.Helper()
	unique := time.Now().UTC().UnixNano()
	var organizationID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (slug, display_name)
		VALUES ($1, 'Policy Test')
		RETURNING id::text
	`, fmt.Sprintf("policy-%s-%d", suffix, unique)).Scan(&organizationID); err != nil {
		t.Fatalf("insert policy organization: %v", err)
	}
	return organizationID
}

func assertPolicyAudit(t *testing.T, ctx context.Context, tx pgx.Tx, targetID, action string, want int) {
	t.Helper()
	var count int
	if err := tx.QueryRow(ctx, `
		SELECT count(*)
		FROM audit_events
		WHERE target_id = $1 AND action = $2 AND actor_id = 'operator-ci'
	`, targetID, action).Scan(&count); err != nil {
		t.Fatalf("query policy audit: %v", err)
	}
	if count != want {
		t.Fatalf("audit count for %s = %d, want %d", action, count, want)
	}
}
