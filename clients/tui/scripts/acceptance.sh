#!/usr/bin/env bash
# tui2 black-box acceptance for the locally verifiable items of
# SCENARIOS.zh-CN.md §13 (and the MAP figure 9 paths). Drives the real host +
# layout program through the shared test driver layer: an isolated tmux server
# (its own socket, no user config) when tmux is installed, otherwise the
# built-in Go/PTY harness. Force one with TUI2_TEST_DRIVER=tmux|pty. The
# assertions below are driver-neutral and assert only what the
# screen/clipboard/process table shows.
#
# Coverage:
#   §1  cold start picker, create, typing
#   §1b slot title bar action buttons: icons, ✕ unbind+rebind, ⟳ restart
#   §13.2 external tmux resize -> inner PTY stty size follows and restores
#   §2  split, empty pane, click/Tab focus, picker bind, Esc->PTY
#   §2b claim keys, Ctrl-F double-click forward, wheel three conditions
#   §2/§8 mouse: click focuses the slot (title highlight), picker row click
#        binds, sidebar tab row click switches
#   §13.13 paste: bracket multiline, >max_paste_bytes chunk order
#   §4  scrollback [↑N], wheel, y copy via OSC52, Esc live
#   §3  exit badge [exited N], Ctrl-E restart, close=unbind + rebind
#   §3  crash recovery: kill -9 program, frame kept, auto restart, rebind
#   §10 Ctrl-T tab, 1..9 switch
#   §5  : prompt help/quit, ? help, host quit confirm deny/allow
#   §9  divider drag changes ratio and PTY winsize (stty size)
#   legacy.py pixel replica: golden screen diffs, picker/prompt, scroll, badge
#   v3ui.py legacy v3 replica: golden chrome rows, pane focus/click, split +
#        collapse hint, footer scenes, ctrl-shift-v key delivery, tab create
#   §13.5 Ctrl-Q deny stays usable, allow exits clean, no leftovers
#   ENDPOINTS: real isolated dev daemon over local-unix (list/attach/input/
#        resize/kill/restart), command+daemon coexistence, offline notice and
#        unsupported tcp connect_mode
#   LOG/ROUTE: old registry with webrtc/cloud routes: no shared-layer log on
#        the alt screen, diagnostics in the log file, route pruning/offline
#        notice/local degradation, TUI2_ROUTES opt-in, startup failure logging
#
# Usage: bash clients/tui/scripts/acceptance.sh   (requires go; tmux optional)
#
# SIGHUP environment: the isolated daemons kill terminals with SIGHUP. A
# `nohup`/SIGHUP-ignoring parent leaks SIG_IGN into the daemon's interactive
# `sh`, which then correctly survives the daemon's SIGHUP-based kill and makes
# the two killed-terminal assertions fail (a test-environment artifact, not a
# product regression). Re-exec once with the disposition restored, so the
# script is stable under nohup and SIGHUP-ignoring CI runners too; where GNU
# `env --default-signal` is unavailable (e.g. macOS) the documented
# foreground-run caveat still applies.
set -u
if [[ "${TUI2_ACCEPTANCE_HUP_RESET:-}" != "1" ]] && env --default-signal=HUP true >/dev/null 2>&1; then
  TUI2_ACCEPTANCE_HUP_RESET=1 exec env --default-signal=HUP bash "${BASH_SOURCE[0]}" "$@"
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
# shellcheck source=libdriver.sh
source "$ROOT/clients/tui/scripts/libdriver.sh"
GO="${GO:-go}"
SOCK="tui2-acc-$$"
SESSION="tui2acc"
WORK="$(mktemp -d)"
# The TUI host loads the shared CLI endpoint registry at startup (M2), so the
# whole acceptance run uses an isolated XDG tree: no developer endpoints.yaml
# leaks into the black-box assertions, and the CLI-registry section below can
# write to it deterministically.
export XDG_CONFIG_HOME="$WORK/xdg/config"
export XDG_STATE_HOME="$WORK/xdg/state"
export XDG_RUNTIME_DIR="$WORK/xdg/run"
mkdir -p "$XDG_CONFIG_HOME" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR"
# Program-side config file driven end to end: the acceptance view keeps the
# panes full width (sidebar off) with the default gutter and clock. Both the
# host (theme) and tui2-shell (layout) read $ANYTTY_TUI2_CONFIG.
cat >"$WORK/tui2-config.json" <<'CFG'
{ "sidebar": false, "gap": 1, "clock": { "enabled": true, "format": "24h" } }
CFG
export ANYTTY_TUI2_CONFIG="$WORK/tui2-config.json"
COLS=120
ROWS=36
PASS=0
FAIL=0

cleanup() {
  [[ -n "${TUNNEL_PID:-}" ]] && kill "$TUNNEL_PID" 2>/dev/null || true
  [[ -n "${FWD_PID:-}" ]] && kill "$FWD_PID" 2>/dev/null || true
  [[ -n "${REMOTE_SOCK:-}" ]] && pkill -f -- "$REMOTE_SOCK" 2>/dev/null || true
  # The isolated dev daemon(s) live only for this run; never leak a daemon
  # bound to this run's signaling port or unix socket.
  pkill -f -- "${DEV_BIN:-/nonexistent-dev-bin}.*$WORK" 2>/dev/null || true
  driver_kill_all 2>/dev/null || true
  driver_shutdown
  rm -rf "$WORK"
}
trap cleanup EXIT

ok() { printf 'PASS  %s\n' "$1"; PASS=$((PASS + 1)); }
bad() { printf 'FAIL  %s\n' "$1"; FAIL=$((FAIL + 1)); }

if ! command -v "$GO" >/dev/null 2>&1 && [ -z "${TUI2_HARNESS_BIN:-}" ]; then
  echo "SKIP  go not installed (set GO=/path/to/go or TUI2_HARNESS_BIN=...)"
  exit 0
fi
for tool in xxd md5sum sed awk cut; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "SKIP  $tool not installed"
    exit 0
  fi
done
# Pick the driver before the first session starts: tmux when installed, the
# built-in pty harness otherwise. TUI2_TEST_DRIVER forces one backend.
TUI2_TMUX=(tmux -L "$SOCK" -f /dev/null)
driver_init || exit 0
# The protocol-only Python reference shell is exercised below; its interpreter
# is the only extra requirement (stdlib only, no pip packages).
PY="${PY:-python3}"
NODE="${NODE:-node}"
if ! command -v "$PY" >/dev/null 2>&1; then
  echo "SKIP  python3 not installed (set PY=/path/to/python3)"
  exit 0
fi
# The isolated dev daemon must not contend with a developer daemon on the
# default signaling ports: pick two free loopback ports for this run.
DEV_SIGNAL_PORT="$("$PY" -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()')"
export ANYTTY_DIRECT_SIGNALING_LISTEN="127.0.0.1:$DEV_SIGNAL_PORT"
export ANYTTY_DIRECT_ICE_TCP_LISTEN="127.0.0.1:$((DEV_SIGNAL_PORT + 1))"

echo "== building tui2 and tui2-shell =="
if ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2" ./clients/tui/cmd/tui2); then
  bad "go build ./clients/tui/cmd/tui2"
  exit 1
fi
if ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2-shell" ./clients/tui/cmd/tui2-shell); then
  bad "go build ./clients/tui/cmd/tui2-shell"
  exit 1
fi

# ------------------------------------------------------- driver session API
# tm/cap/capraw below are driver-neutral adapters defined in libdriver.sh.
cap() { tm -- capture-pane -pt "$SESSION" 2>/dev/null || true; }
capraw() { tm -- capture-pane -pet "$SESSION" 2>/dev/null || true; }
left() { cap | cut -c1-$((COLS / 2 - 1)); }
right() { cap | cut -c$((COLS / 2 + 1))-; }
send() { tm -- send-keys -t "$SESSION" "$@"; }
sendl() { tm -- send-keys -t "$SESSION" -l "$@"; }

host_pid() { pgrep -x tui2 -a 2>/dev/null | awk -v p="$WORK/tui2" 'index($0, p) { print $1; exit }'; }
shell_pid() { pgrep -x tui2-shell -a 2>/dev/null | awk -v p="$WORK/tui2-shell" 'index($0, p) { print $1; exit }'; }

