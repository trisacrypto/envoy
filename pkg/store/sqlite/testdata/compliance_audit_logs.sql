-- Compliance Audit Logs:
--   * Each possible actor/action/resource enum value is represented in at least 1 row
--   * All resource_id and actor_id values are valid for other test data row IDs
INSERT INTO compliance_audit_log (id, actor_id, actor_type, resource_id, resource_type, resource_modified, action, change_notes, signature, key_id, algorithm) VALUES
    -- ID: "01JZ1HNFJ9KTA3Z6Q4RB3X9W2T"
    (x'0197c31abe499e943f9ae4c2c7d4f05a', '01HWQE29RW1S1D8ZN58M528A1M', 'user', '01HV6QS6AK4KNS46Q9HEB7DTPR', 'account', '2024-01-01T12:00:05.120005-10:00', 'create', 'test_user_create_account', x'1234567890abcdef', 'abcdef1234567890', 'MOCK'),
    -- ID: "01JZ1HNFJ9CXXT86XY2KFEJZCC"
    (x'0197c31abe49677ba41bbe14dee97d8c', '01HWQEJJDMS5EKNARHPJEDMHA4', 'api_key', '01HWQE29RW1S1D8ZN58M528A1M', 'user', '2024-02-02T13:10:15.131015-10:00', 'update', 'test_api_key_update_user', x'1234567890abcdef', 'abcdef1234567890', 'MOCK'),
    -- ID: "01JZ1HNFJ9Z1964DR4GQ8AWHDA"
    (x'0197c31abe49f85262370485d0ae45aa', '01JXTGSFRC88HAY8V173976Z9D', 'sunrise', '01HWQEJJDMS5EKNARHPJEDMHA4', 'api_key', '2024-03-03T14:20:25.142025-10:00', 'delete', 'test_sunrise_delete_api_key', x'1234567890abcdef', 'abcdef1234567890', 'MOCK'),
    -- ID: "01JZ1HNFJ96A43DZCFYER9NP70"
    (x'0197c31abe49328836fd8ff3b09ad8e0', '01HWQE29RW1S1D8ZN58M528A1M', 'user', '01JXTQCDE6ZES5MPXNW7K19QVQ', 'counterparty', '2024-04-04T15:30:35.153035-10:00', 'create', 'test_user_create_counterparty', x'1234567890abcdef', 'abcdef1234567890', 'MOCK'),
    -- ID: "01JZ1HNFJ9GRFCV135M3KWYEBH"
    (x'0197c31abe49861ecd8465a0e7cf3971', '01HWQEJJDMS5EKNARHPJEDMHA4', 'api_key', '01JXTGSFRC88HAY8V173976Z9D', 'sunrise', '2024-05-05T16:40:45.164045-10:00', 'delete', 'test_api_key_delete_sunrise', x'1234567890abcdef', 'abcdef1234567890', 'MOCK'),
    -- ID: "01JZ1HNFJ91E1A0HFFZ3XBWKAF"
    (x'0197c31abe490b82a045eff8fabe4d4f', '01JXTGSFRC88HAY8V173976Z9D', 'sunrise', '2c891c75-14fa-4c71-aa07-6405b98db7a3', 'transaction', '2024-06-06T17:50:55.175055-10:00', 'update', 'test_sunrise_update_transaction', x'1234567890abcdef', 'abcdef1234567890', 'MOCK')
;
