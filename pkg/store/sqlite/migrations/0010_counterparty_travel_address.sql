-- Adds a column for the travel address of a counterparty.
BEGIN;

ALTER TABLE counterparties ADD COLUMN travel_address TEXT DEFAULT NULL;

COMMIT;