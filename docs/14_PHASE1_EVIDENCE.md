# 14 — Phase 1 Evidence

Status: Complete  
Date: 2026-08-14

## Claim

Phase 1 establishes a reproducible **Docker/Linux workload identity lab** backed by a real SPIRE Server and Agent. It proves workload attestation and short-lived X.509-SVID lifecycle behavior. It does not prove service authorization or production readiness.

## Permanent CI evidence

`.github/workflows/identity-lab.yml` runs two independent jobs on pull requests and pushes to `main`:

### SPIRE config validation

- fetches SPIRE v1.15.2 from the upstream release;
- verifies the pinned upstream SHA-256;
- validates the server configuration with the SPIRE binary;
- validates the agent configuration with the SPIRE binary.

### Docker workload attestation

A complete ephemeral runtime is created on an Ubuntu GitHub-hosted runner and must prove:

1. SPIRE Server starts and passes health checking.
2. A one-time join token attests the Linux SPIRE Agent.
3. Four Docker-label registration entries are created.
4. `frontend` receives only `spiffe://workload-trust.test/lab/frontend`.
5. `orders-api` receives only its expected identity.
6. `payment-api` receives only its expected identity.
7. `admin-api` receives only its expected identity.
8. `untrusted` receives no registered lab identity.
9. `frontend-wrong-environment` receives no registered lab identity.
10. Two distinct recreated `frontend` containers independently obtain the same logical identity.
11. A running `frontend` workload receives at least two X.509 context updates with distinct certificate expiration timestamps, proving automatic SVID rotation.
12. The runtime is torn down after success or failure.

## Security evidence boundaries

The tests intentionally do not claim more than they verify:

- Docker labels are sufficient for the controlled lab, not a high-assurance production attestation source against an attacker who controls Docker workload creation.
- `join_token` plus `insecure_bootstrap` are development/bootstrap choices only.
- a valid SVID authenticates identity but grants no service authorization by itself.
- root compromise of the Docker/SPIRE host is outside the V1 lab protection boundary.
- no custom cryptography is implemented.
- no workload private key is emitted into normal CI output or committed state.

## Phase 1 exit

Phase 1 is considered complete when both permanent workflows pass on the exact merge candidate and again on the resulting `main` merge commit. The exact GitHub run IDs and commit SHAs are recorded in the pull-request discussion/handoff rather than hard-coded into this durable design document.
