-- Advanced provider settings: an optional generation temperature (NULL means
-- "do not send it") and a list of preset model names (stored as a JSON array)
-- to pick from when configuring a repo.
ALTER TABLE providers ADD COLUMN temperature REAL;
ALTER TABLE providers ADD COLUMN models TEXT NOT NULL DEFAULT '[]';
