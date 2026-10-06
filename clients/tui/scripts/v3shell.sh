#!/usr/bin/env bash
# v3shell.sh - build / launch / test the Go replica of the legacy v3 TUI
# (clients/tui/examples/v3shell, recommended coralline-candy profile).
#
#   bash clients/tui/scripts/v3shell.sh run      # build + launch in this TTY (default)
#   bash clients/tui/scripts/v3shell.sh demo     # launch with the 1.txt demo state
#   bash clients/tui/scripts/v3shell.sh check    # offline: build + go test + golden diff
#   bash clients/tui/scripts/v3shell.sh test     # tmux black-box: chrome + interactions + real PTY
#
# Options:
#   --isolated   run with a throwaway XDG state/config/runtime dir (clean-room)
#   --keep       (test) keep the tmux session and print how to attach
# Requirements: go; run/demo need a real TTY, test needs tmux (SKIPs without it).
set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
GO="${GO:-go}"
CMD="run"
ISOLATED=0
KEEP=0
COLS=120
ROWS=32

while [ $# -gt 0 ]; do
  case "$1" in
    run|demo|check|test) CMD="$1" ;;
    --isolated) ISOLATED=1 ;;
    --keep) KEEP=1 ;;
    -h|--help) awk 'NR>1 && /^#/ {sub(/^# ?/, ""); print; next} NR>1 {exit}' "$0"; exit 0 ;;
    *) echo "v3shell.sh: unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
done

if { [ "$CMD" = run ] || [ "$CMD" = demo ]; } && { [ ! -t 0 ] || [ ! -t 1 ]; }; then
  echo "v3shell.sh $CMD needs a real TTY; use 'test' for the headless black-box run" >&2
  exit 1
fi

if ! command -v "$GO" >/dev/null 2>&1; then
  echo "SKIP  go not installed (set GO=/path/to/go)"
  if [ "$CMD" = check ] || [ "$CMD" = test ]; then exit 0; fi
  exit 1
fi

WORK="$(mktemp -d "${TMPDIR:-/tmp}/v3shell.XXXXXX")"
cleanup() { rm -rf "$WORK"; }
trap cleanup EXIT

echo "== building tui2 + tui2-v3shell =="
if ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2" ./clients/tui/cmd/tui2); then
  echo "FAIL  go build ./clients/tui/cmd/tui2" >&2
  exit 1
fi
if ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2-v3shell" ./clients/tui/examples/v3shell); then
  echo "FAIL  go build ./clients/tui/examples/v3shell" >&2
  exit 1
fi
echo "PASS  go build ./clients/tui/examples/v3shell"

# ---------------------------------------------------------------- offline
if [ "$CMD" = check ]; then
  echo
  echo "== go test (golden parity + interaction) =="
  (cd "$ROOT" && "$GO" test -count=1 ./clients/tui/examples/v3shell/) || exit 1

  echo
  echo "== selftest vs golden (120x32, 181x56) =="
  GO_GOLDEN="$ROOT/clients/tui/examples/v3shell/testdata/golden"
  check_screen() { # check_screen <cols> <rows> <golden>
    local cols="$1" rows="$2" golden="$3"
    "$WORK/tui2-v3shell" -selftest -cols "$cols" -rows "$rows" \
      | sed -e 's/[[:space:]]*$//' >"$WORK/selftest.txt"
    if diff -q "$GO_GOLDEN/$golden" "$WORK/selftest.txt" >/dev/null 2>&1; then
      echo "PASS  selftest ${cols}x${rows} == $golden"
    else
      echo "FAIL  selftest ${cols}x${rows} != $golden"
      diff "$GO_GOLDEN/$golden" "$WORK/selftest.txt" | head -10
      exit 1
    fi
  }
  check_screen 120 32 demo_live_120x32.txt
  check_screen 181 56 demo_live_181x56.txt

  echo
  echo "== footer scenes =="
  "$WORK/tui2-v3shell" -footer-lines >"$WORK/footer-lines.txt"
  grep -E '^(HELP|LIVE|PANE|PICKER|PROMPT|RESIZE|SYSTEM|TAB)\|' "$WORK/footer-lines.txt" >"$WORK/footer-8.txt"
  if diff -q "$ROOT/clients/tui/examples/python-shell/golden/v3_footer_120x32.txt" "$WORK/footer-8.txt" >/dev/null 2>&1; then
    echo "PASS  footer scenes (8 golden rows)"
  else
    echo "FAIL  footer scenes differ"
    diff "$ROOT/clients/tui/examples/python-shell/golden/v3_footer_120x32.txt" "$WORK/footer-8.txt" | head -10
    exit 1
  fi
  echo
  echo "v3shell check ok"
  exit 0
