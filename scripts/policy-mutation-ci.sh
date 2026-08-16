#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-}"
VIEWER_PORT=18083
OPERATOR_PORT=18084
VIEWER_TOKEN="$(openssl rand -hex 32)"
OPERATOR_TOKEN="$(openssl rand -hex 32)"
SERVER_PID=""

[[ -n "${DATABASE_URL}" ]] || { echo "DATABASE_URL is required" >&2; exit 1; }

cleanup() {
  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  unset VIEWER_TOKEN OPERATOR_TOKEN
  rm -f /tmp/wtp-policy-viewer.log /tmp/wtp-policy-operator.log /tmp/wtp-policy-body.json
}
trap cleanup EXIT

sql_scalar() {
  psql "${DATABASE_URL}" -X -qAt -v ON_ERROR_STOP=1 "$@"
}

stop_server() {
  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
    SERVER_PID=""
  fi
}

start_server() {
  local role="$1"
  local port="$2"
  local token="$3"
  local actor_id="$4"
  local logfile="/tmp/wtp-policy-${role}.log"

  WTP_LISTEN_ADDR="127.0.0.1:${port}" \
  WTP_OPERATOR_ID="${actor_id}" \
  WTP_OPERATOR_ROLE="${role}" \
  WTP_OPERATOR_TOKEN="${token}" \
  DATABASE_URL="${DATABASE_URL}" \
    "${ROOT}/.lab/control-plane" >"${logfile}" 2>&1 &
  SERVER_PID=$!

  local deadline=$((SECONDS + 15))
  while (( SECONDS < deadline )); do
    if curl --silent --show-error --fail "http://127.0.0.1:${port}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
      echo "Control plane exited during policy test startup for role ${role}." >&2
      sed -E 's#(postgres(ql)?://)[^@]+@#\1[redacted]@#g' "${logfile}" >&2 || true
      return 1
    fi
    sleep 0.5
  done
  echo "Control plane did not become healthy for policy role ${role}." >&2
  return 1
}

http_status() {
  curl --silent --show-error \
    --output /tmp/wtp-policy-body.json \
    --write-out '%{http_code}' \
    "$@"
}

assert_response_safe() {
  local source_id="$1"
  local destination_id="$2"
  local reason="$3"
  if grep -Fq "${source_id}" /tmp/wtp-policy-body.json || \
     grep -Fq "${destination_id}" /tmp/wtp-policy-body.json || \
     grep -Fq "${reason}" /tmp/wtp-policy-body.json || \
     grep -Fq 'source_spiffe_id' /tmp/wtp-policy-body.json || \
     grep -Fq 'destination_spiffe_id' /tmp/wtp-policy-body.json || \
     grep -Fq 'change_reason' /tmp/wtp-policy-body.json; then
    echo "Policy response exposed rule material." >&2
    exit 1
  fi
}

cd "${ROOT}"
mkdir -p .lab
./scripts/db-migrate.sh up
go build -trimpath -o .lab/control-plane ./apps/control-plane

ORG_ID="$(sql_scalar -c "INSERT INTO organizations (slug, display_name) VALUES ('policy-mutation-ci', 'Policy Mutation CI') RETURNING id::text")"
SOURCE_ID='spiffe://policy-mutation.test/prod/frontend'
DESTINATION_ID='spiffe://policy-mutation.test/prod/orders-api'
CREATE_REASON='reviewed-initial-route-do-not-echo'
APPEND_REASON='reviewed-deny-change-do-not-echo'
CREATE_BODY="$(printf '{\"organization_id\":\"%s\",\"name\":\"frontend to orders\",\"initial_version\":{\"source_spiffe_id\":\"%s\",\"destination_spiffe_id\":\"%s\",\"action\":\"connect\",\"effect\":\"allow\",\"change_reason\":\"%s\"}}' "${ORG_ID}" "${SOURCE_ID}" "${DESTINATION_ID}" "${CREATE_REASON}")"

start_server viewer "${VIEWER_PORT}" "${VIEWER_TOKEN}" viewer-policy-ci

STATUS="$(http_status \
  --request POST \
  --header 'Content-Type: application/json' \
  --data "${CREATE_BODY}" \
  "http://127.0.0.1:${VIEWER_PORT}/v1/policies")"
[[ "${STATUS}" == "401" ]] || { echo "Unauthenticated policy mutation returned ${STATUS}, want 401." >&2; exit 1; }

STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${VIEWER_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${CREATE_BODY}" \
  "http://127.0.0.1:${VIEWER_PORT}/v1/policies")"
[[ "${STATUS}" == "403" ]] || { echo "Viewer policy mutation returned ${STATUS}, want 403." >&2; exit 1; }

PRE_POLICIES="$(sql_scalar -v org="${ORG_ID}" -c "SELECT count(*) FROM access_policies WHERE organization_id = :'org'::uuid")"
PRE_AUDITS="$(sql_scalar -c "SELECT count(*) FROM audit_events WHERE action LIKE 'access_policy.%'")"
[[ "${PRE_POLICIES}" == "0" && "${PRE_AUDITS}" == "0" ]] || {
  echo "Unauthorized policy requests changed persistent state." >&2
  exit 1
}
echo "PASS policy authorization: unauthenticated=401 viewer=403 no-state-change"

stop_server
start_server operator "${OPERATOR_PORT}" "${OPERATOR_TOKEN}" operator-policy-ci

STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${CREATE_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/policies")"
[[ "${STATUS}" == "201" ]] || { echo "Operator policy create returned ${STATUS}, want 201." >&2; exit 1; }
assert_response_safe "${SOURCE_ID}" "${DESTINATION_ID}" "${CREATE_REASON}"

POLICY_ID="$(sql_scalar -v org="${ORG_ID}" -c "SELECT id::text FROM access_policies WHERE organization_id = :'org'::uuid AND name = 'frontend to orders'")"
VERSION1_ID="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT id::text FROM access_policy_versions WHERE policy_id = :'policy'::uuid AND version = 1")"
[[ -n "${POLICY_ID}" && -n "${VERSION1_ID}" ]] || { echo "Created policy/version missing." >&2; exit 1; }
IFS='|' read -r POLICY_STATUS POLICY_REVISION ACTIVE_COUNT VERSION_COUNT <<<"$(sql_scalar -v policy="${POLICY_ID}" <<'SQL'
SELECT status || '|' || revision::text || '|' ||
       CASE WHEN active_version_id IS NULL THEN '0' ELSE '1' END || '|' ||
       (SELECT count(*)::text FROM access_policy_versions WHERE policy_id = :'policy'::uuid)
FROM access_policies
WHERE id = :'policy'::uuid;
SQL
)"
[[ "${POLICY_STATUS}" == "draft" && "${POLICY_REVISION}" == "1" && "${ACTIVE_COUNT}" == "0" && "${VERSION_COUNT}" == "1" ]] || {
  echo "Created policy state is inconsistent." >&2
  exit 1
}
CREATE_AUDITS="$(sql_scalar -v policy="${POLICY_ID}" <<'SQL'
SELECT count(*) FROM audit_events
WHERE target_id = :'policy'
  AND action = 'access_policy.create_with_version'
  AND actor_type = 'operator'
  AND actor_id = 'operator-policy-ci';
SQL
)"
[[ "${CREATE_AUDITS}" == "1" ]] || { echo "Policy create audit attribution is incorrect." >&2; exit 1; }

APPEND_BODY="$(printf '{\"expected_revision\":1,\"source_spiffe_id\":\"%s\",\"destination_spiffe_id\":\"%s\",\"action\":\"connect\",\"effect\":\"deny\",\"change_reason\":\"%s\"}' "${SOURCE_ID}" "${DESTINATION_ID}" "${APPEND_REASON}")"
STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${APPEND_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/policies/${POLICY_ID}/versions")"
[[ "${STATUS}" == "201" ]] || { echo "Policy version append returned ${STATUS}, want 201." >&2; exit 1; }
assert_response_safe "${SOURCE_ID}" "${DESTINATION_ID}" "${APPEND_REASON}"

VERSION2_ID="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT id::text FROM access_policy_versions WHERE policy_id = :'policy'::uuid AND version = 2")"
POLICY_REVISION="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT revision FROM access_policies WHERE id = :'policy'::uuid")"
VERSION_COUNT="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT count(*) FROM access_policy_versions WHERE policy_id = :'policy'::uuid")"
APPEND_AUDITS="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT count(*) FROM audit_events WHERE target_id = :'policy' AND action = 'access_policy.append_version' AND actor_id = 'operator-policy-ci'")"
[[ -n "${VERSION2_ID}" && "${POLICY_REVISION}" == "2" && "${VERSION_COUNT}" == "2" && "${APPEND_AUDITS}" == "1" ]] || {
  echo "Appended policy version state/audit is inconsistent." >&2
  exit 1
}

STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${APPEND_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/policies/${POLICY_ID}/versions")"
[[ "${STATUS}" == "409" ]] || { echo "Stale policy revision returned ${STATUS}, want 409." >&2; exit 1; }
STALE_REVISION="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT revision FROM access_policies WHERE id = :'policy'::uuid")"
STALE_VERSIONS="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT count(*) FROM access_policy_versions WHERE policy_id = :'policy'::uuid")"
STALE_AUDITS="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT count(*) FROM audit_events WHERE target_id = :'policy' AND action = 'access_policy.append_version'")"
[[ "${STALE_REVISION}" == "2" && "${STALE_VERSIONS}" == "2" && "${STALE_AUDITS}" == "1" ]] || {
  echo "Stale policy append changed persistent/audit state." >&2
  exit 1
}

