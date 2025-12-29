## Quick Demo (Local)

StackWarden = minimal dashboard (HTML) + Go API + Go Agent.

### What you can demo
- **Health checks** (API + Agent)
- **Ports** (via Agent → API → UI)
- **Audit log** of `/ports` reads (in-memory, last 50 events)

---

## Run it

### 1) Start the Agent
In one terminal:

```bash
cd agent
go run .
