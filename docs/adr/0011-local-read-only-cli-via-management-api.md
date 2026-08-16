# ADR-0011 — Local read-only CLI through the management API

Status: Accepted  
Date: 2026-08-16

## Context

Phase 2 needs an operator inspection surface before the console phase. A CLI that opens PostgreSQL directly would bypass the same authentication/authorization boundary used by the management API, distribute database credentials to another process and create a second interpretation of product state.

The current management listener is intentionally loopback-only. This phase does not approve remote management.

## Decision

Implement `wtpctl` as a **read-only client of the existing loopback management HTTP API**.

The CLI does not connect directly to PostgreSQL and does not accept `DATABASE_URL` as a requirement.

Current commands:

```text
wtpctl health
wtpctl ready
wtpctl workloads --organization-id <uuid>
wtpctl status --organization-id <uuid>
```

`health` and `ready` use the generic unauthenticated endpoints. `workloads` and `status` use the same bearer authentication/authorization boundary as other `/v1/*` reads.

`status` consumes `GET /v1/diagnostics?organization_id=<uuid>`, protected by `diagnostics:read`. The diagnostics payload is aggregate-only:

- workload health counts;
- registration reconciliation counts;
- policy lifecycle counts;
- audit/security-event counts;
- latest audit timestamp.

It does not expose registration selectors/parent identity, policy rule identities/change reasons, audit metadata or credentials.

## Local transport boundary

For this Phase 2 CLI:

- `WTP_API_URL` must use plain `http` and target `localhost` or a loopback IP;
- URL userinfo, path, query and fragment are rejected;
- redirects are not followed;
- authenticated commands require the same high-entropy bearer token format as the server;
- HTTP requests have a bounded timeout;
- response bodies are bounded to 1 MiB;
- non-2xx errors expose only HTTP status and safe machine error code, not raw response bodies.

Plain loopback HTTP is accepted only because the server and CLI are intentionally local to the same Phase 2 host boundary. This ADR does not approve remote plaintext management traffic.

## Consequences

### Positive

- one server-side authorization model remains authoritative;
- CLI users do not need PostgreSQL credentials;
- diagnostics cannot silently bypass API redaction rules;
- the later console can reuse the same management API boundary instead of learning a second data path.

### Limits

- this is not remote administration;
- the CLI adds no mutation commands;
- `workloads` intentionally exposes the existing authorized workload inventory, including workload SPIFFE IDs;
- `status` is aggregate diagnostics, not full audit/security-event inspection;
- the single configured local principal/session limitation remains.

## Evidence

Permanent CLI integration CI runs a real PostgreSQL-backed control plane and the compiled `wtpctl` binary, proving health/ready/status/workloads, viewer authorization, wrong-token refusal, non-loopback rejection, operation without `DATABASE_URL` in the CLI environment, safe aggregate status output and no audit mutation caused by diagnostic reads.
