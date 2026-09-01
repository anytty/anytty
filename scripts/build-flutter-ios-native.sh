#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
developer_dir="${DEVELOPER_DIR:-/Applications/Xcode.app/Contents/Developer}"
minimum_ios="${ANYTTY_IOS_DEPLOYMENT_TARGET:-15.0}"
build_root="${repo_root}/.artifacts/flutter-ios-native"
output_root="${1:-${repo_root}/clients/flutter/ios/Native}"
ghostty_root="${repo_root}/third_party/ghostty"
terminal_source="${repo_root}/clients/flutter/native/terminal/anytty_terminal_input.c"
terminal_include="${repo_root}/clients/flutter/native/terminal"
client_headers="${repo_root}/clients/mobile/ios/native/include"
cloud_controller_address="${ANYTTY_CLOUD_CONTROLLER_ADDRESS:-cloud.anytty.com:443}"
cloud_controller_server_name="${ANYTTY_CLOUD_CONTROLLER_SERVER_NAME:-cloud.anytty.com}"
cloud_controller_ca_pem_base64="${ANYTTY_CLOUD_CONTROLLER_CA_PEM_BASE64:-}"

for value in "${cloud_controller_address}" "${cloud_controller_server_name}" "${cloud_controller_ca_pem_base64}"; do
  if [[ "${value}" =~ [[:space:]] ]]; then
    echo "AnyTTY Cloud build configuration must not contain whitespace" >&2
    exit 1
  fi
done

if [[ ! -x "${developer_dir}/usr/bin/xcodebuild" ]]; then
  echo "Full Xcode is required at ${developer_dir}" >&2
  exit 1
fi
if ! DEVELOPER_DIR="${developer_dir}" xcrun --sdk iphoneos --show-sdk-path >/dev/null 2>&1; then
  echo "The iPhoneOS SDK is unavailable from ${developer_dir}" >&2
  exit 1
fi
if [[ ! -f "${ghostty_root}/build.zig" ]]; then
  echo "Ghostty submodule is unavailable; run: git submodule update --init" >&2
  exit 1
fi
if [[ ! -f "${client_headers}/anytty_client.h" ]]; then
  echo "AnyTTY Client C headers are unavailable" >&2
  exit 1
fi

cloud_ldflags="-checklinkname=0 -X github.com/anytty/anytty/client/mobileconfig.ControllerAddress=${cloud_controller_address} -X github.com/anytty/anytty/client/mobileconfig.ControllerServerName=${cloud_controller_server_name} -X github.com/anytty/anytty/client/mobileconfig.ControllerCAPEMBase64=${cloud_controller_ca_pem_base64}"

build_go_archive() {
  local sdk="$1" target="$2" destination="$3"
  local sdk_root
  sdk_root="$(DEVELOPER_DIR="${developer_dir}" xcrun --sdk "${sdk}" --show-sdk-path)"
  mkdir -p "$(dirname "${destination}")"
  (
    cd "${repo_root}"
    DEVELOPER_DIR="${developer_dir}" SDKROOT="${sdk_root}" \
      GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
      CC="xcrun --sdk ${sdk} clang" \
      CGO_CFLAGS="-isysroot ${sdk_root} -target ${target}" \
      CGO_LDFLAGS="-isysroot ${sdk_root} -target ${target}" \
      go build -trimpath -buildmode=c-archive -ldflags="${cloud_ldflags}" \
        -o "${destination}" ./clients/mobile/ios/native/go/ioslib
  )
}

build_terminal_archive() {
  local sdk="$1" target="$2" ghostty_slice="$3" destination="$4"
  local sdk_root object ghostty_library
  sdk_root="$(DEVELOPER_DIR="${developer_dir}" xcrun --sdk "${sdk}" --show-sdk-path)"
  object="${destination%.a}.o"
  ghostty_library="$(find "${ghostty_slice}" -maxdepth 1 -type f -name 'libghostty-vt*.a' -print -quit)"
  if [[ -z "${ghostty_library}" ]]; then
    echo "Ghostty static library is unavailable in ${ghostty_slice}" >&2
    exit 1
  fi
  mkdir -p "$(dirname "${destination}")"
  DEVELOPER_DIR="${developer_dir}" xcrun --sdk "${sdk}" clang \
    -std=c11 -O2 -fPIC -Wall -Wextra -Werror \
    -isysroot "${sdk_root}" -target "${target}" \
    -DGHOSTTY_STATIC \
    -I"${ghostty_slice}/Headers" \
    -I"${terminal_include}" \
    -c "${terminal_source}" -o "${object}"
  DEVELOPER_DIR="${developer_dir}" xcrun --sdk "${sdk}" libtool -static \
    -o "${destination}" "${object}" "${ghostty_library}"
}

rm -rf "${build_root}"
mkdir -p "${build_root}" "${output_root}"

build_go_archive iphoneos "arm64-apple-ios${minimum_ios}" \
  "${build_root}/client/iphoneos/libanytty_client.a"
build_go_archive iphonesimulator "arm64-apple-ios${minimum_ios}-simulator" \
  "${build_root}/client/iphonesimulator/libanytty_client.a"

rm -rf "${output_root}/AnyTTYClient.xcframework"
DEVELOPER_DIR="${developer_dir}" xcodebuild -create-xcframework \
  -library "${build_root}/client/iphoneos/libanytty_client.a" -headers "${client_headers}" \
  -library "${build_root}/client/iphonesimulator/libanytty_client.a" -headers "${client_headers}" \
  -output "${output_root}/AnyTTYClient.xcframework"

ghostty_prefix="${build_root}/ghostty"
(
  cd "${ghostty_root}"
  DEVELOPER_DIR="${developer_dir}" zig build \
    -Demit-lib-vt=true \
    -Demit-xcframework=true \
    -Doptimize=ReleaseFast \
    --prefix "${ghostty_prefix}"
)
ghostty_xcframework="${ghostty_prefix}/lib/ghostty-vt.xcframework"

build_terminal_archive iphoneos "arm64-apple-ios${minimum_ios}" \
  "${ghostty_xcframework}/ios-arm64" \
  "${build_root}/terminal/iphoneos/libanytty_terminal_input.a"
build_terminal_archive iphonesimulator "arm64-apple-ios${minimum_ios}-simulator" \
  "${ghostty_xcframework}/ios-arm64-simulator" \
  "${build_root}/terminal/iphonesimulator/libanytty_terminal_input.a"

terminal_headers="${build_root}/terminal/headers"
mkdir -p "${terminal_headers}"
cp "${terminal_include}/anytty_terminal_input.h" "${terminal_headers}/"
rm -rf "${output_root}/AnyTTYTerminalInput.xcframework"
DEVELOPER_DIR="${developer_dir}" xcodebuild -create-xcframework \
  -library "${build_root}/terminal/iphoneos/libanytty_terminal_input.a" -headers "${terminal_headers}" \
  -library "${build_root}/terminal/iphonesimulator/libanytty_terminal_input.a" -headers "${terminal_headers}" \
  -output "${output_root}/AnyTTYTerminalInput.xcframework"
