---
title: Release routines
description: Automate dev and main releases — versioning, the confirm gate, and the -dev tag option.
---

A **release routine** automates the mechanical steps of cutting a release on
GitLab: compute the next version, open or drive the merge request, wait for CI,
approve, react, **pause for your confirmation**, merge, and tag.

## Two flows

| Flow | What it does | Target |
|---|---|---|
| **Dev** | Releases a single existing merge request. | `development` |
| **Main** | Creates a `development → main` merge request itself, then merges & tags it. | `main` |

Both flows **pause on a confirmation gate** before merging — nothing is merged or
tagged until you confirm. The step ledger is:

```
compute_tag → create_mr → wait_pipeline → approve → react → confirm → merge → tag → notify
```

Because the ledger is checkpointed, a **blocked** run can be **resumed** and skips
the steps it already completed.

## Version bumps

The next tag is computed from the base tag and the commits since it, by mode:

| Bump | Result |
|---|---|
| **major** | `(major+1).0.0`, regardless of commits. |
| **minor** | Raises the minor per **feat** commit; the patch per trailing **fix** (the primary mode). |
| **patch** | Raises the patch per releasable commit; feats ignored. |

Commits are classified by **conventional-commit** subjects (`feat:` / `fix:`, with
optional scope and `!`). The release modal shows a **live preview** of the exact
next tag before you launch, so the version never surprises you at the gate.

## How the main flow picks its base tag

By default the **main flow ignores `-dev` prerelease tags** and bases the next
version on the highest **pure** `X.Y.Z` release. So if your releases stalled at
`v0.4.2` while a `-dev` line ran ahead to `v2.0.0-dev`, a major release computes
`v1.0.0` — the `-dev` tags are not counted.

### Counting `-dev` tags

The release modal has a **Count `-dev` prereleases** toggle for exactly this
situation:

- **Off** (default) — base on the highest pure `X.Y.Z` release; `-dev` tags are
  ignored.
- **On** — base on the highest tag **including** `-dev`, **promoting** the latest
  dev version by dropping the suffix (e.g. `v2.0.0-dev → v2.0.0`), then apply the
  bump on top.

A promotion counts as a real release even when there are no new commits since the
dev tag — so it is **not** treated as "nothing to release". The live preview
reflects the toggle, so pick the bump that lands where you want:

```
tags: … v0.4.2, v1.0.0-dev … v2.0.0-dev

Count -dev  OFF, major  →  v1.0.0   (base v0.4.2)
Count -dev  ON,  minor  →  v2.0.0   (promote v2.0.0-dev)
Count -dev  ON,  major  →  v3.0.0   (bump the promoted core)
```

:::tip[No code, one-time fix]
Prefer not to toggle every time? Create a clean release tag once at the version
you're really on (`git tag v2.0.0 <commit> && git push origin v2.0.0`). From then
on the default logic bases every release on it correctly.
:::

## Reactions & options

- **Reaction emojis** — the react step awards a chosen set of emoji reactions on
  the released MR (GitLab award-emoji *names*). Defaults to 👍 / 🌱.
- **Remove source branch after merge** — off by default.
- **Merge when pipeline succeeds** — main flow defaults this **off** (it already
  gates on a green pipeline in `wait_pipeline`).

The branch pickers warn when the conventional `development` / `main` branches are
absent, so you pick real branches instead of hitting a mid-run failure.
