#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/anytty-install-test.XXXXXX")"
cleanup() { rm -rf "$work_dir"; }
trap cleanup EXIT

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) echo "installer test does not support $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "installer test does not support $(uname -m)" >&2; exit 1 ;;
esac

version="v9.9.9-test"
archive_base="anytty-$version-$os-$arch"
release_dir="$work_dir/release"
package_dir="$work_dir/package/$archive_base"
archive_name="$archive_base.tar.gz"
mkdir -p "$release_dir" "$package_dir"
printf '#!/usr/bin/env sh\nprintf "test anytty\\n"\n' >"$package_dir/anytty"
chmod 0755 "$package_dir/anytty"
cp "$repo_root/tui/docs/tui-v3.recommended.yaml" "$package_dir/tui-v3.yaml"
tar -C "$work_dir/package" -czf "$release_dir/$archive_name" "$archive_base"

if command -v shasum >/dev/null 2>&1; then
  checksum="$(shasum -a 256 "$release_dir/$archive_name" | awk '{print $1}')"
else
  checksum="$(sha256sum "$release_dir/$archive_name" | awk '{print $1}')"
fi
printf '%s  %s\n' "$checksum" "$archive_name" >"$release_dir/SHA256SUMS"

install_dir="$work_dir/bin"
config_home="$work_dir/config"
mock_bin="$work_dir/mock-bin"
codesign_log="$work_dir/codesign.log"
mkdir -p "$mock_bin"
cat >"$mock_bin/codesign" <<'EOF'
#!/usr/bin/env sh
printf '%s\n' "$*" >>"$ANYTTY_TEST_CODESIGN_LOG"
EOF
chmod 0755 "$mock_bin/codesign"
HOME="$work_dir/home" \
XDG_CONFIG_HOME="$config_home" \
ANYTTY_VERSION="$version" \
ANYTTY_RELEASE_BASE_URL="file://$release_dir" \
ANYTTY_TEST_CODESIGN_LOG="$codesign_log" \
PATH="$mock_bin:$PATH" \
  sh "$repo_root/install.sh" --bin-dir "$install_dir"

cmp "$package_dir/anytty" "$install_dir/anytty"
cmp "$repo_root/tui/docs/tui-v3.recommended.yaml" "$config_home/anytty/tui-v3.yaml"
if [[ "$os" == darwin ]]; then
  grep -q -- '--force --sign -' "$codesign_log"
  grep -q -- '--identifier com.anytty.cli' "$codesign_log"
  grep -q -- '--verify --strict' "$codesign_log"
else
  [[ ! -e "$codesign_log" ]]
fi

printf 'version: 1\ntui:\n  profile: keep-user-config\n' >"$config_home/anytty/tui-v3.yaml"
HOME="$work_dir/home" \
XDG_CONFIG_HOME="$config_home" \
ANYTTY_VERSION="$version" \
ANYTTY_RELEASE_BASE_URL="file://$release_dir" \
ANYTTY_TEST_CODESIGN_LOG="$codesign_log" \
PATH="$mock_bin:$PATH" \
  sh "$repo_root/install.sh" --bin-dir "$install_dir"

grep -q 'profile: keep-user-config' "$config_home/anytty/tui-v3.yaml"
