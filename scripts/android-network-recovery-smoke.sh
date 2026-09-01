#!/usr/bin/env bash
set -euo pipefail

adb_bin="${ADB:-adb}"
package_name="${ANYTTY_ANDROID_PACKAGE:-com.anytty.app}"
activity_name="${ANYTTY_ANDROID_ACTIVITY:-com.anytty.app.MainActivity}"
background_seconds="${ANYTTY_BACKGROUND_SECONDS:-30}"
settle_seconds="${ANYTTY_RECOVERY_SETTLE_SECONDS:-30}"
log_directory="no_backup/anytty-diagnostics"
native_log_name="native-connection.log"
app_log_name="app-events.log"

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
  "${adb_bin}" shell dumpsys battery reset >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

read_log_stream() {
  local base_name="$1"
  local suffix
  local content
  for suffix in .3 .2 .1 ""; do
    if content="$(
      "${adb_bin}" exec-out run-as "${package_name}" \
        cat "${log_directory}/${base_name}${suffix}" 2>/dev/null
    )" && [[ -n "${content}" ]]; then
      printf '%s\n' "${content}"
    fi
  done
}

last_supervisor_line() {
  awk '/endpoint_supervisor/ { line = $0 } END { print line }'
}

supervisor_is_ready() {
  local line
  line="$(last_supervisor_line)"
  [[ "${line}" == *" phase=ready "* && "${line}" == *" error_code=" ]]
}

extract_after_marker() {
  local marker="$1"
  awk -v marker="${marker}" '
    found { print }
    $0 == marker { found = 1; next }
    END { if (!found) exit 1 }
  '
}

app_recovery_is_complete() {
  awk '
    /web event=foreground_recovery_done / { recovery_done = 1 }
    /web event=endpoint_resume_settled / {
      total = resumed = failed = -1
      for (field_index = 1; field_index <= NF; field_index++) {
        split($field_index, field, "=")
        if (field[1] == "total") total = field[2] + 0
        if (field[1] == "resumed") resumed = field[2] + 0
        if (field[1] == "failed") failed = field[2] + 0
      }
      endpoints_settled = total > 0 && resumed == total && failed == 0
    }
    /web event=session_state / { last_session_state = $0 }
    END {
      exit !(recovery_done && endpoints_settled && last_session_state ~ / to=connected /)
    }
  '
}

foreground_service_is_active() {
  "${adb_bin}" shell dumpsys activity services "${package_name}" | awk '
    /AnyTTYConnectionService/ { connection_service = 1 }
    connection_service && /isForeground=true/ { active = 1 }
    END { exit !active }
  '
}

"${adb_bin}" shell am start -W -n "${package_name}/${activity_name}" >/dev/null
deadline=$((SECONDS + settle_seconds))
baseline_native_log=""
while (( SECONDS <= deadline )); do
  baseline_native_log="$(read_log_stream "${native_log_name}")"
  if supervisor_is_ready <<<"${baseline_native_log}" && foreground_service_is_active; then
    break
  fi
  sleep 1
done

if [[ -z "${baseline_native_log}" ]]; then
  echo "native recovery log is unavailable; install a debug APK and open a demanded endpoint first" >&2
  exit 1
fi
if ! supervisor_is_ready <<<"${baseline_native_log}" || ! foreground_service_is_active; then
  echo "no demanded endpoint is READY before backgrounding; open a machine and retry" >&2
  exit 1
fi
baseline_app_log="$(read_log_stream "${app_log_name}")"
if [[ -z "${baseline_app_log}" ]]; then
  echo "app recovery log is unavailable; install a debug APK and retry" >&2
  exit 1
fi
native_marker="$(tail -n 1 <<<"${baseline_native_log}")"
app_marker="$(tail -n 1 <<<"${baseline_app_log}")"

"${adb_bin}" shell input keyevent KEYCODE_HOME
"${adb_bin}" shell dumpsys battery unplug >/dev/null
"${adb_bin}" shell dumpsys deviceidle force-idle >/dev/null
idle_state="$(
  "${adb_bin}" shell dumpsys deviceidle | awk '
    /mState=/ {
      sub(/^.*mState=/, "")
      sub(/[[:space:]\r].*$/, "")
      print
      exit
    }
  '
)"
if [[ "${idle_state}" != "IDLE" ]]; then
  echo "device did not enter deep idle; observed state=${idle_state:-unknown}" >&2
  exit 1
fi
sleep "${background_seconds}"
"${adb_bin}" shell dumpsys deviceidle unforce >/dev/null
"${adb_bin}" shell dumpsys battery reset >/dev/null
"${adb_bin}" shell am start -W -n "${package_name}/${activity_name}" >/dev/null

deadline=$((SECONDS + settle_seconds))
recovery_native_log=""
recovery_app_log=""
recovery_complete=false
while (( SECONDS <= deadline )); do
  native_log="$(read_log_stream "${native_log_name}")"
  app_log="$(read_log_stream "${app_log_name}")"
  if ! recovery_native_log="$(extract_after_marker "${native_marker}" <<<"${native_log}")"; then
    echo "native recovery marker was lost during log rotation" >&2
    exit 1
  fi
  if ! recovery_app_log="$(extract_after_marker "${app_marker}" <<<"${app_log}")"; then
    echo "app recovery marker was lost during log rotation" >&2
    exit 1
  fi
  if supervisor_is_ready <<<"${recovery_native_log}" && app_recovery_is_complete <<<"${recovery_app_log}"; then
    recovery_complete=true
    break
  fi
  sleep 1
done

awk '/endpoint_supervisor/ { lines[++count] = $0 } END { start = count > 80 ? count - 79 : 1; for (line_index = start; line_index <= count; line_index++) print lines[line_index] }' <<<"${recovery_native_log}"
awk '/web event=(foreground_recovery|endpoint_resume|session_state)/ { lines[++count] = $0 } END { start = count > 80 ? count - 79 : 1; for (line_index = start; line_index <= count; line_index++) print lines[line_index] }' <<<"${recovery_app_log}"

if [[ "${recovery_complete}" != true ]]; then
  echo "foreground recovery did not reach native READY and renderer connected before the deadline" >&2
  exit 1
fi

echo "foreground recovery smoke test passed"
