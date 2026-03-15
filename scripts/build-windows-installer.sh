#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="${APP_NAME:-Pause}"
WINDOWS_PLATFORM="${WINDOWS_PLATFORM:-windows/amd64}"
WINDOWS_ARCH_LABEL="${WINDOWS_ARCH_LABEL:-}"
WINDOWS_OUTPUT_DIR="${WINDOWS_OUTPUT_DIR:-}"
WINDOWS_NSIS_TEMPLATE="${WINDOWS_NSIS_TEMPLATE:-${ROOT_DIR}/scripts/windows-installer/project.nsi}"
WINDOWS_WEBVIEW2="${WINDOWS_WEBVIEW2:-download}"
WAILS_TAGS="${WAILS_TAGS:-wails}"
USE_CLEAN="${USE_CLEAN:-0}"
GENERATE_CHECKSUMS="${GENERATE_CHECKSUMS:-1}"

print_help() {
  cat <<'EOF'
Usage:
  ./scripts/build-windows-installer.sh [options]

Options:
  --platform <windows/amd64|windows/arm64|...>  Build target platform
  --arch-label <label>                          Artifact folder label (default derived from platform)
  --output-dir <path>                           Artifact output dir
  --webview2 <download|browser|embed|error>     Wails WebView2 mode
  --tags <go_build_tags>                        Build tags (default: wails)
  --nsis-template <path>                        NSIS template path
  --clean                                       Enable wails -clean
  --no-clean                                    Disable wails -clean (default)
  --checksums                                   Generate SHA256SUMS.txt (default)
  --no-checksums                                Skip checksum generation
  -h, --help                                    Show this help

Environment variables:
  WINDOWS_PLATFORM, WINDOWS_ARCH_LABEL, WINDOWS_OUTPUT_DIR, WINDOWS_WEBVIEW2,
  WINDOWS_NSIS_TEMPLATE, WAILS_TAGS, USE_CLEAN, GENERATE_CHECKSUMS
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform)
      WINDOWS_PLATFORM="$2"
      shift 2
      ;;
    --arch-label)
      WINDOWS_ARCH_LABEL="$2"
      shift 2
      ;;
    --output-dir)
      WINDOWS_OUTPUT_DIR="$2"
      shift 2
      ;;
    --webview2)
      WINDOWS_WEBVIEW2="$2"
      shift 2
      ;;
    --tags)
      WAILS_TAGS="$2"
      shift 2
      ;;
    --nsis-template)
      WINDOWS_NSIS_TEMPLATE="$2"
      shift 2
      ;;
    --clean)
      USE_CLEAN="1"
      shift
      ;;
    --no-clean)
      USE_CLEAN="0"
      shift
      ;;
    --checksums)
      GENERATE_CHECKSUMS="1"
      shift
      ;;
    --no-checksums)
      GENERATE_CHECKSUMS="0"
      shift
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

default_arch_label_from_platform() {
  case "${WINDOWS_PLATFORM}" in
    windows/amd64) echo "windows-x64" ;;
    windows/arm64) echo "windows-arm64" ;;
    windows/386) echo "windows-x86" ;;
    *) echo "windows-build" ;;
  esac
}

if [[ -z "${WINDOWS_ARCH_LABEL}" ]]; then
  WINDOWS_ARCH_LABEL="$(default_arch_label_from_platform)"
fi
if [[ -z "${WINDOWS_OUTPUT_DIR}" ]]; then
  WINDOWS_OUTPUT_DIR="${ROOT_DIR}/build/bin/${WINDOWS_ARCH_LABEL}"
fi

cd "${ROOT_DIR}"

if ! command -v nsis >/dev/null 2>&1 && ! command -v makensis >/dev/null 2>&1; then
  echo "ERROR: NSIS not found. Install it first (macOS: brew install nsis)." >&2
  exit 1
fi

