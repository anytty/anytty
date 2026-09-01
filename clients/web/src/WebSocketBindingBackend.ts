import {
  BindingOperation,
  type BindingOperationCode,
  type ProtoBindingBackend,
} from '@anytty/ui'

const OP_AUTH = 0x01
const OP_ACCEPTED = 0x20
const OP_ACK = 0x21
const OP_ERROR = 0x22
const OP_EVENT = 0x30
const RESPONSE_HEADER_BYTES = 21
const BRIDGE_PROTOCOL = 'anytty.binding.v1'
const MAX_BRIDGE_MESSAGE_BYTES = 4 * 1024 * 1024
const AUTH_TOKEN_BYTES = 43
const BRIDGE_OPEN_REQUEST_TIMEOUT_MS = 45_000
const BRIDGE_REQUEST_TIMEOUT_MS = 15_000

type PendingBridgeRequest = {
  resolve(handle: bigint): void
  reject(error: Error): void
  timeout: ReturnType<typeof setTimeout>
}

export interface BindingBridgeEndpoint {
  path: '/api/bridge'
  token: string
}

/** WebSocketBindingBackend owns only the authenticated browser-to-Go binary bridge. */
export class WebSocketBindingBackend implements ProtoBindingBackend {
  private socket: WebSocket | null = null
  private connectPromise: Promise<void> | null = null
  private nextRequestId = 0n
  private readonly pending = new Map<bigint, PendingBridgeRequest>()
  private readonly abandoned = new Set<bigint>()
  private onEvent: ((payload: Uint8Array) => void) | null = null
  private onClosed: ((error: Error) => void) | null = null
  private closed = false
  private intentionalClose = false

  constructor(private readonly endpoint: () => Promise<BindingBridgeEndpoint>) {}

  start(onEvent: (payload: Uint8Array) => void, onClosed: (error: Error) => void): void {
    this.onEvent = onEvent
    this.onClosed = onClosed
  }

  async request(operation: BindingOperationCode, payload: Uint8Array, handle?: bigint, signal?: AbortSignal): Promise<bigint> {
    await this.ensureConnected()
    if (signal?.aborted) throw abortError(signal)
    const requestId = ++this.nextRequestId
    const socket = this.socket
    if (!socket || socket.readyState !== WebSocket.OPEN) throw new Error('Go binding bridge is unavailable')
    const frame = encodeBridgeRequestFrame(operation, requestId, payload, handle)
    const result = new Promise<bigint>((resolve, reject) => {
      let settled = false
      let abort: (() => void) | undefined
      const timeout = globalThis.setTimeout(() => {
        if (settled) return
        settled = true
        if (abort) signal?.removeEventListener('abort', abort)
        this.pending.delete(requestId)
        this.abandoned.add(requestId)
        socket.close()
        reject(new Error('Go binding bridge request timed out'))
      }, bridgeRequestTimeoutMs(operation))
      abort = () => {
        if (settled) return
        settled = true
        globalThis.clearTimeout(timeout)
        this.pending.delete(requestId)
        this.abandoned.add(requestId)
        reject(signal ? abortError(signal) : new DOMException('Aborted', 'AbortError'))
      }
      signal?.addEventListener('abort', abort, { once: true })
      this.pending.set(requestId, {
        resolve(value) {
          if (settled) return
          settled = true
          globalThis.clearTimeout(timeout)
          if (abort) signal?.removeEventListener('abort', abort)
          resolve(value)
        },
        reject(error) {
          if (settled) return
          settled = true
          globalThis.clearTimeout(timeout)
          if (abort) signal?.removeEventListener('abort', abort)
          reject(error)
        },
        timeout,
      })
      try {
        socket.send(frame)
      } catch (error) {
        if (settled) return
        settled = true
        globalThis.clearTimeout(timeout)
        if (abort) signal?.removeEventListener('abort', abort)
        this.pending.delete(requestId)
        reject(error)
      }
    })
    return await result
  }

  async close(): Promise<void> {
    if (this.closed) return
    this.closed = true
    this.intentionalClose = true
    const socket = this.socket
    this.socket = null
    socket?.close()
    this.rejectAll(new Error('Go binding bridge is closed'))
  }

  private async ensureConnected(): Promise<void> {
    if (this.closed) throw new Error('Go binding backend is closed')
    if (this.socket?.readyState === WebSocket.OPEN) return
    if (this.connectPromise) return await this.connectPromise
    this.connectPromise = this.connect()
    try {
      await this.connectPromise
    } finally {
      this.connectPromise = null
    }
  }

