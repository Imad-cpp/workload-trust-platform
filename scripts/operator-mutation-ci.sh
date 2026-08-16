#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-}"
VIEWER_PORT=18081
OPERATOR_PORT=18082
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
  rm -f /tmp/wtp-viewer.log /tmp/wtp-operator.log /tmp/wtp-http-body.json
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
  local logfile="/tmp/wtp-${role}.log"

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
      echo "Control plane exited during startup for role ${role}." >&2
      sed -E 's#(postgres(ql)?://)[^@]+@#\1[redacted]@#g' "${logfile}" >&2 || true
      return 1
    fi
    sleep 0.5
  done

  echo "Control plane did not become healthy for role ${role}." >&2
  return 1
}

http_status() {
  curl --silent --show-error \
    --output /tmp/wtp-http-body.json \
    --write-out '%{http_code}' \
    "$@"
}

cd "${ROOT}"
mkdir -p .lab
./scripts/db-migrate.sh up
go build -trimpath -o .lab/control-plane ./apps/control-plane

ORG_ID="$(sql_scalar -c "INSERT INTO organizations (slug, display_name) VALUES ('operator-mutation-ci', 'Operator Mutation CI') RETURNING id::text")"
TRUST_DOMAIN_ID="$(sql_scalar -v org="${ORG_ID}" <<'SQL'
INSERT INTO trust_domains (organization_id, name)
VALUES (:'org'::uuid, 'operator-mutation.test')
RETURNING id::text;
SQL
)"
WORKLOAD_ID="$(sql_scalar -v org="${ORG_ID}" -v trust_domain="${TRUST_DOMAIN_ID}" <<'SQL'
INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id)
VALUES (:'org'::uuid, :'trust_domain'::uuid, 'api', 'integration', 'spiffe://operator-mutation.test/integration/api')
RETURNING id::text;
SQL
)"

PARENT_ID='spiffe://operator-mutation.test/spire/agent/integration'
SELECTOR='docker:label:com.workload-trust.name:api'
CREATE_BODY="$(printf '{\"workload_id\":\"%s\",\"selectors\":[\"%s\"],\"parent_spiffe_id\":\"%s\",\"x509_svid_ttl_seconds\":300}' "${WORKLOAD_ID}" "${SELECTOR}" "${PARENT_ID}")"

start_server viewer "${VIEWER_PORT}" "${VIEWER_TOKEN}" viewer-ci

STATUS="$(http_status \
  --request POST \
  --header 'Content-Type: application/json' \
  --data "${CREATE_BODY}" \
  "http://127.0.0.1:${VIEWER_PORT}/v1/registration-rules")"
[[ "${STATUS}" == "401" ]] || { echo "Unauthenticated mutation returned ${STATUS}, want 401." >&2; exit 1; }

STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${VIEWER_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${CREATE_BODY}" \
  "http://127.0.0.1:${VIEWER_PORT}/v1/registration-rules")"
[[ "${STATUS}" == "403" ]] || { echo "Viewer mutation returned ${STATUS}, want 403." >&2; exit 1; }

STATUS="$(http_status \
  --header "Authorization: Bearer ${VIEWER_TOKEN}" \
  "http://127.0.0.1:${VIEWER_PORT}/v1/workloads?organization_id=${ORG_ID}")"
[[ "${STATUS}" == "200" ]] || { echo "Viewer workload read returned ${STATUS}, want 200." >&2; exit 1; }

PRE_RULES="$(sql_scalar -v workload="${WORKLOAD_ID}" <<'SQL'
SELECT count(*) FROM registration_rules WHERE workload_id = :'workload'::uuid;
SQL
)"
PRE_AUDITS="$(sql_scalar <<'SQL'
SELECT count(*) FROM audit_events WHERE action LIKE 'registration_rule.%desired_state';
SQL
)"
[[ "${PRE_RULES}" == "0" && "${PRE_AUDITS}" == "0" ]] || {
  echo "Unauthorized requests changed registration or audit state." >&2
  exit 1
}
echo "PASS operator authorization: unauthenticated=401 viewer=403 viewer-read=200"

stop_server
start_server operator "${OPERATOR_PORT}" "${OPERATOR_TOKEN}" operator-ci

STATUS="$(http_status \
  --request POST \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${CREATE_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/registration-rules")"
[[ "${STATUS}" == "201" ]] || { echo "Operator create returned ${STATUS}, want 201." >&2; exit 1; }
if grep -Fq "${PARENT_ID}" /tmp/wtp-http-body.json || grep -Fq "${SELECTOR}" /tmp/wtp-http-body.json || grep -Fq 'selectors' /tmp/wtp-http-body.json; then
  echo "Create response exposed sensitive desired-state fields." >&2
  exit 1
