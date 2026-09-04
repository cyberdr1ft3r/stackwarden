# StackWarden Project Status

Last updated: 2026-09-04

## Current Phase

Security-baseline consolidation.

## Current `main`

Current head after the project-memory merge from PR #19: `fb2c293`.

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
- Latest `main`, including PR #19, is merged into the branch.
- Issue #20 implementation and verification are complete on the branch.
- `make test`, `make build`, focused security tests, Linux manual API checks, and Windows cross-compilation pass.
- The environment did not provide `ss`; `netstat` confirmed the API on `127.0.0.1:8080` with no `:9091` agent TCP listener.
- PR #18 is ready for final maintainer review but must not be treated as `main` behavior until merged.

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

- Issue #20: implementation and local verification are complete in PR #18; close when the PR is merged.
- Issue #21: add CI and security quality gates after PR #18 is merged.

## Current Objective

1. Complete final review and maintainer merge of PR #18 without broadening its scope.
2. Add CI and security quality gates through Issue #21.
3. Move from transient port-event state toward the broader `discovery -> snapshot -> drift -> exposure -> policy` model.

## Blockers / Unresolved Items

- PR #18 remains unmerged and needs final maintainer review.
- No automated PR checks are currently reported; Issue #21 is intended to add CI/security gates.
- Persistence architecture for snapshots/drift/audit is not yet decided.
- Authentication/authorization beyond the minimal proposed write token is not yet a settled long-term design.
- Deployment/service model (systemd units, package/install flow, upgrade strategy) needs a dedicated design/task.
- The boundary between supported remediation actions and observation-only features needs to be expanded deliberately, not ad hoc.

## Next Recommended Action

Perform final maintainer review of PR #18 and merge it if approved. Then execute Issue #21 to add automated CI and security quality gates for the verified baseline.
