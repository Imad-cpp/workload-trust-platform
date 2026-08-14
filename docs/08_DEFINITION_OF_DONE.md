# 08 — Definition of Done

Status: Release gate  
Date: 2026-08-14

V1 is complete only when every required item below is evidenced.

## Product

- [ ] V1 scope implemented without silently expanding non-goals.
- [ ] Four-workload reference lab is reproducible from a clean checkout.
- [ ] Operator can see identities, workloads, policies and audit evidence.

## Identity

- [x] Workloads obtain expected SPIFFE identities via attestation. (Phase 1 CI)
- [x] Workload cannot simply request/select another workload identity in the controlled Docker lab. (Phase 1 negative-selector CI; production selector strength remains a documented limitation.)
- [x] X.509-SVID lifecycle demonstrated and tested, including automatic rotation. (Phase 1 CI)
- [x] No long-lived service-identity secret is required by demo workloads. (Phase 1 lab)

## Authorization

- [ ] Default deny service authorization enforced.
- [ ] Allowed service path succeeds.
- [ ] Forbidden service path fails.
- [ ] Policy absence/corruption fails closed.
- [ ] Decision contains safe, explainable reason metadata.

## Security

- [ ] Threat model reviewed against completed V1 implementation.
- [ ] Security invariants have automated evidence where testable.
- [ ] No private key or credential leakage in logs/API/UI test corpus.
- [x] Server-side authorization protects every currently implemented operator mutation. (ADR-0009 + Operator Mutation Integration)
- [x] Every currently implemented security-critical mutation generates attributable append-only audit evidence. (registration desired-state + SPIRE reconciliation CI)
- [ ] Dependency/container/secret/static-analysis checks pass for release candidate.

## Data

- [x] PostgreSQL migrations apply from empty database. (Phase 2 CI)
- [x] rollback strategy is tested. (Phase 2 CI apply/rollback/apply)
- [x] policy version history is preserved against UPDATE/DELETE at the database layer. (Phase 2 migration tests)
- [x] no workload private key is stored in product tables. (Phase 2 schema)
- [x] registration desired-state mutation and operator audit are atomic; audit failure rolls back mutation. (Phase 2 real PostgreSQL test)
- [x] stale registration revisions fail without lost-update mutation. (Phase 2 real PostgreSQL/HTTP tests)

## Reliability

- [ ] identity runtime unavailable behavior is documented/tested.
- [ ] policy component unavailable behavior is documented/tested.
- [x] workload restart/re-attestation behavior is tested. (Phase 1 CI)
- [ ] no ambiguous trust state silently becomes allow.

## UX/API

- [ ] critical console flows have browser tests.
- [x] current HTTP API authentication, authorization, validation and stable error behavior are tested. (Go + Operator Mutation Integration)
- [ ] accessibility baseline for console: keyboard navigation, focus visibility, semantic labels and sufficient target sizes.

## Documentation

- [ ] architecture matches completed V1 implementation.
- [ ] threat model matches completed V1 implementation.
- [ ] V1 security limitations are explicit.
- [x] operator setup and identity-lab runbook exist. (Phase 1/2)
- [ ] API/CLI usage documented for completed V1.
- [ ] changelog and release notes prepared.

## Release

- [ ] protected main or equivalent repository rules are verified.
- [ ] PR required for release candidate changes.
- [ ] all permanent required workflows pass on exact release commit.
- [ ] release tag points to exact verified commit.
- [ ] GitHub Release notes match committed release notes.
