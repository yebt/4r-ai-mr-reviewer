-- default_profile_id is the reviewer voice profile pre-selected when humanizing
-- this repo's reviews. NULL means no default (the user picks a profile ad hoc).
ALTER TABLE repos ADD COLUMN default_profile_id TEXT;
