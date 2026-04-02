#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/scripts/app_identity.sh"

APP_NAME="${APP_NAME:-Pause}"
APP_ICON_SOURCE="${APP_ICON_SOURCE:-${ROOT_DIR}/assets/branding/app-icon-1024.png}"
APP_ICON_TARGET="${ROOT_DIR}/build/appicon.png"
HELPER_NAME="${HELPER_NAME:-PauseLoginHelper}"
CODE_SIGN_IDENTITY="${PAUSE_CODESIGN_IDENTITY:--}"
STAGING_DIR="${ROOT_DIR}/build/.dmg-staging"
APP_VERSION="1.0.0"
APP_VERSION_OVERRIDE="${APP_VERSION_OVERRIDE:-}"
PACKAGE_VERSION=""
USE_CLEAN="${USE_CLEAN:-1}"
MACOS_PLATFORM="${MACOS_PLATFORM:-split}"
MACOS_ARCH_LABEL="${MACOS_ARCH_LABEL:-}"
MACOS_OUTPUT_BASE_DIR="${MACOS_OUTPUT_BASE_DIR:-${ROOT_DIR}/build/bin}"
MACOS_OUTPUT_DIR_COMPAT="${MACOS_OUTPUT_DIR:-}"

OUTPUT_SET=0
OUTPUT_DIR_SET=0
OUTPUT_ARG=""
OUTPUT_DIR_ARG=""

APP_BUNDLE=""
APP_INFO_PLIST=""
DMG_LAYOUT_DS_STORE_TEMPLATE="${DMG_LAYOUT_DS_STORE_TEMPLATE:-${ROOT_DIR}/assets/dmg/dmg-layout.dsstore}"
VITE_UPDATES_URL="${VITE_UPDATES_URL:-}"

refresh_paths() {
  APP_BUNDLE="${ROOT_DIR}/build/bin/${APP_NAME}.app"
  APP_INFO_PLIST="${APP_BUNDLE}/Contents/Info.plist"
}

print_help() {
  cat <<'USAGE'
Usage:
  ./scripts/build-dmg.sh [options]

Options:
  --platform <target>        Build target: split (default), darwin/arm64, darwin/amd64, darwin/universal
  --arch-label <label>       Override architecture label for single-platform build
  --name <app_name>          Override app name (default: Pause)
  --output <abs_or_rel_path> Override DMG output path (single-platform only)
  --output-dir <path>        Output directory (single-platform); base directory (split mode)
  --icon <path>              Override app icon source
  --bundle-id <bundle_id>    Override app bundle id
  --codesign <identity>      Override codesign identity ("-" means ad-hoc)
  --version <version>        Override CFBundleShortVersionString/CFBundleVersion
  --clean                    Force wails build -clean (default)
  --no-clean                 Skip wails build -clean
  -h, --help                 Show this help

Environment variables:
  APP_NAME, APP_ICON_SOURCE, APP_BUNDLE_ID, PAUSE_CODESIGN_IDENTITY,
  APP_VERSION_OVERRIDE, USE_CLEAN, MACOS_PLATFORM, MACOS_ARCH_LABEL,
  MACOS_OUTPUT_BASE_DIR, MACOS_OUTPUT_DIR, DMG_LAYOUT_DS_STORE_TEMPLATE,
  VITE_UPDATES_URL
USAGE
}

require_tool() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "ERROR: missing required command: ${name}" >&2
    exit 1
  fi
}

is_valid_platform() {
  local platform="$1"
  case "${platform}" in
    darwin/arm64|darwin/amd64|darwin/universal)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

platform_label() {
  local platform="$1"
  case "${platform}" in
    darwin/arm64)
      echo "macos-arm64"
      ;;
    darwin/amd64)
      echo "macos-x64"
      ;;
    darwin/universal)
      echo "macos-universal"
      ;;
    *)
      echo ""
      ;;
  esac
}

