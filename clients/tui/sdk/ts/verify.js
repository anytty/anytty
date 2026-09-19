#!/usr/bin/env node
/**
 * Runs the official conformance suite against this SDK through the one
 * standard: `tui2-sdk-verify --cmd`.
 *
 *   node tui2/sdk/ts/verify.js
 *
 * Requires Go only to build the runner (the runner is the single source of
 * truth for fixtures); the SDK itself is Node-only.
 */

'use strict';

const { spawnSync } = require('child_process');
const path = require('path');

const sdkDir = __dirname;
const repoRoot = path.resolve(sdkDir, '..', '..', '..');
const conformance = path.join(sdkDir, 'conformance.js');

const result = spawnSync('go', ['run', './tui2/cmd/tui2-sdk-verify',
  '--fixtures', 'tui2/conformance/fixtures.jsonl',
  '--cmd', `node ${conformance}`], {
  cwd: repoRoot,
  stdio: 'inherit',
  env: Object.assign({}, process.env, { GOWORK: 'off' }),
});

if (result.error) {
  process.stderr.write(`ts verify: ${result.error.message}\n`);
  process.exit(2);
}
process.exit(result.status === null ? 2 : result.status);
