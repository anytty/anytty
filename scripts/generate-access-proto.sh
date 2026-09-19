#!/usr/bin/env bash
# generate-access-proto.sh — regenerate protoc-gen-go outputs for the access
# protobuf trees used by this repository.
#
# Why staging is required:
#   The checked-in .pb.go files use a historical logical layout: sources are
#   recorded as "apipb/file.proto", "wirepb/terminal.proto",
#   "remoteauthpb/remote_auth.proto", while the .proto files on disk live under
#   "proto/access/..." and import each other as "access/apipb/...". This script
#   copies the trees into a temporary staging root using the historical logical
#   paths, rewrites the "access/" import prefix, and runs buf there. Keeping the
#   logical paths stable is what keeps file_apipb_file_proto_init() and all
#   cross-file init calls unchanged.
#
# Toolchain:
#   buf v1.47.2 and protoc-gen-go v1.36.11. If they are missing the script
#   installs them through the GOPROXY mirror used by this environment
#   (proxy.golang.org DNS is broken here, do not use the default proxy).
#
# Usage:
#   scripts/generate-access-proto.sh apipb/file.proto [wirepb/terminal.proto provider/v1/provider.proto ...]
#
# Notes:
#   - Generate only the files you modified; the staging tree still compiles all
#     dependencies for correct resolution.
#   - The existing target header's `protoc` version line is preserved so diffs
#     stay limited to real schema changes (not the builder version).
#   - The embedded raw descriptor will contain the current go_package
#     (.../proto/access/apipb); the checked-in files predate the directory move,
#     so rawDesc byte changes are expected and reviewed as part of the diff.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [[ $# -eq 0 ]]; then
  echo "usage: scripts/generate-access-proto.sh <module/relative.proto> [...]" >&2
  echo "example: scripts/generate-access-proto.sh apipb/file.proto wirepb/terminal.proto" >&2
  exit 2
fi

bin_dir="${ANYTTY_PROTO_BIN:-/tmp/opencode/bin}"
mkdir -p "$bin_dir"
export PATH="$bin_dir:$PATH"

if [[ ! -x "$bin_dir/buf" ]]; then
  echo "installing buf v1.47.2 into $bin_dir" >&2
  GOWORK=off GOFLAGS= GOPROXY=https://goproxy.cn,direct GOBIN="$bin_dir" \
    go install github.com/bufbuild/buf/cmd/buf@v1.47.2
fi
if [[ ! -x "$bin_dir/protoc-gen-go" ]]; then
  echo "installing protoc-gen-go v1.36.11 into $bin_dir" >&2
  GOWORK=off GOFLAGS= GOPROXY=https://goproxy.cn,direct GOBIN="$bin_dir" \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
fi

stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT

# Stage with the historical logical paths: apipb/, remoteauthpb/, wirepb/, cloud/.
cp -R proto/access/apipb "$stage/apipb"
cp -R proto/access/remoteauthpb "$stage/remoteauthpb"
cp -R proto/access/wirepb "$stage/wirepb"
cp -R proto/cloud "$stage/cloud"
cp -R proto/provider "$stage/provider"
# Rewrite the on-disk import prefix to the staged logical prefix.
find "$stage/apipb" -name '*.proto' -print0 | xargs -0 sed -i \
  -e 's#import "access/apipb/#import "apipb/#g' \
  -e 's#import "access/remoteauthpb/#import "remoteauthpb/#g'

cp scripts/buf.gen.access.yaml "$stage/buf.gen.access.yaml"

args=(generate --template "$stage/buf.gen.access.yaml")
for proto_path in "$@"; do
  if [[ ! -f "$stage/$proto_path" ]]; then
    echo "unknown proto path: $proto_path" >&2
    exit 2
  fi
  args+=(--path "$proto_path")
done

(cd "$stage" && buf "${args[@]}")

for proto_path in "$@"; do
  generated="${proto_path%.proto}.pb.go"
  case "$proto_path" in
    provider/*) target="proto/$generated" ;;
    *) target="proto/access/$generated" ;;
  esac
  if [[ ! -f "$stage/$generated" ]]; then
    echo "buf did not generate $generated" >&2
    exit 1
  fi
  # Preserve the recorded protoc builder version of the checked-in file so the
  # diff only reflects schema changes; new files get a stable default version.
  existing_version="$(grep -m1 'protoc        ' "$target" 2>/dev/null || true)"
  if [[ -z "$existing_version" || "$existing_version" == *"(unknown)"* ]]; then
    existing_version="// \tprotoc        ${ANYTTY_PROTO_PROTOC_VERSION:-v7.36.0}"
  fi
  tmp="$(mktemp)"
  sed "s#^//[[:space:]]*protoc[[:space:]].*#${existing_version}#" "$stage/$generated" > "$tmp"
  mv "$tmp" "$stage/$generated"
  mkdir -p "$(dirname "$target")"
  cp "$stage/$generated" "$target"
  echo "generated $target"
done
