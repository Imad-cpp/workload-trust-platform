#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LAB_DIR="${ROOT}/.lab"
RUNTIME_DIR="/tmp/workload-trust-lab"
COMPOSE="${ROOT}/examples/identity-lab/compose.yaml"

stop_pidfile() {
  local file="$1"
  if [[ -f "${file}" ]]; then
    local pid
    pid="$(cat "${file}" 2>/dev/null || true)"
    if [[ "${pid}" =~ ^[0-9]+$ ]] && kill -0 "${pid}" 2>/dev/null; then
      kill "${pid}" 2>/dev/null || true
      for _ in $(seq 1 20); do
        kill -0 "${pid}" 2>/dev/null || break
        sleep 0.1
      done
      kill -9 "${pid}" 2>/dev/null || true
    fi
    rm -f "${file}"
  fi
}

if command -v docker >/dev/null 2>&1 && [[ -f "${COMPOSE}" ]]; then
  docker compose -f "${COMPOSE}" down --remove-orphans >/dev/null 2>&1 || true
fi

stop_pidfile "${LAB_DIR}/agent.pid"
stop_pidfile "${LAB_DIR}/server.pid"
rm -rf "${RUNTIME_DIR}"

echo "SPIRE identity lab stopped."
