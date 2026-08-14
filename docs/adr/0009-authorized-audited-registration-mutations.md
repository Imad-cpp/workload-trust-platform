# ADR-0009 — Authorize and transactionally audit local registration mutations

Status: Accepted  
Date: 2026-08-14

## Context

ADR-0007 introduced an authenticated single-principal local operator boundary, while deliberately withholding HTTP mutations until authorization and audit guarantees existed. Phase 2 now needs an operator workflow for changing registration desired state without weakening the rule that every security-sensitive mutation is authenticated, authorized and attributable.

A bearer credential by itself proves only possession of the configured credential. It does not answer whether the authenticated principal may write trust configuration, and a mutation followed by best-effort audit would permit unaudited state when the audit write fails.

## Decision

The current local operator boundary uses a small fail-closed role/permission mapping:

- `viewer` may read workloads and registration inventory capabilities;
- `operator` additionally receives `registrations:write`;
- a missing `WTP_OPERATOR_ROLE` defaults to `viewer`;
- unknown roles receive no permissions and are rejected by configuration/authorization checks;
- authorization is enforced server-side after authentication and before mutation request handling.

The first operator mutation routes are:

- `POST /v1/registration-rules` — create registration desired state at revision 1;
- `PATCH /v1/registration-rules/{id}` — full desired-state replacement, not JSON Merge Patch. The caller must provide `expected_revision` and stale revisions fail with `409`.

Mutation input is bounded and validated:

- `Content-Type` must be `application/json`;
- body size is limited to 64 KiB;
- unknown JSON fields and trailing JSON values are rejected;
- parent SPIFFE IDs must be canonical absolute `spiffe://` URIs;
- selector count/type/value/control-character constraints are enforced;
- X.509-SVID TTL is constrained to the product-supported range.

Every successful registration desired-state mutation and its operator audit event are committed in the **same PostgreSQL transaction**. If the audit insert fails, the registration mutation is rolled back. Audit metadata records safe state summaries (role, desired state, revisions, selector count and TTL) but not selector values or parent SPIFFE IDs.

Mutation responses intentionally omit selectors and parent SPIFFE IDs. Internal database errors are mapped to generic HTTP errors.

## Separation from reconciliation

The HTTP mutation changes PostgreSQL desired state and marks the rule `pending`. It does not directly call SPIRE. The existing reconciler remains the separate component that converges desired state into owned SPIRE entries. This preserves a clear operator/API → desired-state → reconciliation boundary.

## Security interpretation

This is server-side authorization for the **current local registration mutation surface**. It is not an enterprise/multi-user RBAC system:

- there is still one configured local operator principal per control-plane process;
- the role is configured locally, not stored in an identity/session database;
- there is no OAuth, SSO, browser session or remote management exposure;
- the HTTP listener remains loopback-only.

A later remote or multi-principal operator design must supersede the identity/session aspects without weakening the permission and transactional-audit invariants.

## Evidence

Permanent CI proves:

- unauthenticated registration mutation → `401`;
- authenticated `viewer` mutation → `403` and no registration/audit row change;
- `viewer` read remains allowed;
- `operator` create → `201`, pending revision 1 + attributed audit;
- authorized replacement → `200`, pending revision 2 + attributed audit;
- stale replacement → `409` with no revision/audit change;
- desired parent/selector material does not appear in mutation responses or audit metadata;
- an intentionally failed audit insert rolls back the registration mutation in PostgreSQL.
