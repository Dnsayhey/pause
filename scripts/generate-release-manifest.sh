#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_ROOT="${ARTIFACTS_ROOT:-${ROOT_DIR}/build/bin}"
OUTPUT_DIR="${OUTPUT_DIR:-${ROOT_DIR}/build/bin/release}"
RELEASE_CHANNEL="${RELEASE_CHANNEL:-local}"
RELEASE_VERSION="${RELEASE_VERSION:-}"

print_help() {
  cat <<'EOF'
Usage:
  ./scripts/generate-release-manifest.sh [options]

Options:
  --artifacts-root <path>   Root dir to scan artifacts (default: build/bin)
  --output-dir <path>       Output dir for manifest/checksums (default: build/bin/release)
  --version <version>       Release version (default: from wails.json productVersion)
  --channel <name>          Release channel label (default: local)
  -h, --help                Show this help

Scanned artifact extensions:
  .dmg, .exe, .msi, .zip, .blockmap, .msix, .appx
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --artifacts-root)
      ARTIFACTS_ROOT="$2"
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR="$2"
      shift 2
      ;;
    --version)
      RELEASE_VERSION="$2"
      shift 2
      ;;
    --channel)
      RELEASE_CHANNEL="$2"
      shift 2
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      echo "ERROR: unknown option '$1'" >&2
      print_help >&2
      exit 1
      ;;
  esac
done

if [[ -z "${RELEASE_VERSION}" ]]; then
  RELEASE_VERSION="$(
    sed -n 's/.*"productVersion":[[:space:]]*"\([^"]*\)".*/\1/p' "${ROOT_DIR}/wails.json" | head -n 1
  )"
fi
if [[ -z "${RELEASE_VERSION}" ]]; then
  RELEASE_VERSION="unknown"
fi

if [[ ! -d "${ARTIFACTS_ROOT}" ]]; then
  echo "ERROR: artifacts root not found: ${ARTIFACTS_ROOT}" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"
CHECKSUM_FILE="${OUTPUT_DIR}/SHA256SUMS"
MANIFEST_FILE="${OUTPUT_DIR}/release-manifest.txt"

sha256_file() {
  local file="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${file}" | awk '{print $1}'
    return 0
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${file}" | awk '{print $1}'
    return 0
  fi
  echo "ERROR: missing checksum command (shasum or sha256sum)" >&2
  exit 1
}

file_size_bytes() {
  local file="$1"
  if stat -f '%z' "${file}" >/dev/null 2>&1; then
    stat -f '%z' "${file}"
    return 0
  fi
  stat -c '%s' "${file}"
}

ARTIFACTS=()
while IFS= read -r file; do
  ARTIFACTS+=("${file}")
done < <(
  find "${ARTIFACTS_ROOT}" -type f \
    \( -name "*.dmg" -o -name "*.exe" -o -name "*.msi" -o -name "*.zip" -o -name "*.blockmap" -o -name "*.msix" -o -name "*.appx" \) \
    | sort
)

if [[ "${#ARTIFACTS[@]}" -eq 0 ]]; then
  echo "ERROR: no artifacts found under ${ARTIFACTS_ROOT}" >&2
  exit 1
fi

GENERATED_AT_UTC="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
GIT_COMMIT="$(git -C "${ROOT_DIR}" rev-parse --short HEAD 2>/dev/null || echo unknown)"

{
  echo "generated_at_utc=${GENERATED_AT_UTC}"
  echo "release_version=${RELEASE_VERSION}"
  echo "release_channel=${RELEASE_CHANNEL}"
  echo "git_commit=${GIT_COMMIT}"
  echo "artifacts_root=${ARTIFACTS_ROOT}"
  echo ""
  echo "[files]"
} > "${MANIFEST_FILE}"

: > "${CHECKSUM_FILE}"
for file in "${ARTIFACTS[@]}"; do
  rel_path="${file#${ROOT_DIR}/}"
  size="$(file_size_bytes "${file}")"
  sha256="$(sha256_file "${file}")"
  printf "%s  %s\n" "${sha256}" "${rel_path}" >> "${CHECKSUM_FILE}"
  printf "%s | %s bytes | sha256=%s\n" "${rel_path}" "${size}" "${sha256}" >> "${MANIFEST_FILE}"
done

echo "Done."
echo "  manifest:  ${MANIFEST_FILE}"
echo "  checksums: ${CHECKSUM_FILE}"
