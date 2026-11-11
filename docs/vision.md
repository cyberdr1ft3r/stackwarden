# StackWarden Vision

## Mission

StackWarden is a security-first control plane for understanding and safely operating servers.

The 28 October 2026 planning baseline proposes focusing that mission on a lightweight, single-server, application-centred Docker hosting panel. That direction is recorded as proposed D-016 and is not merged functionality or an accepted architecture yet.

It should help a sysadmin answer five progressively harder questions:

1. **Discovery** — What is running and listening on this host?
2. **Snapshot** — What did the host look like at a known point in time?
3. **Drift** — What changed?
4. **Exposure** — Which changes affect security or reachability?
5. **Policy** — What state should be allowed, and what safe action should happen when reality diverges?

The goal is not to reproduce a full enterprise monitoring suite or remote-shell product. The goal is a small, understandable, self-hosted operational security layer that combines visibility, change awareness, and tightly controlled remediation.

## Target user

Primary user: a sysadmin / infrastructure operator with blue-team responsibilities managing Linux servers and needing a quick, trustworthy view of host exposure and operational state.

Windows observation is supported where already implemented, but Linux remains the primary managed-server target.

Under proposed D-016, applications and websites become the home screen. The operator would see actual containers and services, health, domains, published-port facts, cautiously described exposure, drift, and audit without StackWarden claiming ownership of workloads it merely observes.

## Product principles

### Security before convenience

A management product that makes the host easier to compromise has failed. Safe network defaults, narrow privilege boundaries, validated operations, and explicit writes matter more than one-click convenience.

### Visibility before remediation

The operator should be able to observe current state without enabling mutation. Remediation features should be added only after the relevant state is visible and understandable.

### Small trusted computing base

Keep the architecture simple:

```text
Browser UI
   |
   v
Go API (policy boundary)
   |
   v
Go Agent (privileged boundary)
   |
   v
Host OS / services / sockets / tools
```

The browser has no direct host authority. The API decides what is allowed. The agent performs narrowly defined host operations.

### Drift is more useful than raw inventory alone

A port list is useful; knowing that port 8443 became publicly bound ten minutes ago is more useful. StackWarden should progressively turn raw host facts into change and exposure information.

### Explain rather than overclaim

A listening port is not automatically a vulnerability. Exposure findings should explain what changed, why it may matter, and what an operator should verify.

### Git-backed project discipline

Architecture, decisions, risks, status, and handoffs live in the repository so humans and coding agents can resume work without relying on conversation memory.

## Current capabilities

On current `main`, StackWarden provides:

- API and agent health/version information.
- Host CPU/memory/disk/uptime metrics.
- Listening-port discovery.
- Port-change events including exposure changes.
- Recent in-memory audit events.
- A shared tool catalog.
- Server-side install/status/uninstall flows for supported tools such as Portainer CE and DDEV.
- A minimal static dashboard.

These are foundations, not the finished product.

## Near-term direction

The management-plane security baseline, mandatory CI quality gates, and D-012 durable-storage architecture are merged. Durable storage itself is not implemented.

Proposed D-016 changes the intended sequence, subject to review: first reconcile product direction, then correct unsafe Portainer installation defaults, add bounded read-only Docker application discovery, establish shared secure storage/auth/audit foundations, and only then add typed digest-based application reconciliation and proxy integration. Deployment, ownership adoption, and Docker mutation are not implied by discovery.

The proposed deployment model starts with prebuilt immutable image digests. GitHub/GitLab source deployments would use external CI and registries; StackWarden would not build arbitrary source locally. A typed `AppSpec` would be validated at both API and agent boundaries, and arbitrary Compose would not be accepted as user input.

## Explicit non-goals for now

- Generic browser-based SSH/terminal access.
- Arbitrary command execution.
- Becoming a replacement for every monitoring/SIEM/orchestration product.
- Silent public exposure of management services.
- Large multi-tenant SaaS architecture without an approved project-direction change.
- Adding a database solely because future persistence is expected; persistence architecture must be decided first.
- Mail or DNS hosting, reseller/billing features, Kubernetes/multi-host orchestration, local image builds, and adoption of unmanaged workloads in the proposed v1.

## Definition of success

A successful StackWarden should let an operator deploy a small trusted service, reach it safely, see what the host is exposing, notice important drift, understand why it matters, and perform only deliberate, reviewable, bounded actions.
