-- telegram_targets holds configured Telegram notification chats. The bot token
-- is not stored here; token_ref names the encrypted entry in the secrets table.
CREATE TABLE telegram_targets (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    chat_id    TEXT NOT NULL,
    thread_id  TEXT NOT NULL DEFAULT '',
    token_ref  TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);

-- At most one telegram target may be the default.
CREATE UNIQUE INDEX telegram_targets_single_default ON telegram_targets(is_default) WHERE is_default = 1;
