-- Adds more meta information to secure envelopes for tracking
BEGIN;

ALTER TABLE secure_envelopes ADD COLUMN remote TEXT DEFAULT NULL;
ALTER TABLE secure_envelopes ADD COLUMN reply_to TEXT REFERENCES secure_envelopes (id) DEFAULT NULL;
ALTER TABLE secure_envelopes ADD COLUMN transfer_state INTEGER DEFAULT 0;

COMMIT;