# StackWarden Handoff

Last updated: 2026-09-04

## Next concrete engineering task

After the D-012 architecture PR merges, implement [Issue #23](https://github.com/cyberdr1ft3r/stackwarden/issues/23): API-owned durable SQLite inventory and history storage.

Do not begin implementation from this architecture branch and do not add persistence code to the D-012 PR.

## Read first

- Issue #23 and all comments.
- `docs/architecture/durable-storage.md`
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

## Required implementation sequence

1. Add the API storage/repository boundary and select a reviewed pure-Go SQLite driver.
2. Add secure path/bootstrap handling and embedded checksum-verified migrations.
3. Implement the host, revision, snapshot, listener, service, tool, drift, and audit schema.
4. Persist complete normalized snapshots atomically and deduplicate revisions.
5. Add bounded history queries, retention/pruning, backup/restore validation, corruption refusal, and safe shutdown.
6. Integrate handlers without changing route authorization or response compatibility.

## Completion conditions

- Every acceptance criterion in Issue #23 and the D-012 architecture is tested.
- No Bearer token, credential, environment content, request body, PID, raw command output, or automatic host fingerprint is persisted.
- Migration, rollback, concurrency, retention, indexed query, restart, backup/restore, corruption, permission, and path/symlink tests pass.
- Linux behavior remains functional; Windows agent/API code compiles without execution.
- All six mandatory CI checks pass.
- Project memory describes persistence as current behavior only after the implementation PR merges.
