-- Adds a table for tracking sunrise tokens sent out to counterparties who are not on
-- either the TRISA or TRP networks. These tokens are used for verification and
-- linkage to a specific transaction or envelope set.
BEGIN;

-- The contacts table stores email address information for compliance officers and
-- other travel rule contacts that the Envoy system may want to email either for
-- troubleshooting a compliance transfer or for sunrise purposes.
-- NOTE: a contact is uniquely associated with a Counterparty.
-- NOTE: ensure email address is not case-sensitive when stored.
CREATE TABLE IF NOT EXISTS contacts (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL DEFAULT '',
    email           TEXT NOT NULL UNIQUE,
    role            TEXT NOT NULL DEFAULT '',
    counterparty_id TEXT NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL,
    FOREIGN KEY (counterparty_id) REFERENCES counterparties(id) ON DELETE CASCADE
);

-- The sunrise table stores verification tokens and enough information to manage
-- sunrise accounts and resend tokens without loading the transaction from the db.
-- NOTE: the verification tokens are only associated with the contacts via email (not ID).
-- NOTE: ensure email address is not case-sensitive when stored.
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
    FOREIGN KEY (email) REFERENCES contacts(email) ON DELETE CASCADE
);

COMMIT;