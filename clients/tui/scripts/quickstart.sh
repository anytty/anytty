#!/usr/bin/env bash
# quickstart.sh — one-shot build + verify (+ run) for the tui2 / herdr work.
#
# Usage (from anywhere; run with bash >= 3.2):
#   bash clients/tui/scripts/quickstart.sh                 # build + all tests
#   bash clients/tui/scripts/quickstart.sh --fast          # build + Go tests + conformance only
#   bash clients/tui/scripts/quickstart.sh --run           # ... then launch herdr
#   bash clients/tui/scripts/quickstart.sh --run --with-credentials
#   bash clients/tui/scripts/quickstart.sh --acceptance    # also run the (long) acceptance suite
#   bash clients/tui/scripts/quickstart.sh --stop          # stop the quickstart stack + drop stale sockets
#
# Why an isolated XDG_STATE_HOME: an older anytty daemon may own the shared
# access store lock, so a v3 stack on the default state directory cannot start.
# The quickstart stack lives in its own state dir and never touches it.
#
# Binaries are built into $OUT (default: /tmp/anytty-quickstart) so the repo
# stays clean; nothing here overwrites an installed anytty.

set -u

ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
OUT="${ANYTTY_QUICKSTART_OUT:-${TMPDIR:-/tmp}}"
OUT="${OUT%/}/anytty-quickstart"
STATE="${ANYTTY_QUICKSTART_STATE:-$HOME/.local/state/anytty-v3-herdr}"
# The canonical socket path comes from XDG_RUNTIME_DIR (fallback TMPDIR), not
# from XDG_STATE_HOME, so both must be isolated or the run would attach to a
# stack (and persisted layout) owned by another instance.
RUNTIME_DIR="${ANYTTY_QUICKSTART_RUNTIME:-$STATE/runtime}"

RUN=0
FAST=0
STOP=0
ACCEPTANCE=0
WITH_CREDENTIALS=0
for arg in "$@"; do
  case "$arg" in
    --run) RUN=1 ;;
    --fast) FAST=1 ;;
    --stop) STOP=1 ;;
    --acceptance) ACCEPTANCE=1 ;;
    --with-credentials) WITH_CREDENTIALS=1 ;;
    -h | --help)
      sed -n '2,20p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "unknown argument: $arg (see --help)" >&2
      exit 2
      ;;
  esac
done

# --- toolchain ---------------------------------------------------------------
if ! command -v go >/dev/null 2>&1; then
  for candidate in "$HOME/.local/share/go-toolchains/go1.26.7/bin" /opt/homebrew/bin /usr/local/bin; do
    if [ -x "$candidate/go" ]; then
      PATH="$candidate:$PATH"
      export PATH
      break
    fi
  done
fi
if ! command -v go >/dev/null 2>&1; then
  echo "FAIL  go not found (set GO=/path/to/go or add it to PATH)" >&2
  exit 1
fi

mkdir -p "$OUT"
FAILED=0

step() {
  printf '\n== %s ==\n' "$1"
}

run_step() {
  name="$1"
  shift
  step "$name"
  if "$@"; then
    echo "ok    $name"
  else
    echo "FAIL  $name"
    FAILED=1
  fi
}

clean_stale_provider() {
  runtime_dir="${TMPDIR:-/tmp}/anytty-$(id -u)"
  socket="$runtime_dir/anytty-v3-wire7.sock.provider"
  if [ ! -e "$socket" ]; then
    return 0
  fi
  if ps ax -o command= | grep -q 'anytty.*v3-wire7'; then
    return 0
  fi
  rm -f "$socket"
  echo "      removed stale provider socket $socket"
}

