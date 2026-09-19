#!/usr/bin/env bash
# 用 buf 驱动 protoc-gen-dart 生成 Flutter/客户端 Dart proto。
# 依赖：buf（PATH 或 BUF_BIN）、dart、protoc-gen-dart（dart pub global activate protoc_plugin）。
# 生成物沿用历史扁平布局（apipb/、bindingpb/、remoteauthpb/、wirepb/、cloud/），
# 因此先在暂存目录重写 import 前缀，避免 Flutter 源码 import 路径变化。
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
proto_root="${repo_root}/proto"
output_root="${repo_root}/clients/flutter/lib/src/generated/proto"
buf_bin="${BUF_BIN:-buf}"
protoc_gen_dart="${PROTOC_GEN_DART:-${PUB_CACHE:-${HOME}/.pub-cache}/bin/protoc-gen-dart}"
dart_bin="${DART_BIN:-dart}"

if ! command -v "${buf_bin}" >/dev/null 2>&1 && [[ ! -x "${buf_bin}" ]]; then
  echo "buf is unavailable; install it or set BUF_BIN (e.g. go install github.com/bufbuild/buf/cmd/buf@v1.47.2)" >&2
  exit 1
fi
if [[ ! -x "${protoc_gen_dart}" ]]; then
  echo "protoc-gen-dart is unavailable; run: dart pub global activate protoc_plugin" >&2
  exit 1
fi
if ! command -v "${dart_bin}" >/dev/null 2>&1; then
  echo "dart is unavailable; install the Dart SDK or set DART_BIN" >&2
  exit 1
fi

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/anytty-dart-proto.XXXXXX")"
cleanup() { rm -rf "$work_dir"; }
trap cleanup EXIT
stage="$work_dir/stage"
mkdir -p "$stage"

cp -r "${proto_root}/access/apipb" "$stage/apipb"
cp -r "${proto_root}/access/bindingpb" "$stage/bindingpb"
cp -r "${proto_root}/access/remoteauthpb" "$stage/remoteauthpb"
cp -r "${proto_root}/access/wirepb" "$stage/wirepb"
cp -r "${proto_root}/cloud" "$stage/cloud"
find "$stage" -type f -name '*.pb.go' -delete

python3 - "$stage" <<'PY'
import glob, os, sys
stage = sys.argv[1]
for path in glob.glob(stage + "/**/*.proto", recursive=True):
    content = open(path).read()
    content = content.replace('import "access/apipb/', 'import "apipb/')
    content = content.replace('import "access/bindingpb/', 'import "bindingpb/')
    content = content.replace('import "access/remoteauthpb/', 'import "remoteauthpb/')
    content = content.replace('import "access/wirepb/', 'import "wirepb/')
    open(path, 'w').write(content)
PY

template="$work_dir/buf.gen.dart.yaml"
cat >"$template" <<YAML
version: v2
plugins:
  - local: ${protoc_gen_dart}
    out: ${output_root}
YAML

mkdir -p "${output_root}"
find "${output_root}" -type f -name '*.dart' -delete

(
  cd "$stage"
  "${buf_bin}" generate --template "$template" \
    --path apipb \
    --path bindingpb \
    --path remoteauthpb \
    --path wirepb \
    --path cloud/v1
)

find "${output_root}" -type d -empty -delete
"${dart_bin}" format "${output_root}"
