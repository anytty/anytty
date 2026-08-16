#!/usr/bin/env bash
set -euo pipefail

ios_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "${ios_root}/../../.." && pwd)"
developer_dir="${DEVELOPER_DIR:-/Applications/Xcode.app/Contents/Developer}"
project="${ios_root}/App/App.xcodeproj"
root_manifest="${ios_root}/App/App/PrivacyInfo.xcprivacy"
package_manifest="${ios_root}/App/CapApp-SPM/Sources/CapApp-SPM/PrivacyInfo.xcprivacy"
info_plist="${ios_root}/App/App/Info.plist"
source_capacitor_config="${repo_root}/clients/mobile/capacitor.config.ts"
ios_capacitor_config="${ios_root}/App/App/capacitor.config.json"
icon="${ios_root}/App/App/Assets.xcassets/AppIcon.appiconset/AppIcon-512@2x.png"
android_gradle="${repo_root}/clients/mobile/android/app/build.gradle"
app_path="${1:-}"

fail() {
  echo "iOS release verification failed: $*" >&2
  exit 1
}

require_text() {
  local file="$1" value="$2"
  rg -q --fixed-strings "${value}" "${file}" || fail "${file} is missing ${value}"
}

[[ -d "${developer_dir}" ]] || fail "Xcode developer directory is unavailable: ${developer_dir}"
for plist in "${info_plist}" "${root_manifest}" "${package_manifest}"; do
  plutil -lint "${plist}" >/dev/null || fail "invalid plist: ${plist}"
done

for localization in \
  "${ios_root}/App/App/en.lproj/InfoPlist.strings" \
  "${ios_root}/App/App/zh-Hans.lproj/InfoPlist.strings"; do
  plutil -lint "${localization}" >/dev/null || fail "invalid localization: ${localization}"
  require_text "${localization}" "NSCameraUsageDescription"
  require_text "${localization}" "NSLocalNetworkUsageDescription"
done

for value in \
  NSPrivacyCollectedDataTypeDeviceID \
  NSPrivacyCollectedDataTypeCoarseLocation \
  NSPrivacyCollectedDataTypeOtherUsageData \
  NSPrivacyAccessedAPICategoryFileTimestamp \
  NSPrivacyAccessedAPICategorySystemBootTime \
  NSPrivacyAccessedAPICategoryUserDefaults; do
  require_text "${root_manifest}" "${value}"
done
for value in C617.1 3B52.1 35F9.1 CA92.1; do
  require_text "${root_manifest}" "${value}"
done

tracking="$(/usr/libexec/PlistBuddy -c 'Print :NSPrivacyTracking' "${root_manifest}")"
[[ "${tracking}" == "false" ]] || fail "tracking must remain disabled unless ATT and disclosures are added"
if /usr/libexec/PlistBuddy -c 'Print :NSAppTransportSecurity:NSAllowsArbitraryLoads' "${info_plist}" >/dev/null 2>&1; then
  arbitrary_loads="$(/usr/libexec/PlistBuddy -c 'Print :NSAppTransportSecurity:NSAllowsArbitraryLoads' "${info_plist}")"
  [[ "${arbitrary_loads}" == "false" ]] || fail "NSAllowsArbitraryLoads must not be enabled"
fi
require_text "${source_capacitor_config}" "appStartPath: 'index.html'"
ios_start_path="$(node -e 'const fs = require("fs"); console.log(JSON.parse(fs.readFileSync(process.argv[1], "utf8")).server?.appStartPath ?? "")' "${ios_capacitor_config}")"
[[ "${ios_start_path}" == "index.html" ]] || fail "generated Capacitor config must start at index.html"

icon_width="$(sips -g pixelWidth "${icon}" 2>/dev/null | awk '/pixelWidth:/ { print $2; exit }')"
icon_height="$(sips -g pixelHeight "${icon}" 2>/dev/null | awk '/pixelHeight:/ { print $2; exit }')"
icon_alpha="$(sips -g hasAlpha "${icon}" 2>/dev/null | awk '/hasAlpha:/ { print $2; exit }')"
[[ "${icon_width}" == "1024" && "${icon_height}" == "1024" ]] || fail "App Store icon must be 1024 x 1024"
[[ "${icon_alpha}" == "no" ]] || fail "App Store icon must not contain transparency"

