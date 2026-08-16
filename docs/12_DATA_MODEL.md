# 12 — Data Model

Status: Phase 2 active schema/workflows  
Date: 2026-08-16

Migrations `000001_control_plane`, `000002_registration_reconciliation` and `000003_policy_revision` are the executable schema source for the current control-plane slices.

## Implemented entities

### organization

Logical ownership boundary. Organization ownership remains explicit even though operator identity is still a single configured local principal rather than a persistent multi-user tenancy/session model.

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
- audit insert failure rolls the registration mutation back;
- mutation marks reconciliation pending but does not invoke SPIRE directly.

### access_policy

Mutable policy envelope with:

- organization ownership;
- unique organization/name pair;
- lifecycle status (`draft`, `active`, `disabled`);
- optional `active_version_id`;
- optimistic `revision` added by migration `000003_policy_revision` and defaulting to 1;
- timestamps.

`active_version_id` is selected desired policy state. It does not prove that a service authorization decision is being enforced.

### access_policy_version

Immutable append-only policy snapshot containing source/destination SPIFFE IDs, `connect` action, `allow|deny` effect, creator attribution, change reason, version number and timestamp.

Semantics:

- policy creation inserts immutable version 1;
- changing policy content appends another row instead of rewriting history;
- PostgreSQL triggers reject UPDATE and DELETE;
- appending a version requires matching `expected_revision` and advances the parent policy revision;
- activation candidate must belong to the target policy;
- stored immutable content is revalidated before activation;
- activation changes only the policy envelope (`active_version_id`, status, revision), never the historical version row;
- policy create/version/activation and their audit record share the same PostgreSQL transaction;
- audit failure rolls state back.

### audit_event

Append-only security-relevant event with actor, action, target, correlation ID, metadata and timestamp. PostgreSQL rejects UPDATE/DELETE; the Go repository exposes append only.

Registration/policy audit summaries deliberately omit raw selector/parent values and policy source/destination SPIFFE IDs/change-reason text.

### security_event

Runtime security signal container for future denied-access, unexpected-identity and reconciliation-drift evidence.

## Current operator authorization state

Roles/permissions live in process configuration/code, not database tables. There are no operator/user/session/role tables yet.

- `viewer` is the safe default and receives current read permissions;
- `operator` receives registration writes plus policy write/activation permissions;
- `policies:activate` is distinct from `policies:write`.

## Not yet implemented

- explicit node inventory table;
- persistent multi-principal operator/session/role model;
- service authorization decision/enforcement state;
- retention/tamper-evidence beyond append-only database triggers.

## Prohibited storage

The application database must not store workload private keys. The current schema contains no workload private-key column.

## Migration rules

Migrations are reproducible from an empty database and are applied in order through `000003`. Rollback runs in reverse order. Permanent CI proves apply/rollback/re-apply and semantically verifies the policy revision default after re-apply. Destructive migration requires data-loss review; security history must not be silently rewritten; tests use real PostgreSQL.
