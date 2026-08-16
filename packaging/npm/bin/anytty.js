#!/usr/bin/env node

import { spawnSync } from 'node:child_process'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const packageRoot = dirname(dirname(fileURLToPath(import.meta.url)))
const executable = join(packageRoot, 'vendor', process.platform === 'win32' ? 'anytty.exe' : 'anytty')
const result = spawnSync(executable, process.argv.slice(2), { stdio: 'inherit' })
if (result.error) {
  console.error(`Unable to run AnyTTY: ${result.error.message}`)
  process.exit(1)
}
process.exit(result.status ?? 1)