# --- stop mode ---------------------------------------------------------------
if [ "$STOP" = 1 ]; then
  step "stop quickstart stack ($STATE)"
  if [ -x "$OUT/anytty" ]; then
    XDG_STATE_HOME="$STATE" XDG_RUNTIME_DIR="$RUNTIME_DIR" "$OUT/anytty" access stop >/dev/null 2>&1 || true
    XDG_STATE_HOME="$STATE" XDG_RUNTIME_DIR="$RUNTIME_DIR" "$OUT/anytty" pool stop >/dev/null 2>&1 || true
  fi
  clean_stale_provider
  echo "ok    quickstart stack stopped (your other anytty daemons are untouched)"
  exit 0
fi

# --- build -------------------------------------------------------------------
run_step "build tui2" sh -c "cd '$ROOT' && go build -o '$OUT/tui2' ./clients/tui/cmd/tui2"
run_step "build herdr" sh -c "cd '$ROOT' && go build -o '$OUT/herdr' ./clients/tui/examples/herdr"
run_step "build anytty" sh -c "cd '$ROOT' && go build -o '$OUT/anytty' ./cmd/anytty"
if [ "$FAILED" = 1 ]; then
  echo
  echo "build failed; fix that before running tests" >&2
  exit 1
fi

# --- tests -------------------------------------------------------------------
run_step "go: clients/tui + proto/ui" sh -c "cd '$ROOT' && TMPDIR=/tmp ANYTTY_ALLOW_NESTED=1 go test -count=1 ./clients/tui/... ./proto/ui/..."
run_step "conformance: go + python + ts" sh -c "cd '$ROOT' && bash clients/tui/scripts/sdk-verify.sh"

if [ "$FAST" = 0 ]; then
  run_step "python widget tests" sh -c "cd '$ROOT' && PYTHONPATH=clients/tui/sdk/python python3 -m unittest discover -s clients/tui/sdk/python/tests -t clients/tui/sdk/python -q"
  run_step "ts widget tests" sh -c "cd '$ROOT' && node --test clients/tui/sdk/ts/test/widgets_visual.test.js clients/tui/sdk/ts/test/widgets_content.test.js"
  run_step "v3 pixel parity (python-shell)" sh -c "cd '$ROOT' && PYTHONPATH=clients/tui/sdk/python:clients/tui/examples/python-shell python3 clients/tui/examples/python-shell/v3_parity_test.py"
fi

if [ "$ACCEPTANCE" = 1 ]; then
  if [ "${BASH_VERSINFO[0]:-0}" -ge 4 ]; then
    run_step "acceptance suite" sh -c "cd '$ROOT' && TMPDIR=/tmp bash clients/tui/scripts/acceptance.sh"
  else
    step "acceptance suite"
    echo "SKIP  needs bash >= 4 (coproc); rerun with /opt/homebrew/bin/bash"
  fi
fi

# --- summary -----------------------------------------------------------------
step "summary"
if [ "$FAILED" = 1 ]; then
  echo "FAILED — see the failing step(s) above"
else
  echo "all checks passed"
  echo "binaries: $OUT/{tui2,herdr,anytty}"
fi

# --- run ---------------------------------------------------------------------
if [ "$RUN" = 1 ] && [ "$FAILED" = 0 ]; then
  mkdir -p "$STATE" "$RUNTIME_DIR"
  chmod 700 "$RUNTIME_DIR" 2>/dev/null || true
  if [ "$WITH_CREDENTIALS" = 1 ]; then
    mkdir -p "$STATE/anytty/remote-v2"
    ln -sfn "$HOME/.local/state/anytty/remote-v2/credentials" "$STATE/anytty/remote-v2/credentials"
    echo "linked real credentials into $STATE"
  fi
  clean_stale_provider
  step "launching herdr (q or ctrl-q to quit; the host asks to confirm)"
  echo "state:   $STATE"
  echo "runtime: $RUNTIME_DIR"
  echo "routes:  all configured kinds race by default; narrow with TUI2_ROUTES=local-unix,ssh"
  echo
  XDG_STATE_HOME="$STATE" XDG_RUNTIME_DIR="$RUNTIME_DIR" TUI2_SHELL="$OUT/herdr" exec "$OUT/anytty"
fi

exit "$FAILED"
