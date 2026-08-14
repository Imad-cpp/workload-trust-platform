#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

redact_log() {
  # Join-token values are one-time-use and consumed at attestation, but SPIRE
  # embeds them in the generated agent ID. Keep CI failure output hygienic.
  sed -E 's#(spiffe://workload-trust\.test/spire/agent/join_token/)[A-Za-z0-9._-]+#\1[REDACTED]#g'
}

cleanup() {
  local exit_code=$?
  if [[ ${exit_code} -ne 0 ]]; then
    echo "--- SPIRE server log (tail, redacted) ---" >&2
    tail -n 120 "${ROOT}/.lab/logs/server.log" 2>/dev/null | redact_log >&2 || true
    echo "--- SPIRE agent log (tail, redacted) ---" >&2
    tail -n 180 "${ROOT}/.lab/logs/agent.log" 2>/dev/null | redact_log >&2 || true
  fi
  "${ROOT}/scripts/lab-down.sh" >/dev/null 2>&1 || true
  exit ${exit_code}
}
trap cleanup EXIT

"${ROOT}/scripts/lab-up.sh"
"${ROOT}/scripts/lab-register.sh"
"${ROOT}/scripts/lab-verify.sh"
