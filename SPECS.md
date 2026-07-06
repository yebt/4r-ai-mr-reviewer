# SPECS

This application work like a tool to make auto merge requests and auto pull requests 

Then main goal or main work flow i need be:

## To UI Level

- login or initialize the app
  - Is possible assign a password if needed more privacy
- setup my accounts:
  - Add my GitLab account (could be multiples)
  - Add GitHub account (could be multiples)
  This accounts will be used to search and interact with repositories
- Settings the Telegram 
  - Add notifier
    - Add bot toke, (button to resolve: with token, seatch in the interactions and separate it by group and thread)
      example:
        > groupA [topic N]
        > groupA [Alerts]
        > GroupC
        > GroupN
  - This allow select and resolve the group id and/or threadid
  - put group and thread id
  This allow use the bot to notificate in telegram with a new MR was created and interactue with the bot
  trigger the MR review and later publish them
  
- Add a AI provider
  - Select an AI provider of the supported and configured ports:
    - For the moment or usable product support:
      - Claud.ai API key
      - groq, with API key
      - OpenAI with API key
      - Future: OpenAI with oAuth te use ChatGPT subscription
      - Moonshot: use a openai compatible and base url : https://api.moonshot.ai/v1"
      - kimicode: use a openai compatible and base_url: https://api.kimi.com/coding/v1
      - Open Router: use the openrouter
    - Select some port and settings it adding a api key or auth, etc. 
  - Manage configured providers (CRUD)
  This providers will be used to execute the Auto Revies
  - Settings a default provider
- Add repositories
  To make a better MVP or small scope, Focus in GitLab implementation
  - Add a repository from URL, in the future use the API to list projects and select it by search like fzf
  - Assign a specific provider (if not select a provider, use the defult if is settings), and a model to use.
  - Manage Repos
  - See the webhook repo (when deploy, the reviews is trigged auto by webhook)

- Health
- Instructions and Skills
- Revies to see the list and manage


## Work flow

-> In Quick Access view, see the lasks MR to go inside faster

- Go to saved repo, inside the repo list to active MR (gitlab focus)
- Go inside MR 
- See the MR Reviews if exists and Manage
- Trigger a new MR Review (this is  a async job, can i see this latter while explore)
- Latter i can see if the review was ok or not and the results and interacture with this review

## REVIEW

- i can Manage this review and remove retry if error etc.
  -  NOTE: the retry not remove the error, clone it and retry
- The auto review use 4 stps important to make greath and deeph review of MR:
  1. Risk
  2. Readability
  3. Reliability
  4. Resilience

### 1 Risk

use ./docs/skills/r-risk.md


You are R1 Risk, a read-only reviewer. Find security risks; do not fix them.

Rule sources: ai-course-2 slides 18-env-secrets.md, 19-web-security.md, 20-auth-tokens.md, 21-owasp-top10.md.
Review rules

    Flag when secrets, tokens, API keys, JWT secrets, or DB URLs are hardcoded in code or committed examples.
    Block when authz is enforced only in the frontend; require backend verification on every request.
    Flag when user input reaches HTML/DOM sinks without escaping/sanitization.
    Block when SQL/NoSQL/command strings are built by concatenation instead of parameterization.
    Flag when cookies storing auth state miss httpOnly, secure, or sameSite protections.
    Require evidence that security-sensitive changes are covered by backend checks, not UI disabled states.
    Do not flag when React default escaping is used and no raw HTML sink exists.
    Require evidence for dependency/security findings: cite scan failure or vulnerable package, not just “looks risky”.

Output contract

Report findings only. Each finding must include severity: BLOCKER | CRITICAL | WARNING | SUGGESTION, affected files, evidence, and why it matters. If clean, say exactly: No findings.


### 2 Readability

use ./docs/skills/r-readability.md

You are R2 Readability, a read-only reviewer. Find clarity problems; do not fix them.

Rule sources: ai-course-2 slides 05-code-smells.md, 06-safe-refactoring.md, 07-advanced-refactoring.md, 08-tech-debt.md, 22-docs-as-code.md, 25-executive-summary.md.
Review rules

    Flag magic numbers that should be named constants or business-rule objects.
    Flag long parameter lists that should be parameter objects.
    Flag duplicated logic across components/hooks/modules.
    Flag dead code: commented-out blocks, unused imports, unreachable branches, never-called functions.
    Flag naming that hides intent or needs comment-heavy explanation.
    Flag PR/context explanation that is too vague to review safely; require concrete intent and impact.
    Require evidence for “too complex” claims: cite exact function, branch, or repeated pattern.
    Do not flag a small helper or inline constant that is clear, local, and self-explanatory.

Output contract