fi

# ------------------------------------------------------------------ tmux test
if [ "$CMD" = test ]; then
  if ! command -v tmux >/dev/null 2>&1; then
    echo "SKIP  tmux not installed"
    exit 0
  fi
  GO_GOLDEN="$ROOT/clients/tui/examples/v3shell/testdata/golden"
  SOCK="v3shell-$$"
  SESSION=""
  PASS=0
  FAIL=0
  ok() { printf 'PASS  %s\n' "$1"; PASS=$((PASS + 1)); }
  bad() { printf 'FAIL  %s\n' "$1"; FAIL=$((FAIL + 1)); }
  tm() { tmux -L "$SOCK" -f /dev/null "$@"; }
  driver_osc52() { tm show-buffer 2>/dev/null; }
  cap() { tm capture-pane -pt "$SESSION" 2>/dev/null || true; }
  capraw() { tm capture-pane -pet "$SESSION" 2>/dev/null || true; }
  send() { tm send-keys -t "$SESSION" "$@"; }
  sendl() { tm send-keys -t "$SESSION" -l "$@"; }
  send_bytes() { # send_bytes <raw string>
    local hex i hexes=()
    hex=$(printf '%s' "$1" | xxd -p -c 100000)
    for ((i = 0; i < ${#hex}; i += 2)); do hexes+=("${hex:i:2}"); done
    tm send-keys -t "$SESSION" -H "${hexes[@]}"
  }
  mouse() { send_bytes "$(printf '\x1b[<%d;%d;%d%s' "$1" "$2" "$3" "$4")"; }
  mclick() { mouse 0 "$1" "$2" M; sleep 0.05; mouse 0 "$1" "$2" m; }
  must() { # must <secs> <desc> <regex>
    local deadline=$((SECONDS + $1))
    while ((SECONDS < deadline)); do
      if cap | grep -Eq "$3"; then ok "$2"; return; fi
      sleep 0.2
    done
    bad "$2"
  }
  v3_line_match() { # v3_line_match <line> <desc>
    local n="$1" desc="$2" golden deadline=$((SECONDS + 6))
    golden=$(sed -n "${n}p" "$GO_GOLDEN/demo_live_120x32.txt")
    while :; do
      if [ "$(cap | sed -e 's/[[:space:]]*$//' | sed -n "${n}p")" = "$golden" ]; then ok "$desc"; return; fi
      ((SECONDS >= deadline)) && break
      sleep 0.3
    done
    bad "$desc"
  }
  stop() {
    if [ "$KEEP" -eq 1 ]; then
      echo "kept session: tmux -L $SOCK attach -t $SESSION"
      return
    fi
    tm kill-server 2>/dev/null || true
  }
  trap cleanup EXIT

  export XDG_STATE_HOME="$WORK/state" XDG_CONFIG_HOME="$WORK/config" XDG_RUNTIME_DIR="$WORK/run"
  mkdir -p "$XDG_STATE_HOME" "$XDG_CONFIG_HOME" "$XDG_RUNTIME_DIR"

  echo
  echo "== v3go chrome (demo state, 120x32) =="
  SESSION="v3go-demo"
  tm new-session -d -s "$SESSION" -x "$COLS" -y 32 \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-v3shell -demo\"; echo V3GOEXIT:\$?; exec bash'"
  sleep 0.9
  v3_line_match 1 "header chip row matches the golden"
  v3_line_match 2 "window frame + title + action group match the golden"
  v3_line_match 32 "bottom status bar matches the golden"
  send C-p
  must 3 "PANE footer group shows the v3 recommended keys" 'CTRL\+D.*VSPLIT'
  # Original geometry: sibling cards touch (their own borders), no separator.
  send '%'
  v3go_adjacent=0
  deadline=$((SECONDS + 6))
  while ((SECONDS < deadline)); do
    if cap | sed -n 2p | grep -q '┐┌' && [ "$(cap | grep -c '│││' || true)" -eq 0 ]; then v3go_adjacent=1; break; fi
    sleep 0.3
  done
  if [ "$v3go_adjacent" -eq 1 ]; then ok "split cards are adjacent (no separator column)"; else bad "split cards are adjacent (no separator column)"; fi
  send h
  send z
  v3go_zoom=0
  deadline=$((SECONDS + 6))
  while ((SECONDS < deadline)); do
    if cap | grep -q 'anytty-surface@hs' && ! cap | grep -q 'opencode@hs'; then v3go_zoom=1; break; fi
    sleep 0.3
  done
  if [ "$v3go_zoom" -eq 1 ]; then ok "z zooms the focused card (other leaf hidden)"; else bad "z zooms the focused card"; fi
  send z
  must 5 "second z unzooms" 'opencode@hs'
  send Escape
  sleep 0.3
  must 3 "Esc returns to the live footer" '󰌌 CTRL'
  send_bytes $'\x1b[118;6u'
  must 5 "ctrl-shift-v (CSI-u) reaches the program" 'paste requested'
  send Escape
  sleep 0.2
  mclick 53 1
  must 5 "header create button adds a tab (T3)" 'T3'
  send C-q
  must 5 "Ctrl-Q opens the host confirmation" 'Quit tui2\?'
  send Enter
  must 8 "quitting restores the terminal" 'V3GOEXIT:0'

  echo
  echo "== v3go real PTYs =="
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="v3go-pty"
  tm new-session -d -s "$SESSION" -x "$COLS" -y 32 \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-v3shell\"; echo V3GOPTYEXIT:\$?; exec bash'"
  tm set-option -g set-clipboard on
  sleep 0.9
  must 5 "cold start opens the terminal picker" 'Terminal Picker'
  must 5 "picker partitions by endpoint (local tab)" '▸ ● local'
  must 5 "cold start picker offers + New terminal" '\+ New terminal'
  send Enter
  send Enter
  must 6 "enter creates and binds a terminal" 'term-1@local'
  sendl 'echo V3GO-OK'
  send Enter
  must 6 "bound terminal accepts input" 'V3GO-OK'
  send C-p
  send '%'
  # main creates an empty panel first; choose the explicit create action.
  send Escape
  send C-f
  sleep 0.2
  send Enter
  send Enter
  must 5 "Ctrl-P % creates an empty panel, then explicit terminal create binds it" 'term-2@local'
  send C-p
  # Copy mode is modal; clicking the other pane must end it and take input.
  send C-p
  send h
  sleep 0.4
  for i in 1 2 3; do mouse 64 20 10 M; sleep 0.05; done
  must 3 "wheel enters the copy scene (COPY badge)" 'COPY'
  mclick 100 10
  must 3 "clicking another pane leaves copy mode" '󰌌 CTRL'
  sendl 'echo COPY-EXIT-OK'
  send Enter
  must 5 "typing after the click reaches the other pane" 'COPY-EXIT-OK'
  # The session is per pane: returning to the scrolled pane resumes copy, and
  # scrolling back to the bottom closes it automatically (old AtFrozenBottom).
  mclick 20 10
  must 3 "clicking back resumes the copy scene (COPY badge)" 'COPY'
  for i in $(seq 1 40); do mouse 65 20 10 M; sleep 0.03; done
  sleep 0.5
  must 3 "scrolling back to the bottom leaves copy mode" '󰌌 CTRL'
  # Copy scene end to end: search a known string, select it, copy through the
  # host (OSC52) and leave with G.
  mclick 20 10
  sleep 0.4
  sendl "printf 'COPY''ME\\n'"
  send Enter
  sleep 0.8
  send_bytes $'\x1b[99;6u'
  must 3 "ctrl-shift-c enters the copy scene (COPY badge)" 'COPY'
  send /
  sleep 0.2
  sendl "COPYME"
  sleep 0.2
  send Enter
  sleep 0.8
  send Space
  sleep 0.2
  send End
  sleep 0.2
  if capraw | grep -q '43m'; then ok "selection is highlighted (old ansi 8/3 colors)"; else bad "selection is highlighted (old ansi 8/3 colors)"; fi
  send y
  v3go_copied=0
  deadline=$((SECONDS + 6))
  while ((SECONDS < deadline)); do
    if driver_osc52 "$SESSION" 2>/dev/null | grep -q 'COPYME'; then v3go_copied=1; break; fi
    sleep 0.3
  done
  if [ "$v3go_copied" -eq 1 ]; then ok "y copies the selection to the clipboard (OSC52)"; else bad "y copies the selection to the clipboard (OSC52)"; fi
  send G
  sleep 0.4
  must 3 "G leaves the copy scene" '󰌌 CTRL'
  mclick 100 10
  sleep 0.4
  send C-p
  sleep 0.3
  send x
  sleep 0.6
  v3go_closed=0
  deadline=$((SECONDS + 6))
  while ((SECONDS < deadline)); do
    if ! cap | grep -q 'term-2@local'; then v3go_closed=1; break; fi
    sleep 0.3
  done
  if [ "$v3go_closed" -eq 1 ]; then ok "x closes the focused split card"; else bad "x closes the focused split card"; fi
  send C-q
  must 5 "Ctrl-Q opens the host confirmation" 'Quit tui2\?'
  send Enter
  must 8 "quitting restores the terminal" 'V3GOPTYEXIT:0'
  tm set-option -g set-clipboard on
  stop

  echo
  echo "v3shell test: $PASS passed, $FAIL failed"
  [ "$FAIL" -eq 0 ]
  exit $?
fi

# ------------------------------------------------------------------ run/demo

# ensure_local_stack: the registry's `local` endpoint is a daemon route to the
# canonical socket, so terminal.create waits for the dial budget (default
# 3s+5s) when the stack is down. Bring it up first; --isolated skips this
# (with no registry the host creates local PTYs directly).
ensure_local_stack() {
  local bin="$WORK/anytty" sock=""
  if ! (cd "$ROOT" && "$GO" build -o "$bin" ./cmd/anytty) >"$WORK/anytty-build.log" 2>&1; then
    echo "warn: cannot build anytty; local terminals may wait for the dial budget" >&2
    return 0
  fi
  sock=$("$bin" pool status 2>/dev/null | awk '/^Socket/{print $2}')
  if [ -n "$sock" ] && [ -S "$sock" ]; then
    return 0
  fi
  echo "== starting the local pool+access stack (local terminals need it) =="
  if ! "$bin" pool start >"$WORK/pool-start.log" 2>&1; then
    echo "warn: pool start failed (see $WORK/pool-start.log); continuing" >&2
    tail -3 "$WORK/pool-start.log" >&2 || true
  fi
}

if [ "$ISOLATED" -eq 1 ]; then
  export XDG_STATE_HOME="$WORK/state" XDG_CONFIG_HOME="$WORK/config" XDG_RUNTIME_DIR="$WORK/run"
  mkdir -p "$XDG_STATE_HOME" "$XDG_CONFIG_HOME" "$XDG_RUNTIME_DIR"
  echo "(isolated XDG: $WORK)"
fi

if [ "$ISOLATED" -eq 0 ]; then
  ensure_local_stack
fi

SHELL_CMD="$WORK/tui2-v3shell"
if [ "$CMD" = demo ]; then
  SHELL_CMD="$SHELL_CMD -demo"
fi

cat <<'BANNER'
============================================================
 anytty v3 replica (Go)
   Ctrl-P PANE    Ctrl-R SIZE    Ctrl-O FLOAT   Ctrl-T TAB
   Ctrl-W WS      Ctrl-F PICK    Ctrl-G SYSTEM  Ctrl-Q quit
   live: typing goes to the focused terminal
============================================================
BANNER

"$WORK/tui2" -shell "$SHELL_CMD"
status=$?
exit $status
