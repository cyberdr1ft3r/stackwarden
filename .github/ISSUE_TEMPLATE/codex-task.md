---
name: Codex task
description: Scoped implementation/review task for StackWarden
about: Create a task with source-of-truth context, security constraints, and completion checks
---

# Objective

<!-- What concrete outcome is required? -->

# Why

<!-- Operational/security/product reason. -->

# In scope

- 

# Out of scope

- 

# Source of truth / context

- `AGENTS.md`
- `PROJECT_MEMORY.md`
- `docs/project/STATUS.md`
- `docs/project/DECISIONS.md`
- `docs/project/RISKS.md`
- `docs/project/HANDOFF.md`

<!-- Add relevant docs/PRs/issues/files. -->

# Security considerations

<!-- Network exposure, auth, agent privilege, shell/commands, filesystem paths, secrets, third-party installers, etc. -->

# Acceptance criteria

- [ ] Behavior matches the approved objective.
- [ ] No unrelated scope was added.
- [ ] Security boundaries remain explicit and deny/safe by default.
- [ ] Tests cover changed behavior and failure/denied paths where relevant.
- [ ] `make test` passes for code changes.
- [ ] `make build` passes when build/runtime wiring changes.
- [ ] Project memory/handoff files are updated if state changed.
- [ ] Draft PR summarizes files changed, tests, assumptions, risks, and unresolved decisions.

# Manual verification

<!-- Commands/steps that prove behavior on a representative host. -->

# Open questions

- 