  private async connect(): Promise<void> {
    const endpoint = await this.endpoint()
    const socket = new WebSocket(bindingBridgeURL(endpoint), BRIDGE_PROTOCOL)
    socket.binaryType = 'arraybuffer'
    this.socket = socket
    await new Promise<void>((resolve, reject) => {
      let settled = false
      const finish = (failure?: unknown) => {
        if (settled) return
        settled = true
        globalThis.clearTimeout(timeout)
        if (failure) {
          socket.onclose = null
          socket.close()
          reject(failure)
          return
        }
        resolve()
      }
      const timeout = globalThis.setTimeout(() => finish(new Error('Go binding bridge authentication timed out')), 2_000)
      socket.onerror = () => finish(new Error('Go binding bridge connection failed'))
      socket.onclose = () => finish(new Error('Go binding bridge closed during authentication'))
      socket.onopen = () => {
        if (socket.protocol !== BRIDGE_PROTOCOL) {
          finish(new Error('Go binding bridge protocol negotiation failed'))
          return
        }
        if (!/^[A-Za-z0-9_-]{43}$/.test(endpoint.token)) {
          finish(new Error('Go binding bridge credential is invalid'))
          return
        }
        const token = new TextEncoder().encode(endpoint.token)
        if (token.byteLength !== AUTH_TOKEN_BYTES) {
          finish(new Error('Go binding bridge credential is invalid'))
          return
        }
        const auth = new Uint8Array(1 + AUTH_TOKEN_BYTES)
        auth[0] = OP_AUTH
        auth.set(token, 1)
        try {
          socket.send(auth)
        } catch (error) {
          finish(error)
        }
      }
      socket.onmessage = (event: MessageEvent<ArrayBuffer>) => {
        let frame: ReturnType<typeof decodeBridgeFrame>
        try {
          frame = decodeBridgeFrame(new Uint8Array(event.data))
        } catch (error) {
          finish(error)
          return
        }
        if (frame.operation !== OP_ACK) {
          finish(new Error('Go binding bridge authentication failed'))
          return
        }
        socket.onmessage = (message) => this.handleMessage(socket, new Uint8Array(message.data as ArrayBuffer))
        socket.onclose = () => this.handleClosed(socket, new Error('Go binding bridge disconnected'))
        finish()
      }
    })
  }

  private handleMessage(socket: WebSocket, bytes: Uint8Array): void {
    if (this.socket !== socket) return
    let frame: ReturnType<typeof decodeBridgeFrame>
    try {
      frame = decodeBridgeFrame(bytes)
    } catch (error) {
      socket.onclose = null
      socket.close()
      this.handleClosed(socket, error instanceof Error ? error : new Error('invalid Go binding bridge frame'))
      return
    }
    if (frame.operation === OP_EVENT) {
      this.onEvent?.(frame.payload)
      return
    }
    if (this.abandoned.delete(frame.requestId)) {
      if (frame.operation === OP_ACCEPTED && socket.readyState === WebSocket.OPEN) {
        try {
          socket.send(encodeBridgeRequestFrame(BindingOperation.CANCEL, ++this.nextRequestId, new Uint8Array(), frame.handle))
        } catch {
          // The abandoned operation is already rejected; cancellation is best-effort.
        }
      }
      return
    }
    const pending = this.pending.get(frame.requestId)
    if (!pending) return
    this.pending.delete(frame.requestId)
    globalThis.clearTimeout(pending.timeout)
    if (frame.operation === OP_ERROR) {
      pending.reject(new Error(new TextDecoder().decode(frame.payload) || 'Go binding request failed'))
      return
    }
    if (frame.operation !== OP_ACCEPTED && frame.operation !== OP_ACK) {
      pending.reject(new Error('unexpected Go binding response'))
      return
    }
    pending.resolve(frame.handle)
  }

  private handleClosed(socket: WebSocket, error: Error): void {
    if (this.socket !== socket) return
    this.socket = null
    this.rejectAll(error)
    if (!this.intentionalClose) this.onClosed?.(error)
  }

  private rejectAll(error: Error): void {
    for (const request of this.pending.values()) {
      globalThis.clearTimeout(request.timeout)
      request.reject(error)
    }
    this.pending.clear()
    this.abandoned.clear()
  }
}

export function bindingBridgeURL(endpoint: BindingBridgeEndpoint, browserLocation: Pick<Location, 'protocol' | 'host'> = globalThis.location): string {
  if (endpoint.path !== '/api/bridge') throw new Error('Go binding bridge path is invalid')
  const protocol = browserLocation.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${browserLocation.host}${endpoint.path}`
}

export function encodeBridgeRequestFrame(
  operation: BindingOperationCode,
  requestId: bigint,
  payload: Uint8Array,
  handle?: bigint,
): Uint8Array {
  const headerBytes = handle === undefined ? 9 : 17
  if (payload.byteLength > MAX_BRIDGE_MESSAGE_BYTES - headerBytes) {
    throw new Error('Go binding request exceeds bridge message limit')
  }
  const frame = new Uint8Array(headerBytes + payload.byteLength)
  const view = new DataView(frame.buffer)
  view.setUint8(0, operation)
  view.setBigUint64(1, requestId)
  if (handle !== undefined) view.setBigUint64(9, handle)
  frame.set(payload, headerBytes)
  return frame
}

export function decodeBridgeFrame(bytes: Uint8Array): { operation: number; requestId: bigint; handle: bigint; payload: Uint8Array } {
  if (bytes.byteLength > MAX_BRIDGE_MESSAGE_BYTES) throw new Error('Go binding response exceeds bridge message limit')
  if (bytes.byteLength < RESPONSE_HEADER_BYTES) throw new Error('Go binding response is truncated')
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  const length = view.getUint32(17)
  if (RESPONSE_HEADER_BYTES + length !== bytes.byteLength) throw new Error('Go binding response length mismatch')
  return {
    operation: view.getUint8(0),
    requestId: view.getBigUint64(1),
    handle: view.getBigUint64(9),
    payload: bytes.slice(RESPONSE_HEADER_BYTES),
  }
}

function bridgeRequestTimeoutMs(operation: BindingOperationCode): number {
  return operation === BindingOperation.OPEN_SESSION ? BRIDGE_OPEN_REQUEST_TIMEOUT_MS : BRIDGE_REQUEST_TIMEOUT_MS
}

function abortError(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new DOMException('Aborted', 'AbortError')
}
