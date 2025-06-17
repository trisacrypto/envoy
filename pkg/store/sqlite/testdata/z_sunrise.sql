-- Sunrise message records
-- NOTE: this file must run after `transactions.sql` and `counterparties.sql`
INSERT INTO sunrise (id,envelope_id,email,expiration,signature,status,sent_on,verified_on,created,modified) VALUES
    -- ID: "01JXTGSFRC88HAY8V173976Z9D"
    -- NOTE: the signature's timestamp and RecordID will differ from this record, but the signature will load in vero
    (x'0197750cbf0c4222af236138d2737d2d','b04dc71c-7214-46a5-a514-381ef0bcc494','compliance@daybreak.example.com','2024-11-16T17:43:53-05:00',x'0197750cbf0c4222af236138d2737d2db0ccb8e986f8a8c93097ffe008098fb5a6f3d5b5844b140accf8033974223d6f390fb4fdd3afe5f991a1c6ba56395cd93013783b0c5174a3362c22e0fa1f9f40d23b4abf4405cd24b60eacf0ef001a3abc0c9e803118ee98bb7ffbd563cd021c95bde00a88f26b4a55','pending','2024-11-16T17:29:05-05:00','2024-11-16T17:37:37-05:00', '2024-11-16T17:28:57-05:00', '2024-11-16T17:37:37-05:00')
;
