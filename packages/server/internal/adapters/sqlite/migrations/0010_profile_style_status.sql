-- style_guide_status tracks async distillation of a profile's samples into its
-- style_guide: '' (none), 'pending', 'ready', or 'error'. style_guide_error
-- holds the last failure reason. Both are server-managed (never set via Save).
ALTER TABLE profiles ADD COLUMN style_guide_status TEXT NOT NULL DEFAULT '';
ALTER TABLE profiles ADD COLUMN style_guide_error TEXT NOT NULL DEFAULT '';
