#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
source "${ROOT_DIR}/scripts/app_identity.sh"
LEGACY_APP_BUNDLE_ID="com.wails.Pause"
APP_PATH="/Applications/Pause.app"
DATA_DIR_NEW="${HOME}/Library/Application Support/Pause"
CACHE_NEW="${HOME}/Library/Caches/${APP_BUNDLE_ID}"
WEBKIT_NEW="${HOME}/Library/WebKit/${APP_BUNDLE_ID}"
LOGS_NEW="${HOME}/Library/Logs/Pause"

echo "Stopping Pause..."
pkill -9 -x Pause 2>/dev/null || true
pkill -9 -x PauseLoginHelper 2>/dev/null || true

USER_ID="$(id -u)"
echo "Unregistering startup items..."
# New startup path (ServiceManagement-managed labels)
launchctl bootout "gui/${USER_ID}/${APP_BUNDLE_ID}" 2>/dev/null || true
launchctl disable "gui/${USER_ID}/${APP_BUNDLE_ID}" 2>/dev/null || true
launchctl bootout "gui/${USER_ID}/${HELPER_BUNDLE_ID}" 2>/dev/null || true
launchctl disable "gui/${USER_ID}/${HELPER_BUNDLE_ID}" 2>/dev/null || true
launchctl bootout "gui/${USER_ID}/${LEGACY_APP_BUNDLE_ID}" 2>/dev/null || true
launchctl disable "gui/${USER_ID}/${LEGACY_APP_BUNDLE_ID}" 2>/dev/null || true

echo "Clearing preferences domains..."
defaults delete "${APP_BUNDLE_ID}" >/dev/null 2>&1 || true
defaults delete "com.wails.Pause" >/dev/null 2>&1 || true

TARGETS=(
  "${APP_PATH}"
  "${DATA_DIR_NEW}"
  "${CACHE_NEW}"
  "${WEBKIT_NEW}"
  "${LOGS_NEW}"
)

echo "Removing files..."
for target in "${TARGETS[@]}"; do
  if [[ -e "${target}" || -L "${target}" ]]; then
    rm -rf "${target}"
    echo "removed: ${target}"
  else
    echo "not_found: ${target}"
  fi
done

echo "Pause full cleanup complete."
