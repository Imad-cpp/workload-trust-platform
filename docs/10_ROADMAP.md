# 10 — Roadmap

Status: Directional  
Date: 2026-08-16

## Phase 0 — Foundation ✅

Completed on main: product/source-of-truth, V1 scope, architecture/threat/security/identity/policy/data/API docs, ADRs, Definition of Done and foundation CI.

## Phase 1 — SPIFFE/SPIRE identity lab ✅

Completed/evidenced: reproducible SPIRE v1.15.2 Server/Agent lab, verified release download, Docker attestation, four demo identities, positive/negative identity tests, restart/re-attestation, automatic short-lived X.509-SVID rotation and permanent exact-main CI evidence.

Phase 1 establishes identity only. It does not claim service authorization or production readiness.

## Phase 2 — Go control plane 🚧

Completed in implemented Phase 2 slices:

- Go control-plane + one-shot reconciliation processes;
- loopback-only management HTTP surface with generic health/readiness;
- high-entropy bearer authentication and attributable local operator identity;
- fail-closed `viewer`/`operator` permission mapping, defaulting to read-only viewer;
- PostgreSQL 18 schema/migrations through `000003_policy_revision` and append-only audit/policy history;
- authenticated workload inventory read path;
- authenticated + server-authorized registration desired-state POST/PATCH;
- strict/bounded mutation input and safe output/error envelopes;
- optimistic registration revision protection and transactional registration audit;
- official SPIRE v1.15.2 Entry API reconciliation, drift repair and foreign-entry refusal;
- authorized policy create/version/activation management routes;
- immutable policy history with `connect` + `allow|deny` V1 content;
- optimistic policy revisions and distinct `policies:activate` permission;
- stored-version revalidation before activation;
- transactionally coupled policy state + audit with forced-audit-failure rollback evidence;
- live policy tests for 401/403/201/409/200, stale writes and foreign-version activation refusal;
- aggregate `diagnostics:read` organization status endpoint with no sensitive rule/audit payloads;
- local read-only `wtpctl` commands for health, readiness, workload inventory and aggregate status;
- CLI loopback-only URL validation, redirect refusal, bearer validation, bounded responses and no direct PostgreSQL dependency;
- permanent real PostgreSQL/SPIRE/operator/policy/CLI integration CI;
- module graph/gofmt/shell-syntax/vet/race/real PostgreSQL CI.

Remaining before Phase 2 exit:

- completed control-plane security/failure review;
- Phase 2 consolidated evidence and exact-main exit verification;
- decision on whether V1 local operator remains single-principal or requires a multi-principal/session model before the console phase.

Policy activation in Phase 2 means selected desired state only. It does not enforce workload access. CLI diagnostics remain local read-only management inspection only.

## Phase 3 — Authorization enforcement

- mTLS reference path;
- verified SPIFFE identity extraction;
- deterministic default-deny policy engine consuming selected active policy versions;
- allowed/forbidden service-call integration suite;
- policy-absence/corruption failure semantics;
- safe decision reason metadata.

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

Kubernetes/cloud attestation, federation, Envoy/service-mesh integrations, GitOps/policy-as-code, enterprise identity/RBAC/SSO, HA/managed control plane, and workload access/security intelligence are post-V1 candidates unless explicitly promoted by a later decision.
