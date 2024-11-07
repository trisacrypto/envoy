-- Adds a table for tracking sunrise tokens sent out to counterparties who are not on
-- either the TRISA or TRP networks. These tokens are used for verification and
-- linkage to a specific transaction or envelope set.
BEGIN;

-- The sunrise table stores verification tokens and enough information to manage
-- sunrise accounts and resend tokens without loading the transaction from the db.
CREATE TABLE IF NOT EXISTS sunrise (
    id              TEXT PRIMARY KEY,
    envelope_id     TEXT NOT NULL,
    email           TEXT NOT NULL,
    expiration      DATETIME NOT NULL,
    signature       BLOB NOT NULL,
    status          TEXT NOT NULL,
    sent_on         DATETIME DEFAULT NULL,
    verified_on     DATETIME DEFAULT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL,
    UNIQUE(envelope_id, email),
    FOREIGN KEY (envelope_id) REFERENCES transactions(id) ON DELETE CASCADE
);

COMMIT;