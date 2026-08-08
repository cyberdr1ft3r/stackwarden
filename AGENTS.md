# Codex Repository Instructions

This file is the entry point for every meaningful Codex task in StackWarden.

## Before changing anything

1. Read the full user task, GitHub issue, and relevant comments.
2. Inspect the repository and the files affected by the task.
3. Read the persistent project context:
   - `PROJECT_MEMORY.md`
   - `docs/project/STATUS.md`
   - `docs/project/DECISIONS.md`
   - `docs/project/RISKS.md`
   - `docs/project/ROADMAP.md`
   - `docs/project/HANDOFF.md`
4. Read `docs/vision.md` and `docs/threat-model.md` for product/security-sensitive work.
5. Load every applicable skill under `.agents/skills/` and always load `project-memory` for meaningful repository work.
6. Summarize the current source-of-truth understanding before implementing.
7. Identify conflicts or unresolved decisions instead of guessing.

## Source-of-truth order

When information conflicts, use this order:

1. The current approved GitHub issue and its review comments.
2. Merged code and detailed documentation under `docs/`.
3. `docs/project/DECISIONS.md`.
4. `PROJECT_MEMORY.md` and `docs/project/STATUS.md`.
5. Older PR descriptions, commits, and external conversation context.

Never treat an open PR as merged behavior. Record unresolved conflicts explicitly.

## Architecture boundary

StackWarden is a security-first server visibility and operations control plane.

- Browser/UI: presentation only; it must not have direct OS authority.
- API: policy/gatekeeper layer, validation, authentication/authorization, audit, and agent mediation.
- Agent: privileged execution boundary for approved host observations and narrowly defined actions.
- OS/runtime: the managed host and its services.

Do not bypass these boundaries for convenience.

## Security rules

- Default to local/private exposure. Do not make management interfaces public merely to simplify access.
- Prefer SSH tunnels, private networks, or an explicitly secured reverse proxy for remote access.
- Read-only observation is safer than mutation; new write paths must be explicit, authenticated, bounded, and audited.
- Never add arbitrary shell/command execution APIs.
- Prefer structured argv and allowlisted/catalog-defined actions over shell strings.
- Validate identifiers and paths before filesystem use. Prevent path traversal and escaping managed directories.
- Never commit secrets, tokens, credentials, production host data, or sensitive inventory snapshots.
- Treat the agent as a high-trust component. Keep its transport and permissions as narrow as practical.
- Preserve safe failure modes: disabled-by-default writes, deny-by-default authorization, bounded output/timeouts, and explicit errors.

## Engineering rules

- Work on a dedicated branch for each issue/task.
- Keep changes scoped to the task; do not silently add features or dependencies.
- Preserve Go module/workspace boundaries (`agent`, `api`, `pkg`).
- Run `gofmt` on changed Go code.
- Run `make test` for meaningful code changes; run `make build` when build behavior is touched.
- Add/update tests whenever behavior or a security boundary changes.
- Keep Linux as the primary server target while preserving supported Windows observation behavior unless an approved task says otherwise.
- Open a draft pull request linked to the task/issue; do not merge automatically.

## Persistent memory updates

At the end of every meaningful task:

- Update `docs/project/STATUS.md` when state, blockers, current milestone, or next action changed.
- Replace `docs/project/HANDOFF.md` with the next concrete action and its completion conditions.
- Add accepted decisions to `docs/project/DECISIONS.md`.
- Update `docs/project/RISKS.md` when risks change or are discovered.
- Update `docs/project/ROADMAP.md` when sequencing/dependencies change.
- Update `PROJECT_MEMORY.md` only when stable goals, architecture, product facts, or operating rules change.

Keep memory files factual and compact. They are source-of-truth state, not chat transcripts.

## Completion report

Every meaningful task summary must include:

- Skills used
- Project memory files reviewed and updated
- Files changed
- Commands/tests/checks run and their results
- Security implications
- Assumptions and unresolved decisions
- Scope intentionally not implemented
- Current blockers
- Next recommended action
