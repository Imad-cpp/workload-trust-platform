# ADR-0004 — Use PostgreSQL for product state

Status: Accepted  
Date: 2026-08-14

## Context

The control plane needs durable relational state for organizations, workloads, registration rules, policy versions, audit metadata and reconciliation state.

## Decision

Use PostgreSQL as the product-state source of truth.

SPIRE retains authority for the identity-runtime state it owns; the product database stores desired/product state and integration observations, not workload private keys.

## Consequences

- reconciliation logic must make desired versus observed state explicit;
- migration and rollback tests are release requirements;
- application tables must not become a duplicate secret/PKI store.
