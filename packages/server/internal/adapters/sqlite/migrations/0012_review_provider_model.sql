-- provider_id and model pin a review to a specific provider/model chosen at
-- launch. Empty (the default) means resolve from the repo, then the default
-- provider — the pre-existing behavior.
ALTER TABLE reviews ADD COLUMN provider_id TEXT NOT NULL DEFAULT '';
ALTER TABLE reviews ADD COLUMN model TEXT NOT NULL DEFAULT '';
