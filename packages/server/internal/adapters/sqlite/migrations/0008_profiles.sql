-- profiles holds humanization profiles: a user's writing voice used later to
-- rephrase review comments. samples is a JSON array of raw writing samples;
-- style_guide is an LLM-distilled cache filled by a later slice (empty for now).
-- Samples are not secret, so nothing here is encrypted.
CREATE TABLE profiles (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    language    TEXT NOT NULL DEFAULT '',
    formality   TEXT NOT NULL DEFAULT '',
    emojis      INTEGER NOT NULL DEFAULT 0,
    samples     TEXT NOT NULL DEFAULT '[]',
    style_guide TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
