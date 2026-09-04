# StackWarden Decisions

Last updated: 2026-09-04

Accepted decisions are append-oriented. Do not rewrite history to make a later decision look original; supersede older decisions explicitly.

## D-001 - Security-first control plane

Status: Accepted

StackWarden is a security-first server visibility and operations control plane, not a generic server dashboard.

The product progression is:

`discovery -> snapshot -> drift -> exposure -> policy`

## D-002 - UI has no direct host authority

Status: Accepted

The browser/UI is presentation and interaction only. Privileged OS operations must not be implemented in the browser or by bypassing the API/agent boundary.

## D-003 - API is the policy boundary

Status: Accepted

The API is responsible for externally facing policy enforcement, validation, authorization/authentication as applicable, audit recording, and mediation of agent actions.

## D-004 - Agent is the privileged execution boundary

Status: Accepted

The agent is the component permitted to perform privileged host inspection/actions. Because its compromise has high impact, its interface must remain narrow and local/private.

## D-005 - No arbitrary remote shell surface

Status: Accepted

StackWarden must not expose arbitrary shell/command execution through the management plane. Host mutations should use structured argv and allowlisted/catalog-defined operations with validated inputs.

## D-006 - Read before write

Status: Accepted

Observation/read functionality is the foundation. Write/remediation capabilities must be treated as higher risk, deliberately enabled, validated, audited, and bounded.

## D-007 - Private/local management by default

Status: Accepted

Management interfaces should be local/private by default. Remote use should prefer SSH tunnels, private networking, or an explicitly secured reverse proxy rather than simply opening management ports to the Internet.

## D-008 - Linux primary; preserve Windows visibility

Status: Accepted

Linux is the primary managed-server target. Existing Windows metrics/port observation is supported behavior and should not be broken without an approved decision.

## D-009 - Shared catalog controls supported tool actions

Status: Accepted

Supported third-party tool operations are described through shared catalog metadata. Tool-specific install/status/uninstall behavior must stay explicit and reviewable; new tools do not become an excuse for general command execution.

## D-010 - Project memory lives in Git

Status: Accepted

Project objectives, current state, decisions, risks, roadmap, and next-agent handoff are maintained in repository files. Meaningful Codex tasks must read and update them when state changes.

## D-011 - Open PRs are not source-of-truth behavior

Status: Accepted

Open PRs may describe intended direction but are not treated as current product behavior until merged. Status/memory documents must preserve this distinction.

## D-012 - Durable storage architecture

Status: Accepted on 2026-09-04; implementation tracked by Issue #23

The initial durable store is API-owned SQLite for one local managed host. Schema and repository APIs are host-scoped from version 1 to preserve a migration path to multi-host operation.

Complete snapshots reference deduplicated normalized inventory revisions containing listener, minimally described service, and catalog-tool observations. Drift and structured administrative audit events are durable; live metrics, PIDs, raw command output, secrets, environment contents, and unnecessary host fingerprints remain ephemeral.

SQLite uses embedded forward migrations, WAL with full synchronous durability, serialized writes, bounded retention, secure local filesystem permissions, verified backup/restore, corruption refusal, and no silent in-memory fallback. The agent and UI never access the database directly.

The complete schema, lifecycle, security controls, tradeoffs, and implementation acceptance criteria are in [`docs/architecture/durable-storage.md`](../architecture/durable-storage.md).

## D-013 - Security-baseline interface

Status: Accepted and merged in PR #18

The API binds to loopback by default, and a non-loopback bind requires explicit opt-in. API-to-agent traffic uses a local Unix socket with restrictive directory/socket permissions.

Read operations use `/v1/read/*`. State-changing operations use `/v1/write/*`, remain disabled by default, and require server-side Bearer authorization when enabled. The static UI may hold the write token only in page memory and must not persist or log it.

Legacy compatibility may expose read-only aliases, but no legacy or alternate write endpoint may bypass the centralized write gate.

## D-014 - Required CI quality gates

Status: Accepted and merged by Issue #21 / PR #22

Pull requests and pushes to `main` must run stable, separately named checks for Go formatting, `go vet` across every maintained module, tests, Linux builds, compile-only Windows compatibility, and `govulncheck`.

Workflows use read-only repository permissions, require no production secrets, avoid privileged/deployment behavior, and pin trusted actions to immutable commits. Branch protection must require the stable checks after the workflow is merged.

## D-015 - Patched Go toolchain baseline

Status: Accepted and merged by Issue #21 / PR #22

StackWarden requires Go 1.25.13 or newer. CI pins Go 1.25.13 so formatting, analysis, tests, builds, Windows compilation, and vulnerability results are reproducible.

Go 1.22 is no longer a supported build baseline because current `govulncheck` identifies reachable standard-library vulnerabilities that are fixed only in maintained Go releases.