# 06 — API Boundaries

Status: Phase 2 boundary  
Date: 2026-08-14

## Operator HTTP API — current slice

The management HTTP listener remains **loopback-only**. Every `/v1/*` request requires the configured bearer credential before route-level authorization is evaluated.

Current roles are intentionally small and local:

- `viewer` — read permissions;
- `operator` — read permissions plus `registrations:write`.

A missing role defaults to `viewer`; unsupported roles fail closed. This is a single configured local principal with server-side role authorization, not a remote/multi-user session or enterprise RBAC system.

### Exposed routes

- `GET /healthz` — generic process liveness; unauthenticated;
- `GET /readyz` — generic PostgreSQL readiness; unauthenticated;
- `GET /v1/workloads?organization_id=<uuid>` — authenticated/authorized workload inventory;
- `POST /v1/registration-rules` — authenticated + `registrations:write`; create desired registration state;
- `PATCH /v1/registration-rules/{id}` — authenticated + `registrations:write`; full desired-state replacement with `expected_revision` optimistic concurrency.

The `PATCH` route is **not** JSON Merge Patch. It replaces the mutable desired-state fields supplied by the API and increments revision only when `expected_revision` still matches.

### Mutation request/response behavior

- `application/json` is required;
- mutation bodies are capped at 64 KiB;
- unknown JSON fields and trailing JSON values are rejected;
- parent SPIFFE IDs, selectors and TTL are strictly validated;
- unauthorized roles are rejected before the mutation service executes;
- mutation responses contain safe summary fields and do not echo parent SPIFFE IDs or selector values;
- create returns `201` + `Location`;
- replacement returns `200`;
- stale revision returns `409`;
- internal repository/database errors remain generic.

Each successful registration mutation and its operator audit event commit in one PostgreSQL transaction. An audit failure aborts the desired-state mutation.

## SPIRE reconciliation boundary

Operator HTTP writes only PostgreSQL desired state. SPIRE mutation remains a separate reconciler boundary:

```text
Authenticated + authorized operator HTTP mutation
        |
        v
PostgreSQL registration desired state + operator audit
        |
        v
Go reconciler
        |
        v
SPIRE Server Entry API (local Unix socket)
        |
        v
SPIRE Agent / Workload API
```

Reconciliation rules remain:

- SDK/runtime aligned to SPIRE v1.15.2;
- managed entry hint `wtp-rule:<registration-rule-id>`;
- update/delete refused for foreign entries;
- `ALREADY_EXISTS` adopted only for the exact owned rule and only after full desired-state comparison;
- reconciliation errors fail rather than becoming implicit success;
- reconciliation audit metadata omits parent/selector values.

The ownership hint is a non-cryptographic controller convention inside a trusted local SPIRE management boundary.

## HTTP security/error behavior

- generated request correlation IDs;
- `Cache-Control: no-store`;
- `X-Content-Type-Options: nosniff`;
- bounded readiness/repository/mutation timeouts;
- stable machine-readable errors;
- missing/bad auth → `401` with `WWW-Authenticate`;
- authenticated but unauthorized → `403`;
- invalid JSON/input → `400`;
- unsupported media type → `415`;
- oversized mutation body → `413`;
- revision conflict → `409`;
- unknown authenticated `/v1/*` route → stable `404`.

## Future operator resources

Still planned:

- organizations/trust-domain mutation workflows;
- policy version/activation workflows;
- audit/security-event inspection APIs;
- multi-principal/session identity and authorization if/when the management surface expands beyond the local lab.

## Authorization/enforcement boundary — future Phase 3

Operator authorization above is **management authorization**, not workload service authorization.

Future service authorization inputs are expected to include verified source SPIFFE ID, destination identity/resource, requested action and active policy version. No private key or raw sensitive credential belongs in that API.
