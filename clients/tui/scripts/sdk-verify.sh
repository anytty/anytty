#!/usr/bin/env bash
# TUI v2 SDK conformance for every official binding, through one standard:
# tui2-sdk-verify + clients/tui/conformance/fixtures.jsonl. Any language SDK uses the
# same command; missing interpreters are reported and skipped.
#
# Usage: bash clients/tui/scripts/sdk-verify.sh
set -u

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
GO="${GO:-go}"
PY="${PY:-python3}"
NODE="${NODE:-node}"
FIXTURES="clients/tui/conformance/fixtures.jsonl"
FAIL=0

run_sdk() { # run_sdk <label> <command...>
  local label="$1"
  shift
  echo
  echo "== $label =="
  if ! (cd "$ROOT" && "$GO" run ./clients/tui/cmd/tui2-sdk-verify --fixtures "$FIXTURES" --cmd "$*"); then
    FAIL=$((FAIL + 1))
  fi
}

echo "== go SDK (in process) =="
if ! (cd "$ROOT" && "$GO" run ./clients/tui/cmd/tui2-sdk-verify --fixtures "$FIXTURES"); then
  FAIL=$((FAIL + 1))
fi

if command -v "$PY" >/dev/null 2>&1; then
  run_sdk "python SDK" "$PY clients/tui/sdk/python/conformance.py"
else
  echo "SKIP  python SDK ($PY not installed)"
fi

if command -v "$NODE" >/dev/null 2>&1; then
  run_sdk "ts SDK" "$NODE clients/tui/sdk/ts/conformance.js"
else
  echo "SKIP  ts SDK ($NODE not installed)"
fi

echo
if ((FAIL)); then
  echo "sdk-verify.sh: $FAIL binding(s) failed"
  exit 1
fi
echo "sdk-verify.sh: all bindings passed"
