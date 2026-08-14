#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="1.15.2"
LAB_DIR="${ROOT}/.lab"
DOWNLOAD_DIR="${LAB_DIR}/downloads"
BIN_DIR="${LAB_DIR}/bin"

case "$(uname -s)" in
  Linux) ;;
  *) echo "SPIRE identity lab currently supports Linux only." >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64)
    ARCH="amd64"
    SHA256="3874d07ffeb6640bafb9fe6a538de06151f155d5ed2f8e8a51f138d2f51b8105"
    ;;
  aarch64|arm64)
    ARCH="arm64"
    SHA256="92e782b285c50c62cdf37fdfa8917ea68fa57685b3bf99d03db36da4095678fa"
    ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

ARCHIVE="spire-${VERSION}-linux-${ARCH}-musl.tar.gz"
URL="https://github.com/spiffe/spire/releases/download/v${VERSION}/${ARCHIVE}"

mkdir -p "${DOWNLOAD_DIR}" "${BIN_DIR}"

if [[ ! -x "${BIN_DIR}/spire-server" || ! -x "${BIN_DIR}/spire-agent" ]]; then
  echo "Fetching SPIRE v${VERSION} (${ARCH})..."
  curl --fail --location --silent --show-error "${URL}" -o "${DOWNLOAD_DIR}/${ARCHIVE}"
  printf '%s  %s\n' "${SHA256}" "${DOWNLOAD_DIR}/${ARCHIVE}" | sha256sum --check --status

  EXTRACT_DIR="${DOWNLOAD_DIR}/extract"
  rm -rf "${EXTRACT_DIR}"
  mkdir -p "${EXTRACT_DIR}"
  tar -xzf "${DOWNLOAD_DIR}/${ARCHIVE}" -C "${EXTRACT_DIR}"

  SERVER_PATH="$(find "${EXTRACT_DIR}" -type f -path '*/bin/spire-server' -print -quit)"
  AGENT_PATH="$(find "${EXTRACT_DIR}" -type f -path '*/bin/spire-agent' -print -quit)"
  [[ -n "${SERVER_PATH}" && -n "${AGENT_PATH}" ]] || {
    echo "SPIRE archive did not contain expected binaries." >&2
    exit 1
  }

  install -m 0755 "${SERVER_PATH}" "${BIN_DIR}/spire-server"
  install -m 0755 "${AGENT_PATH}" "${BIN_DIR}/spire-agent"
fi

"${BIN_DIR}/spire-server" --version
"${BIN_DIR}/spire-agent" --version
