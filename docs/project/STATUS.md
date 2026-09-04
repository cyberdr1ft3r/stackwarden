# StackWarden Project Status

Last updated: 2026-09-04

## Current Phase

M2 durable inventory and snapshot architecture.

## Current `main`

Current head: `abf0348b0208aa10daf7bf6b031cae80561312ba` (`Add enforceable Go CI quality gates (#22)`).

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
- Loopback-only API default with explicit non-loopback opt-in.
- Unix-socket API-to-agent transport with restrictive permissions.
- Versioned read/write route separation.
- Writes disabled by default and Bearer authorization for enabled writes.
- Validated managed tool/template paths and structured command execution.
- Six mandatory CI checks for formatting, static analysis, tests, Linux builds, Windows compilation, and vulnerability scanning.

## Active / Pending Work

- Issue #20 and PR #18 are complete and merged on `main`.
- Issue #21 and PR #22 are complete and merged; the `Protect main` ruleset requires all six CI checks.
- D-012 now selects API-owned SQLite and a host-scoped schema for the first single-host durable store.
- Issue #23 tracks implementation of the accepted durable-storage architecture after the D-012 architecture PR merges.

## Current Objective

1. Review and merge the D-012 architecture without adding database code.
2. Implement the accepted architecture through Issue #23.
3. Build the drift and exposure models on durable snapshots.

## Blockers / Unresolved Items

- Durable persistence is not implemented; current snapshot, port-event, and audit state is still volatile until Issue #23 is complete.
- Authentication/authorization beyond the merged minimal write token is not yet a settled long-term design.
- Deployment/service model (systemd units, package/install flow, upgrade strategy) needs a dedicated design/task.
- The boundary between supported remediation actions and observation-only features needs to be expanded deliberately, not ad hoc.
- Remote host enrollment/authentication and the eventual trigger for moving beyond single-instance SQLite remain deferred decisions.

## Next Recommended Action

After the D-012 architecture PR is approved and merged, implement Issue #23 exactly against `docs/architecture/durable-storage.md`, beginning with the API storage interface, secure SQLite bootstrap, and schema migrations.
