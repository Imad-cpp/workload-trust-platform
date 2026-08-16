# 19 — CLI Diagnostics Evidence

Status: Phase 2 implemented slice  
Date: 2026-08-16

## What this slice proves

The project now has a local read-only operator CLI that uses the same management HTTP authentication/authorization boundary as the control plane. It does not connect directly to PostgreSQL and does not add another management write path.

## Commands

```text
wtpctl health
wtpctl ready
wtpctl workloads --organization-id <uuid>
wtpctl status --organization-id <uuid>
```

`health`/`ready` consume generic public liveness/readiness endpoints. `workloads` and `status` require the configured bearer credential.

## Safe diagnostics surface

`status` consumes an authenticated `diagnostics:read` endpoint and returns aggregate fields only:

- workload total/health counts;
- registration pending/converged/error counts;
- policy draft/active/disabled counts;
- audit/security-event counts;
- latest audit timestamp.

The diagnostics response intentionally contains no:

- registration selectors;
- parent SPIFFE identity;
- policy source/destination SPIFFE identities;
- policy change reasons;
- audit/security metadata;
- bearer token.

`workloads` remains the existing authorized inventory command and may contain workload SPIFFE IDs; it is not the redacted aggregate status surface.

## CLI transport/security behavior

Permanent/unit evidence proves:

- non-loopback `WTP_API_URL` values are rejected;
- only local HTTP management URLs are accepted in this phase;
- URL credentials/path/query/fragment are rejected;
- redirects are not followed;
- authenticated commands reject short/whitespace-bearing bearer tokens;
- management requests have a five-second client timeout;
- responses over 1 MiB are rejected without partial output;
- non-2xx responses expose safe status/error code only rather than the raw response body;
- health does not send an Authorization header.

## Live integration evidence

`CLI Diagnostics Integration` runs a real PostgreSQL service, applies the current migrations, builds the control plane and `wtpctl`, seeds synthetic diagnostics state and launches a viewer control-plane process.

It proves:

```text
wtpctl health -> valid JSON
wtpctl ready -> valid JSON
viewer wtpctl status -> success
viewer wtpctl workloads -> success
wrong bearer token -> safe 401 failure
non-loopback WTP_API_URL -> rejected before request
status output -> aggregate-only leak checks pass
DATABASE_URL removed from CLI environment -> commands still work
read-only commands -> audit-event count unchanged
```

The Go real-PostgreSQL suite separately verifies the aggregate diagnostics repository and HTTP error/redaction behavior.

## Claim boundary

This slice provides local operational inspection only. It does not:

- enable remote management;
- add CLI mutations;
- provide full audit-event contents;
- replace the later operator console;
- implement workload service authorization;
- claim production readiness.
