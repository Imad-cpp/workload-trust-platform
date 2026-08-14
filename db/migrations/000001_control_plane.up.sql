BEGIN;

CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    display_name text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_slug_format CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$')
);

CREATE TABLE trust_domains (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT trust_domains_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT trust_domains_org_name_unique UNIQUE (organization_id, name)
);

CREATE TABLE workloads (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    trust_domain_id uuid NOT NULL REFERENCES trust_domains(id) ON DELETE RESTRICT,
    name text NOT NULL,
    environment text NOT NULL,
    spiffe_id text NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'unknown',
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT workloads_status CHECK (status IN ('unknown', 'healthy', 'degraded', 'offline', 'disabled')),
    CONSTRAINT workloads_identity_unique UNIQUE (organization_id, environment, name)
);

CREATE INDEX workloads_org_environment_idx ON workloads (organization_id, environment, name);
CREATE INDEX workloads_last_seen_idx ON workloads (organization_id, last_seen_at DESC NULLS LAST);

CREATE TABLE registration_rules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    workload_id uuid NOT NULL REFERENCES workloads(id) ON DELETE CASCADE,
    selectors jsonb NOT NULL,
    desired_state text NOT NULL DEFAULT 'present',
    revision bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT registration_rules_desired_state CHECK (desired_state IN ('present', 'absent')),
    CONSTRAINT registration_rules_selectors_array CHECK (jsonb_typeof(selectors) = 'array'),
    CONSTRAINT registration_rules_workload_unique UNIQUE (workload_id)
);

CREATE TABLE access_policies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'draft',
    active_version_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT access_policies_status CHECK (status IN ('draft', 'active', 'disabled')),
    CONSTRAINT access_policies_org_name_unique UNIQUE (organization_id, name)
);

CREATE TABLE access_policy_versions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id uuid NOT NULL REFERENCES access_policies(id) ON DELETE RESTRICT,
    version bigint NOT NULL,
    source_spiffe_id text NOT NULL,
    destination_spiffe_id text NOT NULL,
    action text NOT NULL,
    effect text NOT NULL,
    created_by text NOT NULL,
    change_reason text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT access_policy_versions_effect CHECK (effect IN ('allow', 'deny')),
    CONSTRAINT access_policy_versions_policy_version_unique UNIQUE (policy_id, version)
);

ALTER TABLE access_policies
    ADD CONSTRAINT access_policies_active_version_fk
    FOREIGN KEY (active_version_id) REFERENCES access_policy_versions(id) ON DELETE RESTRICT;

CREATE OR REPLACE FUNCTION reject_access_policy_version_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'access_policy_versions are append-only';
END;
$$;

CREATE TRIGGER access_policy_versions_no_update
BEFORE UPDATE ON access_policy_versions
FOR EACH ROW EXECUTE FUNCTION reject_access_policy_version_mutation();

CREATE TRIGGER access_policy_versions_no_delete
BEFORE DELETE ON access_policy_versions
FOR EACH ROW EXECUTE FUNCTION reject_access_policy_version_mutation();

CREATE TABLE audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid REFERENCES organizations(id) ON DELETE RESTRICT,
    actor_type text NOT NULL,
    actor_id text NOT NULL,
    action text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    correlation_id text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_events_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX audit_events_org_created_idx ON audit_events (organization_id, created_at DESC);
CREATE INDEX audit_events_correlation_idx ON audit_events (correlation_id);

CREATE OR REPLACE FUNCTION reject_audit_event_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'audit_events are append-only';
END;
$$;

CREATE TRIGGER audit_events_no_update
BEFORE UPDATE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();

CREATE TRIGGER audit_events_no_delete
BEFORE DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();

CREATE TABLE security_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid REFERENCES organizations(id) ON DELETE RESTRICT,
    event_type text NOT NULL,
    severity text NOT NULL,
    source_spiffe_id text,
    destination_spiffe_id text,
    reason_code text NOT NULL,
    correlation_id text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    observed_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT security_events_severity CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    CONSTRAINT security_events_metadata_object CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX security_events_org_observed_idx ON security_events (organization_id, observed_at DESC);

COMMIT;
