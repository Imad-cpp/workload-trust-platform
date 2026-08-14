# 08 — Definition of Done

Status: Release gate  
Date: 2026-08-14

V1 is complete only when every required item below is evidenced.

## Product

- [ ] V1 scope implemented without silently expanding non-goals.
- [ ] Four-workload reference lab is reproducible from a clean checkout.
- [ ] Operator can see identities, workloads, policies and audit evidence.

## Identity

- [ ] Workloads obtain expected SPIFFE identities via attestation.
- [ ] Workload cannot simply request/select another workload identity.
- [ ] X.509-SVID lifecycle demonstrated and tested.
- [ ] No long-lived service-identity secret is required by demo workloads.

## Authorization

- [ ] Default deny enforced.
- [ ] Allowed path succeeds.
- [ ] Forbidden path fails.
- [ ] Policy absence/corruption fails closed.
- [ ] Decision contains safe, explainable reason metadata.

## Security

- [ ] Threat model reviewed against implementation.
- [ ] Security invariants have automated evidence where testable.
- [ ] No private key or credential leakage in logs/API/UI test corpus.
- [ ] Operator mutation authorization is server-side.
- [ ] Security-critical mutations generate audit records.
- [ ] Dependency/container/secret/static-analysis checks pass.

## Data

- [ ] PostgreSQL migrations apply from empty database.
- [ ] rollback strategy is tested.
- [ ] policy version history is preserved.
- [ ] no workload private key is stored in product tables.

## Reliability

- [ ] identity runtime unavailable behavior is documented/tested.
- [ ] policy component unavailable behavior is documented/tested.
- [ ] workload restart behavior is tested.
- [ ] no ambiguous trust state silently becomes allow.

## UX/API

- [ ] critical console flows have browser tests.
- [ ] API validation and stable error behavior tested.
- [ ] accessibility baseline for console: keyboard navigation, focus visibility, semantic labels and sufficient target sizes.

## Documentation

- [ ] architecture matches implementation.
- [ ] threat model matches implementation.
- [ ] V1 security limitations are explicit.
- [ ] operator setup and demo runbook exist.
- [ ] API/CLI usage documented.
- [ ] changelog and release notes prepared.

## Release

- [ ] protected main or equivalent repository rules are verified.
- [ ] PR required for release candidate changes.
- [ ] all permanent required workflows pass on exact release commit.
- [ ] release tag points to exact verified commit.
- [ ] GitHub Release notes match committed release notes.
