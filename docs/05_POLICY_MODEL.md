# 05 — Policy Model

Status: Draft  
Date: 2026-08-14

## Principle

Identity answers **who is this workload?** Policy answers **what may this identity do?**

## V1 authorization tuple

A minimal policy decision is based on:

```text
source identity
destination identity
action
effect
```

V1 action begins with service connection/call semantics. Protocol-specific attributes can be added only when enforcement semantics are clear.

## Example

```yaml
source: spiffe://demo.internal/prod/orders-api
destination: spiffe://demo.internal/prod/payment-api
action: connect
effect: allow
```

Everything not explicitly allowed is denied.

## Required semantics

- policies are explicit and deterministic;
- no wildcard is introduced without documented matching semantics and tests;
- deny/allow conflict behavior must be defined before implementation;
- malformed policy cannot be activated;
- policy mutation creates a new version rather than silently rewriting historical evidence;
- active policy can be traced to an operator action.

## V1 demo matrix

| Source | Destination | Expected |
|---|---|---|
| frontend | orders-api | ALLOW |
| orders-api | payment-api | ALLOW |
| frontend | payment-api | DENY |
| payment-api | admin-api | DENY |

## Later dimensions

Not V1 requirements:

- HTTP method/path;
- port/protocol;
- namespace/environment selectors;
- workload labels;
- time windows;
- risk signals;
- federation-aware policy.
