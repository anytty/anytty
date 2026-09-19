#!/usr/bin/env bash
# tui2 dev loop: build the host, start a layout program with -dev (frame log +
# crash notices) and hot reload on save.
#
# Usage:
#   bash clients/tui/scripts/dev.sh [--log <file>] [<target>] [-- <host flags>]
#
# <target> is one of (case-insensitive):
#   shell | go        the default tui2-shell (default: shell)
#   python | ts       clients/tui/templates/{python,ts}
#   <dir|file>        any layout program: .go builds via `go build`,
#                     .py runs with python3, .js runs with node; directories
#                     are resolved to main.go/program.py/program.js
#
# Examples:
#   bash clients/tui/scripts/dev.sh
#   bash clients/tui/scripts/dev.sh python
#   bash clients/tui/scripts/dev.sh ts -- -routes local-unix
#   bash clients/tui/scripts/dev.sh /tmp/my-program.py
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "$ROOT"
GO="${GO:-go}"
PY="${PY:-python3}"
NODE="${NODE:-node}"
if ! command -v "$GO" >/dev/null 2>&1; then
  for candidate in "$HOME"/.local/share/go-toolchains/*/bin /usr/local/go/bin; do
    if [[ -x "$candidate/go" ]]; then
      export PATH="$candidate:$PATH"
      break
    fi
  done
fi
if ! command -v "$GO" >/dev/null 2>&1; then
  echo "dev.sh: go not found (set GO=/path/to/go)" >&2
  exit 1
fi
export GOTOOLCHAIN="${GOTOOLCHAIN:-local}"
BIN_DIR="${TUI2_BIN_DIR:-$ROOT/clients/tui/.bin}"
mkdir -p "$BIN_DIR"

LOG_FLAG=()
if [[ "${1:-}" == "--log" ]]; then
  LOG_FLAG=(-protocol-log "$2")
  shift 2
fi
TARGET="${1:-shell}"
if [[ $# -gt 0 ]]; then shift; fi

case "$(basename "$TARGET" | tr '[:upper:]' '[:lower:]')" in
  shell|go|go-*) ;;
  python) TARGET="$ROOT/clients/tui/templates/python" ;;
  ts|js) TARGET="$ROOT/clients/tui/templates/ts" ;;
esac
if [[ -d "$TARGET" ]]; then
  for name in main.go program.py program.js; do
    [[ -f "$TARGET/$name" ]] && TARGET="$TARGET/$name" && break
  done
fi

WATCH=()
SHELL_CMD=""
case "$TARGET" in
  shell|go|go-*)
    echo "dev.sh: building tui2 and tui2-shell ..." >&2
    "$GO" build -o "$BIN_DIR/tui2" ./clients/tui/cmd/tui2
    "$GO" build -o "$BIN_DIR/tui2-shell" ./clients/tui/cmd/tui2-shell
    SHELL_CMD="$BIN_DIR/tui2-shell"
    ;;
  *.go)
    echo "dev.sh: building $TARGET ..." >&2
    "$GO" build -o "$BIN_DIR/dev-program" "$TARGET"
    SHELL_CMD="$BIN_DIR/dev-program"
    WATCH=(-watch "$(dirname "$(realpath "$TARGET")")")
    ;;
  *.py)
    SHELL_CMD="$PY $(realpath "$TARGET")"
    WATCH=(-watch "$(realpath "$TARGET")")
    ;;
  *.js)
    SHELL_CMD="$NODE $(realpath "$TARGET")"
    WATCH=(-watch "$(realpath "$TARGET")")
    ;;
  *)
    echo "dev.sh: target not found: $TARGET" >&2
    exit 2
    ;;
esac
if [[ ! -x "$BIN_DIR/tui2" ]]; then
  "$GO" build -o "$BIN_DIR/tui2" ./clients/tui/cmd/tui2
fi

LOG_PATH="${XDG_STATE_HOME:-$HOME/.local/state}/anytty/tui2-dev.log"
echo "dev.sh: -shell $SHELL_CMD" >&2
echo "dev.sh: protocol log $LOG_PATH" >&2
echo "dev.sh: hot reload ${WATCH[*]:-(off; pass a template/file or -watch)}" >&2
exec "$BIN_DIR/tui2" -dev ${LOG_FLAG[@]+"${LOG_FLAG[@]}"} ${WATCH[@]+"${WATCH[@]}"} -shell "$SHELL_CMD" "$@"
