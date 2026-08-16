#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-}"
PORT=18085
TOKEN="$(openssl rand -hex 32)"
WRONG_TOKEN="$(openssl rand -hex 32)"
SERVER_PID=""

[[ -n "${DATABASE_URL}" ]] || { echo "DATABASE_URL is required for integration setup" >&2; exit 1; }

cleanup() {
  if [[ -n "${SERVER_PID}" ]]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  unset TOKEN WRONG_TOKEN
  rm -f /tmp/wtp-cli-control-plane.log /tmp/wtp-cli-health.json /tmp/wtp-cli-ready.json /tmp/wtp-cli-status.json /tmp/wtp-cli-workloads.json /tmp/wtp-cli-error.txt
}
trap cleanup EXIT

sql_scalar() {
  psql "${DATABASE_URL}" -X -qAt -v ON_ERROR_STOP=1 "$@"
}

run_cli() {
  env -u DATABASE_URL \
    WTP_API_URL="http://127.0.0.1:${PORT}" \
    WTP_OPERATOR_TOKEN="${TOKEN}" \
    "${ROOT}/.lab/wtpctl" "$@"
}

cd "${ROOT}"
mkdir -p .lab
./scripts/db-migrate.sh up
go build -trimpath -o .lab/control-plane ./apps/control-plane
go build -trimpath -o .lab/wtpctl ./apps/wtpctl

ORG_ID="$(sql_scalar <<'SQL'
INSERT INTO organizations (slug, display_name)
VALUES ('cli-diagnostics-ci', 'CLI Diagnostics CI')
RETURNING id::text;
SQL
)"
TRUST_DOMAIN_ID="$(sql_scalar -v org="${ORG_ID}" <<'SQL'
INSERT INTO trust_domains (organization_id, name)
VALUES (:'org'::uuid, 'cli-diagnostics.test')
RETURNING id::text;
SQL
)"
WORKLOAD_ID="$(sql_scalar -v org="${ORG_ID}" -v domain="${TRUST_DOMAIN_ID}" <<'SQL'
INSERT INTO workloads (organization_id, trust_domain_id, name, environment, spiffe_id, status)
VALUES (:'org'::uuid, :'domain'::uuid, 'frontend', 'ci', 'spiffe://cli-diagnostics.test/ci/frontend', 'healthy')
RETURNING id::text;
SQL
)"

sql_scalar -v org="${ORG_ID}" -v workload="${WORKLOAD_ID}" <<'SQL' >/dev/null
INSERT INTO registration_rules (
  organization_id, workload_id, selectors, parent_spiffe_id,
  x509_svid_ttl_seconds, reconcile_status
) VALUES (
  :'org'::uuid,
  :'workload'::uuid,
  '["docker:label:service:frontend"]'::jsonb,
  'spiffe://cli-diagnostics.test/spire/agent/ci',
  300,
  'pending'
);

INSERT INTO access_policies (organization_id, name, status)
VALUES (:'org'::uuid, 'frontend-to-orders', 'active');

INSERT INTO audit_events (
  organization_id, actor_type, actor_id, action, target_type,
  target_id, correlation_id, metadata
) VALUES (
  :'org'::uuid, 'operator', 'cli-seed', 'diagnostics.seeded',
  'organization', :'org', 'cli-diag-1', '{"private_fixture":"must-not-appear"}'::jsonb
);

INSERT INTO security_events (
  organization_id, event_type, severity, source_spiffe_id,
  destination_spiffe_id, reason_code, correlation_id, metadata
) VALUES (
  :'org'::uuid, 'diagnostics-test', 'info',
  'spiffe://cli-diagnostics.test/ci/frontend',
  'spiffe://cli-diagnostics.test/ci/orders-api',
  'seeded', 'cli-diag-1', '{"private_fixture":"must-not-appear"}'::jsonb
);
SQL

AUDITS_BEFORE="$(sql_scalar -v org="${ORG_ID}" <<'SQL'
SELECT count(*) FROM audit_events WHERE organization_id = :'org'::uuid;
SQL
)"

WTP_LISTEN_ADDR="127.0.0.1:${PORT}" \
WTP_OPERATOR_ID="viewer-cli-ci" \
WTP_OPERATOR_ROLE="viewer" \
WTP_OPERATOR_TOKEN="${TOKEN}" \
DATABASE_URL="${DATABASE_URL}" \
  .lab/control-plane >/tmp/wtp-cli-control-plane.log 2>&1 &
