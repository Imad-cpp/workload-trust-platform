#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATABASE_URL="${DATABASE_URL:-}"
DIRECTION="${1:-up}"

[[ -n "${DATABASE_URL}" ]] || {
  echo "DATABASE_URL is required" >&2
  exit 1
}

has_table() {
  local table="$1"
  psql "${DATABASE_URL}" -Atqc "SELECT to_regclass('public.${table}') IS NOT NULL" | grep -qx t
}

has_column() {
  local table="$1"
  local column="$2"
  psql "${DATABASE_URL}" -Atqc "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='${table}' AND column_name='${column}')" | grep -qx t
}

case "${DIRECTION}" in
  up)
    if ! has_table organizations; then
      psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${ROOT}/db/migrations/000001_control_plane.up.sql"
    fi
    if ! has_column registration_rules parent_spiffe_id; then
      psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${ROOT}/db/migrations/000002_registration_reconciliation.up.sql"
    fi
    ;;
  down)
    if has_column registration_rules parent_spiffe_id; then
      psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${ROOT}/db/migrations/000002_registration_reconciliation.down.sql"
    fi
    if has_table organizations; then
      psql "${DATABASE_URL}" -v ON_ERROR_STOP=1 -f "${ROOT}/db/migrations/000001_control_plane.down.sql"
    fi
    ;;
  *)
    echo "Usage: DATABASE_URL=... $0 [up|down]" >&2
    exit 2
    ;;
esac
