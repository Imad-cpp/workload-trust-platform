# 06 — API Boundaries

Status: Phase 2 boundary  
Date: 2026-08-16

## Operator HTTP API — current slice

The management HTTP listener remains **loopback-only**. Every `/v1/*` request requires the configured bearer credential before route-level authorization is evaluated.

Current roles are intentionally small and local:

- `viewer` — management read permissions;
- `operator` — reads plus current registration/policy mutation permissions.

A missing role defaults to `viewer`; unsupported roles fail closed. This remains a single configured local principal, not a remote/multi-user session or enterprise RBAC system.

### Exposed routes

- `GET /healthz` — generic liveness; unauthenticated;
- `GET /readyz` — generic PostgreSQL readiness; unauthenticated;
- `GET /v1/workloads?organization_id=<uuid>` — authenticated/authorized inventory;
- `POST /v1/registration-rules` — requires `registrations:write`;
- `PATCH /v1/registration-rules/{id}` — requires `registrations:write`; full desired-state replacement with `expected_revision`;
- `POST /v1/policies` — requires `policies:write`; creates a draft policy plus immutable version 1;
- `POST /v1/policies/{id}/versions` — requires `policies:write`; appends an immutable version using `expected_revision`;
- `POST /v1/policies/{id}/activate` — requires distinct `policies:activate`; selects an owned valid version using `expected_revision`.

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

Current management rules:

- V1 action is `connect` only;
- effects are `allow` or `deny`;
- activation candidate must belong to the target policy;
- stored immutable policy content is revalidated before activation;
- `active_version_id` is selected desired state only;
- active state does not grant network/service access by itself.

## SPIRE reconciliation boundary

Registration HTTP writes PostgreSQL desired state. SPIRE mutation remains a separate reconciler boundary:

```text
PostgreSQL registration desired state + operator audit
        |
        v
Go reconciler
        |
        v
SPIRE Server Entry API (local Unix socket)
```

Managed entry hint is `wtp-rule:<registration-rule-id>`; foreign entries fail closed. The marker is a non-cryptographic ownership convention inside a trusted local management boundary.

## HTTP security/error behavior

- generated request correlation IDs;
- `Cache-Control: no-store`;
- `X-Content-Type-Options: nosniff`;
- bounded operation timeouts;
- missing/bad auth → `401`;
- authenticated but unauthorized → `403`;
- invalid input → `400`;
- oversized body → `413`;
- unsupported media type → `415`;
- stale revision → `409`;
- unknown authenticated `/v1/*` route → stable `404`.

## Future management resources

Still planned: organization/trust-domain mutation workflows, audit/security-event inspection APIs and any multi-principal/session identity model chosen before remote management.

## Authorization/enforcement boundary — Phase 3

Operator authorization above protects **management operations**. It is not workload service authorization.

Phase 3 must consume a verified source SPIFFE identity, destination identity/resource, requested `connect` action and selected active policy state; enforce deterministic default-deny semantics; and fail closed on missing/corrupt policy state. No private key or raw management credential belongs in that decision API.
