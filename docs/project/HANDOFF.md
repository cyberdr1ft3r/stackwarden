# StackWarden Handoff

Last updated: 2026-09-04

## Next concrete engineering task

Complete review of Issue #21's CI pull request, require its stable checks in GitHub branch protection after merge, then open a dedicated architecture issue to resolve D-012 for durable inventory and snapshots.

Issue #20 / PR #18 is merged on `main` as `005aa04d4755ab76a4c0ae1ecd963469e5235105`.

## Read first

- Issue #21 and its CI pull request, including checks, reviews, and unresolved threads.
- `AGENTS.md`
- `PROJECT_MEMORY.md`
- `docs/project/STATUS.md`
- `docs/project/DECISIONS.md`
- `docs/project/RISKS.md`
- `docs/project/ROADMAP.md`
- `docs/vision.md`
- `docs/threat-model.md`
- `.agents/skills/project-memory/SKILL.md`
- `.agents/skills/security-boundary/SKILL.md`

## Issue #21 completion conditions

- Pull requests and pushes to `main` trigger checks with stable names for formatting, static analysis, tests, builds, Windows compilation, and vulnerability scanning.
- Workflows use read-only permissions, immutable action pins, no production secrets, and no deployment/publishing behavior.
- Equivalent local commands are documented and pass.
- Representative malformed Go formatting is proven to fail the formatting gate.
- The new pull request's actual GitHub Actions checks pass.
- A maintainer configures every stable check as required for `main` and requires branches to be current before merge.

## Completion conditions

After CI is merged and enforced, resolve D-012 through a dedicated architecture decision covering host identity, snapshot schema, retention, single-host versus future multi-host assumptions, and storage tradeoffs. Do not implement a database until that decision is accepted.
