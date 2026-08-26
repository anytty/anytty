#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
version="${1:-}"
asset_dir="${2:-$repo_root/.artifacts/release/$version}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "usage: scripts/write-release-metadata.sh vX.Y.Z[-PRERELEASE] [ASSET_DIRECTORY]" >&2
  exit 2
fi
if [[ ! -d "$asset_dir" ]]; then
  echo "release asset directory does not exist: $asset_dir" >&2
  exit 1
fi

source_version="$(tr -d '[:space:]' <"$repo_root/VERSION")"
[[ "$source_version" == "$version" ]] || {
  echo "release tag $version does not match VERSION $source_version" >&2
  exit 1
}

package_version="$(node -e 'const p=require(process.argv[1]); process.stdout.write(p.version)' "$repo_root/package.json")"
[[ "v$package_version" == "$version" ]] || {
  echo "release tag $version does not match package.json v$package_version" >&2
  exit 1
}

commit="$(git -C "$repo_root" rev-parse HEAD)"
build_date="$(date -u +%Y-%m-%d)"
build_info="$asset_dir/BUILD_INFO.txt"
cat >"$build_info" <<EOF
AnyTTY $version

Source: https://github.com/anytty/anytty
Commit: $commit
Build date: $build_date

CLI targets:
- darwin/amd64
- darwin/arm64
- linux/amd64
- linux/arm64
- windows/amd64
- windows/arm64
EOF

android_assets=(
  "anytty-$version-android-armeabi-v7a.apk"
  "anytty-$version-android-arm64-v8a.apk"
  "anytty-$version-android-x86_64.apk"
  "anytty-$version-android-universal.apk"
  "anytty-$version-android-play.aab"
)
android_assets_present=0
for android_asset in "${android_assets[@]}"; do
  if [[ -f "$asset_dir/$android_asset" ]]; then
    android_assets_present=$((android_assets_present + 1))
  fi
done
if (( android_assets_present > 0 && android_assets_present != ${#android_assets[@]} )); then
  echo "Android release assets are incomplete" >&2
  exit 1
fi
if (( android_assets_present == ${#android_assets[@]} )); then
  cat >>"$build_info" <<'EOF'

Android:
- Package: com.anytty.app
- Minimum SDK: 24
- Target SDK: 36
- APKs: armeabi-v7a, arm64-v8a, x86_64, universal
- Play bundle: signed AAB with ABI splits enabled
- ABIs: armeabi-v7a, arm64-v8a, x86_64
- Signing: Android upload key
EOF
fi

(
  cd "$asset_dir"
  files=(anytty-* BUILD_INFO.txt)
  (( ${#files[@]} > 1 )) || { echo "release assets are missing" >&2; exit 1; }
  LC_ALL=C shasum -a 256 "${files[@]}" >SHA256SUMS
)

printf '%s\n' "$asset_dir"
