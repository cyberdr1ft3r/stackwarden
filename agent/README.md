# StackWarden local agent

The agent is the only StackWarden component permitted to access the local
Docker Engine. Read-only application discovery uses the fixed
`/var/run/docker.sock` path on Unix-like hosts and performs exactly one
`GET /v1.41/containers/json?all=1` request. The browser and API cannot select
another socket, Engine URL, method, path, command, or request body.

Collection is single-flight, timeout-controlled, byte/count/string bounded, and
fully decoded before the API receives a typed result. Windows builds preserve
the endpoint contract but report Docker socket discovery as unavailable.