[[ -f "${ios_root}/App/CapApp-SPM/Binaries/AnyTTYClient.xcframework/Info.plist" ]] || \
  fail "Go XCFramework is missing; run ios/scripts/build-go-xcframework.sh first"

settings="$(DEVELOPER_DIR="${developer_dir}" xcodebuild \
  -project "${project}" \
  -scheme App \
  -configuration Release \
  -sdk iphoneos \
  -destination 'generic/platform=iOS' \
  -showBuildSettings 2>/dev/null)"
setting_value() {
  local key="$1"
  printf '%s\n' "${settings}" | awk -v key="${key}" '$1 == key && $2 == "=" { print $3; exit }'
}

bundle_id="$(setting_value PRODUCT_BUNDLE_IDENTIFIER)"
ios_version="$(setting_value MARKETING_VERSION)"
ios_build="$(setting_value CURRENT_PROJECT_VERSION)"
minimum_ios="$(setting_value IPHONEOS_DEPLOYMENT_TARGET)"
device_family="$(setting_value TARGETED_DEVICE_FAMILY)"
[[ "${bundle_id}" == "com.anytty.app" ]] || fail "unexpected bundle ID: ${bundle_id}"
[[ -n "${ios_version}" && -n "${ios_build}" ]] || fail "version and build number are required"
[[ "${minimum_ios}" == "15.0" ]] || fail "unexpected deployment target: ${minimum_ios}"
[[ "${device_family}" == "1,2" ]] || fail "release must remain universal for iPhone and iPad"

android_version="$(awk '/^[[:space:]]*versionName / { gsub(/"/, "", $2); print $2; exit }' "${android_gradle}")"
android_build="$(awk '/^[[:space:]]*versionCode / { print $2; exit }' "${android_gradle}")"
[[ "${ios_version}" == "${android_version}" ]] || fail "iOS ${ios_version} and Android ${android_version} versions differ"
[[ "${ios_build}" == "${android_build}" ]] || fail "iOS ${ios_build} and Android ${android_build} build numbers differ"
require_text "${repo_root}/clients/mobile/src/AnyTTYApp.tsx" "https://cloud.anytty.com/privacy"

if [[ -n "${app_path}" ]]; then
  [[ -d "${app_path}" ]] || fail "built app is unavailable: ${app_path}"
  built_info="${app_path}/Info.plist"
  built_manifest="${app_path}/PrivacyInfo.xcprivacy"
  built_capacitor_config="${app_path}/capacitor.config.json"
  [[ -f "${built_manifest}" ]] || fail "built app does not contain a root PrivacyInfo.xcprivacy"
  [[ -f "${built_capacitor_config}" ]] || fail "built app does not contain capacitor.config.json"
  built_start_path="$(node -e 'const fs = require("fs"); console.log(JSON.parse(fs.readFileSync(process.argv[1], "utf8")).server?.appStartPath ?? "")' "${built_capacitor_config}")"
  [[ "${built_start_path}" == "index.html" ]] || fail "built app may launch blank because appStartPath is not index.html"
  plutil -lint "${built_info}" "${built_manifest}" >/dev/null || fail "built app contains an invalid plist"
  [[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "${built_info}")" == "${ios_version}" ]] || fail "built app version differs"
  [[ "$(/usr/libexec/PlistBuddy -c 'Print :CFBundleVersion' "${built_info}")" == "${ios_build}" ]] || fail "built app number differs"
  executable="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "${built_info}")"
  DEVELOPER_DIR="${developer_dir}" xcrun lipo -archs "${app_path}/${executable}" | rg -q '(^| )arm64( |$)' || fail "built app is missing arm64"
  while IFS= read -r manifest; do
    plutil -lint "${manifest}" >/dev/null || fail "invalid embedded privacy manifest: ${manifest}"
  done < <(find "${app_path}" -name PrivacyInfo.xcprivacy -type f -print)
  if [[ -d "${app_path}/_CodeSignature" ]]; then
    codesign --verify --deep --strict "${app_path}" || fail "built app signature is invalid"
  fi
fi

echo "iOS release verification passed (${bundle_id} ${ios_version} build ${ios_build}, iOS ${minimum_ios}+)."
