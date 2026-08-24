#!/usr/bin/env bash

set -euo pipefail

fail() {
  printf '%s\n' "Android APK boundary failed: $*" >&2
  exit 1
}

if [[ $# -ne 1 ]]; then
  printf '%s\n' "usage: $0 APP_APK" >&2
  exit 2
fi

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
app_apk="$1"
[[ -s "$app_apk" ]] || fail "APK is missing or empty: $app_apk"

for tool in node unzip strings grep awk cmp; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool is unavailable: $tool"
done

resolve_llvm_readelf() {
  if [[ -n "${ANYTTY_LLVM_READELF:-}" && -x "$ANYTTY_LLVM_READELF" ]]; then
    printf '%s\n' "$ANYTTY_LLVM_READELF"
    return
  fi
  local android_sdk_root="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$HOME/Library/Android/sdk}}"
  local ndk_version="${ANYTTY_ANDROID_NDK_VERSION:-27.2.12479018}"
  local host_tag
  case "$(uname -s)" in
    Darwin) host_tag='darwin-x86_64' ;;
    Linux) host_tag='linux-x86_64' ;;
    *) host_tag='' ;;
  esac
  if [[ -n "$host_tag" && -x "$android_sdk_root/ndk/$ndk_version/toolchains/llvm/prebuilt/$host_tag/bin/llvm-readelf" ]]; then
    printf '%s\n' "$android_sdk_root/ndk/$ndk_version/toolchains/llvm/prebuilt/$host_tag/bin/llvm-readelf"
    return
  fi
  command -v llvm-readelf 2>/dev/null || fail 'llvm-readelf is unavailable'
}
if ! unzip -tqq "$app_apk" >/dev/null 2>&1; then
  fail "APK archive integrity check failed: $app_apk"
fi

resolve_apkanalyzer() {
  if [[ -n "${ANYTTY_APKANALYZER:-}" && -x "$ANYTTY_APKANALYZER" ]]; then
    printf '%s\n' "$ANYTTY_APKANALYZER"
    return
  fi
  if [[ -n "${ANDROID_HOME:-}" && -x "$ANDROID_HOME/cmdline-tools/latest/bin/apkanalyzer" ]]; then
    printf '%s\n' "$ANDROID_HOME/cmdline-tools/latest/bin/apkanalyzer"
    return
  fi
  command -v apkanalyzer 2>/dev/null || fail 'apkanalyzer is unavailable'
}

resolve_aapt2() {
  if [[ -n "${ANYTTY_AAPT2:-}" && -x "$ANYTTY_AAPT2" ]]; then
    printf '%s\n' "$ANYTTY_AAPT2"
    return
  fi
  local android_sdk_root="${ANDROID_HOME:-${ANDROID_SDK_ROOT:-$HOME/Library/Android/sdk}}"
  if [[ -x "$android_sdk_root/build-tools/36.0.0/aapt2" ]]; then
    printf '%s\n' "$android_sdk_root/build-tools/36.0.0/aapt2"
    return
  fi
  command -v aapt2 2>/dev/null || fail 'aapt2 is unavailable'
}

apkanalyzer="$(resolve_apkanalyzer)"
aapt2="$(resolve_aapt2)"
llvm_readelf="$(resolve_llvm_readelf)"
apk_entries="$(unzip -Z1 "$app_apk")" || fail "APK is not a readable ZIP archive: $app_apk"

expected_abis_value="${ANYTTY_ANDROID_EXPECTED_ABIS:-arm64-v8a x86_64}"
if [[ "$expected_abis_value" == *$'\n'* || "$expected_abis_value" == *$'\r'* ]]; then
  fail 'ANYTTY_ANDROID_EXPECTED_ABIS must not contain CR or LF'
fi
if [[ "$expected_abis_value" =~ [[:cntrl:]] ]]; then
  fail 'ANYTTY_ANDROID_EXPECTED_ABIS may only use spaces or commas as separators'
