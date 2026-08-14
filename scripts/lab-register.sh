#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER="${ROOT}/.lab/bin/spire-server"
SERVER_SOCKET="/tmp/workload-trust-lab/server.sock"
PARENT_ID="spiffe://workload-trust.test/agent/lab"
TRUST_DOMAIN="spiffe://workload-trust.test"

[[ -x "${SERVER}" ]] || { echo "Run scripts/lab-up.sh first." >&2; exit 1; }

register_workload() {
  local name="$1"
  local spiffe_id="${TRUST_DOMAIN}/lab/${name}"

  if "${SERVER}" entry show -socketPath "${SERVER_SOCKET}" -spiffeID "${spiffe_id}" 2>/dev/null | grep -Fq "${spiffe_id}"; then
    echo "Registration already exists: ${spiffe_id}"
    return
  fi

  "${SERVER}" entry create \
    -socketPath "${SERVER_SOCKET}" \
    -parentID "${PARENT_ID}" \
    -spiffeID "${spiffe_id}" \
    -selector "docker:label:com.workload-trust.name:${name}" \
    -selector "docker:label:com.workload-trust.environment:lab" >/dev/null

  echo "Registered: ${spiffe_id}"
}

register_workload frontend
register_workload orders-api
register_workload payment-api
register_workload admin-api
