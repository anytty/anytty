#!/usr/bin/env node
import { mkdirSync } from 'node:fs'
import { delimiter, dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { execFileSync } from 'node:child_process'
const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const output = join(root, 'plugin/sdk-ts/src/generated')
mkdirSync(output, { recursive: true })
execFileSync('protoc', ['-I', join(root, 'proto'), `--es_out=${output}`, '--es_opt=target=ts,import_extension=js', 'apipb/plugin.proto'], {
  cwd: root, stdio: 'inherit', env: { ...process.env, PATH: `${join(root, 'node_modules/.bin')}${delimiter}${process.env.PATH ?? ''}` },
})
