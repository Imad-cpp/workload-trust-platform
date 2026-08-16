# 06 — API Boundaries

Status: Phase 2 boundary  
Date: 2026-08-16

## Operator HTTP API — current slice

The management HTTP listener remains **loopback-only**. Every `/v1/*` request requires the configured bearer credential before route-level authorization is evaluated.

Current roles are intentionally small and local:

- `viewer` — management read permissions, including diagnostics;
- `operator` — reads plus current registration/policy mutation permissions.

A missing role defaults to `viewer`; unsupported roles fail closed. This remains a single configured local principal, not a remote/multi-user session or enterprise RBAC system.

### Exposed routes

- `GET /healthz` — generic liveness; unauthenticated;
- `GET /readyz` — generic PostgreSQL readiness; unauthenticated;
- `GET /v1/workloads?organization_id=<uuid>` — authenticated/authorized inventory;
- `GET /v1/diagnostics?organization_id=<uuid>` — requires `diagnostics:read`; aggregate organization status only;
- `POST /v1/registration-rules` — requires `registrations:write`;
- `PATCH /v1/registration-rules/{id}` — requires `registrations:write`; full desired-state replacement with `expected_revision`;
- `POST /v1/policies` — requires `policies:write`; creates a draft policy plus immutable version 1;
- `POST /v1/policies/{id}/versions` — requires `policies:write`; appends an immutable version using `expected_revision`;
- `POST /v1/policies/{id}/activate` — requires distinct `policies:activate`; selects an owned valid version using `expected_revision`.

### Diagnostics response boundary

`GET /v1/diagnostics` exposes aggregate counts and a latest-audit timestamp only:

- workload total/healthy/degraded/offline;
- registration pending/converged/error;
- policy draft/active/disabled;
- audit/security-event counts;
- latest audit timestamp.

It does not expose registration selectors/parent SPIFFE IDs, policy source/destination identities/change reasons, audit/security metadata or credentials. A missing organization returns a stable `404`; internal repository errors remain generic.

### Mutation request/response behavior

- `application/json` is required;
- mutation bodies are capped at 64 KiB;
- unknown fields and trailing JSON values are rejected;
- relevant SPIFFE IDs and mutation fields are strictly validated;
- unauthorized roles are rejected before mutation services execute;
- internal repository/database errors remain generic;
- stale revisions return `409` without lost-update state;
- policy responses contain safe summary fields and do not echo source/destination SPIFFE IDs or change-reason text.

Registration mutation + audit and policy create/version/activation + audit each commit in the **same PostgreSQL transaction**. Forced audit failures are tested to roll the corresponding state change back.

## CLI boundary

`wtpctl` is a read-only client of this same management API. It does not connect directly to PostgreSQL.

Current commands:

```text
wtpctl health
wtpctl ready
wtpctl workloads --organization-id <uuid>
wtpctl status --organization-id <uuid>
```

For this Phase 2 boundary:

- `WTP_API_URL` must be local plain HTTP to `localhost` or a loopback IP;
- userinfo/path/query/fragment on the base URL are rejected;
- redirects are refused;
- authenticated commands use `WTP_OPERATOR_TOKEN` and the existing server-side read authorization;
- `DATABASE_URL` is not required or consumed by the CLI;
- requests have a five-second timeout;
- responses over 1 MiB are rejected without partial output;
- non-2xx CLI errors expose safe HTTP status/error code rather than raw server body.

This does not approve remote plaintext management traffic. The CLI remains inside the same local management trust boundary as the loopback server.

## Policy-management boundary

Policy management is not policy enforcement.

```text
Authenticated + authorized operator
        |
        v
Policy create / immutable version append / activation
        |
        v
PostgreSQL active_version_id + append-only history + audit
        |
        v
[Phase 3 enforcement boundary — not implemented yet]
```

V1 management policy action is `connect`; effects are `allow|deny`; the activation candidate must belong to the target policy and stored immutable content is revalidated before activation. `active_version_id` is selected desired state only and grants no service access by itself.

## SPIRE reconciliation boundary

Registration HTTP writes PostgreSQL desired state. SPIRE mutation remains a separate reconciler boundary through the v1.15.2 Entry API over the local Unix socket. Managed entry hint is `wtp-rule:<registration-rule-id>`; foreign entries fail closed. The marker is a non-cryptographic ownership convention inside a trusted local management boundary.

## HTTP security/error behavior

- generated request correlation IDs;
- `Cache-Control: no-store`;
- `X-Content-Type-Options: nosniff`;
- bounded operation timeouts;
- missing/bad auth → `401`;
- authenticated but unauthorized → `403`;
- invalid input → `400`;
- oversized mutation body → `413`;
- unsupported media type → `415`;
- stale revision → `409`;
- unknown authenticated `/v1/*` route → stable `404`.

## Future management resources

Still planned: organization/trust-domain mutation workflows, full audit/security-event inspection APIs and any multi-principal/session identity model chosen before remote management.

## Authorization/enforcement boundary — Phase 3

Operator authorization above protects **management operations**. It is not workload service authorization.

Phase 3 must consume a verified source SPIFFE identity, destination identity/resource, requested `connect` action and selected active policy state; enforce deterministic default-deny semantics; and fail closed on missing/corrupt policy state. No private key or raw management credential belongs in that decision API.
