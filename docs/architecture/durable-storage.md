# D-012: Durable Storage Architecture

- Status: Accepted
- Date: 2026-09-04
- Implementation: [Issue #23](https://github.com/cyberdr1ft3r/stackwarden/issues/23)

## Context

StackWarden currently keeps the previous listener snapshot, the latest 100 port-change events, and the latest 50 audit events in API memory. Restarting the API removes the evidence needed to explain drift, review administrative activity, or establish a trustworthy historical baseline.

Persistence must improve continuity without turning StackWarden into a general telemetry warehouse or moving privileged authority into the database.

## Decision summary

- Use SQLite as the initial storage engine through a repository interface owned by the API.
- Keep the first implementation single-host, but assign an opaque host ID and require `host_id` in every host-scoped storage operation from schema version 1.
- Store immutable snapshot occurrences that reference deduplicated, normalized inventory revisions.
- Persist listeners, minimally described services, detected catalog tools, drift events, and structured administrative audit events.
- Keep live metrics, PIDs, raw command output, collection diagnostics, credentials, tokens, and environment contents ephemeral.
- Use embedded, append-only forward migrations; WAL mode; full synchronous durability; foreign keys; bounded writer serialization; and explicit retention.
- Default to `/var/lib/stackwarden/api/stackwarden.db`, owned by the API service account, with a `0700` directory and `0600` database, WAL, and shared-memory files.
- Fail startup on migration, permission, corruption, or unsupported-schema errors. Never silently replace the database or fall back to volatile storage.

## Durable versus ephemeral data

Persist data needed to answer:

- which managed host was observed;
- what listeners, services, and supported tools existed at an observation time;
- what changed between valid snapshots;
- which administrative action was requested and how it completed.

Keep these ephemeral:

- CPU, memory, disk, and uptime gauges. They are live health data, not yet an approved time-series product.
- Process IDs, because they are short-lived identifiers and add little durable value.
- Raw `ss`, `netstat`, package-manager, Compose, or command output.
- Authorization headers, API Bearer tokens, credentials, environment values, request bodies, and tool configuration contents.
- Full hostnames, machine IDs, network-interface inventories, and other host fingerprints unless a later approved feature needs them.
- Failed or partial inventory payloads. A collection failure may emit a bounded operational/audit error code, but must not become a drift baseline.

## Initial topology and multi-host path

The first implementation manages one local host. On first successful startup, the API creates one `managed_hosts` row with an application-generated UUID and a fixed local-instance key. It does not derive identity from hostname, machine ID, MAC address, or another host fingerprint.

All repository methods and host-scoped tables require `host_id` from the start. The initial API selects only its configured local host and exposes no host enrollment, remote-agent transport, tenant, role, or host-switching UI.

Future multi-host work can add authenticated enrollment and a server-oriented repository implementation without changing snapshot, drift, or audit identity semantics. SQLite remains a single-instance store; concurrent multi-node API operation requires a later database decision.

## Storage engine

SQLite is the correct initial engine because StackWarden is a small, self-hosted, single-API process with one writer, modest query volume, and a preference for a minimal operational footprint.

| Option | Benefits | Costs / reason not selected initially |
| --- | --- | --- |
| SQLite | ACID transactions, foreign keys, indexes, mature backup/integrity tooling, one protected file, no database daemon | One writer at a time; shared-filesystem and active/active use are unsafe; file permissions and WAL backup require care |
| PostgreSQL | Strong concurrent writers, remote/multi-node operation, mature operations | Adds credentials, network exposure, service lifecycle, and operational burden before multi-host requirements exist |
| bbolt / key-value store | Embedded and simple deployment | Relational history queries, migrations, retention, and referential integrity become application code |
| JSON/flat files | No database dependency | Weak atomicity, concurrency, indexing, migrations, and corruption recovery |
| In-memory only | Simplest runtime | Fails the durability, drift-history, and forensic requirements |

The implementation should use `database/sql` with a maintained pure-Go SQLite driver so Linux builds and compile-only Windows checks do not require CGO. The dependency and version must receive normal CI and vulnerability review when added.

## Layer boundaries

```text
Agent collectors
  -> typed live observations
API collector/orchestrator
  -> validation + normalization
Inventory/drift service
  -> transaction request
Storage repository interface
  -> SQLite implementation
API handlers
  -> service queries / response DTOs
UI
```

- The agent collects live facts and never opens the database.
- The API validates agent responses, assigns the configured `host_id`, computes normalized inventory and drift, and owns retention.
- HTTP handlers and UI code never issue SQL or depend on SQLite row layouts.
- Storage types remain separate from API response types and agent wire types.
- The repository interface supports transactions and always takes a host ID for host-scoped methods.
- Existing `/v1/read/*` and `/v1/write/*` authorization boundaries remain unchanged.

## Collection and presentation semantics

- The API collector captures one complete snapshot after both storage and the agent become healthy, then every five minutes by default.
- `STACKWARDEN_SNAPSHOT_INTERVAL` may configure a duration from one minute through 24 hours. Invalid or disabled values fail startup; the initial implementation does not allow unbounded/disabled durable capture.
- Collection is single-flight. A slow collection is not overlapped, and no database transaction is held while calling the agent.
- `GET /v1/read/ports` remains a live agent-backed compatibility endpoint and no longer creates a snapshot as a side effect.
- `GET /v1/read/audit` and `/v1/read/port-events` retain their response compatibility while reading bounded durable history.
- New read-only history endpoints may expose latest snapshot, snapshot list/detail, and typed drift history. They must use bounded pagination and remain behind the API service layer.
- The first implementation adds no browser/API operation for forcing collection, changing retention, deleting history, or exporting the database.
- Existing in-memory history is not migrated. The first complete collection after upgrade establishes the durable baseline and emits no synthetic “opened” events.

## Proposed entity model

```text
managed_hosts 1---* host_snapshots *---1 inventory_revisions
                                      |---* listener_observations
                                      |---* service_observations
                                      `---* tool_observations

managed_hosts 1---* drift_events ---0..1 from/to host_snapshots
managed_hosts 1---* audit_events
schema_migrations (database-wide)
```

All IDs are lowercase canonical UUID strings generated by the API. UUIDv7 is preferred for new records because it remains opaque while providing useful insertion locality; timestamps, not UUID ordering, are authoritative.

All timestamps are UTC Unix milliseconds in `INTEGER` columns. API responses may continue rendering RFC3339. The API clock is authoritative for receipt, persistence, drift, and audit times.

### `schema_migrations`

| Column | Notes |
| --- | --- |
| `version INTEGER PRIMARY KEY` | Monotonically increasing schema version |
| `name TEXT NOT NULL` | Stable migration name |
| `checksum TEXT NOT NULL` | SHA-256 of embedded migration SQL |
| `applied_at_ms INTEGER NOT NULL` | Successful commit time |

### `managed_hosts`

| Column | Notes |
| --- | --- |
| `id TEXT PRIMARY KEY` | Application-generated host UUID |
| `local_instance_key TEXT NOT NULL UNIQUE` | Fixed local key for initial single-host lookup |
| `display_name TEXT NULL` | Explicit operator label; no automatic hostname capture |
| `os TEXT NOT NULL`, `arch TEXT NOT NULL` | Coarse platform facts |
| `created_at_ms`, `last_seen_at_ms INTEGER NOT NULL` | Lifecycle timestamps |
| `retired_at_ms INTEGER NULL` | Soft retirement; no automatic host deletion |

### `inventory_revisions`

One normalized inventory body may be referenced by many snapshot occurrences.

| Column | Notes |
| --- | --- |
| `id TEXT PRIMARY KEY` | Revision UUID |
| `host_id TEXT NOT NULL` | `managed_hosts(id)` |
| `content_hash BLOB NOT NULL` | SHA-256 of versioned canonical inventory |
| `format_version INTEGER NOT NULL` | Canonicalization format |
| `created_at_ms INTEGER NOT NULL` | First observation |
| `listener_count`, `service_count`, `tool_count INTEGER NOT NULL` | Validation/deduplication aids |

Constraints: `UNIQUE(host_id, content_hash, format_version)` and `UNIQUE(id, host_id)`.

### `host_snapshots`

| Column | Notes |
| --- | --- |
| `id TEXT PRIMARY KEY` | Snapshot UUID |
| `host_id TEXT NOT NULL` | Managed host |
| `revision_id TEXT NOT NULL` | Deduplicated inventory revision for the same host |
| `collected_at_ms`, `persisted_at_ms INTEGER NOT NULL` | Observation and commit times |
| `collector_version TEXT NOT NULL` | StackWarden collector/schema producer version |

Only complete, validated collections are inserted. Partial collections never become a baseline.

### `listener_observations`

| Column | Notes |
| --- | --- |
| `revision_id TEXT NOT NULL` | Parent inventory revision; cascade on delete |
| `listener_key TEXT NOT NULL` | Hash/key of normalized protocol, address, and port |
| `protocol TEXT NOT NULL` | Normalized `tcp`/`udp` |
| `local_address TEXT NOT NULL` | Canonical IP or wildcard representation |
| `local_port INTEGER NOT NULL` | Range 1-65535 |
| `state TEXT NULL` | Bounded normalized state |
| `process_name TEXT NULL` | Bounded executable name only; no arguments/path |
| `exposure TEXT NOT NULL` | `local`, `private`, `public`, or `unknown` |

Primary key: `(revision_id, listener_key)`. PIDs and raw command output are not stored.

### `service_observations`

| Column | Notes |
| --- | --- |
| `revision_id TEXT NOT NULL` | Parent inventory revision; cascade on delete |
| `service_key TEXT NOT NULL` | Stable key within the observation source |
| `name TEXT NOT NULL` | Bounded service/process name |
| `manager TEXT NOT NULL` | Bounded source such as `process`, `systemd`, or `windows_scm` |
| `state TEXT NOT NULL` | Normalized state |
| `version TEXT NULL` | Bounded version when safely available |

Primary key: `(revision_id, service_key)`. Initial collection may use only safely available process-derived service facts; adding systemd/SCM inventory is separate collector scope.

### `tool_observations`

| Column | Notes |
| --- | --- |
| `revision_id TEXT NOT NULL` | Parent inventory revision; cascade on delete |
| `tool_id TEXT NOT NULL` | Validated shared-catalog ID |
| `install_kind TEXT NOT NULL` | Catalog install kind |
| `staged`, `installed`, `running INTEGER NOT NULL` | Boolean state |
| `version TEXT NULL` | Bounded normalized version |

Primary key: `(revision_id, tool_id)`. Status command output, paths, and errors are not stored.

### `drift_events`

| Column | Notes |
| --- | --- |
| `id TEXT PRIMARY KEY`, `host_id TEXT NOT NULL` | Event and host identity |
| `from_snapshot_id`, `to_snapshot_id TEXT NULL` | Set null when snapshot retention removes a referenced snapshot |
| `kind TEXT NOT NULL` | Bounded event kind |
| `resource_type TEXT NOT NULL`, `resource_key TEXT NOT NULL` | Listener/service/tool reference |
| `details_json TEXT NOT NULL` | Versioned, validated structured details only |
| `detected_at_ms INTEGER NOT NULL` | Detection time |

Events are self-describing enough to survive snapshot pruning. Free-form command output is prohibited.

`details_json` uses a kind-specific schema, rejects unknown fields, and is capped at 4 KiB. Human-readable descriptions are rendered from structured values rather than stored command/error text.

### `audit_events`

Administrative mutations use two append-only events with the same request ID: `requested`, then `completed` or `failed`. A missing terminal event means the outcome is unknown after interruption.

| Column | Notes |
| --- | --- |
| `id TEXT PRIMARY KEY`, `host_id TEXT NULL` | Event identity and optional host |
| `request_id TEXT NOT NULL` | Random correlation UUID, not a credential |
| `phase TEXT NOT NULL` | `requested`, `completed`, or `failed` |
| `actor_kind TEXT NOT NULL` | Bounded value such as `api_token` or `system`; no token identity/value |
| `action TEXT NOT NULL` | Bounded action name |
| `target_type`, `target_id TEXT NULL` | Validated resource reference |
| `outcome TEXT NOT NULL`, `error_code TEXT NULL` | Structured result; no raw error/output |
| `occurred_at_ms INTEGER NOT NULL` | API event time |

Constraint: `UNIQUE(request_id, phase)`. A state-changing action must not reach the agent if its `requested` audit event cannot be committed.

Read-only observations may append one `completed` or `failed` event without blocking the read when audit persistence is unavailable. State-changing actions use the strict request/outcome sequence. Audit rows are append-only until retention pruning.

## Relationships and deletion behavior

- `inventory_revisions.host_id` references `managed_hosts.id` with `ON DELETE RESTRICT`.
- `host_snapshots.(revision_id, host_id)` references `inventory_revisions.(id, host_id)` so a snapshot cannot cross host ownership.
- Listener, service, and tool observations reference their revision with `ON DELETE CASCADE`.
- Drift events reference the host with `ON DELETE RESTRICT` and snapshots with `ON DELETE SET NULL`.
- Audit events reference hosts with `ON DELETE SET NULL` so administrative evidence can outlive an explicitly purged host.
- Drift events have a deterministic uniqueness constraint across host, from/to snapshots, kind, resource type, and resource key.
- Managed hosts are soft-retired. The first implementation has no physical host-delete or user-facing purge operation.
- Snapshot pruning deletes occurrences first, then deletes only revisions no longer referenced by any snapshot.

## Required indexes

- `host_snapshots(host_id, collected_at_ms DESC)`
- `host_snapshots(revision_id)`
- `inventory_revisions(host_id, content_hash, format_version)` unique
- `listener_observations(revision_id, protocol, local_address, local_port)`
- `service_observations(revision_id, name)`
- `tool_observations(revision_id, tool_id)`
- `drift_events(host_id, detected_at_ms DESC)`
- `drift_events(host_id, kind, detected_at_ms DESC)`
- `audit_events(host_id, occurred_at_ms DESC)`
- `audit_events(action, occurred_at_ms DESC)`
- `audit_events(request_id, occurred_at_ms)`

Query plans for latest snapshot, bounded history, drift-by-kind, and audit-by-action must use indexes in tests.

## Snapshot transaction and deduplication

1. Collect all required scopes before opening a write transaction.
2. Validate and normalize bounded fields.
3. Sort normalized listener, service, and tool tuples and hash an unambiguous length-prefixed representation prefixed by `format_version`.
4. Begin an immediate write transaction.
5. Reuse the host-scoped inventory revision on hash match after verifying counts and stored normalized rows; otherwise insert the revision and child observations.
6. Insert a new snapshot occurrence even when the revision is reused, preserving observation cadence.
7. Compare the previous complete snapshot and insert deterministic drift events in the same transaction.
8. Update `managed_hosts.last_seen_at_ms` and commit atomically.

The first valid snapshot creates no drift. Retries use deterministic event uniqueness within the from/to snapshot pair so they cannot duplicate drift records.

## Concurrency and SQLite configuration

The API is the only database owner. Use WAL mode with:

- `PRAGMA foreign_keys=ON` on every connection;
- `PRAGMA journal_mode=WAL`;
- `PRAGMA synchronous=FULL`;
- `PRAGMA busy_timeout=5000`;
- `PRAGMA temp_store=MEMORY`.

Allow a small bounded read pool and serialize write transactions through the repository. Snapshot/revision/drift commits, audit appends, and pruning each use short transactions; no agent call, network call, command, JSON response write, or long hash computation occurs inside a database transaction.

SQLite is not placed on NFS or another network filesystem. A second API process sharing the same database is unsupported. Lock/busy exhaustion returns an explicit error and never silently drops durable state.

## Retention and pruning

Initial defaults:

- snapshots: 30 days and at most 10,000 occurrences per host;
- drift events: 180 days;
- audit events: 90 days.

Retention is configurable through validated API-owned settings with finite defaults. Pruning runs after a successful snapshot and at most once per day, deletes in bounded batches, and always preserves the newest two complete snapshots for every active host.

Age and count limits are both enforced: records older than the age limit are eligible for deletion, and the oldest remaining records are deleted when the count limit is exceeded. The newest-two safeguard takes precedence over both limits.

Snapshot deletion sets drift snapshot references to null. Orphaned inventory revisions and their listener/service/tool children are then deleted in the same maintenance transaction. Drift and audit rows are independently age-pruned. Managed hosts are soft-retired and are never automatically deleted.

The implementation must not expose a general deletion API. Explicit host purge, legal hold, event acknowledgement, and per-host policy are deferred. Automatic `VACUUM` on every prune is prohibited; periodic WAL checkpoint and incremental vacuum may be performed as bounded maintenance.

## Migrations and schema versioning

- Embed ordered SQL migrations in the API binary.
- Migrations are append-only, checksum-verified, and applied once under an exclusive startup lock.
- `schema_migrations` is authoritative. Refuse to open a database with a newer unknown schema or a changed checksum.
- Apply each migration transactionally and record its version only in the same commit.
- Do not perform automatic down migrations.
- Migration failure rolls back, leaves the original database intact, and prevents normal API startup.
- Before a destructive/rebuild migration, require a verified backup and enough free space for both old and new forms.

Tests must cover a new database, every supported prior schema, interrupted/failed migration rollback, checksum mismatch, and newer-schema refusal.

## Filesystem and sensitive-data controls

- Configuration: `STACKWARDEN_DB_PATH`; default `/var/lib/stackwarden/api/stackwarden.db`.
- Require an absolute, cleaned path. Reject symlinks and paths escaping the configured API data directory.
- The API service account owns the parent directory and database-related files.
- Minimum default modes: directory `0700`; database, `-wal`, `-shm`, lock, and backup files `0600`.
- Do not place the database beneath the UI/static root or agent tools directory.
- Refuse startup on ownership or permission mismatch instead of broadening access.
- SQLite provides no application-level encryption. Operators should use encrypted storage where host inventory confidentiality requires it.
- Backups receive the same permissions and sensitivity classification as the live database.

The schema and repository API must have no fields for Bearer tokens, authorization headers, credentials, environment contents, arbitrary command output, or request bodies.

## Backup, restore, corruption, and shutdown

- Create online backups with SQLite's backup API or `VACUUM INTO` to a `0600` temporary file, verify it, fsync it and its directory, then atomically rename it.
- Never copy only the main database file while WAL mode is active.
- An offline backup may be taken only after a clean checkpoint and close; copy all required database files if closure cannot be guaranteed.
- Restore is an offline operator action: stop the API, preserve the current files, validate ownership/modes, run `PRAGMA integrity_check`, verify schema compatibility, atomically replace, then restart.
- On corruption, preserve the database/WAL files for diagnosis, report an explicit unhealthy state, and require restore or an explicit recovery tool. Never delete and recreate automatically.
- On shutdown, stop collection, reject new mutations, drain bounded writes, checkpoint WAL, close the database, and report timeout/failure rather than claiming a clean shutdown.

Backup scheduling, remote backup upload, encryption/key management, and point-in-time recovery are deferred.

## Future implementation acceptance criteria

- Add an API-owned storage package and repository interface; the UI and agent have no database access.
- Add a reviewed pure-Go SQLite dependency and embedded schema migrations.
- Enforce the configured absolute database path, ownership, `0700` directory mode, and `0600` database-related file modes without following symlinks.
- Bootstrap one opaque local managed-host UUID without collecting a hostname or machine ID.
- Persist complete listener/service/tool inventory snapshots and survive API restart.
- Capture snapshots in a single-flight API scheduler with the documented interval bounds; live `/v1/read/ports` requests must not mutate history.
- Deduplicate normalized inventory revisions while preserving each snapshot occurrence.
- Atomically create snapshots and deterministic drift events; the first snapshot creates no drift, and opened/closed/exposure events retain current `diffPorts` semantics.
- Persist structured administrative request/outcome audit events, fail closed before mutation if the request event cannot be committed, and never persist secrets or raw output.
- Provide bounded latest/history queries without changing existing route authorization or response compatibility.
- Apply the documented default retention and orphan cleanup while preserving the latest two snapshots.
- Implement online backup, offline restore validation, corruption refusal, and clean shutdown behavior.
- Test migrations, rollback, deduplication, transaction atomicity, concurrent reads/serialized writes, busy timeout, retention, foreign-key deletion, indexed query plans, restart durability, backup/restore, corruption, permissions, symlink/path denial, and sensitive-data exclusion.
- Keep Linux runtime behavior functional and compile supported Windows observation/API code without executing Windows binaries in Linux CI.
- Pass all six mandatory CI checks.

## Explicit non-goals

- No database implementation in the D-012 architecture PR.
- No PostgreSQL deployment, active/active API, remote/shared SQLite, or automatic SQLite-to-PostgreSQL replication.
- No multi-host enrollment, remote agent transport, tenant model, RBAC, or host-switching UI.
- No metrics time-series retention, logs/SIEM replacement, packet/process history, or arbitrary inventory blobs.
- No persistence of tokens, credentials, environment/config contents, command output, request bodies, or full host fingerprints.
- No policy/remediation expansion, deployment automation, backup upload service, or encryption key management.
- No user-facing delete/purge API in the first implementation.

## Remaining decisions

- Select and pin the pure-Go SQLite driver during implementation after license, maintenance, cross-compilation, and vulnerability review.
- Decide the eventual remote-host enrollment/authentication protocol before enabling multi-host operation.
- Decide whether later service inventory warrants dedicated systemd/Windows SCM collectors; the initial schema supports it without requiring that expansion.
