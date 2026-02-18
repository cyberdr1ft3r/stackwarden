# StackWarden

Minimal dashboard (HTML) + Go API + Go Agent for DevOps, security, and server visibility.

## Quickstart
1) Clone and enter the repo:
```bash
git clone https://github.com/m0b3u/stackwarden.git
cd stackwarden
```
2) Start the services (separate terminals):
```bash
make run-agent
make run-api
```
3) Open the UI: http://127.0.0.1:8080/

### Platform notes
- Linux/macOS: use `make run-api` (requires only Go + Make).
- Windows (PowerShell, no make required): use `go run ./api` (and `go run ./agent`).

### Remote server access (safe defaults)
By default the API binds to loopback only (`127.0.0.1:8080`), so reach it via a tunnel/reverse proxy instead of opening TCP 8080 directly.

SSH tunnel example:
```bash
ssh -L 8080:127.0.0.1:8080 user@server
# then browse locally:
# http://127.0.0.1:8080/
```

## Configuration
- `API_BIND` (default `127.0.0.1:8080`): API listen address.
- `ALLOW_NONLOCAL_BIND` (default unset): must be `1` to allow non-loopback API binds.
- `AGENT_SOCKET` (default `/run/stackwarden/agent.sock`): Unix socket path used by API and agent.
- `STACKWARDEN_WRITE_ENABLED` (default `false`): enables `/v1/write/*` endpoints.
- `STACKWARDEN_TOKEN`: required Bearer token when write endpoints are enabled.

Useful commands:
- `make test` — run Go tests across modules
- `make build` — build binaries into `./bin/`

## Features
- Health checks (API + Agent)
- Ports visibility (API proxies the agent)
- Audit log (last 50 events)
- Tool catalog with server-side installs (agent executes installs)
- Minimal UI (no frameworks) with manual refresh controls

## Tool installs
- Installs are executed on the **server** by the agent (no browser downloads required).
- Tool files and artifacts are staged under `/var/lib/stackwarden/tools/<id>/`.
- Compose tools require Docker (or Docker Compose) running on the host.
- DDEV install flow targets Debian/Ubuntu; other operating systems return “not supported yet.”
- Bundles remain downloadable as an optional client-side ZIP.

## Manual API checks (optional)
```bash
curl -s http://127.0.0.1:8080/v1/read/health
curl -s http://127.0.0.1:8080/v1/read/agent/health
curl -s http://127.0.0.1:8080/v1/read/version
curl -s http://127.0.0.1:8080/v1/read/metrics
curl -s http://127.0.0.1:8080/v1/read/ports
curl -s http://127.0.0.1:8080/v1/read/audit
curl -X POST -H "Authorization: Bearer $STACKWARDEN_TOKEN" -s http://127.0.0.1:8080/v1/write/tools/portainer/install
```

## Security verification checklist
- `ss -ltnp | rg ':8080'` shows API bound to `127.0.0.1:8080` (unless explicitly overridden with `ALLOW_NONLOCAL_BIND=1`).
- `ss -ltnp | rg 9091` shows no TCP listener for the agent.
- `curl -i -X POST http://127.0.0.1:8080/v1/write/tools/portainer/install` returns `403` + `{"error":"write_disabled"}` by default.
- With `STACKWARDEN_WRITE_ENABLED=true`, same request without token returns `401` + `{"error":"unauthorized"}`.
- With `STACKWARDEN_WRITE_ENABLED=true` and correct Bearer token, `/v1/write/*` works.

## Architecture
```
[ Browser UI ]
      |
      v
[ Go API ] ----(unix:///run/stackwarden/agent.sock)---> [ Go Agent ] ----> OS (ports, health, metrics)
```

## License
MIT
