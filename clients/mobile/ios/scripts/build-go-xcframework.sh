#!/usr/bin/env bash
set -euo pipefail

ios_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "${ios_root}/../../.." && pwd)"
developer_dir="${DEVELOPER_DIR:-/Applications/Xcode.app/Contents/Developer}"
minimum_ios="${ANYTTY_IOS_DEPLOYMENT_TARGET:-15.0}"
build_root="${ios_root}/App/build/anytty-client"
headers="${ios_root}/native/include"
output="${ios_root}/App/CapApp-SPM/Binaries/AnyTTYClient.xcframework"
cloud_controller_address="${ANYTTY_CLOUD_CONTROLLER_ADDRESS:-cloud.anytty.com:443}"
cloud_controller_server_name="${ANYTTY_CLOUD_CONTROLLER_SERVER_NAME:-cloud.anytty.com}"
cloud_controller_ca_pem_base64="${ANYTTY_CLOUD_CONTROLLER_CA_PEM_BASE64:-}"

for value in "${cloud_controller_address}" "${cloud_controller_server_name}" "${cloud_controller_ca_pem_base64}"; do
  if [[ "${value}" =~ [[:space:]] ]]; then
    echo "AnyTTY Cloud build configuration must not contain whitespace" >&2
    exit 1
  fi
done
cloud_ldflags="-checklinkname=0 -X github.com/anytty/anytty/client/mobileconfig.ControllerAddress=${cloud_controller_address} -X github.com/anytty/anytty/client/mobileconfig.ControllerServerName=${cloud_controller_server_name} -X github.com/anytty/anytty/client/mobileconfig.ControllerCAPEMBase64=${cloud_controller_ca_pem_base64}"

if [[ ! -d "${developer_dir}" ]]; then
  echo "Xcode developer directory is unavailable: ${developer_dir}" >&2
  exit 1
fi

build_archive() {
  local sdk="$1" goarch="$2" target="$3" destination="$4"
  local sdk_root
  sdk_root="$(DEVELOPER_DIR="${developer_dir}" xcrun --sdk "${sdk}" --show-sdk-path)"
  mkdir -p "$(dirname "${destination}")"
  (
    cd "${repo_root}"
    DEVELOPER_DIR="${developer_dir}" SDKROOT="${sdk_root}" \
      GOOS=ios GOARCH="${goarch}" CGO_ENABLED=1 \
      CC="xcrun --sdk ${sdk} clang" \
      CGO_CFLAGS="-isysroot ${sdk_root} -target ${target}" \
      CGO_LDFLAGS="-isysroot ${sdk_root} -target ${target}" \
      go build -trimpath -buildmode=c-archive -ldflags="${cloud_ldflags}" \
        -o "${destination}" ./clients/mobile/ios/native/go/ioslib
  )
}

rm -rf "${build_root}" "${output}"
build_archive iphoneos arm64 "arm64-apple-ios${minimum_ios}" \
  "${build_root}/iphoneos/libanytty_client.a"
build_archive iphonesimulator arm64 "arm64-apple-ios${minimum_ios}-simulator" \
  "${build_root}/iphonesimulator-arm64/libanytty_client.a"
build_archive iphonesimulator amd64 "x86_64-apple-ios${minimum_ios}-simulator" \
  "${build_root}/iphonesimulator-x86_64/libanytty_client.a"

mkdir -p "${build_root}/iphonesimulator"
DEVELOPER_DIR="${developer_dir}" xcrun lipo -create \
  "${build_root}/iphonesimulator-arm64/libanytty_client.a" \
  "${build_root}/iphonesimulator-x86_64/libanytty_client.a" \
  -output "${build_root}/iphonesimulator/libanytty_client.a"

DEVELOPER_DIR="${developer_dir}" xcodebuild -create-xcframework \
  -library "${build_root}/iphoneos/libanytty_client.a" -headers "${headers}" \
  -library "${build_root}/iphonesimulator/libanytty_client.a" -headers "${headers}" \
  -output "${output}"
