# StackWarden Project Memory Skill

Use this skill for every meaningful StackWarden repository task.

## Purpose

Keep repository state self-contained so a new coding agent can continue work without relying on chat history.

## Required reads

Before implementation, read:
- `PROJECT_MEMORY.md`
- `docs/project/STATUS.md`
- `docs/project/DECISIONS.md`
- `docs/project/RISKS.md`
- `docs/project/ROADMAP.md`
- `docs/project/HANDOFF.md`

For architecture/security work also read `docs/vision.md` and `docs/threat-model.md`.

## Rules

- Distinguish merged behavior from open-PR proposals.
- Do not convert assumptions into project facts.
- If sources conflict, follow the source-of-truth order in `AGENTS.md`.
- Record unresolved questions rather than silently deciding them.
- Keep memory files compact, factual, and current.
- Reference issue/PR numbers when they establish state.

## End-of-task update

Update only files whose facts changed:

- `STATUS.md`: current milestone, merged/pending state, blockers, next action.
- `HANDOFF.md`: replace with the next concrete task, required reads, checks, and completion conditions.
- `DECISIONS.md`: append/supersede accepted decisions; do not rewrite history.
- `RISKS.md`: add/change risks and mitigation state.
- `ROADMAP.md`: change only when sequencing/dependencies changed.
- `PROJECT_MEMORY.md`: change only stable project purpose, architecture, rules, or long-lived facts.

Do not use project memory as a raw activity log.