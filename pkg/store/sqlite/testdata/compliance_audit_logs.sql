-- Compliance Audit Logs
-- 1 row represents each enum column value (mixed)
-- 2 each for user, api key, and sunrise (IDs match the other test data)
--  * User name: 'Admin User'
--  * APIKey description: 'Full permissions keys'
--  * Sunrise txn ID: '01JXTGSFRC88HAY8V173976Z9D'
INSERT INTO compliance_audit_log (id, timestamp, actor_id, actor_type, resource_id, resource_type, action, resource_action_meta, signature) VALUES
    ('fa71445e-0781-481f-aaad-b7a3ab0d15df', '2024-01-01T12:00:05.120005-10:00', x'018f2ee1271c0e42d47ea5450a242834', 'user', '2c891c75-14fa-4c71-aa07-6405b98db7a3', 'transaction', 'create', 'test_user_create_transaction', x''),
    ('a3fdc41a-d4ff-44ab-bce7-7a27ce276d19', '2024-02-02T13:10:15.131015-10:00', x'0197750cbf0c4222af236138d2737d2d', 'api_key', x'018f2ee1d49935bf09d5913b8c13d51a', 'user', 'update', 'test_api_key_update_user', x''),
    ('92928476-caa8-4ca9-9878-1d3326714391', '2024-03-03T14:20:25.142025-10:00', x'018f2ee949b4c95d3aab11b49cda4544', 'sunrise', x'018f2eea7377bbde57a557d86d5597a0', 'api_key', 'delete', 'test_sunrise_delete_api_key', x''),
    ('c9f68e80-4eb1-4cd1-ac2a-69236cf71877', '2024-04-04T15:30:35.153035-10:00', x'018f2ee1271c0e42d47ea5450a242834', 'user', x'0197757635c6fbb25a5bb5e1e614df77', 'counterparty', 'create', 'test_user_create_counterparty', x''),
    ('e81789ca-880e-413a-ba26-f274c6d02b12', '2024-05-05T16:40:45.164045-10:00', x'018f2ee949b4c95d3aab11b49cda4544', 'api_key', x'018ECD7C995324EB921AE98B9676EAD8', 'account', 'update', 'test_api_key_update_account', x''),
    ('f55d0945-f61f-4718-a9d4-9de7005eef00', '2024-06-06T17:50:55.175055-10:00', x'0197750cbf0c4222af236138d2737d2d', 'sunrise', x'0197750cbf0c4222af236138d2737d2d', 'sunrise', 'delete', 'test_sunrise_delete_sunrise', x'')
;
