# ADR-0006 — Keep the operator HTTP API read-only and loopback-only before authentication

Status: Superseded by ADR-0007 for authentication; loopback/read-only restrictions remain active  
Date: 2026-08-14

## Context

Phase 2 needed a real control-plane process and an HTTP boundary before operator authentication/authorization had been implemented. Exposing mutation endpoints or allowing an unauthenticated management listener to bind publicly would have created a security boundary before the project had the controls to defend it.

## Historical decision

The initial operator API was read-only and loopback-only while operator authentication/authorization was absent:

- non-loopback `WTP_LISTEN_ADDR` values were rejected;
- health, readiness and read-only workload inventory were the only HTTP operations;
- workload/policy/trust-domain mutation endpoints were not implemented;
- internal repository/service mutation capabilities did not imply a public HTTP mutation surface;
- stable JSON errors did not expose database details.

## Supersession

ADR-0007 adds local operator authentication. It intentionally retains the loopback-only listener and the absence of HTTP mutation endpoints. A later ADR is still required before remote access or state-changing operator HTTP routes are introduced.
