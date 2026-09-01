#!/usr/bin/env bash

set -euo pipefail

fail() {
  printf '%s\n' "Flutter Android APK boundary failed: $*" >&2
  exit 1
}

if [[ $# -ne 1 ]]; then
  printf '%s\n' "usage: $0 APP_APK" >&2
  exit 2
fi

app_apk="$1"
[[ -s "$app_apk" ]] || fail "APK is missing or empty: $app_apk"

for tool in unzip strings grep awk sort wc; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool is unavailable: $tool"
done

android_sdk_root="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$HOME/Library/Android/sdk}}"
ndk_version="${ANYTTY_ANDROID_NDK_VERSION:-28.2.13676358}"

resolve_android_tool() {
  local override="$1" relative="$2" fallback="$3"
  if [[ -n "$override" && -x "$override" ]]; then
    printf '%s\n' "$override"
    return
  fi
  if [[ -x "$android_sdk_root/$relative" ]]; then
    printf '%s\n' "$android_sdk_root/$relative"
    return
  fi
  command -v "$fallback" 2>/dev/null || fail "$fallback is unavailable"
}

host_tag=""
case "$(uname -s)" in
  Darwin) host_tag="darwin-x86_64" ;;
  Linux) host_tag="linux-x86_64" ;;
esac
[[ -n "$host_tag" ]] || fail "unsupported verification host: $(uname -s)"

apkanalyzer="$(resolve_android_tool "${ANYTTY_APKANALYZER:-}" "cmdline-tools/latest/bin/apkanalyzer" apkanalyzer)"
apksigner="$(resolve_android_tool "${ANYTTY_APKSIGNER:-}" "build-tools/36.0.0/apksigner" apksigner)"
llvm_readelf="$(resolve_android_tool "${ANYTTY_LLVM_READELF:-}" "ndk/$ndk_version/toolchains/llvm/prebuilt/$host_tag/bin/llvm-readelf" llvm-readelf)"

unzip -tqq "$app_apk" >/dev/null 2>&1 || fail "APK archive integrity check failed: $app_apk"
if [[ "${ANYTTY_ANDROID_REQUIRE_SIGNATURE:-0}" == "1" ]]; then
  "$apksigner" verify --verbose "$app_apk" >/dev/null \
    || fail "APK signature verification failed: $app_apk"
fi

max_bytes="${ANYTTY_FLUTTER_ANDROID_MAX_APK_BYTES:-83886080}"
[[ "$max_bytes" =~ ^[1-9][0-9]*$ ]] || fail "APK byte budget is invalid: $max_bytes"
apk_bytes="$(wc -c <"$app_apk" | tr -d '[:space:]')"
(( apk_bytes <= max_bytes )) \
  || fail "APK exceeds byte budget ($apk_bytes > $max_bytes)"

apk_entries="$(unzip -Z1 "$app_apk")" || fail "APK is not a readable ZIP archive: $app_apk"
expected_abis_value="${ANYTTY_ANDROID_EXPECTED_ABIS:-arm64-v8a}"
[[ "$expected_abis_value" != *$'\n'* && "$expected_abis_value" != *$'\r'* ]] \
  || fail "ANYTTY_ANDROID_EXPECTED_ABIS must not contain CR or LF"
