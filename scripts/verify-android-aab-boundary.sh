#!/usr/bin/env bash

set -euo pipefail

fail() {
  printf '%s\n' "Android AAB boundary failed: $*" >&2
  exit 1
}

if [[ $# -ne 1 ]]; then
  printf '%s\n' "usage: $0 APP_AAB" >&2
  exit 2
fi

app_aab="$1"
[[ -s "$app_aab" ]] || fail "AAB is missing or empty: $app_aab"

for tool in jarsigner unzip awk grep sort; do
  command -v "$tool" >/dev/null 2>&1 || fail "required tool is unavailable: $tool"
done

unzip -tqq "$app_aab" >/dev/null 2>&1 || fail "AAB archive integrity check failed: $app_aab"
jarsigner -verify "$app_aab" >/dev/null 2>&1 || fail "AAB signature verification failed: $app_aab"

aab_entries="$(unzip -Z1 "$app_aab")" || fail "AAB is not a readable ZIP archive: $app_aab"
expected_abis_value="${ANYTTY_ANDROID_EXPECTED_ABIS:-armeabi-v7a arm64-v8a x86_64}"
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
    native_path="base/lib/$abi/$library"
    if ! printf '%s\n' "$aab_entries" | grep -F -x -- "$native_path" >/dev/null; then
      fail "missing required native library: $native_path"
    fi
  done
done

packaged_abis="$({
  printf '%s\n' "$aab_entries" | awk -F/ '$1 == "base" && $2 == "lib" && NF == 4 && $4 ~ /\.so$/ { print $3 }'
} | LC_ALL=C sort -u)"
expected_abis_sorted="$(printf '%s\n' "${expected_abis[@]}" | LC_ALL=C sort -u)"
if [[ "$packaged_abis" != "$expected_abis_sorted" ]]; then
  fail "packaged ABIs do not match expected ABIs (expected: ${expected_abis[*]}; found: ${packaged_abis//$'\n'/, })"
fi

printf '%s\n' "Android AAB boundary passed: $app_aab"