fi
expected_abis_value="${expected_abis_value//,/ }"
IFS=' ' read -r -a expected_abis <<<"$expected_abis_value"
(( ${#expected_abis[@]} > 0 )) || fail 'ANYTTY_ANDROID_EXPECTED_ABIS must name at least one ABI'

for abi in "${expected_abis[@]}"; do
  [[ "$abi" =~ ^[A-Za-z0-9_.-]+$ ]] || fail "invalid expected ABI: $abi"
  for library in libanytty_client.so libanytty_client_jni.so; do
    native_path="lib/$abi/$library"
    if ! printf '%s\n' "$apk_entries" | grep -F -x -- "$native_path" >/dev/null; then
      fail "missing required native library: $native_path"
    fi
  done
done

tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/anytty-apk-boundary.XXXXXX")"
trap 'rm -rf "$tmp_dir"' EXIT

manifest_xml="$tmp_dir/AndroidManifest.xml"
resource_table="$tmp_dir/resources.txt"
if ! "$apkanalyzer" manifest print "$app_apk" >"$manifest_xml"; then
  fail 'could not decode final APK manifest'
fi
if ! "$aapt2" dump resources "$app_apk" >"$resource_table"; then
  fail 'could not decode final APK resource table'
fi
node "$repo_root/clients/mobile/scripts/verify-android-merged-manifest.mjs" "$manifest_xml" "$resource_table" || fail 'final APK manifest contract failed'

resolve_xml_resource_path() {
  local resource_name="$1"
  awk -v target="xml/$resource_name" '
    $1 == "resource" && $3 == target { matched = 1; next }
    matched && $1 == "resource" { exit }
    matched && $1 == "()" && $2 == "(file)" && $4 == "type=XML" { print $3; exit }
  ' "$resource_table"
}

for resource in network_security_config backup_rules data_extraction_rules; do
  resource_path="$(resolve_xml_resource_path "$resource")"
  [[ "$resource_path" =~ ^res/[A-Za-z0-9_./-]+\.xml$ ]] || fail "invalid compiled path for xml/$resource: $resource_path"
  if ! printf '%s\n' "$apk_entries" | grep -F -x -- "$resource_path" >/dev/null; then
    fail "compiled resource is missing from APK: xml/$resource -> $resource_path"
  fi
  output="$tmp_dir/$resource.xml"
  if ! "$aapt2" dump xmltree "$app_apk" --file "$resource_path" >"$output"; then
    fail "could not decode final APK resource: $resource"
  fi
done

index_html="$tmp_dir/index.html"
if ! unzip -p "$app_apk" assets/public/index.html >"$index_html" || [[ ! -s "$index_html" ]]; then
  fail 'final APK is missing assets/public/index.html'
fi
node "$repo_root/clients/mobile/scripts/verify-android-artifact-resources.mjs" \
  "$tmp_dir/network_security_config.xml" \
  "$tmp_dir/backup_rules.xml" \
  "$tmp_dir/data_extraction_rules.xml" \
  "$index_html" || fail 'final APK resource contract failed'

capacitor_config="$tmp_dir/capacitor.config.json"
if ! unzip -p "$app_apk" assets/capacitor.config.json >"$capacitor_config" || [[ ! -s "$capacitor_config" ]]; then
  fail 'final APK is missing assets/capacitor.config.json'
fi
node --input-type=module - "$capacitor_config" <<'NODE' || fail 'final APK Capacitor logging must be disabled'
import { readFileSync } from 'node:fs'
const config = JSON.parse(readFileSync(process.argv[2], 'utf8'))
if (config.loggingBehavior !== 'none') process.exit(1)
NODE

web_root="$tmp_dir/web"
mkdir -p "$web_root"
if ! unzip -qq "$app_apk" 'assets/public/*' -d "$web_root"; then
  fail 'could not extract final APK web assets'
fi
node "$repo_root/clients/mobile/scripts/verify-production-bundle.mjs" \
  "$web_root/assets/public" || fail 'final APK web JavaScript/CSS contract failed'

dex_packages="$tmp_dir/dex-packages.txt"
if ! "$apkanalyzer" dex packages --defined-only "$app_apk" >"$dex_packages"; then
  fail 'could not inspect final APK DEX packages'
fi
if grep -E -e 'org\.java_websocket' -e 'org\.slf4j' "$dex_packages" >/dev/null; then
  fail 'removed JVM bridge dependency is present in the final APK'
fi
if printf '%s\n' "$apk_entries" | grep -F -x -- 'META-INF/services/org.slf4j.spi.SLF4JServiceProvider' >/dev/null; then
  fail 'removed SLF4J service provider is present in the final APK'
fi

native_libraries="$(printf '%s\n' "$apk_entries" | grep -E '^lib/[^/]+/[^/]+\.so$')"
[[ -n "$native_libraries" ]] || fail 'final APK contains no native libraries'
native_index=0
while IFS= read -r native_path; do
  [[ -n "$native_path" ]] || continue
  native_index=$((native_index + 1))
  native_file="$tmp_dir/native-$native_index.so"
  native_strings="$tmp_dir/native-$native_index.strings"
  unzip -p "$app_apk" "$native_path" >"$native_file" || fail "could not extract native library: $native_path"
  strings "$native_file" >"$native_strings" || fail "could not inspect strings in $native_path"
  load_alignments="$("$llvm_readelf" -lW "$native_file" | awk '$1 == "LOAD" { print $NF }')" \
    || fail "could not inspect ELF segments in $native_path"
  [[ -n "$load_alignments" ]] || fail "native library has no ELF LOAD segments: $native_path"
  while IFS= read -r alignment; do
    [[ "$alignment" =~ ^0x[0-9A-Fa-f]+$ ]] || fail "invalid ELF LOAD alignment in $native_path: $alignment"
    (( alignment >= 0x4000 )) || fail "native library is not 16 KB page aligned: $native_path ($alignment)"
  done <<<"$load_alignments"
  if grep -F \
    -e 'ANYTTY_ANDROID_GO_TAGS' \
    -e 'anytty_android_spike' \
    -e 'createSpike' \
    -e 'android-spike-daemon' \
    -e 'android-managed-1' \
    -e 'anytty-go-client-%d' "$native_strings" >/dev/null; then
    fail "forbidden Android dev/spike marker found in $native_path"
  fi
done <<<"$native_libraries"

printf '%s\n' "Android APK boundary passed: $app_apk"
