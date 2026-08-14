# 15 — Go Control-Plane Foundation

Status: Phase 2 foundation slice  
Date: 2026-08-14

## Purpose

This slice turns the documented architecture into the first running product control plane without prematurely opening an unauthenticated management surface.

It establishes:

- a Go process with structured logging and graceful shutdown;
- PostgreSQL-backed product state;
- a read-only workload inventory repository;
- append-only audit primitives;
- immutable access-policy history at the database layer;
- health/readiness endpoints;
- stable JSON HTTP errors and request correlation;
- real PostgreSQL integration tests in CI.

It does **not** complete Phase 2. SPIRE reconciliation, authenticated operator mutations, registration-rule workflows, policy activation workflows and broader CLI diagnostics remain future work.

## Runtime

The control-plane entrypoint is:

```text
apps/control-plane/main.go
```

Default listener:

```text
127.0.0.1:8080
```

The process rejects non-loopback listener addresses while the operator API is unauthenticated.

Required environment:

```text
DATABASE_URL=postgres://...
```

Optional environment:

```text
WTP_LISTEN_ADDR=127.0.0.1:8080
```

## Current HTTP surface

### `GET /healthz`

Process liveness only. It does not claim dependency readiness.

### `GET /readyz`

Checks PostgreSQL connectivity with a bounded timeout. Failure returns a generic `503` response and does not expose the underlying database error.

### `GET /v1/workloads?organization_id=<uuid>`

Returns organization-scoped workload inventory in deterministic order.

No workload mutation route exists. Unsupported methods receive a stable JSON `405` response with `Allow: GET`.

Unknown routes return a stable JSON `404` envelope.

Every response path passes through:

- generated request correlation ID;
- `Cache-Control: no-store`;
- `X-Content-Type-Options: nosniff`.

## PostgreSQL schema foundation

Migration `000001_control_plane` introduces:

- `organizations`;
- `trust_domains`;
- `workloads`;
- `registration_rules`;
- `access_policies`;
- `access_policy_versions`;
- `audit_events`;
- `security_events`.

### History integrity

`audit_events` rejects UPDATE and DELETE at the PostgreSQL trigger layer.

`access_policy_versions` also rejects UPDATE and DELETE. Changing policy history therefore requires creating a new version rather than rewriting an old one.

The application audit repository exposes an `Append` capability only; it deliberately has no update/delete methods.

## Database lifecycle evidence

CI executes the real PostgreSQL migration lifecycle:

```text
empty database
  -> migrate up
  -> seed representative product state
  -> prove policy-history mutation is rejected
  -> prove audit-history mutation is rejected
  -> migrate down
  -> prove schema removal
  -> migrate up again
```

Repository integration tests run inside database transactions and roll back their test state.

## Go dependency baseline

The Go module graph is committed and checked with `go mod tidy` before compilation. CI fails if `go.mod` or `go.sum` would change.

The first database driver is `pgx/v5`; the control plane uses a bounded `pgxpool` configuration and verifies connectivity during startup.

## Security boundaries

- there is no remote operator authentication yet;
- therefore the HTTP listener is loopback-only;
- there are no HTTP mutation endpoints;
- database/repository errors are not returned verbatim to clients;
- database URLs are never intentionally logged by the control plane;
- workload private keys are not part of the product schema;
- service authorization and mTLS enforcement remain Phase 3 work.

## Local development

Start PostgreSQL:

```bash
docker compose -f deploy/dev/compose.yaml up -d postgres
```

Apply the schema:

```bash
export DATABASE_URL='postgres://workload_trust:local-development-only@127.0.0.1:5432/workload_trust?sslmode=disable'
./scripts/db-migrate.sh up
```

Run the control plane:

```bash
go run ./apps/control-plane
```

The credentials above are intentionally local-development-only values from `.env.example`; they are not production credentials.
