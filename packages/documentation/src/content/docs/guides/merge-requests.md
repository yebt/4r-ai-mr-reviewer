---
title: Creating merge requests
description: Open a merge request between two branches with an AI-drafted title and description.
---

Beyond reviewing existing MRs, 4R can **open a new one** for you and draft its
title and description from the diff between two branches.

## The flow

From a repository's **Flow** workspace, press **New MR**. Then:

1. **Pick the branches** — a searchable **source** and **target**. Both must
   already exist (4R does not create branches or commits).
2. **Choose a voice** (optional) — a [voice profile](/guides/humanize/) to write
   the description in, or **Plain English**.
3. **Generate with AI** — the model reads the diff between the two branches and
   drafts a title and a Markdown description.
4. **Review & edit** — the title and description land in editable fields. Nothing
   is created until you confirm.
5. **Create merge request** — opens the MR on GitLab and links you to it.

The draft is always reviewed and editable before anything is created — an
accidental click-away asks to confirm rather than throwing away your work.

## What the model sees

The description is grounded **only** in the branch comparison — the commit
subjects and the per-file diff between `target` and `source`. The diff is capped
in size so a very large comparison can't blow the model's context; when it is
truncated, the prompt says so and the model summarizes from what it was shown.

## Which provider is used

MR generation uses the same **repo → default** provider resolution as reviews
(there is no per-draft override): the repo's provider/model if set, otherwise the
default provider. See [Providers & models](/guides/providers-and-models/).

:::note[Out of scope]
Creating branches and committing files is intentionally out of scope — the two
branches you select must already exist. 4R drafts and opens the MR; it does not
push code.
:::
