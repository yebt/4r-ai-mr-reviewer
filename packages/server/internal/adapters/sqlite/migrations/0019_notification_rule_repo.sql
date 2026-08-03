-- Add an optional per-repo scope to notification rules. An empty repo_id keeps
-- a rule GLOBAL; a non-empty repo_id scopes it to one repo. Routing applies
-- OVERRIDE semantics: a repo with its own scoped rules replaces the global rules
-- for that repo (see internal/app/notifications/service.go).
--
-- The uniqueness key is widened from (event, notifier_kind, notifier_id) to
-- (event, repo_id, notifier_id) so a global rule and a repo-scoped rule for the
-- same event and notifier target can coexist. SQLite cannot alter an inline
-- constraint in place, so the table is rebuilt (the standard 12-step recreate).
CREATE TABLE notification_rules_new (
    id            TEXT PRIMARY KEY,
    event         TEXT NOT NULL,
    notifier_kind TEXT NOT NULL,
    notifier_id   TEXT NOT NULL,
    repo_id       TEXT NOT NULL DEFAULT '',
    enabled       INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL,
    -- A given event may route to a given notifier target at most once per scope
    -- (global or a specific repo); a duplicate would deliver twice.
    UNIQUE (event, repo_id, notifier_id)
);

INSERT INTO notification_rules_new (id, event, notifier_kind, notifier_id, repo_id, enabled, created_at)
    SELECT id, event, notifier_kind, notifier_id, '', enabled, created_at FROM notification_rules;

DROP TABLE notification_rules;
ALTER TABLE notification_rules_new RENAME TO notification_rules;

-- Fan-out looks up enabled rules by event on every fired notification.
CREATE INDEX notification_rules_event ON notification_rules(event, enabled);
