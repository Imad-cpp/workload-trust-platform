# 17 — Operator Registration Mutation Evidence

Status: Phase 2 implemented slice  
Date: 2026-08-14

## Claim

The current **local** registration mutation surface is authenticated, server-authorized, bounded/validated, optimistic-concurrency protected and transactionally audited.

This evidence applies to the currently implemented registration desired-state routes only. It does not establish remote/multi-user operator security or workload service authorization.

## Live HTTP + PostgreSQL evidence

Permanent `Operator Mutation Integration` CI starts a fresh PostgreSQL database and the real control-plane binary with synthetic credentials. It proves:

1. unauthenticated `POST /v1/registration-rules` returns `401`;
2. authenticated `viewer` POST returns `403` and creates neither registration nor mutation-audit state;
3. the same viewer can still perform its allowed workload read (`200`);
4. authenticated `operator` create returns `201`, persists revision 1 as `pending`, and writes exactly one audit event attributed to the configured operator ID;
5. mutation response does not echo parent SPIFFE ID or selector value;
6. authorized full replacement returns `200`, persists revision 2 as `pending`, and appends exactly one replacement audit event;
7. a stale replacement using expected revision 1 returns `409`, leaves revision 2 unchanged and appends no extra replacement audit;
8. operator mutation audit metadata contains no parent SPIFFE ID, selector value or `join_token` material.

## Real PostgreSQL transactional evidence

The Go quality suite additionally installs a test-only PostgreSQL trigger that intentionally rejects the operator audit insert. `CreateDesired` then fails and the suite verifies that the registration row count remains zero.

That test proves mutation and accountability are not best-effort sequential writes: the state change and audit evidence are in one PostgreSQL transaction.

## Input/error evidence

Go HTTP tests cover:

- unknown JSON field rejection;
- non-JSON media rejection;
- 64 KiB body limit;
- invalid desired-state/SPIFFE/selector/TTL validation;
- internal database error redaction;
- authentication before authorization;
- viewer denial before mutation service execution.

## Separation from SPIRE

The operator routes do not call SPIRE directly. They produce pending PostgreSQL desired state. The separately evidenced SPIRE reconciler consumes that desired state and performs ownership-safe convergence.

## Non-claims

This slice does **not** prove:

- service-to-service authorization or default-deny enforcement;
- remote operator transport/session security;
- OAuth/SSO or enterprise/multi-user RBAC;
- policy activation safety;
- production-grade node attestation, HA or production readiness.
