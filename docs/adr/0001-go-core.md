# ADR-0001 — Use Go for the security/infrastructure core

Status: Accepted  
Date: 2026-08-14

## Context

The product interacts with workload identity, gRPC, TLS, infrastructure processes and a Go-native SPIFFE ecosystem. The core should be deployable as small static-ish binaries and support strong concurrency/networking tooling.

## Decision

Use Go for the control plane, CLI, policy/security core and infrastructure integrations. Use TypeScript only for the web console unless a concrete need justifies otherwise.

## Consequences

- the portfolio demonstrates a systems/backend language not used by the earlier projects;
- SPIFFE Go libraries integrate naturally;
- implementation must maintain clear package boundaries and avoid premature service fragmentation.