echo "build config:"
echo "  app_name=${APP_NAME}"
echo "  windows_platform=${WINDOWS_PLATFORM}"
echo "  windows_arch_label=${WINDOWS_ARCH_LABEL}"
echo "  windows_output_dir=${WINDOWS_OUTPUT_DIR}"
echo "  windows_webview2=${WINDOWS_WEBVIEW2}"
echo "  wails_tags=${WAILS_TAGS}"
echo "  use_clean=${USE_CLEAN}"
echo "  generate_checksums=${GENERATE_CHECKSUMS}"

mkdir -p "${ROOT_DIR}/build/bin"
mkdir -p "${WINDOWS_OUTPUT_DIR}"

if [[ -f "${WINDOWS_NSIS_TEMPLATE}" ]]; then
  mkdir -p "${ROOT_DIR}/build/windows/installer"
  cp "${WINDOWS_NSIS_TEMPLATE}" "${ROOT_DIR}/build/windows/installer/project.nsi"
  echo "using NSIS template: ${WINDOWS_NSIS_TEMPLATE}"
else
  echo "WARNING: NSIS template not found, wails default template will be used: ${WINDOWS_NSIS_TEMPLATE}" >&2
fi

STAMP_FILE="$(mktemp /tmp/pause-win-build-stamp-XXXXXX)"
cleanup_stamp() {
  rm -f "${STAMP_FILE}"
}
trap cleanup_stamp EXIT
touch "${STAMP_FILE}"

if command -v wails >/dev/null 2>&1; then
  WAILS_CMD=(wails)
else
  echo "wails not found in PATH, using 'go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2'"
  WAILS_CMD=(go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2)
fi

WAILS_ARGS=(
  build
  -platform "${WINDOWS_PLATFORM}"
  -tags "${WAILS_TAGS}"
  -nsis
  -webview2 "${WINDOWS_WEBVIEW2}"
)
if [[ "${USE_CLEAN}" == "1" ]]; then
  WAILS_ARGS+=(-clean)
fi

echo "[1/2] Building ${APP_NAME} Windows installer (${WINDOWS_PLATFORM})"
"${WAILS_CMD[@]}" "${WAILS_ARGS[@]}"

echo "[2/2] Collecting Windows artifacts into ${WINDOWS_OUTPUT_DIR}"

GENERATED_FILES=()
while IFS= read -r -d '' file; do
  GENERATED_FILES+=("${file}")
done < <(
  find "${ROOT_DIR}/build/bin" \
    -maxdepth 1 \
    -type f \
    \( -name "*.exe" -o -name "*.msi" -o -name "*.zip" -o -name "*.blockmap" -o -name "*.msix" -o -name "*.appx" \) \
    -newer "${STAMP_FILE}" \
    -print0
)

if [[ "${#GENERATED_FILES[@]}" -eq 0 ]]; then
  echo "WARNING: no new installer artifacts detected in build/bin." >&2
  echo "You can still inspect ${ROOT_DIR}/build/bin manually." >&2
  exit 0
fi

for file in "${GENERATED_FILES[@]}"; do
  base="$(basename "${file}")"
  target="${WINDOWS_OUTPUT_DIR}/${base}"
  mv -f "${file}" "${target}"
  echo "moved: ${target}"
done

if [[ "${GENERATE_CHECKSUMS}" == "1" ]]; then
  CHECKSUM_FILE="${WINDOWS_OUTPUT_DIR}/SHA256SUMS.txt"
  if command -v shasum >/dev/null 2>&1; then
    (
      cd "${WINDOWS_OUTPUT_DIR}"
      shasum -a 256 *.exe *.msi *.zip *.blockmap *.msix *.appx 2>/dev/null > "SHA256SUMS.txt" || true
    )
  elif command -v sha256sum >/dev/null 2>&1; then
    (
      cd "${WINDOWS_OUTPUT_DIR}"
      sha256sum *.exe *.msi *.zip *.blockmap *.msix *.appx 2>/dev/null > "SHA256SUMS.txt" || true
    )
  fi
  if [[ -f "${CHECKSUM_FILE}" ]]; then
    echo "checksums: ${CHECKSUM_FILE}"
  fi
fi

echo "Done. Windows artifacts:"
echo "  ${WINDOWS_OUTPUT_DIR}"
