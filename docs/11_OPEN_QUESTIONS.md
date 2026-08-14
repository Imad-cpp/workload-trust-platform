# 11 — Open Questions

Status: Active  
Date: 2026-08-14

These questions do not block creation of the repository. Some block later implementation milestones.

## Brand / licensing

1. Final product/company name. `ZeroMesh` is rejected due to naming collision.
2. Public licensing model: permissive, copyleft/open-core, or other. Do not add a license until this is intentionally chosen.

## Identity lab

3. Exact SPIRE Docker/Linux workload-attestor selectors used in the V1 reference lab.
4. Final V1 trust-domain naming convention.
5. Node attestation mechanism for the local/reference deployment.

## Enforcement

6. Purpose-built Go reference proxy versus Envoy-based enforcement for V1. Benchmark both against implementation complexity, security semantics and educational/product value.
7. Whether V1 policy operates only at service connection level or includes HTTP method/path in a later V1.x milestone.

## Product model

8. Single-organization implementation with future-ready schema versus full multi-tenancy in V1. Current recommendation: one organization operationally, explicit organization IDs in core data model where cheap.
9. Audit retention/tamper-evidence mechanism appropriate for public V1.

## Operations

10. Minimum supported Docker Engine/Linux versions.
11. Supported deployment target for public demo beyond local Compose, if any.