fi

RULE_ID="$(sql_scalar -v workload="${WORKLOAD_ID}" <<'SQL'
SELECT id::text
FROM registration_rules
WHERE workload_id = :'workload'::uuid
ORDER BY created_at DESC
LIMIT 1;
SQL
)"
[[ -n "${RULE_ID}" ]] || { echo "Created registration rule was not persisted." >&2; exit 1; }
IFS='|' read -r CREATE_STATE CREATE_REVISION CREATE_RECONCILE <<<"$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT desired_state || '|' || revision::text || '|' || reconcile_status
FROM registration_rules
WHERE id = :'rule'::uuid;
SQL
)"
[[ "${CREATE_STATE}" == "present" && "${CREATE_REVISION}" == "1" && "${CREATE_RECONCILE}" == "pending" ]] || {
  echo "Created desired state is inconsistent." >&2
  exit 1
}
CREATE_AUDITS="$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE target_id = :'rule'
  AND action = 'registration_rule.create_desired_state'
  AND actor_type = 'operator'
  AND actor_id = 'operator-ci';
SQL
)"
[[ "${CREATE_AUDITS}" == "1" ]] || { echo "Create mutation does not have exactly one attributed audit event." >&2; exit 1; }

PATCH_BODY="$(printf '{\"expected_revision\":1,\"desired_state\":\"absent\",\"selectors\":[\"%s\"],\"parent_spiffe_id\":\"%s\",\"x509_svid_ttl_seconds\":600}' "${SELECTOR}" "${PARENT_ID}")"
STATUS="$(http_status \
  --request PATCH \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${PATCH_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/registration-rules/${RULE_ID}")"
[[ "${STATUS}" == "200" ]] || { echo "Operator replace returned ${STATUS}, want 200." >&2; exit 1; }
if grep -Fq "${PARENT_ID}" /tmp/wtp-http-body.json || grep -Fq "${SELECTOR}" /tmp/wtp-http-body.json || grep -Fq 'selectors' /tmp/wtp-http-body.json; then
  echo "Replace response exposed sensitive desired-state fields." >&2
  exit 1
fi

IFS='|' read -r REPLACE_STATE REPLACE_REVISION REPLACE_RECONCILE <<<"$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT desired_state || '|' || revision::text || '|' || reconcile_status
FROM registration_rules
WHERE id = :'rule'::uuid;
SQL
)"
[[ "${REPLACE_STATE}" == "absent" && "${REPLACE_REVISION}" == "2" && "${REPLACE_RECONCILE}" == "pending" ]] || {
  echo "Replaced desired state is inconsistent." >&2
  exit 1
}
REPLACE_AUDITS="$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE target_id = :'rule'
  AND action = 'registration_rule.replace_desired_state'
  AND actor_type = 'operator'
  AND actor_id = 'operator-ci';
SQL
)"
[[ "${REPLACE_AUDITS}" == "1" ]] || { echo "Replace mutation does not have exactly one attributed audit event." >&2; exit 1; }

STATUS="$(http_status \
  --request PATCH \
  --header "Authorization: Bearer ${OPERATOR_TOKEN}" \
  --header 'Content-Type: application/json' \
  --data "${PATCH_BODY}" \
  "http://127.0.0.1:${OPERATOR_PORT}/v1/registration-rules/${RULE_ID}")"
[[ "${STATUS}" == "409" ]] || { echo "Stale revision returned ${STATUS}, want 409." >&2; exit 1; }
FINAL_REVISION="$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT revision FROM registration_rules WHERE id = :'rule'::uuid;
SQL
)"
FINAL_REPLACE_AUDITS="$(sql_scalar -v rule="${RULE_ID}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE target_id = :'rule' AND action = 'registration_rule.replace_desired_state';
SQL
)"
[[ "${FINAL_REVISION}" == "2" && "${FINAL_REPLACE_AUDITS}" == "1" ]] || {
  echo "Stale revision changed persistent or audit state." >&2
  exit 1
}

LEAK_COUNT="$(sql_scalar -v parent="${PARENT_ID}" -v selector="${SELECTOR}" <<'SQL'
SELECT count(*)
FROM audit_events
WHERE action LIKE 'registration_rule.%desired_state'
  AND (
    metadata::text LIKE '%' || :'parent' || '%'
    OR metadata::text LIKE '%' || :'selector' || '%'
    OR metadata::text LIKE '%join_token%'
  );
SQL
)"
[[ "${LEAK_COUNT}" == "0" ]] || { echo "Operator mutation audit metadata leaked desired-state material." >&2; exit 1; }

echo "PASS operator mutation lifecycle: create=201 replace=200 stale=409 atomic-audit=verified"
