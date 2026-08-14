# 13 — SPIFFE/SPIRE Identity Lab

Status: Phase 1 implementation  
Date: 2026-08-14

## Purpose

The identity lab proves the first security-critical product invariant with a real SPIRE runtime:

> a Docker workload receives a SPIFFE identity only when runtime-derived selectors match an operator-created registration entry.

This phase does **not** implement service authorization or mTLS policy enforcement. Those belong to later phases.

## Runtime shape

SPIRE Server and SPIRE Agent run directly on the Linux host. Demo workload probes run as Docker containers and connect only to the local SPIFFE Workload API Unix socket.

```text
Linux host
├── SPIRE Server
├── SPIRE Agent
│   ├── Docker WorkloadAttestor
│   ├── /var/run/docker.sock
│   └── /tmp/workload-trust-lab/agent.sock
└── Docker workload probes
    ├── frontend
    ├── orders-api
    ├── payment-api
    ├── admin-api
    ├── untrusted
    └── frontend-wrong-environment
```

The host-native agent is intentional for this lab: it gives the Docker WorkloadAttestor direct access to the host process/cgroup view and Docker Engine API without creating a nested container trust boundary just to run the attestor.

## Trust domain

The lab uses:

```text
workload-trust.test
```

The `.test` top-level domain is reserved for testing. This is a lab naming decision, not a final product/company domain.

## Agent bootstrap

The lab uses the SPIRE `join_token` NodeAttestor and requests the stable lab agent identity:

```text
spiffe://workload-trust.test/spire/agent/lab
```

Join tokens are one-time bootstrap credentials. The agent config also uses `insecure_bootstrap = true` **only for the local lab** to bootstrap trust in the local SPIRE Server. This is explicitly not a production node-attestation/bootstrap design.

## Workload selectors

Each registered workload requires two Docker-label selectors:

```text
docker:label:com.workload-trust.name:<workload>
docker:label:com.workload-trust.environment:lab
```

Example:

```text
docker:label:com.workload-trust.name:frontend
docker:label:com.workload-trust.environment:lab
```

Expected identity:

```text
spiffe://workload-trust.test/lab/frontend
```

Labels are appropriate for proving Docker workload-attestation mechanics in a controlled lab. They are **not** claimed as a sufficient high-assurance production selector if an attacker controls Docker workload creation and can freely choose labels. Stronger production attestation is a later design problem.

## Positive tests

The lab must prove all four expected identities:

| Container | Expected SPIFFE ID |
|---|---|
| frontend | `spiffe://workload-trust.test/lab/frontend` |
| orders-api | `spiffe://workload-trust.test/lab/orders-api` |
| payment-api | `spiffe://workload-trust.test/lab/payment-api` |
| admin-api | `spiffe://workload-trust.test/lab/admin-api` |

## Negative tests

Two negative probes are release-blocking for this phase:

1. `untrusted`: valid lab environment label but no registered workload name.
2. `frontend-wrong-environment`: valid frontend name but `environment=dev`, proving the two selectors are conjunctive rather than accepting a partial match.

Neither may receive any registered `spiffe://workload-trust.test/lab/...` identity.

## Supply-chain pinning

`scripts/fetch-spire.sh` downloads SPIRE `v1.15.2` release archives and verifies the upstream release SHA-256 before installing the binaries into `.lab/bin`.

Supported lab architectures:

- Linux amd64
- Linux arm64

The `.lab` directory is ignored by Git and contains ephemeral binaries, state, logs and private identity runtime material.

## Run locally

Requirements:

- Linux;
- Docker Engine + Compose plugin;
- `curl`, `tar`, `sha256sum`, standard POSIX/Linux shell tools;
- permission to access `/var/run/docker.sock`.

Run:

```bash
./scripts/lab-up.sh
./scripts/lab-register.sh
./scripts/lab-verify.sh
./scripts/lab-down.sh
```

Or execute the CI-equivalent lifecycle:

```bash
./scripts/lab-ci.sh
```

## Security boundaries

- no workload private key is copied into the repository or host application database;
- workload probes use the local Workload API socket;
- the lab containers run with `network_mode: none` because identity retrieval requires no application network access;
- the join token is held only in shell memory for agent startup and is not printed or committed;
- runtime state is ephemeral and excluded from Git;
- authorization has not been implemented yet, so a successful SVID must not be described as permission to call another service.
