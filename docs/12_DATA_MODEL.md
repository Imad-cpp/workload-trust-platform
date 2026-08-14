# 12 — Data Model

Status: Conceptual  
Date: 2026-08-14

This is a domain model, not a final SQL schema.

## Core entities

### organization

Logical ownership boundary. V1 may operate with one organization while keeping ownership explicit in persistent data.

### trust_domain

Represents a configured SPIFFE trust domain and integration state.

### node

Observed/managed execution node associated with identity runtime state.

### workload

Logical software workload known to the product.

Key concepts:

- stable product ID;
- display name;
- environment;
- expected SPIFFE ID;
- status/last observation;
- organization ownership.

### registration_rule

Operator-desired relationship between attestation selectors and expected workload identity. Must be versionable/auditable or otherwise produce sufficient mutation evidence.

### access_policy

Logical policy object.

### access_policy_version

Immutable policy snapshot including source, destination, action/effect and activation metadata.

### audit_event

Security-relevant operator/system event with actor, action, target, correlation ID, timestamp and safe change metadata.

### security_event

Runtime security signal such as denied access, unexpected identity state or reconciliation drift.

## Prohibited storage

The application database must not be used to store workload private keys.

## Migration rules

- migrations are forward reproducible from an empty database;
- rollback implications are documented and tested;
- destructive migration requires explicit data-loss review;
- security policy history must not be silently rewritten by schema changes.
