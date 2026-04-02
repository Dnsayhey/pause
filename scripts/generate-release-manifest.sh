#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ARTIFACTS_ROOT="${ARTIFACTS_ROOT:-${ROOT_DIR}/build/bin}"
OUTPUT_DIR="${OUTPUT_DIR:-${ROOT_DIR}/build/bin/release}"
RELEASE_CHANNEL="${RELEASE_CHANNEL:-local}"
RELEASE_VERSION="${RELEASE_VERSION:-}"
GITHUB_REPOSITORY_SLUG="${GITHUB_REPOSITORY_SLUG:-${GITHUB_REPOSITORY:-}}"
RELEASE_TAG="${RELEASE_TAG:-}"

print_help() {
  cat <<'EOF'
Usage:
  ./scripts/generate-release-manifest.sh [options]

Options:
  --artifacts-root <path>   Root dir to scan artifacts (default: build/bin)
  --output-dir <path>       Output dir for manifest/checksums (default: build/bin/release)
  --version <version>       Release version (default: from wails.json productVersion)
  --channel <name>          Release channel label (recommend: stable; default: local)
  --repo <owner/name>       GitHub repo slug used to build download URLs
  --release-tag <tag>       Release tag used in download URLs (default: v<version>)
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
    --repo)
      GITHUB_REPOSITORY_SLUG="$2"
      shift 2
      ;;
    --release-tag)
      RELEASE_TAG="$2"
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
RELEASE_VERSION="${RELEASE_VERSION#v}"

if [[ -z "${RELEASE_TAG}" ]]; then
  RELEASE_TAG="v${RELEASE_VERSION}"
fi

if [[ ! -d "${ARTIFACTS_ROOT}" ]]; then
  echo "ERROR: artifacts root not found: ${ARTIFACTS_ROOT}" >&2
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"
CHECKSUM_FILE="${OUTPUT_DIR}/SHA256SUMS"
MANIFEST_FILE="${OUTPUT_DIR}/release-manifest.txt"
UPDATES_FILE="${OUTPUT_DIR}/updates.json"

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

resolve_repository_slug() {
  if [[ -n "${GITHUB_REPOSITORY_SLUG}" ]]; then
    echo "${GITHUB_REPOSITORY_SLUG}"
    return 0
  fi

  local remote_url
  remote_url="$(git -C "${ROOT_DIR}" remote get-url origin 2>/dev/null || true)"
  if [[ -z "${remote_url}" ]]; then
    return 0
  fi

  remote_url="${remote_url%.git}"
  case "${remote_url}" in
    git@github.com:*)
      echo "${remote_url#git@github.com:}"
      ;;
    https://github.com/*)
      echo "${remote_url#https://github.com/}"
      ;;
    ssh://git@github.com/*)
      echo "${remote_url#ssh://git@github.com/}"
      ;;
  esac
}

infer_asset_os() {
  local name="$1"
  case "${name}" in
    *-macos-*) echo "macos" ;;
    *-windows-*) echo "windows" ;;
    *-linux-*) echo "linux" ;;
    *) echo "unknown" ;;
  esac
}

infer_asset_arch() {
  local name="$1"
  case "${name}" in
    *-arm64*) echo "arm64" ;;
    *-amd64*|*-x64*) echo "x64" ;;
    *-x86*) echo "x86" ;;
    *) echo "unknown" ;;
  esac
}

infer_asset_kind() {
  local name="$1"
  case "${name}" in
    *-setup.exe) echo "installer" ;;
    *.dmg) echo "disk-image" ;;
    *.msi) echo "installer" ;;
    *.msix|*.appx) echo "package" ;;
    *.zip) echo "archive" ;;
    *.blockmap) echo "blockmap" ;;
    *) echo "file" ;;
  esac
}

append_asset_json() {
  local file="$1"
  local rel_path="$2"
  local size="$3"
  local sha256="$4"
  local basename
  local os_name
  local arch
  local kind
  local download_url="null"

  basename="$(basename "${file}")"
  os_name="$(infer_asset_os "${basename}")"
  arch="$(infer_asset_arch "${basename}")"
  kind="$(infer_asset_kind "${basename}")"

  if [[ -n "${REPOSITORY_SLUG}" ]]; then
    download_url="https://github.com/${REPOSITORY_SLUG}/releases/download/${RELEASE_TAG}/${basename}"
  fi

  jq -n \
    --arg file "${basename}" \
    --arg rel_path "${rel_path}" \
    --arg os "${os_name}" \
    --arg arch "${arch}" \
    --arg kind "${kind}" \
    --arg sha256 "${sha256}" \
    --argjson size "${size}" \
    --arg url "${download_url}" \
    '{
      name: $file,
      path: $rel_path,
      os: $os,
      arch: $arch,
      kind: $kind,
      sha256: $sha256,
      size: $size,
      url: (if $url == "null" then null else $url end)
    }'
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
REPOSITORY_SLUG="$(resolve_repository_slug)"
RELEASE_URL=""
if [[ -n "${REPOSITORY_SLUG}" ]]; then
  RELEASE_URL="https://github.com/${REPOSITORY_SLUG}/releases/tag/${RELEASE_TAG}"
fi

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
ASSET_JSON_LINES=()
for file in "${ARTIFACTS[@]}"; do
  rel_path="${file#${ROOT_DIR}/}"
  size="$(file_size_bytes "${file}")"
  sha256="$(sha256_file "${file}")"
  printf "%s  %s\n" "${sha256}" "${rel_path}" >> "${CHECKSUM_FILE}"
  printf "%s | %s bytes | sha256=%s\n" "${rel_path}" "${size}" "${sha256}" >> "${MANIFEST_FILE}"
  ASSET_JSON_LINES+=("$(append_asset_json "${file}" "${rel_path}" "${size}" "${sha256}")")
done

printf '%s\n' "${ASSET_JSON_LINES[@]}" | jq -s \
  --arg generated_at_utc "${GENERATED_AT_UTC}" \
  --arg release_version "${RELEASE_VERSION}" \
  --arg release_channel "${RELEASE_CHANNEL}" \
  --arg release_tag "${RELEASE_TAG}" \
  --arg git_commit "${GIT_COMMIT}" \
  --arg release_url "${RELEASE_URL}" \
  --arg repo "${REPOSITORY_SLUG}" \
  '{
    schema_version: 1,
    generated_at_utc: $generated_at_utc,
    release: {
      version: $release_version,
      channel: $release_channel,
      tag: $release_tag,
      commit: $git_commit,
      repository: (if $repo == "" then null else $repo end),
      url: (if $release_url == "" then null else $release_url end)
    },
    assets: .
  }' > "${UPDATES_FILE}"

echo "Done."
echo "  manifest:  ${MANIFEST_FILE}"
echo "  checksums: ${CHECKSUM_FILE}"
echo "  updates:   ${UPDATES_FILE}"
