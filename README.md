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

### Remote server
Open http://<server-ip>:8080/ and ensure your firewall / security group allows inbound TCP 8080.

## Configuration
- `AGENT_BIND` (default `:9091`): agent listen address.
- `API_BIND` (default `:8080`): API listen address (binds to all interfaces by default).
- `AGENT_BASE` (default `http://127.0.0.1:9091`): base URL the API uses to reach the agent.

Useful commands:
- `make test` — run Go tests across modules
- `make build` — build binaries into `./bin/`

## Features
- Health checks (API + Agent)
- Ports visibility (API proxies the agent)
- Audit log (last 50 events)
- Minimal UI (no frameworks) with manual refresh controls

## Manual API checks (optional)
```bash
curl -s http://127.0.0.1:8080/health
curl -s http://127.0.0.1:8080/agent/health
curl -s http://127.0.0.1:8080/version
curl -s http://127.0.0.1:8080/metrics
curl -s http://127.0.0.1:8080/ports
curl -s http://127.0.0.1:8080/audit
```

## Architecture
```
[ Browser UI ]
      |
      v
[ Go API ] ----> [ Go Agent ] ----> OS (ports, health, metrics)
```

## License
MIT
