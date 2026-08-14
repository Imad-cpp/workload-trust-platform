ALTER TABLE access_policies
    ADD COLUMN IF NOT EXISTS revision BIGINT NOT NULL DEFAULT 1;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'access_policies_revision_positive'
          AND conrelid = 'access_policies'::regclass
    ) THEN
        ALTER TABLE access_policies
            ADD CONSTRAINT access_policies_revision_positive CHECK (revision > 0);
    END IF;
END;
$$;
