# 10 — Roadmap

Status: Directional  
Date: 2026-08-14

## Phase 0 — Foundation

- product source of truth;
- V1/non-V1;
- architecture;
- threat model;
- security invariants;
- identity model;
- policy model;
- data model;
- API boundaries;
- ADRs;
- Definition of Done.

Exit: architecture is internally consistent enough to scaffold without inventing trust semantics during coding.

## Phase 1 — SPIFFE/SPIRE identity lab

- reproducible SPIRE Server/Agent environment;
- Docker/Linux attestation experiment;
- four demo workloads;
- X.509-SVID retrieval and rotation;
- negative identity tests;
- selector/registration ADR.

## Phase 2 — Go control plane

- database/migrations;
- workload and registration model;
- SPIRE reconciliation;
- policy model and versioning;
- audit foundation;
- CLI diagnostics.

## Phase 3 — Authorization enforcement

- mTLS reference path;
- verified SPIFFE identity extraction;
- deterministic policy engine;
- allow/deny integration suite;
- failure semantics.

## Phase 4 — Operator console

- inventory;
- policy management;
- audit/security events;
- diagnostics;
- browser E2E.

## Phase 5 — Security/failure engineering

- malformed/expired identity cases;
- unavailable identity/policy components;
- unauthorized local workload cases;
- log/key leakage checks;
- adversarial policy tests;
- CI hardening.

## Phase 6 — V1.0.0 release

- clean-install runbook;
- security review;
- exact-commit permanent CI;
- release notes;
- tag/release integrity evidence;
- portfolio case study.

## Post-V1 candidates

Evaluate based on user need, not portfolio optics:

- Kubernetes workload identity;
- cloud node/workload attestation;
- federation;
- Envoy/service-mesh integrations;
- policy-as-code/GitOps;
- enterprise identity/RBAC/SSO;
- high availability;
- managed control plane;
- workload access graph and security intelligence.
