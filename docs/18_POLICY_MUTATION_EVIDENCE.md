# 18 — Policy Versioning and Activation Evidence

Status: Phase 2 implemented slice  
Date: 2026-08-16

## What this slice proves

The control plane now manages versioned policy desired state through authenticated and server-authorized local HTTP routes while preserving immutable history and transactionally coupled operator audit evidence.

### Management authorization

- every `/v1/*` request is authenticated before route authorization;
- `viewer` can receive `policies:read` but cannot use `policies:write` or `policies:activate`;
- `operator` receives the current local policy-management permissions;
- activation uses a distinct `policies:activate` permission;
- the policy manager also rejects non-operator mutation actors as defense in depth.

### Policy state

Current V1 policy content is intentionally narrow:

```text
source SPIFFE ID
destination SPIFFE ID
action = connect
effect = allow | deny
```

The slice proves:

1. Policy creation produces a `draft` policy at revision 1 and immutable version 1.
2. Appending policy content creates a new immutable version and advances the policy revision.
3. A stale `expected_revision` returns conflict without creating another version or audit event.
4. Activation selects an existing immutable version that belongs to the target policy and advances the policy revision.
5. A version belonging to another policy cannot be activated.
6. A malformed version inserted directly as a test fixture cannot be activated because stored content is revalidated before activation.
7. `access_policy_versions` continues to reject UPDATE and DELETE at the PostgreSQL layer.

### Transactional audit behavior

Policy create, version append and activation place the product-state change and attributed operator audit record in the **same PostgreSQL transaction**.

Real PostgreSQL tests intentionally force audit insertion failures and prove:

- version append rolls back;
- policy revision does not advance;
- activation rolls back;
- active-version pointer/status do not change.

Audit summaries contain safe policy/version/revision/effect metadata but deliberately omit source/destination SPIFFE IDs and change-reason text.

### Migration evidence

`000003_policy_revision` is part of the actual migration runner, not a detached migration file. Permanent CI proves:

```text
000001 up -> 000002 up -> 000003 up
migration invariants
000003 down -> 000002 down -> 000001 down
000001 up -> 000002 up -> 000003 up
semantic revision-default verification
```

The control-plane quality workflow also parses each relevant shell script individually before executing migration tests.

## Live HTTP evidence

`Policy Mutation Integration` exercises a real control-plane process and real PostgreSQL service. It proves:

```text
unauthenticated policy create -> 401
viewer policy create -> 403, no state/audit change
operator create -> 201
append immutable version -> 201
stale append -> 409, no extra state/audit
activate selected version -> 200
foreign-version activation -> 404, active state unchanged
policy response/audit leak checks -> pass
direct immutable version UPDATE -> refused
```

## Claim boundary

This slice proves **policy management and activation state**, not workload service authorization.

`active_version_id` means the control plane has selected the policy version intended for later evaluation. It does **not** prove that a service request is authenticated, evaluated, allowed or denied at an enforcement point.

Still deferred to Phase 3:

- mTLS service path;
- verified source SPIFFE identity extraction at the enforcement boundary;
- deterministic default-deny evaluator;
- allowed/forbidden service-call integration tests;
- policy absence/corruption fail-closed behavior at enforcement time.

No production-readiness claim is made.
