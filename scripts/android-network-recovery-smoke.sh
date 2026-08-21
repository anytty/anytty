#!/usr/bin/env bash
set -euo pipefail

adb_bin="${ADB:-adb}"
package_name="${ANYTTY_ANDROID_PACKAGE:-com.anytty.app}"
activity_name="${ANYTTY_ANDROID_ACTIVITY:-com.anytty.app.MainActivity}"
background_seconds="${ANYTTY_BACKGROUND_SECONDS:-30}"
settle_seconds="${ANYTTY_RECOVERY_SETTLE_SECONDS:-12}"
log_path="cache/anytty-debug-share/anytty-native-connection.log"

if ! [[ "${background_seconds}" =~ ^[0-9]+$ && "${settle_seconds}" =~ ^[0-9]+$ ]]; then
  echo "background and settle durations must be non-negative integers" >&2
  exit 2
fi

device_count="$("${adb_bin}" devices | awk 'NR > 1 && $2 == "device" { count += 1 } END { print count + 0 }')"
if [[ "${device_count}" -ne 1 ]]; then
  echo "exactly one authorized adb device is required; found ${device_count}" >&2
  exit 2
fi

cleanup() {
  "${adb_bin}" shell dumpsys deviceidle unforce >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

read_native_log() {
  "${adb_bin}" shell run-as "${package_name}" sh -c "cat ${log_path} 2>/dev/null" || true
}

"${adb_bin}" shell am start -W -n "${package_name}/${activity_name}" >/dev/null
sleep "${settle_seconds}"
baseline_log="$(read_native_log)"
if [[ -z "${baseline_log}" ]]; then
  echo "native recovery log is unavailable; install a debug APK and open a demanded endpoint first" >&2
  exit 1
fi
if ! awk '/endpoint_supervisor.*phase=ready/ { found = 1 } END { exit !found }' <<<"${baseline_log}"; then
  echo "no demanded endpoint is READY before backgrounding; open a machine and retry" >&2
  exit 1
fi
baseline_lines="$(awk 'END { print NR }' <<<"${baseline_log}")"

"${adb_bin}" shell input keyevent KEYCODE_HOME
"${adb_bin}" shell dumpsys deviceidle force-idle >/dev/null
sleep "${background_seconds}"
"${adb_bin}" shell dumpsys deviceidle unforce >/dev/null
"${adb_bin}" shell am start -W -n "${package_name}/${activity_name}" >/dev/null
sleep "${settle_seconds}"

log_output="$(read_native_log)"
if [[ -z "${log_output}" ]]; then
  echo "native recovery log disappeared during the test" >&2
  exit 1
fi
current_lines="$(awk 'END { print NR }' <<<"${log_output}")"
if [[ "${current_lines}" -le "${baseline_lines}" ]]; then
  echo "no new native recovery records were written after foreground resume" >&2
  exit 1
fi
recovery_log="$(tail -n "+$((baseline_lines + 1))" <<<"${log_output}")"

awk '/endpoint_supervisor/ { lines[++count] = $0 } END { start = count > 80 ? count - 79 : 1; for (index = start; index <= count; index++) print lines[index] }' <<<"${recovery_log}"

if ! awk '/endpoint_supervisor.*phase=ready/ { found = 1 } END { exit !found }' <<<"${recovery_log}"; then
  echo "no supervisor READY projection was observed after foreground recovery" >&2
  exit 1
fi

echo "foreground recovery smoke test passed"
