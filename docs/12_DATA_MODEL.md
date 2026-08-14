# 12 — Data Model

Status: Phase 2 foundation implemented  
Date: 2026-08-14

Migration `db/migrations/000001_control_plane.up.sql` is the current executable schema source for the first control-plane slice. This document explains its product meaning and boundaries.

## Implemented entities

### organization

Logical ownership boundary. The schema keeps organization ownership explicit even though the current HTTP API has no multi-user tenancy/authentication model yet.

### trust_domain

Organization-owned SPIFFE trust-domain configuration record.

### workload

Logical software workload known to the product.

Current stored fields include:

- stable UUID;
- organization and trust-domain ownership;
- name/environment;
- SPIFFE ID;
- health/status observation;
- last-seen timestamp;
- created/updated timestamps.

Workload identity uniqueness is constrained both by SPIFFE ID and organization/environment/name.

### registration_rule

Desired registration state for a workload with JSON-array selectors and a revision counter. SPIRE reconciliation is not yet implemented, so this table is desired product state rather than proof that SPIRE has converged.

### access_policy

Logical policy object with lifecycle status and a pointer to an active immutable version.

### access_policy_version

Append-only policy snapshot containing:

- source SPIFFE ID;
- destination SPIFFE ID;
- action/effect;
- creator attribution string;
- change reason;
- monotonically intended version number per policy.

PostgreSQL triggers reject UPDATE and DELETE so historical versions cannot be silently rewritten.

### audit_event

Append-only security-relevant event with actor, action, target, correlation ID, metadata and timestamp. PostgreSQL triggers reject UPDATE and DELETE; the Go application repository exposes append only.

### security_event

Runtime security signal container for future denied-access, unexpected-identity and reconciliation-drift evidence.

## Planned but not yet implemented

- explicit node inventory table;
- richer observed SPIRE registration state/reconciliation checkpoints;
- operator/user/session/role tables;
- policy activation transaction workflow;
- retention/tamper-evidence beyond append-only database triggers.

## Prohibited storage

The application database must not store workload private keys. The current schema contains no workload private-key column.

## Migration rules

- migrations are reproducible from an empty database;
- rollback is tested in CI;
- destructive migration requires explicit data-loss review;
- policy/audit history must not be silently rewritten by schema changes;
- migration tests run against real PostgreSQL, not an in-memory substitute.