expected_abis_value="${expected_abis_value//,/ }"
IFS=' ' read -r -a expected_abis <<<"$expected_abis_value"
(( ${#expected_abis[@]} > 0 )) || fail "at least one expected ABI is required"

for abi in "${expected_abis[@]}"; do
  [[ "$abi" =~ ^[A-Za-z0-9_.-]+$ ]] || fail "invalid expected ABI: $abi"
  for library in libanytty_client.so libanytty_terminal_input.so libapp.so libflutter.so; do
    native_path="lib/$abi/$library"
    printf '%s\n' "$apk_entries" | grep -F -x -- "$native_path" >/dev/null \
      || fail "missing required native library: $native_path"
  done
done

packaged_abis="$({
  printf '%s\n' "$apk_entries" \
    | awk -F/ '$1 == "lib" && NF == 3 && $3 ~ /\.so$/ { print $2 }'
} | LC_ALL=C sort -u)"
expected_abis_sorted="$(printf '%s\n' "${expected_abis[@]}" | LC_ALL=C sort -u)"
[[ "$packaged_abis" == "$expected_abis_sorted" ]] \
  || fail "packaged ABIs do not match expected ABIs (expected: ${expected_abis[*]}; found: ${packaged_abis//$'\n'/, })"

if printf '%s\n' "$apk_entries" | grep -E -i \
  -e '(^|/)assets/public/' \
  -e '(^|/)capacitor\.config\.json$' \
  -e '\.(html?|m?js|wasm)$' \
  -e 'kernel_blob\.bin$' \
  -e 'libVkLayer_khronos_validation\.so$' >/dev/null; then
  fail "APK contains a Web/JavaScript or debug-runtime artifact"
fi

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/anytty-flutter-apk-boundary.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT
manifest_xml="$tmp_dir/AndroidManifest.xml"
dex_packages="$tmp_dir/dex-packages.txt"
"$apkanalyzer" manifest print "$app_apk" >"$manifest_xml" \
  || fail "could not decode final APK manifest"
"$apkanalyzer" dex packages --defined-only "$app_apk" >"$dex_packages" \
  || fail "could not inspect final APK DEX packages"

grep -F 'package="com.anytty.app"' "$manifest_xml" >/dev/null \
  || fail "final package id is not com.anytty.app"
grep -F 'android:name="com.anytty.app.MainActivity"' "$manifest_xml" >/dev/null \
  || fail "native Flutter launcher activity is missing"
if grep -F 'android:debuggable="true"' "$manifest_xml" >/dev/null; then
  fail "release APK is debuggable"
fi
if grep -E -i \
  -e 'io\.flutter\.plugins\.urllauncher\.WebViewActivity' \
  -e 'com\.getcapacitor' \
  -e 'org\.apache\.cordova' "$manifest_xml" "$dex_packages" >/dev/null; then
  fail "WebView, Capacitor, or Cordova runtime is present"
fi

native_libraries="$(printf '%s\n' "$apk_entries" | grep -E '^lib/[^/]+/[^/]+\.so$')"
[[ -n "$native_libraries" ]] || fail "final APK contains no native libraries"
native_index=0
while IFS= read -r native_path; do
  [[ -n "$native_path" ]] || continue
  native_index=$((native_index + 1))
  native_file="$tmp_dir/native-$native_index.so"
  native_strings="$tmp_dir/native-$native_index.strings"
  unzip -p "$app_apk" "$native_path" >"$native_file" \
    || fail "could not extract native library: $native_path"
  strings "$native_file" >"$native_strings" \
    || fail "could not inspect strings in $native_path"
  load_alignments="$("$llvm_readelf" -lW "$native_file" | awk '$1 == "LOAD" { print $NF }')" \
    || fail "could not inspect ELF segments in $native_path"
  [[ -n "$load_alignments" ]] || fail "native library has no ELF LOAD segments: $native_path"
  while IFS= read -r alignment; do
    [[ "$alignment" =~ ^0x[0-9A-Fa-f]+$ ]] \
      || fail "invalid ELF LOAD alignment in $native_path: $alignment"
    (( alignment >= 0x4000 )) \
      || fail "native library is not 16 KiB page aligned: $native_path ($alignment)"
  done <<<"$load_alignments"
  if grep -F \
    -e 'ANYTTY_ANDROID_GO_TAGS' \
    -e 'anytty_android_spike' \
    -e 'createSpike' \
    -e 'android-spike-daemon' \
    -e 'android-managed-1' \
    -e 'anytty-go-client-%d' "$native_strings" >/dev/null; then
    fail "forbidden development marker found in $native_path"
  fi
done <<<"$native_libraries"

printf '%s\n' "Flutter Android APK boundary passed: $app_apk ($apk_bytes bytes; ABIs: ${expected_abis[*]})"
