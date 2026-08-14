# Changelog

All notable changes to this project will be documented here.

## Unreleased

### Added

- Phase 0 product, architecture and security foundation.
- Initial ADRs for Go, SPIFFE/SPIRE, modular control plane and PostgreSQL.
- Initial V1 release gate and roadmap.
- Phase 1 SPIFFE/SPIRE Linux identity lab with verified release download.
- Real Docker workload attestation using conjunctive Docker-label selectors.
- Positive and negative X.509-SVID identity probes in GitHub Actions.
- Deterministic authorized-entry synchronization handling.
- Restart/re-attestation evidence across distinct Docker containers.
- Automatic short-lived X.509-SVID rotation evidence from the streaming Workload API.
- Redacted SPIRE failure diagnostics for CI troubleshooting.
- Phase 2 Go control-plane process with loopback-only HTTP API.
- PostgreSQL control-plane schema and apply/rollback/apply migration tests.
- Real PostgreSQL workload, registration and audit repository integration tests.
- Append-only audit events and access-policy versions enforced by PostgreSQL triggers.
- Go module-integrity, formatting, vet and race-detector CI.
- Local operator bearer authentication for `/v1/*` with explicit actor attribution and constant-time credential-digest comparison.
- Registration reconciliation state, SPIRE entry binding and stable reconciliation error codes.
- SPIRE v1.15.2 Entry API client over the local Unix management socket.
- Ownership-safe SPIRE create/update/delete reconciliation using `wtp-rule:<rule-id>` hints.
- Real desired-state lifecycle CI covering workload identity issuance, drift update, foreign-entry refusal and deletion.
- ADR-0007 for the local authenticated operator boundary and ADR-0008 for SPIRE reconciliation ownership/failure semantics.

### Changed

- ADR-0006 is superseded for authentication by ADR-0007 while its loopback/read-only restrictions remain active.
- The migration runner now upgrades an existing Phase 2 foundation schema before applying reconciliation fields.
