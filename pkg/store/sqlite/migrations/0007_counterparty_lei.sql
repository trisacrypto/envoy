-- Adds fields to ensure a transaction can be archived.
BEGIN;

ALTER TABLE counterparties ADD COLUMN lei string DEFAULT NULL;

-- LEI can be either NULL or a unique non-empty string.
CREATE UNIQUE INDEX counterparties_lei_unique ON counterparties(lei);

COMMIT;