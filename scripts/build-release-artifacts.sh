#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
version="${1:-}"
output_dir="${2:-$repo_root/.artifacts/release/$version}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "usage: scripts/build-release-artifacts.sh vX.Y.Z[-PRERELEASE] [OUTPUT_DIRECTORY]" >&2
  exit 2
fi
if [[ -e "$output_dir" ]]; then
  echo "release output already exists: $output_dir" >&2
  exit 1
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/anytty-release.XXXXXX")"
cleanup() { rm -rf "$work_dir"; }
trap cleanup EXIT
mkdir -p "$output_dir"

targets=(
  darwin/amd64
  darwin/arm64
  linux/amd64
  linux/arm64
  windows/amd64
  windows/arm64
)

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  artifact_base="anytty-$version-$goos-$goarch"
  package_dir="$work_dir/$artifact_base"
  binary_name="anytty"
  [[ "$goos" == windows ]] && binary_name="anytty.exe"
  mkdir -p "$package_dir"

  (
    cd "$repo_root"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOWORK=off \
      go build -trimpath -ldflags="-s -w -X main.version=$version" \
      -o "$package_dir/$binary_name" ./cmd/anytty
  )
  install -m 0644 "$repo_root/LICENSE" "$package_dir/LICENSE"
  install -m 0644 "$repo_root/NOTICE" "$package_dir/NOTICE"
  install -m 0644 "$repo_root/cmd/anytty/THIRD_PARTY_NOTICES.txt" "$package_dir/THIRD_PARTY_NOTICES.txt"

  if [[ "$goos" == windows ]]; then
    (cd "$work_dir" && zip -q -r "$output_dir/$artifact_base.zip" "$artifact_base")
  else
    tar -C "$work_dir" -czf "$output_dir/$artifact_base.tar.gz" "$artifact_base"
  fi
  rm -rf "$package_dir"
done

(
  cd "$output_dir"
  LC_ALL=C shasum -a 256 anytty-* >SHA256SUMS
)
printf '%s\n' "$output_dir"
