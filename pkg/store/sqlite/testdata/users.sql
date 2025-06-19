-- Test users for each role; the password is supersecret-[role] for each
INSERT INTO users (id, name, email, password, role_id, last_login, created, modified) VALUES
    (x'018f2ee1271c0e42d47ea5450a242834', 'Admin User', 'admin@example.com', '$argon2id$v=19$m=65536,t=1,p=2$9ihQHJnCW+bojgqoUWYc/A==$GBaUbq36VeFsoqpHfDZXSzUu+1JUXjO2ein7Bis2r4I=', 1, '2024-04-30T07:07:42-05:00', ' 2024-04-30T07:00:58.652-05:00', '2024-04-30T07:07:42-05:00'),
    (x'018f2ee190f9c50ec5e33ea1ef21d103', 'Compliance User', 'compliance@example.com', '$argon2id$v=19$m=65536,t=1,p=2$hx85bfGI6dBBKNtrcYfQzQ==$1BTMu2bsBlmyzw24F5Y/W6mwtEIvZm3M5e2YGaPc69E=', 2, '2024-04-30T07:07:49-05:00', '2024-04-30T07:01:25.753-05:00', '2024-04-30T07:07:49-05:00'),
    (x'018f2ee1d49935bf09d5913b8c13d51a', 'Observer User', 'observer@example.com', '$argon2id$v=19$m=65536,t=1,p=2$xKG48Fp5R4nZsfC3cSp0dA==$6XZwT5pg6t44ovN55BC59/F4he8qHHxj22lzYrJUbws=', 3, '2024-04-30T07:07:55-05:00', '2024-04-30T07:01:43.065-05:00', '2024-04-30T07:07:55-05:00')
;

-- One API key with complete permissions and one API key with read-only permissions
-- client ID: ISoIuDiGkpVpAyCrLGYrKU    secret: Dah5FqQT8tHtC9UablExfhb2GbmfrJrSiHAXBnDzKI1OQoTa
-- client ID: TPAkoalHEorqAENISHvxYY    secret: HEACkMCWytZquAQQAQoxHKs0LB3h0Mppx93PeSpA5nCVpxYJ
INSERT INTO api_keys (id, description, client_id, secret, last_seen, created, modified) VALUES
    (x'018f2ee949b4c95d3aab11b49cda4544', 'Full permissions keys', 'ISoIuDiGkpVpAyCrLGYrKU', '$argon2id$v=19$m=65536,t=1,p=2$XndK1CI4C1mbOcE25aV8PA==$9NlkyH58LyOmH7oNg38VmB49uoIpa89k7afqABbh+o8=', '2024-04-30T07:13:03-05:00', '2024-04-30T07:09:51.796-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 'Readonly keys', 'TPAkoalHEorqAENISHvxYY', '$argon2id$v=19$m=65536,t=1,p=2$8J11ntVv8i3YBGA74QCS/w==$mOINU411zwT0lNO03UBkMI7l9Mz7rA3XAiQpDIXVVh0=', '2024-04-30T07:13:47-05:00', '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:13:47-05:00')
;

INSERT INTO api_key_permissions (api_key_id, permission_id, created, modified) VALUES
    (x'018f2ee949b4c95d3aab11b49cda4544', 1, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 2, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 3, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 4, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 5, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 6, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 7, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 8, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 9, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 10, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 11, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 12, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 13, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 14, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 15, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 16, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2ee949b4c95d3aab11b49cda4544', 17, '2024-04-30T07:13:03-05:00', '2024-04-30T07:13:03-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 2, '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:11:08.023-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 4, '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:11:08.023-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 7, '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:11:08.023-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 9, '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:11:08.023-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 12, '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:11:08.023-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 14, '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:11:08.023-05:00'),
    (x'018f2eea7377bbde57a557d86d5597a0', 17, '2024-04-30T07:11:08.023-05:00', '2024-04-30T07:11:08.023-05:00')
;

INSERT INTO reset_password_link (id, user_id, email, expiration, signature, sent_on, created, modified) VALUES
    -- For the user "Observer User" (ID: "01JXTGSFRC88HAY8V173976Z9D")
    -- NOTE: the signature's timestamp will differ from the created/sent/modified timestamp but the signature will load in vero
    (x'0197750cbf0c4222af236138d2737d2d', x'018f2ee1d49935bf09d5913b8c13d51a', "observer@example.com", '2024-11-16T17:43:53-05:00', x'0197750cbf0c4222af236138d2737d2db0ccb8e986f8a8c93097ffe008098fb5a6f3d5b5844b140accf8033974223d6f390fb4fdd3afe5f991a1c6ba56395cd93013783b0c5174a3362c22e0fa1f9f40d23b4abf4405cd24b60eacf0ef001a3abc0c9e803118ee98bb7ffbd563cd021c95bde00a88f26b4a55', '2024-11-16T17:28:45-05:00', '2024-11-16T17:28:57-05:00', '2024-11-16T17:28:57-05:00')
;
