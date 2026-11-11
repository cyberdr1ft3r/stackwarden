# StackWarden Handoff

Last updated: 2026-11-11

## Next concrete engineering task

Open a dedicated security/maintenance issue and PR to update the Go and CI
baseline from 1.25.13 to a release that fixes the current reachable
standard-library findings (currently Go 1.26.11), then restore a green
`govulncheck@v1.7.0` gate.

Do not fold the toolchain change into Issue #27. Keep Issue #27 draft pending
Issue #25/D-016 product-direction review and do not merge Docker observation as
accepted ownership or deployment behavior.

## Read first

- Current Go security release notes and all eleven reported vulnerability records.
- Issue #25, Issue #27, and their draft PR review state.
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

1. Confirm the minimum maintained Go release fixing every reachable finding.
2. Update module directives, CI setup, README, decisions, risks, and memory in one scoped PR.
3. Run formatting, vet, tests, Linux build, Windows compile, and the pinned vulnerability scan on the exact toolchain.
4. Do not change runtime behavior or feature scope in the maintenance PR.
5. After the gate is green, review/rebase the Issue #27 draft against the accepted product direction.

## Completion conditions

- All six mandatory CI checks pass on the exact PR head.
- No reachable standard-library vulnerability remains in the pinned scan.
- Linux behavior and Windows compilation remain functional.
- The Issue #27 draft remains read-only, bounded, and explicitly pending product review.
