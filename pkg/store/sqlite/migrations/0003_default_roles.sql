-- Populate the database with initial roles and permissions data
BEGIN;

INSERT INTO roles (id, title, description, is_default, created, modified) VALUES
    (1, 'Admin', 'Full access and all permissions; able to manage, view, edit, and delete all resources.', 'f', datetime('now'), datetime('now')),
    (2, 'Compliance', 'Able to manage transactions, counterparties, and customer accounts but not settings or access resources.', 't', datetime('now'), datetime('now')),
    (3, 'Observer', 'Read only access, can view resources but not edit or delete them.', 'f', datetime('now'), datetime('now'))
;

INSERT INTO permissions (id, title, description, created, modified) VALUES
    (1, 'users:manage', 'Can create, edit, and delete users', datetime('now'), datetime('now')),
    (2, 'users:view', 'Can view users registered on the node', datetime('now'), datetime('now')),
    (3, 'apikeys:manage', 'Can create apikeys and view associated secret', datetime('now'), datetime('now')),
    (4, 'apikeys:view', 'Can view apikeys created on the node', datetime('now'), datetime('now')),
    (5, 'apikeys:revoke', 'Can revoke apikeys and delete them', datetime('now'), datetime('now')),
    (6, 'counterparties:manage', 'Can create, edit, and delete counterparties', datetime('now'), datetime('now')),
    (7, 'counterparties:view', 'Can view counterparty details', datetime('now'), datetime('now')),
    (8, 'accounts:manage', 'Can create, edit, and delete accounts and crypto addresses', datetime('now'), datetime('now')),
    (9, 'accounts:view', 'Can view accounts and crypto addresses', datetime('now'), datetime('now')),
    (10, 'travelrule:manage', 'Can create, accept, reject, and archive transactions and send secure envelopes', datetime('now'), datetime('now')),
    (11, 'travelrule:delete', 'Can delete transactions and associated secure envelopes', datetime('now'), datetime('now')),
    (12, 'travelrule:view', 'Can view travel rule transactions and secure envelopes', datetime('now'), datetime('now')),
    (13, 'config:manage', 'Can manage the configuration of the node', datetime('now'), datetime('now')),
    (14, 'config:view', 'Can view the configuration of the node', datetime('now'), datetime('now')),
    (15, 'pki:manage', 'Can create and edit certificates and sealing keys', datetime('now'), datetime('now')),
    (16, 'pki:delete', 'Can delete certificates and sealing keys', datetime('now'), datetime('now')),
    (17, 'pki:view', 'Can view certificates and sealing keys', datetime('now'), datetime('now'))
;

INSERT INTO role_permissions (role_id, permission_id, created, modified) VALUES
    -- Admin Permissions
    (1, 1, datetime('now'), datetime('now')),
    (1, 2, datetime('now'), datetime('now')),
    (1, 3, datetime('now'), datetime('now')),
    (1, 4, datetime('now'), datetime('now')),
    (1, 5, datetime('now'), datetime('now')),
    (1, 6, datetime('now'), datetime('now')),
    (1, 7, datetime('now'), datetime('now')),
    (1, 8, datetime('now'), datetime('now')),
    (1, 9, datetime('now'), datetime('now')),
    (1, 10, datetime('now'), datetime('now')),
    (1, 11, datetime('now'), datetime('now')),
    (1, 12, datetime('now'), datetime('now')),
    (1, 13, datetime('now'), datetime('now')),
    (1, 14, datetime('now'), datetime('now')),
    (1, 15, datetime('now'), datetime('now')),
    (1, 16, datetime('now'), datetime('now')),
    (1, 17, datetime('now'), datetime('now')),

    -- Compliance Permissions
    (2, 2, datetime('now'), datetime('now')),
    (2, 4, datetime('now'), datetime('now')),
    (2, 6, datetime('now'), datetime('now')),
    (2, 7, datetime('now'), datetime('now')),
    (2, 8, datetime('now'), datetime('now')),
    (2, 9, datetime('now'), datetime('now')),
    (2, 10, datetime('now'), datetime('now')),
    (2, 12, datetime('now'), datetime('now')),
    (2, 14, datetime('now'), datetime('now')),
    (2, 17, datetime('now'), datetime('now')),

    -- Observer Permissions
    (3, 2, datetime('now'), datetime('now')),
    (3, 4, datetime('now'), datetime('now')),
    (3, 7, datetime('now'), datetime('now')),
    (3, 9, datetime('now'), datetime('now')),
    (3, 12, datetime('now'), datetime('now')),
    (3, 14, datetime('now'), datetime('now')),
    (3, 17, datetime('now'), datetime('now'))
;

COMMIT;