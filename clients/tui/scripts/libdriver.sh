#!/usr/bin/env bash
# Test driver abstraction shared by the tui2 smoke and acceptance scripts.
#
# Primitives (all drivers):
#   driver_spawn <name> <cols> <rows> <command>   start a session (command is
#                                                 run through the shell)
#   driver_send <name> <key>...                   tmux-style key names
#   driver_send_literal <name> <text>             literal text, no key lookup
#   driver_send_bytes <name> <hex>...             raw bytes (two hex digits per
#                                                 argument, like tmux -H)
#   driver_capture <name>                         visible text rows
#   driver_capture_raw <name>                     visible rows with SGR runs
#   driver_resize <name> <cols> <rows>            resize the window
#   driver_cursor <name>                          "x,y" (zero-based)
#   driver_osc52 <name>                           last OSC 52 clipboard text
#   driver_kill <name>                            terminate one session
#   driver_kill_all                               terminate every session
#   driver_shutdown                               release the driver backend
#   driver_supports <capability>                  0 when the driver can
#                                                 assert the capability
#
# Backends:
#   tmux  the historical driver; used whenever tmux is installed
#   pty   the built-in Go/PTY harness (clients/tui/cmd/tui2-harness)
#
# Environment:
#   TUI2_TEST_DRIVER=auto|tmux|pty  force a backend (auto prefers tmux)
#   TUI2_HARNESS_BIN=<path>         prebuilt tui2-harness (pty driver)
#   TUI2_TMUX=(tmux ...)            tmux command vector (tmux driver)
#   GO=go                           used to build the pty harness
#   WORK=<dir>                      build/output directory for the harness
#
# The tm adapter below accepts the subset of tmux invocations the scripts use,
# so both drivers run the exact same assertions.

: "${TUI2_TEST_DRIVER:=auto}"
DRIVER=""
HARNESS_BIN=""
TUI2_DRIVER_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
if ! declare -p TUI2_TMUX >/dev/null 2>&1; then
  TUI2_TMUX=(tmux)
fi
DRIVER_CAPTURE_TMP="${TMPDIR:-/tmp}/tui2-driver-$$"

driver_init() {
  local requested="$TUI2_TEST_DRIVER" go_bin="${GO:-go}"
  case "$requested" in
    tmux) DRIVER=tmux ;;
    pty) DRIVER=pty ;;
    auto | "")
      if command -v tmux >/dev/null 2>&1; then
        DRIVER=tmux
      else
        DRIVER=pty
      fi
      ;;
    *)
      printf 'driver: unknown TUI2_TEST_DRIVER=%s (want auto|tmux|pty)\n' "$requested" >&2
      return 2
      ;;
  esac
  if [ "$DRIVER" = tmux ]; then
    printf '== test driver: tmux (TUI2_TEST_DRIVER=%s) ==\n' "$requested"
    return 0
  fi
  if ! command -v base64 >/dev/null 2>&1; then
    printf 'SKIP  test driver pty needs base64\n'
    return 3
  fi
  if [ -n "${TUI2_HARNESS_BIN:-}" ]; then
    HARNESS_BIN="$TUI2_HARNESS_BIN"
  else
    if ! command -v "$go_bin" >/dev/null 2>&1; then
      printf 'SKIP  test driver pty needs go (set GO=... or TUI2_HARNESS_BIN=...)\n'
      return 3
    fi
    HARNESS_BIN="${WORK:-${TMPDIR:-/tmp}}/tui2-harness"
    if ! (cd "$TUI2_DRIVER_ROOT" && "$go_bin" build -o "$HARNESS_BIN" ./clients/tui/cmd/tui2-harness); then
      printf 'FAIL  build tui2-harness for the pty driver\n'
      return 2
    fi
  fi
  if [ ! -x "$HARNESS_BIN" ]; then
    printf 'SKIP  tui2-harness not executable: %s\n' "$HARNESS_BIN"
    return 3
  fi
  coproc TUI2_HRN { "$HARNESS_BIN" serve; }
  # Coprocess descriptors do not survive into pipeline subshells, but regular
  # shell descriptors duplicated from them do. The capture helpers below are
  # routinely used as `capture | grep`, so duplicate the pipes first.
  exec {TUI2_HRN_IN}>&"${TUI2_HRN[1]}"
  exec {TUI2_HRN_OUT}<&"${TUI2_HRN[0]}"
  printf '== test driver: pty (TUI2_TEST_DRIVER=%s, harness=%s) ==\n' "$requested" "$HARNESS_BIN"
  return 0
}

