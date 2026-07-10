-- phase reports fine-grained progress while a review is running (e.g. the
-- current 4R lens in a multi-pass review). Empty otherwise.
ALTER TABLE reviews ADD COLUMN phase TEXT NOT NULL DEFAULT '';
