#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-}"
DIRECTION="${1:-up}"

[[ -n "${DATABASE_URL}" ]] || {
  echo "DATABASE_URL is required" >&2
  exit 1
}

case "${DIRECTION}" in
  up)
    psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${ROOT}/db/migrations/000001_control_plane.up.sql"
    ;;
  down)
    psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${ROOT}/db/migrations/000001_control_plane.down.sql"
    ;;
  *)
    echo "Usage: DATABASE_URL=... $0 [up|down]" >&2
    exit 2
    ;;
esac
