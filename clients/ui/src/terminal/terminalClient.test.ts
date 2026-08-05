import { describe, expect, it, vi, type Mock } from 'vitest'
import { TerminalClient, type TerminalClientCallbacks, type TerminalProtocolEvent, type TerminalProtocolSession } from './terminalClient'

describe('TerminalClient', () => {
  it('opens terminal channels by terminalId and emits terminal lifecycle messages', () => {
    const session = createMockTerminalProtocolSession()
    const callbacks = callbacksForTest()
    const client = new TerminalClient(callbacks)

    client.connect('terminal-1', session)

    expect(session.openedTerminalIds).toEqual(['terminal-1'])
    expect(session.openedLabels).toEqual(['proto-terminal:terminal-1'])
    return vi.waitFor(() => expect(callbacks.onLifecycle).toHaveBeenCalledWith({
      type: 'terminal.channelOpen',
      machineId: 'machine-local',
      terminalId: 'terminal-1',
    }))
  })

  it('decodes output, snapshot, and terminal info messages without pane/session fields', () => {
    const session = createMockTerminalProtocolSession()
    const callbacks = callbacksForTest()
    const client = new TerminalClient(callbacks)

    client.connect('terminal-1', session)
    session.emit('terminal-1', { type: 'output', data: new TextEncoder().encode('hello') })
    session.emit('terminal-1', { type: 'snapshot', snapshot: {
      text: 'prompt',
      cols: 120,
      rows: 36,
      replay: '\x1b[Hprompt',
    } })
    session.emit('terminal-1', { type: 'info', info: {
      terminal_id: 'terminal-1',
      machine_id: 'machine-local',
      title: 'zsh',
      state: 'running',
      command: '/bin/zsh',
    } })

    expect(callbacks.onOutput).toHaveBeenCalledWith(new TextEncoder().encode('hello'))
    expect(callbacks.onSnapshot).toHaveBeenCalledWith({ text: 'prompt', cols: 120, rows: 36, replay: '\x1b[Hprompt' })
    expect(callbacks.onTerminalInfo).toHaveBeenCalledWith(
      expect.objectContaining({ terminalId: 'terminal-1', machineId: 'machine-local' }),
    )
    expect(JSON.stringify(callbacks.onTerminalInfo.mock.calls)).not.toMatch(/pane|session|window|workspace|tab/i)
  })

  it('reattaches to a new transport without changing terminal identity', async () => {
    const first = createMockTerminalProtocolSession()
    const second = createMockTerminalProtocolSession()
    const callbacks = callbacksForTest()
    const client = new TerminalClient(callbacks)

    client.connect('terminal-1', first)
    await vi.waitFor(() => expect(callbacks.onLifecycle).toHaveBeenCalledTimes(1))
    client.reattach(second)

    expect(first.closedTerminalIds).not.toContain('terminal-1')
    expect(second.openedTerminalIds).toEqual(['terminal-1'])
    await vi.waitFor(() => expect(callbacks.onLifecycle).toHaveBeenLastCalledWith({
      type: 'terminal.channelOpen',
      machineId: 'machine-local',
      terminalId: 'terminal-1',
    }))
  })

  it('sends input and resize through the interface channel and reports dropped input through callbacks', async () => {
    const session = createMockTerminalProtocolSession()
    const callbacks = callbacksForTest()
    const client = new TerminalClient(callbacks)

    client.connect('terminal-1', session)
    await vi.waitFor(() => expect(callbacks.onLifecycle).toHaveBeenCalledTimes(1))
    client.sendInput('ls\n')
    session.emit('terminal-1', { type: 'resizeControl', control: { canResize: true, reason: 'owner' } })
    client.sendResize(100, 30)

    expect(session.sentText('terminal-1')).toContain('ls\n')
    expect(session.sentResize('terminal-1')).toEqual({ cols: 100, rows: 30 })

    session.closeTerminalChannel('terminal-1')
    client.sendInput('dropped')
    expect(callbacks.onInputDropped).toHaveBeenCalledTimes(1)
    expect(callbacks.onInputSendFailed).toHaveBeenCalledWith('terminal channel is not open')
  })

  it('loads scrollback through the active protocol session', async () => {
    const session = createMockTerminalProtocolSession()
    const callbacks = callbacksForTest()
    const client = new TerminalClient(callbacks)

    client.connect('terminal-1', session)
    await vi.waitFor(() => expect(callbacks.onLifecycle).toHaveBeenCalledTimes(1))

    const page = await client.loadScrollback(100, 50, false, { cols: 51 })

    expect(session.scrollbackRequests).toEqual([{ terminalId: 'terminal-1', offset: 100, limit: 50, cols: 51 }])
    expect(page).toMatchObject({
      beforeOffset: 100,
      limit: 50,
      rows: 1,
      replay: 'older',
      hasMore: false,
      alternate: false,
    })
  })

  it('forwards scrollback cancellation to the active protocol session', async () => {
    const session = createMockTerminalProtocolSession()
    const callbacks = callbacksForTest()
    const client = new TerminalClient(callbacks)
    const controller = new AbortController()
    client.connect('terminal-1', session)
    await vi.waitFor(() => expect(callbacks.onLifecycle).toHaveBeenCalledTimes(1))

    await client.loadScrollback(0, 50, false, { signal: controller.signal })

    expect(session.scrollbackRequests).toEqual([{
      terminalId: 'terminal-1',
      offset: 0,
      limit: 50,
      signal: controller.signal,
    }])
  })

  it('ignores stale async terminal opens after switching to another terminal', async () => {
    const session = new DeferredTerminalProtocolSession()
    const callbacks = callbacksForTest()
    const client = new TerminalClient(callbacks)

    client.connect('terminal-1', session)
    client.connect('terminal-2', session)

    const staleChannel = session.resolveOpen('terminal-1')
    const activeChannel = session.resolveOpen('terminal-2')

    await vi.waitFor(() => expect(callbacks.onLifecycle).toHaveBeenCalledTimes(1))
    expect(callbacks.onLifecycle).toHaveBeenCalledWith({
      type: 'terminal.channelOpen',
      machineId: 'machine-local',
      terminalId: 'terminal-2',
    })
    expect(staleChannel.readyState).toBe('closed')
    expect(activeChannel.readyState).toBe('open')
    expect(session.closedTerminalIds).toContain('terminal-1')
  })
})

