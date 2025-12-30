# ToolCTL - Vision

Goal: Self-hosted Linux server dashboard for DevOps deployments + basic hardening.

## v0 Target
- OS: Ubuntu 24.04 (or 22.04)
- Default access: localhost only + SSH tunnel
- Components:
  - UI (web dashboard)
  - API (auth, audit, settings)
  - Agent (root, whitelisted operations) via Unix socket

## Principles
- No arbitrary shell execution from the UI
- RBAC (viewer/operator/admin)
- Audit log for every action
- Idempotent operations (ensure desired state)
