#!/usr/bin/env bash
set -euo pipefail

ios_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "${ios_root}/../../.." && pwd)"
output="${ios_root}/App/CapApp-SPM/Sources/CapApp-SPM/Generated"

command -v protoc >/dev/null || { echo "protoc is required" >&2; exit 1; }
command -v protoc-gen-swift >/dev/null || { echo "protoc-gen-swift is required" >&2; exit 1; }

mkdir -p "${output}"
rm -f "${output}"/*.pb.swift
cd "${repo_root}"
protoc -I proto --swift_out="${output}" \
  --swift_opt=Visibility=Public,FileNaming=PathToUnderscores \
  proto/apipb/access_remote.proto \
  proto/apipb/application.proto \
  proto/apipb/common.proto \
  proto/apipb/events.proto \
  proto/apipb/file.proto \
  proto/apipb/history.proto \
  proto/apipb/runtime.proto \
  proto/apipb/storage.proto \
  proto/apipb/terminal.proto \
  proto/apipb/workbench.proto \
  proto/remoteauthpb/remote_auth.proto \
  proto/bindingpb/client_binding.proto \
  proto/cloud/v1/common.proto \
  proto/cloud/v1/enrollment.proto
