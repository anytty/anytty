#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
eval "$("$repo_root/scripts/version-info.sh")"

VERSION_VALUE="$ANYTTY_VERSION" \
RELEASE_VALUE="$ANYTTY_RELEASE_VERSION" \
node - "$repo_root/package.json" "$repo_root/package-lock.json" <<'NODE'
const fs = require('fs');

for (const file of process.argv.slice(2)) {
  const data = JSON.parse(fs.readFileSync(file, 'utf8'));
  data.version = process.env.RELEASE_VALUE;
  if (file.endsWith('package-lock.json') && data.packages && data.packages['']) {
    data.packages[''].version = process.env.RELEASE_VALUE;
  }
  fs.writeFileSync(file, `${JSON.stringify(data, null, 2)}\n`);
}
NODE

VERSION_VALUE="$ANYTTY_VERSION" perl -0pi -e 's/^version: .*$/version: $ENV{VERSION_VALUE}/m' \
  "$repo_root/clients/flutter/pubspec.yaml"

printf '%s\n' "Synced release version $ANYTTY_VERSION (tag $ANYTTY_RELEASE_TAG)"
