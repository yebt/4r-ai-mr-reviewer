-- repos are tracked repositories. Deleting an account cascades to its repos;
-- deleting a provider clears the assignment (NULL = use the default provider).
CREATE TABLE repos (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL,
    account_id  TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    provider_id TEXT REFERENCES providers(id) ON DELETE SET NULL,
    model       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL
);

CREATE INDEX repos_account ON repos(account_id);
