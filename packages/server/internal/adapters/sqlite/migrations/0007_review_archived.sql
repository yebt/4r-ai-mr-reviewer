-- archived soft-hides a review from the main list while keeping its full
-- history. Reviews are active (0) by default.
ALTER TABLE reviews ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;
