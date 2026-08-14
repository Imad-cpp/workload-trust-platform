# 15 — Go Control-Plane Foundation

Status: Phase 2 active  
Date: 2026-08-14

## Purpose

The Phase 2 control plane now includes an authenticated local read boundary and an internal SPIRE desired-state reconciler while keeping remote management and HTTP mutation deliberately closed.

It establishes:

- a Go process with structured logging and graceful shutdown;
- PostgreSQL-backed product state;
- a read-only workload inventory repository;
- append-only audit primitives and immutable access-policy history;
- generic health/readiness endpoints;
- authenticated `/v1/*` reads for a configured local operator principal;
- registration reconciliation state and a one-shot SPIRE reconciler;
- official SPIRE Entry API integration over a local Unix socket;
- stable JSON HTTP errors/request correlation;
- real PostgreSQL and real SPIRE integration CI.

It does not complete Phase 2. Operator authorization beyond the single configured principal, authenticated operator mutation workflows, policy activation workflows, CLI diagnostics and the Phase 2 security/evidence exit remain future work.

## Operator runtime

Entrypoint:

```text
apps/control-plane/main.go
```

Default listener:

```text
127.0.0.1:8080
```

The process still rejects non-loopback listener addresses.

Required environment:

```text
DATABASE_URL=postgres://...
WTP_OPERATOR_ID=local-admin
WTP_OPERATOR_TOKEN=<at-least-32-bytes-of-high-entropy-secret>
```

Optional listener override must remain loopback:

```text
WTP_LISTEN_ADDR=127.0.0.1:8080
```

## Current HTTP surface

- `GET /healthz` — generic process liveness;
- `GET /readyz` — generic PostgreSQL readiness;
- authenticated `GET /v1/workloads?organization_id=<uuid>` — deterministic workload inventory.

No workload mutation route exists. There is no HTTP mutation endpoint for registration/policy state either.

`/v1/*` uses the ADR-0007 bearer authenticator. The configured plaintext credential is not retained by the authenticator after construction; a SHA-256 digest is compared to request credentials using a constant-time fixed-length comparison. This is an interim local high-entropy bearer mechanism, not a password or remote session scheme.

## SPIRE reconciliation runtime

Entrypoint:

```text
apps/reconciler/main.go
```

Configuration:

```text
DATABASE_URL=postgres://...
WTP_SPIRE_SERVER_SOCKET=/tmp/workload-trust-lab/server.sock
```

The socket path must be absolute. The reconciler reads registration desired state from PostgreSQL and uses the SPIRE v1.15.2 public Entry API to create, update or remove owned entries.

Ownership is checked using `wtp-rule:<rule-id>` plus the persisted SPIRE entry binding. Foreign entries fail closed. The marker is a controller convention, not cryptographic proof against a privileged SPIRE administrator.

## PostgreSQL schema

`000001_control_plane` introduces core product state/history tables. `000002_registration_reconciliation` adds parent identity, TTL, SPIRE binding, convergence status/timestamp and stable error-code fields to registration rules.

`audit_events` and `access_policy_versions` remain append-only at the PostgreSQL layer.

## Evidence

Permanent CI covers:

```text
Go module graph / gofmt / vet / race
PostgreSQL migrate up -> test -> down -> up
real PostgreSQL repository tests
SPIRE identity regression lab
PostgreSQL desired state -> SPIRE create -> workload SVID
owned selector/TTL drift -> in-place SPIRE update
foreign matching SPIRE entry -> ownership refusal
owned desired absent -> SPIRE delete -> workload loses identity
append-only reconciliation audit evidence
```

The reconciliation workflow never intentionally prints the ephemeral Phase 1 parent agent SPIFFE ID, which embeds join-token bootstrap material.

## Security boundaries

- operator HTTP remains loopback-only;
- `/v1/*` is authenticated but the project does not yet claim a multi-role authorization system;
- health/readiness remain unauthenticated and generic;
- no HTTP mutation endpoint exists;
- SPIRE management access is local and highly privileged;
- database/repository errors are not returned verbatim to HTTP clients;
- database URLs/operator credentials are not intentionally logged;
- workload private keys are not part of the product schema;
- service authorization and mTLS enforcement remain Phase 3 work.

## Local development

Start PostgreSQL and apply migrations:

```bash
docker compose -f deploy/dev/compose.yaml up -d postgres
export DATABASE_URL='postgres://workload_trust:local-development-only@127.0.0.1:5432/workload_trust?sslmode=disable'
./scripts/db-migrate.sh up
```

Run the local operator API:

```bash
export WTP_OPERATOR_ID='local-admin'
export WTP_OPERATOR_TOKEN="$(openssl rand -hex 32)"
go run ./apps/control-plane
```

Run one reconciliation cycle against a local SPIRE Server:

```bash
export WTP_SPIRE_SERVER_SOCKET='/tmp/workload-trust-lab/server.sock'
go run ./apps/reconciler
```

Local example database credentials are not production credentials.
