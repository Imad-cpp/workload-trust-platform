BEGIN;

ALTER TABLE access_policies
    ADD COLUMN revision bigint NOT NULL DEFAULT 1;

ALTER TABLE access_policies
    ADD CONSTRAINT access_policies_revision_positive CHECK (revision > 0);

COMMIT;
