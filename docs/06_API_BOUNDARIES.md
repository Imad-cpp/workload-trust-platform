# 06 — API Boundaries

Status: Phase 2 boundary  
Date: 2026-08-14

## Operator API — current slice

The current HTTP API is intentionally unauthenticated **and therefore loopback-only and read-only**.

Exposed routes:

- `GET /healthz` — process liveness;
- `GET /readyz` — PostgreSQL readiness;
- `GET /v1/workloads?organization_id=<uuid>` — organization-scoped workload inventory.

No HTTP mutation endpoint exists in this phase.

Before any non-loopback listener or state-changing operator route is introduced, the project requires an explicit operator authentication/authorization design and corresponding tests. See ADR-0006.

## HTTP behavior

- JSON responses for application routes and errors;
- stable machine-readable error codes;
- generated request correlation IDs;
- bounded readiness/repository timeouts;
- `Cache-Control: no-store`;
- `X-Content-Type-Options: nosniff`;
- internal database errors are not reflected to clients;
- unsupported methods return `405` with `Allow` metadata;
- unknown routes return a stable JSON `404`.

## Future operator resources

Planned, not currently exposed as mutation APIs:

- organizations;
- trust domains;
- nodes;
- registration rules;
- policies and versions;
- audit events;
- security events.

## SPIRE integration boundary — future

The product will reconcile desired workload/registration state with SPIRE using supported management surfaces selected during implementation. SPIRE internals are not modified.

Requirements remain:

- least-privilege integration credential;
- explicit error mapping;
- idempotent reconciliation where possible;
- drift detection;
- audit linkage from operator mutation to resulting identity-registration change.

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
