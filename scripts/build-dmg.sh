#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APP_NAME="${APP_NAME:-Pause}"
source "${ROOT_DIR}/scripts/app_identity.sh"
HELPER_NAME="${HELPER_NAME:-PauseLoginHelper}"
CODE_SIGN_IDENTITY="${PAUSE_CODESIGN_IDENTITY:--}"
APP_BUNDLE="${ROOT_DIR}/build/bin/${APP_NAME}.app"
DMG_OUTPUT="${ROOT_DIR}/build/bin/${APP_NAME}.dmg"
STAGING_DIR="${ROOT_DIR}/build/.dmg-staging"
APP_INFO_PLIST="${APP_BUNDLE}/Contents/Info.plist"
APP_VERSION="1.0.0"

cd "${ROOT_DIR}"

echo "[1/4] Building ${APP_NAME}.app"
if command -v wails >/dev/null 2>&1; then
  WAILS_CMD=(wails)
else
  echo "wails not found in PATH, using 'go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2'"
  WAILS_CMD=(go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2)
fi
WAILS_LDFLAGS="-X pause/internal/meta.AppBundleID=${APP_BUNDLE_ID}"
"${WAILS_CMD[@]}" build -platform darwin/universal -clean -skipbindings -tags wails -ldflags "${WAILS_LDFLAGS}"

if [[ ! -d "${APP_BUNDLE}" ]]; then
  echo "ERROR: app bundle not found: ${APP_BUNDLE}" >&2
  exit 1
fi

# Keep app/helper bundle identifiers aligned with runtime startup-manager logic.
if [[ -f "${APP_INFO_PLIST}" ]]; then
  /usr/libexec/PlistBuddy -c "Set :CFBundleIdentifier ${APP_BUNDLE_ID}" "${APP_INFO_PLIST}" >/dev/null 2>&1 \
    || /usr/libexec/PlistBuddy -c "Add :CFBundleIdentifier string ${APP_BUNDLE_ID}" "${APP_INFO_PLIST}" >/dev/null
fi
if [[ -f "${APP_INFO_PLIST}" ]]; then
  APP_VERSION="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${APP_INFO_PLIST}" 2>/dev/null || echo '1.0.0')"
fi

echo "[2/4] Embedding login helper (${HELPER_BUNDLE_ID})"
HELPER_APP="${APP_BUNDLE}/Contents/Library/LoginItems/${HELPER_NAME}.app"
HELPER_CONTENTS="${HELPER_APP}/Contents"
HELPER_EXEC="${HELPER_CONTENTS}/MacOS/${HELPER_NAME}"
HELPER_INFO_PLIST="${HELPER_CONTENTS}/Info.plist"
TMP_HELPER_SRC="$(mktemp /tmp/pause-loginhelper-XXXXXX.m)"
cleanup_helper_src() {
  rm -f "${TMP_HELPER_SRC}"
}
trap cleanup_helper_src EXIT

mkdir -p "${HELPER_CONTENTS}/MacOS"

cat > "${TMP_HELPER_SRC}" <<'EOF'
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
EOF

# Try universal helper binary first; fallback to host arch if needed.
if ! clang -fobjc-arc -framework Cocoa -mmacosx-version-min=10.13 -arch arm64 -arch x86_64 "${TMP_HELPER_SRC}" -o "${HELPER_EXEC}" 2>/dev/null; then
  clang -fobjc-arc -framework Cocoa -mmacosx-version-min=10.13 "${TMP_HELPER_SRC}" -o "${HELPER_EXEC}"
fi
chmod 755 "${HELPER_EXEC}"

cat > "${HELPER_INFO_PLIST}" <<EOF
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
EOF

# Sign nested helper and main app so legacy SMLoginItemSetEnabled can resolve the helper.
codesign --force --deep --sign "${CODE_SIGN_IDENTITY}" "${HELPER_APP}"
codesign --force --deep --sign "${CODE_SIGN_IDENTITY}" "${APP_BUNDLE}"

echo "[3/4] Preparing DMG staging folder"
rm -rf "${STAGING_DIR}"
mkdir -p "${STAGING_DIR}"
cp -R "${APP_BUNDLE}" "${STAGING_DIR}/"
ln -s /Applications "${STAGING_DIR}/Applications"

echo "[4/4] Creating DMG"
rm -f "${DMG_OUTPUT}"
hdiutil create \
  -volname "${APP_NAME}" \
  -srcfolder "${STAGING_DIR}" \
  -ov \
  -format UDZO \
  "${DMG_OUTPUT}"

echo "Done: ${DMG_OUTPUT}"
