#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="${ROOT}/examples/identity-lab/compose.yaml"
TRUST_PREFIX="spiffe://workload-trust.test/lab/"

probe() {
  local service="$1"
  docker compose -f "${COMPOSE}" run --rm --no-deps "${service}" 2>&1
}

assert_identity() {
  local service="$1"
  local expected="${TRUST_PREFIX}${service}"
  local output

  output="$(probe "${service}")" || {
    echo "Identity probe failed for ${service}:" >&2
    echo "${output}" >&2
    exit 1
  }

  grep -Fq "${expected}" <<<"${output}" || {
    echo "Expected SPIFFE ID not found for ${service}: ${expected}" >&2
    echo "${output}" >&2
    exit 1
  }

  echo "PASS identity: ${service} -> ${expected}"
}

assert_no_lab_identity() {
  local service="$1"
  local output status

  set +e
  output="$(probe "${service}")"
  status=$?
  set -e

  if grep -Fq "${TRUST_PREFIX}" <<<"${output}"; then
    echo "Unexpected trusted lab identity issued to ${service}." >&2
    echo "${output}" >&2
    exit 1
  fi

  echo "PASS negative identity: ${service} received no registered lab identity (exit=${status})"
}

assert_identity frontend
assert_identity orders-api
assert_identity payment-api
assert_identity admin-api
assert_no_lab_identity untrusted
assert_no_lab_identity frontend-wrong-environment

echo "SPIRE Docker workload-attestation verification passed."
