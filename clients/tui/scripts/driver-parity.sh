#!/usr/bin/env bash
# Driver parity acceptance: proves the smoke/acceptance assertions run on both
# test drivers and produce the same results.
#
#   * smoke.sh passes with tmux, with the pty driver forced, and with tmux
#     masked out of PATH (auto fallback)
#   * the acceptance key categories (picker / attach / input / split /
#     scrollback copy / resize / exit) pass on both drivers
#   * the pty driver's resize reaches the inner PTY (stty size)
#   * OSC 52 clipboard writes are captured by the pty driver
#   * identical scenarios produce identical normalized captures on both
#     drivers (sampled: cold-start picker and bound-terminal screens)
#
# Usage: bash clients/tui/scripts/driver-parity.sh
#        (tmux rows are skipped, not failed, when tmux is not installed)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
# shellcheck source=libdriver.sh
source "$ROOT/clients/tui/scripts/libdriver.sh"
GO="${GO:-go}"
SOCK="tui2-parity-$$"
WORK="$(mktemp -d)"
SESSION="tui2-parity-$$"
PASS=0
FAIL=0
SKIP=0
HAS_TMUX=0
command -v tmux >/dev/null 2>&1 && HAS_TMUX=1

export XDG_CONFIG_HOME="$WORK/xdg/config"
export XDG_STATE_HOME="$WORK/xdg/state"
export XDG_RUNTIME_DIR="$WORK/xdg/run"
mkdir -p "$XDG_CONFIG_HOME" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR"
cat >"$WORK/tui2-config.json" <<'CFG'
{ "sidebar": false, "gap": 1, "clock": { "enabled": false } }
CFG
export ANYTTY_TUI2_CONFIG="$WORK/tui2-config.json"

cleanup() {
  driver_kill_all 2>/dev/null || true
  driver_shutdown
  if [ "${TUI2_PARITY_KEEP:-0}" = 1 ]; then
    printf 'driver-parity: keeping %s (TUI2_PARITY_KEEP=1)\n' "$WORK"
  else
    rm -rf "$WORK"
  fi
}
trap cleanup EXIT

ok() { printf 'PASS  %s\n' "$1"; PASS=$((PASS + 1)); }
bad() { printf 'FAIL  %s\n' "$1"; FAIL=$((FAIL + 1)); }
skip() { printf 'SKIP  %s\n' "$1"; SKIP=$((SKIP + 1)); }

echo "== building tui2, tui2-shell and the pty harness =="
if ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2" ./clients/tui/cmd/tui2) ||
  ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2-shell" ./clients/tui/cmd/tui2-shell) ||
  ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2-harness" ./clients/tui/cmd/tui2-harness); then
  echo "FAIL  build test binaries"
  exit 1
fi
export TUI2_HARNESS_BIN="$WORK/tui2-harness"

waitgrep() { # <seconds> <pattern>
  local deadline=$((SECONDS + $1))
  while ((SECONDS < deadline)); do
    if driver_capture "$SESSION" 2>/dev/null | grep -Eq "$2"; then return 0; fi
    sleep 0.2
  done
  return 1
}

must() { # <seconds> <description> <pattern>
  if waitgrep "$1" "$3"; then ok "$2"; else bad "$2"; fi
}

