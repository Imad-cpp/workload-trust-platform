# 09 — Decisions

Status: Active  
Date: 2026-08-14

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

## Product decisions

- `ZeroMesh` is not used as the final brand because the name already collides with existing projects/companies.
- `Workload Trust Platform` is a neutral engineering working name only.
- V1 targets Docker/Linux before Kubernetes/cloud integrations.
- X.509-SVID is the primary V1 workload authentication mechanism.
- Workload authorization is deterministic and default-deny.
- Custom cryptography is prohibited.

## Phase 1 lab decisions

- SPIRE lab runtime is pinned to v1.15.2 and release archives are SHA-256 verified before use.
- Lab trust domain is `workload-trust.test`.
- The local lab agent uses a one-time join token and `insecure_bootstrap`; neither is a production bootstrap claim.
- Docker workload identity in the lab requires both workload-name and environment labels.
- Phase 1 proves identity issuance and negative attestation cases only; workload authorization remains later work.

## Phase 2 decisions

- Go toolchain/module metadata is pinned and CI rejects a non-tidy module graph before compilation.
- PostgreSQL migrations are tested apply/rollback/apply against real PostgreSQL.
- `audit_events` and `access_policy_versions` are append-only at the PostgreSQL layer.
- `/v1/*` requires the ADR-0007 bearer credential; health/readiness remain generic and unauthenticated.
- the HTTP listener remains loopback-only.
- local operator roles are `viewer` and `operator`; missing role defaults to `viewer`, unsupported roles fail closed.
- registration POST/PATCH requires server-side `registrations:write` authorization.
- registration PATCH is a full desired-state replacement with optimistic `expected_revision`, not JSON Merge Patch.
- registration mutation + operator audit commit atomically in one PostgreSQL transaction; audit failure rolls back state.
- mutation responses/audit metadata omit selector values and parent SPIFFE IDs.
- registration desired state is reconciled to SPIRE through the official Entry API over the local Unix management socket.
- managed SPIRE entries use `wtp-rule:<rule-id>` as a non-cryptographic ownership convention; foreign entries fail closed rather than being adopted/mutated.

See `docs/adr/` for rationale.
