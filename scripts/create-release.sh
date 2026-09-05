#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
if [[ -n "${1:-}" ]]; then
  version="$1"
else
  eval "$("$repo_root/scripts/version-info.sh")"
  version="$ANYTTY_RELEASE_TAG"
fi
asset_dir="${2:-$repo_root/.artifacts/release/$version}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "usage: scripts/create-release.sh vX.Y.Z[-PRERELEASE] [ASSET_DIRECTORY]" >&2
  exit 2
fi
for asset in \
  "anytty-$version-darwin-amd64.tar.gz" \
  "anytty-$version-darwin-arm64.tar.gz" \
  "anytty-$version-linux-amd64.tar.gz" \
  "anytty-$version-linux-arm64.tar.gz" \
  "anytty-$version-windows-amd64.zip" \
  "anytty-$version-windows-arm64.zip" \
  "anytty-$version-android-armeabi-v7a.apk" \
  "anytty-$version-android-arm64-v8a.apk" \
  "anytty-$version-android-x86_64.apk" \
  "anytty-$version-android-universal.apk" \
  "anytty-$version-android-play.aab" \
  BUILD_INFO.txt SHA256SUMS; do
  [[ -s "$asset_dir/$asset" ]] || { echo "release asset is missing: $asset_dir/$asset" >&2; exit 1; }
done

gh auth status >/dev/null 2>&1 || { echo "gh is not authenticated" >&2; exit 1; }
if gh release view "$version" --repo anytty/anytty >/dev/null 2>&1; then
  echo "release already exists: https://github.com/anytty/anytty/releases/tag/$version" >&2
  exit 1
fi

release_args=(
  "$version"
  "$asset_dir"/*
  --repo anytty/anytty
  --verify-tag
  --title "AnyTTY $version"
)
notes_file="${ANYTTY_RELEASE_NOTES_FILE:-$repo_root/.github/release-notes/$version.md}"
if [[ -s "$notes_file" ]]; then
  release_args+=(--notes-file "$notes_file")
else
  release_args+=(--generate-notes)
fi
if [[ "$version" == *-* ]]; then
  release_args+=(--prerelease)
fi
gh release create "${release_args[@]}"