driver_shutdown() {
  if [ "$DRIVER" = pty ] && [ -n "${TUI2_HRN_PID:-}" ]; then
    local hrn_pid="$TUI2_HRN_PID" i
    printf 'quit\n' >&"$TUI2_HRN_IN" 2>/dev/null || true
    for i in 1 2 3 4 5 6 7 8 9 10; do
      kill -0 "$hrn_pid" 2>/dev/null || break
      sleep 0.1
    done
    kill "$hrn_pid" 2>/dev/null || true
    wait "$hrn_pid" 2>/dev/null || true
  fi
  if [ -n "${TUI2_HRN_IN:-}" ]; then exec {TUI2_HRN_IN}>&- 2>/dev/null || true; fi
  if [ -n "${TUI2_HRN_OUT:-}" ]; then exec {TUI2_HRN_OUT}<&- 2>/dev/null || true; fi
  rm -rf "$DRIVER_CAPTURE_TMP" 2>/dev/null || true
}

# driver_supports <capability>: both current drivers cover the capabilities the
# assertions need. It exists so a backend can declare a gap instead of silently
# producing a wrong result; callers print SKIP with the named reason.
driver_supports() {
  case "$1" in
    resize | osc52 | raw | cursor | mouse | paste) return 0 ;;
    transient-notice)
      # The layout program keeps one transient status slot, so a later
      # endpoint notice can overwrite an earlier one before a capture. tmux's
      # slower session startup lets the notice queue settle; the pty driver
      # starts faster and can lose the race for a specific message. The
      # durable log-file assertion still runs under both drivers.
      [ "$DRIVER" = tmux ] && return 0
      return 1
      ;;
    *) return 1 ;;
  esac
}

# _hrn_req <request>: send one line to the pty harness and read one response.
_hrn_req() {
  local response
  if ! printf '%s\n' "$1" >&"$TUI2_HRN_IN" 2>/dev/null; then
    printf 'driver: harness write failed\n' >&2
    return 3
  fi
  if ! IFS= read -r -t 30 response <&"$TUI2_HRN_OUT"; then
    printf 'driver: harness response timeout for: %s\n' "$1" >&2
    return 3
  fi
  case "$response" in
    ERR*)
      printf 'driver: %s\n' "$response" >&2
      return 1
      ;;
    OK* | HIT | MISS | ALIVE | EXITED*)
      printf '%s\n' "$response"
      return 0
      ;;
    *)
      printf 'driver: unexpected harness response: %s\n' "$response" >&2
      return 1
      ;;
  esac
}

_b64() { printf '%s' "$1" | base64 | tr -d '\n'; }

_hrn_capture() { # <name> <raw:0|1>
  local tmp="$DRIVER_CAPTURE_TMP-capture"
  local command="capture"
  [ "$2" = 1 ] && command="capture_raw"
  _hrn_req "$command $1 $tmp" >/dev/null || return 1
  cat "$tmp"
}

driver_spawn() { # <name> <cols> <rows> <command>
  case "$DRIVER" in
    tmux)
      "${TUI2_TMUX[@]}" new-session -d -s "$1" -x "$2" -y "$3" "$4"
      ;;
    pty)
      _hrn_req "spawn $1 $2 $3 $(_b64 "$4")" >/dev/null
      ;;
  esac
}

driver_send() { # <name> <key>...
  local name="$1"
  shift
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" send-keys -t "$name" "$@" ;;
    pty)
      local key
      for key in "$@"; do
        _hrn_req "key $name $(_b64 "$key")" >/dev/null || return 1
      done
      ;;
  esac
}

driver_send_literal() { # <name> <text>...
  local name="$1"
  shift
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" send-keys -t "$name" -l "$@" ;;
    pty)
      local text
      for text in "$@"; do
        _hrn_req "send $name $(_b64 "$text")" >/dev/null || return 1
      done
      ;;
  esac
}

driver_send_bytes() { # <name> <hex>...
  local name="$1"
  shift
  local hex=""
  local part
  for part in "$@"; do
    hex+="$part"
  done
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" send-keys -t "$name" -H "$@" ;;
    pty) _hrn_req "sendhex $name $hex" >/dev/null ;;
  esac
}

driver_capture() { _hrn_capture_tmux "$1" 0; }

driver_capture_raw() { _hrn_capture_tmux "$1" 1; }

# _hrn_capture_tmux keeps one adapter for both capture variants: the tmux
# backend uses capture-pane, the pty backend asks the harness to write a file.
_hrn_capture_tmux() { # <name> <raw:0|1>
  case "$DRIVER" in
    tmux)
      if [ "$2" = 1 ]; then
        "${TUI2_TMUX[@]}" capture-pane -pet "$1" 2>/dev/null || true
      else
        "${TUI2_TMUX[@]}" capture-pane -pt "$1" 2>/dev/null || true
      fi
      ;;
    pty) _hrn_capture "$1" "$2" ;;
  esac
}

