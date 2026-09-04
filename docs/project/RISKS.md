# StackWarden Risk Register

Last updated: 2026-09-04

## R-001 - Privileged agent compromise

Severity: Critical
Status: Active

The agent can perform host-level observations/actions. A remotely reachable or overly general agent interface could become a direct host-compromise path.

Controls/direction:
- Keep agent transport local/private.
- Never expose arbitrary command execution.
- Validate identifiers/paths.
- Prefer structured allowlisted actions.
- Restrict socket/service permissions.

## R-002 - Public management exposure

Severity: High
Status: Mitigated on current `main`; continuously regression-tested

An Internet-reachable management API increases attack surface substantially. Current `main` binds to loopback by default and requires explicit opt-in for non-loopback exposure.

Direction:
- Loopback/private by default.
- Explicit override for non-local binds.
- Use SSH tunnel/private network/secured reverse proxy for remote use.

## R-003 - Unauthenticated or insufficiently gated writes

Severity: Critical
Status: Mitigated on current `main`; continuously regression-tested

Tool installation/uninstallation changes host state and may invoke privileged package/container operations. Write capabilities require explicit gating and authentication/authorization.

Current `main` keeps writes disabled by default and applies centralized Bearer authorization to every supported `/v1/write/*` action. Route-level tests cover disabled, missing-token, invalid-token, allowed, malicious-input, and legacy-path cases.

## R-004 - Shell injection / unsafe execution composition

Severity: Critical
Status: Must remain continuously reviewed

Any free-form shell composition influenced by API/user input can become command execution. Avoid `sh -c` patterns and pass validated argv directly wherever possible.

Current `main` install/status/uninstall flows use structured argv. This remains a continuous review requirement for catalog additions.

## R-005 - Path traversal / managed-directory escape

Severity: High
Status: Must remain continuously reviewed

Tool IDs or future resource identifiers must not escape `/var/lib/stackwarden/...` or other managed roots. Validate allowed characters and verify resolved paths remain under expected bases.

Current `main` validates API, agent, and embedded-template tool IDs; checks lexical containment; rejects symlinks at managed tool directories, nested staged-file paths, and Compose-file paths; and includes traversal/symlink regression tests. Managed-path behavior must remain under review as new artifact types are added.

## R-006 - Volatile operational history

Severity: Medium
Status: Active; architecture accepted, implementation pending Issue #23

Audit events and port-change snapshots are currently in memory. Restarting the API loses history, limiting forensic usefulness and durable drift analysis.

Mitigation is defined by D-012; volatility remains until Issue #23 is implemented.

## R-007 - Security model drift as features expand

Severity: High
Status: Active

Adding convenient remediation/tool features can gradually turn StackWarden into a generic remote administration shell.

Controls:
- Route meaningful changes through issues/PRs.
- Update decisions/threat model.
- Require narrow action contracts and tests.

## R-008 - Tool installer supply-chain/privilege risk

Severity: High
Status: Active

Installing third-party tools may add repositories, fetch packages/images, and run privileged installers. Catalog entries need reviewable sources, bounded commands, clear platform assumptions, and safe failure handling.

## R-009 - Incomplete long-term auth model

Severity: High
Status: Open

Current `main` uses a minimal Bearer token for enabled writes, but long-term identity, roles, session handling, token lifecycle, and multi-operator authorization are not yet decided.

## R-010 - Deployment hardening not codified

Severity: Medium
Status: Open

Systemd/service users, filesystem ownership, socket group ownership, upgrade/rollback, log retention, and reverse-proxy deployment are not yet fully codified as a reproducible production model.

## R-011 - Failed teardown loses recovery configuration

Severity: High
Status: Mitigated on current `main`

Removing a staged tool directory after a failed Compose or package uninstall can leave host resources active while deleting the configuration needed to retry or diagnose teardown.

Current `main` retains the staged directory whenever uninstall does not complete and tests the failed-Compose recovery path. Future uninstall actions must preserve this safe failure mode.

## R-012 - CI checks exist but are not enforced

Severity: High
Status: Mitigated on current `main`

The `Protect main` ruleset requires all six CI checks. Workflow and ruleset changes remain security-sensitive repository administration.

## R-013 - Unsupported Go toolchain vulnerabilities

Severity: High
Status: Mitigated on current `main`

Go 1.22 is end-of-life and current vulnerability data identifies reachable standard-library vulnerabilities in the prior 1.22.2 development toolchain. StackWarden now requires and CI pins Go 1.25.13, and the vulnerability gate fails when reachable findings are present.

## R-014 - Durable inventory disclosure or tampering

Severity: High
Status: Design mitigation accepted; implementation pending Issue #23

Listener, service, tool, drift, and administrative history can reveal host posture or mislead operators if disclosed or modified.

D-012 minimizes collected fields, prohibits secrets/raw output, isolates the database under an API-owned `0700` directory with `0600` files, and requires validated paths, corruption refusal, and equally protected backups. Host/disk compromise remains outside application-level protection; encrypted storage is an operator responsibility.

## R-015 - Database growth, corruption, or unrecoverable migration

Severity: High
Status: Design mitigation accepted; implementation pending Issue #23

Unbounded history can exhaust disk, while an unsafe migration, WAL copy, or automatic recreation can destroy forensic state.

D-012 defines finite retention, bounded pruning, transactional checksum-verified migrations, online backup, offline restore validation, clean shutdown, and fail-closed corruption handling. These controls require fault-injection and recovery tests before persistence is considered complete.