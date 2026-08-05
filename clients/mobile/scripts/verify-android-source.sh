#!/usr/bin/env bash
set -euo pipefail
exec node "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/verify-android-source.mjs"
