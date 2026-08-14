# Workload Trust Platform

> Working engineering name. Final product/brand name is intentionally undecided.

Workload Trust Platform is a security infrastructure product for giving software workloads verifiable identities and enforcing least-privilege service-to-service access without long-lived workload credentials.

The first public milestone focuses on Linux and Docker workloads. It uses SPIFFE as the workload identity standard, SPIRE as the identity runtime, X.509-SVIDs for workload authentication, and deterministic authorization policies for service-to-service access.

## Product mission

Give every software workload a verifiable identity and make service access explicit, short-lived, auditable, and default-deny.

## V1 outcome

A reproducible local environment demonstrates all of the following end to end:

1. Docker/Linux workloads are attested rather than self-identifying.
2. Workloads receive short-lived X.509-SVIDs through the SPIFFE Workload API.
3. Allowed service-to-service calls succeed over mutually authenticated TLS.
4. Disallowed calls fail closed.
5. Identity and policy lifecycle events are visible through an operator control plane and audit trail.
6. Restarted workloads obtain fresh identity material without reusing long-lived application secrets.
7. The complete security-critical path is covered by automated integration and failure tests.

## Planned technology

- Go — control plane, CLI, policy and infrastructure integrations
- Next.js + React + TypeScript — operator console
- PostgreSQL — product configuration and audit source of truth
- SPIFFE / SPIRE — workload identity standard and runtime
- gRPC — internal control and integration APIs where appropriate
- Docker Compose / Linux — V1 deployment and reference lab
- GitHub Actions — quality, security and release evidence

## Documentation

Start with:

- [`docs/00_PRODUCT.md`](docs/00_PRODUCT.md)
- [`docs/01_ARCHITECTURE.md`](docs/01_ARCHITECTURE.md)
- [`docs/02_THREAT_MODEL.md`](docs/02_THREAT_MODEL.md)
- [`docs/03_SECURITY_INVARIANTS.md`](docs/03_SECURITY_INVARIANTS.md)
- [`docs/07_V1_SCOPE.md`](docs/07_V1_SCOPE.md)
- [`docs/08_DEFINITION_OF_DONE.md`](docs/08_DEFINITION_OF_DONE.md)
- [`docs/09_DECISIONS.md`](docs/09_DECISIONS.md)
- [`docs/11_OPEN_QUESTIONS.md`](docs/11_OPEN_QUESTIONS.md)

## Current status

**Phase 0 — Product, architecture and security foundation.**

No production-readiness claim is made. No custom cryptographic primitive is planned.
