# StackWarden Vision

## Mission

StackWarden is a security-first control plane for understanding and safely operating servers.

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

The immediate priority is hardening the existing management plane before broadening its power. Open PR #18 proposes the current security-baseline direction but is not merged behavior.

After the security baseline, the project should design durable inventory/snapshot storage, then build a drift model, then exposure classification, and only then broader policy/remediation.

## Explicit non-goals for now

- Generic browser-based SSH/terminal access.
- Arbitrary command execution.
- Becoming a replacement for every monitoring/SIEM/orchestration product.
- Silent public exposure of management services.
- Large multi-tenant SaaS architecture without an approved project-direction change.
- Adding a database solely because future persistence is expected; persistence architecture must be decided first.

## Definition of success

A successful StackWarden should let an operator deploy a small trusted service, reach it safely, see what the host is exposing, notice important drift, understand why it matters, and perform only deliberate, reviewable, bounded actions.