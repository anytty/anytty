// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest'
import { BindingOperation } from '@anytty/ui'
import { WebSocketBindingBackend, bindingBridgeURL, decodeBridgeFrame, encodeBridgeRequestFrame } from './WebSocketBindingBackend'

const MAX_MESSAGE_BYTES = 4 * 1024 * 1024

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('Web binding bridge allocation limits', () => {
  it('counts the 9-byte request header before allocation', () => {
    const payload = new Uint8Array(MAX_MESSAGE_BYTES - 9)
    expect(encodeBridgeRequestFrame(BindingOperation.OPEN_SESSION, 1n, payload))
      .toHaveLength(MAX_MESSAGE_BYTES)
    expect(() => encodeBridgeRequestFrame(
      BindingOperation.OPEN_SESSION,
      1n,
      new Uint8Array(MAX_MESSAGE_BYTES - 8),
    )).toThrow('exceeds bridge message limit')
  })

  it('counts the 17-byte handled request header before allocation', () => {
    const payload = new Uint8Array(MAX_MESSAGE_BYTES - 17)
    expect(encodeBridgeRequestFrame(BindingOperation.EXECUTE, 2n, payload, 3n))
      .toHaveLength(MAX_MESSAGE_BYTES)
    expect(() => encodeBridgeRequestFrame(
      BindingOperation.EXECUTE,
      2n,
      new Uint8Array(MAX_MESSAGE_BYTES - 16),
      3n,
    )).toThrow('exceeds bridge message limit')
  })

  it('rejects oversized responses before slicing payload', () => {
    expect(() => decodeBridgeFrame(new Uint8Array(MAX_MESSAGE_BYTES + 1)))
      .toThrow('exceeds bridge message limit')
  })
})

describe('Web binding bridge URL', () => {
  it('uses the current secure origin for Local Web', () => {
    expect(bindingBridgeURL(
      { path: '/api/bridge', token: 'A'.repeat(43) },
      { protocol: 'https:', host: 'terminal.example' },
    ))
      .toBe('wss://terminal.example/api/bridge')
  })

})

