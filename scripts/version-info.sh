#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
version_file="${ANYTTY_VERSION_FILE:-$repo_root/VERSION}"
version="$(tr -d '[:space:]' <"$version_file")"

if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?\+[0-9]+$ ]]; then
  echo "invalid release version in $version_file: $version" >&2
  echo "expected: X.Y.Z[-PRERELEASE]+BUILD" >&2
  exit 1
fi

release_version="${version%%+*}"
build_number="${version##*+}"
release_tag="v$release_version"

printf 'ANYTTY_VERSION=%q\n' "$version"
printf 'ANYTTY_RELEASE_VERSION=%q\n' "$release_version"
printf 'ANYTTY_BUILD_NUMBER=%q\n' "$build_number"
printf 'ANYTTY_RELEASE_TAG=%q\n' "$release_tag"