function callbacksForTest(): TerminalClientCallbacks & {
  onOutput: Mock<TerminalClientCallbacks['onOutput']>
  onSnapshot: Mock<TerminalClientCallbacks['onSnapshot']>
  onTerminalInfo: Mock<NonNullable<TerminalClientCallbacks['onTerminalInfo']>>
  onLifecycle: Mock<NonNullable<TerminalClientCallbacks['onLifecycle']>>
  onInputDropped: Mock<NonNullable<TerminalClientCallbacks['onInputDropped']>>
  onInputSendFailed: Mock<NonNullable<TerminalClientCallbacks['onInputSendFailed']>>
} {
  return {
    onOutput: vi.fn<TerminalClientCallbacks['onOutput']>(),
    onSnapshot: vi.fn<TerminalClientCallbacks['onSnapshot']>(),
    onTerminalInfo: vi.fn<NonNullable<TerminalClientCallbacks['onTerminalInfo']>>(),
    onLifecycle: vi.fn<NonNullable<TerminalClientCallbacks['onLifecycle']>>(),
    onError: vi.fn<TerminalClientCallbacks['onError']>(),
    onClose: vi.fn<TerminalClientCallbacks['onClose']>(),
    onOpen: vi.fn<NonNullable<TerminalClientCallbacks['onOpen']>>(),
    onInputDropped: vi.fn<NonNullable<TerminalClientCallbacks['onInputDropped']>>(),
    onInputSendFailed: vi.fn<NonNullable<TerminalClientCallbacks['onInputSendFailed']>>(),
  }
}

function createMockTerminalProtocolSession(): MockTerminalProtocolSession {
  return new MockTerminalProtocolSession()
}

class MockTerminalProtocolSession implements TerminalProtocolSession {
  readonly openedTerminalIds: string[] = []
  readonly openedLabels: string[] = []
  readonly closedTerminalIds: string[] = []
  readonly scrollbackRequests: Array<{ terminalId: string; offset: number; limit: number; signal?: AbortSignal; cols?: number }> = []
  private readonly channels = new Map<string, MockTerminalProtocolChannel>()
  private readonly subscribers = new Map<string, Set<(event: TerminalProtocolEvent) => void>>()

