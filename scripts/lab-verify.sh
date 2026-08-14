#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE="${ROOT}/examples/identity-lab/compose.yaml"
TRUST_PREFIX="spiffe://workload-trust.test/lab/"
SYNC_TIMEOUT_SECONDS=15
ROTATION_TIMEOUT_SECONDS=30

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

run_detached_probe() {
  local service="$1"
  local cid status output

  cid="$(docker compose -f "${COMPOSE}" run -d --no-deps "${service}")"
  status="$(docker wait "${cid}")"
  output="$(docker logs "${cid}" 2>&1)"
  docker rm "${cid}" >/dev/null

  [[ "${status}" == "0" ]] || {
    echo "Detached identity probe failed for ${service} (container=${cid:0:12}, exit=${status})." >&2
    echo "${output}" >&2
    return 1
  }

  grep -Fq "${TRUST_PREFIX}${service}" <<<"${output}" || {
    echo "Detached probe returned no expected identity for ${service}." >&2
    echo "${output}" >&2
    return 1
  }

  printf '%s\n' "${cid}"
}

assert_restart_reissue() {
  local service="$1"
  local first_cid second_cid

  first_cid="$(run_detached_probe "${service}")"
  second_cid="$(run_detached_probe "${service}")"

  [[ "${first_cid}" != "${second_cid}" ]] || {
    echo "Restart test unexpectedly reused the same Docker container ID." >&2
    return 1
  }

  echo "PASS restart/re-issuance: ${service} obtained its identity from two distinct containers (${first_cid:0:12} -> ${second_cid:0:12})"
}

assert_svid_rotation() {
  local service="$1"
  local expected="${TRUST_PREFIX}${service}"
  local cid logs
  local deadline=$((SECONDS + ROTATION_TIMEOUT_SECONDS))
  local identity_updates=0
  local validity_updates=0

  cid="$(docker compose -f "${COMPOSE}" run -d --no-deps "${service}" api watch -socketPath /run/spire/agent.sock)"

  cleanup_rotation_probe() {
    docker rm -f "${cid}" >/dev/null 2>&1 || true
  }

  while (( SECONDS < deadline )); do
    logs="$(docker logs "${cid}" 2>&1 || true)"
    identity_updates="$(grep -Fc "SPIFFE ID:		${expected}" <<<"${logs}" || true)"
    validity_updates="$(grep -F "SVID Valid Until:" <<<"${logs}" | sed -E 's/^[^:]+:[[:space:]]*//' | sort -u | grep -c . || true)"

    if (( identity_updates >= 2 && validity_updates >= 2 )); then
      cleanup_rotation_probe
      echo "PASS SVID rotation: ${service} received ${identity_updates} workload-context updates with ${validity_updates} distinct certificate expirations"
      return 0
    fi

    if ! docker inspect "${cid}" >/dev/null 2>&1; then
      echo "SVID watch container disappeared before rotation evidence was observed." >&2
      echo "${logs}" >&2
      return 1
    fi

    sleep 1
  done

  logs="$(docker logs "${cid}" 2>&1 || true)"
  cleanup_rotation_probe
  echo "SVID rotation was not observed for ${service} within ${ROTATION_TIMEOUT_SECONDS}s." >&2
  echo "${logs}" >&2
  return 1
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

# `docker compose run` creates a fresh one-off container each time. Prove a
# restarted/recreated workload can re-attest to the same logical identity
# without carrying a long-lived service credential.
assert_restart_reissue frontend

# Keep one workload process attached to the streaming Workload API and require
# multiple X.509 context updates with different certificate expiry timestamps.
assert_svid_rotation frontend

echo "SPIRE Docker workload-attestation and SVID lifecycle verification passed."
