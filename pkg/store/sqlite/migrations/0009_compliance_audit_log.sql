-- Adds a table for compliance audit logging.
BEGIN;

-- The compliance_audit_log table stores signed (immutable) audit logs for any
-- changes to other database objects that are related to financial compliance.
CREATE TABLE IF NOT EXISTS compliance_audit_log (
    id BLOB PRIMARY KEY,
    actor_id BLOB NOT NULL,
    actor_type TEXT NOT NULL,
    resource_id BLOB NOT NULL,
    resource_type TEXT NOT NULL,
    resource_modified DATETIME NOT NULL,
    action TEXT NOT NULL,
    resource_action_meta TEXT DEFAULT NULL,
    signature BLOB NOT NULL,
    key_id TEXT NOT NULL
);

-- Timestamp indexes are often useful for sorting/filtering by resource_modified time.
CREATE INDEX IF NOT EXISTS idx_cal_resource_modified ON compliance_audit_log(resource_modified);

COMMIT;
