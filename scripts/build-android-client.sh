#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
sdk_root="${ANDROID_SDK_ROOT:-${HOME}/Library/Android/sdk}"
ndk_version="${ANYTTY_ANDROID_NDK_VERSION:-27.2.12479018}"
ndk_root="${ANDROID_NDK_ROOT:-${sdk_root}/ndk/${ndk_version}}"
output_root="${1:-${repo_root}/clients/mobile/android/app/build/generated/anyttyJniLibs}"
api="${ANYTTY_ANDROID_API:-24}"
cloud_controller_address="${ANYTTY_CLOUD_CONTROLLER_ADDRESS:-cloud.anytty.com:443}"
cloud_controller_server_name="${ANYTTY_CLOUD_CONTROLLER_SERVER_NAME:-cloud.anytty.com}"
cloud_controller_ca_pem_base64="${ANYTTY_CLOUD_CONTROLLER_CA_PEM_BASE64:-}"

for value in "${cloud_controller_address}" "${cloud_controller_server_name}" "${cloud_controller_ca_pem_base64}"; do
  if [[ "${value}" =~ [[:space:]] ]]; then
    echo "AnyTTY Cloud build configuration must not contain whitespace" >&2
    exit 1
  fi
done
cloud_ldflags="-checklinkname=0 -extldflags=-Wl,-z,max-page-size=16384 -X github.com/anytty/anytty/client/mobileconfig.ControllerAddress=${cloud_controller_address} -X github.com/anytty/anytty/client/mobileconfig.ControllerServerName=${cloud_controller_server_name} -X github.com/anytty/anytty/client/mobileconfig.ControllerCAPEMBase64=${cloud_controller_ca_pem_base64}"

if [[ ! -d "${ndk_root}" ]]; then
  echo "Android NDK ${ndk_version} is not installed at ${ndk_root}" >&2
  exit 1
fi

case "$(uname -s)" in
  Darwin) host_tag="darwin-x86_64" ;;
  Linux) host_tag="linux-x86_64" ;;
  *) echo "unsupported Android build host: $(uname -s)" >&2; exit 1 ;;
esac

toolchain="${ndk_root}/toolchains/llvm/prebuilt/${host_tag}/bin"
include_dir="${repo_root}/client/binding/cabi"
jni_source="${repo_root}/clients/mobile/android/app/src/main/cpp/anytty_client_jni.c"

build_abi() {
  local abi="$1" goarch="$2" triple="$3" goarm="${4:-}"
  local destination="${output_root}/${abi}"
  mkdir -p "${destination}"
  (
    cd "${repo_root}"
    export GOOS=android GOARCH="${goarch}" CGO_ENABLED=1 CC="${toolchain}/${triple}${api}-clang"
    if [[ -n "${goarm}" ]]; then
      export GOARM="${goarm}"
    else
      unset GOARM
    fi
    # Pion 的 Android interface adapter 仍使用受控 linkname；Go 1.23+ 需要显式允许该上游实现。
    go build -trimpath -buildmode=c-shared -ldflags="${cloud_ldflags}" \
      -o "${destination}/libanytty_client.so" ./client/binding/cabi/androidlib
  )
  "${toolchain}/${triple}${api}-clang" -shared -fPIC \
    -I"${include_dir}" "${jni_source}" \
    -L"${destination}" -lanytty_client \
    -Wl,-soname,libanytty_client_jni.so -Wl,-z,max-page-size=16384 \
    -o "${destination}/libanytty_client_jni.so"
  rm -f "${destination}/libanytty_client.h"
}

build_abi armeabi-v7a arm armv7a-linux-androideabi 7
build_abi arm64-v8a arm64 aarch64-linux-android
build_abi x86_64 amd64 x86_64-linux-android
