# ADR-0010 — Transactionally audited policy versioning and activation

Status: Accepted  
Date: 2026-08-16

## Context

Phase 2 needs a management workflow for service-access policy state before Phase 3 can enforce decisions. Policy history must remain attributable and immutable, malformed policy must not become active, and concurrent operator writes must not silently overwrite one another.

The management surface is still loopback-only and uses the ADR-0007 local bearer principal with ADR-0009-style server-side role authorization. This decision concerns **policy management state**, not workload service enforcement.

## Decision

Use `access_policies` as the mutable policy envelope and `access_policy_versions` as immutable append-only snapshots.

For the current V1 policy model:

- the only defined action is `connect`;
- effects are canonical `allow` or `deny`;
- source and destination identities must be canonical SPIFFE IDs;
- wildcard semantics are not introduced;
- creating a policy creates immutable version 1 and leaves the policy in `draft`;
- appending policy content always creates a new immutable version;
- `access_policies.revision` provides optimistic concurrency for append and activation operations;
- policy activation requires the distinct `policies:activate` permission;
- an activation candidate must belong to the target policy and is revalidated from stored immutable data immediately before activation;
- successful policy create/version/activation and its operator audit insert commit in the **same PostgreSQL transaction**;
- an audit insertion failure aborts the policy state change;
- policy API responses and audit summaries omit source/destination SPIFFE IDs and change-reason text.

An active policy version is the control plane's selected desired policy state. **Activation does not mean that network or service authorization is enforced.** Verified workload identity extraction, default-deny evaluation and allow/deny enforcement remain Phase 3 responsibilities.

## Authorization

Current local permissions are:

- `policies:read` — viewer/operator;
- `policies:write` — operator;
- `policies:activate` — operator.

`policies:activate` is intentionally distinct from `policies:write` so activation privilege can be narrowed later without changing the API model.

## Concurrency and failure behavior

Append and activation require `expected_revision`. A stale revision fails with conflict and does not add policy/audit state.

Activation fails when:

- the target policy or requested version does not exist;
- the version belongs to another policy;
- the stored version does not satisfy the current V1 validation rules;
- the expected policy revision is stale;
- audit insertion or transaction commit fails.

No failure becomes an implicit activation.

## Evidence

Permanent CI proves:

- migration `000003_policy_revision` participates in apply/rollback/re-apply;
- policy versions reject UPDATE/DELETE;
- viewer policy mutation is denied before state change;
- policy create, version append, stale conflict and activation behavior through the live HTTP process;
- foreign-version activation is refused;
- malformed directly seeded stored policy cannot be activated;
- forced audit failures roll back version append and activation;
- policy response/audit evidence does not expose rule identity/change-reason material;
- Go race-detector and real PostgreSQL tests cover the management implementation.

## Consequences

### Positive

- policy history remains immutable and attributable;
- activation is explicit rather than an accidental side effect of editing;
- stale operator writes fail instead of silently overwriting newer state;
- the management model is ready to feed a later deterministic enforcement engine.

### Limits

- there is still one configured local management principal per process;
- there is no remote management/session model;
- there is no policy evaluation cache or enforcement point yet;
- active policy state alone grants no workload access;
- default-deny service authorization remains unimplemented until Phase 3.
