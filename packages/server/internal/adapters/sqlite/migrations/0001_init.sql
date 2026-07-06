-- app_meta stores single-row runtime metadata such as the KDF salt and the
-- password verifier used to unlock the secret cipher.
CREATE TABLE app_meta (
    key   TEXT PRIMARY KEY,
    value BLOB NOT NULL
);

-- secrets holds encrypted sensitive values (API keys, tokens). The plaintext
-- never touches disk; ciphertext is nonce || AES-256-GCM output.
CREATE TABLE secrets (
    name       TEXT PRIMARY KEY,
    ciphertext BLOB NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
