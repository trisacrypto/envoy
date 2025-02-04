-- Adds fields to ensure a transaction can be archived.
BEGIN;

ALTER TABLE transactions ADD COLUMN archived BOOLEAN DEFAULT 0;
ALTER TABLE transactions ADD COLUMN archived_on DATETIME DEFAULT NULL;

-- NOTE: cannot alter table to add a check constraint in sqlite
-- In the v1.0.0 release when we collapse the migrations into a single file, this
-- constraint check should be added to ensure boolean data integrity.
-- ALTER TABLE transactions ADD CONSTRAINT CHECK (archived IN (0,1));

COMMIT;