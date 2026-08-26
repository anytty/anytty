#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
mobile_dir="$repo_root/clients/mobile"
android_dir="$repo_root/clients/mobile/android"
default_store="$repo_root/clients/mobile/android/certificates/upload-keystore.jks"
default_cert="$repo_root/clients/mobile/android/certificates/upload-key.crt"
default_alias="anytty-upload"

store_file="${ANYTTY_ANDROID_UPLOAD_STORE_FILE:-$default_store}"
cert_file="${ANYTTY_ANDROID_UPLOAD_CERT_FILE:-$default_cert}"
key_alias="${ANYTTY_ANDROID_UPLOAD_KEY_ALIAS:-$default_alias}"
store_password="${ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD:-}"
key_password="${ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD:-}"

sdk_root="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-$HOME/Library/Android/sdk}}"
ndk_version="${ANYTTY_ANDROID_NDK_VERSION:-27.2.12479018}"
ndk_root="${ANDROID_NDK_ROOT:-$sdk_root/ndk/$ndk_version}"

fail() {
  echo "ERROR: $*" >&2
  exit 1
}

password_file="${store_file}.passwords"

load_password_file() {
  [[ -f "$password_file" ]] || return 0
  set -a
  # shellcheck disable=SC1090
  source "$password_file"
  set +a
}

create_password_file() {
  mkdir -p "$(dirname "$password_file")"
  store_password="${store_password:-$(openssl rand -base64 24 | tr -d '[:space:]' | tr '/+' '_-')}"
  key_password="${key_password:-$store_password}"
  umask 077
  cat > "$password_file" <<EOF
ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD='$store_password'
ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD='$key_password'
EOF
  chmod 600 "$password_file"
}

read_secret() {
  local name="$1"
  local value="$2"
  if [[ -n "$value" ]]; then
    printf '%s' "$value"
    return
  fi
  if [[ ! -t 0 ]]; then
    fail "$name is not set and stdin is not interactive"
  fi
  local secret
  read -r -s -p "Enter $name: " secret
  echo
  if [[ -z "$secret" ]]; then
    fail "$name must not be empty"
  fi
  printf '%s' "$secret"
}

for cmd in keytool jarsigner npm openssl ./gradlew; do
  if [[ "$cmd" == "./gradlew" ]]; then
    [[ -x "$android_dir/gradlew" ]] || fail "missing $android_dir/gradlew"
  else
    command -v "$cmd" >/dev/null 2>&1 || fail "missing required command: $cmd"
  fi
done

[[ -d "$sdk_root" ]] || fail "Android SDK not found at $sdk_root; set ANDROID_SDK_ROOT or ANDROID_HOME"
[[ -d "$ndk_root" ]] || fail "Android NDK $ndk_version not found at $ndk_root; set ANDROID_NDK_ROOT or install the NDK"

mkdir -p "$(dirname "$store_file")" "$(dirname "$cert_file")"

if [[ ! -f "$store_file" ]]; then
  echo "Upload keystore not found at: $store_file"
  if [[ ! -t 0 && ( -z "$store_password" || -z "$key_password" ) ]]; then
    load_password_file
    if [[ -z "$store_password" || -z "$key_password" ]]; then
      echo "Non-interactive shell detected; generating upload keystore passwords."
      create_password_file
    fi
  fi
  echo "Creating a new upload keystore..."
  store_password="$(read_secret "ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD" "$store_password")"
  key_password="$(read_secret "ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD" "$key_password")"
  keytool -genkeypair -v \
    -keystore "$store_file" \
    -storepass "$store_password" \
    -keypass "$key_password" \
    -alias "$key_alias" \
    -keyalg RSA \
    -keysize 2048 \
    -validity 10000 \
    -dname "CN=AnyTTY Upload, O=AnyTTY, L=Unknown, ST=Unknown, C=US"
else
  echo "Using existing upload keystore: $store_file"
  if [[ ! -t 0 && ( -z "$store_password" || -z "$key_password" ) ]]; then
    load_password_file
    if [[ -z "$store_password" || -z "$key_password" ]]; then
      fail "upload keystore passwords are missing; set ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD and ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD or create $password_file"
    fi
  fi
  store_password="$(read_secret "ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD" "$store_password")"
  key_password="$(read_secret "ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD" "$key_password")"
fi

if [[ -z "$store_password" || -z "$key_password" ]]; then
  fail "upload keystore passwords must not be empty"
fi

tmp_cert="$(mktemp)"
cleanup() { rm -f "$tmp_cert"; }
trap cleanup EXIT

keytool -exportcert -rfc \
  -keystore "$store_file" \
  -storepass "$store_password" \
  -alias "$key_alias" \
  -file "$tmp_cert" >/dev/null 2>&1

if [[ ! -s "$tmp_cert" ]]; then
  fail "keystore $store_file does not contain alias $key_alias"
fi

if [[ -f "$cert_file" ]] && cmp -s "$tmp_cert" "$cert_file"; then
  echo "Upload public key already present: $cert_file"
else
  if [[ -f "$cert_file" ]]; then
    echo "Replaced existing upload public key with the keystore certificate."
  fi
  cp "$tmp_cert" "$cert_file"
  echo "Exported upload public key: $cert_file"
fi

echo
echo "Before uploading to Google Play, register this upload key if it is not already registered:"
echo "  Play Console -> Release -> App signing -> Upload key -> upload $cert_file"
echo

echo "Building mobile web assets..."
(
  cd "$mobile_dir"
  npm run cap:build
)

echo "Building signed AAB..."
export ANYTTY_ANDROID_UPLOAD_STORE_FILE="$store_file"
export ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD="$store_password"
export ANYTTY_ANDROID_UPLOAD_KEY_ALIAS="$key_alias"
export ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD="$key_password"
(
  cd "$android_dir"
  ANDROID_HOME="$sdk_root" ./gradlew bundleRelease
)

aab_path="$android_dir/app/build/outputs/bundle/release/app-release.aab"
[[ -f "$aab_path" ]] || fail "AAB was not produced at $aab_path"

echo
echo "Verifying AAB signature..."
"$repo_root/scripts/verify-android-aab-boundary.sh" "$aab_path"

echo
echo "Signed AAB ready: $aab_path"
echo "Remember to keep these files private and never commit them:"
echo "  $store_file"
if [[ -f "$password_file" ]]; then
  echo "  $password_file"
fi
echo "  your upload keystore passwords"
