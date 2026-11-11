# StackWarden API

## Read-only Docker application discovery

`GET /v1/read/docker/apps` asks the local agent for one bounded Docker Engine
container-list observation. The response is typed and reports one of
`available`, `partial`, `unavailable`, or `unknown`; an unavailable
daemon is not represented as a successful empty inventory.

Returned container fields are limited to a Docker-scoped ID, name, image
reference, normalized state and optional running value, bounded status text,
published-port facts, and optional Compose project/service grouping hints.
Compose labels are untrusted hints and do not establish StackWarden ownership.
Published ports do not prove Internet reachability. The response references
`/v1/read/ports` for independent host-listener observations without inventing
container-to-listener attribution.

The endpoint never returns environment variables, mounts, private label values,
credentials, raw Engine JSON, or a client-selectable Docker socket. Expected
Docker absence, permission denial, timeout, malformed data, and response-limit
failures are represented by bounded reason codes rather than raw errors.
