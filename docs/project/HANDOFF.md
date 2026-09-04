# StackWarden Handoff

Last updated: 2026-09-04

## Next concrete engineering task

Execute Issue #20: `Complete and validate StackWarden security baseline` against open PR #18: `Harden local-only baseline; split read/write API and lock down agent`.

Do not start Issue #21 or a new remediation/tool feature before Issue #20 is complete.

## Read first

- Issue #20, including all acceptance criteria and comments.
- PR #18 description, current diff, checks, reviews, and unresolved threads.
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

## Required work

Verify and fix PR #18 so that:

1. `make run-api` succeeds with a loopback-safe default.
2. Non-loopback API binds remain denied unless explicitly enabled.
3. UI and API read routes consistently use valid `/v1/read/*` behavior.
4. Writes remain disabled by default.
5. Every enabled write route requires server-side Bearer authorization.
6. The supported UI write flow supplies authorization without persisting or leaking the token.
7. No legacy or alternate unprotected write route remains.
8. The agent uses the intended Unix socket with restrictive permissions.
9. Tool identifiers and managed paths cannot traverse outside their allowed root.
10. No affected flow uses arbitrary or user-influenced shell-string execution.
11. Linux runtime behavior and supported Windows observation compilation remain intact.
12. Tests cover safe defaults, denied paths, malicious input, allowed paths, and compatibility regressions.
13. `gofmt`, `make test`, and `make build` pass.
14. All applicable PR review threads are resolved based on verified fixes.

## Manual verification

At minimum, validate on Linux:

```bash
make run-api
ss -ltnp | grep ':8080'
ss -ltnp | grep ':9091' || true

curl -i http://127.0.0.1:8080/v1/read/health
curl -i http://127.0.0.1:8080/v1/read/version
curl -i http://127.0.0.1:8080/v1/read/port-events

curl -i -X POST http://127.0.0.1:8080/v1/write/tools/portainer/install
```

Repeat write-path verification with writes enabled, first without a token, then with an invalid token, and finally with the approved authenticated flow. Do not include real tokens in logs or committed files.

## Completion conditions

The task is complete when:

- Every Issue #20 acceptance criterion is verified.
- Required fixes and tests are committed to PR #18.
- `make test` and `make build` pass.
- Security and compatibility findings are documented.
- `STATUS.md`, `RISKS.md`, `DECISIONS.md`, and this handoff are updated where facts changed.
- PR #18 is ready to merge or clearly blocked with concrete reasons.
- No unrelated product scope is introduced.

## After PR #18

Proceed to Issue #21 for GitHub Actions CI and security quality gates. After CI is established, resolve D-012 through a dedicated architecture decision for durable inventory, snapshots, drift, and audit storage before implementing a database.
