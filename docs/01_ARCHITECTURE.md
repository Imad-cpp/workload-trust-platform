# 01 — Architecture

Status: Draft foundation  
Date: 2026-08-14

## Architectural goal

Keep V1 operationally simple while preserving clear domain boundaries. The first implementation is a modular control plane, not a fleet of independent microservices.

## System context

```text
Operator
   |
   v
Web Console / CLI
   |
   v
Control Plane API -------------------- PostgreSQL
   |                                      |
   |                                      +-- desired product state
   |                                      +-- policy versions
   |                                      +-- audit records
   |
   +--> SPIRE integration --> SPIRE Server
   |                              |
   |                          SPIRE Agents
   |                              |
   |                          Workloads
   |
   +--> Authorization policy distribution / enforcement integration
```

## Core components

### Operator console

Next.js + React + TypeScript application for operator workflows. It is not part of the workload trust path.

Initial views:

- overview;
- nodes;
- workloads;
- identities;
- policies;
- audit log;
- security events;
- settings.

### Control plane

Go application responsible for product state and operator-facing APIs.

Logical modules:

- organizations;
- trust domains;
- nodes;
- workloads;
- registration rules;
- access policies;
- policy versions;
- SPIRE reconciliation;
- audit;
- security events.

### PostgreSQL

Source of truth for product configuration and audit metadata. It is not a storage location for workload private keys.

### SPIRE

Identity runtime responsible for node/workload attestation and SVID issuance. V1 integrates rather than forks SPIRE.

### Workload API

Workloads obtain identity material from the local SPIFFE Workload Endpoint. V1 prefers Unix domain socket transport in the reference environment.

### Enforcement layer

V1 will provide a reference enforcement path that:

1. establishes authenticated workload identity using X.509-SVIDs;
2. extracts/verifies SPIFFE identities;
3. evaluates deterministic source/destination/action policy;
4. denies by default;
5. records decision evidence without exposing key material.

The exact implementation mechanism (purpose-built Go proxy versus Envoy-based integration) is an explicit Phase 1/2 decision and must be benchmarked before lock-in.

## Deployment topology — V1 lab

```text
Docker/Linux host

SPIRE Server
SPIRE Agent
PostgreSQL
Control Plane
Console
Reference Enforcement Component

Demo workloads:
- frontend
- orders-api
- payment-api
- admin-api
```

## Repository shape

```text
apps/
  console/
  control-plane/
  cli/
internal/
  identity/
  policy/
  audit/
  spire/
  authz/
deploy/
  docker/
  spire/
examples/
  zero-trust-lab/
docs/
  adr/
```

The exact package layout can evolve after the Go scaffold exists; domain boundaries should remain explicit.

## Trust boundaries

1. Operator/browser ↔ control plane
2. Control plane ↔ PostgreSQL
3. Control plane ↔ SPIRE management surface
4. SPIRE Server ↔ SPIRE Agent
5. Workload ↔ local Workload API
6. Workload ↔ enforcement layer ↔ destination workload
7. CI/release system ↔ repository/artifacts

Every boundary requires explicit authentication, authorization and transport assumptions before V1 release.

## Availability posture

V1 is not HA. Security behavior is more important than availability under ambiguous trust state. Policy or identity verification failures must not become implicit allow decisions.
