# StackWarden Project Status

Last updated: 2026-09-04

## Current Phase

CI and security quality gates.

## Current `main`

Current head: `005aa04d4755ab76a4c0ae1ecd963469e5235105` (`Complete StackWarden security baseline (#18)`).

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

## Active / Pending Work

- Issue #20 and PR #18 are complete and merged on `main`.
- Issue #21 is implementing pull-request and `main`-push checks for formatting, vet, tests, Linux builds, Windows compilation, and vulnerability analysis.
- Issue #21 raises the supported build baseline from end-of-life Go 1.22 to patched Go 1.25.13 because current vulnerability analysis finds reachable standard-library vulnerabilities under the old toolchain.
- Branch protection still requires maintainer configuration after stable check names exist.

## Current Objective

1. Complete Issue #21 and require its stable checks through branch protection.
2. Resolve D-012 through a dedicated durable inventory/snapshot architecture decision.
3. Move from transient port-event state toward the broader `discovery -> snapshot -> drift -> exposure -> policy` model.

## Blockers / Unresolved Items

- CI is not enforceable until the Issue #21 workflow is merged and its checks are selected in a `main` branch protection rule/ruleset.
- Repository branch-protection and Actions settings were not readable by the automation integration; a maintainer must verify them in GitHub.
- Persistence architecture for snapshots/drift/audit is not yet decided.
- Authentication/authorization beyond the merged minimal write token is not yet a settled long-term design.
- Deployment/service model (systemd units, package/install flow, upgrade strategy) needs a dedicated design/task.
- The boundary between supported remediation actions and observation-only features needs to be expanded deliberately, not ad hoc.

## Next Recommended Action

Complete and merge Issue #21 after all GitHub Actions checks pass, then configure those checks as required for `main`. Next, resolve D-012 in a dedicated architecture issue before implementing durable inventory or a database.
