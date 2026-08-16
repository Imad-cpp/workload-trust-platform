# Changelog

All notable changes to this project will be documented here.

## Unreleased

### Added

- Phase 0 product, architecture and security foundation.
- Initial ADRs for Go, SPIFFE/SPIRE, modular control plane and PostgreSQL.
- Initial V1 release gate and roadmap.
- Phase 1 SPIFFE/SPIRE Linux identity lab with verified release download.
- Real Docker workload attestation using conjunctive Docker-label selectors.
- Positive/negative X.509-SVID probes, restart/re-attestation and automatic rotation CI.
- Phase 2 Go control-plane process and PostgreSQL schema/repository foundation.
- Append-only audit events and access-policy versions enforced by PostgreSQL.
- Go module-integrity, formatting, vet and race-detector CI.
- Local high-entropy bearer authentication and explicit operator actor attribution.
- Registration reconciliation state and official SPIRE v1.15.2 Entry API integration.
- Ownership-safe SPIRE create/update/delete reconciliation with drift and foreign-entry tests.
- Real PostgreSQL → SPIRE → Workload API reconciliation lifecycle CI.
- Fail-closed local `viewer`/`operator` permission mapping with read-only default.
- Authenticated + server-authorized registration desired-state POST/PATCH endpoints.
- Strict bounded JSON mutation validation and safe mutation response/error envelopes.
- Optimistic registration revision checks returning stable conflict behavior.
- Transactionally coupled registration mutation + operator audit, including forced-audit-failure rollback test.
- Live Operator Mutation Integration CI proving 401/403/201/200/409 behavior and safe audit metadata.
- ADR-0007/0008/0009 and Phase 2 evidence documentation.

### Changed

- ADR-0006 is superseded for authentication by ADR-0007 while loopback containment remains active.
- The operator HTTP surface now permits registration desired-state mutations only for principals with `registrations:write`.
- Registration PATCH is defined as a full desired-state replacement guarded by `expected_revision`.
