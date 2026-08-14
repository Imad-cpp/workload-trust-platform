# ADR-0003 — Keep V1 control plane modular, not microservices

Status: Accepted  
Date: 2026-08-14

## Context

The long-term product may eventually need separately scalable components. V1 does not yet have evidence that independent services are worth the operational and security complexity.

## Decision

Build one deployable Go control plane with explicit internal modules. Split services only when load, trust boundary or team ownership provides evidence for it.

## Consequences

- simpler local/reproducible deployment;
- fewer network trust boundaries during V1;
- domain contracts must remain explicit to avoid a monolithic tangle.
