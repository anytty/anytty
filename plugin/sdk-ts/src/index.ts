import { clone, create } from '@bufbuild/protobuf'
import type { Readable, Writable } from 'node:stream'
import {
  PluginBridgeFrameSchema, PluginCommandSchema, PluginRegistrationSchema, PluginStateRequestSchema,
  type PluginCommand, type PluginResult, type PluginRegistration, type PluginRegisterRequest,
  type PluginMessage, type PluginBatch, type PluginStateRequest, type PluginStateSnapshot, type PluginBridgeFrame,
} from './generated/apipb/plugin_pb.js'
import { FrameDecoder, encodeFrame, MAX_FRAME_BYTES } from './framing.js'
export * from './generated/apipb/plugin_pb.js'
export { FrameDecoder, encodeFrame, MAX_FRAME_BYTES } from './framing.js'

export class PluginError extends Error {
  constructor(readonly code: string, message: string) { super(message); this.name = 'PluginError' }
}
export interface CallOptions { signal?: AbortSignal; timeoutMs?: number }
type Pending = { resolve: (value: PluginResult) => void; reject: (error: Error) => void; cleanup: () => void; endpoint: string }
const MAX_PENDING = 128
// Buffered writes can emit an error after all RPC callers have been rejected.
const ignoreClosedStreamError = (): void => {}

/** One multiplexer owns stdio. Waiting Receive never holds the output writer. */
class Transport {
  private readonly pending = new Map<string, Pending>()
  private next = 0n
  private failure?: Error
  private bufferedOutput = 0
  private readonly decoder = new FrameDecoder()
  constructor(private readonly input: Readable, private readonly output: Writable) {
    input.on('data', this.onData); input.on('end', this.onEnd); input.on('close', this.onClose); input.on('error', this.onError)
    output.on('error', this.onError); output.on('close', this.onClose)
  }
  private onData = (chunk: Buffer | string): void => {
    try { this.decoder.push(typeof chunk === 'string' ? Buffer.from(chunk) : chunk, frame => this.deliver(frame)) }
    catch (error) { this.fail(error instanceof Error ? error : new Error(String(error))) }
  }
  private onEnd = (): void => { try { this.decoder.finish() } catch (error) { this.fail(error as Error); return }; this.fail(new PluginError('TRANSPORT_CLOSED', 'Plugin host disconnected')) }
  private onClose = (): void => this.fail(new PluginError('TRANSPORT_CLOSED', 'Plugin transport closed'))
  private onError = (error: Error): void => this.fail(error)
  private deliver(frame: PluginBridgeFrame): void {
    if (frame.requestId.length === 0 || frame.requestId.length > 160 || frame.payload.case !== 'result') throw new Error('Invalid plugin response envelope')
    const pending = this.pending.get(frame.requestId)
    // A cancelled/deadline-expired request can still produce a late reply.
    if (!pending) return
    if (frame.viaEndpointId && frame.viaEndpointId !== pending.endpoint) throw new Error('Plugin response endpoint mismatch')
    this.pending.delete(frame.requestId); pending.cleanup()
    pending.resolve(frame.payload.value)
  }
  execute(endpoint: string, command: PluginCommand, options: CallOptions): Promise<PluginResult> {
    if (this.failure) return Promise.reject(this.failure)
    if (options.signal?.aborted) return Promise.reject(new PluginError('CANCELLED', 'Plugin request cancelled before send'))
    if (this.pending.size >= MAX_PENDING) return Promise.reject(new PluginError('RESOURCE_EXHAUSTED', 'Too many pending plugin requests'))
    const timeout = options.timeoutMs ?? 30_000
    if (!Number.isFinite(timeout) || timeout <= 0 || timeout > 120_000) return Promise.reject(new RangeError('timeoutMs must be between 1 and 120000'))
    const requestId = (++this.next).toString()
    let bytes: Uint8Array
    try { bytes = encodeFrame(create(PluginBridgeFrameSchema, { requestId, viaEndpointId: endpoint, deadlineUnixMillis: BigInt(Date.now() + timeout), payload: { case: 'command', value: command } })) }
    catch (error) { return Promise.reject(error) }
    if (this.bufferedOutput + bytes.length > MAX_FRAME_BYTES * 2) return Promise.reject(new PluginError('RESOURCE_EXHAUSTED', 'Plugin output buffer is full'))
    return new Promise((resolve, reject) => {
      const abort = (): void => { this.reject(requestId, new PluginError('CANCELLED', 'Plugin request cancelled; dispatched writes may already have executed'), true) }
      const timer = setTimeout(() => this.reject(requestId, new PluginError('DEADLINE_EXCEEDED', 'Plugin request deadline exceeded; dispatched writes may already have executed'), true), timeout)
      const cleanup = (): void => { clearTimeout(timer); options.signal?.removeEventListener('abort', abort) }
      this.pending.set(requestId, { resolve, reject, cleanup, endpoint })
      options.signal?.addEventListener('abort', abort, { once: true })
      this.bufferedOutput += bytes.length
      try {
        this.output.write(bytes, (error?: Error | null) => {
          this.bufferedOutput -= bytes.length
          if (error) this.fail(error)
        })
      } catch (error) { this.bufferedOutput -= bytes.length; this.fail(error as Error) }
    })
  }
  private reject(id: string, error: Error, cancelRemote = false): void {
    const pending = this.pending.get(id)
    if (!pending) return
    this.pending.delete(id); pending.cleanup(); pending.reject(error)
    if (cancelRemote && !this.failure) {
      const bytes = encodeFrame(create(PluginBridgeFrameSchema, { requestId: id, viaEndpointId: pending.endpoint, payload: { case: 'cancelRequestId', value: id } }))
      // A small reserved budget keeps cancellation available behind regular data.
      if (this.bufferedOutput + bytes.length > MAX_FRAME_BYTES * 2 + 65536) { this.fail(new Error('Plugin cancellation output budget exhausted')); return }
      this.bufferedOutput += bytes.length
      try { this.output.write(bytes, (writeError?: Error | null) => { this.bufferedOutput -= bytes.length; if (writeError) this.fail(writeError) }) }
      catch (writeError) { this.bufferedOutput -= bytes.length; this.fail(writeError as Error) }
    }
  }

