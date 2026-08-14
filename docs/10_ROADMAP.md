# 10 — Roadmap

Status: Directional  
Date: 2026-08-14

## Phase 0 — Foundation ✅

Completed on main:

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
- Definition of Done;
- foundation CI validation.

## Phase 1 — SPIFFE/SPIRE identity lab ✅

Completed/evidenced:

- reproducible SPIRE Server/Agent environment;
- verified upstream SPIRE v1.15.2 release download;
- Docker/Linux attestation using runtime Docker selectors;
- four registered demo workload identities;
- positive and negative identity tests;
- workload recreation/re-attestation;
- automatic short-lived X.509-SVID rotation;
- CI diagnostics/redaction;
- permanent CI evidence on exact main commit.

Phase 1 establishes identity only. It does not claim service authorization or production readiness.

## Phase 2 — Go control plane 🚧

Completed in the current foundation slice:

- Go control-plane process with graceful shutdown and structured logging;
- loopback-only read-only HTTP surface;
- health/readiness endpoints;
- PostgreSQL 18 schema/migrations;
- workload inventory read repository and endpoint;
- append-only audit repository primitive;
- append-only access-policy version history at the database layer;
- real PostgreSQL migration/repository integration tests;
- Go formatting, module-graph, vet and race-detector CI.

Remaining before Phase 2 exit:

- explicit operator authentication/authorization model;
- authenticated mutation/service workflows;
- SPIRE desired-state reconciliation;
- registration-rule lifecycle and drift handling;
- policy activation workflow and audit linkage;
- CLI diagnostics/inspection surface;
- security review of the completed control-plane boundary;
- Phase 2 evidence document and exact-main verification.

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
