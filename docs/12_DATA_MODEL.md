# 12 — Data Model

Status: Phase 2 active schema  
Date: 2026-08-14

Migrations `000001_control_plane` and `000002_registration_reconciliation` are the executable schema source for the current control-plane slices.

## Implemented entities

### organization

Logical ownership boundary. Organization ownership remains explicit even though the current operator mechanism is a single configured local principal rather than a multi-user tenancy model.

### trust_domain

Organization-owned SPIFFE trust-domain configuration record.

### workload

Logical software workload known to the product with stable UUID, organization/trust-domain ownership, name/environment, expected SPIFFE ID, status observation and timestamps.

### registration_rule

Desired SPIRE registration state for a workload.

Current reconciliation fields include:

- selector JSON array;
- `desired_state` (`present`/`absent`);
- revision;
- parent SPIFFE ID;
- X.509-SVID TTL;
- bound SPIRE entry ID;
- reconciliation status (`pending`/`converged`/`error`);
- last reconciliation timestamp;
- stable last error code.

The bound entry ID is unique when present. TTL is constrained by the product schema. A registration rule is product desired state; `converged` means the reconciler successfully observed/applied the current rule, not that service authorization exists.

### access_policy

Logical policy object with lifecycle status and a pointer to an active immutable version.

### access_policy_version

Append-only policy snapshot containing source/destination SPIFFE IDs, action/effect, creator attribution, change reason and version number. PostgreSQL triggers reject UPDATE and DELETE.

### audit_event

Append-only security-relevant event with actor, action, target, correlation ID, metadata and timestamp. PostgreSQL triggers reject UPDATE and DELETE; the Go repository exposes append only.

Reconciliation events use system actor `spire-reconciler`. Their metadata intentionally excludes parent SPIFFE IDs and selector values.

### security_event

Runtime security signal container for future denied-access, unexpected-identity and reconciliation-drift evidence.

## Not yet implemented

- explicit node inventory table;
- operator/user/session/role tables;
- HTTP mutation workflows for registration/policy desired state;
- policy activation transaction workflow;
- retention/tamper-evidence beyond append-only database triggers.

## Prohibited storage

The application database must not store workload private keys. The current schema contains no workload private-key column.

## Migration rules

- migrations are reproducible from an empty database and can upgrade the existing Phase 2 foundation schema;
- rollback is tested in CI;
- destructive migration requires explicit data-loss review;
- policy/audit history must not be silently rewritten by schema changes;
- migration tests run against real PostgreSQL, not an in-memory substitute.
