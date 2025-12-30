# StackWarden

Minimal dashboard (HTML) + Go API + Go Agent for DevOps, security, and server visibility.

StackWarden is a lightweight control panel that lets you:
- Check API & agent health
- View listening ports on servers
- See an audit trail of sensitive actions

---

## Features

- **Health checks**
  - API health
  - Agent health

- **Ports visibility**
  - API calls the agent to list listening ports
  - Works on Linux and Windows

- **Audit log**
  - Records every `/ports` read
  - In-memory, last 50 events
  - Includes time, action, and success/failure

- **Minimal UI**
  - No frameworks
  - Manual refresh buttons (no polling spam)
  - Fast and lightweight

---
#  Vision

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


## Quick Demo (Local)

StackWarden = minimal dashboard (HTML) + Go API + Go Agent.

### What you can demo
- Health checks (API + Agent)
- Ports (via Agent → API → UI)
- Audit log of `/ports` reads (in-memory, last 50 events)

---

## Run it

### 1) Start the Agent
In one terminal:

```bash
cd agent
go run .
```

By default, the agent listens on:
```
http://127.0.0.1:9091
```

---

### 2) Start the API
In another terminal:

```bash
cd api
# optional: point API to a different agent base URL
# export AGENT_BASE="http://127.0.0.1:9091"
go run .
```

The API exposes:
- `GET /health`
- `GET /agent/health`
- `GET /ports`
- `GET /audit`

---

### 3) Open the UI

If the API serves the UI:
```
http://127.0.0.1:8080
```

Or open directly:
```
ui/index.html
```

---

## Demo Flow

1. Open **Home**
   - Check API + Agent health

2. Open **Ports**
   - Click **Refresh Ports**
   - See current listening ports (proto, address, port, PID)

3. Open **Audit**
   - Click **Refresh Audit**
   - See `ports.read` events with timestamps and success status

---

## Manual API checks (optional)

```bash
curl -s http://127.0.0.1:8080/health
curl -s http://127.0.0.1:8080/agent/health
curl -s http://127.0.0.1:8080/ports
curl -s http://127.0.0.1:8080/audit
```

---

## Architecture

```
[ Browser UI ]
      |
      v
[ Go API ] ----> [ Go Agent ] ----> OS (ports, health)
```

- UI talks to API
- API proxies sensitive operations to Agent
- Agent runs on the target server

---

## Audit Log

- Stored in memory (no database)
- Last 50 events only
- Reset on API restart
- Each event includes:
  - `time` (RFC3339)
  - `action` (e.g. `ports.read`)
  - `result` (true / false)

---

## Why StackWarden?

StackWarden is designed to be:
- **Fast**
- **Auditable**
- **Extensible**
- **Secure by design**

It gives you a foundation for building a real server control plane without heavyweight dependencies.

---

## License
MIT