driver_resize() { # <name> <cols> <rows>
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" resize-window -t "$1" -x "$2" -y "$3" ;;
    pty) _hrn_req "resize $1 $2 $3" >/dev/null ;;
  esac
}

driver_cursor() { # <name>
  local response
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" display-message -p -t "$1" '#{cursor_x},#{cursor_y}' 2>/dev/null || true ;;
    pty)
      response="$(_hrn_req "cursor $1")" || return 1
      printf '%s\n' "${response#OK }"
      ;;
  esac
}

driver_osc52() { # <name>
  local response tmp="$DRIVER_CAPTURE_TMP-osc52"
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" show-buffer 2>/dev/null ;;
    pty)
      response="$(_hrn_req "osc52 $1 $tmp")" || return 1
      [ "$response" = HIT ] || return 1
      cat "$tmp"
      ;;
  esac
}

driver_kill() { # <name>
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" kill-session -t "$1" 2>/dev/null || true ;;
    pty) _hrn_req "kill $1" >/dev/null || true ;;
  esac
}

driver_kill_all() {
  case "$DRIVER" in
    tmux) "${TUI2_TMUX[@]}" kill-server 2>/dev/null || true ;;
    pty) [ -n "${TUI2_HRN_PID:-}" ] && _hrn_req "killall" >/dev/null || true ;;
  esac
}

# ---------------------------------------------------------------- tm adapter
# tm <tmux args> translates the tmux invocations used by the scripts to the
# active driver. In tmux mode it is a thin pass-through, so the historical
# behavior is preserved byte for byte.
tm() {
  if [ "$DRIVER" = tmux ]; then
    "${TUI2_TMUX[@]}" "$@"
    return
  fi
  [ "$1" = "--" ] && shift
  local command="$1"
  shift
  case "$command" in
    kill-server) driver_kill_all ;;
    new-session) _tm_new_session "$@" ;;
    send-keys) _tm_send_keys "$@" ;;
    capture-pane) _tm_capture_pane "$@" ;;
    resize-window) _tm_resize_window "$@" ;;
    show-buffer) driver_osc52 "${SESSION:-}" ;;
    display-message) _tm_display_message "$@" ;;
    set-option) : ;; # clipboard/history settings are implicit in the pty driver
    kill-session) _tm_kill_session "$@" ;;
    *)
      printf 'driver: unsupported tmux command for the pty driver: %s\n' "$command" >&2
      return 2
      ;;
  esac
}

_tm_new_session() {
  local name="" cols="" rows="" command=""
  while [ $# -gt 0 ]; do
    case "$1" in
      -d) shift ;;
      -s) name="$2"; shift 2 ;;
      -x) cols="$2"; shift 2 ;;
      -y) rows="$2"; shift 2 ;;
      --) shift ;;
      *) command="$1"; shift ;;
    esac
  done
  driver_spawn "$name" "$cols" "$rows" "$command"
}

_tm_send_keys() {
  local name="" literal=0 hex=0
  while [ $# -gt 0 ]; do
    case "$1" in
      -t) name="$2"; shift 2 ;;
      -l) literal=1; shift ;;
      -H) hex=1; shift ;;
      --) shift ;;
      *) break ;;
    esac
  done
  if [ "$hex" = 1 ]; then
    driver_send_bytes "$name" "$@"
  elif [ "$literal" = 1 ]; then
    driver_send_literal "$name" "$@"
  else
    driver_send "$name" "$@"
  fi
}

_tm_capture_pane() {
  local name="" raw=0
  while [ $# -gt 0 ]; do
    case "$1" in
      -t) name="$2"; shift 2 ;;
      -p) shift ;;
      -e) raw=1; shift ;;
      -pt) name="$2"; shift 2 ;;
      -pe) raw=1; shift ;;
      -pet) raw=1; name="$2"; shift 2 ;;
      -p*) : ;; # only -p variants appear in the scripts
      *) shift ;;
    esac
  done
  if [ "$raw" = 1 ]; then driver_capture_raw "$name"; else driver_capture "$name"; fi
}

_tm_resize_window() {
  local name="" cols="" rows=""
  while [ $# -gt 0 ]; do
    case "$1" in
      -t) name="$2"; shift 2 ;;
      -x) cols="$2"; shift 2 ;;
      -y) rows="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  driver_resize "$name" "$cols" "$rows"
}

_tm_display_message() {
  local name="" format=""
  while [ $# -gt 0 ]; do
    case "$1" in
      -p) shift ;;
      -t) name="$2"; shift 2 ;;
      *) format="$1"; shift ;;
    esac
  done
  case "$format" in
    *cursor_x*) driver_cursor "$name" ;;
    *) : ;;
  esac
}

_tm_kill_session() {
  local name=""
  while [ $# -gt 0 ]; do
    case "$1" in
      -t) name="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  driver_kill "$name"
}
