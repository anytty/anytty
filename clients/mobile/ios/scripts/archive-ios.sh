#!/usr/bin/env bash
set -euo pipefail

ios_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "${ios_root}/../../.." && pwd)"
developer_dir="${DEVELOPER_DIR:-/Applications/Xcode.app/Contents/Developer}"
team="${ANYTTY_IOS_DEVELOPMENT_TEAM:-}"
archive_path="${ANYTTY_IOS_ARCHIVE_PATH:-${ios_root}/DerivedData-release/Archives/AnyTTY.xcarchive}"

[[ -d "${developer_dir}" ]] || { echo "Xcode developer directory is unavailable: ${developer_dir}" >&2; exit 1; }
[[ -n "${team}" ]] || {
  echo "Set ANYTTY_IOS_DEVELOPMENT_TEAM to the Apple Developer Team ID used for com.anytty.app." >&2
  exit 1
}
[[ ! -e "${archive_path}" ]] || {
  echo "Archive already exists; choose a new ANYTTY_IOS_ARCHIVE_PATH: ${archive_path}" >&2
  exit 1
}

(
  cd "${repo_root}"
  npm run build --workspace @anytty/mobile
)
"${ios_root}/scripts/sync-web-assets.sh"
DEVELOPER_DIR="${developer_dir}" "${ios_root}/scripts/build-go-xcframework.sh"
DEVELOPER_DIR="${developer_dir}" "${ios_root}/scripts/verify-ios-release.sh"

provisioning_args=()
if [[ "${ANYTTY_IOS_ALLOW_PROVISIONING_UPDATES:-NO}" == "YES" ]]; then
  provisioning_args+=("-allowProvisioningUpdates")
fi

mkdir -p "$(dirname "${archive_path}")"
DEVELOPER_DIR="${developer_dir}" xcodebuild \
  -project "${ios_root}/App/App.xcodeproj" \
  -scheme App \
  -configuration Release \
  -destination 'generic/platform=iOS' \
  -archivePath "${archive_path}" \
  DEVELOPMENT_TEAM="${team}" \
  CODE_SIGN_STYLE=Automatic \
  "${provisioning_args[@]}" \
  clean archive

DEVELOPER_DIR="${developer_dir}" "${ios_root}/scripts/verify-ios-release.sh" \
  "${archive_path}/Products/Applications/App.app"

echo "Signed archive: ${archive_path}"
echo "Open it in Xcode Organizer, then choose Distribute App > App Store Connect."