helper_arch_flags_for_platform() {
  local platform="$1"
  case "${platform}" in
    darwin/arm64)
      echo "-arch arm64"
      ;;
    darwin/amd64)
      echo "-arch x86_64"
      ;;
    darwin/universal)
      echo "-arch arm64 -arch x86_64"
      ;;
    *)
      echo ""
      ;;
  esac
}

join_by() {
  local delimiter="$1"
  shift
  local result=""
  local item
  for item in "$@"; do
    if [[ -n "${result}" ]]; then
      result+="${delimiter}"
    fi
    result+="${item}"
  done
  echo "${result}"
}

resolve_package_version() {
  local version="${APP_VERSION_OVERRIDE}"
  if [[ -z "${version}" && -f "${ROOT_DIR}/VERSION" ]]; then
    version="$(head -n 1 "${ROOT_DIR}/VERSION" | tr -d '[:space:]')"
  fi
  if [[ -z "${version}" && -f "${ROOT_DIR}/wails.json" ]]; then
    version="$(sed -n 's/.*"productVersion":[[:space:]]*"\([^"]*\)".*/\1/p' "${ROOT_DIR}/wails.json" | head -n 1)"
  fi
  if [[ -z "${version}" ]]; then
    version="${APP_VERSION}"
  fi
  version="$(echo "${version}" | tr -d '[:space:]')"
  version="${version#v}"
  if [[ -z "${version}" ]]; then
    version="0.0.0"
  fi
  echo "${version}"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --platform)
      MACOS_PLATFORM="$2"
      shift 2
      ;;
    --arch-label)
      MACOS_ARCH_LABEL="$2"
      shift 2
      ;;
    --name)
      APP_NAME="$2"
      shift 2
      ;;
    --output)
      OUTPUT_SET=1
      OUTPUT_ARG="$2"
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR_SET=1
      OUTPUT_DIR_ARG="$2"
      shift 2
      ;;
    --icon)
      APP_ICON_SOURCE="$2"
      shift 2
      ;;
    --bundle-id)
      APP_BUNDLE_ID="$2"
      shift 2
      ;;
    --codesign)
      CODE_SIGN_IDENTITY="$2"
      shift 2
      ;;
    --version)
      APP_VERSION_OVERRIDE="$2"
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

refresh_paths

TARGET_PLATFORMS=()
if [[ "${MACOS_PLATFORM}" == "split" ]]; then
  TARGET_PLATFORMS=(darwin/arm64 darwin/amd64)
elif is_valid_platform "${MACOS_PLATFORM}"; then
  TARGET_PLATFORMS=("${MACOS_PLATFORM}")
else
  echo "ERROR: invalid --platform '${MACOS_PLATFORM}'" >&2
  echo "Allowed values: split, darwin/arm64, darwin/amd64, darwin/universal" >&2
  exit 1
fi

