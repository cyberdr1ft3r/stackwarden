# StackWarden Project Status

Last updated: 2026-09-04

## Current Phase

Security-baseline consolidation.

## Current `main`

Head inspected before the project-memory merge: `51144040ba1b2cb06d3c54cfa742c39819252669`.

Implemented on `main`:

- Go API, Go agent, shared Go package, static HTML UI.
- Root Go workspace and Makefile.
- API/agent health and version endpoints.
- Linux and Windows host metrics.
- Linux and Windows listening-port visibility.
- In-memory API audit log (recent events only).
- In-memory port snapshots and opened/closed/exposure event tracking.
- Shared tool catalog.
- Server-side tool installation staged under `/var/lib/stackwarden/tools/<id>/`.
- Tool status and uninstall support.
- Portainer CE and DDEV catalog entries.

## Active / Pending Work

PR #18: `Harden local-only baseline; split read/write API and lock down agent`

State as of 2026-09-04:

- Open and not merged.
- Mergeable when inspected.
- Base: `main`.
- Current head: `c5f58e03e29997b6ed3032a9284c762c23fea14e`.
- Follow-up fixes have been added since the initial security review, but the complete Issue #20 acceptance criteria still require final verification before merge.

PR #18 proposes:

- API loopback-only by default.
- Explicit override required for non-loopback bind.
- Unix socket for API-to-agent transport.
- `/v1/read/*` vs `/v1/write/*` separation.
- Writes disabled by default.
- Bearer token required when writes are enabled.
- Safer tool ID/path validation.
- Removal of remaining shell-string execution patterns in affected flows.
- UI route and write-token compatibility fixes.

Do not document these as current `main` behavior until PR #18 is fully validated and merged.

Open tracking issues:

- Issue #20: complete and validate the security baseline in PR #18. This is the next engineering task.
- Issue #21: add CI and security quality gates after Issue #20 / PR #18 is complete.

## Current Objective

1. Complete Issue #20 by validating and, where necessary, fixing PR #18 against the security model.
2. Merge PR #18 only after every security, compatibility, build, and test acceptance criterion is verified.
3. Add CI and security quality gates through Issue #21.
4. Move from transient port-event state toward the broader `discovery -> snapshot -> drift -> exposure -> policy` model.

## Blockers / Unresolved Items

- PR #18 still needs final Issue #20 validation before merge.
- Persistence architecture for snapshots/drift/audit is not yet decided.
- Authentication/authorization beyond the minimal proposed write token is not yet a settled long-term design.
- Deployment/service model (systemd units, package/install flow, upgrade strategy) needs a dedicated design/task.
- The boundary between supported remediation actions and observation-only features needs to be expanded deliberately, not ad hoc.

## Next Recommended Action

Execute Issue #20 against PR #18. Re-read the current PR diff and review state, verify the API bind behavior, read/write route compatibility, UI token flow, Unix-socket permissions, write-route coverage, tool/path validation, absence of arbitrary shell surfaces, Linux behavior, and supported Windows compilation. Run `make test` and `make build`, update the project memory with the verified result, and leave PR #18 either ready to merge or explicitly blocked.
