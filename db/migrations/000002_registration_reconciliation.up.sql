BEGIN;

ALTER TABLE registration_rules
    ADD COLUMN parent_spiffe_id text,
    ADD COLUMN x509_svid_ttl_seconds integer NOT NULL DEFAULT 600,
    ADD COLUMN spire_entry_id text,
    ADD COLUMN reconcile_status text NOT NULL DEFAULT 'pending',
    ADD COLUMN last_reconciled_at timestamptz,
    ADD COLUMN last_error_code text;

ALTER TABLE registration_rules
    ADD CONSTRAINT registration_rules_x509_ttl_range
        CHECK (x509_svid_ttl_seconds BETWEEN 60 AND 86400),
    ADD CONSTRAINT registration_rules_reconcile_status
        CHECK (reconcile_status IN ('pending', 'converged', 'error'));

CREATE UNIQUE INDEX registration_rules_spire_entry_unique
    ON registration_rules (spire_entry_id)
    WHERE spire_entry_id IS NOT NULL;

COMMIT;
