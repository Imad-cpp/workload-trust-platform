#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LAB_DIR="${ROOT}/.lab"
RUNTIME_DIR="/tmp/workload-trust-lab"
SERVER="${LAB_DIR}/bin/spire-server"
AGENT="${LAB_DIR}/bin/spire-agent"
SERVER_SOCKET="${RUNTIME_DIR}/server.sock"
AGENT_SOCKET="${RUNTIME_DIR}/agent.sock"
AGENT_ID="spiffe://workload-trust.test/spire/agent/lab"

"${ROOT}/scripts/lab-down.sh" >/dev/null 2>&1 || true
"${ROOT}/scripts/fetch-spire.sh"

docker info >/dev/null
[[ -S /var/run/docker.sock ]] || { echo "Docker socket not found at /var/run/docker.sock" >&2; exit 1; }

rm -rf "${LAB_DIR}/data" "${LAB_DIR}/logs" "${RUNTIME_DIR}"
mkdir -p "${LAB_DIR}/data/server" "${LAB_DIR}/data/agent" "${LAB_DIR}/logs" "${RUNTIME_DIR}"

"${SERVER}" validate -config "${ROOT}/deploy/spire/server.conf"
"${AGENT}" validate -config "${ROOT}/deploy/spire/agent.conf"

(
  cd "${ROOT}"
  nohup "${SERVER}" run -config "${ROOT}/deploy/spire/server.conf" >"${LAB_DIR}/logs/server.log" 2>&1 &
  echo $! >"${LAB_DIR}/server.pid"
)

for _ in $(seq 1 60); do
  if [[ -S "${SERVER_SOCKET}" ]] && "${SERVER}" healthcheck -socketPath "${SERVER_SOCKET}" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done
"${SERVER}" healthcheck -socketPath "${SERVER_SOCKET}" >/dev/null

TOKEN="$(${SERVER} token generate -socketPath "${SERVER_SOCKET}" -spiffeID "${AGENT_ID}" | awk '{print $2}' | tr -d '\r')"
[[ -n "${TOKEN}" ]] || { echo "Failed to generate SPIRE join token." >&2; exit 1; }

(
  cd "${ROOT}"
  nohup "${AGENT}" run -config "${ROOT}/deploy/spire/agent.conf" -joinToken "${TOKEN}" >"${LAB_DIR}/logs/agent.log" 2>&1 &
  echo $! >"${LAB_DIR}/agent.pid"
)
unset TOKEN

for _ in $(seq 1 80); do
  if [[ -S "${AGENT_SOCKET}" ]] && "${AGENT}" healthcheck -socketPath "${AGENT_SOCKET}" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done
"${AGENT}" healthcheck -socketPath "${AGENT_SOCKET}" >/dev/null

AGENT_LIST="$(${SERVER} agent list -socketPath "${SERVER_SOCKET}")"
grep -Fq "${AGENT_ID}" <<<"${AGENT_LIST}" || {
  echo "Expected lab agent was not attested." >&2
  exit 1
}

echo "SPIRE identity lab is ready."
echo "Agent: ${AGENT_ID}"
