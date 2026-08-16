# 05 — Policy Model

Status: Phase 2 management model implemented; enforcement pending  
Date: 2026-08-16

## Principle

Identity answers **who is this workload?** Policy answers **what may this identity do?**

Policy management and policy enforcement are separate boundaries. Phase 2 now stores, versions and activates policy desired state. Phase 3 will evaluate that state against verified workload identities and enforce decisions.

## V1 authorization tuple

The V1 policy tuple is deliberately narrow:

```text
source identity
destination identity
action = connect
effect = allow | deny
```

Protocol-specific attributes are not accepted until their enforcement semantics are defined and tested.

## Example

```yaml
source: spiffe://demo.internal/prod/orders-api
destination: spiffe://demo.internal/prod/payment-api
action: connect
effect: allow
```

The intended Phase 3 enforcement model is default-deny: anything not explicitly allowed is denied. **Default-deny service enforcement is not implemented in Phase 2.**

## Implemented management semantics

- source/destination identities must be canonical SPIFFE IDs;
- V1 accepts only the `connect` action;
- effects are canonical `allow` or `deny`;
- no wildcard matching exists;
- policy creation produces immutable version 1 and a draft policy;
- changing policy content appends a new immutable version rather than editing history;
- `access_policy_versions` rejects UPDATE/DELETE at the database layer;
- append/activation require optimistic `expected_revision`;
- activation requires a version owned by the target policy;
- stored version content is revalidated before activation;
- policy activation and its attributed operator audit commit in the same PostgreSQL transaction;
- audit failure rolls activation/version mutation back;
- active policy can be traced to an operator action.

`active_version_id` means **selected desired policy state**, not proven service authorization.

## V1 demo matrix for Phase 3

| Source | Destination | Expected |
|---|---|---|
| frontend | orders-api | ALLOW |
| orders-api | payment-api | ALLOW |
| frontend | payment-api | DENY |
| payment-api | admin-api | DENY |

These ALLOW/DENY outcomes remain an enforcement acceptance target; the current Phase 2 implementation does not claim them yet.

## Later dimensions

Not V1 requirements:

- HTTP method/path;
- port/protocol;
- namespace/environment selectors;
- workload labels;
- time windows;
- risk signals;
- federation-aware policy.
