#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="${ROOT}/.lab/bin/spire-server"
SERVER_SOCKET="/tmp/workload-trust-lab/server.sock"
AGENT_ID_FILE="${ROOT}/.lab/agent-id"
COMPOSE="${ROOT}/examples/identity-lab/compose.yaml"
TRUST_PREFIX="spiffe://workload-trust.test/lab/"
DATABASE_URL="${DATABASE_URL:-}"

[[ -n "${DATABASE_URL}" ]] || { echo "DATABASE_URL is required" >&2; exit 1; }

cleanup() {
  unset PARENT_ID
  "${ROOT}/scripts/lab-down.sh" >/dev/null 2>&1 || true
}
trap cleanup EXIT

sql_scalar() {
  psql "${DATABASE_URL}" -X -qAt -v ON_ERROR_STOP=1 "$@"
}

run_reconciler() {
  DATABASE_URL="${DATABASE_URL}" \
  WTP_SPIRE_SERVER_SOCKET="${SERVER_SOCKET}" \
    go run ./apps/reconciler
}

probe() {
  local service="$1"
  docker compose -f "${COMPOSE}" run --rm --no-deps "${service}" 2>&1
}

wait_identity() {
  local service="$1"
  local expected="$2"
  local output=""
  local status=1
  local deadline=$((SECONDS + 20))

  while (( SECONDS < deadline )); do
    set +e
    output="$(probe "${service}")"
    status=$?
    set -e

    if [[ ${status} -eq 0 ]] && grep -Fq "${expected}" <<<"${output}"; then
      echo "PASS reconciled identity: ${service} -> ${expected}"
      return 0
    fi
    if grep -Fq "${TRUST_PREFIX}" <<<"${output}" && ! grep -Fq "${expected}" <<<"${output}"; then
      echo "Unexpected workload identity returned for ${service}." >&2
      return 1
    fi
    sleep 1
  done

  echo "Identity did not converge for ${service}." >&2
  return 1
}

wait_no_identity() {
  local service="$1"
  local output=""
  local deadline=$((SECONDS + 20))

  while (( SECONDS < deadline )); do
    set +e
    output="$(probe "${service}")"
    set -e
    if ! grep -Fq "${TRUST_PREFIX}" <<<"${output}"; then
      echo "PASS reconciled removal: ${service} received no lab identity"
      return 0
    fi
    sleep 1
  done

  echo "A lab identity remained available to ${service}." >&2
  return 1
}

cd "${ROOT}"
"${ROOT}/scripts/lab-up.sh"
"${ROOT}/scripts/db-migrate.sh" up

[[ -x "${SERVER}" ]] || { echo "SPIRE server binary is unavailable." >&2; exit 1; }
[[ -s "${AGENT_ID_FILE}" ]] || { echo "Attested agent identity state is unavailable." >&2; exit 1; }
PARENT_ID="$(cat "${AGENT_ID_FILE}")"

ORG_ID="$(sql_scalar -c "INSERT INTO organizations (slug, display_name) VALUES ('reconcile-ci', 'Reconciliation CI') RETURNING id::text")"
TRUST_DOMAIN_ID="$(sql_scalar -v org="${ORG_ID}" <<'SQL'
INSERT INTO trust_domains (organization_id, name)
VALUES (:'org'::uuid, 'workload-trust.test')
RETURNING id::text;
SQL
)"
WORKLOAD_ID="$(sql_scalar -v org="${ORG_ID}" -v trust_domain="${TRUST_DOMAIN_ID}" <<'SQL'
INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id)
VALUES (:'org'::uuid, :'trust_domain'::uuid, 'frontend', 'lab', 'spiffe://workload-trust.test/lab/frontend')
RETURNING id::text;
SQL
)"
RULE_ID="$(sql_scalar -v org="${ORG_ID}" -v workload="${WORKLOAD_ID}" -v parent="${PARENT_ID}" <<'SQL'
INSERT INTO registration_rules (
  organization_id,
  workload_id,
  selectors,
  desired_state,
  parent_spiffe_id,
  x509_svid_ttl_seconds
) VALUES (
  :'org'::uuid,
  :'workload'::uuid,
  '["docker:label:com.workload-trust.name:frontend","docker:label:com.workload-trust.environment:lab"]'::jsonb,
  'present',
  :'parent',
  60
)
RETURNING id::text;
SQL
)"

