#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="${APP_NAME:-Pause}"
WINDOWS_PLATFORM="${WINDOWS_PLATFORM:-windows/amd64}"
WINDOWS_ARCH_LABEL="${WINDOWS_ARCH_LABEL:-windows-x64}"
WINDOWS_OUTPUT_DIR="${WINDOWS_OUTPUT_DIR:-${ROOT_DIR}/build/bin/${WINDOWS_ARCH_LABEL}}"
WINDOWS_NSIS_TEMPLATE="${WINDOWS_NSIS_TEMPLATE:-${ROOT_DIR}/scripts/windows-installer/project.nsi}"
WINDOWS_WEBVIEW2="${WINDOWS_WEBVIEW2:-download}"
WAILS_TAGS="${WAILS_TAGS:-wails}"
USE_CLEAN="${USE_CLEAN:-0}"

cd "${ROOT_DIR}"

if ! command -v nsis >/dev/null 2>&1 && ! command -v makensis >/dev/null 2>&1; then
  echo "ERROR: NSIS not found. Install it first (macOS: brew install nsis)." >&2
  exit 1
fi

mkdir -p "${ROOT_DIR}/build/bin"
mkdir -p "${WINDOWS_OUTPUT_DIR}"

if [[ -f "${WINDOWS_NSIS_TEMPLATE}" ]]; then
  mkdir -p "${ROOT_DIR}/build/windows/installer"
  cp "${WINDOWS_NSIS_TEMPLATE}" "${ROOT_DIR}/build/windows/installer/project.nsi"
  echo "using NSIS template: ${WINDOWS_NSIS_TEMPLATE}"
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
    \( -name "*.exe" -o -name "*.msi" -o -name "*.zip" -o -name "*.blockmap" \) \
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

echo "Done. Windows artifacts:"
echo "  ${WINDOWS_OUTPUT_DIR}"
