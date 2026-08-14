# Workload Trust Platform

> Working engineering name. Final product/brand name is intentionally undecided.

Workload Trust Platform is a security infrastructure product for giving software workloads verifiable identities and enforcing least-privilege service-to-service access without long-lived workload credentials.

The V1 direction combines SPIFFE/SPIRE workload identity, a Go control plane, PostgreSQL product state, deterministic authorization, and an operator console. The implementation is intentionally staged so identity, control-plane trust, and authorization claims are proven separately.

## Product mission

Give every software workload a verifiable identity and make service access explicit, short-lived, auditable, and default-deny.

## Verified progress

### Phase 1 — workload identity ✅

Permanent CI proves:

- real Docker workload attestation through SPIRE;
- expected SPIFFE IDs for four workloads;
- negative selector cases;
- restart/re-attestation across distinct containers;
- automatic short-lived X.509-SVID rotation.

### Phase 2 — control-plane foundation 🚧

The current branch introduces:

- Go control-plane process;
- PostgreSQL schema/migrations;
- read-only workload inventory;
- append-only audit primitives;
- append-only access-policy versions;
- health/readiness and stable JSON errors;
- real PostgreSQL integration tests.

The operator HTTP API is deliberately **loopback-only and read-only** until authentication/authorization is designed. SPIRE reconciliation and service authorization are not implemented yet.

## Technology

- Go — control plane, CLI, policy and infrastructure integrations
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
- [`docs/08_DEFINITION_OF_DONE.md`](docs/08_DEFINITION_OF_DONE.md)
- [`docs/09_DECISIONS.md`](docs/09_DECISIONS.md)
- [`docs/10_ROADMAP.md`](docs/10_ROADMAP.md)
- [`docs/13_IDENTITY_LAB.md`](docs/13_IDENTITY_LAB.md)
- [`docs/14_PHASE1_EVIDENCE.md`](docs/14_PHASE1_EVIDENCE.md)
- [`docs/15_CONTROL_PLANE_FOUNDATION.md`](docs/15_CONTROL_PLANE_FOUNDATION.md)

## Local control-plane development

```bash
docker compose -f deploy/dev/compose.yaml up -d postgres
export DATABASE_URL='postgres://workload_trust:local-development-only@127.0.0.1:5432/workload_trust?sslmode=disable'
./scripts/db-migrate.sh up
go run ./apps/control-plane
```

The example credentials are local-development-only values. No production deployment or production-readiness claim is made.

## Current security boundary

- no custom cryptographic primitive;
- no workload private-key storage in the product database;
- no remote operator API yet;
- no HTTP mutation API before operator authentication/authorization;
- no service authorization claim until Phase 3.
