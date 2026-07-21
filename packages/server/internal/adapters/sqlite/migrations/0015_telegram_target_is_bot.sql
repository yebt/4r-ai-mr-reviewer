-- is_bot marks the single telegram target whose bot token drives the
-- interactive webhook receiver (the Telegram bot). It mirrors is_default: at
-- most one target may be the bot, enforced by clearing others in a transaction.
ALTER TABLE telegram_targets ADD COLUMN is_bot INTEGER NOT NULL DEFAULT 0;
