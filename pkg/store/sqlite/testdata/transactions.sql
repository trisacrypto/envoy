-- Counterparty Fixtures
INSERT INTO counterparties (id, source, directory_id, registered_directory, protocol, common_name, endpoint, name, website, country, business_category, vasp_categories, verified_on, ivms101, created, modified) VALUES
    (x'018f305df388d9fef7ee5b5b98756d0a', 'gds', '83237113-db52-4b75-bea0-1c60d6662370', 'trisatest.dev', 'trisa', 'api.alice.vaspbot.net', 'api.alice.vaspbot.net:443', 'Alice VASP', 'https://alice.vaspbot.net', 'US', 'PRIVATE_ORGANIZATION', '["Exchange","DEX"]', '2023-04-08T18:19:44+00:00', null, '2024-04-30T13:56:54.664-05:00', '2024-04-30T13:56:54.664-05:00'),
    (x'018f306466bd88adf67fa5f268575183', 'user', null, null, 'trp', 'bob.vaspbot.net', 'https://api.bob.vaspbot.net', 'Bob VASP', 'https://bob.vaspbot.net', 'GB', 'PRIVATE_ORGANIZATION', '["DEX"]', null, null, '2024-04-30T14:03:57.373-05:00', '2024-04-30T14:03:57.373-05:00'),
    (x'018f3079ac61294ecc5ea4e33a02929b', 'gds', '2666abb0-5e92-4d02-a9ba-5539323e9683', 'trisatest.dev', 'trisa', 'zip.vaspbot.net', 'zip.vaspbot.net:443', 'Zip Wallet, Inc.', 'https://zip.vaspbot.net', 'BR', 'PRIVATE_ORGANIZATION', '["Exchange","Custodial"]', '2023-04-08T18:19:44+00:00', null, '2024-04-30T14:27:11.457-05:00', '2024-04-30T14:27:11.457-05:00')
;

-- Transaction Fixtures
INSERT INTO transactions (id, source, status, counterparty, counterparty_id, originator, originator_address, beneficiary, beneficiary_address, virtual_asset, amount, last_update, created, modified) VALUES
    ('c20a7cdf-5c23-4b44-b7cd-a29cd00761a3', 'local', 'pending', 'AliceVASP', x'018f305df388d9fef7ee5b5b98756d0a', 'Mary Tilcott', 'mjJ9xufmdSfZLRUXV6Ac3r64M6bbrxCu48', 'Sarah Radfeld', '19nFejdNSUhzkAAdwAvP3wc53o8dL326QQ', 'BTC', 0.0003842, '2024-04-30T14:09:57-05:00', '2024-04-30T14:09:57-05:00', '2024-04-30T14:09:57-05:00')
;

-- Secure Envelope Fixtures
-- TODO: how to easily create secure envelope fixtures?