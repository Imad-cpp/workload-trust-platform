# 03 — Security Invariants

Status: Required  
Date: 2026-08-14

These invariants are release-blocking. A change that violates one requires an explicit security decision and corresponding architecture review.

1. **A workload cannot choose its trusted identity.** Trusted identity is derived from configured attestation/registration rules.
2. **Authentication never implies authorization.** Possession of a valid SVID does not grant service access without a matching policy.
3. **Authorization is default-deny.** Missing, malformed, unavailable or ambiguous authorization state does not become allow.
4. **No custom cryptographic primitives.** Use established libraries and SPIFFE/SPIRE mechanisms.
5. **No long-lived service-identity credential in V1 demo workloads.** The core demo must prove service identity without embedded persistent API keys/passwords.
6. **Workload private keys are never persisted by the product database, returned by operator APIs, or displayed in the console.**
7. **Security-sensitive operator mutations are authenticated, authorized and audited.**
8. **Policy history is attributable.** Effective policy versions can be traced to who changed what and when.
9. **Trust-domain boundaries are explicit.** No implicit cross-domain trust.
10. **Identity and policy verification failures fail closed.**
11. **Sensitive values are redacted from logs.** Automated tests cover representative failure paths.
12. **Release claims require exact-commit evidence.** V1.0.0 is published only from a verified commit after required workflows succeed on that commit.
13. **Threat-model limitations are documented.** The project does not claim protection outside its verified trust assumptions.
