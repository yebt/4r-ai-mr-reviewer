-- archived soft-hides a routine run from the active list while keeping its full
-- history. Runs are active (0) by default.
ALTER TABLE routine_run ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;
