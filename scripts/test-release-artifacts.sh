#!/usr/bin/env bash
# 发布产物 smoke：只构建当前主机的 release 归档，验证归档内容、
# 三件套可执行以及隔离的 daemon+access 生命周期（不触碰用户环境）。
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) echo "release artifact smoke does not support $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "release artifact smoke does not support $(uname -m)" >&2; exit 1 ;;
esac

work_dir="$(mktemp -d "${TMPDIR:-/tmp}/anytty-release-smoke.XXXXXX")"
cleanup() { rm -rf "$work_dir"; }
trap cleanup EXIT

version="v0.0.0-smoke"
out_dir="$work_dir/release"
ANYTTY_RELEASE_TARGETS="$os/$arch" bash "$repo_root/scripts/build-release-artifacts.sh" "$version" "$out_dir"

archive_base="anytty-$version-$os-$arch"
tar -xzf "$out_dir/$archive_base.tar.gz" -C "$work_dir"
package_dir="$work_dir/$archive_base"

for binary in anytty tui2 tui2-shell; do
  [ -x "$package_dir/$binary" ] || { echo "release archive is missing $binary" >&2; exit 1; }
done
"$package_dir/anytty" --version >/dev/null
"$package_dir/tui2" -version >/dev/null
"$package_dir/tui2-shell" -version >/dev/null

export XDG_STATE_HOME="$work_dir/state"
export XDG_CONFIG_HOME="$work_dir/config"
export XDG_RUNTIME_DIR="$work_dir/run"
mkdir -p "$XDG_STATE_HOME" "$XDG_CONFIG_HOME" "$XDG_RUNTIME_DIR"
socket="$XDG_RUNTIME_DIR/anytty-v2-wire7.sock"
log_file="$XDG_STATE_HOME/anytty/anytty.log"

"$package_dir/anytty" --socket "$socket" --log-file "$log_file" daemon start
"$package_dir/anytty" --socket "$socket" --log-file "$log_file" daemon status
"$package_dir/anytty" --socket "$socket" --log-file "$log_file" daemon stop

echo "release artifact smoke ok: $archive_base (anytty, tui2, tui2-shell + daemon/access lifecycle)"
