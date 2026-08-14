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
- Phase 2 Go control-plane process with loopback-only read-only HTTP API.
- PostgreSQL control-plane schema and apply/rollback/apply migration tests.
- Real PostgreSQL workload and audit repository integration tests.
- Append-only audit events and access-policy versions enforced by PostgreSQL triggers.
- Go module-integrity, formatting, vet and race-detector CI.
- ADR-0006 protecting the unauthenticated operator boundary from non-loopback/mutation exposure.