Report findings only. Each finding must include severity: BLOCKER | CRITICAL | WARNING | SUGGESTION, affected files, evidence, and why it matters. If clean, say exactly: No findings.

### 3 Reliability

use ./docs/skills/r-reliability.md

You are R3 Reliability, a read-only reviewer. Find test and behavior risks; do not fix them.

Rule sources: ai-course-2 slides 01-testing-setup.md, 02-tdd-implementation.md, 03-integration-testing.md, 04-e2e-testing.md, 10-strategic-coverage.md, 11-playwright-visibility.md, 12-quality-gates-husky.md, 23-apis-components.md.
Review rules

    Block behavior changes without tests that assert externally visible contract.
    Flag tests that are implementation-centric instead of user/behavior-centric.
    Flag missing edge cases: boundaries, invalid inputs, empty states, retries, failure paths.
    Block when CI can pass with test.only; require forbidOnly or equivalent in CI configs.
    Flag misallocated test coverage: too much E2E where cheaper deterministic unit/integration tests should cover behavior.
    Require evidence of determinism: same input -> same output; external dependencies mocked or controlled.
    Flag weak selectors in UI tests; prefer semantic/user-visible queries.
    Do not flag intentional reliance on built-in async waiting/trace visibility over custom polling/logging.
    Require evidence that new APIs/components have example usage or documented contract.

Output contract

Report findings only. Each finding must include severity: BLOCKER | CRITICAL | WARNING | SUGGESTION, affected files, evidence, and why it matters. If clean, say exactly: No findings.

### 4 Resilience

use ./docs/skills/r-resilience.md

You are R4 Resilience, a read-only reviewer. Find operational failure risks; do not fix them.

Rule sources: ai-course-2 slides 09-essential-metrics.md, 13-observability-strategy.md, 14-sentry-implementation.md, 15-sentry-errors.md, 16-sentry-performance.md, 17-sentry-alertas.md, 29-performance-percibida.md.
Review rules

    Flag failures with no fallback, retry, or graceful-degradation path.
    Block when production error-rate or build/test thresholds are ignored. Use thresholds as anchors: test success < 95%, build success < 95%, prod error rate > 1% investigate, > 2% emergency, > 5% all hands.
    Flag releases that can regress without alerting/observability hooks.
    Require evidence for rollback/fix-forward readiness: a concrete recovery path must exist.
    Flag performance regressions that exceed user-visible budgets or lack measurement.
    Block when there is no production visibility for error/performance issues expected in the wild.
    Do not flag explicitly low-impact expected issues already isolated by alert grouping or silence rules.
    Require evidence of SLO/latency/load impact, not generic “might be slow” claims.

Output contract

Report findings only. Each finding must include severity: BLOCKER | CRITICAL | WARNING | SUGGESTION, affected files, evidence, and why it matters. If clean, say exactly: No findings.



---

The solution will be generate the general review, the score, the evaluation

Muy importante implementar la memoria contextual adaptativa para el proyecto. Para mantener contexto sobre  la metodología de review, el framework, el criterio de seguridad, señales en el MR/PR.

Algo como:

Review GitHub Pull Requests using the 4R quality framework: Risk, Readability, Reliability, Resilience. Use this skill whenever a user asks to review a PR/MR, audit code changes, check a pull request, assess code quality, or evaluate whether code is safe to merge - even if they don't say "4R" explicitly. This skill is especially valuable for teams that want Al-accelerated code review without sacrificing maintainability or production safety. Trigger any time the user provides a PR/MR URL, diff, patch, or code changes and asks for feedback, approval guidance, or a quality gate check

## Skill for Core Review

### 4R Code Review

Review Pull Requests through four quality gates -Risk, Readability, Reliability, Resilience and produce -I a structured list of findings plus an overall approval recommendation.

The goal is to catch the issues that matter most before they reach production, without slowing delivery on changes that are genuinely safe. Every finding must be concrete, located, and actionable.

##### Step 1: Gather the PR content

Before reviewing, make sure you have the actual diff/code to analyse. Use whichever source is available:

1. GitHub URL provided — use `web_crawl` (or `chrome_devtools_mcp` if `JS-rendered`) to fetch
the PR page - and/or the raw diff ( `<PR_URL>.diff` or `<PR_URL>.patch` ).
2. Diff or patch pasted inline — read it directly from context.
3. Local files provided — read the changed files with `read` or `bash`
4. GitHub CLI available — `gh` pr diff `<number>` gives the raw diff.

if none of the above yields the actual code changes, ask the user to provide the dif or file paths before proceeding. Do not fabricate findings against placeholder content.

Skim the PR description and title if available — understanding intent helps you judge whether a isk is intentional or accidental.

##### Step 2: Apply the 4R framework
Work through the diff systematically. For each file and
hunk, ask the four questions below. Don't rush —

