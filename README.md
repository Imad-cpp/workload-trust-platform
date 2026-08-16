# Workload Trust Platform

> Working engineering name. Final product/brand name is intentionally undecided.

Workload Trust Platform is a security infrastructure product for giving software workloads verifiable identities and enforcing least-privilege service-to-service access without long-lived workload credentials.

The V1 direction combines SPIFFE/SPIRE workload identity, a Go control plane, PostgreSQL product state, deterministic authorization and an operator console. Identity, management control and workload authorization claims are proved in separate stages.

## Product mission

Give every software workload a verifiable identity and make service access explicit, short-lived, auditable and default-deny.

## Verified progress

### Phase 1 — workload identity ✅

Permanent CI proves real Docker workload attestation, expected SPIFFE IDs, negative selector cases, restart/re-attestation and automatic short-lived X.509-SVID rotation.

### Phase 2 — control plane 🚧

Implemented/evidenced slices include:

- Go control-plane and one-shot SPIRE reconciler;
- PostgreSQL schema/migrations through policy revision migration `000003`, real integration tests and append-only security history;
- loopback-only high-entropy bearer authentication;
- fail-closed local `viewer` / `operator` management permissions;
- authenticated workload inventory and aggregate `diagnostics:read` status;
- server-authorized registration desired-state `POST`/`PATCH` with optimistic revisions and atomic audit;
- official SPIRE v1.15.2 Entry API reconciliation over the local Unix socket;
- ownership-safe SPIRE create/update/delete, drift repair and foreign-entry refusal;
- server-authorized policy creation, immutable version append and explicit activation;
- distinct `policies:activate` permission and optimistic policy revisions;
- canonical SPIFFE policy validation and stored-version revalidation before activation;
- policy state + operator audit in one PostgreSQL transaction, including forced-audit-failure rollback tests;
- read-only `wtpctl` for health, readiness, workload inventory and aggregate organization status;
- CLI through the existing loopback management API with no direct PostgreSQL credential/path, redirect refusal and bounded responses;
- permanent real SPIRE, operator-mutation, policy-mutation and CLI-diagnostics CI.

The current role model is still one configured local principal, not remote/multi-user enterprise RBAC. An `active_version_id` is selected **management desired state only**; service-to-service authorization, default-deny evaluation and mTLS enforcement are not implemented yet.

## Technology

- Go — control plane, reconciler, local read-only CLI, future policy enforcement integrations
- Next.js + React + TypeScript — planned operator console
- PostgreSQL — product configuration and audit source of truth
- SPIFFE / SPIRE — workload identity standard/runtime
- Docker / Linux — V1 reference environment
- GitHub Actions — quality, security and release evidence

## Documentation

Start with:

- [`docs/00_PRODUCT.md`](docs/00_PRODUCT.md)
- [`docs/01_ARCHITECTURE.md`](docs/01_ARCHITECTURE.md)
- [`docs/02_THREAT_MODEL.md`](docs/02_THREAT_MODEL.md)
- [`docs/03_SECURITY_INVARIANTS.md`](docs/03_SECURITY_INVARIANTS.md)
- [`docs/05_POLICY_MODEL.md`](docs/05_POLICY_MODEL.md)
- [`docs/06_API_BOUNDARIES.md`](docs/06_API_BOUNDARIES.md)
- [`docs/08_DEFINITION_OF_DONE.md`](docs/08_DEFINITION_OF_DONE.md)
- [`docs/09_DECISIONS.md`](docs/09_DECISIONS.md)
- [`docs/10_ROADMAP.md`](docs/10_ROADMAP.md)
- [`docs/13_IDENTITY_LAB.md`](docs/13_IDENTITY_LAB.md)
- [`docs/14_PHASE1_EVIDENCE.md`](docs/14_PHASE1_EVIDENCE.md)
- [`docs/15_CONTROL_PLANE_FOUNDATION.md`](docs/15_CONTROL_PLANE_FOUNDATION.md)
- [`docs/16_OPERATOR_AUTH_AND_RECONCILIATION.md`](docs/16_OPERATOR_AUTH_AND_RECONCILIATION.md)
- [`docs/17_OPERATOR_MUTATION_EVIDENCE.md`](docs/17_OPERATOR_MUTATION_EVIDENCE.md)
- [`docs/18_POLICY_MUTATION_EVIDENCE.md`](docs/18_POLICY_MUTATION_EVIDENCE.md)
- [`docs/19_CLI_DIAGNOSTICS_EVIDENCE.md`](docs/19_CLI_DIAGNOSTICS_EVIDENCE.md)

## Local control-plane development

```bash
docker compose -f deploy/dev/compose.yaml up -d postgres
export DATABASE_URL='postgres://workload_trust:local-development-only@127.0.0.1:5432/workload_trust?sslmode=disable'
./scripts/db-migrate.sh up
export WTP_OPERATOR_ID='local-admin'
export WTP_OPERATOR_ROLE='viewer'
export WTP_OPERATOR_TOKEN="$(openssl rand -hex 32)"
go run ./apps/control-plane
```

Use the CLI from another local shell without exposing the database URL to the CLI process:

```bash
export WTP_API_URL='http://127.0.0.1:8080'
export WTP_OPERATOR_TOKEN='<same-local-bearer-token>'
go run ./apps/wtpctl health
go run ./apps/wtpctl ready
go run ./apps/wtpctl status --organization-id '<organization-uuid>'
go run ./apps/wtpctl workloads --organization-id '<organization-uuid>'
```

`status` is aggregate diagnostics. `workloads` intentionally returns the existing authorized workload inventory and may include workload SPIFFE IDs.

Set `WTP_OPERATOR_ROLE=operator` only for a local principal that should be allowed to change registration/policy management state.

No production deployment or production-readiness claim is made.

## Current security boundary

- no custom cryptographic primitive;
- no workload private-key storage in the product database;
- management HTTP remains loopback-only;
- `/v1/*` requires a local high-entropy bearer credential;
- `diagnostics:read` protects aggregate diagnostics for viewer/operator;
- registration writes require `registrations:write`;
- policy writes require `policies:write`; activation requires distinct `policies:activate`;
- missing role defaults to `viewer`; unknown roles fail closed;
- registration/policy state + audit are transactionally coupled;
- policy versions are immutable and malformed/foreign versions cannot be activated through the management workflow;
- `wtpctl` stays local/read-only, does not connect to PostgreSQL, rejects redirects/non-loopback endpoints and bounds management responses;
- management responses/audits avoid echoing sensitive desired-state detail;
- SPIRE reconciliation refuses foreign entries by explicit ownership marker/binding checks;
- active policy state is not a service-authorization result;
- no workload service-authorization claim until Phase 3.