send_bytes() { # send_bytes <raw string>
  local hex i
  hex=$(printf '%s' "$1" | xxd -p -c 100000)
  local -a bytes=()
  for ((i = 0; i < ${#hex}; i += 2)); do bytes+=("${hex:i:2}"); done
  tm -- send-keys -t "$SESSION" -H "${bytes[@]}"
}

mouse() { # mouse <code> <x> <y> <M|m>
  send_bytes "$(printf '\x1b[<%d;%d;%d%s' "$1" "$2" "$3" "$4")"
}
mclick() { mouse 0 "$1" "$2" M; sleep 0.05; mouse 0 "$1" "$2" m; }
wheelup() { mouse 64 "$1" "$2" M; }

paste_text() { # bracketed paste of literal text
  send_bytes $'\x1b[200~'
  send_bytes "$1"
  send_bytes $'\x1b[201~'
}
paste_file() { # bracketed paste of a file, in 4 KiB byte chunks
  send_bytes $'\x1b[200~'
  while IFS= read -r line; do
    local -a bytes=()
    local i
    for ((i = 0; i < ${#line}; i += 2)); do bytes+=("${line:i:2}"); done
    tm -- send-keys -t "$SESSION" -H "${bytes[@]}"
  done < <(xxd -p -c 4096 "$1")
  send_bytes $'\x1b[201~'
}

waitgrep() { # waitgrep <secs> <regex>
  local deadline=$((SECONDS + $1))
  while ((SECONDS < deadline)); do
    if cap | grep -Eq "$2"; then return 0; fi
    sleep 0.2
  done
  return 1
}
must() { # must <secs> <desc> <regex>
  if waitgrep "$1" "$3"; then ok "$2"; else bad "$2"; fi
}
must_side() { # must_side <secs> <left|right> <desc> <regex>
  local deadline=$((SECONDS + $1))
  while ((SECONDS < deadline)); do
    if [ "$2" = left ] && left | grep -Eq "$4"; then ok "$3"; return 0; fi
    if [ "$2" = right ] && right | grep -Eq "$4"; then ok "$3"; return 0; fi
    sleep 0.2
  done
  bad "$3"
}

tm kill-server 2>/dev/null || true
tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo EXIT:\$?; exec bash'"
tm set-option -g set-clipboard on
tm set-option -g history-limit 5000
sleep 0.5

# ============================================================ §1 cold start
echo
echo "== §1 cold start =="
must 10 "cold start opens the picker" 'select a terminal'
must 3 "picker lists + New terminal" 'New terminal'
send Enter
must 8 "enter creates and binds term-1" 'term-1'
sendl 'echo SMOKE-OK'
send Enter
must 8 "typing reaches the focused PTY" 'SMOKE-OK'

# ============================================ §13.2 external resize follows
echo
echo "== §13.2 external resize (SIGWINCH) =="
sendl 'echo RS-A $(stty size)'
send Enter
must 5 "inner PTY starts at the full pane size (32 118)" 'RS-A 32 118'
tm resize-window -t "$SESSION" -x 100 -y 30
sleep 0.6
sendl 'echo RS-B $(stty size)'
send Enter
must 8 "tmux resize-window makes the inner PTY follow (26 98)" 'RS-B 26 98'
tm resize-window -t "$SESSION" -x "$COLS" -y "$ROWS"
sleep 0.6
sendl 'echo RS-C $(stty size)'
send Enter
must 8 "resizing back restores the inner PTY (32 118)" 'RS-C 32 118'

# ========================================== §1b slot title bar buttons
echo
echo "== §1b slot title bar action buttons =="
# The 1-row title bar is program-drawn (pos overlay): title on the left, the
# ⟳ ⇔ ⇕ ✕ group at the right edge of the (sidebar-off, full-width) slot.
must 5 "slot title bar shows the recommended action icons" '󰑐   󰅖'

# ✕ = unbind-style close: the slot empties, the terminal stays in the picker.
mclick 120 2
close_ok=0
if waitgrep 5 'Ctrl-F 选择终端'; then
  send C-f
  if waitgrep 5 'select a terminal'; then
    send Enter
    if waitgrep 8 'bound term-1'; then close_ok=1; fi
  fi
fi
if ((close_ok)); then
  ok "title-bar ✕ unbinds; the picker rebinds the live terminal"
else
  bad "title-bar ✕ unbinds; the picker rebinds the live terminal"
fi

# ⟳ = restart: the exit badge lives in the program title row; clicking the
# warning-colored icon rebuilds the PTY under the same id.
sendl 'exit 7'
send Enter
revive_ok=0
if waitgrep 8 '\[exited 7\]'; then
  mclick 114 2
  sleep 0.5
  sendl 'echo REVIVE-OK'
  send Enter
  if waitgrep 8 'REVIVE-OK'; then revive_ok=1; fi
fi
if ((revive_ok)); then
  ok "title-bar ⟳ restarts the exited slot and it accepts input"
else
  bad "title-bar ⟳ restarts the exited slot and it accepts input"
fi

# ================================================== §2 split / focus / pane
echo
echo "== §2 split, focus, esc->PTY =="
send C-p
sleep 0.3
send '%'
must 8 "Ctrl-P then % shows a second empty slot" 'Ctrl-F 选择终端'
must 3 "footer switches to PANE" 'PANE'
send C-f
must 8 "Ctrl-F opens the picker from PANE" 'select a terminal'
send Down
send Enter
must 8 "picker binds the focused (right) empty slot to term-2" 'bound term-2'
must_side 3 right "right slot shows the term-2 terminal" 'term-2'
mclick 20 6
sendl 'echo LEFT-OK'
send Enter
must_side 8 left "click focuses the left slot (input lands there)" 'LEFT-OK'
sleep 0.4
if right | grep -Eq 'LEFT-OK'; then bad "click does not leave focus on the right slot"; else ok "click does not leave focus on the right slot"; fi
send C-p
sleep 0.3
send Tab
send Escape
sleep 0.3
sendl 'echo RIGHT-OK'
send Enter
must_side 8 right "Tab moves focus, Esc exits PANE, keys reach that PTY" 'RIGHT-OK'
mclick 20 6
sleep 0.3
mclick 95 6
sendl 'echo CLICK2-OK'
send Enter
must_side 8 right "clicking a terminal switches focus back" 'CLICK2-OK'

# ============================================== §2b pass-through and wheel
echo
echo "== §2b pass-through: claim, Ctrl-F forward, wheel =="
sendl 'seq 1 200; cat -v'
send Enter
sleep 0.8
send C-f
sleep 0.05
send C-f
must_side 5 right "Ctrl-F double-click forwards ^F to the PTY" '\^F'
send C-p
sleep 0.3
send C-p
sleep 0.3
sendl 'W'
sleep 0.5
must_side 5 right "unclaimed key W reaches the PTY" '\^FW'
if right | grep -Eq '\^P'; then bad "claim key Ctrl-P does not reach the PTY"; else ok "claim key Ctrl-P does not reach the PTY"; fi
# wheel with no mouse tracking: scrollback is the program's business
wheelup 95 6
must 5 "wheel without tracking goes to the program ([↑N])" '\[↑[0-9]+\]'
must 3 "scroll mode shows the recommended COPY hints" 'PGUP 󰁝 OLDER'
send Escape
sleep 0.4
send C-c
sleep 0.4
# wheel with mouse tracking on: passthrough to the PTY
sendl 'printf "\033[?1000h"; cat -v'
send Enter
sleep 0.8
wheelup 95 6
must_side 5 right "wheel with tracking+focused+wheel declares pass-through" '\^\[\[<64;'
send C-c
sleep 0.4
sendl 'printf "\033[?1000l"'
send Enter
sleep 0.5

# ============================================================== §13.13 paste
echo
echo "== paste: bracket + long chunking =="
paste_text $'echo PASTE-A\necho PASTE-B'
sleep 0.6
if cap | grep -Eq '│PASTE-A'; then bad "multi-line paste is not executed before enter"; else ok "multi-line paste is not executed before enter"; fi
send Enter
must_side 8 right "multi-line paste runs after enter (A)" '│PASTE-A'
must_side 3 right "multi-line paste runs after enter (B)" '│PASTE-B'

# long paste: > 64 KiB forces chunking; assert byte count and content order.
{
  for i in $(seq 1 900); do
    printf 'L%05d %s\n' "$i" "$(printf 'x%.0s' $(seq 1 70))"
  done
} >"$WORK/expected.bin"
expected_bytes=$(wc -c <"$WORK/expected.bin")
# Re-assert bracketed paste on, then spool the paste into a file: zsh turns
# DEC 2004 off while a foreground command runs, and only a bracketed paste is
# chunked by the host.
sendl "printf '\\033[?2004h'; cat > $WORK/paste-raw.bin"
send Enter
sleep 0.4
paste_file "$WORK/expected.bin"
sleep 0.6
# The trailing chunk marker sits on a canonical line of its own, so the first
# C-d flushes it and only the second one (empty line) is EOF for cat.
send C-d
sleep 0.3
send C-d
deadline=$((SECONDS + 10))
while ((SECONDS < deadline)); do
  size=$(wc -c <"$WORK/paste-raw.bin" 2>/dev/null || echo 0)
  ((size >= expected_bytes)) && break
  sleep 0.3
done
size=$(wc -c <"$WORK/paste-raw.bin" 2>/dev/null || echo 0)
# two chunks (ceil(expected/65536) = 2) add two 12-byte bracket wrappers.
chunks=$(((expected_bytes + 65535) / 65536))
want_bytes=$((expected_bytes + chunks * 12))
if ((size == want_bytes)); then
  ok "long paste chunked into $chunks blocks (bytes $size)"
else
  bad "long paste chunked into $chunks blocks (got $size, want $want_bytes)"
fi
got_md5=$(sed -e 's/\x1b\[20[01]~//g' "$WORK/paste-raw.bin" 2>/dev/null | md5sum | cut -d' ' -f1)
want_md5=$(md5sum <"$WORK/expected.bin" | cut -d' ' -f1)
if [ "$got_md5" = "$want_md5" ]; then
  ok "long paste keeps byte order after removing chunk markers"
else
  bad "long paste keeps byte order (md5 $got_md5 != $want_md5)"
fi

# ===================================================== §4 scrollback + copy
echo
echo "== §4 scrollback and copy =="
sendl 'seq 1 300'
send Enter
sleep 0.8
send PageUp
must 5 "PgUp shows the [↑N] badge" '\[↑[0-9]+\]'
must 3 "SCROLL footer hints appear" 'PGUP 󰁝 OLDER'
wheelup 95 6
sleep 0.5
send y
must 5 "y copy reports success" 'copied visible screen'
deadline=$((SECONDS + 5))
buffer=""
while ((SECONDS < deadline)); do
  buffer=$(tm -- show-buffer 2>/dev/null || true)
  [ -n "$buffer" ] && break
  sleep 0.2
done
if [ -z "$buffer" ]; then
  bad "y copy lands in the tmux clipboard (OSC 52)"
else
  if printf '%s\n' "$buffer" | awk 'BEGIN{ok=1; prev=""} { if ($0 !~ /^[0-9]+$/) ok=0; if (prev != "" && $0 != prev + 1) ok=0; prev=$0 } END{ exit ok?0:1 }'; then
    ok "y copy lands consecutive visible rows in the tmux clipboard"
  else
    bad "y copy buffer is not the visible numeric window: $(printf '%s' "$buffer" | head -2 | tr '\n' ',')"
  fi
fi
send Escape
sleep 0.5
if cap | grep -Eq '\[↑[0-9]+\]'; then bad "Esc returns to live (badge gone)"; else ok "Esc returns to live (badge gone)"; fi
must 3 "live footer is back" '󰌌 CTRL'

# ==================================== §3 exit badge, restart, unbind rebind
echo
echo "== §3 exit / restart / close=unbind =="
sendl 'exit 7'
send Enter
must 8 "exit shows the [exited N] badge" '\[exited 7\]'
send C-e
must 8 "Ctrl-E restarts the same terminal (badge gone)" 'term-2'
sleep 0.5
if cap | grep -Eq '\[exited'; then bad "Ctrl-E restart clears the exited badge"; else ok "Ctrl-E restart clears the exited badge"; fi
sendl 'echo RESTART-OK'
send Enter
must_side 8 right "restarted terminal accepts input" 'RESTART-OK'
send C-p
sleep 0.3
send x
sleep 0.6
send C-w
sleep 0.4
must 3 "x closes the focused slot (one slot left)" '1/1 term-1'
send C-w
sleep 0.3
if cap | grep -Eq 'Ctrl-F 选择终端'; then bad "closing a bound slot removes the pane"; else ok "closing a bound slot removes the pane"; fi
send C-f
must 5 "closed terminal is still offered by the picker" 'term-2'
send Escape
sleep 0.3
send C-f
sleep 0.3
send Down
send Enter
must 8 "picker rebinds the closed terminal (unbind semantics)" 'bound term-2'

# ======================================================== §3 crash recovery
echo
echo "== §3 crash recovery =="
old_pid=$(shell_pid)
if [ -z "$old_pid" ]; then
  bad "layout program is running before the crash test"
else
  kill -9 "$old_pid"
  sleep 0.1
  if cap | grep -Eq 'term-'; then ok "last good tree is kept right after kill -9"; else bad "last good tree is kept right after kill -9"; fi
  deadline=$((SECONDS + 8))
  new_pid=""
  while ((SECONDS < deadline)); do
    new_pid=$(shell_pid)
    if [ -n "$new_pid" ] && [ "$new_pid" != "$old_pid" ]; then break; fi
    sleep 0.2
  done
  if [ -n "$new_pid" ] && [ "$new_pid" != "$old_pid" ]; then
    ok "host restarts the program with a new epoch (pid $old_pid -> $new_pid)"
  else
    bad "host restarts the program with a new epoch"
  fi
  must 5 "restart rebuilds the default layout (tab 1)" '\[⎇ 1:1\]'
fi
sleep 0.4
sendl 'echo CRASH-OK'
send Enter
must 8 "terminal survived the crash and is usable" 'CRASH-OK'
# A confirmation that is open while the program dies must vanish with the new
# epoch and must never execute.
send C-q
must 5 "host confirmation is open before the second crash" 'Quit tui2\?'
old_pid=$(shell_pid)
if [ -n "$old_pid" ]; then
  kill -9 "$old_pid"
  deadline=$((SECONDS + 8))
  new_pid=""
  while ((SECONDS < deadline)); do
    new_pid=$(shell_pid)
    if [ -n "$new_pid" ] && [ "$new_pid" != "$old_pid" ]; then break; fi
    sleep 0.2
  done
  sleep 0.5
  if cap | grep -Eq 'Quit tui2\?'; then bad "new epoch clears the open confirmation"; else ok "new epoch clears the open confirmation"; fi
  if [ -n "$(host_pid)" ]; then ok "old quit intent did not execute (host still alive)"; else bad "old quit intent did not execute (host still alive)"; fi
else
  bad "layout program is running before the confirmation crash"
fi

# ==================================================== §10 tabs + §5 prompt
echo
echo "== §10 tabs / §5 prompt / help =="
send C-t
must 5 "Ctrl-T adds a tab (2/2)" '\[⎇ 2:2\]'
must 3 "new tab opens the picker" 'select a terminal'
send Escape
sleep 0.3
send C-p
sleep 0.3
send 1
sleep 0.4
must 3 "1..9 switches back to tab 1" '\[⎇ 1:1\]'
send 2
sleep 0.4
must 3 "2 switches to tab 2" '\[⎇ 2:2\]'
send 1
sleep 0.3
send ':'
must 5 ": opens the command prompt" 'command'
sendl 'help'
send Enter
must 5 "prompt executes help" 'pane mode'
send Escape
sleep 0.3
send C-p
sleep 0.3
send ':'
sleep 0.3
sendl 'quit'
send Enter
must 5 "quit triggers the host confirmation" 'requests quit\?'
send Escape
sleep 0.4
if cap | grep -Eq 'requests quit'; then bad "Esc denies quit and closes the confirmation"; else ok "Esc denies quit and closes the confirmation"; fi
send C-p
sleep 0.2
send Escape
sleep 0.2
sendl 'echo DENY-OK'
send Enter
must 8 "UI stays usable after a denied quit" 'DENY-OK'
send C-p
sleep 0.3
send '?'
must 5 "? opens the help overlay" 'NORMAL'
send Escape
sleep 0.3

# ======================================================== §9 divider drag
echo
echo "== §9 divider drag + PTY winsize =="
send C-p
sleep 0.3
send '%'
must 5 "split again for the drag test" 'Ctrl-F 选择终端'
send C-f
sleep 0.3
send Down
send Enter
must 5 "bind term-2 to the right slot" 'bound term-2'
sendl 'stty size'
send Enter
must 5 "right PTY starts at half width" '32 58'
mouse 0 60 10 M
sleep 0.1
mouse 32 50 10 M
sleep 0.1
mouse 32 40 10 M
sleep 0.1
mouse 32 30 10 M
sleep 0.1
mouse 0 30 10 m
sleep 0.5
sendl 'stty size'
send Enter
must 8 "drag divider and the PTY winsize follows (88)" '32 88'
# drag back
mouse 0 30 10 M
sleep 0.1
mouse 32 60 10 M
sleep 0.1
mouse 0 60 10 m
sleep 0.5
sendl 'stty size'
send Enter
must 8 "drag back restores the original width (58)" '32 58'

# ==================================================== §5 kill authorization
echo
echo "== §5 kill confirmation =="
send C-p
sleep 0.3
send ':'
sleep 0.3
sendl 'kill'
send Enter
must 5 "kill triggers the host confirmation" 'Allow terminal.kill\?'
send Escape
sleep 0.4
if cap | grep -Eq 'Allow terminal.kill'; then bad "Esc denies kill and closes the confirmation"; else ok "Esc denies kill and closes the confirmation"; fi
sendl 'echo KILL-DENY-OK'
send Enter
must 8 "UI stays usable after a denied kill" 'KILL-DENY-OK'
send C-p
sleep 0.3
send ':'
sleep 0.3
sendl 'kill'
send Enter
must 5 "kill confirmation opens again" 'Allow terminal.kill\?'
send Enter
must 5 "allow executes terminal.kill" 'killed term-2'
must 5 "killed slot shows the exited badge" '\[exited [0-9]+\]'
send C-e
must 5 "Ctrl-E recovers the killed terminal" 'term-2'
sleep 0.4
sendl 'echo KILL-RESTART-OK'
send Enter
must_side 8 right "terminal is usable after kill + restart" 'KILL-RESTART-OK'

# ============================================== §2/§9 " split + vertical drag
echo
echo "== §2/§9 vertical split + vertical divider drag =="
send C-p
sleep 0.3
send x
sleep 0.5
send '"'
must 5 "quote splits stacked and focuses the empty slot" 'Ctrl-F 选择终端'
send C-f
sleep 0.3
send Down
send Enter
must 5 "bind term-2 to the stacked slot" 'bound term-2'
sendl 'stty size'
send Enter
must 5 "stacked PTY is full width" '15 118'
mouse 0 60 18 M
sleep 0.1
mouse 32 60 12 M
sleep 0.1
mouse 32 60 10 M
sleep 0.1
mouse 0 60 10 m
sleep 0.5
sendl 'stty size'
send Enter
must 8 "vertical divider drag changes PTY rows" '23 118'

# ==================================== §2/§8 mouse: focus, picker, sidebar
echo
echo "== mouse: slot focus, picker click, sidebar click =="
# 1) Slot focus: the bottom slot (term-2) is focused; clicking the top slot
# content moves the focus highlight (the program title run) to term-1.
focusseq=$'\x1b[1m\x1b[38;2;240;171;252m'
mclick 60 4
sleep 0.6
if capraw | grep -qF "${focusseq}▎term-1"; then
  ok "clicking a non-focused slot focuses it (title highlight)"
else
  bad "clicking a non-focused slot focuses it (title highlight)"
fi
if capraw | grep -qF "${focusseq}▎term-2"; then
  bad "the previously focused slot loses the focus highlight"
else
  ok "the previously focused slot loses the focus highlight"
fi

# 2) Picker click: Ctrl-T opens a fresh tab and its picker; a click on the
# term-1 row binds it in one press (not just moving the selection).
send C-t
must 5 "Ctrl-T opens a fresh tab with the picker" 'select a terminal'
pick_row=$(cap | grep -n 'term-1.*local' | head -1 | cut -d: -f1)
if [ -n "$pick_row" ]; then
  mclick 45 "$pick_row"
  must 8 "clicking a picker row binds that terminal" 'bound term-1'
  sleep 0.5
  if capraw | grep -qF "${focusseq}▎term-1"; then
    ok "the click-bound terminal is focused in the slot"
  else
    bad "the click-bound terminal is focused in the slot"
  fi
else
  bad "picker lists term-1 for the click test"
  bad "clicking a picker row binds that terminal"
  bad "the click-bound terminal is focused in the slot"
fi

# 3) Sidebar click: Ctrl-W reveals the sidebar (the config keeps it hidden);
# its tab rows are clickable and switch the active tab back to 1.
send C-w
sleep 0.5
if cap | grep -Eq '^│.*1:1'; then
  ok "sidebar lists clickable tab rows"
  side_row=$(cap | grep -n '^│.*1:1' | head -1 | cut -d: -f1)
  mclick 10 "$side_row"
  must 5 "clicking a sidebar tab row switches the active tab" '\[⎇ 1:1\]'
  send C-w
  sleep 0.3
else
  bad "sidebar lists clickable tab rows"
  bad "clicking a sidebar tab row switches the active tab"
fi

# ========================================================== §13.5 quit
echo
echo "== §13.5 Ctrl-Q deny / allow =="
send C-q
must 5 "Ctrl-Q opens the host confirmation" 'Quit tui2\?'
send Escape
sleep 0.5
if cap | grep -Eq 'Quit tui2'; then bad "Esc denies quit and closes the box"; else ok "Esc denies quit and closes the box"; fi
sendl 'echo STILL-OK'
send Enter
must 8 "UI stays usable after denied Ctrl-Q" 'STILL-OK'
send C-q
must 5 "Ctrl-Q confirmation opens again" 'Quit tui2\?'
send Enter
must 8 "allow quits cleanly and restores the terminal" 'EXIT:0'
sleep 0.5
if { [ -n "$(host_pid)" ] || [ -n "$(shell_pid)" ]; }; then
  bad "no clients/tui/tui2-shell processes are left behind"
else
  ok "no clients/tui/tui2-shell processes are left behind"
fi

# ====================== Python layout program (protocol, no Go framework)
# Proves the decoupling contract end to end: the same host runs a layout
# program written in another language, launched via "-shell <command+args>".
echo
echo "== python shell: layout program with zero framework dependency =="
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2pyacc"
tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "bash -c '\"$WORK/tui2\" -shell \"$PY $ROOT/clients/tui/examples/python-shell/shell.py\"; echo PYEXIT:\$?; exec bash'"
sleep 0.8

must 10 "python shell: tab bar renders ([1:1])" '\[1:1\]'
must 3 "python shell: footer renders (PICKER while the picker is open)" 'PICKER'
must 8 "python shell: cold start opens the picker" 'select a terminal'
send Enter
must 8 "python shell: enter creates and binds term-1" 'bound term-1'
must 3 "python shell: footer returns to NORMAL after binding" 'NORMAL'
sendl 'echo PY-BIND-OK'
send Enter
must 8 "python shell: typing reaches the focused PTY" 'PY-BIND-OK'

# x unbinds the only slot; the picker then re-attaches the existing terminal.
send C-p
sleep 0.3
send x
sleep 0.4
send C-f
sleep 0.3
send Enter
must 8 "python shell: picker re-attaches an existing terminal" 'bound term-1'
sendl 'echo PY-REBIND-OK'
send Enter
must 8 "python shell: re-bound existing terminal accepts input" 'PY-REBIND-OK'

# Ctrl-P + % split, picker creates term-2 in the new slot, x closes it.
send C-p
sleep 0.3
send '%'
must 8 "python shell: Ctrl-P then % splits an empty slot" 'Ctrl-F 选择终端'
send C-f
sleep 0.3
send Down
send Enter
must 8 "python shell: picker creates term-2 in the split" 'bound term-2'
send C-p
sleep 0.3
send x
sleep 0.4
must 5 "python shell: x closes the focused slot (slot 1/1)" 'slot 1/1'

# Ctrl-F + enter is the single-key path of the picker.
send C-f
sleep 0.3
send Enter
must 8 "python shell: Ctrl-F picker binds with a single key" 'bound term-1'

send C-q
must 5 "python shell: Ctrl-Q opens the host confirmation" 'Quit tui2\?'
send Enter
must 8 "python shell: quitting restores the terminal" 'PYEXIT:0'

# ================= legacy.py: pixel replica of the legacy default TUI
# Runs the pure-stdlib replica of shell/main.go, diffs real tmux captures
# against the old-formula golden files (parity_test.py) and exercises the
# replica interactions end to end.
echo
echo "== legacy shell: pixel replica (shell/main.go) =="
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2legacy"
tm new-session -d -s "$SESSION" -x "$COLS" -y 32 \
  "bash -c '\"$WORK/tui2\" -shell \"$PY $ROOT/clients/tui/examples/python-shell/legacy.py\"; echo LEGACYEXIT:\$?; exec bash'"
sleep 0.9

GOLDEN_DIR="$ROOT/clients/tui/examples/python-shell/golden"
golden_match() { # golden_match <golden-name> <description> (retries for redraws)
  local name="$1" desc="$2" got="$WORK/got-$1.txt" deadline=$((SECONDS + 6))
  while :; do
    cap | sed -e 's/[[:space:]]*$//' | sed -n '1,32p' >"$got"
    if diff -q "$GOLDEN_DIR/${name}_120x32.txt" "$got" >/dev/null 2>&1; then
      ok "$desc"
      return
    fi
    ((SECONDS >= deadline)) && break
    sleep 0.3
  done
  bad "$desc"
}
cursor_pos() { tm -- display-message -p -t "$SESSION" '#{cursor_x},#{cursor_y}' 2>/dev/null || true; }

if "$PY" "$ROOT/clients/tui/examples/python-shell/parity_test.py" >"$WORK/legacy-parity.log" 2>&1; then
  ok "legacy: offline per-cell parity (14 scenarios x 2 viewports + recommended preset)"
else
  bad "legacy: offline per-cell parity (see $WORK/legacy-parity.log)"
fi

must 10 "legacy: layout program renders the legacy tab bar" 'WS main  ▎1 main ×  \+'
golden_match cold_start_picker "legacy: cold-start picker screen matches the golden line by line"
if cap | sed -e 's/[[:space:]]*$//' | tail -1 | grep -Eq 'ws:main tabs:1 panes:1$'; then
  ok "legacy: footer right side is right-aligned"
else
  bad "legacy: footer right side is right-aligned"
fi
if cap | grep -Eq '^┌ status .*┐┌ ▎ unconnected '; then
  ok "legacy: sidebar status box and pane border share one row"
else
  bad "legacy: sidebar status box and pane border share one row"
fi

send Escape
sleep 0.3
golden_match single_pane "legacy: static single-pane screen matches the golden line by line"

send C-p
sleep 0.2
send '%'
sleep 0.4
golden_match split_row "legacy: horizontal split layout/borders match the golden line by line"
must 3 "legacy: split increments the sidebar pane count" 'panes     2'

send C-f
sleep 0.3
must 3 "legacy: picker title and selected row" '┌ Terminal Picker'
must 3 "legacy: picker offers + New terminal" '▸ \+ New terminal'
must 3 "legacy: picker hint line" 'enter/click attach · esc close'
send Escape
sleep 0.3

send ':'
sleep 0.3
if [ "$(cursor_pos)" = "35,8" ]; then
  ok "legacy: prompt cursor is re-homed at (35,8)"
else
  bad "legacy: prompt cursor is re-homed at (35,8) (got $(cursor_pos))"
fi
sendl 'he'
sleep 0.3
if [ "$(cursor_pos)" = "37,14" ]; then
  ok "legacy: prompt cursor follows the filtered input"
else
  bad "legacy: prompt cursor follows the filtered input (got $(cursor_pos))"
fi
must 3 "legacy: prompt filters commands (help)" '▸ help'
send Escape
sleep 0.3

# Bind a real terminal in the focused (right) slot, then exercise the badge.
send C-f
sleep 0.3
send Down
send Enter
must 8 "legacy: picker creates and binds term-1" 'bound term-1'
send Escape
sleep 0.3
must 5 "legacy: focused terminal pane has the focus marker" '┌ ▎term-1'
# Wait for the freshly created PTY before driving it (the marker can be up
# before the shell has started).
sendl 'echo LEGACY-READY'
send Enter
must 8 "legacy: bound terminal accepts input" 'LEGACY-READY'
sendl 'exit 7'
send Enter
must 8 "legacy: exit badge shows [exited 7]" '\[exited 7\]'
send C-e
sleep 0.6
if cap | grep -Eq '\[exited'; then
  bad "legacy: Ctrl-E restarts the exited terminal (badge gone)"
else
  ok "legacy: Ctrl-E restarts the exited terminal (badge gone)"
fi
send Escape
sleep 0.3
sendl 'echo LEGACY-REVIVE-OK'
send Enter
must 8 "legacy: restarted terminal accepts input" 'LEGACY-REVIVE-OK'
sendl 'exit 0'
send Enter
must 8 "legacy: zero exit code shows the bare [exited] badge" '\[exited\]'
if cap | grep -Eq '\[exited 0\]'; then bad "legacy: zero exit code must not print [exited 0]"; else ok "legacy: zero exit code must not print [exited 0]"; fi

send C-e
sleep 0.5
sendl 'seq 1 300'
send Enter
sleep 0.8
send C-p
sleep 0.2
send PageUp
must 5 "legacy: PgUp shows the [↑10] scrollback badge" '\[↑10\]'
must 3 "legacy: scroll footer hints switch to SCROLL" 'PgUp/PgDn\] SCROLL \+10'
send 'y'
must 5 "legacy: y copy reports success in the toast" 'copied visible screen'
send Escape
sleep 0.4
if cap | grep -Eq '\[↑[0-9]+\]'; then bad "legacy: Esc returns to live (scroll badge gone)"; else ok "legacy: Esc returns to live (scroll badge gone)"; fi

send C-q
must 5 "legacy: Ctrl-Q opens the host confirmation" 'Quit tui2\?'
send Enter
must 8 "legacy: quitting restores the terminal" 'LEGACYEXIT:0'

# ================= v3ui.py: pixel replica of the legacy v3 TUI (surface)
# Runs the card/header/footer chrome replica with the 1.txt demo state:
# golden chrome rows, pane focus/click, split + collapse hint, footer scenes,
# ⇧V (ctrl-shift-v) delivery and the header create button.
echo
echo "== v3 shell: legacy v3 chrome replica =="
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2v3"
tm new-session -d -s "$SESSION" -x "$COLS" -y 32 \
  "bash -c '\"$WORK/tui2\" -shell \"$PY $ROOT/clients/tui/examples/python-shell/v3ui.py --demo\"; echo V3EXIT:\$?; exec bash'"
sleep 0.9

if "$PY" "$ROOT/clients/tui/examples/python-shell/v3_parity_test.py" >"$WORK/v3-parity.log" 2>&1; then
  ok "v3: offline formula parity (2 viewports + 1.txt capture oracle + scenes)"
else
  bad "v3: offline formula parity (see $WORK/v3-parity.log)"
fi

v3_line_match() { # v3_line_match <line> <description>
  local n="$1" desc="$2" got="$WORK/v3-line-$1.txt" golden deadline=$((SECONDS + 6))
  golden=$(sed -n "${n}p" "$GOLDEN_DIR/v3_ui_120x32.txt")
  while :; do
    cap | sed -e 's/[[:space:]]*$//' | sed -n "${n}p" >"$got"
    if [ "$(cat "$got")" = "$golden" ]; then ok "$desc"; return; fi
    ((SECONDS >= deadline)) && break
    sleep 0.3
  done
  bad "$desc"
}

v3_line_match 1 "v3: header chip row matches the golden"
v3_line_match 2 "v3: window frame + title + action group match the golden"
v3_line_match 32 "v3: bottom status bar matches the golden"

# Pane focus: clicking a content area moves focus and the card title.
mclick 80 5
must 5 "v3: clicking the right pane focuses it (title opencode@hs)" 'opencode@hs'
mclick 20 5
must 5 "v3: clicking the left pane focuses it (title anytty-surface@hs)" 'anytty-surface@hs'

# Ctrl-P + % splits the window and the new pane shows the collapse hint.
# The demo pane already paints one hint line, so wait until the split's own
# hint line is on screen (count 2) before clicking: the host resolves the
# mouse hit against its last committed view, and clicking before the split
# is rendered would resolve against the unsplit pane.
send C-p
sleep 0.3
send '%'
v3_hints=0
deadline=$((SECONDS + 6))
while ((SECONDS < deadline)); do
  v3_hints=$(cap | grep -c 'Click to collapse' || true)
  ((v3_hints >= 2)) && break
  sleep 0.3
done
if [ "$v3_hints" -ge 2 ]; then ok "v3: Ctrl-P % adds a pane with the Click to collapse hint"; else bad "v3: Ctrl-P % adds a pane with the Click to collapse hint ($v3_hints hint lines)"; fi
must 3 "v3: PANE footer group shows the v3 recommended keys" 'CTRL\+D.*VSPLIT'
# Click the first (gutter) line of the two-line hint: both rendered hint
# lines carry the same collapse target, so the click must toggle the pane.
mclick 34 3
v3_collapsed=0
deadline=$((SECONDS + 6))
while ((SECONDS < deadline)); do
  v3_hints=$(cap | grep -c 'Click to collapse' || true)
  if [ "$v3_hints" -le 1 ]; then v3_collapsed=1; break; fi
  sleep 0.3
done
if [ "$v3_collapsed" -eq 1 ]; then ok "v3: clicking the hint collapses the pane"; else bad "v3: clicking the hint collapses the pane ($v3_hints hint lines)"; fi
send z
v3_hints=0
deadline=$((SECONDS + 6))
while ((SECONDS < deadline)); do
  v3_hints=$(cap | grep -c 'Click to collapse' || true)
  ((v3_hints >= 2)) && break
  sleep 0.3
done
if [ "$v3_hints" -ge 2 ]; then ok "v3: z expands the collapsed pane again"; else bad "v3: z expands the collapsed pane again ($v3_hints hint lines)"; fi
send Escape
sleep 0.3
must 3 "v3: Esc returns to the live footer" '󰌌 CTRL'

# ⇧V: the enhanced-keyboard name must reach the program (M3 key gap).
send_bytes $'\x1b[118;6u'
must 5 "v3: ctrl-shift-v (CSI-u) reaches the program" 'paste requested'
send Escape
sleep 0.2

# Header create button adds a tab.
mclick 53 1
must 5 "v3: header create button adds a tab (T3)" 'T3'
send C-q
must 5 "v3: Ctrl-Q opens the host confirmation" 'Quit tui2\?'
send Enter
must 8 "v3: quitting restores the terminal" 'V3EXIT:0'

# ============ v3ui.py real panes/floats: split layout, PTY sizes, floats
# Runs the same replica without --demo so every pane is a real PTY. Verifies
# the geometry end to end: stacked/side-by-side splits with draggable
# dividers, per-pane PTY winsize, input isolation, floating windows
# (new/bind/drag/collapse/close/focus) and external resize following.
echo
echo "== v3 shell: real split panes and floating windows =="
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2v3pty"
tm new-session -d -s "$SESSION" -x 120 -y 40 \
  "bash -c '\"$WORK/tui2\" -shell \"$PY $ROOT/clients/tui/examples/python-shell/v3ui.py\"; echo V3PTYEXIT:\$?; exec bash'"
sleep 1.0

v3drag() { # v3drag <x1> <y1> <x2> <y2> (1-based cells)
  mouse 0 "$1" "$2" M; sleep 0.15
  mouse 32 "$3" "$4" M; sleep 0.15
  mouse 0 "$3" "$4" m; sleep 0.4
}
v3top() { cap | sed -n '3,29p'; }
v3bot() { cap | sed -n '31,38p'; }

must 10 "v3pty: cold start picker offers a new terminal" 'New terminal'
send Enter
must 8 "v3pty: term-1 binds at the full content rect" 'term-1@local'
sendl 'echo V1 $(stty size)'
send Enter
must 6 "v3pty: single pane PTY is 36x118" 'V1 36 118'

# stacked split (Ctrl-E = old panel.split_down): [17, 18] + divider
send C-p; sleep 0.3
send C-e
must 8 "v3pty: Ctrl-E stacks panes and auto-creates term-2" 'term-2@local'
send Escape; sleep 0.3
sendl 'echo S-BOT $(stty size)'
send Enter
must 6 "v3pty: bottom pane PTY follows the stack (16x118)" 'S-BOT 16 118'
send C-p; sleep 0.3
send h; sleep 0.2
send Escape; sleep 0.3
sendl 'echo S-TOP $(stty size)'
send Enter
must 6 "v3pty: top pane PTY follows the stack (17x118)" 'S-TOP 17 118'

# divider drag: divider is 1-based row 21; drop at row 30 -> 26/7 split
v3drag 60 21 60 30
sendl 'echo D-TOP $(stty size)'
send Enter
must 6 "v3pty: divider drag resizes the top PTY (26x118)" 'D-TOP 26 118'
send C-p; sleep 0.3
send l; sleep 0.2
send Escape; sleep 0.3
sendl 'echo D-BOT $(stty size)'
send Enter
must 6 "v3pty: divider drag resizes the bottom PTY (7x118)" 'D-BOT 7 118'

# input isolation across the split
sendl 'echo XBOT-MARK'
send Enter
sleep 0.4
if v3top | grep -q 'XBOT-MARK'; then bad "v3pty: bottom input does not leak into the top pane"; else ok "v3pty: bottom input does not leak into the top pane"; fi
send C-p; sleep 0.3
send h; sleep 0.2
send Escape; sleep 0.3
sendl 'echo XTOP-MARK'
send Enter
sleep 0.4
if v3bot | grep -q 'XTOP-MARK'; then bad "v3pty: top input does not leak into the bottom pane"; else ok "v3pty: top input does not leak into the bottom pane"; fi

# side-by-side split (Ctrl-D = old panel.split_right): [58, 59] + divider
send C-p; sleep 0.3
send x; sleep 0.5
send C-p; sleep 0.3
send C-d
must 8 "v3pty: Ctrl-D splits side by side and auto-creates a pane" 'term-3@local'
send Escape; sleep 0.3
sendl 'echo R-RIGHT $(stty size)'
send Enter
must 6 "v3pty: right pane PTY follows the row split (36x57)" 'R-RIGHT 36 57'
send C-p; sleep 0.3
send h; sleep 0.2
send Escape; sleep 0.3
sendl 'echo R-LEFT $(stty size)'
send Enter
must 6 "v3pty: left pane PTY follows the row split (36x58)" 'R-LEFT 36 58'
# row divider drag: divider is 1-based column 61; drop at 80 -> 77/38
v3drag 61 10 80 10
sendl 'echo RD-LEFT $(stty size)'
send Enter
must 6 "v3pty: row divider drag resizes the left PTY (36x77)" 'RD-LEFT 36 77'
send C-p; sleep 0.3
send l; sleep 0.2
send Escape; sleep 0.3
sendl 'echo RD-RIGHT $(stty size)'
send Enter
must 6 "v3pty: row divider drag resizes the right PTY (36x38)" 'RD-RIGHT 36 38'

# floating window: new + auto terminal + PTY size
send C-o; sleep 0.3
send n
must 10 "v3pty: Ctrl-O n creates a floating window and binds a terminal" 'term-4@local'
must 5 "v3pty: floating chrome shows the action group" '◎.*▾.*󰁌.*󰅖'
must 5 "v3pty: footer float counter increments" '󰹙 1'
sendl 'echo FLT $(stty size)'
send Enter
must 6 "v3pty: floating PTY equals its content rect (28x94)" 'FLT 28 94'

# focus switching between the float and the main pane
mclick 5 10
sendl 'echo MAIN2 $(stty size)'
send Enter
must 6 "v3pty: clicking the main pane focuses its PTY (36x77)" 'MAIN2 36 77'
mclick 60 20
sendl 'echo FLT2 $(stty size)'
send Enter
must 6 "v3pty: clicking the float focuses its PTY again (28x94)" 'FLT2 28 94'

# float drag (title row implicit capture)
v3_before=$(cap | grep -n 'term-4@local' | head -1 | cut -d: -f1)
v3drag 60 6 70 3
v3_after=$(cap | grep -n 'term-4@local' | head -1 | cut -d: -f1)
if [ -n "$v3_after" ] && [ "$v3_after" != "$v3_before" ]; then
  ok "v3pty: dragging the title moves the floating window"
else
  bad "v3pty: dragging the title moves the floating window (before=$v3_before after=$v3_after)"
fi

# float collapse keeps only the title row, expand restores the content
send C-o; sleep 0.3
send z; sleep 0.5
if cap | grep -q 'term-4@local'; then ok "v3pty: collapsed float keeps its title row"; else bad "v3pty: collapsed float keeps its title row"; fi
if cap | grep -q 'FLT2 28 94'; then bad "v3pty: collapsed float hides its content"; else ok "v3pty: collapsed float hides its content"; fi
send z; sleep 0.5
send Escape; sleep 0.3
must 6 "v3pty: expanding the float restores its content" 'FLT2 28 94'

# external resize follows in the float and in every split pane
tm resize-window -t "$SESSION" -x 100 -y 30
sleep 0.8
sendl 'echo RFLT $(stty size)'
send Enter
must 8 "v3pty: external resize makes the floating PTY follow (26x94)" 'RFLT 26 94'
send C-o; sleep 0.3
send x; sleep 0.5
if cap | grep -q 'term-4@local'; then bad "v3pty: closing the float removes its window"; else ok "v3pty: closing the float removes its window"; fi
# The row split ratio was dragged to 77/115, so the 100-col resize keeps
# that ~2:1 ratio (95 = 64 + 31).
sendl 'echo RS-LEFT $(stty size)'
send Enter
must 8 "v3pty: external resize follows in the left PTY (26x64)" 'RS-LEFT 26 64'
send C-p; sleep 0.3
send l; sleep 0.2
send Escape; sleep 0.3
sendl 'echo RS-RIGHT $(stty size)'
send Enter
must 8 "v3pty: external resize follows in the right PTY (26x31)" 'RS-RIGHT 26 31'

send C-q
must 5 "v3pty: Ctrl-Q opens the host confirmation" 'Quit tui2\?'
send Enter
must 8 "v3pty: quitting restores the terminal" 'V3PTYEXIT:0'

# ============ v3ui.py recursive split tree: left 1 / right 2 layout =========
# Splitting only ever partitions the focused leaf: after a row split the right
# leaf is split column-wise, giving three independent cards. Verified leaf by
# leaf: PTY = card rect - 2, the nested divider drag only touches its own
# Split, closing promotes the sibling, external resize keeps every ratio, and
# the footer key groups carry their old per-key colors.
echo
echo "== v3 shell: recursive split tree (left 1 / right 2) =="
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2v3tree"
tm new-session -d -s "$SESSION" -x 120 -y 40 \
  "bash -c '\"$WORK/tui2\" -shell \"$PY $ROOT/clients/tui/examples/python-shell/v3ui.py\"; echo V3TREEEXIT:\$?; exec bash'"
sleep 1.0

must 10 "v3tree: cold start picker offers a terminal" 'New terminal'
send Enter
must 8 "v3tree: term-1 binds to the single leaf" 'term-1@local'
# row split: left | right (right leaf focused)
send C-p; sleep 0.3; send '%'; sleep 1.0
must 8 "v3tree: row split adds the right card (term-2)" 'term-2@local'
send Escape; sleep 0.3
# split only the focused right leaf: right becomes top/bottom (term-3)
send C-p; sleep 0.3; send '"'; sleep 1.0
must 8 "v3tree: splitting the right leaf adds a third card (term-3)" 'term-3@local'
send Escape; sleep 0.3

# three leaves, three independent PTY sizes
sendl 'echo T-RB $(stty size)'; send Enter
must 6 "v3tree: right-bottom leaf PTY is 16x57 (card 59x18)" 'T-RB 16 57'
send C-p; sleep 0.3; send h; sleep 0.2; send Escape; sleep 0.3
sendl 'echo T-RT $(stty size)'; send Enter
must 6 "v3tree: right-top leaf PTY is 17x57 (card 59x19)" 'T-RT 17 57'
send C-p; sleep 0.3; send h; sleep 0.2; send Escape; sleep 0.3
sendl 'echo T-LF $(stty size)'; send Enter
must 6 "v3tree: left leaf PTY is 36x58 (card 60x38)" 'T-LF 36 58'

# drag only the right nested divider (1-based row 21 -> 27): 23/10 right split
v3drag 91 21 91 27
send C-p; sleep 0.3; send l; sleep 0.2; send Escape; sleep 0.3
sendl 'echo R-T $(stty size)'; send Enter
must 6 "v3tree: the dragged right-top leaf is 23x57" 'R-T 23 57'
send C-p; sleep 0.3; send l; sleep 0.2; send Escape; sleep 0.3
sendl 'echo R-B $(stty size)'; send Enter
must 6 "v3tree: the dragged right-bottom leaf is 10x57" 'R-B 10 57'
send C-p; sleep 0.3; send h; sleep 0.2; send h; sleep 0.2; send Escape; sleep 0.3
sendl 'echo R-L $(stty size)'; send Enter
must 6 "v3tree: the drag leaves the left leaf untouched (36x58)" 'R-L 36 58'

# footer key group colors: NORMAL >=7 distinct truecolors with the old codes
v3tree_colors() { capraw | tail -1 | grep -o '38;2;[0-9]*;[0-9]*;[0-9]*' | sort -u; }
v3tree_color_count=$(v3tree_colors | wc -l | tr -d ' ')
if [ "$v3tree_color_count" -ge 7 ]; then
  ok "v3tree: NORMAL footer shows >=7 distinct key colors ($v3tree_color_count)"
else
  bad "v3tree: NORMAL footer shows >=7 distinct key colors ($v3tree_color_count)"
fi
v3tree_want_colors='38;2;240;171;252 38;2;253;230;138 38;2;188;189;252 38;2;125;211;252 38;2;134;239;172 38;2;252;154;135 38;2;245;192;212'
v3tree_missing=0
for code in $v3tree_want_colors; do
  v3tree_colors | grep -q "^$code$" || v3tree_missing=$((v3tree_missing + 1))
done
if [ "$v3tree_missing" -eq 0 ]; then
  ok "v3tree: NORMAL footer P/R/O/T/W/F/G key colors match the old tokens"
else
  bad "v3tree: NORMAL footer key colors missing $v3tree_missing of 7"
fi
# PANE scene: badge PANE + X/SPLIT/H/L/Q groups => >=5 distinct colors
send C-p; sleep 0.4
v3tree_pane_count=$(v3tree_colors | wc -l | tr -d ' ')
if [ "$v3tree_pane_count" -ge 5 ] && v3tree_colors | grep -q '^38;2;131;229;200$'; then
  ok "v3tree: PANE footer has >=5 key colors incl. VSPLIT copy ($v3tree_pane_count)"
else
  bad "v3tree: PANE footer colors ($v3tree_pane_count)"
fi
send Escape; sleep 0.3
# picker scene: badge uses footer-key-picker #fc9a87
send C-f; sleep 0.5
if v3tree_colors | grep -q '^38;2;252;154;135$'; then
  v3tree_picker_count=$(v3tree_colors | wc -l | tr -d ' ')
  if [ "$v3tree_picker_count" -ge 4 ]; then
    ok "v3tree: PICK footer badge uses picker color, >=4 key colors"
  else
    bad "v3tree: PICK footer key color count ($v3tree_picker_count)"
  fi
else
  bad "v3tree: PICK footer badge uses the picker color"
fi
send Escape; sleep 0.3

# external resize keeps every nested ratio; restore returns the same sizes
tm resize-window -t "$SESSION" -x 100 -y 30
sleep 0.8
sendl 'echo Z-L $(stty size)'; send Enter
must 6 "v3tree: resize keeps the left ratio (26x48)" 'Z-L 26 48'
send C-p; sleep 0.3; send l; sleep 0.2; send Escape; sleep 0.3
sendl 'echo Z-T $(stty size)'; send Enter
must 6 "v3tree: resize keeps the right-top ratio (16x47)" 'Z-T 16 47'
send C-p; sleep 0.3; send l; sleep 0.2; send Escape; sleep 0.3
sendl 'echo Z-B $(stty size)'; send Enter
must 6 "v3tree: resize keeps the right-bottom ratio (7x47)" 'Z-B 7 47'
tm resize-window -t "$SESSION" -x 120 -y 40
sleep 1.0
sendl 'echo ZB2 $(stty size)'; send Enter
must 6 "v3tree: restoring the window restores the dragged sizes (10x57)" 'ZB2 10 57'

# close the right-bottom leaf: its Split promotes the right-top sibling
v3tree_titles=$(cap | grep -o '┌─ □' | wc -l | tr -d ' ')
send C-p; sleep 0.3; send x; sleep 1.0
send Escape; sleep 0.5
v3tree_after=$(cap | grep -o '┌─ □' | wc -l | tr -d ' ')
if [ "$v3tree_titles" -eq 3 ] && [ "$v3tree_after" -eq 2 ]; then
  ok "v3tree: closing the right-bottom leaf promotes the sibling (3 cards -> 2)"
else
  bad "v3tree: closing the right-bottom leaf ($v3tree_titles -> $v3tree_after cards)"
fi
sendl 'echo WARMUP'; send Enter
must 5 "v3tree: the restored right leaf keeps the focus" 'WARMUP'
sendl 'echo C-R $(stty size)'; send Enter
must 6 "v3tree: the restored right leaf takes the full height (36x57)" 'C-R 36 57'

send C-q
must 5 "v3tree: Ctrl-Q opens the host confirmation" 'Quit tui2\?'
send Enter
must 8 "v3tree: quitting restores the terminal" 'V3TREEEXIT:0'

# ================= recommended profile: icons + command endpoints
# The default shell profile is the legacy coralline-candy recommended config:
# Nerd Font icons plus configured command endpoints (endpoint = the argv of a
# local PTY, the v1 stand-in for the legacy v3 endpoints.yaml routes).
echo
echo "== recommended profile: icons and command endpoints =="
cat >"$WORK/tui2-endpoint-config.json" <<'CFG'
{
  "sidebar": false,
  "gap": 1,
  "clock": { "enabled": false },
  "endpoints": [
    { "name": "remote", "kind": "command", "label": "fake-remote",
      "argv": ["sh", "-lc", "echo REMOTE-READY; exec cat"] }
  ]
}
CFG
cat >"$WORK/legacy-recommended.json" <<'CFG'
{ "preset": "recommended" }
CFG
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2recommended"
tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell --config $WORK/tui2-endpoint-config.json\"; echo EPEXIT:\$?; exec bash'"
sleep 0.9

# Icons of the recommended preset (Nerd Font PUA codepoints in the capture).
must 10 "recommended: workspace chip shows the 󰙅 folder icon" '󰙅 local'
must 3 "recommended: tab bar shows the 󰐕 new-tab icon" '󰐕'
must 3 "recommended: footer mode badge shows the 󰱼 picker icon" '󰱼 PICK'
must 3 "recommended: picker footer uses the old recommended group" '󰱼 PICK  ·  ↑/↓ SELECT ·  ENTER 󰋺 ATTACH'

# The picker lists the configured endpoint (name + connection icon) and a
# single Enter creates it: terminal.create{endpoint, argv, cwd, env}.
must 8 "recommended: picker lists the configured endpoint" 'fake-remote'
must 3 "recommended: endpoint row carries the 󰌷 connection icon" '󰌷 fake-remote'
send Enter
must 8 "recommended: selecting the endpoint binds remote:term-1" 'bound remote:term-1'
must 8 "recommended: fake endpoint command shows REMOTE-READY" 'REMOTE-READY'
sendl 'echo EP-INPUT-OK'
send Enter
must 8 "recommended: input reaches the endpoint PTY and echoes" 'EP-INPUT-OK'
must 3 "recommended: footer mode badge back to the 󰌌 live icon" '󰌌 CTRL'
must 3 "recommended: NORMAL footer uses the old recommended labels" '󰌌 CTRL  ·  P  PANE ·  T 󰓩 TAB ·  W 󰙅 WORKSPACE ·  F 󰱼 PICK'
must 3 "recommended: footer shows the 󰓩 tab icon" 'T 󰓩 TAB'
if cap | grep -Eq 'spacebar|split-h|split-v|layout_toggle|resize\.|floating|zoom'; then
  bad "recommended: footer has no raw key/action names"
  cap | grep -Eo 'spacebar|split-h|split-v|layout_toggle|resize\.|floating|zoom' | sort -u | head -3
else
  ok "recommended: footer has no raw key/action names"
fi
send C-p
sleep 0.3
must 3 "recommended: PANE footer uses the old recommended labels" 'X 󰅖 CLOSE ·  %  VSPLIT ·  "  HSPLIT ·  TAB 󰜴 FOCUS'
send Escape
sleep 0.3

# legacy.py with the recommended preset: same pixels, recommended icons.
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2legacyrec"
tm new-session -d -s "$SESSION" -x "$COLS" -y 32 \
  "bash -c '\"$WORK/tui2\" -shell \"$PY $ROOT/clients/tui/examples/python-shell/legacy.py --config $WORK/legacy-recommended.json\"; echo LEGACYRECEXIT:\$?; exec bash'"
sleep 0.9
must 10 "recommended: legacy.py preset renders the 󰙅 workspace icon" '󰙅 local'
must 3 "recommended: legacy.py preset renders the 󰐕 new-tab icon" '󰐕'
send Escape
sleep 0.5
must 3 "recommended: legacy.py preset footer uses the old NORMAL group" '󰌌 CTRL  ·  P  PANE ·  T 󰓩 TAB'
send C-q
must 5 "recommended: legacy.py preset session quits" 'Quit tui2\?'
send Enter
must 8 "recommended: legacy.py preset session exits cleanly" 'LEGACYRECEXIT:0'

# ============================================= daemon endpoint: local-unix
# M3 of the endpoint work: connect a real isolated dev daemon through the
# daemon protocol and exercise list/attach/input/resize/kill plus the
# coexistence of a local command endpoint and an offline endpoint notice.
echo
echo "== daemon endpoint: local-unix =="
DEV_SH="$ROOT/scripts/anytty-dev.sh"
DEV_BIN="${ANYTTY_DEV_ROOT:-$HOME/.local/share/anytty-dev}/bin/anytty-dev"
EP_DAEMON=0
EP_SOCK=""
if [[ -f "$DEV_SH" ]]; then
  if [[ ! -x "$DEV_BIN" ]]; then
    bash "$DEV_SH" build >"$WORK/dev-build.log" 2>&1 || true
  fi
  if [[ -x "$DEV_BIN" && -f "$DEV_SH" ]]; then
    DEV_ENV_OUT="$(bash "$DEV_SH" env 2>/dev/null || true)"
    EP_SOCK="$(printf '%s\n' "$DEV_ENV_OUT" | sed -n 's/^ANYTTY_DEV_SOCKET=//p')"
    # `daemon start` exits non-zero when the daemon is already running, which
    # is a success for an isolated dev environment.
    bash "$DEV_SH" daemon start >/dev/null 2>&1 || true
    if [[ -n "$EP_SOCK" && -S "$EP_SOCK" ]]; then
      EP_DAEMON=1
    fi
  fi
fi
if ((!EP_DAEMON)); then
  echo "SKIP  anytty-dev daemon unavailable; set ANYTTY_DEV_ROOT or run scripts/anytty-dev.sh build"
else
  epcli() { "$DEV_BIN" --socket "$EP_SOCK" "$@"; }
  EP_NAME="ep-daemon-$$"
  if epcli v3 new --name "$EP_NAME" -- sh >"$WORK/ep-create.log" 2>&1; then
    ok "daemon: cli v3 new creates a daemon terminal"
  else
    bad "daemon: cli v3 new creates a daemon terminal"
    sed -n '1,3p' "$WORK/ep-create.log"
  fi

  cat >"$WORK/tui2-daemon-config.json" <<CFG
{
  "sidebar": false,
  "gap": 1,
  "clock": { "enabled": false },
  "endpoints": [
    { "name": "localcmd", "kind": "command", "label": "local-cmd", "argv": ["sh"] },
    { "name": "dev", "kind": "daemon", "label": "dev-daemon", "socket": "$EP_SOCK" }
  ]
}
CFG
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2daemon"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell --config $WORK/tui2-daemon-config.json\"; echo DAEMONEXIT:\$?; exec bash'"
  sleep 0.8

  must 10 "daemon: cold start opens the picker" 'select a terminal'
  must 5 "daemon: picker groups the endpoint (group header)" '│  dev +│'
  if cap | grep -Eq '󰌷 dev-daemon +endpoint · daemon'; then
    ok "daemon: configured endpoint row carries the daemon label"
  else
    bad "daemon: configured endpoint row carries the daemon label"
  fi
  if cap | grep -Eq "$EP_NAME +dev · live"; then
    ok "daemon: picker lists the daemon terminal with its endpoint label"
  else
    bad "daemon: picker lists the daemon terminal with its endpoint label"
  fi

  # One click on the daemon terminal row attaches it (snapshot + live stream).
  ep_row=$(cap | grep -n "$EP_NAME" | head -1 | cut -d: -f1)
  if [ -n "$ep_row" ]; then
    mclick 45 "$ep_row"
  else
    send Down
    send Enter
  fi
  must 8 "daemon: click attaches the daemon terminal" "$EP_NAME"
  sendl 'echo EP-DAEMON-OK'
  send Enter
  must 8 "daemon: input reaches the daemon PTY and echoes" 'EP-DAEMON-OK'
  sendl 'stty size'
  send Enter
  must 5 "daemon: remote PTY starts at the full pane size (32 118)" '32 118'
  tm resize-window -t "$SESSION" -x 100 -y 30
  sleep 0.6
  sendl 'echo EP-RS $(stty size)'
  send Enter
  must 8 "daemon: tmux resize-window resizes the daemon PTY (26 98)" 'EP-RS 26 98'
  tm resize-window -t "$SESSION" -x "$COLS" -y "$ROWS"
  sleep 0.6
  sendl 'echo EP-RS2 $(stty size)'
  send Enter
  must 8 "daemon: resizing back restores the daemon PTY (32 118)" 'EP-RS2 32 118'

  # kill through the TUI confirmation: the slot shows the exit badge and the
  # daemon inventory reports the terminal as exited.
  send C-p
  sleep 0.3
  send ':'
  sleep 0.3
  sendl 'kill'
  send Enter
  must 5 "daemon: kill triggers the host confirmation" 'Allow terminal.kill\?'
  send Enter
  must 5 "daemon: killed daemon terminal shows the exit badge" '\[exited [0-9]+\]'
  cli_exited=0
  for _ in $(seq 1 25); do
    if epcli v3 ls 2>/dev/null | grep -Eq "$EP_NAME +.*exited"; then cli_exited=1; break; fi
    sleep 0.2
  done
  if ((cli_exited)); then
    ok "daemon: cli v3 ls reports the killed terminal as exited"
  else
    bad "daemon: cli v3 ls reports the killed terminal as exited"
    echo "DEBUG daemon ls:"; epcli v3 ls 2>&1 | sed -n '1,6p'
    echo "DEBUG slot:"; cap | sed -n 2p
    echo "DEBUG status:"; cap | tail -1
    echo "DEBUG daemon log:"; tail -6 "$XDG_STATE_HOME/anytty/anytty.log" 2>/dev/null
  fi

  # Ctrl-E restarts the killed daemon terminal from the daemon spec.
  send C-e
  sleep 1.5
  sendl 'echo EP-RESTART-OK'
  send Enter
  must 8 "daemon: Ctrl-E restarts the daemon terminal" 'EP-RESTART-OK'

  # Coexistence: the command endpoint and the daemon endpoint share one view.
  send C-p
  sleep 0.3
  send '%'
  must 5 "daemon: split for the coexistence test" 'Ctrl-F 选择终端'
  send C-f
  sleep 0.3
  must 5 "daemon: picker lists the local command endpoint" 'local-cmd'
  local_row=$(cap | grep -n 'local-cmd' | head -1 | cut -d: -f1)
  if [ -n "$local_row" ]; then
    mclick 45 "$local_row"
  else
    send Down
    send Enter
  fi
  must 8 "daemon: local command endpoint binds beside the daemon" 'bound localcmd'
  sendl 'echo LOCAL-OK'
  send Enter
  must_side 8 right "daemon: local command endpoint accepts input" 'LOCAL-OK'
  must_side 3 left "daemon: daemon terminal is still shown beside the local one" "$EP_NAME"
  mclick 20 6
  sendl 'echo EP-LOCAL-COEXIST'
  send Enter
  must_side 8 left "daemon: daemon terminal still accepts input in the split" 'EP-LOCAL-COEXIST'

  epcli v3 kill "$EP_NAME" >/dev/null 2>&1 || true
  epcli v3 remove "$EP_NAME" >/dev/null 2>&1 || true
fi

# Offline endpoint: the notice must surface and the endpoint must stay listed.
echo
echo "== daemon endpoint: offline and unsupported modes =="
if ((!EP_DAEMON)); then
  echo "SKIP  anytty-dev daemon unavailable"
else
  cat >"$WORK/tui2-offline-config.json" <<CFG
{
  "sidebar": false,
  "clock": { "enabled": false },
  "endpoints": [
    { "name": "dead", "kind": "daemon", "label": "dead-daemon", "socket": "$WORK/no-such-daemon.sock" }
  ]
}
CFG
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2offline"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell --config $WORK/tui2-offline-config.json\"; echo OFFLINEEXIT:\$?; exec bash'"
  sleep 0.8
  must 10 "offline: picker still lists the offline endpoint" 'dead-daemon.*endpoint · daemon'
  must 8 "offline: health transition emits a readable notice" 'endpoint dead offline'
  send Escape
  send C-q
  must 5 "offline: session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "offline: offline endpoint did not break shutdown" 'OFFLINEEXIT:0'

  cat >"$WORK/tui2-webrtc-config.json" <<CFG
{
  "sidebar": false,
  "clock": { "enabled": false },
  "endpoints": [
    { "name": "webrtc", "kind": "daemon", "label": "webrtc-daemon", "socket": "unused", "connect_mode": "direct-webrtc-tcp" }
  ]
}
CFG
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2webrtc"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell --config $WORK/tui2-webrtc-config.json\"; echo WEBRTCEXIT:\$?; exec bash'"
  sleep 0.8
  must 10 "offline: unsupported webrtc endpoint stays listed" 'webrtc-daemon.*endpoint · daemon'
  must 8 "offline: webrtc connect_mode reports a readable notice" 'signaling_addresses requires at least one value'
  send Escape
  send C-q
  must 5 "offline: webrtc session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "offline: webrtc endpoint did not break shutdown" 'WEBRTCEXIT:0'

  # tcp without an address is a config error, not a silent default.
  cat >"$WORK/tui2-tcp-noaddr.json" <<CFG
{
  "sidebar": false,
  "clock": { "enabled": false },
  "endpoints": [
    { "name": "tcp", "kind": "daemon", "label": "tcp-no-address", "connect_mode": "tcp" }
  ]
}
CFG
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2tcpnoaddr"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell --config $WORK/tui2-tcp-noaddr.json\"; echo TCPNOADDREXIT:\$?; exec bash'"
  sleep 0.8
  must 8 "offline: tcp without address reports a config error" 'address is required for connect_mode tcp'
  send C-q
  must 5 "offline: tcp-no-address session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "offline: tcp-no-address endpoint did not break shutdown" 'TCPNOADDREXIT:0'
fi

# ==================================== remote daemon: tcp / ssh tunnels (P0)
# One isolated second daemon stands in for a remote host. Two tunnels expose
# its unix socket: the tcp connect mode through a local TCP port (real
# `ssh -N -L 127.0.0.1:PORT:/remote/daemon.sock localhost` when ssh is
# available, otherwise the clients/tui/scripts/remote-bridge Go proxy) and the P0
# local-unix path through `ssh -N -L local.sock:/remote/daemon.sock`.
echo
echo "== daemon endpoint: remote via tcp / ssh tunnels =="
REMOTE_SOCK=""
REMOTE_XDG="$WORK/remote-xdg"
if ((!EP_DAEMON)); then
  echo "SKIP  anytty-dev daemon unavailable"
else
  REMOTE_SOCK="$WORK/remote-daemon.sock"
  mkdir -p "$REMOTE_XDG/config" "$REMOTE_XDG/state" "$REMOTE_XDG/run"
  REMOTE_ENV=(env XDG_CONFIG_HOME="$REMOTE_XDG/config" XDG_STATE_HOME="$REMOTE_XDG/state" XDG_RUNTIME_DIR="$REMOTE_XDG/run" ANYTTY_HISTORY_DISABLE=1 ANYTTY_DIRECT_SIGNALING_LISTEN=127.0.0.1:0 ANYTTY_DIRECT_ICE_TCP_LISTEN=127.0.0.1:0)
  "${REMOTE_ENV[@]}" "$DEV_BIN" --socket "$REMOTE_SOCK" --log-file "$WORK/remote-daemon.log" daemon start >/dev/null 2>&1 || true
  for _ in $(seq 1 50); do [[ -S "$REMOTE_SOCK" ]] && break; sleep 0.1; done
  remote_cli() { "${REMOTE_ENV[@]}" "$DEV_BIN" --socket "$REMOTE_SOCK" "$@"; }
  if [[ ! -S "$REMOTE_SOCK" ]]; then
    echo "SKIP  second isolated daemon did not start (see $WORK/remote-daemon.log)"
    REMOTE_SOCK=""
  fi
fi
if [[ -z "${REMOTE_SOCK:-}" ]]; then
  echo "SKIP  remote tunnel checks"
else
  ok "remote: second isolated daemon started (independent socket + XDG)"
  REMOTE_NAME="ep-remote-$$"
  if remote_cli v3 new --name "$REMOTE_NAME" -- sh >"$WORK/remote-create.log" 2>&1; then
    ok "remote: cli v3 new creates a terminal on the remote daemon"
  else
    bad "remote: cli v3 new creates a terminal on the remote daemon"
    sed -n '1,3p' "$WORK/remote-create.log"
  fi

  SSH_REMOTE=0
  if command -v ssh >/dev/null 2>&1 && ssh -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout=3 localhost true >/dev/null 2>&1; then
    SSH_REMOTE=1
  fi
  free_port() { "$PY" -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()'; }
  wait_port() { # wait_port <port>
    for _ in $(seq 1 50); do
      if "$PY" -c "import socket,sys; s=socket.socket(); s.settimeout(0.2); sys.exit(0 if s.connect_ex(('127.0.0.1',$1))==0 else 1)" 2>/dev/null; then
        return 0
      fi
      sleep 0.1
    done
    return 1
  }

  # ---- tcp connect mode: local TCP port -> remote daemon socket ----------
  TUNNEL_PID=""
  stop_tunnel() {
    if [[ -n "$TUNNEL_PID" ]]; then
      kill "$TUNNEL_PID" 2>/dev/null || true
      wait "$TUNNEL_PID" 2>/dev/null || true
      TUNNEL_PID=""
    fi
  }
  # The Go bridge is always built: it is the portable tunnel and the fallback
  # when ssh cannot rebind the original port for the reconnect leg.
  if (cd "$ROOT" && "$GO" build -o "$WORK/remote-bridge" ./clients/tui/scripts/remote-bridge) >"$WORK/bridge-build.log" 2>&1; then
    ok "remote: built the go tunnel proxy (portable ssh -L stand-in)"
  else
    bad "remote: build go tunnel proxy"
  fi
  TCP_PORT=$(free_port)
  ORIGINAL_PORT=$TCP_PORT
  if ((SSH_REMOTE)); then
    TUNNEL_KIND="ssh -L TCP->unix"
    ssh -N -o BatchMode=yes -o StrictHostKeyChecking=no -o ExitOnForwardFailure=yes \
      -L "127.0.0.1:$TCP_PORT:$REMOTE_SOCK" localhost >"$WORK/ssh-tcp.log" 2>&1 &
    TUNNEL_PID=$!
  else
    TUNNEL_KIND="go remote-bridge (ssh localhost unavailable)"
    "$WORK/remote-bridge" -listen "127.0.0.1:$TCP_PORT" -unix "$REMOTE_SOCK" >"$WORK/bridge.log" 2>&1 &
    TUNNEL_PID=$!
  fi
  echo "NOTE  tcp tunnel: $TUNNEL_KIND  127.0.0.1:$TCP_PORT -> $REMOTE_SOCK"
  if wait_port "$TCP_PORT"; then
    ok "remote: tcp tunnel accepts connections"
  else
    bad "remote: tcp tunnel accepts connections"
  fi

  cat >"$WORK/tui2-remote-config.json" <<CFG
{
  "sidebar": false,
  "gap": 1,
  "clock": { "enabled": false },
  "endpoints": [
    { "name": "remote", "kind": "daemon", "label": "remote-daemon", "connect_mode": "tcp", "address": "127.0.0.1:$TCP_PORT" },
    { "name": "sshcmd", "kind": "command", "label": "ssh-cmd", "argv": ["ssh", "localhost", "sh", "-lc", "echo SSH-CMD-OK; exec cat"] }
  ]
}
CFG
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2remote"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell --config $WORK/tui2-remote-config.json\"; echo REMOTEEXIT:\$?; exec bash'"
  sleep 0.9

  must 10 "remote: cold start opens the picker" 'select a terminal'
  must 5 "remote: picker groups the tcp endpoint" '│  remote +│'
  if cap | grep -Eq '󰌷 remote-daemon +endpoint · daemon tcp'; then
    ok "remote: endpoint row labels the tcp connect mode"
  else
    bad "remote: endpoint row labels the tcp connect mode"
  fi
  if cap | grep -Eq "$REMOTE_NAME +remote · live"; then
    ok "remote: picker lists the remote terminal with its endpoint label"
  else
    bad "remote: picker lists the remote terminal with its endpoint label"
  fi
  remote_row=$(cap | grep -n "$REMOTE_NAME" | head -1 | cut -d: -f1)
  if [ -n "$remote_row" ]; then
    mclick 45 "$remote_row"
  else
    send Down
    send Enter
  fi
  must 8 "remote: click attaches the tcp terminal" "$REMOTE_NAME"
  sendl 'echo REMOTE-OK'
  send Enter
  must 8 "remote: input reaches the remote daemon over tcp" 'REMOTE-OK'
  sendl 'stty size'
  send Enter
  must 5 "remote: tcp attach uses the full pane size (32 118)" '32 118'
  tm resize-window -t "$SESSION" -x 100 -y 30
  sleep 0.6
  sendl 'echo REMOTE-RS $(stty size)'
  send Enter
  must 8 "remote: tmux resize follows over tcp (26 98)" 'REMOTE-RS 26 98'
  tm resize-window -t "$SESSION" -x "$COLS" -y "$ROWS"
  sleep 0.6
  sendl 'echo REMOTE-RS2 $(stty size)'
  send Enter
  must 8 "remote: resizing back restores the tcp PTY (32 118)" 'REMOTE-RS2 32 118'

  # Remote exit + restart through the tcp link.
  if ! remote_cli v3 kill "$REMOTE_NAME" >"$WORK/remote-kill.log" 2>&1; then
    echo "DEBUG remote kill rc=$?"; sed -n '1,4p' "$WORK/remote-kill.log"
  fi
  if waitgrep 10 "\[exited [0-9]+\]"; then
    ok "remote: killed remote terminal shows the exit badge"
  else
    bad "remote: killed remote terminal shows the exit badge"
    echo "DEBUG remote ls:"; remote_cli v3 ls 2>&1 | sed -n '1,6p'
    echo "DEBUG slot:"; cap | sed -n 2p
    echo "DEBUG status:"; cap | tail -1
    echo "DEBUG remote daemon log:"; tail -6 "$WORK/remote-daemon.log" 2>/dev/null
  fi
  send C-e
  sleep 1.5
  sendl 'echo REMOTE-RESTART-OK'
  send Enter
  must 8 "remote: Ctrl-E restarts the remote terminal over tcp" 'REMOTE-RESTART-OK'

  # Tunnel loss: health goes offline with a readable notice; restarting the
  # tunnel on the same address reconnects and the live stream resumes.
  stop_tunnel
  must 8 "remote: losing the tunnel flips health offline" 'endpoint remote offline'
  restart_tunnel() {
    if ((SSH_REMOTE)); then
      ssh -N -o BatchMode=yes -o StrictHostKeyChecking=no -o ExitOnForwardFailure=yes \
        -L "127.0.0.1:$ORIGINAL_PORT:$REMOTE_SOCK" localhost >"$WORK/ssh-tcp2.log" 2>&1 &
      TUNNEL_PID=$!
      if wait_port "$ORIGINAL_PORT"; then
        return 0
      fi
      stop_tunnel
    fi
    "$WORK/remote-bridge" -listen "127.0.0.1:$ORIGINAL_PORT" -unix "$REMOTE_SOCK" >"$WORK/bridge2.log" 2>&1 &
    TUNNEL_PID=$!
    wait_port "$ORIGINAL_PORT"
  }
  restart_tunnel || bad "remote: restart the tunnel on the original address"
  must 8 "remote: restarting the tunnel reconnects the endpoint" 'endpoint remote connected'
  sendl 'echo REMOTE-REATTACH-OK'
  send Enter
  must 8 "remote: input works again after the tunnel restarts" 'REMOTE-REATTACH-OK'

  # P0 command path: the old `ssh host anytty ...` PTY still works as a
  # command endpoint beside the daemon endpoint.
  send C-p
  sleep 0.3
  send C-f
  sleep 0.3
  must 5 "remote: picker lists the ssh command endpoint" 'ssh-cmd'
  ssh_row=$(cap | grep -n 'ssh-cmd' | head -1 | cut -d: -f1)
  if [ -n "$ssh_row" ]; then
    mclick 45 "$ssh_row"
  else
    send Down
    send Enter
  fi
  must 8 "remote: ssh command endpoint binds" 'bound sshcmd'
  sendl 'echo SSH-CMD-OK'
  send Enter
  must 8 "remote: ssh command endpoint echoes through its PTY" 'SSH-CMD-OK'

  send Escape
  send C-q
  must 5 "remote: session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "remote: remote endpoints did not break shutdown" 'REMOTEEXIT:0'
  stop_tunnel
fi

# ------------------------------- P0 local-unix: ssh -L local.sock:remote
echo
echo "== daemon endpoint: ssh unix-socket forward (P0) =="
FWD_PID=""
stop_fwd() {
  if [[ -n "$FWD_PID" ]]; then
    kill "$FWD_PID" 2>/dev/null || true
    wait "$FWD_PID" 2>/dev/null || true
    FWD_PID=""
  fi
}
if [[ -z "${REMOTE_SOCK:-}" ]] || [[ ! -S "${REMOTE_SOCK:-/nonexistent}" ]]; then
  echo "SKIP  remote daemon unavailable"
elif ((!${SSH_REMOTE:-0})); then
  echo "SKIP  ssh localhost unavailable; tcp section already covers the Go bridge"
else
  FWD_SOCK="$WORK/remote-fwd.sock"
  rm -f "$FWD_SOCK"
  ssh -N -o BatchMode=yes -o StrictHostKeyChecking=no -o ExitOnForwardFailure=yes \
    -L "$FWD_SOCK:$REMOTE_SOCK" localhost >"$WORK/ssh-unix.log" 2>&1 &
  FWD_PID=$!
  for _ in $(seq 1 50); do [[ -S "$FWD_SOCK" ]] && break; sleep 0.1; done
  if [[ ! -S "$FWD_SOCK" ]]; then
    bad "remote-ssh: ssh -L created the local unix forward"
    cat "$WORK/ssh-unix.log" 2>/dev/null | sed -n '1,3p'
  else
    ok "remote-ssh: ssh -L created the local unix forward"
    cat >"$WORK/tui2-remote-ssh-config.json" <<CFG
{
  "sidebar": false,
  "clock": { "enabled": false },
  "endpoints": [
    { "name": "sshtun", "kind": "daemon", "label": "ssh-tunnel", "socket": "$FWD_SOCK" }
  ]
}
CFG
    tm kill-session -t "$SESSION" 2>/dev/null || true
    SESSION="tui2remotessh"
    tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
      "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell --config $WORK/tui2-remote-ssh-config.json\"; echo SSHTUNEXIT:\$?; exec bash'"
    sleep 0.9
    if cap | grep -Eq "$REMOTE_NAME +sshtun · live"; then
      ok "remote-ssh: picker lists the terminal behind the unix forward"
    else
      bad "remote-ssh: picker lists the terminal behind the unix forward"
    fi
    ssh_row=$(cap | grep -n "$REMOTE_NAME" | head -1 | cut -d: -f1)
    if [ -n "$ssh_row" ]; then
      mclick 45 "$ssh_row"
    else
      send Down
      send Enter
    fi
    sendl 'echo SSH-UNIX-OK'
    send Enter
    must 8 "remote-ssh: input reaches the daemon through ssh -L unix" 'SSH-UNIX-OK'
    stop_fwd
    must 8 "remote-ssh: dropping the forward flips health offline" 'endpoint sshtun offline'
    # ssh leaves the local socket file behind after SIGTERM; remove it before
    # rebinding the same path.
    rm -f "$FWD_SOCK"
    ssh -N -o BatchMode=yes -o StrictHostKeyChecking=no -o ExitOnForwardFailure=yes \
      -L "$FWD_SOCK:$REMOTE_SOCK" localhost >"$WORK/ssh-unix2.log" 2>&1 &
    FWD_PID=$!
    for _ in $(seq 1 50); do [[ -S "$FWD_SOCK" ]] && break; sleep 0.1; done
    must 8 "remote-ssh: forward restart reconnects the endpoint" 'endpoint sshtun connected'
    sendl 'echo SSH-UNIX-AGAIN'
    send Enter
    must 8 "remote-ssh: input works again through the restarted forward" 'SSH-UNIX-AGAIN'
    send C-q
    must 5 "remote-ssh: session quits cleanly" 'Quit tui2\?'
    send Enter
    must 8 "remote-ssh: ssh tunnel endpoint did not break shutdown" 'SSHTUNEXIT:0'
    stop_fwd
  fi
fi
if [[ -n "${REMOTE_SOCK:-}" ]] && [[ -S "${REMOTE_SOCK:-/nonexistent}" ]]; then
  remote_cli v3 kill "$REMOTE_NAME" >/dev/null 2>&1 || true
  remote_cli v3 remove "$REMOTE_NAME" >/dev/null 2>&1 || true
  remote_cli daemon stop >/dev/null 2>&1 || true
  REMOTE_SOCK=""
fi


# ================= starter templates + dev loop (M3 starters / M2 dev)
# The three official starters must load through their SDK and show the same
# chrome in the host; the -dev loop must hot-reload on save, write a decoded
# bidirectional frame log and surface crashes as readable notices.
echo
echo "== starter templates: go / python / ts =="
TEMPLATES="$ROOT/clients/tui/templates"
if (cd "$ROOT" && "$GO" build -o "$WORK/template-go" ./clients/tui/templates/go) >"$WORK/template-go-build.log" 2>&1; then
  ok "template/go: go build (starter compiles against the Go SDK)"
else
  bad "template/go: go build"
  sed -n '1,3p' "$WORK/template-go-build.log"
fi

template_check() { # template_check <label> <session> <shell-cmd> <exit-tag>
  local label="$1" session="$2" shellcmd="$3" exittag="$4"
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="$session"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$shellcmd\"; echo $exittag:\$?; exec bash'"
  sleep 0.9
  must 10 "$label: tab bar renders" '\[main\]'
  must 5 "$label: cold start picker lists sources" 'New terminal'
  send Enter
  must 8 "$label: Enter creates and binds a terminal" 'bound term-1'
  must 3 "$label: footer key hints render after the bind" 'Ctrl-F picker'
  must_side 5 left "$label: the bound terminal component is shown" 'term-1'
  send C-q
  must 5 "$label: Ctrl-Q opens the host confirmation" 'Quit tui2\?'
  send Enter
  must 8 "$label: quits cleanly" "$exittag:0"
}

if [[ -x "$WORK/template-go" ]]; then
  template_check "template/go" "tui2tplgo" "$WORK/template-go" "TPLGOEXIT"
fi
template_check "template/python" "tui2tplpy" "$PY $TEMPLATES/python/program.py" "TPLPYEXIT"
if command -v "$NODE" >/dev/null 2>&1; then
  template_check "template/ts" "tui2tplts" "$NODE $TEMPLATES/ts/program.js" "TPLTSXIT"
else
  echo "SKIP  template/ts (node not installed)"
fi

echo
echo "== dev loop: hot reload + frame log + crash notices =="
DEV_SRC="$WORK/tpl-dev.py"
DEV_LOG="$WORK/dev-protocol.jsonl"
DEV_TEMPLATE="$TEMPLATES/python/program.py"
cp "$DEV_TEMPLATE" "$DEV_SRC"
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2devloop"
tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "bash -c 'TUI2_PYTHON_SDK=\"$ROOT/clients/tui/sdk/python\" \"$WORK/tui2\" -dev -protocol-log \"$DEV_LOG\" -watch \"$DEV_SRC\" -shell \"$PY $DEV_SRC\"; echo TPLDEVEXIT:\$?; exec bash'"
sleep 0.9
must 10 "dev: starter loads and renders" '\[main\]'
send Enter
must 8 "dev: terminal binds before the reload" 'bound term-1'
# Protocol log: both directions, decoded records with a timestamp.
if [[ -s "$DEV_LOG" ]]; then
  ok "dev: -protocol-log writes decoded frames to the file"
else
  bad "dev: -protocol-log writes decoded frames to the file"
fi
if grep -q '"dir":"host->program"' "$DEV_LOG" && grep -q '"dir":"program->host"' "$DEV_LOG"; then
  ok "dev: frame log decodes both directions"
else
  bad "dev: frame log decodes both directions"
  sed -n '1,2p' "$DEV_LOG"
fi
if grep -q '"type":"HELLO"' "$DEV_LOG" && grep -q '"type":"VIEW"' "$DEV_LOG" && grep -q '"ts":"' "$DEV_LOG"; then
  ok "dev: frame log carries timestamps and decoded types"
else
  bad "dev: frame log carries timestamps and decoded types"
fi
if cap | grep -Eq '"dir"|"type":"(VIEW|HELLO)"'; then
  bad "dev: no protocol log lines reach the terminal"
else
  ok "dev: no protocol log lines reach the terminal"
fi
# Hot reload: a save changes the chrome and keeps the last good tree.
sed -i 's/" main "/" DEVX "/' "$DEV_SRC"
must 10 "dev: saving the watched file hot-reloads (chrome changes)" 'DEVX'
must 5 "dev: the reload notice names the changed file" 'reloaded'
if cap | grep -Eq 'term-1'; then
  ok "dev: the last good frame survives the reload"
else
  bad "dev: the last good frame survives the reload"
fi
send C-q
must 5 "dev: loop session quits" 'Quit tui2\?'
send Enter
must 8 "dev: loop session exits cleanly" 'TPLDEVEXIT:0'

# Crash visibility: the first run prints to stderr and exits 3; the restart is
# healthy and renders the queued crash notice (reason, countdown, stderr).
cat >"$WORK/crash-once.py" <<'PY'
import os, sys
flag = os.environ["TUI2_CRASH_FLAG"]
if os.path.exists(flag):
    os.remove(flag)
    sys.stderr.write("DEV-CRASH-MARKER\n")
    sys.stderr.flush()
    raise SystemExit(3)
sys.path.insert(0, os.environ["TUI2_PYTHON_SDK"])
from tui2sdk import App, Client, builder


class Recovery(App):
    def __init__(self):
        self.lines = ["CRASH-RECOVERED"]

    def commit(self):
        self.client.commit(builder.col(*[builder.text(line) for line in self.lines]).build(), [])

    def on_hello(self, hello):
        self.commit()

    def on_sources(self, event):
        self.commit()

    def on_notice(self, event):
        self.lines.append(event["message"])
        self.commit()


Client(sys.stdin.buffer, sys.stdout.buffer).run(Recovery())
PY
CRASH_FLAG="$WORK/crash-once.flag"
: >"$CRASH_FLAG"
tm kill-session -t "$SESSION" 2>/dev/null || true
SESSION="tui2devcrash"
tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  "bash -c 'TUI2_PYTHON_SDK=\"$ROOT/clients/tui/sdk/python\" TUI2_CRASH_FLAG=\"$CRASH_FLAG\" \"$WORK/tui2\" -dev -shell \"$PY $WORK/crash-once.py\"; echo TPLCRASHEXIT:\$?; exec bash'"
sleep 1.5
must 10 "dev: a crashed program restarts and reports the crash" 'CRASH-RECOVERED'
must 8 "dev: crash notice carries the reason and restart countdown" 'restarting in'
must 8 "dev: crash notice carries the program stderr tail" 'DEV-CRASH-MARKER'
DEV_DEFAULT_LOG="$XDG_STATE_HOME/anytty/tui2-dev.log"
if [[ -s "$DEV_DEFAULT_LOG" ]] && grep -q 'DEV-CRASH-MARKER' "$DEV_DEFAULT_LOG"; then
  ok "dev: -dev default protocol log records the crash diagnostic"
else
  bad "dev: -dev default protocol log records the crash diagnostic"
fi
send C-q
must 5 "dev: crash session quits" 'Quit tui2\?'
send Enter
must 8 "dev: crash session exits cleanly" 'TPLCRASHEXIT:0'

# ================= CLI registry -> TUI (M3, shared client layer)
# The CLI pairs/manages endpoints into the shared registry; the TUI host
# loads that registry at startup, connects through the shared client layer
# and publishes the daemon terminals/endpoint badges to the picker.
echo
echo "== cli registry -> tui connection (shared client layer) =="
if ((!EP_DAEMON)); then
  echo "SKIP  anytty-dev daemon unavailable"
else
  REG_NAME="reg-$$"
  REG_TERM="REG-TERM-$$"
  DEAD_NAME="deadreg-$$"
  REGISTRY_FILE="$XDG_CONFIG_HOME/anytty/endpoints.yaml"
  regcli() { "$DEV_BIN" endpoint "$@"; }

  if regcli add local "$REG_NAME" --socket "$EP_SOCK" --label reg-daemon >"$WORK/reg-add.log" 2>&1; then
    ok "registry: CLI endpoint add writes the shared registry"
  else
    bad "registry: CLI endpoint add writes the shared registry"
    sed -n '1,3p' "$WORK/reg-add.log"
  fi
  if grep -q "$REG_NAME" "$REGISTRY_FILE"; then
    ok "registry: endpoints.yaml contains the CLI-paired endpoint"
  else
    bad "registry: endpoints.yaml contains the CLI-paired endpoint"
  fi
  if epcli v3 new --name "$REG_TERM" -- sh >"$WORK/reg-term.log" 2>&1; then
    ok "registry: CLI creates a daemon terminal for the paired endpoint"
  else
    bad "registry: CLI creates a daemon terminal for the paired endpoint"
  fi

  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2registry"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo REGISTRYEXIT:\$?; exec bash'"
  sleep 1
  must 10 "registry: TUI picker lists the CLI-paired daemon terminal" "$REG_TERM.*$REG_NAME · live"
  if regcli list 2>/dev/null | grep -q "$REG_NAME"; then
    ok "registry: CLI endpoint list shows the paired endpoint"
  else
    bad "registry: CLI endpoint list shows the paired endpoint"
  fi
  reg_row=$(cap | grep -n "$REG_TERM" | head -1 | cut -d: -f1)
  if [ -n "$reg_row" ]; then
    mclick 45 "$reg_row"
  else
    send Down
    send Enter
  fi
  must 8 "registry: click attaches the CLI-paired daemon terminal" "$REG_TERM"
  sendl 'echo REG-OK'
  send Enter
  must 8 "registry: input reaches the terminal via the shared client layer" 'REG-OK'
  sendl 'exit'
  send Enter
  must 8 "registry: typed exit shows the daemon exit badge" '\[exited 0\]'
  send C-e
  sleep 1.5
  sendl 'echo REG-RESTART-OK'
  send Enter
  must 8 "registry: Ctrl-E restarts the daemon terminal" 'REG-RESTART-OK'

  # Offline endpoint: the host still lists it with a readable health badge and
  # notice, and never crashes.
  regcli add local "$DEAD_NAME" --socket "$WORK/no-such-registry.sock" --label dead-reg >"$WORK/reg-dead.log" 2>&1 || true
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2registrydead"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo REGDEADEXIT:\$?; exec bash'"
  sleep 1
  must 10 "registry: picker lists the offline paired endpoint with a badge" 'dead-reg.*endpoint · offline'
  must 8 "registry: offline paired endpoint emits a readable notice" "endpoint $DEAD_NAME offline"
  send Escape
  send C-q
  must 5 "registry: session quits cleanly with an offline registry endpoint" 'Quit tui2\?'
  send Enter
  must 8 "registry: host did not crash on the offline endpoint" 'REGDEADEXIT:0'
fi

# ================= legacy registry read-only (TUI2_ENDPOINTS, M3)
# A dev build runs under an isolated XDG tree; TUI2_ENDPOINTS makes it list
# endpoints already paired in another (production) XDG without re-pairing,
# copying or writing the file. Explicit registries win by endpoint name and
# the dev default registry is still merged.
echo
echo "== legacy registry read-only via TUI2_ENDPOINTS =="
if ((!EP_DAEMON)); then
  echo "SKIP  anytty-dev daemon unavailable"
else
  LEGACY_XDG="$WORK/legacy-xdg"
  LEGACY_CONFIG="$LEGACY_XDG/anytty"
  LEGACY_REGISTRY="$LEGACY_CONFIG/endpoints.yaml"
  LEGACY_NAME="legacy-$$"
  LEGACY_TERM="LEGACY-TERM-$$"
  EMPTY_XDG="$WORK/legacy-empty-xdg"
  mkdir -p "$LEGACY_CONFIG" "$EMPTY_XDG"

  # 1) The "old" XDG writes its registry with the normal CLI pairing command;
  #    the host below only ever reads it (no `endpoint add` at TUI time).
  if XDG_CONFIG_HOME="$LEGACY_XDG" XDG_STATE_HOME="$WORK/legacy-xdg/state" "$DEV_BIN" endpoint add local "$LEGACY_NAME" --socket "$EP_SOCK" --label legacy-daemon >"$WORK/legacy-add.log" 2>&1; then
    ok "legacy: old XDG endpoint add writes the legacy registry"
  else
    bad "legacy: old XDG endpoint add writes the legacy registry"
    sed -n '1,3p' "$WORK/legacy-add.log"
  fi
  # The same name in the dev registry points at a dead socket: only the
  # explicit legacy registry may win, so the live terminal stays visible.
  "$DEV_BIN" endpoint add local "$LEGACY_NAME" --socket "$WORK/legacy-shadow.sock" --label legacy-shadow >"$WORK/legacy-shadow.log" 2>&1 || true
  if epcli v3 new --name "$LEGACY_TERM" -- sh >"$WORK/legacy-term.log" 2>&1; then
    ok "legacy: daemon terminal exists for the legacy endpoint"
  else
    bad "legacy: daemon terminal exists for the legacy endpoint"
  fi
  LEGACY_BEFORE="$(md5sum "$LEGACY_REGISTRY" | cut -d' ' -f1)"

  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2legacy"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c 'XDG_CONFIG_HOME=\"$XDG_CONFIG_HOME\" XDG_STATE_HOME=\"$XDG_STATE_HOME\" XDG_RUNTIME_DIR=\"$XDG_RUNTIME_DIR\" ANYTTY_TUI2_CONFIG=\"$ANYTTY_TUI2_CONFIG\" TUI2_ENDPOINTS=\"$LEGACY_REGISTRY\" \"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo LEGACYEXIT:\$?; exec bash'"
  sleep 1
  must 10 "legacy: TUI2_ENDPOINTS lists the legacy terminal (explicit beats same-name dev entry)" "$LEGACY_TERM.*$LEGACY_NAME · live"
  must 8 "legacy: dev registry endpoint stays merged under its own name" "$REG_TERM.*$REG_NAME · live"
  legacy_row=$(cap | grep -n "$LEGACY_TERM" | head -1 | cut -d: -f1)
  if [ -n "$legacy_row" ]; then
    mclick 45 "$legacy_row"
  else
    send Down
    send Enter
  fi
  must 8 "legacy: click attaches the legacy endpoint terminal" "$LEGACY_TERM"
  sendl 'echo LEGACY-OK'
  send Enter
  must 8 "legacy: input reaches the legacy endpoint without re-pairing" 'LEGACY-OK'
  LEGACY_AFTER="$(md5sum "$LEGACY_REGISTRY" | cut -d' ' -f1)"
  if [ "$LEGACY_BEFORE" = "$LEGACY_AFTER" ]; then
    ok "legacy: host never wrote back to the legacy registry"
  else
    bad "legacy: host never wrote back to the legacy registry"
  fi
  send C-q
  must 5 "legacy: session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "legacy: host exits cleanly with an explicit legacy registry" 'LEGACYEXIT:0'

  # 2) Missing explicit file: one readable warning, host stays alive and never
  #    crashes. The isolated XDG has no default registry, so nothing overwrites
  #    the warning status.
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2legacymissing"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c 'XDG_CONFIG_HOME=\"$EMPTY_XDG\" XDG_STATE_HOME=\"$XDG_STATE_HOME\" XDG_RUNTIME_DIR=\"$XDG_RUNTIME_DIR\" ANYTTY_TUI2_CONFIG=\"$ANYTTY_TUI2_CONFIG\" TUI2_ENDPOINTS=\"$WORK/legacy-missing.yaml\" \"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo LEGACYMISSINGEXIT:\$?; exec bash'"
  sleep 1
  must 8 "legacy: missing explicit registry emits a readable notice" 'endpoint registry .*legacy-missing\.yaml'
  send Escape
  send C-q
  must 5 "legacy: missing-registry session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "legacy: missing registry did not crash the host" 'LEGACYMISSINGEXIT:0'

  # 3) Corrupt explicit file: same readable-error contract for a format error.
  printf 'version: 3\nendpoints: [broken\n' >"$WORK/legacy-corrupt.yaml"
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2legacycorrupt"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c 'XDG_CONFIG_HOME=\"$EMPTY_XDG\" XDG_STATE_HOME=\"$XDG_STATE_HOME\" XDG_RUNTIME_DIR=\"$XDG_RUNTIME_DIR\" ANYTTY_TUI2_CONFIG=\"$ANYTTY_TUI2_CONFIG\" TUI2_ENDPOINTS=\"$WORK/legacy-corrupt.yaml\" \"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo LEGACYCORRUPTEXIT:\$?; exec bash'"
  sleep 1
  must 8 "legacy: corrupt explicit registry emits a readable notice" 'endpoint registry .*legacy-corrupt\.yaml'
  send Escape
  send C-q
  must 5 "legacy: corrupt-registry session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "legacy: corrupt registry did not crash the host" 'LEGACYCORRUPTEXIT:0'
fi

# ================= log takeover + route policy (old registry, M1/M2) =======
# An old registry can carry direct/cloud routes beside a local one. The TUI
# must never print shared-layer logs (anytty connect / network attempt /
# trace_id= / webrtc) over the alt screen: the standard logger goes to the log
# file (default $XDG_STATE_HOME/anytty/tui2.log, or TUI2_LOG_FILE/-log-file),
# host diagnostics go to file + notice. By default only local-unix (plus a
# credential-backed ssh route) is dialed: direct/cloud endpoints stay listed
# offline with a single readable notice, and a mixed endpoint degrades to
# local-unix. Opting in via TUI2_ROUTES dials them, still logging to file only.
echo
echo "== log takeover + route policy (old registry with webrtc/cloud) =="
POLICY_LOG_MARKER='trace_id=|anytty (connect|cloud|network)|webrtc selected'
if ((!EP_DAEMON)); then
  echo "SKIP  anytty-dev daemon unavailable"
else
  POLICY_XDG="$WORK/policy-xdg"
  POLICY_REGISTRY="$POLICY_XDG/anytty/endpoints.yaml"
  POLICY_DEFAULT_LOG="$XDG_STATE_HOME/anytty/tui2.log"
  POLICY_OPT_LOG="$WORK/policy-opt-tui2.log"
  POLICY_FAIL_LOG="$WORK/policy-fail-tui2.log"
  MIX_NAME="mix-$$"
  MIX_TERM="MIX-TERM-$$"
  CLOUD_NAME="cloud-$$"
  mkdir -p "$POLICY_XDG/anytty"
  policycli() { env XDG_CONFIG_HOME="$POLICY_XDG" XDG_STATE_HOME="$WORK/policy-xdg/state" "$DEV_BIN" endpoint "$@"; }

  # Build the old registry with the real CLI: mix1 = local-unix + webrtc,
  # directonly = webrtc only, cloud1 = managed-webrtc only. The mixed local
  # route is pinned to the live daemon identity so it can actually attach.
  DEVICE_ID="$(epcli access identity 2>/dev/null | sed -n 's/^Device[[:space:]]*//p')"
  DEVICE_FP="$(epcli access identity 2>/dev/null | sed -n 's/^Fingerprint[[:space:]]*//p')"
  if policycli add local "$MIX_NAME" --socket "$EP_SOCK" --label mixed-daemon \
      --device-id "$DEVICE_ID" --device-fingerprint "$DEVICE_FP" >"$WORK/policy-add.log" 2>&1 \
    && policycli route add direct "$MIX_NAME" direct --signaling-address 127.0.0.1:1 --ice-tcp-address 127.0.0.1:1 >>"$WORK/policy-add.log" 2>&1 \
    && policycli add direct directonly --signaling-address 127.0.0.1:1 --ice-tcp-address 127.0.0.1:1 \
      --device-id device-direct --device-fingerprint ed25519-sha256:direct --label direct-only >>"$WORK/policy-add.log" 2>&1 \
    && policycli add cloud "$CLOUD_NAME" --device-id device-cloud --device-fingerprint ed25519-sha256:cloud \
      --target-device-id device-cloud --account-profile-ref cloud-profile --label cloud-only >>"$WORK/policy-add.log" 2>&1; then
    ok "policy: real old registry carries local+webrtc, webrtc-only and cloud-only endpoints"
  else
    bad "policy: real old registry carries local+webrtc, webrtc-only and cloud-only endpoints"
    sed -n '1,3p' "$WORK/policy-add.log"
  fi
  if epcli v3 new --name "$MIX_TERM" -- sh >"$WORK/policy-term.log" 2>&1; then
    ok "policy: daemon terminal exists for the mixed endpoint"
  else
    bad "policy: daemon terminal exists for the mixed endpoint"
  fi

  # ---- default policy: no webrtc/cloud dial, no log on the terminal --------
  # The policy run uses its own config home so the old registry is the only
  # endpoint source: the pruned-endpoint notice cannot be buried by the dev
  # registry's other offline notices.
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2policy"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c 'XDG_CONFIG_HOME=\"$POLICY_XDG\" XDG_STATE_HOME=\"$XDG_STATE_HOME\" XDG_RUNTIME_DIR=\"$XDG_RUNTIME_DIR\" ANYTTY_TUI2_CONFIG=\"$ANYTTY_TUI2_CONFIG\" TUI2_ENDPOINTS=\"$POLICY_REGISTRY\" \"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo POLICYEXIT:\$?; exec bash'"
  sleep 1.2

  must 10 "policy: mixed endpoint degrades to local and lists its terminal live" "$MIX_TERM.*$MIX_NAME · live"
  must 8 "policy: cloud-only endpoint stays listed offline" 'cloud-only.*endpoint · offline'
  must 8 "policy: webrtc-only endpoint stays listed offline" 'direct-only.*endpoint · offline'
  if driver_supports transient-notice; then
    must 8 "policy: pruned endpoint has a one-line notice" "endpoint $CLOUD_NAME offline"
  else
    echo "SKIP  policy: pruned endpoint has a one-line notice (pty driver: the single transient notice slot can be overwritten by a later endpoint notice before capture; the durable log assertion below still runs)"
  fi
  if cap | grep -Eq "$POLICY_LOG_MARKER"; then
    bad "policy: no shared-layer log lines reach the terminal"
    cap | grep -E "$POLICY_LOG_MARKER" | head -2
  else
    ok "policy: no shared-layer log lines reach the terminal"
  fi
  if [ -s "$POLICY_DEFAULT_LOG" ] && grep -q 'tui2 start' "$POLICY_DEFAULT_LOG" && grep -q "endpoint $CLOUD_NAME offline" "$POLICY_DEFAULT_LOG"; then
    ok "policy: default log file records startup and offline diagnostics"
  else
    bad "policy: default log file records startup and offline diagnostics"
    sed -n '1,4p' "$POLICY_DEFAULT_LOG" 2>/dev/null
  fi
  # Attach through the degraded local route and prove input works.
  mix_row=$(cap | grep -n "$MIX_TERM" | head -1 | cut -d: -f1)
  if [ -n "$mix_row" ]; then
    mclick 45 "$mix_row"
  else
    send Down
    send Enter
  fi
  must 8 "policy: degraded local route attaches the terminal" "$MIX_TERM"
  sendl 'echo POLICY-OK'
  send Enter
  must 8 "policy: input reaches the terminal through the degraded route" 'POLICY-OK'
  send C-q
  must 5 "policy: session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "policy: terminal restored after quit" 'POLICYEXIT:0'
  sleep 0.5
  if cap | grep -Eq "$POLICY_LOG_MARKER"; then
    bad "policy: no log lines appear on the terminal after quitting"
  else
    ok "policy: no log lines appear on the terminal after quitting"
  fi

  # ---- opt-in webrtc: shared diagnostics go to the file, not the screen ----
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2policyopt"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c 'XDG_CONFIG_HOME=\"$POLICY_XDG\" XDG_STATE_HOME=\"$XDG_STATE_HOME\" XDG_RUNTIME_DIR=\"$XDG_RUNTIME_DIR\" ANYTTY_TUI2_CONFIG=\"$ANYTTY_TUI2_CONFIG\" TUI2_ENDPOINTS=\"$POLICY_REGISTRY\" TUI2_LOG_FILE=\"$POLICY_OPT_LOG\" TUI2_ROUTES=local-unix,direct-webrtc-tcp \"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo POLICYOPTEXIT:\$?; exec bash'"
  sleep 1.5
  must 10 "policy: opt-in webrtc keeps the picker usable" 'select a terminal'
  if cap | grep -Eq "$POLICY_LOG_MARKER"; then
    bad "policy: opt-in webrtc diagnostics stay off the terminal"
    cap | grep -E "$POLICY_LOG_MARKER" | head -2
  else
    ok "policy: opt-in webrtc diagnostics stay off the terminal"
  fi
  if grep -Eq 'anytty (connect|network attempt)' "$POLICY_OPT_LOG"; then
    ok "policy: shared-layer diagnostics are redirected to the log file"
  else
    bad "policy: shared-layer diagnostics are redirected to the log file"
    sed -n '1,6p' "$POLICY_OPT_LOG" 2>/dev/null
  fi
  send Escape
  send C-q
  must 5 "policy: opt-in session quits cleanly" 'Quit tui2\?'
  send Enter
  must 8 "policy: opt-in run exits cleanly" 'POLICYOPTEXIT:0'

  # ---- startup failure: -log-file gets the startup line, terminal stays TUI-clean
  tm kill-session -t "$SESSION" 2>/dev/null || true
  SESSION="tui2policyfail"
  tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
    "bash -c 'TUI2_ENDPOINTS=\"$POLICY_REGISTRY\" \"$WORK/tui2\" -log-file \"$POLICY_FAIL_LOG\" -shell \"$WORK/does-not-exist-shell\"; echo POLICYFAILEXIT:\$?; exec bash'"
  sleep 0.8
  if cap | grep -Eq "$POLICY_LOG_MARKER"; then
    bad "policy: startup failure prints no shared-layer log lines"
  else
    ok "policy: startup failure prints no shared-layer log lines"
  fi
  must 5 "policy: startup failure is a readable shell error" 'does-not-exist-shell'
  must 8 "policy: startup failure exits non-zero after the log redirect" 'POLICYFAILEXIT:1'
  if [ -s "$POLICY_FAIL_LOG" ] && grep -q 'tui2 start' "$POLICY_FAIL_LOG"; then
    ok "policy: --log-file exists even when the layout shell cannot start"
  else
    bad "policy: --log-file exists even when the layout shell cannot start"
  fi
fi

# ================= M5 guards: shared dialer only, no tui2 re-implementation
echo
echo "== connection layer guards =="
TUI2_IMPORTS="$(cd "$ROOT" && "$GO" list -f '{{join .Imports "\n"}}' ./clients/tui/endpoint ./clients/tui/cmd/tui2 2>/dev/null || true)"
if printf '%s\n' "$TUI2_IMPORTS" | grep -qx 'github.com/anytty/anytty/access/engine/adapter/protocol'; then
  ok "guard: tui2 imports the shared protocol adapter"
else
  bad "guard: tui2 imports the shared protocol adapter"
fi
if printf '%s\n' "$TUI2_IMPORTS" | grep -qx 'github.com/anytty/anytty/shared/transport/unix'; then
  bad "guard: tui2 still imports the raw unix transport directly"
else
  ok "guard: tui2 no longer imports the raw unix transport directly"
fi

echo
echo "== acceptance summary: $PASS passed, $FAIL failed =="
exit "$FAIL"
