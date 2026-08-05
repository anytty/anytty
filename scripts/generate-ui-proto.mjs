#!/usr/bin/env node

import { readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { delimiter, dirname, join, relative, resolve } from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const protoRoot = join(repoRoot, 'proto')
const outputRoot = join(repoRoot, 'clients', 'ui', 'src', 'generated')
const requestedGroup = process.argv[2] ?? 'all'

if (!['all', 'api', 'wire'].includes(requestedGroup)) {
  throw new Error(`unknown proto generation group: ${requestedGroup}`)
}

function protoFiles(directory) {
  return readdirSync(join(protoRoot, directory), { withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name.endsWith('.proto'))
    .map((entry) => join(protoRoot, directory, entry.name))
    .sort()
}

function runProtoc(args) {
  const environment = {
    ...process.env,
    PATH: `${join(repoRoot, 'node_modules', '.bin')}${delimiter}${process.env.PATH ?? ''}`,
  }
  const result = spawnSync('protoc', args, { cwd: repoRoot, env: environment, stdio: 'inherit' })
  if (result.error?.code === 'ENOENT') {
    throw new Error('protoc is required; on Windows run: winget install --id Google.Protobuf --exact')
  }
  if (result.error) {
    throw result.error
  }
  if (result.status !== 0) {
    process.exit(result.status ?? 1)
  }
}

function normalizeGenerated(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) {
      normalizeGenerated(path)
    } else if (entry.isFile() && entry.name.endsWith('_pb.ts')) {
      writeFileSync(path, `${readFileSync(path, 'utf8').trimEnd()}\n`)
    }
  }
}

if (requestedGroup === 'all' || requestedGroup === 'api') {
  const sources = [
    ...protoFiles('apipb'),
    join(protoRoot, 'bindingpb', 'client_binding.proto'),
    join(protoRoot, 'remoteauthpb', 'remote_auth.proto'),
    ...protoFiles(join('cloud', 'v1')),
  ].map((path) => relative(repoRoot, path))
  runProtoc([
    `--es_out=${relative(repoRoot, outputRoot)}`,
    '--es_opt=target=ts,import_extension=none',
    '-I',
    relative(repoRoot, protoRoot),
    ...sources,
  ])
}

if (requestedGroup === 'all' || requestedGroup === 'wire') {
  const wireRoot = join(protoRoot, 'wirepb')
  runProtoc([
    `--es_out=${relative(repoRoot, join(outputRoot, 'wirepb'))}`,
    '--es_opt=target=ts,import_extension=none',
    '-I',
    relative(repoRoot, wireRoot),
    relative(repoRoot, join(wireRoot, 'terminal.proto')),
  ])
}

normalizeGenerated(outputRoot)
