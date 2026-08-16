BEGIN;

ALTER TABLE access_policies
    DROP CONSTRAINT IF EXISTS access_policies_revision_positive;

ALTER TABLE access_policies
    DROP COLUMN IF EXISTS revision;

COMMIT;
