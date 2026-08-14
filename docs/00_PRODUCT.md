# 00 — Product

Status: Draft foundation  
Date: 2026-08-14

## Mission

Give every software workload a verifiable identity and make service access explicit, short-lived, auditable, and default-deny.

## Problem

Modern systems rely on APIs, workers, containers, virtual machines and CI jobs that need to authenticate to each other. A common implementation is long-lived credentials distributed through environment variables, secret stores or CI configuration. Those credentials are difficult to scope, rotate, attribute and revoke safely. Network position is also frequently treated as a proxy for trust.

The product replaces that model with workload identity:

- identify the running workload through attestation;
- issue short-lived cryptographic identity material;
- authenticate service-to-service communication with that identity;
- evaluate explicit authorization policy;
- record security-relevant lifecycle and access events.

## Target users

### Initial operator

Platform, security or backend engineers running a small Linux/Docker environment who need verifiable service identity without embedding long-lived service credentials.

### Long-term customer categories

- SaaS engineering teams;
- platform engineering teams;
- security engineering teams;
- regulated organizations;
- organizations moving from static service credentials toward zero-trust workload identity;
- hybrid and multi-environment infrastructure operators.

## Product wedge

V1 deliberately starts with one narrow but real outcome:

> Docker/Linux workload identity + automatic X.509-SVID lifecycle + mutual TLS + deterministic service-to-service authorization + operator visibility.

This wedge is valuable independently and creates a foundation for later Kubernetes, cloud attestation, federation and enterprise controls.

## Product principles

1. **Attestation before identity.** A workload never chooses its own identity.
2. **Authentication is not authorization.** A valid workload identity grants no implicit access.
3. **Default deny.** Missing or invalid policy fails closed.
4. **Short-lived identity material.** Workloads do not depend on long-lived application credentials for service identity.
5. **Standards before custom crypto.** Build on SPIFFE/SPIRE and existing cryptographic libraries.
6. **Operator-visible trust.** Identity issuance, policy changes and access denials must be explainable.
7. **Safe failure.** Control-plane degradation must not silently widen access.
8. **Evidence over claims.** Security guarantees are supported by automated tests and documented boundaries.

## Non-goals for V1

- replacing a general-purpose secrets manager;
- implementing a new PKI or cryptographic algorithm;
- building a full service mesh;
- Kubernetes production support;
- AWS/Azure/GCP attestors;
- multi-region HA;
- cross-organization federation;
- billing;
- SAML/enterprise SSO;
- AI-based authorization.

## Long-term product families

Potential future surfaces, subject to evidence and roadmap approval:

- Identity — workload inventory, attestation and lifecycle;
- Access — service-to-service authorization;
- Certificates — automated machine certificate lifecycle;
- Federation — trust across domains and organizations;
- Integrations — Kubernetes, cloud, CI and infrastructure systems;
- Security intelligence — identity/access graph, anomalies and investigation context.
