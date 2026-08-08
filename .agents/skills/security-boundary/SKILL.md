# StackWarden Security Boundary Skill

Use this skill whenever a task touches network exposure, API routes, authentication/authorization, the agent, filesystem paths, commands, installers, services, ports, or privileged host actions.

## Core boundary

```text
Browser/UI -> API/policy boundary -> Agent/privileged boundary -> Host OS
```

Never bypass this boundary for convenience.

## Review checklist

### Network exposure
- What address/socket is bound?
- Is the safe default local/private?
- Is non-local exposure explicit and documented?
- Can the agent be reached directly from an untrusted network?

### API behavior
- Is the operation read-only or state-changing?
- Are writes disabled/denied unless explicitly enabled?
- Is authorization enforced server-side on every write path, including legacy/alternate routes?
- Is the action audited without leaking secrets?

### Agent behavior
- Is the action narrowly defined?
- Are all user-influenced identifiers validated?
- Is command execution structured argv rather than shell-string composition?
- Are timeouts/output bounds applied where external commands can hang or flood output?

### Filesystem
- Can a supplied ID/path escape a managed base directory?
- Are directory/file/socket permissions least-privilege?
- Are temporary/staged files created safely?

### Third-party tooling
- Is the source/repository/image explicit?
- Are package or compose operations catalog-defined?
- Are platform assumptions checked before mutation?
- Does uninstall avoid deleting unrelated operator data?

## Required testing

For changed security boundaries add tests for:
- safe default
- denied path
- malformed/malicious input
- explicit allowed path
- compatibility/regression where relevant

Run `make test` for meaningful changes and `make build` when runtime/build wiring changes.

## Prohibited shortcuts

- Arbitrary remote shell APIs
- `sh -c`/equivalent built from untrusted or weakly validated input
- Public management binds merely for convenience
- Client-side-only authorization
- Silent privilege escalation
- Committing credentials/tokens/host-sensitive state