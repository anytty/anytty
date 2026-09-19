#!/usr/bin/env bash
# tui2 end-to-end smoke test for the local single-machine path.
# Verifies the SCENARIOS §13 subset that M4 requires: the interface appears,
# a terminal can be created and typed into, splits and the picker work, and
# Ctrl-Q exits cleanly with the terminal restored.
#
# The test runs on the shared driver layer (scripts/libdriver.sh): tmux is the
# preferred driver when installed, otherwise the built-in Go/PTY harness runs
# the exact same assertions. Force one with TUI2_TEST_DRIVER=tmux|pty.
#
# Usage: clients/tui/scripts/smoke.sh   (requires go; tmux optional)
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
# shellcheck source=libdriver.sh
source "$ROOT/clients/tui/scripts/libdriver.sh"

SESSION="tui2-smoke-$$"
WORK="$(mktemp -d)"
# The host loads the shared CLI endpoint registry at startup (M2): isolate the
# XDG tree so a developer's paired endpoints never leak into the smoke run.
export XDG_CONFIG_HOME="$WORK/xdg/config"
export XDG_STATE_HOME="$WORK/xdg/state"
export XDG_RUNTIME_DIR="$WORK/xdg/run"
mkdir -p "$XDG_CONFIG_HOME" "$XDG_STATE_HOME" "$XDG_RUNTIME_DIR"
# Keep the panes full width and exercise the program-side config file.
cat >"$WORK/tui2-config.json" <<'CFG'
{ "sidebar": false }
CFG
export ANYTTY_TUI2_CONFIG="$WORK/tui2-config.json"
PASS=0
FAIL=0

cleanup() {
  driver_kill "$SESSION" 2>/dev/null || true
  driver_shutdown
  rm -rf "$WORK"
}
trap cleanup EXIT

ok() { printf 'PASS  %s\n' "$1"; PASS=$((PASS + 1)); }
bad() { printf 'FAIL  %s\n' "$1"; FAIL=$((FAIL + 1)); }

GO="${GO:-go}"
if ! command -v "$GO" >/dev/null 2>&1 && [ -z "${TUI2_HARNESS_BIN:-}" ]; then
  echo "SKIP  go not installed (set GO=/path/to/go or TUI2_HARNESS_BIN=...)"
  exit 0
fi

echo "== building tui2 and tui2-shell =="
if ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2" ./clients/tui/cmd/tui2); then
  bad "$GO build ./clients/tui/cmd/tui2"
  exit 1
fi
if ! (cd "$ROOT" && "$GO" build -o "$WORK/tui2-shell" ./clients/tui/cmd/tui2-shell); then
  bad "$GO build ./clients/tui/cmd/tui2-shell"
  exit 1
fi

driver_init || exit 0

driver_kill "$SESSION" 2>/dev/null || true
driver_spawn "$SESSION" 120 36 \
  "bash -c '\"$WORK/tui2\" -shell \"$WORK/tui2-shell\"; echo EXIT:\$?; sleep 30'"

capture() { driver_capture "$SESSION"; }

wait_for() { # wait_for <seconds> <grep -E pattern>
  local deadline=$((SECONDS + $1))
  while ((SECONDS < deadline)); do
    if capture | grep -Eq "$2"; then
      return 0
    fi
    sleep 0.2
  done
  return 1
}

send() { driver_send "$SESSION" "$@"; }
sendl() { driver_send_literal "$SESSION" "$@"; }

echo "== smoke: local single-machine session =="

# 1. cold start: no terminal, so the picker opens by itself.
if wait_for 8 'select a terminal'; then
  if capture | grep -Eq 'New terminal'; then
    ok "cold start opens the terminal picker"
  else
    bad "cold start opens the terminal picker"
  fi
else
  bad "cold start opens the terminal picker"
fi

# 2. Enter creates a terminal through the picker and binds it (its title is
# painted in the component border).
send Enter
if wait_for 8 'term-1'; then
  ok "picker enter creates and binds a terminal"
else
  bad "picker enter creates and binds a terminal"
fi

# 3. typing reaches the focused PTY and the output is painted.
sendl 'echo hi'
send Enter
if wait_for 8 'echo hi'; then
  ok "typing 'echo hi' reaches the terminal"
else
  bad "typing 'echo hi' reaches the terminal"
fi
# The output line "hi" is distinct from the echoed command line "echo hi".
deadline=$((SECONDS + 8))
hi_lines=0
while ((SECONDS < deadline)); do
  hi_lines="$(capture | grep -c 'hi')"
  ((hi_lines >= 2)) && break
  sleep 0.2
done
if ((hi_lines >= 2)); then
  ok "terminal output 'hi' is painted"
else
  bad "terminal output 'hi' is painted"
fi

# 4. Ctrl-P prefix + % splits side by side into a second empty slot.
send C-p
sleep 0.3
send '%'
if wait_for 8 'Ctrl-F 选择终端'; then
  ok "Ctrl-P then % shows a second empty slot"
else
  bad "Ctrl-P then % shows a second empty slot"
fi
if wait_for 3 'PANE'; then
  ok "footer switches to PANE"
else
  bad "footer switches to PANE"
fi

# 5. Ctrl-F opens the picker from PANE.
send C-f
if wait_for 8 'select a terminal'; then
  ok "Ctrl-F opens the picker"
else
  bad "Ctrl-F opens the picker"
fi

# 6. Esc closes the picker back to NORMAL (the recommended footer badge).
send Escape
sleep 0.5
if capture | grep -Eq '󰌌 CTRL'; then
  if capture | grep -Eq 'select a terminal'; then
    bad "Esc closes the picker back to NORMAL"
  else
    ok "Esc closes the picker back to NORMAL"
  fi
else
  bad "Esc closes the picker back to NORMAL"
fi

# 7. Ctrl-Q asks for confirmation, Enter quits and the terminal is restored.
send C-q
if wait_for 8 'Quit tui2\?'; then
  ok "Ctrl-Q opens the host confirmation"
else
  bad "Ctrl-Q opens the host confirmation"
fi
send Enter
if wait_for 8 'EXIT:0'; then
  ok "confirmation quits cleanly and restores the terminal"
else
  bad "confirmation quits cleanly and restores the terminal"
fi

echo "== summary: $PASS passed, $FAIL failed (driver: $DRIVER) =="
exit "$FAIL"
