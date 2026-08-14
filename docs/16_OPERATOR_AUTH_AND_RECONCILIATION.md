# 16 — Local Operator Authentication and SPIRE Reconciliation Evidence

Status: Phase 2 historical implemented slice; extended by `17_OPERATOR_MUTATION_EVIDENCE.md`  
Date: 2026-08-14

## What this slice originally proved

### Local operator authentication

- all `/v1/*` routes pass through a bearer authenticator;
- missing/malformed/wrong credentials fail with stable `401` behavior;
- configured credentials shorter than 32 bytes are rejected;
- the authenticator compares SHA-256 digests with a constant-time comparison;
- an explicit local operator actor ID is attached to authenticated requests;
- the listener remains loopback-only;
- health/readiness remain generic and unauthenticated.

At the time this slice merged, it intentionally added **no HTTP mutation endpoint**. Registration mutations were introduced later only after ADR-0009 added server-side authorization and transactional audit; current mutation evidence is recorded in `docs/17_OPERATOR_MUTATION_EVIDENCE.md`.

### Desired-state reconciliation

The permanent integration path starts from fresh PostgreSQL and a fresh Phase 1 SPIRE lab, then proves:

1. PostgreSQL registration desired state creates an owned SPIRE entry through the official Entry API.
2. A matching Docker workload obtains the expected X.509-SVID through the Workload API.
3. Selector/TTL drift is reconciled in place on the same SPIRE entry ID.
4. The newly selected workload obtains the identity and the previously selected workload no longer does.
5. A deliberately created foreign entry with the same identity/parent/selectors is not adopted or mutated because its hint is not `wtp-rule:<rule-id>`.
6. Desired `absent` removes the owned entry and a fresh workload can no longer obtain the lab identity.
7. Create/update/delete/ownership-conflict operations leave append-only audit evidence.
8. Reconciliation audit metadata contains no `join_token` parent material.

## Failure semantics

The reconciler fails closed when desired state is invalid, SPIRE operations fail, or ownership does not match. `ALREADY_EXISTS` is not automatically trusted: the entry must carry the exact rule ownership marker and complete desired state is checked/corrected before binding.

## Security interpretation

`wtp-rule:<rule-id>` is a non-cryptographic ownership convention and cannot defend against an attacker who already controls privileged SPIRE management or the product database.

The local bearer boundary is intentionally limited and has since been extended with the fail-closed `viewer`/`operator` authorization layer in ADR-0009. It is still not the final remote/multi-user management identity design.

## Claim boundary

This evidence proves local operator authentication and workload-registration reconciliation. It **does not prove service authorization**, default-deny policy enforcement between workloads, production node attestation, remote operator security, high availability or production readiness.
