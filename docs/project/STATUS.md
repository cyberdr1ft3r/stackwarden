# StackWarden Project Status

Last updated: 2026-11-11

## Current Phase

Product-direction reconciliation. D-016 is proposed, not accepted or implemented.

## Current `main`

Current head: `1692b56893ea474163604a027cfa1e29c530b245` (`Resolve D-012 durable storage architecture (#24)`).

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
- Accepted D-012 durable-storage architecture documentation; no database implementation.

## Active / Pending Work

- Issue #23 remains open for implementation of D-012; its acceptance criteria remain binding unless explicitly revised.
- Issue #25 proposes D-016 and the application-centred roadmap. Documentation in its draft PR is proposed state only.
- Issue #26 is the first security correction: block unsafe new Portainer installations without changing existing deployments.
- Issue #27 is a bounded read-only Docker discovery slice and must remain draft pending product-direction review.

## Current Objective

1. Review proposed D-016 without treating it as accepted behavior.
2. Independently block unsafe new Portainer installations.
3. Develop read-only Docker discovery without ownership, adoption, or mutation.
4. Reconcile Issue #23 into reviewable storage/security/audit, desired-state persistence, and inventory-history slices without silently changing or closing it.

## Blockers / Unresolved Items

- Durable persistence is not implemented; current snapshot, port-event, and audit state is still volatile until Issue #23 is complete.
- Authentication/authorization beyond the merged minimal write token is not yet a settled long-term design.
- Deployment/service model (systemd units, package/install flow, upgrade strategy) needs a dedicated design/task.
- The boundary between supported remediation actions and observation-only features needs to be expanded deliberately, not ad hoc.
- Remote host enrollment/authentication and the eventual trigger for moving beyond single-instance SQLite remain deferred decisions.
- D-016 approval and the exact product boundary remain pending review.
- The bundled Portainer template currently publishes 9443/8000 and mounts the host Docker socket; new installation must be disabled separately without touching existing workloads.
- Installer Compose operations are coupled to HTTP request lifetime and buffer command output before trimming; correction requires a dedicated design beyond the scoped Portainer gate.
- Existing ownership of ports 80/443 must be detected before any future Caddy work; automatic replacement of another reverse proxy is prohibited by the proposal.

## Next Recommended Action

Complete Issue #26 as an independent security fix, then review Issue #27's read-only discovery draft. Do not begin Docker mutation or application reconciliation until D-016 is approved and the required storage/auth/audit foundation is reviewed.
