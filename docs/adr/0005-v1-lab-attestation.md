# ADR-0005 — Use host-native SPIRE with Docker-label attestation for the V1 identity lab

Status: Accepted  
Date: 2026-08-14

## Context

Phase 1 must prove real workload attestation before control-plane development begins. Running the SPIRE Agent inside another container would require additional PID/cgroup and Docker-socket plumbing and would make the first proof depend on a containerized-agent topology that is not itself a product requirement.

The SPIRE Docker WorkloadAttestor derives selectors from runtime container information and Docker Engine metadata.

## Decision

For the Phase 1 Linux reference lab:

- run SPIRE Server and SPIRE Agent directly on the Linux host;
- use SPIRE `v1.15.2` release binaries with upstream SHA-256 verification;
- use trust domain `workload-trust.test`;
- bootstrap the lab agent with the one-time `join_token` NodeAttestor and use SPIRE's generated reserved-namespace agent identity as the registration parent;
- use `insecure_bootstrap = true` only to bootstrap trust in this local test environment;
- use two Docker labels as workload selectors: workload name and `environment=lab`;
- expose the Workload API to probe containers through a Unix socket mount;
- run workload probe containers with networking disabled.

## Consequences

### Positive

- the agent observes Docker workloads from the host namespace naturally;
- CI tests the real Docker WorkloadAttestor instead of a mocked selector source;
- the topology is easy to reproduce on GitHub-hosted Linux runners;
- no privileged workload container is required.

### Limitations

- this is a Linux-only lab;
- join-token + insecure bootstrap are not accepted as the production node-attestation/bootstrap model;
- the join-token attestor embeds the consumed token value in the generated agent ID, so that parent identity remains ephemeral lab state;
- Docker labels are not a high-assurance identity source against an attacker who controls Docker workload creation;
- mTLS authorization is intentionally outside this ADR/phase.

These limitations must remain visible until stronger production deployment/attestation ADRs replace the lab choices.
