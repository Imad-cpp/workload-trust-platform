# ADR-0006 — Keep the operator HTTP API read-only and loopback-only before authentication

Status: Accepted  
Date: 2026-08-14

## Context

Phase 2 needs a real control-plane process and an HTTP boundary so health, readiness and workload inventory can be exercised end to end. Operator authentication and authorization have not yet been designed or implemented.

Exposing mutation endpoints or allowing an unauthenticated management listener to bind publicly would create a security boundary before the project has the controls to defend it.

## Decision

Until an explicit operator authentication/authorization ADR replaces this decision:

- the control-plane HTTP listener must bind only to loopback addresses;
- non-loopback `WTP_LISTEN_ADDR` values are rejected during configuration validation;
- the HTTP API exposes health, readiness and read-only workload inventory only;
- workload/policy/trust-domain mutation endpoints are not implemented;
- internal repository/service mutation capabilities do not imply a public HTTP mutation surface;
- stable JSON errors and generic internal-failure responses must not leak database details.

## Consequences

### Positive

- Phase 2 can validate real process/database/API behavior without pretending the operator boundary is secure;
- accidental public binding fails at startup rather than relying on deployment convention;
- future authentication work has an explicit decision point and cannot be bypassed by adding a convenient POST route.

### Limitations

- the current API is not suitable for remote multi-user operation;
- loopback is a containment control, not an authentication mechanism;
- a local attacker with sufficient host access remains outside this temporary operator-API protection model.

## Supersession requirement

Any change that permits non-loopback operator access or exposes state-changing HTTP endpoints requires an accepted operator authentication/authorization design, corresponding tests, and documentation updates in the same PR.
