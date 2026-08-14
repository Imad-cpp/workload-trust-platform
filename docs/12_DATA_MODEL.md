# 12 — Data Model

Status: Phase 2 active schema/workflows  
Date: 2026-08-14

Migrations `000001_control_plane` and `000002_registration_reconciliation` are the executable schema source for the current control-plane slices.

## Implemented entities

### organization

Logical ownership boundary. Organization ownership remains explicit even though the current operator identity is still a single configured local principal rather than a persistent multi-user tenancy/session model.

### trust_domain

Organization-owned SPIFFE trust-domain configuration record.

### workload

Logical workload with stable UUID, organization/trust-domain ownership, name/environment, expected SPIFFE ID, status observation and timestamps.

### registration_rule

Desired SPIRE registration state for a workload.

Fields include selector JSON array, `desired_state`, monotonic revision, parent SPIFFE ID, X.509-SVID TTL, bound SPIRE entry ID, reconciliation status/timestamp and stable error code.

Mutation semantics:

- create starts at revision 1 and `pending`;
- replacement requires `expected_revision`, takes a row lock and increments revision;
- stale revisions fail without changing row/audit state;
- successful operator mutation and its audit insert share one PostgreSQL transaction;
- audit insert failure rolls back the registration mutation;
- mutation marks reconciliation pending but does not invoke SPIRE directly;
- reconciliation later changes convergence/binding state.

### access_policy / access_policy_version

Logical policy object plus immutable append-only version snapshots. PostgreSQL rejects UPDATE/DELETE of `access_policy_versions`. Operator policy activation workflow is not implemented yet.

### audit_event

Append-only security-relevant event with actor, action, target, correlation ID, metadata and timestamp. PostgreSQL rejects UPDATE/DELETE; the Go repository exposes append only.

Operator registration audits store role/state/revision/selector-count/TTL summaries but deliberately omit selector values and parent SPIFFE IDs. Reconciliation audits similarly omit parent/selector values.

### security_event

Runtime security signal container for future denied-access, unexpected-identity and reconciliation-drift evidence.

## Current operator authorization state

Roles/permissions currently live in process configuration/code, not database tables. There are no operator/user/session/role tables yet. `viewer` is the safe default role; `operator` may write registration desired state.

## Not yet implemented

- explicit node inventory table;
- persistent multi-principal operator/session/role model;
- policy mutation/activation transaction workflow;
- retention/tamper-evidence beyond append-only database triggers.

## Prohibited storage

The application database must not store workload private keys. The current schema contains no workload private-key column.

## Migration rules

Migrations remain reproducible from an empty database; rollback is tested; destructive migration requires data-loss review; security history must not be silently rewritten; tests use real PostgreSQL.
