-- Adds a table for compliance audit logging.
BEGIN;

-- The compliance_audit_log table stores signed (immutable) audit logs for any
-- changes to other database objects that are related to financial compliance.
CREATE TABLE IF NOT EXISTS compliance_audit_log (
    id TEXT PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_action_meta TEXT DEFAULT NULL,
    signature BLOB NOT NULL
);

-- Timestamp indexes are often useful for sorting/filtering by timestamp.
CREATE INDEX IF NOT EXISTS idx_cal_timestamp ON compliance_audit_log(timestamp);

COMMIT;
