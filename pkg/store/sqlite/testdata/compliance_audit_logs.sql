-- Compliance Audit Logs:
--   * Each possible actor/action/resource enum value is represented in at least 1 row
--   * All resource_id and actor_id values are valid for other test data row IDs
INSERT INTO compliance_audit_log (id, timestamp, actor_id, actor_type, resource_id, resource_type, action, resource_action_meta, signature) VALUES
    ('fa71445e-0781-481f-aaad-b7a3ab0d15df', '2024-01-01T12:00:05.120005-10:00', '01HWQE29RW1S1D8ZN58M528A1M', 'user', '01HV6QS6AK4KNS46Q9HEB7DTPR', 'account', 'create', 'test_user_create_account', x'1234567890abcdef'),
    ('a3fdc41a-d4ff-44ab-bce7-7a27ce276d19', '2024-02-02T13:10:15.131015-10:00', '01HWQEJJDMS5EKNARHPJEDMHA4', 'api_key', '01HWQE29RW1S1D8ZN58M528A1M', 'user', 'update', 'test_api_key_update_user', x'1234567890abcdef'),
    ('92928476-caa8-4ca9-9878-1d3326714391', '2024-03-03T14:20:25.142025-10:00', '01JXTGSFRC88HAY8V173976Z9D', 'sunrise', '01HWQEJJDMS5EKNARHPJEDMHA4', 'api_key', 'delete', 'test_sunrise_delete_api_key', x'1234567890abcdef'),
    ('c9f68e80-4eb1-4cd1-ac2a-69236cf71877', '2024-04-04T15:30:35.153035-10:00', '01HWQE29RW1S1D8ZN58M528A1M', 'user', '01JXTQCDE6ZES5MPXNW7K19QVQ', 'counterparty', 'create', 'test_user_create_counterparty', x'1234567890abcdef'),
    ('e81789ca-880e-413a-ba26-f274c6d02b12', '2024-05-05T16:40:45.164045-10:00', '01HWQEJJDMS5EKNARHPJEDMHA4', 'api_key', '01JXTGSFRC88HAY8V173976Z9D', 'sunrise', 'delete', 'test_api_key_delete_sunrise', x'1234567890abcdef'),
    ('f55d0945-f61f-4718-a9d4-9de7005eef00', '2024-06-06T17:50:55.175055-10:00', '01JXTGSFRC88HAY8V173976Z9D', 'sunrise', '2c891c75-14fa-4c71-aa07-6405b98db7a3', 'transaction', 'update', 'test_sunrise_update_transaction', x'1234567890abcdef')
;
