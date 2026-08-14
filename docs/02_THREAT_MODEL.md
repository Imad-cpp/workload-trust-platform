# 02 — Threat Model

Status: Initial  
Date: 2026-08-14

## Security objective

Prevent unauthorized workloads from obtaining trusted identities or using a trusted identity to access services outside explicitly granted policy.

## Protected assets

- trust-domain signing authority and sensitive SPIRE state;
- workload private keys and X.509-SVIDs;
- registration/attestation policy;
- service access policy;
- operator credentials and sessions;
- audit integrity;
- PostgreSQL configuration state;
- CI/release credentials and artifacts.

## Actors

- legitimate platform operator;
- legitimate workload;
- compromised workload process/container;
- malicious local workload;
- attacker with network access;
- attacker with stolen operator session;
- attacker modifying repository/CI dependencies;
- attacker with partial host access.

## Initial abuse cases

### T1 — Workload self-assigns a privileged identity

Control: identity is derived from attestation/registration selectors; workload-supplied identity claims are not trusted.

### T2 — Stolen long-lived service credential enables impersonation

Control: workload identity uses short-lived SVIDs; V1 demo service authentication must not rely on embedded long-lived app credentials.

### T3 — Valid but low-privilege workload accesses admin service

Control: authentication and authorization remain separate; default-deny policy blocks ungranted paths.

### T4 — Expired or invalid SVID is accepted

Control: chain, trust domain, identity and validity are verified on every new authenticated session as required by the enforcement design.

### T5 — Policy service failure widens access

Control: fail closed. Cached policy, if introduced, must have explicit validity semantics and may never transform unknown into allow.

### T6 — Workload reads another workload's identity from the Workload API

Control: local workload attestation and endpoint isolation; V1 integration tests include unauthorized local caller cases.

### T7 — Sensitive key material appears in logs/UI/API

Control: key material is never serialized into product audit/log payloads; log redaction tests are required.

### T8 — Operator changes an allow policy without accountability

Control: append-oriented audit event with actor, timestamp, object, before/after metadata and request correlation; policy versions are immutable once superseded.

### T9 — Compromised application container attempts privilege escalation through metadata manipulation

Control: use runtime-derived attestation selectors and test forged labels/metadata where applicable; document what host-level compromise invalidates the model.

### T10 — Host compromise

A root-level host compromise may defeat local process/workload attestation assumptions. V1 must state this boundary clearly and must not claim protection against a fully compromised host.

### T11 — Compromised control plane changes SPIRE registration policy

Control: strict operator authorization, audit, narrow SPIRE integration privileges, and protected admin endpoints. Stronger separation is a post-V1 hardening path.

### T12 — Supply-chain compromise

Control: dependency pinning/lockfiles, secret scanning, static analysis, dependency review/audit, container scanning, reproducible evidence and exact-commit release gates.

## Required negative tests before V1

- unregistered container cannot obtain the expected identity;
- wrong attestation selectors cannot obtain another workload identity;
- expired/invalid SVID cannot authenticate;
- allowed source can call allowed destination;
- same source cannot call an ungranted destination;
- absence/corruption of policy fails closed;
- restart yields fresh identity lifecycle behavior;
- secrets/private keys do not appear in logs or API responses;
- unauthorized operator cannot mutate trust/policy state;
- audit event is produced for security-critical mutations.

## Explicit limitations

V1 does not claim resistance to a fully compromised root host, compromised SPIRE signing authority, or malicious modification of trusted runtime components. Those are separate threat boundaries requiring stronger hardware/root-of-trust and operational controls.
