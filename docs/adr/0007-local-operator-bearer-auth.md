# ADR-0007 — Authenticate the local operator API with a high-entropy bearer credential

Status: Accepted  
Date: 2026-08-14

## Context

ADR-0006 deliberately kept the first operator HTTP surface unauthenticated, read-only and loopback-only until an explicit authentication design existed. Phase 2 now needs an attributable operator identity for management reads and for future audited workflows, but the project is not yet ready to claim a remote multi-user session/SSO boundary.

Opening the listener beyond the host or introducing state-changing HTTP routes at the same time would combine too many trust-boundary changes in one step.

## Decision

For the current Phase 2 local operator boundary:

- `/v1/*` requires an `Authorization: Bearer <credential>` header;
- the configured bearer credential must contain at least 32 bytes/characters of high-entropy secret material;
- the authenticator stores a SHA-256 digest rather than the configured plaintext credential and compares fixed-size digests with a constant-time comparison;
- the configured `WTP_OPERATOR_ID` becomes the attributable operator principal for the authenticated request;
- the listener remains loopback-only and non-loopback configuration is still rejected at startup;
- `/healthz` and `/readyz` remain unauthenticated but expose only generic liveness/readiness state;
- no HTTP mutation endpoint is introduced by this ADR;
- the credential and database URL must not be intentionally logged.

SHA-256 here is not a password-hashing scheme and is not presented as one. The credential is required to be a high-entropy bearer secret; hashing is used to avoid retaining the configured plaintext in the authenticator and to compare fixed-length values in constant time.

## Consequences

### Positive

- operator reads are no longer anonymously available to every local process that can reach the listener;
- requests can carry an explicit operator principal into later audit/authorization layers;
- remote exposure and mutation remain blocked while those controls are still incomplete;
- authentication tests can cover malformed, missing and incorrect credentials independently of HTTP business logic.

### Limitations

- a static local bearer credential is an interim single-operator mechanism, not a browser session, OAuth, SSO or role model;
- environment/process access may expose the configured secret to a sufficiently privileged local attacker;
- authentication alone is not an authorization model;
- loopback containment remains mandatory in this phase.

## Supersession

A remote or multi-user operator surface should supersede this ADR with an explicit session/identity, authorization, rotation/revocation and transport-security design.
