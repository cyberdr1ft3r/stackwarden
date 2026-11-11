# StackWarden Roadmap

Last updated: 2026-11-11

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

## M1 - Security baseline (complete)

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

Issue #20 / PR #18 completed this milestone on `main`.

## Proposed application-centred sequence (pending D-016 approval)

This sequence is planning guidance from the 28 October 2026 baseline. It is not permission to implement later stages and does not make D-016 accepted.

### A - Product reconciliation

- Review proposed D-016 and its v1 exclusions.
- Preserve D-012 integrity and safety constraints until explicitly superseded.
- Measure the current and future footprint rather than assuming it remains lightweight.

### B - Immediate installer safety and structural preparation

- Block unsafe new Portainer installation at API/catalog and agent boundaries without changing existing deployments or volumes.
- Handle any no-behavior-change structural cleanup in separate reviewable PRs.
- Review DDEV installer supply-chain and privilege risk separately.

### C - Read-only Docker/application discovery

- Observe bounded container, service, image, status, and published-port facts through the local agent.
- Treat Compose labels as untrusted grouping hints; retain unattributed containers.
- Preserve unknown/unavailable states and existing host listeners.
- Do not adopt, deploy, mutate, or infer Internet reachability from a published port.

### D - Shared security, storage, and desired-state foundations

- Implement the common D-012 SQLite bootstrap, migrations, retention, backup/restore, corruption refusal, and administrative audit foundations.
- Design a single-administrator session with CSRF protection and explicit secret/AppSpec rejection policy.
- Decompose Issue #23 without closing or weakening it: (i) common storage/security/audit, (ii) application desired-state/reconciliation persistence, and (iii) inventory/snapshot history.

### E - Typed digest-based application reconciliation

- Introduce a narrow validated `AppSpec` for prebuilt immutable image digests, multi-service apps, databases, and persistent volumes.
- Add desired-state reconciliation only after independent API and agent safety enforcement is reviewed.
- Evaluate Caddy after detecting existing ownership of 80/443; never replace another proxy automatically.

### F - External source build/deploy

- Integrate GitHub/GitLab external CI and registries.
- Do not add local arbitrary source/image builds.

### G - Durable operations and release evidence

- Complete durable drift/exposure history and recovery workflows.
- Test clean installation, upgrade, rollback, backup/restore, and failure recovery.
- Record measured CPU, memory, disk, and idle/runtime footprint.

## Prior roadmap: durable inventory and snapshots

Goal: turn transient observation into durable host state.

Architecture: D-012 selects API-owned SQLite for an initial single-host store with host-scoped schema, deduplicated inventory revisions, durable drift/audit events, finite retention, and secure backup/recovery. Implementation is tracked by Issue #23.

Implementation scope:
- Host identity/model.
- Snapshot schema.
- Persistent port/service/tool inventory.
- Durable audit history.
- Retention policy.
- Single-host vs future multi-host assumptions.

Do not expand Issue #23 into multi-host enrollment, metrics time-series storage, PostgreSQL, or broader remediation.

## Prior M3 - Drift engine

Goal: compare current state with prior/approved state and produce meaningful changes.

Candidate changes:
- Listener added/removed.
- Bind/exposure changed.
- Service/tool state changed.
- Version/configuration facts changed where safely collectible.
- Severity/relevance metadata.
- Acknowledgement/expected-change model.

## Prior M4 - Exposure model

Goal: translate raw drift into security/operator relevance.

Candidate scope:
- Public vs local/private listeners.
- Unexpected management interfaces.
- Newly reachable service ports.
- Known risky/default exposure patterns.
- Contextual explanations and recommended verification steps.

Avoid claiming a vulnerability solely from port presence.

## Prior M5 - Policy and controlled remediation

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
- CI quality gates (Issue #21 / PR #22 complete and enforced); release/versioning automation remains separate.
- Secure upgrade/rollback.
- Documentation/runbooks.
- Threat-model updates and security tests.