# Create from desired PostgreSQL state through the official SPIRE Entry API.
run_reconciler
IFS='|' read -r STATUS ENTRY_ID <<<"$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT reconcile_status || '|' || COALESCE(spire_entry_id, '')
FROM registration_rules
WHERE id = :'rule'::uuid;
SQL
)"
[[ "${STATUS}" == "converged" && -n "${ENTRY_ID}" ]] || {
  echo "Registration rule did not converge after create." >&2
  exit 1
}
ENTRY_VIEW="$(${SERVER} entry show -socketPath "${SERVER_SOCKET}" -entryID "${ENTRY_ID}" 2>/dev/null)"
grep -Fq "wtp-rule:${RULE_ID}" <<<"${ENTRY_VIEW}" || { echo "Owned SPIRE entry marker is missing." >&2; exit 1; }
grep -Fq "spiffe://workload-trust.test/lab/frontend" <<<"${ENTRY_VIEW}" || { echo "Reconciled SPIFFE ID is missing." >&2; exit 1; }
wait_identity frontend "spiffe://workload-trust.test/lab/frontend"

# Drift the desired selector/TTL and require in-place convergence on the same SPIRE entry.
sql_scalar -v rule="${RULE_ID}" <<'SQL' >/dev/null
UPDATE registration_rules
SET
  selectors = '["docker:label:com.workload-trust.name:frontend","docker:label:com.workload-trust.environment:dev"]'::jsonb,
  x509_svid_ttl_seconds = 120,
  revision = revision + 1,
  reconcile_status = 'pending',
  updated_at = now()
WHERE id = :'rule'::uuid;
SQL
run_reconciler
IFS='|' read -r DRIFT_STATUS DRIFT_ENTRY_ID <<<"$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT reconcile_status || '|' || COALESCE(spire_entry_id, '')
FROM registration_rules
WHERE id = :'rule'::uuid;
SQL
)"
[[ "${DRIFT_STATUS}" == "converged" && "${DRIFT_ENTRY_ID}" == "${ENTRY_ID}" ]] || {
  echo "Owned SPIRE drift was not converged in place." >&2
  exit 1
}
wait_identity frontend-wrong-environment "spiffe://workload-trust.test/lab/frontend"
wait_no_identity frontend

# Create a foreign entry matching a second desired rule. The reconciler must
# observe ALREADY_EXISTS but refuse to adopt or mutate the entry because its
# ownership hint does not belong to that rule.
FOREIGN_WORKLOAD_ID="$(sql_scalar -v org="${ORG_ID}" -v trust_domain="${TRUST_DOMAIN_ID}" <<'SQL'
INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id)
VALUES (:'org'::uuid, :'trust_domain'::uuid, 'foreign-ci', 'lab', 'spiffe://workload-trust.test/lab/foreign-ci')
RETURNING id::text;
SQL
)"
FOREIGN_RULE_ID="$(sql_scalar -v org="${ORG_ID}" -v workload="${FOREIGN_WORKLOAD_ID}" -v parent="${PARENT_ID}" <<'SQL'
INSERT INTO registration_rules (
  organization_id,
  workload_id,
  selectors,
  desired_state,
  parent_spiffe_id,
  x509_svid_ttl_seconds
) VALUES (
  :'org'::uuid,
  :'workload'::uuid,
  '["docker:label:com.workload-trust.name:untrusted","docker:label:com.workload-trust.environment:lab"]'::jsonb,
  'present',
  :'parent',
  60
)
RETURNING id::text;
SQL
)"
FOREIGN_ENTRY_ID="foreign-ci-entry"
"${SERVER}" entry create \
  -socketPath "${SERVER_SOCKET}" \
  -entryID "${FOREIGN_ENTRY_ID}" \
  -parentID "${PARENT_ID}" \
  -spiffeID "spiffe://workload-trust.test/lab/foreign-ci" \
  -x509SVIDTTL 60 \
  -selector "docker:label:com.workload-trust.name:untrusted" \
  -selector "docker:label:com.workload-trust.environment:lab" \
  -hint "foreign-controller" >/dev/null

