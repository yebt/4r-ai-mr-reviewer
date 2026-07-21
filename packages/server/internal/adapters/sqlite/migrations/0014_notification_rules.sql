-- notification_rules routes events to assigned notifier targets. Each rule can
-- be toggled on or off. notifier_kind selects the notifier backend (e.g.
-- telegram) and notifier_id identifies the concrete target within it.
CREATE TABLE notification_rules (
    id            TEXT PRIMARY KEY,
    event         TEXT NOT NULL,
    notifier_kind TEXT NOT NULL,
    notifier_id   TEXT NOT NULL,
    enabled       INTEGER NOT NULL DEFAULT 1,
    created_at    TEXT NOT NULL,
    -- A given event may route to a given notifier target at most once; a
    -- duplicate would deliver the same notification twice.
    UNIQUE (event, notifier_kind, notifier_id)
);

-- Fan-out looks up enabled rules by event on every fired notification.
CREATE INDEX notification_rules_event ON notification_rules(event, enabled);