describe('Web binding bridge negotiation', () => {
  it('passes the exact subprotocol and rejects a different negotiated protocol', async () => {
    const sockets: TestWebSocket[] = []
    vi.stubGlobal('WebSocket', testWebSocketClass(sockets))
    const backend = new WebSocketBindingBackend(async () => ({ path: '/api/bridge', token: 'A'.repeat(43) }))

    const request = backend.request(BindingOperation.OPEN_SESSION, new Uint8Array())
    await vi.waitFor(() => expect(sockets).toHaveLength(1))
    const socket = sockets[0]!
    expect(socket.url).toBe('ws://localhost:3000/api/bridge')
    expect(socket.requestedProtocol).toBe('anytty.binding.v1')
    socket.open('different.protocol')

    await expect(request).rejects.toThrow('protocol negotiation failed')
    expect(socket.sent).toHaveLength(0)
  })

  it('sends only the 44-byte auth message before the first request', async () => {
    const sockets: TestWebSocket[] = []
    vi.stubGlobal('WebSocket', testWebSocketClass(sockets))
    const backend = new WebSocketBindingBackend(async () => ({ path: '/api/bridge', token: 'A'.repeat(43) }))

    const request = backend.request(BindingOperation.OPEN_SESSION, new Uint8Array())
    await vi.waitFor(() => expect(sockets).toHaveLength(1))
    const socket = sockets[0]!
    socket.open('anytty.binding.v1')
    await vi.waitFor(() => expect(socket.sent).toHaveLength(1))
    expect(socket.sent[0]).toHaveLength(44)
    expect(socket.sent[0]![0]).toBe(0x01)
    expect(new TextDecoder().decode(socket.sent[0]!.slice(1))).toBe('A'.repeat(43))

    socket.message(responseFrame(0x21, 0n, 0n))
    await vi.waitFor(() => expect(socket.sent).toHaveLength(2))
    expect(socket.sent[1]).toHaveLength(9)
    socket.message(responseFrame(0x20, 1n, 7n))
    await expect(request).resolves.toBe(7n)
    await backend.close()
  })

  it('rejects a request that never receives a bridge response', async () => {
    vi.useFakeTimers()
    try {
      const sockets: TestWebSocket[] = []
      vi.stubGlobal('WebSocket', testWebSocketClass(sockets))
      const backend = new WebSocketBindingBackend(async () => ({ path: '/api/bridge', token: 'A'.repeat(43) }))

      const request = backend.request(BindingOperation.EXECUTE, new Uint8Array(), 1n)
      await vi.advanceTimersByTimeAsync(0)
      expect(sockets).toHaveLength(1)
      const socket = sockets[0]!
      socket.open('anytty.binding.v1')
      await vi.advanceTimersByTimeAsync(0)
      expect(socket.sent).toHaveLength(1)
      socket.message(responseFrame(0x21, 0n, 0n))
      await vi.advanceTimersByTimeAsync(0)
      expect(socket.sent).toHaveLength(2)

      const assertion = expect(request).rejects.toThrow('Go binding bridge request timed out')
      await vi.advanceTimersByTimeAsync(15_000)
      await assertion
      await backend.close()
    } finally {
      vi.useRealTimers()
    }
  })

  it('ignores a delayed close callback from an older socket', async () => {
    const sockets: TestWebSocket[] = []
    vi.stubGlobal('WebSocket', testWebSocketClass(sockets))
    const backend = new WebSocketBindingBackend(async () => ({ path: '/api/bridge', token: 'A'.repeat(43) }))
    const onClosed = vi.fn()
    backend.start(() => undefined, onClosed)

    const firstRequest = backend.request(BindingOperation.OPEN_SESSION, new Uint8Array())
    await vi.waitFor(() => expect(sockets).toHaveLength(1))
    const first = sockets[0]!
    first.open('anytty.binding.v1')
    first.message(responseFrame(0x21, 0n, 0n))
    await vi.waitFor(() => expect(first.sent).toHaveLength(2))
    first.message(responseFrame(0x20, 1n, 7n))
    await expect(firstRequest).resolves.toBe(7n)
    first.disconnect()
    expect(onClosed).toHaveBeenCalledOnce()

    const secondRequest = backend.request(BindingOperation.OPEN_SESSION, new Uint8Array())
    await vi.waitFor(() => expect(sockets).toHaveLength(2))
    const second = sockets[1]!
    second.open('anytty.binding.v1')
    second.message(responseFrame(0x21, 0n, 0n))
    await vi.waitFor(() => expect(second.sent).toHaveLength(2))

    first.disconnect()
    second.message(responseFrame(0x20, 2n, 8n))
    await expect(secondRequest).resolves.toBe(8n)
    expect(onClosed).toHaveBeenCalledOnce()
    await backend.close()
  })
})

type TestWebSocket = {
  readonly url: string
  readonly requestedProtocol: string
  readonly sent: Uint8Array[]
  open(protocol: string): void
  message(bytes: Uint8Array): void
  disconnect(): void
}

function testWebSocketClass(sockets: TestWebSocket[]) {
  return class {
    static readonly OPEN = 1
    readonly sent: Uint8Array[] = []
    binaryType = ''
    readyState = 0
    protocol = ''
    onopen: ((event: Event) => void) | null = null
    onerror: ((event: Event) => void) | null = null
    onclose: ((event: CloseEvent) => void) | null = null
    onmessage: ((event: MessageEvent<ArrayBuffer>) => void) | null = null

    constructor(readonly url: string, readonly requestedProtocol: string) {
      sockets.push(this)
    }

    send(value: ArrayBufferView) {
      this.sent.push(new Uint8Array(value.buffer, value.byteOffset, value.byteLength).slice())
    }

    close() { this.readyState = 3 }

    open(protocol: string) {
      this.protocol = protocol
      this.readyState = 1
      this.onopen?.(new Event('open'))
    }

    message(bytes: Uint8Array) {
      const data = bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer
      this.onmessage?.(new MessageEvent('message', { data }))
    }

    disconnect() {
      this.readyState = 3
      this.onclose?.(new CloseEvent('close'))
    }
  }
}

function responseFrame(operation: number, requestId: bigint, handle: bigint): Uint8Array {
  const bytes = new Uint8Array(21)
  const view = new DataView(bytes.buffer)
  view.setUint8(0, operation)
  view.setBigUint64(1, requestId)
  view.setBigUint64(9, handle)
  view.setUint32(17, 0)
  return bytes
}
