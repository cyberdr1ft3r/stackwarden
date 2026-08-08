# StackWarden Handoff

Last updated: 2026-08-08

## Next concrete engineering task

Review open PR #18: `Harden local-only baseline; split read/write API and lock down agent`.

Do not start a new remediation/tool feature before this review is complete.

## Read first

- `AGENTS.md`
- `PROJECT_MEMORY.md`
- `docs/project/STATUS.md`
- `docs/project/DECISIONS.md`
- `docs/project/RISKS.md`
- `docs/project/ROADMAP.md`
- `docs/vision.md`
- `docs/threat-model.md`
- PR #18 description, diff, tests, and review comments

## Review objectives

Verify that PR #18:

1. Defaults the API to loopback and rejects accidental non-local binds unless explicitly overridden.
2. Moves the agent away from a generally reachable TCP management endpoint.
3. Uses safe Unix-socket creation/permissions and documents deployment ownership/group expectations.
4. Separates read and write API routes without unintentionally breaking UI/API behavior.
5. Keeps writes disabled by default.
6. Requires authentication when writes are enabled and avoids obvious token handling mistakes.
7. Does not leave alternate legacy write routes reachable without the new controls.
8. Removes free-form shell execution from affected tool flows rather than merely moving it elsewhere.
9. Validates tool IDs and ensures managed paths cannot escape their allowed root.
10. Keeps catalog-defined install/status/uninstall behavior functional.
11. Preserves Linux and supported Windows observation behavior where applicable.
12. Has tests covering the security boundary, not only happy paths.
13. Passes `make test`; run `make build` if build/runtime wiring changed.

## Completion conditions

The review task is complete when:

- Security/behavior findings are documented.
- Required fixes, if any, are implemented on the PR branch or captured as explicit blocking review comments.
- Applicable tests pass.
- `STATUS.md`, `RISKS.md`, `DECISIONS.md`, and this handoff are updated to reflect the result.
- PR #18 is either ready to merge or clearly blocked with concrete reasons.
- No new unrelated product scope is introduced.

## After PR #18

If the security baseline is merged successfully, the next planning task should resolve D-012: durable storage architecture for inventory/snapshots/drift/audit. Do not jump directly to a database implementation without that decision.