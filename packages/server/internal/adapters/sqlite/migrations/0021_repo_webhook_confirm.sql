-- Per-repo "require confirmation" gate for webhook-triggered reviews. When set,
-- a webhook-triggered review is created in the awaiting_approval state and held
-- (never enqueued) until the user approves it in the app. A simple column add
-- (no table rebuild needed); defaults to 0 so existing repos keep auto-running.
ALTER TABLE repos ADD COLUMN webhook_require_confirmation INTEGER NOT NULL DEFAULT 0;
