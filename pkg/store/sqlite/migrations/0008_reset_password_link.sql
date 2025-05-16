-- Adds a table for tracking verification tokens sent out to users who have
-- requested a reset password link.
BEGIN;

-- The reset_password_link table stores the details required to associate a
-- request from a user to reset their password to a link that they are emailed
-- to verify that they are who they say they are.
CREATE TABLE IF NOT EXISTS reset_password_link (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    expiration DATETIME NOT NULL,
    signature BLOB DEFAULT NULL,
    sent_on DATETIME DEFAULT NULL,
    created DATETIME DEFAULT NULL,
    modified DATETIME DEFAULT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

COMMIT;
