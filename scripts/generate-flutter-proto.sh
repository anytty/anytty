#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
proto_root="${repo_root}/proto"
output_root="${repo_root}/clients/flutter/lib/src/generated/proto"
protoc_gen_dart="${PUB_CACHE:-${HOME}/.pub-cache}/bin/protoc-gen-dart"

if [[ ! -x "${protoc_gen_dart}" ]]; then
  echo "protoc-gen-dart is unavailable; run: dart pub global activate protoc_plugin" >&2
  exit 1
fi

mkdir -p "${output_root}"
find "${output_root}" -type f -name '*.dart' -delete

mapfile_command=(find "${proto_root}" -type f -name '*.proto' -print0)
proto_files=()
while IFS= read -r -d '' file; do
  proto_files+=("${file}")
done < <("${mapfile_command[@]}")

PATH="$(dirname "${protoc_gen_dart}"):${PATH}" protoc \
  --proto_path="${proto_root}" \
  --dart_out="${output_root}" \
  "${proto_files[@]}"

dart format "${output_root}"
