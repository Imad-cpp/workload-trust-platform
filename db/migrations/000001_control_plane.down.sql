BEGIN;

DROP TABLE IF EXISTS security_events;
DROP TRIGGER IF EXISTS audit_events_no_delete ON audit_events;
DROP TRIGGER IF EXISTS audit_events_no_update ON audit_events;
DROP FUNCTION IF EXISTS reject_audit_event_mutation();
DROP TABLE IF EXISTS audit_events;
ALTER TABLE IF EXISTS access_policies DROP CONSTRAINT IF EXISTS access_policies_active_version_fk;
DROP TABLE IF EXISTS access_policy_versions;
DROP TABLE IF EXISTS access_policies;
DROP TABLE IF EXISTS registration_rules;
DROP TABLE IF EXISTS workloads;
DROP TABLE IF EXISTS trust_domains;
DROP TABLE IF EXISTS organizations;

COMMIT;
