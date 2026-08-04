-- Persist the merge request's source and target branch on the review, so the
-- review view can show "<source> → <target>". Both are captured at run time from
-- the fetched MR changes; a review that never reached the build step keeps them
-- empty. Simple column adds (no table rebuild); default '' keeps existing rows
-- valid.
ALTER TABLE reviews ADD COLUMN source_branch TEXT NOT NULL DEFAULT '';
ALTER TABLE reviews ADD COLUMN target_branch TEXT NOT NULL DEFAULT '';
