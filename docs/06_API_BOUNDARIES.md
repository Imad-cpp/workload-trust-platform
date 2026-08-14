# 06 — API Boundaries

Status: Phase 2 boundary  
Date: 2026-08-14

## Operator HTTP API — current slice

The management HTTP listener remains **loopback-only**. ADR-0007 adds authentication without opening the listener remotely.

Exposed routes:

- `GET /healthz` — process liveness; generic and unauthenticated;
- `GET /readyz` — PostgreSQL readiness; generic and unauthenticated;
- `GET /v1/workloads?organization_id=<uuid>` — authenticated organization-scoped workload inventory.

Every `/v1/*` request requires the configured bearer credential. Missing/malformed/incorrect credentials return a stable `401` response with `WWW-Authenticate` metadata. Operator authentication is required before the route is evaluated.

No HTTP mutation endpoint exists in this phase. The current bearer mechanism authenticates a single configured local principal; it is not yet a multi-role operator authorization model.

## HTTP behavior

- JSON responses for application routes and errors;
- stable machine-readable error codes;
- generated request correlation IDs;
- bounded readiness/repository timeouts;
- `Cache-Control: no-store`;
- `X-Content-Type-Options: nosniff`;
- internal database errors are not reflected to clients;
- unsupported methods return `405` with `Allow` metadata;
- unknown `/v1/*` routes require authentication before returning a stable JSON `404`.

## SPIRE reconciliation boundary — current slice

The one-shot reconciler is an internal process boundary, not an HTTP operator mutation route.

```text
PostgreSQL registration desired state
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

Rules:

- the SPIRE SDK is aligned to v1.15.2 with the reference SPIRE runtime;
- a registration rule binds to a SPIRE entry ID after convergence;
- the expected entry hint is `wtp-rule:<registration-rule-id>`;
- update/delete is refused when the fetched entry is foreign;
- `ALREADY_EXISTS` is accepted only for an entry carrying the same ownership marker, and desired state is rechecked before convergence;
- reconciliation failures remain errors rather than implicit success;
- audit metadata records safe operation/entry evidence and omits parent/selector values.

The ownership hint is not a cryptographic ownership proof. It protects against accidental cross-controller mutation inside a trusted local SPIRE management boundary.

## Future operator resources

Planned mutation/service workflows, not currently exposed as HTTP mutation APIs:

- organizations;
- trust domains;
- workloads/registration rules;
- policies and versions;
- audit/security-event inspection.

## Authorization/enforcement boundary — future Phase 3

Inputs are expected to include:

- verified source SPIFFE ID;
- destination identity/resource;
- requested action;
- active policy version.

Expected output shape:

```json
{
  "decision": "allow|deny",
  "reason": "stable-machine-readable-code",
  "policy_version": "...",
  "correlation_id": "..."
}
```

No private key or raw sensitive credential belongs in this API.
