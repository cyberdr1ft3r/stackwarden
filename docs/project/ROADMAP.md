# StackWarden Roadmap

Last updated: 2026-08-08

This roadmap is sequencing guidance, not permission to silently implement future scope.

## M0 - Foundation (largely complete)

Goal: clone, run, observe basic host state, and exercise the API/agent/UI architecture.

Delivered on `main`:
- Go API + Go agent + static UI.
- Go workspace/Makefile.
- Health/version.
- Ports and metrics.
- Audit events.
- Port change events.
- Shared tool catalog and initial tool operations.

## M1 - Security baseline (current)

Goal: make the existing control plane safe enough to build on.

Required outcomes:
- Local/private API default.
- Agent unavailable as a general TCP management endpoint.
- Explicit read/write separation.
- Writes disabled/denied by default.
- Authentication for enabled write actions.
- No arbitrary shell execution surface.
- Safe tool identifier/path handling.
- Tests for security boundaries.
- Deployment access guidance that does not recommend exposing management ports publicly.

PR #18 currently attempts this milestone and requires review before merge.

## M2 - Durable inventory and snapshots

Goal: turn transient observation into durable host state.

Candidate scope (requires architecture decision first):
- Host identity/model.
- Snapshot schema.
- Persistent port/service/tool inventory.
- Durable audit history.
- Retention policy.
- Single-host vs future multi-host assumptions.

Do not select a database/storage engine until D-012 is resolved.

## M3 - Drift engine

Goal: compare current state with prior/approved state and produce meaningful changes.

Candidate changes:
- Listener added/removed.
- Bind/exposure changed.
- Service/tool state changed.
- Version/configuration facts changed where safely collectible.
- Severity/relevance metadata.
- Acknowledgement/expected-change model.

## M4 - Exposure model

Goal: translate raw drift into security/operator relevance.

Candidate scope:
- Public vs local/private listeners.
- Unexpected management interfaces.
- Newly reachable service ports.
- Known risky/default exposure patterns.
- Contextual explanations and recommended verification steps.

Avoid claiming a vulnerability solely from port presence.

## M5 - Policy and controlled remediation

Goal: encode approved desired state and allow safe corrective operations.

Principles:
- Observation remains available independently from remediation.
- Policy is explicit and reviewable.
- Actions are narrow, validated, auditable, and reversible where practical.
- No generic remote shell.
- Dry-run/preview where useful.
- Permission model appropriate to operation risk.

## Cross-cutting work

These can be scheduled as dedicated issues when needed:
- Production installation/systemd packaging.
- Auth/session/role architecture beyond the minimal security baseline.
- Logging and durable audit export.
- CI/release/versioning pipeline.
- Secure upgrade/rollback.
- Documentation/runbooks.
- Threat-model updates and security tests.