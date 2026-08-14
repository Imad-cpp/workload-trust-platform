# ADR-0002 — Build workload identity on SPIFFE/SPIRE

Status: Accepted  
Date: 2026-08-14

## Context

Inventing a proprietary workload-identity protocol or PKI would add security risk without product value. The project needs portable workload identities, short-lived credentials and runtime identity delivery.

## Decision

Use SPIFFE as the workload identity standard and SPIRE as the V1 identity runtime. X.509-SVIDs are the primary V1 service-authentication credential.

## Consequences

- the product focuses on operator control, policy, visibility and integrations rather than custom cryptography;
- SPIRE remains an external runtime dependency;
- its security/trust assumptions must be represented faithfully in the product threat model;
- management integration must use supported SPIRE surfaces rather than patching SPIRE internals.
