# StackWarden Handoff

Last updated: 2026-11-11

## Next concrete engineering task

After Issue #26 review, implement [Issue #27](https://github.com/cyberdr1ft3r/stackwarden/issues/27) on an independent branch from current `main`: bounded read-only Docker application discovery through the local agent.

Keep the PR in draft pending product-direction review. Do not add Docker mutation, deployment, ownership adoption, arbitrary commands, raw Engine proxying, or persistence.

## Read first

- Issue #27 and all comments.
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

1. Define minimal typed shared DTOs that preserve available/unavailable/unknown states.
2. Keep Docker Engine access inside the local agent with exact read-only operations, timeouts, byte/count/string bounds, and no client-selected socket.
3. Return container name/scoped ID, image reference, status/running state, published-port facts, and optional bounded Compose grouping hints.
4. Mediate through a `/v1/read/*` Go API endpoint without changing write authorization.
5. Add deterministic mocks for absent, denied, timeout, oversized, and malformed Docker responses plus API and Windows compile tests.
6. Update project memory truthfully and open a draft PR without merge or deployment.

## Completion conditions

- Docker unavailable is distinguishable from a successful empty inventory.
- Results and errors are typed, bounded, and exclude environment, mounts, private labels, raw Engine JSON, credentials, and unnecessary host fingerprints.
- Compose labels remain untrusted hints and published ports are not described as proven Internet exposure.
- Negative tests prove there is no Docker write, Exec, arbitrary command/path, remote socket, adoption, or deployment surface.
- Linux behavior remains functional; Windows agent/API code compiles without execution.
- All six mandatory CI checks pass.
- Project memory describes Issue #27 behavior as proposed/draft until merged.
