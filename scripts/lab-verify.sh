#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="${ROOT}/examples/identity-lab/compose.yaml"
TRUST_PREFIX="spiffe://workload-trust.test/lab/"
SYNC_TIMEOUT_SECONDS=15

probe() {
  local service="$1"
  docker compose -f "${COMPOSE}" run --rm --no-deps "${service}" 2>&1
}

assert_identity() {
  local service="$1"
  local expected="${TRUST_PREFIX}${service}"
  local output=""
  local status=1
  local deadline=$((SECONDS + SYNC_TIMEOUT_SECONDS))

  # SPIRE Agent synchronizes authorized entries from the server asynchronously.
  # Poll the real Workload API instead of relying on a fixed sleep so a freshly
  # registered workload is verified as soon as its entry reaches the agent.
  while (( SECONDS < deadline )); do
    set +e
    output="$(probe "${service}")"
    status=$?
    set -e

    if [[ ${status} -eq 0 ]] && grep -Fq "${expected}" <<<"${output}"; then
      echo "PASS identity: ${service} -> ${expected}"
      return 0
    fi

    if grep -Fq "${TRUST_PREFIX}" <<<"${output}" && ! grep -Fq "${expected}" <<<"${output}"; then
      echo "Unexpected lab identity returned for ${service}." >&2
      echo "${output}" >&2
      return 1
    fi

    sleep 1
  done

  echo "Identity probe did not converge for ${service} within ${SYNC_TIMEOUT_SECONDS}s." >&2
  echo "${output}" >&2
  return 1
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

# The first positive probe doubles as a deterministic wait for the agent's
# authorized-entry synchronization cycle. Once it succeeds, all registrations
# were created in the same batch and the negative probes are meaningful.
assert_identity frontend
assert_identity orders-api
assert_identity payment-api
assert_identity admin-api
assert_no_lab_identity untrusted
assert_no_lab_identity frontend-wrong-environment

echo "SPIRE Docker workload-attestation verification passed."
