# StackWarden Handoff

Last updated: 2026-09-04

## Next concrete engineering task

Perform final maintainer review of PR #18, merge it if approved, then execute Issue #21 to add CI and security quality gates.

Issue #20 implementation and local verification are complete on PR #18. Do not merge automatically and do not broaden PR #18 with Issue #21 work.

## Read first

- PR #18 description, current diff, checks, reviews, and unresolved threads.
- Issue #21, including all acceptance criteria and comments, after PR #18 merges.
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

## Verified PR #18 state

- `make run-api` starts on `127.0.0.1:8080`; non-loopback binds fail unless `ALLOW_NONLOCAL_BIND=1`.
- UI reads use `/v1/read/*`; UI writes attach a token held only in page memory.
- Writes return `403 write_disabled` by default. Enabled install and uninstall routes reject missing/invalid tokens and accept the configured Bearer token.
- Legacy install/uninstall paths return 404.
- API-to-agent health succeeds over a Unix socket; the tested socket directory/file modes are `0750`/`0660`.
- Tool traversal, malformed API/agent/template IDs, and symlinks at managed directories, nested staged paths, or Compose files are rejected.
- Failed uninstall teardown retains staged configuration for recovery.
- A repository command review found structured argv execution in the affected flows and no user-influenced shell-string execution.
- Focused security tests, `make test`, `make build`, Windows cross-builds, and Windows test compilation pass.
- Linux curl checks pass. The verification environment lacked `ss`; `netstat` confirmed the API on `127.0.0.1:8080` and no `:9091` agent TCP listener.

## Completion conditions

1. A maintainer confirms PR #18's final diff and resolved review threads, then merges it.
2. Issue #21 adds automated Go formatting, test, build, Windows compilation, and appropriate security checks.
3. Project memory is updated to describe the security baseline as merged `main` behavior.
4. No remediation, persistence, or unrelated product work is bundled into Issue #21.

After CI is established, resolve D-012 through a dedicated architecture decision for durable inventory, snapshots, drift, and audit storage before implementing a database.
