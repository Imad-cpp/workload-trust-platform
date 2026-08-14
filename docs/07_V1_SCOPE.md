# 07 — V1 Scope

Status: Proposed baseline  
Date: 2026-08-14

## V1 product statement

Secretless workload identity and default-deny service authorization for a Docker/Linux environment using SPIFFE/SPIRE.

## Must ship

### Identity lab

- SPIRE Server and Agent in a reproducible local environment;
- at least four demo workloads;
- runtime-derived workload attestation;
- predictable SPIFFE IDs;
- X.509-SVID retrieval and rotation evidence;
- no embedded long-lived workload credential for service identity.

### Control plane

- Go API;
- PostgreSQL schema/migrations;
- workload inventory;
- registration rule management/reconciliation;
- access policy CRUD with version history;
- audit records;
- health/readiness.

### Enforcement

- mutual workload authentication;
- deterministic policy evaluation;
- default deny;
- expected allow/deny demo matrix;
- explainable decision reason.

### Console

- workload inventory;
- identity status;
- policy table/editor;
- audit timeline;
- denied-access/security event visibility;
- complete loading/empty/error/permission states for V1 screens.

### CLI

At minimum:

- inspect status;
- list workloads;
- inspect identity/registration state;
- list/validate policies;
- run demo verification or diagnostics.

### Quality/security

- Go unit tests;
- policy table tests;
- database migration/rollback tests;
- real SPIRE integration tests;
- end-to-end allow/deny tests;
- failure tests;
- secret hygiene;
- dependency/security audit;
- container scanning;
- static analysis;
- browser smoke/E2E for critical console paths;
- exact-commit release verification.

## Explicitly not V1

- Kubernetes;
- cloud attestation;
- multi-region HA;
- federation;
- enterprise SSO;
- billing;
- AI;
- custom cryptography;
- full general-purpose service mesh;
- general-purpose secrets-manager replacement.
