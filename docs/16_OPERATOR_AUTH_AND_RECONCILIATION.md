# 16 — Local Operator Authentication and SPIRE Reconciliation Evidence

Status: Phase 2 implemented slice  
Date: 2026-08-14

## What this slice proves

### Local operator authentication

- all `/v1/*` routes pass through a bearer authenticator;
- missing/malformed/wrong credentials fail with stable `401` behavior;
- configured credentials shorter than 32 bytes are rejected;
- the authenticator compares SHA-256 digests with a constant-time comparison;
- an explicit local operator actor ID is attached to authenticated requests;
- the listener remains loopback-only;
- health/readiness remain generic and unauthenticated;
- no HTTP mutation endpoint is added.

### Desired-state reconciliation

The permanent integration path starts from fresh PostgreSQL and a fresh Phase 1 SPIRE lab, then proves:

1. PostgreSQL registration desired state creates an owned SPIRE entry through the official Entry API.
2. A Docker workload matching the desired selectors obtains the expected X.509-SVID through the Workload API.
3. Selector/TTL drift is reconciled in place on the same SPIRE entry ID.
4. The newly selected workload obtains the identity and the previously selected workload no longer does.
5. A deliberately created foreign entry with the same identity/parent/selectors is returned as an existing SPIRE entry but is not adopted or mutated because its hint is not `wtp-rule:<rule-id>`.
6. Desired `absent` removes the owned entry and a fresh workload can no longer obtain the lab identity.
7. Create/update/delete/ownership-conflict operations leave append-only audit evidence.
8. Reconciliation audit metadata contains no `join_token` parent material.

## Failure semantics

The reconciler fails closed when desired state is invalid, when SPIRE operations fail, or when ownership does not match. It records stable rule error codes rather than marking those cases converged.

An `ALREADY_EXISTS` response is not automatically trusted. The existing entry must carry the exact rule ownership marker. Even an owned existing entry is compared with complete desired identity/parent/selectors/TTL/hint state and drift is corrected before binding.

## Security interpretation

`wtp-rule:<rule-id>` prevents accidental adoption/mutation of entries outside the controller's declared ownership set. It is not a cryptographic ownership mechanism and cannot protect against an attacker who already has privileged SPIRE management or product-database control.

The local bearer boundary is likewise intentionally limited. It is stronger than anonymous loopback access, but it is not the final remote operator authentication/authorization design.

## Claim boundary

This slice proves local operator authentication and workload-registration reconciliation. It **does not prove service authorization**, default-deny policy enforcement between workloads, production node attestation, remote operator security, high availability or production readiness.
