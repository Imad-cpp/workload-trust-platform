# ADR-0008 — Reconcile owned SPIRE registration entries through the public Entry API

Status: Accepted  
Date: 2026-08-14

## Context

PostgreSQL stores the product's desired registration state while SPIRE is the workload-identity runtime. Phase 2 needs a deterministic bridge between those systems without modifying SPIRE internals or treating every SPIRE entry visible to the server as product-owned state.

The SPIRE v1.15.2 public Entry API supports get/create/update/delete operations and is available over the local SPIRE Server management socket. Batch create can return an existing similar entry, so reconciliation must distinguish safe crash recovery from accidental adoption of an entry owned by another controller/operator.

## Decision

- the reconciler uses `spiffe/spire-api-sdk` v1.15.2, aligned with the lab SPIRE runtime;
- the Phase 2 reference integration connects only to an absolute local Unix socket for the SPIRE Server management API;
- PostgreSQL `registration_rules` are desired state; SPIRE entries are observed/runtime state;
- an entry managed for a registration rule carries the hint `wtp-rule:<registration-rule-id>`;
- a bound SPIRE entry is fetched and its ownership hint is checked before update or delete;
- an `ALREADY_EXISTS` create result is adopted only when the returned entry carries the exact ownership hint for that rule;
- an owned existing entry is compared with the complete desired SPIFFE ID, parent, selectors, X.509-SVID TTL and hint before it is marked converged; drift is updated in place;
- a foreign entry is never adopted, updated or deleted and the rule records `ownership_mismatch`;
- reconciliation mutations produce append-only audit evidence before the rule is marked converged;
- reconciliation audit metadata records the operation and SPIRE entry ID only. It deliberately excludes selectors and parent SPIFFE IDs because the Phase 1 join-token agent ID embeds sensitive bootstrap material;
- one rule failure does not silently convert another rule into success; the run reports failures and exits non-zero when any rule fails.

## Ownership boundary

The `wtp-rule:` hint is an ownership **convention inside the trusted SPIRE management boundary**, not a cryptographic ownership proof. A sufficiently privileged SPIRE administrator can forge or change it. The control prevents accidental cross-controller mutation and makes ownership checks explicit; it does not defend against a malicious administrator who already controls the SPIRE management API or product database.

## Failure behavior

- missing/invalid desired parent, selectors or TTL fails the rule closed;
- unavailable SPIRE management operations mark the rule error rather than claiming convergence;
- stale bindings can be repaired on a later run;
- an owned entry left in SPIRE after a process crash can be rebound through `ALREADY_EXISTS`, but drift is corrected before convergence;
- desired `absent` deletes only a currently bound entry whose ownership hint still matches the rule.

## Consequences

### Positive

- desired state is explicit and auditable;
- reconciliation is idempotent across normal crash/retry scenarios;
- foreign SPIRE entries are protected from accidental product mutations;
- workload-identity behavior can be proven end to end from PostgreSQL through SPIRE to the Workload API.

### Limitations

- the current reconciler is a one-shot local process, not an HA controller;
- the local SPIRE management socket is highly privileged and requires host-level protection;
- node-attestation hardening and production bootstrap are outside this lab decision;
- this ADR does not prove service authorization or mTLS enforcement.
