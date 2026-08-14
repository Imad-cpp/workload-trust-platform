# Workload Trust Platform

> Working engineering name. Final product/brand name is intentionally undecided.

Workload Trust Platform is a security infrastructure product for giving software workloads verifiable identities and enforcing least-privilege service-to-service access without long-lived workload credentials.

The V1 direction combines SPIFFE/SPIRE workload identity, a Go control plane, PostgreSQL product state, deterministic authorization, and an operator console. The implementation is staged so identity, control-plane trust and authorization claims are proven separately.

## Product mission

Give every software workload a verifiable identity and make service access explicit, short-lived, auditable and default-deny.

## Verified progress

### Phase 1 — workload identity ✅

Permanent CI proves real Docker workload attestation, expected SPIFFE IDs, negative selector cases, restart/re-attestation and automatic short-lived X.509-SVID rotation.

### Phase 2 — control plane 🚧

Implemented slices now include:

- Go control-plane and one-shot reconciler processes;
- PostgreSQL schema/migrations and real repository integration tests;
- authenticated, still-loopback-only `/v1/*` reads with an explicit local operator principal;
- read-only workload inventory;
- append-only audit primitives and access-policy versions;
- registration-rule desired state, SPIRE entry binding and convergence/error state;
- official SPIRE v1.15.2 Entry API integration over the local Unix management socket;
- ownership-safe create/update/delete reconciliation and drift repair;
- permanent real PostgreSQL-to-SPIRE-to-Workload-API lifecycle CI.

The HTTP surface still has **no mutation endpoint**, no remote listener and no multi-role operator authorization model. Service authorization/mTLS enforcement is Phase 3 work.

## Technology

- Go — control plane, reconciler, CLI/policy/infrastructure integrations
- Next.js + React + TypeScript — planned operator console
- PostgreSQL — product configuration and audit source of truth
- SPIFFE / SPIRE — workload identity standard and runtime
- Docker / Linux — V1 reference environment
- GitHub Actions — quality, security and release evidence

## Documentation

Start with:

- [`docs/00_PRODUCT.md`](docs/00_PRODUCT.md)
- [`docs/01_ARCHITECTURE.md`](docs/01_ARCHITECTURE.md)
- [`docs/02_THREAT_MODEL.md`](docs/02_THREAT_MODEL.md)
- [`docs/03_SECURITY_INVARIANTS.md`](docs/03_SECURITY_INVARIANTS.md)
- [`docs/06_API_BOUNDARIES.md`](docs/06_API_BOUNDARIES.md)
- [`docs/08_DEFINITION_OF_DONE.md`](docs/08_DEFINITION_OF_DONE.md)
- [`docs/09_DECISIONS.md`](docs/09_DECISIONS.md)
- [`docs/10_ROADMAP.md`](docs/10_ROADMAP.md)
- [`docs/13_IDENTITY_LAB.md`](docs/13_IDENTITY_LAB.md)
- [`docs/14_PHASE1_EVIDENCE.md`](docs/14_PHASE1_EVIDENCE.md)
- [`docs/15_CONTROL_PLANE_FOUNDATION.md`](docs/15_CONTROL_PLANE_FOUNDATION.md)
- [`docs/16_OPERATOR_AUTH_AND_RECONCILIATION.md`](docs/16_OPERATOR_AUTH_AND_RECONCILIATION.md)

## Local control-plane development

```bash
docker compose -f deploy/dev/compose.yaml up -d postgres
export DATABASE_URL='postgres://workload_trust:local-development-only@127.0.0.1:5432/workload_trust?sslmode=disable'
./scripts/db-migrate.sh up
export WTP_OPERATOR_ID='local-admin'
export WTP_OPERATOR_TOKEN="$(openssl rand -hex 32)"
go run ./apps/control-plane
```

The example database credentials are local-development-only values. No production deployment or production-readiness claim is made.

## Current security boundary

- no custom cryptographic primitive;
- no workload private-key storage in the product database;
- operator HTTP remains loopback-only;
- `/v1/*` requires a local high-entropy bearer credential;
- no HTTP mutation API yet;
- SPIRE reconciliation refuses foreign entries by explicit ownership marker/binding checks;
- ownership hints are not cryptographic proof against privileged administrators;
- no service authorization claim until Phase 3.
