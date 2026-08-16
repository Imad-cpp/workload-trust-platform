# 15 — Go Control-Plane Foundation

Status: Phase 2 active  
Date: 2026-08-16

## Purpose

The Phase 2 control plane now includes authenticated/authorized local management reads, transactionally audited registration desired-state mutations, versioned policy management/activation, safe aggregate diagnostics, a local read-only CLI, and an internal SPIRE desired-state reconciler. Remote management remains closed.

It establishes:

- Go control-plane/reconciler processes with structured logging and bounded shutdown;
- PostgreSQL product state, append-only audit primitives and immutable policy history;
- generic health/readiness;
- loopback-only `/v1/*` bearer authentication;
- fail-closed local `viewer`/`operator` permission checks;
- authenticated workload reads;
- aggregate `diagnostics:read` status without sensitive rule/audit payloads;
- authorized registration desired-state POST/PATCH with optimistic revisions and atomic audit;
- authorized policy create/version/activation routes with distinct activation permission;
- immutable policy versions and optimistic policy envelope revisions;
- canonical SPIFFE policy validation, stored-version revalidation and atomic activation audit;
- local read-only `wtpctl` through the existing management API, with no direct PostgreSQL dependency;
- registration reconciliation state and ownership-safe SPIRE v1.15.2 Entry API integration;
- real PostgreSQL, live HTTP, CLI and real SPIRE permanent CI.

It does not complete Phase 2. Consolidated control-plane security/failure review, the local-principal decision and Phase 2 exit evidence remain future work. Workload service authorization remains Phase 3.

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
- `GET /v1/diagnostics?organization_id=<uuid>` — aggregate status with `diagnostics:read`;
- `POST /v1/registration-rules` — requires `registrations:write`;
- `PATCH /v1/registration-rules/{id}` — full desired-state replacement requiring `expected_revision` and `registrations:write`;
- `POST /v1/policies` — creates draft policy + immutable version 1 with `policies:write`;
- `POST /v1/policies/{id}/versions` — appends immutable version with `policies:write` + `expected_revision`;
- `POST /v1/policies/{id}/activate` — selects a valid owned version with `policies:activate` + `expected_revision`.

No workload mutation route exists. Registration/policy routes change PostgreSQL management state only. They do not directly enforce workload access.

The diagnostics route returns aggregate workload/registration/policy/audit/security counts and latest audit time. It omits selectors, parent identity, policy rule identity/change reason, audit metadata and credentials.

The local role model is still one configured principal per process, not a remote/multi-user session/RBAC platform.

## Local CLI

Entrypoint: `apps/wtpctl/main.go`

```text
wtpctl health
wtpctl ready
wtpctl workloads --organization-id <uuid>
wtpctl status --organization-id <uuid>
```

`wtpctl` is an HTTP client of the loopback management API; it does not use `DATABASE_URL`. `WTP_API_URL` is restricted to local plain HTTP, redirects are refused, authenticated commands use the existing bearer token, requests time out after five seconds and responses over 1 MiB are rejected without partial output.

`status` uses the aggregate diagnostics route. `workloads` intentionally exposes the existing authorized workload inventory, including workload SPIFFE IDs.

## Transactional management boundary

Registration and policy mutation bodies use strict bounded JSON and stable generic errors. Policy V1 accepts canonical source/destination SPIFFE IDs, `connect`, `allow|deny` and a bounded change reason.

Successful registration mutations and successful policy create/version/activation operations each commit their attributed audit insert in the same PostgreSQL transaction. Intentionally failed audit inserts are tested to roll the corresponding state change back.

Policy responses/audit summaries omit source/destination SPIFFE IDs and change-reason text. Registration responses/audits omit parent/selector values.

## Policy activation semantics

Policy versions are immutable. `access_policies.revision` provides optimistic concurrency. Activation requires a version owned by the target policy and revalidates its stored content immediately before selecting it.

`active_version_id` means selected desired policy state only. It is **not** an enforcement result and does not grant service access by itself.

## SPIRE reconciliation runtime

Entrypoint: `apps/reconciler/main.go`

```text
DATABASE_URL=postgres://...
WTP_SPIRE_SERVER_SOCKET=/tmp/workload-trust-lab/server.sock
```

The reconciler reads pending registration desired state and converges owned SPIRE entries through the v1.15.2 public Entry API over an absolute local Unix socket. `wtp-rule:<rule-id>` plus persisted binding is used as a non-cryptographic ownership convention. Foreign entries fail closed.

## PostgreSQL schema/history

- `000001_control_plane` defines core product/history state;
- `000002_registration_reconciliation` adds SPIRE convergence state;
- `000003_policy_revision` adds optimistic revision state to the policy envelope.

Migration CI applies all three in order, rolls them back in reverse order, reapplies them, and semantically verifies policy revision defaults after re-apply.

`audit_events` and `access_policy_versions` remain append-only at the PostgreSQL layer.

## Permanent evidence

```text
Foundation validation
Go module graph / gofmt / shell syntax / vet / race
PostgreSQL migrations 000001 -> 000002 -> 000003 -> rollback -> re-apply
real PostgreSQL repositories and forced atomic-audit rollback tests
live registration management: 401 / 403 / 201 / 200 / 409
live policy management: 401 / 403 / 201 / 201 / 409 / 200 / foreign 404
live CLI diagnostics: health / ready / status / workloads / wrong-token / non-loopback refusal
CLI status leak checks + no direct DB credential dependency + read-only audit-count proof
SPIRE identity regression lab
PostgreSQL desired registration state -> SPIRE create/update/delete
foreign matching SPIRE entry -> ownership refusal
```

## Security boundaries

- management HTTP remains loopback-only;
- authentication and server-side role authorization protect current `/v1/*` management routes;
- registration and policy state/audit mutations are atomic and attributable;
- health/readiness remain unauthenticated and generic;
- CLI inspection remains local/read-only and does not bypass the API for database access;
- SPIRE management access remains local and highly privileged;
- internal database/repository errors are not reflected verbatim;
- credentials and workload private keys are not intentionally logged/stored by product state;
- policy activation is management state, not service authorization;
- default-deny service enforcement and mTLS remain Phase 3 work.

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

In another local shell, reuse only the management URL/token needed by the CLI:

```bash
export WTP_API_URL='http://127.0.0.1:8080'
export WTP_OPERATOR_TOKEN='<same-local-bearer-token>'
go run ./apps/wtpctl status --organization-id '<organization-uuid>'
```

The CLI does not require `DATABASE_URL`.

Use `WTP_OPERATOR_ROLE=operator` only when local registration/policy management write permission is intended.

```bash
export WTP_SPIRE_SERVER_SOCKET='/tmp/workload-trust-lab/server.sock'
go run ./apps/reconciler
```

Local example database credentials are not production credentials.