SERVER_PID=$!

DEADLINE=$((SECONDS + 15))
while (( SECONDS < DEADLINE )); do
  if curl --silent --show-error --fail "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    echo "Control plane exited during CLI diagnostics startup." >&2
    sed -E 's#(postgres(ql)?://)[^@]+@#\1[redacted]@#g' /tmp/wtp-cli-control-plane.log >&2 || true
    exit 1
  fi
  sleep 0.5
done
curl --silent --show-error --fail "http://127.0.0.1:${PORT}/healthz" >/dev/null

env -u DATABASE_URL -u WTP_OPERATOR_TOKEN \
  WTP_API_URL="http://127.0.0.1:${PORT}" \
  .lab/wtpctl health >/tmp/wtp-cli-health.json

env -u DATABASE_URL -u WTP_OPERATOR_TOKEN \
  WTP_API_URL="http://127.0.0.1:${PORT}" \
  .lab/wtpctl ready >/tmp/wtp-cli-ready.json

run_cli status --organization-id "${ORG_ID}" >/tmp/wtp-cli-status.json
run_cli workloads --organization-id "${ORG_ID}" >/tmp/wtp-cli-workloads.json

python3 - <<'PY'
import json
from pathlib import Path
for path in [
    '/tmp/wtp-cli-health.json',
    '/tmp/wtp-cli-ready.json',
    '/tmp/wtp-cli-status.json',
    '/tmp/wtp-cli-workloads.json',
]:
    json.loads(Path(path).read_text())
PY

STATUS_OUTPUT="$(cat /tmp/wtp-cli-status.json)"
for forbidden in \
  'spiffe://' \
  'selectors' \
  'parent_spiffe_id' \
  'source_spiffe_id' \
  'destination_spiffe_id' \
  'change_reason' \
  'metadata' \
  'private_fixture' \
  "${TOKEN}"
do
  if [[ "${STATUS_OUTPUT}" == *"${forbidden}"* ]]; then
    echo "CLI status output exposed forbidden material: ${forbidden}" >&2
    exit 1
  fi
done

for required in \
  'workloads_total' \
  'registrations_pending' \
  'policies_active' \
  'audit_events' \
  'security_events'
do
  if [[ "${STATUS_OUTPUT}" != *"${required}"* ]]; then
    echo "CLI status output is missing ${required}" >&2
    exit 1
  fi
done

if env -u DATABASE_URL \
  WTP_API_URL="http://127.0.0.1:${PORT}" \
  WTP_OPERATOR_TOKEN="${WRONG_TOKEN}" \
  .lab/wtpctl status --organization-id "${ORG_ID}" >/tmp/wtp-cli-error.txt 2>&1; then
  echo "CLI accepted wrong operator token." >&2
  exit 1
fi
if ! grep -Fq 'HTTP 401 (unauthorized)' /tmp/wtp-cli-error.txt; then
  echo "Wrong-token CLI error did not preserve safe 401 reason." >&2
  cat /tmp/wtp-cli-error.txt >&2
  exit 1
fi
if grep -Fq "${WRONG_TOKEN}" /tmp/wtp-cli-error.txt || grep -Fq "${TOKEN}" /tmp/wtp-cli-error.txt; then
  echo "CLI error output leaked a credential." >&2
  exit 1
fi

if env -u DATABASE_URL \
  WTP_API_URL="http://0.0.0.0:${PORT}" \
  WTP_OPERATOR_TOKEN="${TOKEN}" \
  .lab/wtpctl status --organization-id "${ORG_ID}" >/tmp/wtp-cli-error.txt 2>&1; then
  echo "CLI accepted non-loopback management URL." >&2
  exit 1
fi
if ! grep -Fq 'must target localhost or a loopback IP' /tmp/wtp-cli-error.txt; then
  echo "CLI non-loopback rejection was not explicit." >&2
  cat /tmp/wtp-cli-error.txt >&2
  exit 1
fi

AUDITS_AFTER="$(sql_scalar -v org="${ORG_ID}" <<'SQL'
SELECT count(*) FROM audit_events WHERE organization_id = :'org'::uuid;
SQL
)"
[[ "${AUDITS_AFTER}" == "${AUDITS_BEFORE}" ]] || {
  echo "Read-only diagnostics commands changed audit state." >&2
  exit 1
}

echo "PASS CLI diagnostics: health/ready/status/workloads, no DB dependency, loopback-only, safe output, wrong-token refusal"
