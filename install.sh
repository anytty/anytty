#!/usr/bin/env sh

set -eu

repository="${ANYTTY_REPOSITORY:-anytty/anytty}"
version="${ANYTTY_VERSION:-}"
install_dir="${ANYTTY_INSTALL_DIR:-${HOME}/.local/bin}"

usage() {
  cat <<'EOF'
Install the AnyTTY CLI from GitHub Releases.

Usage: install.sh [--version VERSION] [--bin-dir DIRECTORY]

Environment:
  ANYTTY_VERSION           Release tag, for example v0.0.1-beta.0
  ANYTTY_INSTALL_DIR       Installation directory (default: ~/.local/bin)
  ANYTTY_REPOSITORY        GitHub owner/repository (default: anytty/anytty)
  ANYTTY_RELEASE_BASE_URL  Override the release download base URL
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || { echo "--version requires a value" >&2; exit 2; }
      version="$2"
      shift 2
      ;;
    --bin-dir)
      [ "$#" -ge 2 ] || { echo "--bin-dir requires a value" >&2; exit 2; }
      install_dir="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 1; }

if [ -z "$version" ]; then
  version="$(curl -fsSL "https://raw.githubusercontent.com/${repository}/main/VERSION")"
fi
version="$(printf '%s' "$version" | tr -d '[:space:]')"
case "$version" in
  v*) ;;
  *) version="v$version" ;;
esac

case "$(uname -s)" in
  Darwin) os="darwin" ;;
  Linux) os="linux" ;;
  *) echo "unsupported operating system: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

archive_base="anytty-${version}-${os}-${arch}"
archive_name="${archive_base}.tar.gz"
release_base="${ANYTTY_RELEASE_BASE_URL:-https://github.com/${repository}/releases/download/${version}}"
work_dir="$(mktemp -d "${TMPDIR:-/tmp}/anytty-install.XXXXXX")"
cleanup() { rm -rf "$work_dir"; }
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

curl -fL --retry 3 --retry-delay 1 -o "$work_dir/$archive_name" "$release_base/$archive_name"
curl -fL --retry 3 --retry-delay 1 -o "$work_dir/SHA256SUMS" "$release_base/SHA256SUMS"

expected="$(awk -v name="$archive_name" '$2 == name { print $1; exit }' "$work_dir/SHA256SUMS")"
[ -n "$expected" ] || { echo "checksum not found for $archive_name" >&2; exit 1; }
if command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$work_dir/$archive_name" | awk '{print $1}')"
elif command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$work_dir/$archive_name" | awk '{print $1}')"
else
  echo "shasum or sha256sum is required" >&2
  exit 1
fi
[ "$actual" = "$expected" ] || { echo "checksum verification failed for $archive_name" >&2; exit 1; }

tar -xzf "$work_dir/$archive_name" -C "$work_dir"
[ -f "$work_dir/$archive_base/anytty" ] || { echo "release archive does not contain anytty" >&2; exit 1; }
mkdir -p "$install_dir"
cp "$work_dir/$archive_base/anytty" "$install_dir/anytty"
chmod 0755 "$install_dir/anytty"

printf 'Installed AnyTTY %s to %s/anytty\n' "$version" "$install_dir"
case ":${PATH:-}:" in
  *":$install_dir:"*) ;;
  *) printf 'Add %s to PATH to run anytty from any directory.\n' "$install_dir" ;;
esac
