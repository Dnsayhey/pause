#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/app_identity.sh"
LEGACY_APP_BUNDLE_ID="com.wails.Pause"
APP_PATH="/Applications/Pause.app"
DATA_DIR_NEW="${HOME}/Library/Application Support/Pause"
DATA_DIR_OLD="${HOME}/.pause"
LEGACY_LAUNCH_AGENT_PLIST="${HOME}/Library/LaunchAgents/${APP_BUNDLE_ID}.plist"
PREFERENCES_PLIST_NEW="${HOME}/Library/Preferences/${APP_BUNDLE_ID}.plist"
PREFERENCES_PLIST_OLD="${HOME}/Library/Preferences/com.wails.Pause.plist"
SAVED_STATE_NEW="${HOME}/Library/Saved Application State/${APP_BUNDLE_ID}.savedState"
SAVED_STATE_OLD="${HOME}/Library/Saved Application State/com.wails.Pause.savedState"
CACHE_NEW="${HOME}/Library/Caches/${APP_BUNDLE_ID}"
CACHE_APP_NAME="${HOME}/Library/Caches/Pause"
CACHE_OLD="${HOME}/Library/Caches/com.wails.Pause"
LOGS_NEW="${HOME}/Library/Logs/Pause"
LOGS_OLD="${HOME}/.pause/logs"

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
# Legacy launch agent cleanup (older Pause builds)
launchctl bootout "gui/${USER_ID}" "${LEGACY_LAUNCH_AGENT_PLIST}" 2>/dev/null || true
launchctl disable "gui/${USER_ID}/${APP_BUNDLE_ID}" 2>/dev/null || true

echo "Clearing preferences domains..."
defaults delete "${APP_BUNDLE_ID}" >/dev/null 2>&1 || true
defaults delete "com.wails.Pause" >/dev/null 2>&1 || true

TARGETS=(
  "${APP_PATH}"
  "${DATA_DIR_NEW}"
  "${DATA_DIR_OLD}"
  "${LEGACY_LAUNCH_AGENT_PLIST}"
  "${PREFERENCES_PLIST_NEW}"
  "${PREFERENCES_PLIST_OLD}"
  "${SAVED_STATE_NEW}"
  "${SAVED_STATE_OLD}"
  "${CACHE_NEW}"
  "${CACHE_APP_NAME}"
  "${CACHE_OLD}"
  "${LOGS_NEW}"
  "${LOGS_OLD}"
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

echo "Pause uninstall complete."
