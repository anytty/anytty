#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
sdk_root="${ANDROID_SDK_ROOT:-${HOME}/Library/Android/sdk}"
ndk_version="${ANYTTY_ANDROID_NDK_VERSION:-28.2.13676358}"
ndk_root="${ANDROID_NDK_ROOT:-${sdk_root}/ndk/${ndk_version}}"
api="${ANYTTY_ANDROID_API:-24}"
output_root="${1:-${repo_root}/clients/flutter/android/app/src/main/jniLibs}"
ghostty_root="${repo_root}/third_party/ghostty"
terminal_source="${repo_root}/clients/flutter/native/terminal/anytty_terminal_input.c"
terminal_include="${repo_root}/clients/flutter/native/terminal"
cloud_controller_address="${ANYTTY_CLOUD_CONTROLLER_ADDRESS:-cloud.anytty.com:443}"
cloud_controller_server_name="${ANYTTY_CLOUD_CONTROLLER_SERVER_NAME:-cloud.anytty.com}"
cloud_controller_ca_pem_base64="${ANYTTY_CLOUD_CONTROLLER_CA_PEM_BASE64:-}"

for value in "${cloud_controller_address}" "${cloud_controller_server_name}" "${cloud_controller_ca_pem_base64}"; do
  if [[ "${value}" =~ [[:space:]] ]]; then
    echo "AnyTTY Cloud build configuration must not contain whitespace" >&2
    exit 1
  fi
done

if [[ ! -d "${ndk_root}" ]]; then
  echo "Android NDK ${ndk_version} is unavailable at ${ndk_root}" >&2
  exit 1
fi
if [[ ! -f "${ghostty_root}/build.zig" ]]; then
  echo "Ghostty submodule is unavailable; run: git submodule update --init" >&2
  exit 1
fi

case "$(uname -s)" in
  Darwin) host_tag="darwin-x86_64" ;;
  Linux) host_tag="linux-x86_64" ;;
  *) echo "unsupported Android build host: $(uname -s)" >&2; exit 1 ;;
esac

toolchain="${ndk_root}/toolchains/llvm/prebuilt/${host_tag}/bin"
cloud_ldflags="-checklinkname=0 -extldflags=-Wl,-z,max-page-size=16384 -X github.com/anytty/anytty/client/mobileconfig.ControllerAddress=${cloud_controller_address} -X github.com/anytty/anytty/client/mobileconfig.ControllerServerName=${cloud_controller_server_name} -X github.com/anytty/anytty/client/mobileconfig.ControllerCAPEMBase64=${cloud_controller_ca_pem_base64}"

build_abi() {
  local abi="$1" zig_target="$2" goarch="$3" clang_triple="$4" goarm="${5:-}"
  local ghostty_prefix="${repo_root}/.artifacts/ghostty/android/${abi}"
  local destination="${output_root}/${abi}"

  (
    cd "${ghostty_root}"
    zig build \
      -Demit-lib-vt=true \
      -Demit-xcframework=false \
      -Doptimize=ReleaseFast \
      -Dtarget="${zig_target}.${api}" \
      --prefix "${ghostty_prefix}"
  )

  mkdir -p "${destination}"
  "${toolchain}/${clang_triple}${api}-clang" \
    -std=c11 -O2 -fPIC -shared -Wall -Wextra -Werror \
    -DGHOSTTY_STATIC \
    -I"${ghostty_prefix}/include" \
    -I"${terminal_include}" \
    "${terminal_source}" \
    "${ghostty_prefix}/lib/libghostty-vt.a" \
    -Wl,-soname,libanytty_terminal_input.so \
    -Wl,-z,max-page-size=16384 \
    -Wl,--no-undefined \
    -o "${destination}/libanytty_terminal_input.so"

  (
    cd "${repo_root}"
    export GOOS=android GOARCH="${goarch}" CGO_ENABLED=1
    export CC="${toolchain}/${clang_triple}${api}-clang"
    if [[ -n "${goarm}" ]]; then
      export GOARM="${goarm}"
    else
      unset GOARM
    fi
    go build -trimpath -buildmode=c-shared -ldflags="${cloud_ldflags}" \
      -o "${destination}/libanytty_client.so" \
      ./client/binding/cabi/androidlib
  )
  find "${destination}" -maxdepth 1 -type f -name 'libanytty_client.h' -delete
}

build_abi arm64-v8a aarch64-linux-android arm64 aarch64-linux-android
build_abi armeabi-v7a arm-linux-androideabi arm armv7a-linux-androideabi 7
build_abi x86_64 x86_64-linux-android amd64 x86_64-linux-android
