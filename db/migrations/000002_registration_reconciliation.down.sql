BEGIN;

DROP INDEX IF EXISTS registration_rules_spire_entry_unique;

ALTER TABLE registration_rules
    DROP CONSTRAINT IF EXISTS registration_rules_reconcile_status,
    DROP CONSTRAINT IF EXISTS registration_rules_x509_ttl_range,
    DROP COLUMN IF EXISTS last_error_code,
    DROP COLUMN IF EXISTS last_reconciled_at,
    DROP COLUMN IF EXISTS reconcile_status,
    DROP COLUMN IF EXISTS spire_entry_id,
    DROP COLUMN IF EXISTS x509_svid_ttl_seconds,
    DROP COLUMN IF EXISTS parent_spiffe_id;

COMMIT;