if (( ${#TARGET_PLATFORMS[@]} > 1 )) && (( OUTPUT_SET == 1 )); then
  echo "ERROR: --output only supports single-platform builds" >&2
  exit 1
fi
if (( ${#TARGET_PLATFORMS[@]} > 1 )) && [[ -n "${MACOS_ARCH_LABEL}" ]]; then
  echo "ERROR: --arch-label only supports single-platform builds" >&2
  exit 1
fi

cd "${ROOT_DIR}"

require_tool hdiutil
require_tool codesign
require_tool clang
if [[ ! -x "/usr/libexec/PlistBuddy" ]]; then
  echo "ERROR: missing required command: /usr/libexec/PlistBuddy" >&2
  exit 1
fi

if [[ ! -f "${APP_ICON_SOURCE}" ]]; then
  echo "ERROR: app icon source not found: ${APP_ICON_SOURCE}" >&2
  exit 1
fi

if command -v wails >/dev/null 2>&1; then
  WAILS_CMD=(wails)
else
  echo "wails not found in PATH, using 'go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2'"
  WAILS_CMD=(go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2)
fi
PACKAGE_VERSION="$(resolve_package_version)"

build_one_target() {
  local platform="$1"
  local arch_label="$2"
  local output_dir="$3"
  local dmg_output="$4"
  local use_clean_target="$5"

  mkdir -p "$(dirname "${APP_ICON_TARGET}")"
  cp "${APP_ICON_SOURCE}" "${APP_ICON_TARGET}"
  mkdir -p "${output_dir}"

  echo "[1/4] Building ${APP_NAME}.app (${platform})"
  local wails_ldflags="-X pause/internal/meta.AppBundleID=${APP_BUNDLE_ID}"
  local -a wails_args=(build -platform "${platform}" -skipbindings -tags wails -ldflags "${wails_ldflags}")
  if [[ "${use_clean_target}" == "1" ]]; then
    wails_args+=(-clean)
  fi

  echo "build config:"
  echo "  app_name=${APP_NAME}"
  echo "  app_bundle_id=${APP_BUNDLE_ID}"
  echo "  helper_bundle_id=${HELPER_BUNDLE_ID}"
  echo "  codesign_identity=${CODE_SIGN_IDENTITY}"
  echo "  app_icon_source=${APP_ICON_SOURCE}"
  echo "  platform=${platform}"
  echo "  macos_arch_label=${arch_label}"
  echo "  macos_output_dir=${output_dir}"
  echo "  dmg_output=${dmg_output}"
  echo "  dmg_layout_template=${DMG_LAYOUT_DS_STORE_TEMPLATE}"
  echo "  use_clean=${use_clean_target}"

  if [[ -n "${VITE_UPDATES_URL}" ]]; then
    VITE_UPDATES_URL="${VITE_UPDATES_URL}" "${WAILS_CMD[@]}" "${wails_args[@]}"
  else
    "${WAILS_CMD[@]}" "${wails_args[@]}"
  fi

  if [[ ! -d "${APP_BUNDLE}" ]]; then
    echo "ERROR: app bundle not found: ${APP_BUNDLE}" >&2
    exit 1
  fi

  # Keep app/helper bundle identifiers aligned with runtime startup-manager logic.
  if [[ -f "${APP_INFO_PLIST}" ]]; then
    /usr/libexec/PlistBuddy -c "Set :CFBundleIdentifier ${APP_BUNDLE_ID}" "${APP_INFO_PLIST}" >/dev/null 2>&1 \
      || /usr/libexec/PlistBuddy -c "Add :CFBundleIdentifier string ${APP_BUNDLE_ID}" "${APP_INFO_PLIST}" >/dev/null
    /usr/libexec/PlistBuddy -c "Set :LSUIElement true" "${APP_INFO_PLIST}" >/dev/null 2>&1 \
      || /usr/libexec/PlistBuddy -c "Add :LSUIElement bool true" "${APP_INFO_PLIST}" >/dev/null
  fi
  if [[ -f "${APP_INFO_PLIST}" ]]; then
    if [[ -n "${APP_VERSION_OVERRIDE}" ]]; then
      /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString ${APP_VERSION_OVERRIDE}" "${APP_INFO_PLIST}" >/dev/null 2>&1 \
        || /usr/libexec/PlistBuddy -c "Add :CFBundleShortVersionString string ${APP_VERSION_OVERRIDE}" "${APP_INFO_PLIST}" >/dev/null
      /usr/libexec/PlistBuddy -c "Set :CFBundleVersion ${APP_VERSION_OVERRIDE}" "${APP_INFO_PLIST}" >/dev/null 2>&1 \
        || /usr/libexec/PlistBuddy -c "Add :CFBundleVersion string ${APP_VERSION_OVERRIDE}" "${APP_INFO_PLIST}" >/dev/null
    fi
    APP_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${APP_INFO_PLIST}" 2>/dev/null || echo '1.0.0')"
  fi

  echo "[2/4] Embedding login helper (${HELPER_BUNDLE_ID})"
  local helper_app="${APP_BUNDLE}/Contents/Library/LoginItems/${HELPER_NAME}.app"
  local helper_contents="${helper_app}/Contents"
  local helper_exec="${helper_contents}/MacOS/${HELPER_NAME}"
  local helper_info_plist="${helper_contents}/Info.plist"
  local tmp_helper_src
  tmp_helper_src="$(mktemp /tmp/pause-loginhelper-XXXXXX.m)"

  mkdir -p "${helper_contents}/MacOS"

  cat > "${tmp_helper_src}" <<'OBJC'
#import <Cocoa/Cocoa.h>

static NSString *ResolveMainAppPath(void) {
    NSString *execPath = [[[NSProcessInfo processInfo] arguments] firstObject];
    if (execPath == nil || [execPath length] == 0) {
        return nil;
    }
    NSString *macOSDir = [execPath stringByDeletingLastPathComponent];
    NSString *contentsDir = [macOSDir stringByDeletingLastPathComponent];
    NSString *helperAppDir = [contentsDir stringByDeletingLastPathComponent];
    NSString *loginItemsDir = [helperAppDir stringByDeletingLastPathComponent];
    NSString *libraryDir = [loginItemsDir stringByDeletingLastPathComponent];
    NSString *mainContentsDir = [libraryDir stringByDeletingLastPathComponent];
    NSString *mainAppDir = [mainContentsDir stringByDeletingLastPathComponent];
    return mainAppDir;
}

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        (void)argc;
        (void)argv;
        NSString *mainAppPath = ResolveMainAppPath();
        if (mainAppPath == nil) {
            return 1;
        }
        NSURL *appURL = [NSURL fileURLWithPath:mainAppPath];
        if (appURL == nil) {
            return 1;
        }
        if (@available(macOS 10.15, *)) {
            NSWorkspaceOpenConfiguration *config = [NSWorkspaceOpenConfiguration configuration];
            config.activates = NO;
            [[NSWorkspace sharedWorkspace] openApplicationAtURL:appURL configuration:config completionHandler:nil];
        } else {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
            [[NSWorkspace sharedWorkspace] launchApplication:mainAppPath];
#pragma clang diagnostic pop
        }
    }
    return 0;
}
OBJC

  local helper_arch_flags
  helper_arch_flags="$(helper_arch_flags_for_platform "${platform}")"
  if [[ -z "${helper_arch_flags}" ]]; then
    echo "ERROR: unsupported helper arch mapping for platform: ${platform}" >&2
    rm -f "${tmp_helper_src}"
    exit 1
  fi

  # shellcheck disable=SC2206
  local -a helper_arch_args=(${helper_arch_flags})
  if ! clang -fobjc-arc -framework Cocoa -mmacosx-version-min=10.13 "${helper_arch_args[@]}" "${tmp_helper_src}" -o "${helper_exec}" 2>/dev/null; then
    if [[ "${platform}" != "darwin/universal" ]]; then
      echo "ERROR: failed to compile ${platform} login helper" >&2
      rm -f "${tmp_helper_src}"
      exit 1
    fi
    clang -fobjc-arc -framework Cocoa -mmacosx-version-min=10.13 "${tmp_helper_src}" -o "${helper_exec}"
  fi
  rm -f "${tmp_helper_src}"

  chmod 755 "${helper_exec}"

  cat > "${helper_info_plist}" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>English</string>
  <key>CFBundleExecutable</key>
  <string>${HELPER_NAME}</string>
  <key>CFBundleIdentifier</key>
  <string>${HELPER_BUNDLE_ID}</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>${HELPER_NAME}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${APP_VERSION}</string>
  <key>CFBundleVersion</key>
  <string>${APP_VERSION}</string>
  <key>LSBackgroundOnly</key>
  <true/>
</dict>
</plist>
PLIST

  # Sign nested helper and main app so legacy SMLoginItemSetEnabled can resolve the helper.
  codesign --force --deep --sign "${CODE_SIGN_IDENTITY}" "${helper_app}"
  codesign --force --deep --sign "${CODE_SIGN_IDENTITY}" "${APP_BUNDLE}"

  echo "[3/4] Preparing DMG staging folder"
  rm -rf "${STAGING_DIR}"
  mkdir -p "${STAGING_DIR}"
  cp -R "${APP_BUNDLE}" "${STAGING_DIR}/"
  ln -s /Applications "${STAGING_DIR}/Applications"
  if [[ -f "${DMG_LAYOUT_DS_STORE_TEMPLATE}" ]]; then
    cp "${DMG_LAYOUT_DS_STORE_TEMPLATE}" "${STAGING_DIR}/.DS_Store"
  else
    echo "WARN: DMG layout template not found, Finder will use default layout: ${DMG_LAYOUT_DS_STORE_TEMPLATE}"
  fi

  echo "[4/4] Creating DMG"
  mkdir -p "$(dirname "${dmg_output}")"
  rm -f "${dmg_output}"
  hdiutil create \
    -volname "${APP_NAME}" \
    -srcfolder "${STAGING_DIR}" \
    -ov \
    -format UDZO \
    "${dmg_output}"

  echo "Done: ${dmg_output}"
}

resolve_output_dir() {
  local label="$1"
  local single_mode="$2"

  if [[ "${single_mode}" == "1" ]]; then
    if (( OUTPUT_DIR_SET == 1 )); then
      echo "${OUTPUT_DIR_ARG}"
      return
    fi
    if [[ -n "${MACOS_OUTPUT_DIR_COMPAT}" ]]; then
      echo "${MACOS_OUTPUT_DIR_COMPAT}"
      return
    fi
    echo "${ROOT_DIR}/build/bin/${label}"
    return
  fi

  local base_dir
  if (( OUTPUT_DIR_SET == 1 )); then
    base_dir="${OUTPUT_DIR_ARG}"
  elif [[ -n "${MACOS_OUTPUT_DIR_COMPAT}" ]]; then
    base_dir="${MACOS_OUTPUT_DIR_COMPAT}"
  else
    base_dir="${MACOS_OUTPUT_BASE_DIR}"
  fi
  echo "${base_dir%/}/${label}"
}

BUILT_DMGS=()
single_mode=0
if (( ${#TARGET_PLATFORMS[@]} == 1 )); then
  single_mode=1
fi

echo "macOS build targets: $(join_by ', ' "${TARGET_PLATFORMS[@]}")"
if [[ "${USE_CLEAN}" == "1" && "${single_mode}" == "0" ]]; then
  echo "split mode with --clean: applying -clean only on the first target to keep earlier artifacts"
fi

target_index=0
for platform in "${TARGET_PLATFORMS[@]}"; do
  label="$(platform_label "${platform}")"
  if [[ -z "${label}" ]]; then
    echo "ERROR: unsupported platform label mapping: ${platform}" >&2
    exit 1
  fi

  if [[ "${single_mode}" == "1" && -n "${MACOS_ARCH_LABEL}" ]]; then
    label="${MACOS_ARCH_LABEL}"
  fi

  output_dir="$(resolve_output_dir "${label}" "${single_mode}")"
  if (( OUTPUT_SET == 1 )); then
    dmg_output="${OUTPUT_ARG}"
  else
    dmg_output="${output_dir}/${APP_NAME}-v${PACKAGE_VERSION}-${label}.dmg"
  fi

  use_clean_target="${USE_CLEAN}"
  if [[ "${single_mode}" == "0" && "${target_index}" -gt 0 ]]; then
    use_clean_target="0"
  fi

  build_one_target "${platform}" "${label}" "${output_dir}" "${dmg_output}" "${use_clean_target}"
  BUILT_DMGS+=("${dmg_output}")
  target_index=$((target_index + 1))

done

for dmg in "${BUILT_DMGS[@]}"; do
  if [[ ! -f "${dmg}" ]]; then
    echo "ERROR: expected DMG artifact missing: ${dmg}" >&2
    exit 1
  fi
done

echo "Built DMG artifacts:"
for dmg in "${BUILT_DMGS[@]}"; do
  size_h="$(du -h "${dmg}" | awk '{print $1}')"
  echo "  ${dmg} (${size_h})"
done
