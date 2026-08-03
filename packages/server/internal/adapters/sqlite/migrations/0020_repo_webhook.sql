-- Per-repo GitLab webhook: an opt-in receiver that auto-creates and runs a
-- review when a merge request is opened or updated. webhook_secret gates the
-- endpoint (compared against the X-Gitlab-Token header); webhook_enabled toggles
-- the receiver on. Both are simple column adds (no table rebuild needed).
ALTER TABLE repos ADD COLUMN webhook_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE repos ADD COLUMN webhook_enabled INTEGER NOT NULL DEFAULT 0;
