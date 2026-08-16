# 15 — Go Control-Plane Foundation

Status: Phase 2 active  
Date: 2026-08-14

## Purpose

The Phase 2 control plane now includes authenticated/authorized local management reads, transactionally audited registration desired-state mutations, and an internal SPIRE desired-state reconciler. Remote management remains closed.

It establishes:

- Go control-plane/reconciler processes with structured logging and bounded shutdown;
- PostgreSQL product state, append-only audit primitives and immutable policy history;
- generic health/readiness;
- loopback-only `/v1/*` bearer authentication;
- fail-closed local `viewer`/`operator` permission checks;
- authenticated workload reads;
- authorized registration desired-state POST/PATCH with strict/bounded JSON validation;
- optimistic registration revision protection;
- mutation + operator audit atomicity in PostgreSQL;
- registration reconciliation state and ownership-safe SPIRE v1.15.2 Entry API integration;
- real PostgreSQL, live HTTP and real SPIRE permanent CI.

It does not complete Phase 2. Policy activation, CLI diagnostics, consolidated control-plane security/failure review and the Phase 2 exit evidence remain future work.

## Operator runtime

Entrypoint: `apps/control-plane/main.go`

Default listener: `127.0.0.1:8080`. Non-loopback listener values remain rejected.

Required configuration:

```text
DATABASE_URL=postgres://...
WTP_OPERATOR_ID=local-admin
WTP_OPERATOR_TOKEN=<at-least-32-bytes-of-high-entropy-secret>
WTP_OPERATOR_ROLE=viewer|operator
```

Missing role defaults to `viewer`.

## Current HTTP surface

- `GET /healthz` — generic liveness;
- `GET /readyz` — generic PostgreSQL readiness;
- `GET /v1/workloads?organization_id=<uuid>` — authenticated/authorized inventory;
- `POST /v1/registration-rules` — requires `registrations:write`;
- `PATCH /v1/registration-rules/{id}` — full desired-state replacement requiring `expected_revision` and `registrations:write`.

No workload mutation route exists. Registration mutation routes now exist, but they only change pending PostgreSQL desired state; they do not call SPIRE directly.

The local role model is still one configured principal per process, not a remote/multi-user session/RBAC platform.

## Transactional mutation boundary

Registration create/replace operations validate canonical parent SPIFFE IDs, selector shape/count, TTL, JSON media/body shape and a 64 KiB body limit.

Every successful mutation and its attributed operator audit insert commit in the same PostgreSQL transaction. An intentionally failed audit insert is tested to roll back the desired-state mutation. Responses and audit summaries omit parent SPIFFE IDs and selector values.

## SPIRE reconciliation runtime

Entrypoint: `apps/reconciler/main.go`

```text
DATABASE_URL=postgres://...
WTP_SPIRE_SERVER_SOCKET=/tmp/workload-trust-lab/server.sock
```

The reconciler reads pending registration desired state and converges owned SPIRE entries through the v1.15.2 public Entry API over an absolute local Unix socket. `wtp-rule:<rule-id>` plus persisted binding is used as a non-cryptographic ownership convention. Foreign entries fail closed.

## PostgreSQL schema/history

`000001_control_plane` defines core product/history state. `000002_registration_reconciliation` adds parent identity, TTL, SPIRE binding, convergence and error state.

`audit_events` and `access_policy_versions` remain append-only at the PostgreSQL layer. Registration operator mutations use existing schema and transactional service semantics rather than a new migration.

## Permanent evidence

```text
Foundation validation
Go module graph / gofmt / vet / race
PostgreSQL migrate up -> test -> down -> up
real PostgreSQL repositories and atomic-audit rollback
live operator authz/mutation lifecycle: 401 / 403 / 201 / 200 / 409
SPIRE identity regression lab
PostgreSQL desired state -> SPIRE create -> workload SVID
owned drift -> SPIRE update
foreign matching entry -> ownership refusal
owned desired absent -> SPIRE delete -> workload loses identity
```

## Security boundaries

- management HTTP remains loopback-only;
- authentication and server-side role authorization protect current `/v1/*` management routes;
- registration mutation/audit is atomic and attributable;
- health/readiness remain unauthenticated and generic;
- SPIRE management access remains local and highly privileged;
- internal database/repository errors are not reflected verbatim;
- credentials and workload private keys are not intentionally logged/stored by product state;
- service authorization and mTLS enforcement remain Phase 3 work.

## Local development

```bash
docker compose -f deploy/dev/compose.yaml up -d postgres
export DATABASE_URL='postgres://workload_trust:local-development-only@127.0.0.1:5432/workload_trust?sslmode=disable'
./scripts/db-migrate.sh up
export WTP_OPERATOR_ID='local-admin'
export WTP_OPERATOR_ROLE='viewer'
export WTP_OPERATOR_TOKEN="$(openssl rand -hex 32)"
go run ./apps/control-plane
```

Use `WTP_OPERATOR_ROLE=operator` only when local registration desired-state write permission is intended.

```bash
export WTP_SPIRE_SERVER_SOCKET='/tmp/workload-trust-lab/server.sock'
go run ./apps/reconciler
```

Local example database credentials are not production credentials.