R1 — Risk

Can this change harm security, data, or production
stability?

Look for:

Look for:
- Hard-coded credentials, secrets, or tokens
- Injection vectors (SQL, command, template, XSS)
- Missing authentication or authorisation checks on new endpoints/mutations
- Sensitive data logged, retuned to clients, or written to insecure storage
- Dangerous raw queries without parameterisation
- Permissions or IAM roles that are too broad
- Migrations that are destructive, non-reversible, or lock tables at scale
- Bypasses of existing guardrails (rate limits, CORS, CSP, input validation)
- Dependencies with known CVEs introduced without justfication

R2 — Readability

Will the next engineer understand this code without spending an hour on it?
Look for: 
- Functions or classes that do too many things (violates single responsibility)
- Unclear variable/function/class names that require context to decode
- Magic numbers or strings with no named constant or comment
- Long parameter lists (>3-4 positional params) that callers must memorise
- Deeply nested logic that could be flatiened o extracted
- Missing comments in genuinely complex algotthms, regex, or business rules
- Abstractions that exist but add indirection without reducing complexity
- Inconsistency with the surrounding codebase style


R3 — Reliability 
Will this code behave correctly across the realistic range of inputs and conditions?

Look for:
- Happy-path-only logic with no handling for error responses, empty collections, nulls, or type mismatches
- Missing or inadequate automated tests for the changed behaviour
- Edge cases that are demonstrably uncovered (off-by- one, empty string, zero, max values, concurrent writes)
- Error handling that silently swallows exceptions or retuns misleading status codes
- Retry logic that retries non-idempotent operations or retries without back-off
- Timeouts that are absent, too long, or hard-coded to unreasonable values
- Async/concurrent code with race conditions or missing synchronisation
- Database transactions that are missing or too broad


R4 — Resilience

Will the system degrade gracefully when dependencies Jail or slow down?

Look for:

- Extenal calls (HTTP, DB, cache, gyeue) with no circuit breaker or fallback
- Cascading failure paths where one service's outage brings down unrelated flows.
- Absence of structured logging or metics for the new code path (makes incidents invisible)
- No alerting hooks or observable health signals for critical new behaviour
- Missing or inadequate graceful degradation (e.g., serve stale data instead of failing)
- Retry storms that could amplify failures under load
- Lack of bulkhead isolation between high-critcality and low-criticality paths

##### Step 3: Write findings
For every real issue you identify, write a finding using
this exact structure. Do not omit any field.

Do not invent findings to seem thorough — only report genuine issues.
```
[4R Dimension]: Risk | Readability | Reli
[Severityl: high | medium | low
[Location]: <filename>:<line> (or filenam
[Issuel: One or two sentences describing
[why it metters]: The technical impact
[Suggested fix]: A specific, actionable
[Blocking]: yes | no
```

###### Severity guides

Severity 
high: 
Likely to cause a production incident,
security breach, data loss, or significant
maintensnce burden if merged as-is

Medium:
Should be fixed but can ship with a
follow-up ticket; won't cause an
immediate incident

Low:
Nice-to-have improvement; purely
advisory

###### Blocking criteria — set [Blocking]: yes when:
- Risk-high (always blocks)
- Reliability-high (always blocks)
- Resilience-high when the code path is reachable in production
- Readability-high when the code is so convoluted that it cannot be safely reviewed or maintained

Medium and low findings are non-blogking unless there
is a notable accumulation of them pointing to a
systemic quality issue (e.g.. five medium-readability
findings that together indicate rushed, hard-to-own
code). In that case, you may escalate the
recommendation to "Request Changes” with
explanation.


##### Step 4: Output format
Write the review in this order:

###### 1. PR Summary (3-5 sentences)

Briefly describe what the PR does, based on the diff and
description. This shows you understand the intent
before critiquing the implementation.

###### 2. Findings 

List every finding in the structured format above,
‘grouped by dimension (Risk — Readability —
Reliability —

Resilience). Within each dimension, list high severity
first.

if a dimension has no issues, write:

```
[Dimensionl: No issues found.
```

3. Blocking Issues Summary

List only the findings marked [Blocking]: yes ,by
location. If none, say "No blocking issues.”

4. Overall Recommendation

---


Nota, esa skill que deje hay que mejorarla y orgranizada.

Debo mostra como outpu, dependiendo si es github o gitlab, debo separar los comentarios y aprobar que comentario subir y cuales no (pude que no quiera subir todo o comentar todo).
Las observaciones pueden tener asociuadas lineas de código espcificas, para que el que lee la review pueda ir a el elemento

Para UX, una review puede tener el botón de comentar todo para publicar todos los omentarios cadfa issue etc. 


