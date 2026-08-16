import assert from 'node:assert/strict'
import test from 'node:test'
import { releaseTarget } from './platform.mjs'

test('maps every published release target', () => {
  assert.deepEqual(releaseTarget('darwin', 'arm64'), { os: 'darwin', arch: 'arm64', extension: '.tar.gz' })
  assert.deepEqual(releaseTarget('darwin', 'x64'), { os: 'darwin', arch: 'amd64', extension: '.tar.gz' })
  assert.deepEqual(releaseTarget('linux', 'arm64'), { os: 'linux', arch: 'arm64', extension: '.tar.gz' })
  assert.deepEqual(releaseTarget('linux', 'x64'), { os: 'linux', arch: 'amd64', extension: '.tar.gz' })
  assert.deepEqual(releaseTarget('win32', 'arm64'), { os: 'windows', arch: 'arm64', extension: '.zip' })
  assert.deepEqual(releaseTarget('win32', 'x64'), { os: 'windows', arch: 'amd64', extension: '.zip' })
})

test('rejects targets without release artifacts', () => {
  assert.throws(() => releaseTarget('freebsd', 'x64'), /Unsupported platform/)
  assert.throws(() => releaseTarget('linux', 'ia32'), /Unsupported platform/)
})
