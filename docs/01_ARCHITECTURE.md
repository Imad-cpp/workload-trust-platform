# 01 — Architecture

Status: Evolving implementation  
Date: 2026-08-14

## Architectural goal

Keep V1 operationally simple while preserving clear domain boundaries. The implementation is a modular Go control plane, not a fleet of independent microservices.

## System context

```text
Operator
   |
   |  Phase 2: loopback-only, read-only HTTP
   v
Control Plane API -------------------- PostgreSQL
   |                                      |
   |                                      +-- product state
   |                                      +-- immutable policy versions
   |                                      +-- append-only audit records
   |
   +--> future SPIRE reconciliation --> SPIRE Server
   |                                         |
   |                                     SPIRE Agents
   |                                         |
   |                                     Workloads
   |
   +--> future authorization enforcement
```

Phase 1 already proves the SPIRE workload-identity path independently. Phase 2 begins the product control plane. The two are intentionally not coupled through an untested reconciliation layer yet.

## Implemented components

### Go control plane

Entry point: `apps/control-plane/main.go`.

Current responsibilities:

- validated configuration;
- loopback-only HTTP listener while unauthenticated;
- structured JSON logging;
- graceful shutdown;
- PostgreSQL pool lifecycle;
- health/readiness;
- read-only workload inventory.

Logical internal modules currently include:

- configuration;
- database;
- HTTP API;
- workloads;
- audit.

Future modules remain planned for registration reconciliation, policy/authz, security events and SPIRE integration.

### PostgreSQL

Current source of truth for product configuration/history introduced by `000001_control_plane`:

- organizations;
- trust domains;
- workloads;
- registration rules;
- access policies and append-only versions;
- append-only audit events;
- security events.

It is not a workload private-key store.

### SPIRE identity lab

Phase 1 provides the real identity runtime evidence. SPIRE is still not mutated by the Go control plane in this Phase 2 slice.

## Planned components

### Operator console

Next.js + React + TypeScript application. Not implemented yet.

### SPIRE reconciliation

The control plane will eventually reconcile desired workload registration state with supported SPIRE management surfaces. This is not implemented in the current slice.

### Authorization enforcement

Phase 3 will provide authenticated service identity verification plus deterministic default-deny policy enforcement. A valid SVID is not treated as authorization today.

## Repository shape

```text
apps/
  control-plane/
internal/
  audit/
  config/
  database/
  httpapi/
  workload/
db/
  migrations/
deploy/
  dev/
  spire/
examples/
  identity-lab/
docs/
  adr/
```

The future console/CLI and policy/SPIRE integration packages will be added only when their slices begin.

## Trust boundaries

1. Local operator/process ↔ control-plane HTTP API (temporary loopback containment; no auth yet)
2. Control plane ↔ PostgreSQL
3. Future control plane ↔ SPIRE management surface
4. SPIRE Server ↔ SPIRE Agent
5. Workload ↔ local Workload API
6. Future workload ↔ enforcement layer ↔ destination workload
7. CI/release system ↔ repository/artifacts

Every remotely reachable management boundary requires explicit authentication and authorization before it can be opened.

## Availability posture

V1 is not HA. Security behavior is more important than availability under ambiguous trust state. Policy or identity verification failures must not become implicit allow decisions.
