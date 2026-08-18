---
title: Backups
description: What to back up — the database and the master key — and how to restore.
---

4R keeps **all** state in one SQLite database. Secrets inside it are encrypted, so
a backup is only useful together with the key that decrypts them.

## What to back up

| Item | Where | Why |
|---|---|---|
| **Database** | `AIR_DB_PATH` (default `ai-reviewer.db`) | All accounts, repos, reviews, routines, settings, and the encrypted secrets. |
| **Master key** | Key-file mode: `AIR_KEYFILE_PATH` (default `<db>.key`). Password mode: your `AIR_PASSWORD`. | Without it, the encrypted secrets in the database can't be decrypted. |

:::danger[Back up both]
Backing up the database **without** the key (or password) leaves you with data you
cannot decrypt. Store the key separately from the database backup — a leaked backup
of both together is a leaked vault.
:::

## Taking a backup

Stop the service (or use SQLite's online backup) and copy both files:

```sh
sudo systemctl stop ai-reviewer

cp /var/lib/ai-reviewer/ai-reviewer.db      /backups/ai-reviewer.db
cp /var/lib/ai-reviewer/ai-reviewer.db.key  /secure-store/ai-reviewer.key   # key-file mode

sudo systemctl start ai-reviewer
```

For a hot backup without stopping, use the SQLite CLI:

```sh
sqlite3 /var/lib/ai-reviewer/ai-reviewer.db ".backup '/backups/ai-reviewer.db'"
```

## Restoring

1. Put the database back at `AIR_DB_PATH`.
2. Provide the **same** master key — restore the key file to `AIR_KEYFILE_PATH`, or
   set the same `AIR_PASSWORD`.
3. Start the service. The vault unlocks and the secrets decrypt.

## Rotating the master key

You can change the master password or re-key the vault at runtime from **Settings →
Security** — 4R re-encrypts the secrets under the new key transactionally. After
rotating, **take a fresh backup** of the database and the new key, and remember that
changing `AIR_AUTH_PASSWORD` (a separate setting) also invalidates all sessions.
