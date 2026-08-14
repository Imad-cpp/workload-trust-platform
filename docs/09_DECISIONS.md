# 09 — Decisions

Status: Active  
Date: 2026-08-14

| ID | Decision | Status |
|---|---|---|
| ADR-0001 | Go for security/infrastructure core | Accepted |
| ADR-0002 | SPIFFE/SPIRE for workload identity | Accepted |
| ADR-0003 | Modular control plane for V1, not microservices | Accepted |
| ADR-0004 | PostgreSQL as product-state source of truth | Accepted |

## Product decisions

- `ZeroMesh` is not used as the final brand because the name already collides with existing projects/companies.
- `Workload Trust Platform` is a neutral engineering working name only.
- V1 targets Docker/Linux before Kubernetes/cloud integrations.
- X.509-SVID is the primary V1 workload authentication mechanism.
- Authorization is deterministic and default-deny.
- Custom cryptography is prohibited.

See `docs/adr/` for rationale.
