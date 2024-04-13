-- Schema for user and api key authentication.
BEGIN;

-- Roles are collections of permissions that can be quickly assigned to a user
CREATE TABLE IF NOT EXISTS roles (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL UNIQUE,
    description     TEXT,
    is_default      BOOLEAN DEFAULT false NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

-- Permissions authorize users and api keys to perform actions on the api
CREATE TABLE IF NOT EXISTS permissions (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL UNIQUE,
    description     TEXT,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

-- Maps permissions to roles
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id         TEXT NOT NULL,
    permission_id   TEXT NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE RESTRICT
);

-- Primary authentication table that holds usernames and argon2 derived key passwords
CREATE TABLE IF NOT EXISTS users (
    id              TEXT PRIMARY KEY,
    name            TEXT,
    email           TEXT NOT NULL UNIQUE,
    passwords       TEXT NOT NULL UNIQUE,
    role_id         TEXT NOT NULL,
    last_login      DATETIME,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL,
    FOREIGN KEY (role_id) REFERENCES roles (id) ON DELETE RESTRICT
);

-- API Authentication using authorization bearer tokens and argon2 derived key secrets
CREATE TABLE IF NOT EXISTS api_keys (
    id              TEXT PRIMARY KEY,
    client_id       TEXT NOT NULL,
    secret          TEXT NOT NULL,
    last_seen       DATETIME,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL
);

-- Maps permissions to api keys
CREATE TABLE IF NOT EXISTS api_key_permissions (
    api_key_id      TEXT NOT NULL,
    permission_id   TEXT NOT NULL,
    created         DATETIME NOT NULL,
    modified        DATETIME NOT NULL,
    PRIMARY KEY (api_key_id, permission_id),
    FOREIGN KEY (api_key_id) REFERENCES api_keys (id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions (id) ON DELETE RESTRICT
);

-- Allows selection of all permissions for a user based on their role
DROP VIEW IF EXISTS user_permissions;
CREATE VIEW user_permissions AS
    SELECT u.id AS user_id, p.title AS permission
        FROM users u
        JOIN role_permissions rp ON rp.role_id = u.role_id
        JOIN permissions p ON p.id = rp.permission_id
;

-- Allows selection of all permissions for an api key by title
DROP VIEW IF EXISTS api_key_permission_list;
CREATE VIEW api_key_permission_list AS
    SELECT k.api_key_id, p.title AS permission
        FROM api_key_permissions k
        JOIN permissions p ON p.id = k.permission_id
;


COMMIT;