  fail(error: Error): void {
    if (this.failure) return
    this.failure = error
    for (const id of this.pending.keys()) this.reject(id, error)
    this.input.off('data', this.onData); this.input.off('end', this.onEnd); this.input.off('close', this.onClose); this.input.off('error', this.onError)
    this.output.off('error', this.onError); this.output.off('close', this.onClose)
    this.input.on('error', ignoreClosedStreamError); this.output.on('error', ignoreClosedStreamError)
    this.input.pause()
  }
}

export class Client {
  private registrationValue?: PluginRegistration
  private registering = false
  private readonly lifetime = new AbortController()
  private constructor(private readonly transport: Transport, readonly endpointId: string, private readonly ownsTransport: boolean) {}
  static stdio(input: Readable = process.stdin, output: Writable = process.stdout): Client { return new Client(new Transport(input, output), '', true) }
  forEndpoint(endpointId: string): Client {
    if (this.lifetime.signal.aborted) throw new PluginError('CANCELLED', 'Plugin client closed')
    if (!endpointId || endpointId.length > 160) throw new Error('A bounded endpoint ID is required')
    return new Client(this.transport, endpointId, false)
  }
  get registration(): PluginRegistration | undefined { return this.registrationValue && clone(PluginRegistrationSchema, this.registrationValue) }
  async execute(command: PluginCommand, options: CallOptions = {}): Promise<PluginResult> {
    if (!command.command.case) throw new Error('Plugin command is required')
    const signal = options.signal ? AbortSignal.any([options.signal, this.lifetime.signal]) : this.lifetime.signal
    const result = await this.transport.execute(this.endpointId, clone(PluginCommandSchema, command), { ...options, signal })
    if (this.lifetime.signal.aborted) throw new PluginError('CANCELLED', 'Plugin client closed')
    if (result.result.case === 'error') throw new PluginError(result.result.value.code, result.result.value.message)
    if (!result.result.case) throw new Error('Missing plugin result')
    return result
  }
  async register(request: PluginRegisterRequest, options: CallOptions = {}): Promise<PluginRegistration> {
    if (this.registering || this.registrationValue) throw new Error('Client already registered or registering')
    this.registering = true
    try {
      const result = await this.execute(create(PluginCommandSchema, { command: { case: 'register', value: request } }), options)
      if (result.result.case !== 'registration' || !result.result.value.address || result.result.value.sourceLease.length === 0) throw new Error('Invalid daemon registration')
      this.registrationValue = clone(PluginRegistrationSchema, result.result.value)
      return clone(PluginRegistrationSchema, result.result.value)
    } finally { this.registering = false }
  }
  private lease(): Uint8Array { if (!this.registrationValue) throw new Error('Plugin is not registered'); return this.registrationValue.sourceLease.slice() }
  /** Resolves when daemon accepts delivery, NOT when the target operation completes. */
  async send(message: PluginMessage, options: CallOptions = {}): Promise<void> {
    const result = await this.execute(create(PluginCommandSchema, { command: { case: 'send', value: { sourceLease: this.lease(), message } } }), options)
    if (result.result.case !== 'ack') throw new Error('Missing daemon delivery acknowledgement')
  }
  async receive(waitMs = 25_000, options: CallOptions = {}): Promise<PluginBatch> {
    if (!Number.isInteger(waitMs) || waitMs < 0 || waitMs > 25_000) throw new RangeError('waitMs must be between 0 and 25000')
    const result = await this.execute(create(PluginCommandSchema, { command: { case: 'receive', value: { sourceLease: this.lease(), waitMillis: waitMs, maxMessages: 32 } } }), options)
    if (result.result.case !== 'batch') throw new Error('Missing plugin message batch')
    return result.result.value
  }
  async state(request: PluginStateRequest, options: CallOptions = {}): Promise<PluginStateSnapshot> {
    const copy = clone(PluginStateRequestSchema, request); copy.sourceLease = this.lease()
    const result = await this.execute(create(PluginCommandSchema, { command: { case: 'state', value: copy } }), options)
    if (result.result.case !== 'state') throw new Error('Missing state snapshot')
    return result.result.value
  }
  async unregister(options: CallOptions = {}): Promise<void> {
    if (!this.registrationValue) return
    const result = await this.execute(create(PluginCommandSchema, { command: { case: 'unregister', value: { sourceLease: this.lease() } } }), options)
    if (result.result.case !== 'ack') throw new Error('Missing unregister acknowledgement')
    this.registrationValue = undefined
  }
  /** A child closes only its own calls; root close closes the shared transport. */
  close(): void {
    this.lifetime.abort(); this.registrationValue = undefined
    if (this.ownsTransport) this.transport.fail(new PluginError('TRANSPORT_CLOSED', 'Plugin client closed'))
  }
}
