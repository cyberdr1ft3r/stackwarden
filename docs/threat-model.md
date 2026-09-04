# StackWarden Threat Model

Last updated: 2026-08-08

## Scope

StackWarden is a management plane with a privileged host agent. The main security objective is to prevent a visibility/remediation tool from becoming an easier path to host compromise.

## Assets

- Managed host integrity and availability.
- Root/privileged execution capability of the agent.
- Host inventory, ports, processes, metrics, and operational state.
- Credentials/tokens used to authorize management actions.
- Audit and drift history.
- Third-party tool/package installation trust.

## Trust boundaries

```text
Untrusted/less-trusted browser/network
            |
            v
      [ API boundary ]
            |
            v
     local agent channel
            |
            v
     [ Agent boundary ]
            |
            v
       Managed host
```

### Browser to API

Treat browser input as untrusted. Client-side validation is convenience only. Authorization and validation belong on the API.

### API to agent

This is a high-value boundary. The agent should be reachable only through a local/private mechanism and should expose narrowly defined operations rather than a generic command interface.

### Agent to OS / third parties

Commands, filesystem writes, package managers, Docker/Compose, repositories, and downloaded artifacts cross into privileged or external trust domains. Inputs must be explicit and bounded.

## Primary threats

### T-001 - Remote takeover through exposed management endpoints

An attacker reaches the API or agent from an untrusted network and invokes privileged actions.

Required controls:
- Local/private bind defaults.
- No public agent listener by default.
- Explicit secure remote-access pattern.
- Authentication/authorization for writes.
- Deny/disabled-by-default mutations.

### T-002 - Arbitrary command execution

A tool ID, option, path, or other input is composed into a shell command and results in arbitrary host execution.

Required controls:
- No generic command endpoint.
- Avoid shell-string execution.
- Structured argv.
- Allowlisted/catalog-defined operations.
- Strict identifier/input validation.

### T-003 - Path traversal / filesystem escape

An attacker controls an identifier or path that causes writes/deletes outside StackWarden-managed directories.

Required controls:
- Restricted identifier character sets.
- Clean/resolve paths.
- Verify target remains under expected base.
- Safe file/directory permissions.

### T-004 - Authentication bypass on alternate routes

A new protected write route is added while an old/legacy route remains reachable without the new middleware.

Required controls:
- Centralized route/middleware design.
- Negative tests for legacy/alternate paths.
- Route inventory review whenever namespaces change.

### T-005 - Token/secret leakage

Management credentials appear in logs, audit events, command output, Git, process arguments, or UI errors.

Required controls:
- Never commit secrets.
- Avoid logging auth headers/tokens.
- Redact sensitive outputs.
- Prefer secure environment/file handling over CLI arguments when feasible.

### T-006 - Privilege escalation through installer behavior

A catalog installer invokes package managers, scripts, Docker images, or repositories that have broader effects than intended.

Required controls:
- Explicit sources and commands.
- Platform checks.
- Reviewed catalog metadata.
- Timeouts/bounded output.
- Avoid user-controlled repository/script URLs.

### T-007 - Unsafe uninstall/destructive action

An uninstall path deletes operator data or unrelated services.

Required controls:
- Scope uninstall to assets StackWarden created/manages.
- Separate application removal from destructive data deletion when relevant.
- Make destructive behavior explicit and test it.

### T-008 - Lost forensic/drift history

In-memory audit/snapshot history disappears on restart, weakening security analysis.

Required control direction:
- Durable state after persistence architecture is explicitly decided.

### T-009 - CSRF/browser-origin abuse once writes exist

If browser sessions/cookies are introduced later, an attacker may trigger management actions through a victim browser.

Required future controls:
- Appropriate SameSite/CSRF/origin protections based on the eventual auth model.
- Do not assume Bearer-token and cookie-session threat models are interchangeable.

### T-010 - Multi-user privilege confusion

If StackWarden grows beyond a single trusted operator, insufficient role/action separation could expose high-risk operations.

Required future control:
- Explicit identity/role/permission architecture before multi-user management is claimed supported.

## Security invariants

These should be treated as regression-sensitive:

- UI never directly controls the OS.
- Agent is not a generic remote shell.
- Writes are more restricted than reads.
- Public management exposure is never the default.
- Input-controlled command strings are prohibited.
- Managed filesystem paths cannot escape their base.
- Secrets do not enter repository history or audit output.
- Open PR behavior is not assumed deployed/merged.

## Review requirements for privileged changes

Any PR touching agent actions, installers, networking, auth, paths, or write APIs should document:

- Threat/boundary affected.
- Safe default.
- Denied/malicious cases tested.
- Manual verification if OS behavior is environment-specific.
- New/changed risks in `docs/project/RISKS.md`.

## Current security gap summary

Current `main` predates the full hardened boundary. PR #18 proposes major mitigations including loopback API default, Unix-socket agent transport, read/write route separation, write-disabled-by-default behavior, write Bearer token checks, path validation, and safer command execution. Until merged, those are proposals rather than current guarantees.