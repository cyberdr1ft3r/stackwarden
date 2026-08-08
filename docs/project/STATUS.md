# StackWarden Project Status

Last updated: 2026-08-08

## Current Phase

Security-baseline consolidation.

## Current `main`

Head inspected: `51144040ba1b2cb06d3c54cfa742c39819252669`.

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

State as of 2026-08-08:

- Open.
- Not merged.
- Mergeable when inspected.
- Base: `main` at `51144040ba1b2cb06d3c54cfa742c39819252669`.
- Head: `9f7362968d19057416b3eb4096dba857ac311869`.

PR #18 proposes:

- API loopback-only by default.
- Explicit override required for non-loopback bind.
- Unix socket for API-to-agent transport.
- `/v1/read/*` vs `/v1/write/*` separation.
- Writes disabled by default.
- Bearer token required when writes are enabled.
- Safer tool ID/path validation.
- Removal of remaining shell-string execution patterns in affected flows.

Do not document these as current `main` behavior until the PR is reviewed and merged.

## Current Objective

1. Establish durable project memory and Codex operating rules in Git.
2. Review PR #18 against the security model and current project direction.
3. Merge/fix the security baseline before adding broader privileged remediation features.
4. Move from transient port-event state toward the broader `discovery -> snapshot -> drift -> exposure -> policy` model.

## Blockers / Unresolved Items

- PR #18 needs final review against project security rules before merge.
- Persistence architecture for snapshots/drift/audit is not yet decided.
- Authentication/authorization beyond the minimal proposed write token is not yet a settled long-term design.
- Deployment/service model (systemd units, package/install flow, upgrade strategy) needs a dedicated design/task.
- The boundary between supported remediation actions and observation-only features needs to be expanded deliberately, not ad hoc.

## Next Recommended Action

Review PR #18 as the next engineering task. Verify its diff, tests, route compatibility, Unix-socket permissions, API bind protections, tool/path validation, and absence of arbitrary shell surfaces. Update project memory with the review result before expanding scope.