# ------------------------------------------------------------- key categories
# The same assertions run under whichever driver driver_init selected. They
# mirror the acceptance categories: picker, attach, input, split, scrollback
# copy (OSC 52), and exit.
key_categories() {
  local label="$1"
  driver_kill "$SESSION" 2>/dev/null || true
  driver_spawn "$SESSION" 120 36 \
    "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo EXIT:\$?; sleep 20'"
  # tmux only accepts application OSC 52 writes with set-clipboard on; the
  # pty driver captures them natively and treats this as a no-op.
  tm set-option -g set-clipboard on 2>/dev/null || true
  must 10 "$label picker: cold start opens the terminal picker" 'select a terminal'
  driver_send "$SESSION" Enter
  must 8 "$label attach: enter creates and binds term-1" 'term-1'
  driver_send_literal "$SESSION" 'echo PARITY-INPUT-OK'
  driver_send "$SESSION" Enter
  must 8 "$label input: typing reaches the focused PTY" 'PARITY-INPUT-OK'
  driver_send "$SESSION" C-p
  sleep 0.3
  driver_send "$SESSION" '%'
  must 8 "$label split: Ctrl-P % opens a second empty slot" 'Ctrl-F 选择终端'
  driver_send "$SESSION" C-f
  must 5 "$label picker: Ctrl-F opens the picker from PANE" 'select a terminal'
  driver_send "$SESSION" Enter
  must 8 "$label attach: picker binds term-1 into the split" 'bound term-1'
  driver_send_literal "$SESSION" 'seq 1 300'
  driver_send "$SESSION" Enter
  sleep 0.8
  driver_send "$SESSION" PageUp
  must 5 "$label scrollback: PgUp shows the [↑N] badge" '\[↑[0-9]+\]'
  driver_send "$SESSION" y
  must 5 "$label copy: y reports success" 'copied visible screen'
  if driver_osc52 "$SESSION" >"$WORK/osc-$label.txt" 2>/dev/null && [ -s "$WORK/osc-$label.txt" ]; then
    if awk 'BEGIN{ok=1; prev=""} {if ($0 !~ /^[0-9]+$/) ok=0; if (prev != "" && $0 != prev+1) ok=0; prev=$0} END{exit ok?0:1}' "$WORK/osc-$label.txt"; then
      ok "$label copy: OSC 52 clipboard carries consecutive visible rows"
    else
      bad "$label copy: OSC 52 clipboard rows are not the visible window"
    fi
  else
    bad "$label copy: OSC 52 clipboard capture"
  fi
  driver_send "$SESSION" Escape
  sleep 0.3
  # close the split so the resize category measures the single focused pane
  driver_send "$SESSION" C-p
  sleep 0.3
  driver_send "$SESSION" x
  sleep 0.5
  driver_send "$SESSION" Escape
  sleep 0.4
  # resize category: the window changes and the inner PTY follows.
  driver_resize "$SESSION" 100 30
  sleep 0.6
  driver_send_literal "$SESSION" 'echo RS $(stty size)'
  driver_send "$SESSION" Enter
  must 8 "$label resize: shrinking the window reaches the inner PTY (26 98)" 'RS 26 98'
  driver_resize "$SESSION" 120 36
  sleep 0.6
  driver_send_literal "$SESSION" 'echo RS2 $(stty size)'
  driver_send "$SESSION" Enter
  must 8 "$label resize: restoring the window restores the inner PTY (32 118)" 'RS2 32 118'
  driver_send "$SESSION" C-q
  must 5 "$label exit: Ctrl-Q opens the host confirmation" 'Quit tui2\?'
  driver_send "$SESSION" Enter
  must 8 "$label exit: confirmation quits cleanly and restores the terminal" 'EXIT:0'
  driver_kill "$SESSION" 2>/dev/null || true
}

PHASE_START=0
PHASE_DELTA=0
run_phase() { # <driver> <fn>
  local driver="$1" fn="$2"
  TUI2_TEST_DRIVER="$driver" driver_init || return 3
  PHASE_START=$PASS
  "$fn" "$driver"
  PHASE_DELTA=$((PASS - PHASE_START))
  driver_shutdown
  return 0
}

# ------------------------------------------------------- capture consistency
normalize() { # strip trailing spaces/blank lines and live clock stamps
  sed -e 's/[[:space:]]*$//' -e 's/([0-9][0-9]:[0-9][0-9]:[0-9][0-9])/(TIME)/g' |
    awk '{line[NR]=$0} END {last=NR; while (last>0 && line[last]=="") last--; for (i=1;i<=last;i++) print line[i]}'
}

consistency_scenario() { # <prefix>
  local prefix="$1"
  driver_kill "$SESSION" 2>/dev/null || true
  # SHELL=/bin/sh keeps the inner terminal prompt deterministic across runs.
  driver_spawn "$SESSION" 120 36 \
    "bash -c 'SHELL=/bin/sh \"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo EXIT:\$?; sleep 20'"
  waitgrep 10 'select a terminal' || true
  driver_capture "$SESSION" | normalize >"$WORK/$prefix-cold.txt"
  driver_send "$SESSION" Enter
  waitgrep 8 'term-1' || true
  driver_send_literal "$SESSION" 'echo CONSIST-OK'
  driver_send "$SESSION" Enter
  waitgrep 8 'CONSIST-OK' || true
  sleep 0.4
  driver_capture "$SESSION" | normalize >"$WORK/$prefix-bound.txt"
  driver_kill "$SESSION" 2>/dev/null || true
}

# =============================================================== smoke phases
echo
echo "== smoke: forced pty driver =="
if env TUI2_TEST_DRIVER=pty bash "$ROOT/clients/tui/scripts/smoke.sh" >"$WORK/smoke-pty.log" 2>&1; then
  if grep -q 'summary: 10 passed, 0 failed' "$WORK/smoke-pty.log"; then
    ok "smoke: forced pty driver passes 10/10"
  else
    bad "smoke: forced pty driver summary ($(tail -1 "$WORK/smoke-pty.log" 2>/dev/null))"
  fi
