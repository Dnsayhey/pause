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
  ./scripts/bump-version.sh <new_version>

Updates version in:
  - VERSION
  - wails.json info.productVersion
  - frontend/package.json version
  - frontend/package-lock.json version
  - frontend/package-lock.json packages[""].version (if present)

Example:
  ./scripts/bump-version.sh 0.1.1
EOF
}

if [[ $# -ne 1 ]]; then
  print_help >&2
  exit 1
fi
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
  print_help
  exit 0
fi

NEW_VERSION="$1"
if [[ ! "${NEW_VERSION}" =~ ^[0-9]+(\.[0-9]+){2}([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "ERROR: invalid version format: ${NEW_VERSION}" >&2
  echo "Expected examples: 0.1.1, 1.0.0, 1.2.3-beta.1" >&2
  exit 1
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

update_json_file() {
  local file="$1"
  local jq_expr="$2"
  local tmp_file
  tmp_file="$(mktemp)"
  jq --arg v "${NEW_VERSION}" "${jq_expr}" "${file}" > "${tmp_file}"
  mv "${tmp_file}" "${file}"
}

require_cmd jq
require_file "${WAILS_JSON}"
require_file "${FRONTEND_PACKAGE_JSON}"
require_file "${FRONTEND_PACKAGE_LOCK_JSON}"

echo "${NEW_VERSION}" > "${VERSION_FILE}"

update_json_file "${WAILS_JSON}" '.info.productVersion = $v'
update_json_file "${FRONTEND_PACKAGE_JSON}" '.version = $v'
update_json_file "${FRONTEND_PACKAGE_LOCK_JSON}" '.version = $v | if (.packages? | type) == "object" and (.packages | has("")) then .packages[""].version = $v else . end'

echo "Updated version to ${NEW_VERSION}:"
echo "  ${VERSION_FILE}"
echo "  ${WAILS_JSON} (.info.productVersion)"
echo "  ${FRONTEND_PACKAGE_JSON} (.version)"
echo "  ${FRONTEND_PACKAGE_LOCK_JSON} (.version + .packages[\"\"].version)"

"${ROOT_DIR}/scripts/check-version-sync.sh"
