-- accounts holds configured GitLab connections. The token is not stored here;
-- token_ref names the encrypted entry in the secrets table.
CREATE TABLE accounts (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    base_url   TEXT NOT NULL,
    token_ref  TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- providers holds configured AI providers. api_key_ref names the encrypted
-- entry in the secrets table.
CREATE TABLE providers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL,
    base_url    TEXT NOT NULL DEFAULT '',
    model       TEXT NOT NULL DEFAULT '',
    api_key_ref TEXT NOT NULL,
    is_default  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL
);

-- At most one provider may be the default.
CREATE UNIQUE INDEX providers_single_default ON providers(is_default) WHERE is_default = 1;
