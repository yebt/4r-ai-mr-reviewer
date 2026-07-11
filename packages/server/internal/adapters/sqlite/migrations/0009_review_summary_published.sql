-- summary_published records whether the review's summary/score header has been
-- posted to the merge request. Unposted (0) by default; set on first publish.
ALTER TABLE reviews ADD COLUMN summary_published INTEGER NOT NULL DEFAULT 0;
