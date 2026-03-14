#!/usr/bin/env bash

if [[ -n "${ROOT_DIR:-}" ]]; then
  PAUSE_ROOT_DIR="${ROOT_DIR}"
else
  PAUSE_ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi

PAUSE_BUNDLE_ID_FILE="${PAUSE_ROOT_DIR}/internal/meta/bundle_id.txt"
if [[ ! -f "${PAUSE_BUNDLE_ID_FILE}" ]]; then
  echo "ERROR: missing bundle id source: ${PAUSE_BUNDLE_ID_FILE}" >&2
  return 1
fi

PAUSE_DEFAULT_APP_BUNDLE_ID="$(tr -d '[:space:]' < "${PAUSE_BUNDLE_ID_FILE}")"
if [[ -z "${PAUSE_DEFAULT_APP_BUNDLE_ID}" ]]; then
  echo "ERROR: bundle id source is empty: ${PAUSE_BUNDLE_ID_FILE}" >&2
  return 1
fi

APP_BUNDLE_ID="${APP_BUNDLE_ID:-${PAUSE_APP_BUNDLE_ID:-${PAUSE_DEFAULT_APP_BUNDLE_ID}}}"
HELPER_BUNDLE_ID="${HELPER_BUNDLE_ID:-${APP_BUNDLE_ID}.loginhelper}"
