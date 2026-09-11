import { test } from 'node:test'
import assert from 'node:assert/strict'
import { spawn, execFile } from 'node:child_process'
import { promisify } from 'node:util'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'
import { once } from 'node:events'
import { create } from '@bufbuild/protobuf'
import { Client, PluginRegisterRequestSchema, PluginMessageSchema, PluginStateRequestSchema } from '../dist/index.js'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '../../..')
const register = (instance, daemonService = false) => create(PluginRegisterRequestSchema, {
  address: { pluginId: 'org.anytty.agents', pluginInstanceId: instance, tuiInstanceId: daemonService ? '' : 'ts-tui' }, daemonService,
})

test('TypeScript ↔ Go stdio SDK ↔ two authenticated Protobuf daemon sessions', { timeout: 90_000 }, async () => {
  const directory = await mkdtemp(join(tmpdir(), 'anytty-sdk-ts-bin-')), binary = join(directory, process.platform === 'win32' ? 'interop.exe' : 'interop')
  let child, client
  try {
    await promisify(execFile)('go', ['build', '-o', binary, './plugin/sdk-ts/test/interop'], { cwd: root, timeout: 60_000 })
    child = spawn(binary, [], { stdio: ['pipe', 'pipe', 'pipe'] })
    let stderr = ''; child.stderr.on('data', data => { stderr += data })
    client = Client.stdio(child.stdout, child.stdin)
    const a = client.forEndpoint('alpha'), b = client.forEndpoint('beta')
    const [ra, rb] = await Promise.all([a.register(register('host')), b.register(register('host'))])
    assert.equal(ra.address.daemonId, 'daemon-alpha'); assert.equal(rb.address.daemonId, 'daemon-beta')
    const message = create(PluginMessageSchema, {
      destination: ra.address,
      body: { case: 'payload', value: { schema: 'interop', version: 1, data: Buffer.from('真实 Protobuf 🦀') } },
    })
    const waiting = a.receive(25_000)
    await a.send(message)
    const incoming = await waiting
    assert.equal(incoming.messages.length, 1)
    assert.equal(incoming.messages[0].source.daemonId, 'daemon-alpha')
    assert.equal(Buffer.from(incoming.messages[0].body.value.data).toString(), '真实 Protobuf 🦀')
    assert.equal((await b.receive(0)).messages.length, 0)
    await assert.rejects(b.send(message), error => error.code === 'TARGET_OFFLINE')
    const backend = client.forEndpoint('alpha'); await backend.register(register('backend', true))
    // Abort traverses the Go bridge into the daemon's application request context.
    const cancellation = new AbortController()
    const cancelled = a.receive(25_000, { signal: cancellation.signal })
    await new Promise(resolve => setImmediate(resolve)); cancellation.abort()
    await assert.rejects(cancelled, error => error.code === 'CANCELLED')
    const releasedBy = Date.now() + 1000
    for (;;) {
      try { await a.receive(0); break }
      catch (error) { if (error.code !== 'CONFLICT' || Date.now() >= releasedBy) throw error; await new Promise(resolve => setTimeout(resolve, 5)) }
    }
    const watch = await a.state(create(PluginStateRequestSchema, { collection: 'interop', operation: { case: 'watch', value: {} } }))
    assert.equal(watch.revision, 0n)
    const snapshot = await backend.state(create(PluginStateRequestSchema, {
      collection: 'interop', operation: { case: 'put', value: { expectedRevision: 0n, value: { schema: 'test', version: 1, data: Buffer.from('saved') } } },
    }))
    assert.equal(snapshot.revision, 1n)
    const changed = (await a.receive(0)).messages[0]
    assert.equal(changed.body.case, 'stateChanged'); assert.equal(changed.body.value.revision, 1n)
    await Promise.all([a.unregister(), b.unregister(), backend.unregister()])
    const exited = once(child, 'exit'); client.close(); child.stdin.end()
    const [code] = await exited
    assert.equal(code, 0, stderr)
  } finally {
    client?.close()
    // Only the test-owned helper process is terminated on failure.
    if (child && child.exitCode === null) { const exited = once(child, 'exit'); child.kill(); await exited }
    await rm(directory, { recursive: true, force: true })
  }
})
