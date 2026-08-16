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
- Go module-integrity, formatting, shell-syntax, vet and race-detector CI.
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
- Policy revision migration `000003_policy_revision` integrated into apply/rollback/re-apply flow.
- `policies:read`, `policies:write` and distinct `policies:activate` management permissions.
- Authenticated + server-authorized policy create, immutable version append and activation routes.
- Canonical SPIFFE policy validation with V1 `connect` + `allow|deny` semantics.
- Optimistic policy revision protection and stored-version revalidation before activation.
- Transactionally coupled policy state + operator audit, including forced-audit-failure rollback evidence.
- Live Policy Mutation Integration CI proving 401/403/201/201/409/200, foreign-version refusal and safe audit/response behavior.
- ADR-0010 and Phase 2 policy-management evidence documentation.

### Changed

- ADR-0006 is superseded for authentication by ADR-0007 while loopback containment remains active.
- The operator HTTP surface permits registration desired-state mutations only for principals with `registrations:write`.
- Registration PATCH is defined as a full desired-state replacement guarded by `expected_revision`.
- The migration runner now applies `000001` → `000002` → `000003` and rolls back in reverse order.
- Active policy state now means a selected immutable desired policy version; service authorization/enforcement remains Phase 3.
