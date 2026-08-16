# 09 — Decisions

Status: Active  
Date: 2026-08-16

| ID | Decision | Status |
|---|---|---|
| ADR-0001 | Go for security/infrastructure core | Accepted |
| ADR-0002 | SPIFFE/SPIRE for workload identity | Accepted |
| ADR-0003 | Modular control plane for V1, not microservices | Accepted |
| ADR-0004 | PostgreSQL as product-state source of truth | Accepted |
| ADR-0005 | Host-native SPIRE + Docker-label attestation for Phase 1 lab | Accepted |
| ADR-0006 | Read-only loopback operator API before authentication | Superseded by ADR-0007 for auth |
| ADR-0007 | High-entropy bearer auth for the local operator API | Accepted |
| ADR-0008 | Ownership-safe SPIRE registration reconciliation | Accepted |
| ADR-0009 | Fail-closed roles + transactionally audited registration mutations | Accepted |
| ADR-0010 | Immutable policy versions + transactionally audited activation | Accepted |
| ADR-0011 | Local read-only CLI through the management API | Accepted |

## Product decisions

- `ZeroMesh` is not used as the final brand because the name already collides with existing projects/companies.
- `Workload Trust Platform` is a neutral engineering working name only.
- V1 targets Docker/Linux before Kubernetes/cloud integrations.
- X.509-SVID is the primary V1 workload authentication mechanism.
- Workload authorization is deterministic and intended to be default-deny.
- Custom cryptography is prohibited.

## Phase 1 lab decisions

- SPIRE lab runtime is pinned to v1.15.2 and release archives are SHA-256 verified before use.
- Lab trust domain is `workload-trust.test`.
- The local lab agent uses a one-time join token and `insecure_bootstrap`; neither is a production bootstrap claim.
- Docker workload identity in the lab requires both workload-name and environment labels.
- Phase 1 proves identity issuance and negative attestation cases only; workload authorization remains later work.

## Phase 2 decisions

- Go toolchain/module metadata is pinned and CI rejects a non-tidy module graph before compilation.
- PostgreSQL migrations are tested apply/rollback/re-apply against real PostgreSQL through migration `000003_policy_revision`.
- `audit_events` and `access_policy_versions` are append-only at the PostgreSQL layer.
- `/v1/*` requires the ADR-0007 bearer credential; health/readiness remain generic and unauthenticated.
- management HTTP remains loopback-only.
- local operator roles are `viewer` and `operator`; missing role defaults to `viewer`, unsupported roles fail closed.
- registration POST/PATCH requires `registrations:write`; mutation + audit are atomic and stale revisions fail closed.
- policy create/version append requires `policies:write`; activation requires distinct `policies:activate`.
- V1 policy content is limited to canonical SPIFFE source/destination IDs, `connect`, and `allow|deny`.
- policy history is append-only; edits create new immutable versions.
- policy append/activation requires optimistic `expected_revision`.
- activation is allowed only for a version belonging to the target policy and stored content is revalidated before activation.
- policy create/version/activation + operator audit commit in the same PostgreSQL transaction; audit failure rolls state back.
- `active_version_id` means selected desired policy state only; it is **not** a claim that service traffic is authorized/enforced.
- `diagnostics:read` is available to viewer/operator for aggregate organization diagnostics.
- `wtpctl` is read-only and consumes the existing loopback management API rather than opening PostgreSQL directly.
- the Phase 2 CLI refuses non-loopback management URLs, follows no redirects and does not require `DATABASE_URL`.
- registration desired state is reconciled to SPIRE through the official Entry API over the local Unix management socket.
- managed SPIRE entries use `wtp-rule:<rule-id>` as a non-cryptographic ownership convention; foreign entries fail closed rather than being adopted/mutated.

Phase 3 remains responsible for verified source identity extraction, deterministic default-deny policy evaluation and service-call enforcement.

See `docs/adr/` for rationale.