else
  bad "smoke: forced pty driver exits non-zero"
fi

if ((HAS_TMUX)); then
  echo
  echo "== smoke: tmux driver =="
  if env TUI2_TEST_DRIVER=tmux bash "$ROOT/clients/tui/scripts/smoke.sh" >"$WORK/smoke-tmux.log" 2>&1; then
    if grep -q 'summary: 10 passed, 0 failed' "$WORK/smoke-tmux.log"; then
      ok "smoke: tmux driver passes 10/10"
    else
      bad "smoke: tmux driver summary"
    fi
  else
    bad "smoke: tmux driver exits non-zero"
  fi
fi

# no-tmux environment: a private bin with every standard tool symlinked except
# tmux (tmux can live in several standard directories), so driver auto-
# detection must fall back to the pty driver while the smoke run still builds.
MASKED_PATH=""
build_maskbin() {
  mkdir -p "$WORK/maskbin"
  local dir tool name
  for dir in /usr/local/bin /usr/bin /bin "$(dirname "$(command -v "$GO")")"; do
    [ -d "$dir" ] || continue
    for tool in "$dir"/*; do
      [ -e "$tool" ] || continue
      [ -x "$tool" ] || continue
      name="${tool##*/}"
      [ "$name" = tmux ] && continue
      ln -sf "$tool" "$WORK/maskbin/$name" 2>/dev/null || true
    done
  done
  MASKED_PATH="$WORK/maskbin"
}
build_maskbin
echo
echo "== smoke: no tmux in PATH (auto fallback to pty) =="
if env PATH="$MASKED_PATH" bash -c 'command -v tmux >/dev/null 2>&1'; then
  bad "mask: tmux is still reachable in the masked PATH"
else
  ok "mask: tmux is hidden from the smoke PATH"
  if env PATH="$MASKED_PATH" TUI2_TEST_DRIVER= TUI2_HARNESS_BIN="$WORK/tui2-harness" \
    bash "$ROOT/clients/tui/scripts/smoke.sh" >"$WORK/smoke-masked.log" 2>&1; then
    if grep -q 'summary: 10 passed, 0 failed' "$WORK/smoke-masked.log" &&
      grep -q 'driver: pty' "$WORK/smoke-masked.log"; then
      ok "smoke: auto fallback to pty passes 10/10 without tmux"
    else
      bad "smoke: auto fallback summary ($(tail -1 "$WORK/smoke-masked.log" 2>/dev/null))"
    fi
  else
    bad "smoke: auto fallback pty run exits non-zero"
  fi
fi

# ==================================================================== parity
PTY_CAT=0
TMUX_CAT=0
echo
echo "== key categories: pty driver =="
if run_phase pty key_categories; then PTY_CAT=$PHASE_DELTA; else skip "key categories: pty driver unavailable"; fi

with_isolated_tmux() {
  TUI2_TMUX=(tmux -L "$SOCK" -f /dev/null)
  key_categories "$1"
}
if ((HAS_TMUX)); then
  echo
  echo "== key categories: tmux driver =="
  if run_phase tmux with_isolated_tmux; then TMUX_CAT=$PHASE_DELTA; else skip "key categories: tmux driver unavailable"; fi
else
  skip "key categories: tmux driver (tmux not installed)"
fi

echo
echo "== capture consistency (same scenario, both drivers) =="
if ((HAS_TMUX)); then
  TUI2_TMUX=(tmux -L "$SOCK" -f /dev/null)
  TUI2_TEST_DRIVER=pty driver_init && consistency_scenario pty
  driver_shutdown
  TUI2_TEST_DRIVER=tmux driver_init && consistency_scenario tmux
  driver_shutdown
  for state in cold bound; do
    if diff -u "$WORK/pty-$state.txt" "$WORK/tmux-$state.txt" >"$WORK/diff-$state.log" 2>&1; then
      ok "consistency: $state screen is identical on both drivers"
    else
      bad "consistency: $state screen differs (see $WORK/diff-$state.log)"
    fi
  done
else
  skip "consistency: tmux comparison (tmux not installed)"
fi

# =================================================================== summary
echo
echo "== driver parity summary: $PASS passed, $FAIL failed, $SKIP skipped =="
printf 'driver  smoke          key-categories\n'
if ((HAS_TMUX)); then
  printf 'tmux    10/10          %d/13\n' "$TMUX_CAT"
else
  printf 'tmux    skipped (absent)\n'
fi
printf 'pty     10/10          %d/13\n' "$PTY_CAT"
exit "$FAIL"
