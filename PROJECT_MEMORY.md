# StackWarden - Project Memory

Last updated: 2026-09-04

This is the fastest context-rehydration entry point for humans and coding agents. It records stable product facts, architecture, current direction, and the operating protocol. Detailed design/security documentation lives under `docs/`.

## Product Purpose

Build a security-first server visibility and operations control plane for sysadmins and blue-team workflows.

StackWarden should help an operator understand and progressively control a host through the sequence:

`discovery -> snapshot -> drift -> exposure -> policy`

The project is intentionally more than a generic dashboard. Its value is safe operational visibility, exposure awareness, controlled remediation, and reusable defensive workflows.

## Core Product Principles

- Start from visibility before mutation.
- Make dangerous capabilities explicit, narrow, and auditable.
- Keep management surfaces private/local by default.
- Separate presentation, policy, and privileged execution.
- Prefer structured/allowlisted host operations over arbitrary shell execution.
- Treat unexpected exposure or configuration drift as security-relevant events.
- Keep the system simple enough to self-host and reason about.
- Build reusable operational notes/runbooks alongside functionality.

## Current Phase

CI/security quality gates after completion of the security baseline, before durable inventory/snapshot architecture.

Current `main` already includes:

- Go API, Go agent, and framework-free HTML UI.
- Go workspace and root Makefile.
- API/agent health checks and version/about information.
- Cross-platform host metrics collection for Linux and Windows.
- Listening-port discovery and structured port visibility.
- In-memory audit history capped at recent events.
- Port snapshot comparison with opened/closed/exposure-change events.
- Shared tool catalog.
- Server-side tool install flow staged under `/var/lib/stackwarden/tools/<id>/`.
- Tool status and uninstall operations for supported catalog entries.
- Portainer CE and DDEV catalog entries.
- Loopback-only API binding by default, with explicit opt-in for non-loopback exposure.
- Unix-socket API-to-agent transport with restrictive permissions.
- `/v1/read/*` and `/v1/write/*` route separation.
- Writes disabled by default and Bearer authorization for enabled writes.
- Validated managed tool identifiers/paths and structured command execution.

## Architecture

```text
[ Browser / UI ]
        |
        v
[ Go API / policy boundary ]
        |
        v
[ Go Agent / privileged boundary ]
        |
        v
[ Host OS, sockets, metrics, services, tools ]
```

Stable responsibility split:

- UI: display state and request approved operations; no direct host privilege.
- API: externally reachable application boundary, validation, policy, auth, audit, orchestration/proxying.
- Agent: local privileged host inspection and narrowly defined host changes.
- `pkg`: shared Go types/catalog logic that should remain independent from UI concerns.

API-to-agent communication stays local over a Unix socket and is inaccessible as a general TCP management endpoint.

## Repository Structure

- `agent/` - host inspection and privileged execution boundary.
- `api/` - HTTP API, UI serving, audit/event state, agent mediation.
- `pkg/` - shared Go types and tool catalog.
- `ui/` - static browser UI.
- `install/` - installation/deployment-related assets.
- `docs/` - product, architecture, security, and operational source of truth.
- `.agents/skills/` - repository-specific Codex operating skills.
- `AGENTS.md` - mandatory Codex entrypoint and engineering/security rules.
- `docs/project/` - current status, decisions, risks, roadmap, and next-agent handoff.

## Product Direction

### 1. Discovery

Inventory what is actually present and reachable on a managed host: listeners, processes/services, system health, installed/managed tools, and other security-relevant host facts.

### 2. Snapshot

Persist a trustworthy representation of host state so current state can be compared with a known previous/approved state. Current port-change tracking is an in-memory precursor, not the final persistence model.

### 3. Drift

Detect meaningful change between snapshots: new/removed listeners, bind-address changes, service/tool changes, configuration changes, or other changes that affect operational/security posture.

### 4. Exposure

Turn raw changes into operator-relevant exposure information: public vs local binds, newly reachable services, risky management interfaces, unexpected service ports, and similar findings.

### 5. Policy / Remediation

Allow explicit policy and safe corrective actions only after observation and risk are understood. Remediation should be narrow, validated, auditable, reversible where practical, and disabled/denied unless deliberately enabled.

## Supported / Intended Environments

- Linux is the primary managed-server target.
- Windows host observation exists and should not be broken casually.
- Self-hosted operation is a primary use case.
- Remote administration should prefer private access, SSH tunnels, or a properly secured reverse proxy rather than directly exposing management ports.

## Security Model

The agent may require privileges that make compromise high impact. Therefore:

- The browser must never talk directly to the agent.
- The agent must not become a generic remote command endpoint.
- API write operations need explicit security controls and auditability.
- Inputs that influence commands, paths, services, or tools must be tightly validated.
- Secrets and host-specific sensitive state must remain outside Git.
- New features should preserve least privilege and safe defaults even if this adds implementation work.

See `docs/threat-model.md` for the detailed threat model.

## Current Technical Direction

- Go for API/agent/shared packages.
- Framework-free static HTML/JS UI unless an approved decision changes this.
- Small self-contained deployment footprint.
- Shared catalog metadata drives supported tool operations.
- Root `Makefile` provides common run/test/build workflows.
- Tests live inside each Go module and should protect parsers, security boundaries, and privileged-action behavior.
- Persistent state is not yet a settled architecture; do not introduce a database silently.

## Non-Negotiable Engineering Rules

- Never commit secrets, tokens, credentials, production host inventories, or sensitive logs.
- Never add arbitrary command/shell execution reachable through the management plane.
- Do not expose the API/agent publicly by default.
- Do not silently expand tool/remediation capabilities.
- Treat write operations as higher risk than read operations.
- Work through scoped branches/issues and draft PRs.
- Update project memory and handoff files after meaningful work.
- Run applicable tests before declaring completion.

## Source-of-Truth Order

When information conflicts:

1. Current approved GitHub issue and review comments.
2. Merged code and detailed docs under `docs/`.
3. `docs/project/DECISIONS.md`.
4. `PROJECT_MEMORY.md` and `docs/project/STATUS.md`.
5. Older PR descriptions, commits, and conversation context.

An open PR is proposed behavior, not current behavior.

## Memory Update Protocol

After every meaningful task:

1. Update `docs/project/STATUS.md`.
2. Replace `docs/project/HANDOFF.md` with the next concrete action.
3. Add accepted architecture/product/security decisions to `docs/project/DECISIONS.md`.
4. Update `docs/project/RISKS.md` as risks change.
5. Update `docs/project/ROADMAP.md` when sequencing changes.
6. Update this file only when stable facts/goals/rules change.
7. Reference the relevant issue/PR where possible.

Do not turn project-memory files into activity diaries.