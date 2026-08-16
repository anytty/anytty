#!/usr/bin/env bash
set -euo pipefail

ios_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
repo_root="$(cd "${ios_root}/../../.." && pwd)"
source_dir="${repo_root}/clients/mobile/dist"
target_dir="${ios_root}/App/App/public"

if [[ ! -f "${source_dir}/index.html" ]]; then
  echo "Mobile web output is unavailable. Run the mobile build first." >&2
  exit 1
fi

case "${target_dir}" in
  "${ios_root}"/App/App/public) ;;
  *) echo "Refusing to sync to an unexpected path: ${target_dir}" >&2; exit 1 ;;
esac

mkdir -p "${target_dir}"
rsync -a --delete "${source_dir}/" "${target_dir}/"
