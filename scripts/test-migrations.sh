#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-}"

[[ -n "${DATABASE_URL}" ]] || {
  echo "DATABASE_URL is required" >&2
  exit 1
}

export DATABASE_URL

"${ROOT}/scripts/db-migrate.sh" up

psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO organizations (slug, display_name)
VALUES ('phase2-test', 'Phase 2 Test')
RETURNING id;

DO $$
DECLARE
    org_id uuid;
    domain_id uuid;
    workload_id uuid;
    policy_id uuid;
    policy_version_id uuid;
    audit_id uuid;
    policy_revision bigint;
BEGIN
    SELECT id INTO org_id FROM organizations WHERE slug = 'phase2-test';

    INSERT INTO trust_domains (organization_id, name)
    VALUES (org_id, 'workload-trust.test')
    RETURNING id INTO domain_id;

    INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id, status)
    VALUES (org_id, domain_id, 'frontend', 'lab', 'spiffe://workload-trust.test/lab/frontend', 'healthy')
    RETURNING id INTO workload_id;

    INSERT INTO registration_rules (organization_id, workload_id, selectors)
    VALUES (
        org_id,
        workload_id,
        '["docker:label:com.workload-trust.name:frontend", "docker:label:com.workload-trust.environment:lab"]'::jsonb
    );

    INSERT INTO access_policies (organization_id, name)
    VALUES (org_id, 'frontend-to-orders')
    RETURNING id INTO policy_id;

    SELECT revision INTO policy_revision FROM access_policies WHERE id = policy_id;
    IF policy_revision <> 1 THEN
        RAISE EXCEPTION 'access policy revision default is %, expected 1', policy_revision;
    END IF;

    INSERT INTO access_policy_versions (
        policy_id, version, source_spiffe_id, destination_spiffe_id,
        action, effect, created_by, change_reason
    ) VALUES (
        policy_id, 1,
        'spiffe://workload-trust.test/lab/frontend',
        'spiffe://workload-trust.test/lab/orders-api',
        'connect', 'allow', 'migration-test', 'verify immutable policy history'
    ) RETURNING id INTO policy_version_id;

    BEGIN
        UPDATE access_policy_versions SET effect = 'deny' WHERE id = policy_version_id;
        RAISE EXCEPTION 'append-only policy version update unexpectedly succeeded';
    EXCEPTION
        WHEN raise_exception THEN
            IF SQLERRM <> 'access_policy_versions are append-only' THEN
                RAISE;
            END IF;
    END;

    BEGIN
        DELETE FROM access_policy_versions WHERE id = policy_version_id;
        RAISE EXCEPTION 'append-only policy version delete unexpectedly succeeded';
    EXCEPTION
        WHEN raise_exception THEN
            IF SQLERRM <> 'access_policy_versions are append-only' THEN
                RAISE;
            END IF;
    END;

    INSERT INTO audit_events (
        organization_id, actor_type, actor_id, action, target_type, target_id, correlation_id, metadata
    ) VALUES (
        org_id, 'system', 'migration-test', 'workload.seeded', 'workload', workload_id::text,
        'migration-test-1', '{"source":"ci"}'::jsonb
    ) RETURNING id INTO audit_id;

    BEGIN
        UPDATE audit_events SET action = 'tampered' WHERE id = audit_id;
        RAISE EXCEPTION 'append-only audit update unexpectedly succeeded';
    EXCEPTION
        WHEN raise_exception THEN
            IF SQLERRM <> 'audit_events are append-only' THEN
                RAISE;
            END IF;
    END;

    BEGIN
        DELETE FROM audit_events WHERE id = audit_id;
        RAISE EXCEPTION 'append-only audit delete unexpectedly succeeded';
    EXCEPTION
        WHEN raise_exception THEN
            IF SQLERRM <> 'audit_events are append-only' THEN
                RAISE;
            END IF;
    END;
END
$$;
SQL

"${ROOT}/scripts/db-migrate.sh" down

if psql "${DATABASE_URL}" -Atqc "SELECT to_regclass('public.workloads')" | grep -q workloads; then
  echo "workloads table still exists after rollback" >&2
  exit 1
fi

"${ROOT}/scripts/db-migrate.sh" up

if ! psql "${DATABASE_URL}" -Atqc "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='access_policies' AND column_name='revision')" | grep -qx t; then
  echo "access_policies.revision is missing after re-apply" >&2
  exit 1
fi

if [[ "$(psql "${DATABASE_URL}" -Atqc "INSERT INTO organizations (slug, display_name) VALUES ('phase2-reapply', 'Phase 2 Reapply') RETURNING id" | head -n1)" == "" ]]; then
  echo "organization insert failed after re-apply" >&2
  exit 1
fi

if [[ "$(psql "${DATABASE_URL}" -Atqc "SELECT column_default FROM information_schema.columns WHERE table_schema='public' AND table_name='access_policies' AND column_name='revision')" != "1" ]]; then
  echo "access_policies.revision default is not 1 after re-apply" >&2
  exit 1
fi

echo "PostgreSQL migration apply/rollback/apply, policy-revision, policy-history, and audit append-only tests passed."
