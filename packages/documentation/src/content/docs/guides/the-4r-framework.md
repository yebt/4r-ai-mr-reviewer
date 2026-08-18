---
title: The 4R framework
description: The four lenses every change is reviewed through, and how the score is derived.
---

Every merge request is examined through four independent lenses. Each lens is a
separate rule set and runs as its own **pass**, so a change is judged from four
angles instead of one blurry "looks fine".

## The four lenses

| Lens | Question it answers |
|---|---|
| **R1 · Risk** | Can this harm security, data, or production stability? |
| **R2 · Readability** | Will the next engineer understand it without an hour of digging? |
| **R3 · Reliability** | Does it behave correctly across the realistic range of inputs? |
| **R4 · Resilience** | Does it degrade gracefully when dependencies fail or slow down? |

The goal is to catch what matters **before** it reaches production, while letting
genuinely safe changes through quickly.

## Findings

A finding is the atom of a review. Each one carries:

- **Dimension** — which lens raised it (R1–R4).
- **Severity** — high / medium / low.
- **Location** — `file:line` on the new side of the diff (or file-level when it is
  not tied to a specific line).
- **Issue / Why / Fix** — the problem, why it matters, and how to address it, kept
  as separate parts so they never blur together.
- **Blocking** — whether it counts against the recommendation.

## Deterministic scoring

The **0–100 score** and the **approve / request-changes** recommendation are
computed from the findings by the backend — they are **not** asked of the model.

Why this matters: given the same findings, you always get the same verdict. The
model's job is to *find and locate problems*; the *judgement* is deterministic and
reproducible. A model that is chatty, terse, or differently-tempered can't move the
score on its own.

## Phased, multi-pass reviews

Each lens runs as its own pass with live progress, so results stream in as they
finish rather than arriving in one lump at the end. You see R1 land while R2 is
still thinking.

Next: [Running reviews](/guides/running-reviews/).
