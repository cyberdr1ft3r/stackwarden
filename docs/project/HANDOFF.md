# StackWarden Handoff

Last updated: 2026-11-11

## Next concrete engineering task

Implement [Issue #26](https://github.com/cyberdr1ft3r/stackwarden/issues/26) on an independent branch from current `main`: block unsafe new Portainer installation at the API/catalog and agent boundaries without changing, uninstalling, or deleting any existing deployment or volume.

D-016 remains proposed. Do not add application deployment, adoption, Docker mutation, persistence, or broad installer refactoring to Issue #26.

## Read first

- Issue #26 and all comments.
- `AGENTS.md`
- `PROJECT_MEMORY.md`
- `docs/project/STATUS.md`
- `docs/project/DECISIONS.md`
- `docs/project/RISKS.md`
- `docs/project/ROADMAP.md`
- `docs/vision.md`
- `docs/threat-model.md`
- `.agents/skills/project-memory/SKILL.md`
- `.agents/skills/security-boundary/SKILL.md`

## Required implementation sequence

1. Inspect catalog metadata, embedded templates, API install routing, agent dispatch, staging, status/uninstall, and their tests.
2. Add an explicit default-denied Portainer install capability in shared/catalog behavior and enforce it again inside the agent before staging or command execution.
3. Return a clear compatible API error for disabled/unsupported installation while preserving catalog reads, status, authentication, and existing uninstall scope.
4. Add catalog/API/agent tests proving direct and mediated denial has no Compose, filesystem, deletion, volume, or running-resource side effects.
5. Review DDEV risk without expanding this fix; record any follow-up.
6. Run all six local equivalents and open a separate draft PR without merging or deploying.

## Completion conditions

- Fresh Portainer installs are denied through API and direct agent calls even with enabled writes and valid authorization.
- Denial occurs before staging or Docker/Compose execution and never removes files, containers, or volumes.
- Existing status and explicitly authorized uninstall behavior remain compatible and are not broadened.
- DDEV behavior is unchanged; any risk is documented as follow-up.
- Linux tests/build and Windows compile checks pass, along with formatting, vet, and `govulncheck`.
- Documentation describes the change as draft PR behavior until merged.
