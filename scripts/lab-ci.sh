#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cleanup() {
  local exit_code=$?
  if [[ ${exit_code} -ne 0 ]]; then
    echo "--- SPIRE server log (tail) ---" >&2
    tail -n 80 "${ROOT}/.lab/logs/server.log" 2>/dev/null || true
    echo "--- SPIRE agent log (tail) ---" >&2
    tail -n 80 "${ROOT}/.lab/logs/agent.log" 2>/dev/null || true
  fi
  "${ROOT}/scripts/lab-down.sh" >/dev/null 2>&1 || true
  exit ${exit_code}
}
trap cleanup EXIT

"${ROOT}/scripts/lab-up.sh"
"${ROOT}/scripts/lab-register.sh"
"${ROOT}/scripts/lab-verify.sh"