ACTIVATE_BODY="$(printf '{\"expected_revision\":2,\"version_id\":\"%s\"}' "${VERSION2_ID}")"
STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${ACTIVATE_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/policies/${POLICY_ID}/activate")"
[[ "${STATUS}" == "200" ]] || { echo "Policy activation returned ${STATUS}, want 200." >&2; exit 1; }
assert_response_safe "${SOURCE_ID}" "${DESTINATION_ID}" "${APPEND_REASON}"

IFS='|' read -r ACTIVE_STATUS ACTIVE_REVISION ACTIVE_MATCH <<<"$(sql_scalar -v policy="${POLICY_ID}" -v version="${VERSION2_ID}" <<'SQL'
SELECT status || '|' || revision::text || '|' ||
       CASE WHEN active_version_id = :'version'::uuid THEN '1' ELSE '0' END
FROM access_policies
WHERE id = :'policy'::uuid;
SQL
)"
ACTIVATE_AUDITS="$(sql_scalar -v policy="${POLICY_ID}" -c "SELECT count(*) FROM audit_events WHERE target_id = :'policy' AND action = 'access_policy.activate_version' AND actor_id = 'operator-policy-ci'")"
[[ "${ACTIVE_STATUS}" == "active" && "${ACTIVE_REVISION}" == "3" && "${ACTIVE_MATCH}" == "1" && "${ACTIVATE_AUDITS}" == "1" ]] || {
  echo "Activated policy state/audit is inconsistent." >&2
  exit 1
}

SECOND_BODY="$(printf '{\"organization_id\":\"%s\",\"name\":\"foreign policy\",\"initial_version\":{\"source_spiffe_id\":\"%s\",\"destination_spiffe_id\":\"%s\",\"action\":\"connect\",\"effect\":\"allow\",\"change_reason\":\"foreign-version-fixture\"}}' "${ORG_ID}" "${SOURCE_ID}" "${DESTINATION_ID}")"
STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${SECOND_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/policies")"
[[ "${STATUS}" == "201" ]] || { echo "Second policy create returned ${STATUS}, want 201." >&2; exit 1; }
SECOND_POLICY_ID="$(sql_scalar -v org="${ORG_ID}" -c "SELECT id::text FROM access_policies WHERE organization_id = :'org'::uuid AND name = 'foreign policy'")"
FOREIGN_VERSION_ID="$(sql_scalar -v policy="${SECOND_POLICY_ID}" -c "SELECT id::text FROM access_policy_versions WHERE policy_id = :'policy'::uuid AND version = 1")"
FOREIGN_ACTIVATE_BODY="$(printf '{\"expected_revision\":3,\"version_id\":\"%s\"}' "${FOREIGN_VERSION_ID}")"
STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${FOREIGN_ACTIVATE_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/policies/${POLICY_ID}/activate")"
[[ "${STATUS}" == "404" ]] || { echo "Foreign policy version activation returned ${STATUS}, want 404." >&2; exit 1; }
POST_FOREIGN_STATE="$(sql_scalar -v policy="${POLICY_ID}" -v version="${VERSION2_ID}" <<'SQL'
SELECT revision::text || '|' || CASE WHEN active_version_id = :'version'::uuid THEN '1' ELSE '0' END
FROM access_policies WHERE id = :'policy'::uuid;
SQL
)"
[[ "${POST_FOREIGN_STATE}" == "3|1" ]] || { echo "Foreign version attempt changed active policy state." >&2; exit 1; }

LEAK_COUNT="$(sql_scalar -v source="${SOURCE_ID}" -v destination="${DESTINATION_ID}" -v reason1="${CREATE_REASON}" -v reason2="${APPEND_REASON}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE action LIKE 'access_policy.%'
  AND (
    metadata::text LIKE '%' || :'source' || '%'
    OR metadata::text LIKE '%' || :'destination' || '%'
    OR metadata::text LIKE '%' || :'reason1' || '%'
    OR metadata::text LIKE '%' || :'reason2' || '%'
  );
SQL
)"
[[ "${LEAK_COUNT}" == "0" ]] || { echo "Policy audit metadata leaked rule material." >&2; exit 1; }

if sql_scalar -v version="${VERSION1_ID}" -c "UPDATE access_policy_versions SET effect = 'deny' WHERE id = :'version'::uuid" >/dev/null 2>&1; then
  echo "Immutable policy version accepted UPDATE." >&2
  exit 1
fi

echo "PASS policy mutation lifecycle: create=201 append=201 stale=409 activate=200 foreign=404 atomic-audit=verified"
