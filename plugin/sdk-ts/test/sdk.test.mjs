import { test } from 'node:test'
import assert from 'node:assert/strict'
import { PassThrough, Writable } from 'node:stream'
import { create } from '@bufbuild/protobuf'
import { Client, FrameDecoder, encodeFrame, PluginBridgeFrameSchema, PluginCommandSchema, PluginRegisterRequestSchema, PluginMessageSchema, PluginStateRequestSchema } from '../dist/index.js'

function harness(respond) {
  const input = new PassThrough(), output = new PassThrough(), decoder = new FrameDecoder()
  output.on('data', chunk => decoder.push(chunk, frame => frame.payload.case === 'command' && respond(frame, result => input.write(encodeFrame(create(PluginBridgeFrameSchema, { requestId: frame.requestId, payload: { case: 'result', value: result } }))))))
  const client = Client.stdio(input, output)
  return { input, output, client, close() { client.close(); input.destroy(); output.destroy() } }
}
const ack = { result: { case: 'ack', value: { delivered: 1 } } }
const registerRequest = () => create(PluginRegisterRequestSchema, { address: { pluginId: 'org.anytty.agents', pluginInstanceId: 'ui', tuiInstanceId: 'test-tui' } })
const command = () => create(PluginCommandSchema, { command: { case: 'register', value: registerRequest() } })

function respondRegistration(frame, reply) {
  if (frame.payload.value.command.case !== 'register') return false
  reply({ result: { case: 'registration', value: { address: { ...frame.payload.value.command.value.address, daemonId: frame.viaEndpointId || 'default', registrationEpoch: 9007199254740993n }, sourceLease: Buffer.from(frame.viaEndpointId || 'default'), bootEpoch: 'boot' } } })
  return true
}

test('fragmented framing, large uint64 and multiple frames', () => {
  const value = create(PluginBridgeFrameSchema, { requestId: '7', payload: { case: 'result', value: { result: { case: 'registration', value: { address: { registrationEpoch: 9007199254740993n }, sourceLease: Uint8Array.of(1) } } } } })
  const bytes = encodeFrame(value), decoded = [], decoder = new FrameDecoder()
  for (const byte of Buffer.concat([bytes, bytes])) decoder.push(Uint8Array.of(byte), frame => decoded.push(frame))
  decoder.finish(); assert.equal(decoded.length, 2)
  assert.equal(decoded[0].payload.value.result.value.address.registrationEpoch, 9007199254740993n)
  assert.throws(() => new FrameDecoder().push(Uint8Array.of(0), () => {}), /Empty/)
  assert.throws(() => new FrameDecoder().push(Uint8Array.of(255, 255, 255, 127), () => {}), /size limit/)
  const partial = new FrameDecoder(); partial.push(bytes.subarray(0, -1), () => {}); assert.throws(() => partial.finish(), /Truncated/)
})

test('concurrent receive does not block send; endpoint registrations stay independent', async () => {
  let waiting
  const h = harness((frame, reply) => {
    if (respondRegistration(frame, reply)) return
    const body = frame.payload.value.command
    if (body.case === 'receive') { waiting = reply; return }
    if (body.case === 'send') {
      assert.equal(Buffer.from(body.value.sourceLease).toString(), 'a')
      reply(ack); waiting({ result: { case: 'batch', value: { messages: [body.value.message] } } })
    }
  })
  try {
    const a = h.client.forEndpoint('a'), b = h.client.forEndpoint('b')
    await Promise.all([a.register(registerRequest()), b.register(registerRequest())])
    const detached = a.registration; detached.sourceLease.fill(0)
    assert.equal(Buffer.from(a.registration.sourceLease).toString(), 'a')
    const received = a.receive(25000)
    const sent = create(PluginMessageSchema, { body: { case: 'payload', value: { schema: 'example', version: 1, data: Buffer.from('test') } } })
    await a.send(sent); assert.equal((await received).messages.length, 1)
    assert.equal(b.registration.address.daemonId, 'b')
    a.close(); assert.equal(b.registration.address.daemonId, 'b')
  } finally { h.close() }
})

test('out of order correlation, cancellation, deadlines and late replies', async () => {
  const replies = []
  const h = harness((frame, reply) => replies.push(reply))
  try {
    const first = h.client.execute(command()), second = h.client.execute(command())
    replies[1]({ result: { case: 'ack', value: { delivered: 2 } } }); replies[0](ack)
    assert.equal((await first).result.value.delivered, 1); assert.equal((await second).result.value.delivered, 2)
    const controller = new AbortController()
    const cancelled = h.client.execute(command(), { signal: controller.signal }); controller.abort()
    await assert.rejects(cancelled, error => error.code === 'CANCELLED')
    replies[2](ack) // A late response must not settle another request.
    await assert.rejects(h.client.execute(command(), { timeoutMs: 5 }), error => error.code === 'DEADLINE_EXCEEDED')
    const latest = h.client.execute(command()); replies[3](ack); replies[4](ack)
    assert.equal((await latest).result.case, 'ack')
  } finally { h.close() }
})

test('disconnect and malformed responses reject every pending call', async () => {
  const h = harness(() => {})
  const one = h.client.execute(command()), two = h.client.execute(command())
  const all = Promise.allSettled([one, two]); h.input.end()
  assert.deepEqual((await all).map(value => value.status), ['rejected', 'rejected']); h.close()
  const bad = harness((frame) => bad.input.write(encodeFrame(create(PluginBridgeFrameSchema, { requestId: frame.requestId, payload: { case: 'command', value: command() } }))))
  try { await assert.rejects(bad.client.execute(command()), /response envelope/) } finally { bad.close() }
})

test('typed daemon errors propagate and send requires acknowledgement', async () => {
  const h = harness((frame, reply) => {
    if (respondRegistration(frame, reply)) return
    reply({ result: { case: 'error', value: { code: 'STALE_CONTEXT', message: 'old registration' } } })
  })
  try {
    await h.client.register(registerRequest())
    await assert.rejects(h.client.state(create(PluginStateRequestSchema, { collection: 'agents', operation: { case: 'get', value: {} } })), error => error.code === 'STALE_CONTEXT')
  } finally { h.close() }
})


test('asynchronous output failure rejects calls without an unhandled stream error', async () => {
  const input = new PassThrough()
  const output = new Writable({ write(_chunk, _encoding, done) { setImmediate(() => done(new Error('broken writer'))) } })
  const client = Client.stdio(input, output)
  try { await assert.rejects(client.execute(command()), /broken writer/) } finally { client.close(); input.destroy(); output.destroy() }
})
