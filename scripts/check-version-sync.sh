#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="${ROOT_DIR}/VERSION"
WAILS_JSON="${ROOT_DIR}/wails.json"
FRONTEND_PACKAGE_JSON="${ROOT_DIR}/frontend/package.json"
FRONTEND_PACKAGE_LOCK_JSON="${ROOT_DIR}/frontend/package-lock.json"

print_help() {
  cat <<'EOF'
Usage:
  ./scripts/check-version-sync.sh

Checks whether the following version sources are consistent:
  - VERSION
  - wails.json (info.productVersion)
  - frontend/package.json (version)
  - frontend/package-lock.json (version and packages[""].version if present)
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  print_help
  exit 0
fi

require_file() {
  local file="$1"
  if [[ ! -f "${file}" ]]; then
    echo "ERROR: missing file: ${file}" >&2
    exit 1
  fi
}

require_cmd() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "ERROR: missing required command: ${name}" >&2
    exit 1
  fi
}

require_cmd jq
require_file "${VERSION_FILE}"
require_file "${WAILS_JSON}"
require_file "${FRONTEND_PACKAGE_JSON}"
require_file "${FRONTEND_PACKAGE_LOCK_JSON}"

trimmed_file_version="$(tr -d '[:space:]' < "${VERSION_FILE}")"
wails_version="$(jq -r '.info.productVersion // empty' "${WAILS_JSON}")"
frontend_pkg_version="$(jq -r '.version // empty' "${FRONTEND_PACKAGE_JSON}")"
frontend_lock_version="$(jq -r '.version // empty' "${FRONTEND_PACKAGE_LOCK_JSON}")"
frontend_lock_root_pkg_version="$(jq -r '.packages[""].version // empty' "${FRONTEND_PACKAGE_LOCK_JSON}")"

if [[ -z "${trimmed_file_version}" ]]; then
  echo "ERROR: VERSION is empty" >&2
  exit 1
fi
if [[ -z "${wails_version}" || -z "${frontend_pkg_version}" || -z "${frontend_lock_version}" ]]; then
  echo "ERROR: one or more version fields are missing" >&2
  exit 1
fi

has_error=0

echo "Version sources:"
echo "  VERSION: ${trimmed_file_version}"
echo "  wails.json info.productVersion: ${wails_version}"
echo "  frontend/package.json version: ${frontend_pkg_version}"
echo "  frontend/package-lock.json version: ${frontend_lock_version}"
if [[ -n "${frontend_lock_root_pkg_version}" ]]; then
  echo "  frontend/package-lock.json packages[\"\"].version: ${frontend_lock_root_pkg_version}"
fi

if [[ "${trimmed_file_version}" != "${wails_version}" ]]; then
  echo "Mismatch: VERSION != wails.json info.productVersion" >&2
  has_error=1
fi
if [[ "${trimmed_file_version}" != "${frontend_pkg_version}" ]]; then
  echo "Mismatch: VERSION != frontend/package.json version" >&2
  has_error=1
fi
if [[ "${trimmed_file_version}" != "${frontend_lock_version}" ]]; then
  echo "Mismatch: VERSION != frontend/package-lock.json version" >&2
  has_error=1
fi
if [[ -n "${frontend_lock_root_pkg_version}" && "${trimmed_file_version}" != "${frontend_lock_root_pkg_version}" ]]; then
  echo "Mismatch: VERSION != frontend/package-lock.json packages[\"\"].version" >&2
  has_error=1
fi

if [[ "${has_error}" -eq 1 ]]; then
  exit 1
fi

echo "OK: all version sources are in sync."