set +e
run_reconciler >/tmp/workload-trust-reconcile-expected-conflict.log 2>&1
CONFLICT_STATUS=$?
set -e
[[ ${CONFLICT_STATUS} -ne 0 ]] || { echo "Foreign ownership conflict unexpectedly succeeded." >&2; exit 1; }
IFS='|' read -r FOREIGN_STATUS FOREIGN_ERROR FOREIGN_BOUND_ID <<<"$(sql_scalar -v rule="${FOREIGN_RULE_ID}" <<'SQL'
SELECT reconcile_status || '|' || COALESCE(last_error_code, '') || '|' || COALESCE(spire_entry_id, '')
FROM registration_rules
WHERE id = :'rule'::uuid;
SQL
)"
[[ "${FOREIGN_STATUS}" == "error" && "${FOREIGN_ERROR}" == "ownership_mismatch" && -z "${FOREIGN_BOUND_ID}" ]] || {
  echo "Foreign entry ownership conflict was not recorded safely." >&2
  exit 1
}
FOREIGN_VIEW="$(${SERVER} entry show -socketPath "${SERVER_SOCKET}" -entryID "${FOREIGN_ENTRY_ID}" 2>/dev/null)"
grep -Fq "foreign-controller" <<<"${FOREIGN_VIEW}" || { echo "Foreign SPIRE entry was mutated or removed." >&2; exit 1; }
echo "PASS ownership safety: foreign SPIRE entry was not adopted or mutated"

# Remove the foreign fixture explicitly, mark that rule absent, and require the
# owned entry to be removed through reconciliation.
"${SERVER}" entry delete -socketPath "${SERVER_SOCKET}" -entryID "${FOREIGN_ENTRY_ID}" >/dev/null
sql_scalar -v owned="${RULE_ID}" -v foreign="${FOREIGN_RULE_ID}" <<'SQL' >/dev/null
UPDATE registration_rules
SET desired_state = 'absent', revision = revision + 1, reconcile_status = 'pending', updated_at = now()
WHERE id IN (:'owned'::uuid, :'foreign'::uuid);
SQL
run_reconciler
IFS='|' read -r DELETE_STATUS DELETE_ENTRY_ID <<<"$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT reconcile_status || '|' || COALESCE(spire_entry_id, '')
FROM registration_rules
WHERE id = :'rule'::uuid;
SQL
)"
[[ "${DELETE_STATUS}" == "converged" && -z "${DELETE_ENTRY_ID}" ]] || {
  echo "Owned SPIRE deletion did not converge." >&2
  exit 1
}
POST_DELETE_VIEW="$(${SERVER} entry show -socketPath "${SERVER_SOCKET}" -entryID "${ENTRY_ID}" 2>/dev/null || true)"
if grep -Fq "${ENTRY_ID}" <<<"${POST_DELETE_VIEW}"; then
  echo "Owned SPIRE entry still exists after desired-state deletion." >&2
  exit 1
fi
wait_no_identity frontend-wrong-environment

CREATE_AUDITS="$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE target_id = :'rule' AND metadata->>'operation' = 'create';
SQL
)"
UPDATE_AUDITS="$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE target_id = :'rule' AND metadata->>'operation' = 'update';
SQL
)"
DELETE_AUDITS="$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE target_id = :'rule' AND metadata->>'operation' = 'delete';
SQL
)"
CONFLICT_AUDITS="$(sql_scalar -v rule="${FOREIGN_RULE_ID}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE target_id = :'rule' AND metadata->>'operation' = 'ownership_conflict';
SQL
)"
LEAK_AUDITS="$(sql_scalar -c "SELECT count(*) FROM audit_events WHERE action = 'spire.registration_reconcile' AND metadata::text LIKE '%join_token%'")"
[[ "${CREATE_AUDITS}" -ge 1 && "${UPDATE_AUDITS}" -ge 1 && "${DELETE_AUDITS}" -ge 1 && "${CONFLICT_AUDITS}" -ge 1 ]] || {
  echo "Expected reconciliation audit evidence is incomplete." >&2
  exit 1
}
[[ "${LEAK_AUDITS}" == "0" ]] || { echo "Sensitive parent identity material leaked into audit metadata." >&2; exit 1; }

echo "SPIRE desired-state reconciliation integration passed."
