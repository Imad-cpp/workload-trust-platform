# 06 — API Boundaries

Status: Draft  
Date: 2026-08-14

This document defines responsibilities, not final endpoint syntax.

## Operator API

Used by console and CLI. Initial resource groups:

- session/operator identity;
- organizations (single-organization implementation is acceptable internally for V1 if schema remains explicit);
- trust domains;
- nodes;
- workloads;
- registration rules;
- policies and versions;
- audit events;
- security events;
- health/readiness.

## SPIRE integration boundary

The product reconciles desired workload/registration state with SPIRE using supported management APIs/CLI integrations selected during implementation. SPIRE internals are not modified.

Requirements:

- least-privilege integration credential;
- explicit error mapping;
- idempotent reconciliation where possible;
- drift detection;
- audit linkage from operator mutation to resulting identity-registration change.

## Authorization/enforcement boundary

Inputs:

- verified source SPIFFE ID;
- destination identity/resource;
- requested action;
- active policy version.

Output:

```json
{
  "decision": "allow|deny",
  "reason": "stable-machine-readable-code",
  "policy_version": "...",
  "correlation_id": "..."
}
```

No private key or raw sensitive credential is part of this API.

## API security requirements

- authenticated operator endpoints;
- server-side authorization;
- strict validation;
- stable error envelopes;
- rate limits for externally reachable management surfaces;
- request correlation without logging credentials;
- security-critical mutation audit;
- health endpoints that do not expose sensitive internals.