  async openTerminal(terminalId: string): Promise<MockTerminalProtocolChannel> {
    this.openedTerminalIds.push(terminalId)
    this.openedLabels.push(`proto-terminal:${terminalId}`)
    const channel = new MockTerminalProtocolChannel(`proto-terminal:${terminalId}`)
    this.channels.set(terminalId, channel)
    return channel
  }

  async getConnectionInfo() {
    return {
      path: 'local' as const,
      connectionId: 'mock-connection',
      machineId: 'machine-local',
      relayInUse: false,
    }
  }

  subscribeTerminal(terminalId: string, handler: (event: TerminalProtocolEvent) => void): () => void {
    let handlers = this.subscribers.get(terminalId)
    if (!handlers) {
      handlers = new Set()
      this.subscribers.set(terminalId, handlers)
    }
    handlers.add(handler)
    return () => handlers?.delete(handler)
  }

  closeTerminalChannel(terminalId: string): void {
    this.closedTerminalIds.push(terminalId)
    this.channels.get(terminalId)?.close()
  }

  async loadScrollback(
    terminalId: string,
    offset: number,
    limit: number,
    _alternate?: boolean,
    options?: { signal?: AbortSignal; cols?: number },
  ) {
    this.scrollbackRequests.push({
      terminalId,
      offset,
      limit,
      ...(options?.signal ? { signal: options.signal } : {}),
      ...(options?.cols ? { cols: options.cols } : {}),
    })
    return {
      beforeOffset: offset,
      limit,
      rows: 1,
      replay: 'older',
      hasMore: false,
      alternate: false,
    }
  }

  emit(terminalId: string, event: TerminalProtocolEvent): void {
    for (const handler of this.subscribers.get(terminalId) ?? []) {
      handler(event)
    }
  }

  sentText(terminalId: string): string {
    return this.channels.get(terminalId)?.sentText ?? ''
  }

  sentResize(terminalId: string): { cols: number; rows: number } | undefined {
    return this.channels.get(terminalId)?.sentResize
  }
}

class MockTerminalProtocolChannel {
  readyState: 'connecting' | 'open' | 'closing' | 'closed' = 'open'
  sentText = ''
  sentResize: { cols: number; rows: number } | undefined

  constructor(readonly label: string) {}

  send(data: Uint8Array): void {
    const message = JSON.parse(new TextDecoder().decode(data)) as
      | { type: 'input'; data: string }
      | { type: 'resize'; cols: number; rows: number }
    if (message.type === 'input') {
      this.sentText += message.data
      return
    }
    this.sentResize = { cols: message.cols, rows: message.rows }
  }

  close(): void {
    this.readyState = 'closed'
  }
}

class DeferredTerminalProtocolSession implements TerminalProtocolSession {
  readonly openedTerminalIds: string[] = []
  readonly openedLabels: string[] = []
  readonly closedTerminalIds: string[] = []
  private readonly waiters = new Map<string, (channel: MockTerminalProtocolChannel) => void>()

  async openTerminal(terminalId: string): Promise<MockTerminalProtocolChannel> {
    this.openedTerminalIds.push(terminalId)
    this.openedLabels.push(`proto-terminal:${terminalId}`)
    return new Promise((resolve) => {
      this.waiters.set(terminalId, resolve)
    })
  }

  resolveOpen(terminalId: string): MockTerminalProtocolChannel {
    const channel = new MockTerminalProtocolChannel(`proto-terminal:${terminalId}`)
    const resolve = this.waiters.get(terminalId)
    if (!resolve) throw new Error(`missing pending terminal ${terminalId}`)
    this.waiters.delete(terminalId)
    resolve(channel)
    return channel
  }

  async getConnectionInfo() {
    return {
      path: 'local' as const,
      connectionId: 'mock-connection',
      machineId: 'machine-local',
      relayInUse: false,
    }
  }

  subscribeTerminal(): () => void {
    return () => {}
  }

  async loadScrollback(terminalId: string, offset: number, limit: number) {
    return {
      beforeOffset: offset,
      limit,
      rows: 0,
      replay: '',
      hasMore: false,
      alternate: false,
    }
  }

  closeTerminalChannel(terminalId: string): void {
    this.closedTerminalIds.push(terminalId)
  }
}
