# 04 — Identity Model

Status: Draft  
Date: 2026-08-14

## Standard

SPIFFE defines the workload identity namespace and SVID mechanism. SPIRE is the planned V1 runtime implementation.

## V1 identity form

Primary workload authentication uses X.509-SVIDs.

Example identities:

```text
spiffe://demo.internal/prod/frontend
spiffe://demo.internal/prod/orders-api
spiffe://demo.internal/prod/payment-api
spiffe://demo.internal/prod/admin-api
```

`demo.internal` is illustrative only; the final V1 trust-domain value is an implementation decision.

## Identity naming requirements

A V1 workload identity must be:

- stable across ordinary container restarts when it represents the same logical workload;
- derived from operator-controlled registration/attestation policy;
- environment-aware;
- non-secret;
- human-readable enough for audit and policy review;
- independent from ephemeral IP addresses.

## Identity lifecycle

```text
Node attested
  -> workload starts
  -> local agent observes workload attributes
  -> registration selectors match
  -> SVID issued
  -> workload retrieves SVID through Workload API
  -> SVID rotates before expiry
  -> workload termination / registration removal ends entitlement
```

## Private-key handling

The product control plane does not receive workload private keys. Workload identity delivery remains local to the SPIFFE/SPIRE mechanism wherever possible.

## V1 attestation target

Docker/Linux reference workloads. Selectors should be based on runtime-verifiable properties such as process or container metadata supported by the chosen SPIRE plugins. Exact selectors are defined during the identity lab and captured in an ADR.

## Out of scope V1

- Kubernetes service-account selectors;
- cloud instance identity attestors;
- TPM-backed attestation;
- cross-trust-domain federation;
- JWT-SVID as the primary service-authentication mechanism.
