import { act, cleanup, createEvent, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { createRef } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { ProtoClientSession } from '../core/protoClientSession'
import { Terminal, type TerminalHandle } from './Terminal'
import { DEFAULT_TERMINAL_SETTINGS } from './terminalSettings'
import { ConnectionRecoveryOverlayHost, ConnectionRecoveryOverlayProvider } from '../connection/ConnectionRecoveryOverlay'

interface FakeXTermInstance {
  element: HTMLElement | null
  textarea: HTMLTextAreaElement | null
  bufferLine: { type: 'normal' | 'alternate'; viewportY: number; baseY: number; length: number }
  emitBinary(data: string): void
  emitData(data: string): void
  emitKey(event: KeyboardEvent): boolean
  emitRender(): void
  setMouseModeActive(active: boolean): void
}

const terminalHarness = vi.hoisted(() => ({
  instances: [] as unknown[],
  inputRecoveryFailure: null as string | null,
  unrelatedBanner: false,
  sessionSendInput: vi.fn(),
  sessionSendResize: vi.fn(),
  resizeOwnerRequest: vi.fn(),
  cancelPendingMouseInput: vi.fn(),
  historySnapshot: false,
  historyMetadata: null as null | {
    revision: number
    cols: number
    prependedRows: number
    loadedRows: number
    logicalTotalRows: number
    rowLogicalLineIds: Array<string | undefined>
    rowInLogicalLines: Array<number | undefined>
    rowLogicalStartCols: Array<number | undefined>
    rowTimestampsUnixMs: Array<number | undefined>
    operation?: 'replace' | 'prepend'
    searchMatchRow?: number
    searchMatchRanges?: Array<{ row: number; startCol: number; endCol: number }>
    hasMore: boolean
  },
  historySelection: '',
  historySelectionPosition: undefined as undefined | { start: { x: number; y: number }; end: { x: number; y: number } },
  historyCopy: vi.fn(),
  historySearch: vi.fn(),
  historySearchCancel: vi.fn(),
  channelState: 'open' as 'open' | 'connecting',
  historyLoad: vi.fn(),
  historyReset: vi.fn(),
  historyFreeze: vi.fn(),
  historyResume: vi.fn(() => ''),
  reattach: vi.fn(),
  liveSnapshot: null as null | {
    text: string
    screenReplay: string
    liveReplay: string
    liveRevision: bigint
    liveFullReplace: boolean
    cols: number
    rows: number
    alternateScreen: boolean
  },
  liveSubmitted: vi.fn(),
  liveCompleted: vi.fn(),
  xtermWrites: [] as string[],
  xtermSelections: [] as Array<[number, number, number]>,
  xtermDecorations: [] as Array<{ x?: number; width?: number; height?: number; backgroundColor?: string; foregroundColor?: string; layer?: string }>,
  xtermResets: 0,
  scrollToBottomCalls: 0,
  pendingWriteCallbacks: [] as Array<() => void>,
  autoCompleteWrites: true,
  fitDimensions: { cols: 80, rows: 24 },
  resizeObserverCallback: null as ResizeObserverCallback | null,
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    fit() {}
    proposeDimensions() { return terminalHarness.fitDimensions }
  },
}))

vi.mock('@xterm/addon-canvas', () => ({ CanvasAddon: class {} }))
vi.mock('@xterm/addon-webgl', () => ({ WebglAddon: class {} }))

vi.mock('@xterm/xterm', () => ({
  Terminal: class FakeXTerm {
    cols = 80
    rows = 24
    element: HTMLElement | null = null
    options: Record<string, unknown>
    bufferChangeHandler: (() => void) | null = null
    bufferLine = { type: 'normal', cursorY: 0, viewportY: 0, baseY: 0, length: 24 }
    buffer = {
      active: this.bufferLine,
      normal: this.bufferLine,
      alternate: { ...this.bufferLine, type: 'alternate' },
      onBufferChange: (handler: () => void) => {
        this.bufferChangeHandler = handler
        return { dispose: () => { this.bufferChangeHandler = null } }
      },
    }
    _core = {
      viewport: { scrollBarWidth: 0 },
      coreMouseService: {
        areMouseEventsActive: false,
        triggerMouseEvent: undefined as ((event: {
          col: number
          row: number
          x: number
          y: number
          button: number
          action: number
          ctrl: boolean
          alt: boolean
          shift: boolean
        }) => boolean) | undefined,
      },
      coreService: { decPrivateModes: { applicationCursorKeys: false } },
    }
    dataHandler: ((data: string) => void) | null = null
    binaryHandler: ((data: string) => void) | null = null
    keyHandler: ((event: KeyboardEvent) => boolean) | null = null
    renderHandler: (() => void) | null = null
    textarea: HTMLTextAreaElement | null = null

    constructor(options: Record<string, unknown>) {
      this.options = { ...options }
      terminalHarness.instances.push(this)
    }

    loadAddon() {}
    open(container: HTMLElement) {
      const element = document.createElement('div')
      element.className = 'xterm'
      const screen = document.createElement('div')
      screen.className = 'xterm-screen'
      const textarea = document.createElement('textarea')
      textarea.className = 'xterm-helper-textarea'
      element.append(screen, textarea)
      element.addEventListener('wheel', (event) => {
        if (!this._core.coreMouseService.areMouseEventsActive) return
        this.dataHandler?.(event.deltaY > 0 ? '\x1b[<65;1;1M' : '\x1b[<64;1;1M')
      })
      container.append(element)
      this.element = element
      this.textarea = textarea
    }
    onData(handler: (data: string) => void) {
      this.dataHandler = handler
      return { dispose: () => { this.dataHandler = null } }
    }
    onBinary(handler: (data: string) => void) {
      this.binaryHandler = handler
      return { dispose: () => { this.binaryHandler = null } }
    }
    onCursorMove() { return { dispose() {} } }
    onRender(handler: () => void) {
      this.renderHandler = handler
      return { dispose: () => { this.renderHandler = null } }
    }
    attachCustomKeyEventHandler(handler: (event: KeyboardEvent) => boolean) { this.keyHandler = handler }
    emitBinary(data: string) { this.binaryHandler?.(data) }
    emitData(data: string) { this.dataHandler?.(data) }
    emitBufferChange() { this.bufferChangeHandler?.() }
    emitKey(event: KeyboardEvent) { return this.keyHandler?.(event) ?? true }
    emitRender() { this.renderHandler?.() }
    setMouseModeActive(active: boolean) {
      this._core.coreMouseService.areMouseEventsActive = active
      this.element?.classList.toggle('enable-mouse-events', active)
    }
    resize(cols: number, rows: number) { this.cols = cols; this.rows = rows }
    write(text: string, callback?: () => void) {
      terminalHarness.xtermWrites.push(text)
      if (!callback) return
      if (terminalHarness.autoCompleteWrites) callback()
      else terminalHarness.pendingWriteCallbacks.push(callback)
    }
    reset() { terminalHarness.xtermResets += 1 }
    refresh() {}
    scrollToBottom() {
      terminalHarness.scrollToBottomCalls += 1
      this.bufferLine.viewportY = 0
    }
    scrollToLine(line: number) { this.bufferLine.viewportY = line }
    scrollLines(lines: number) { this.bufferLine.viewportY += lines }
    registerMarker(offset = 0) {
      return { line: this.bufferLine.baseY + this.bufferLine.cursorY + offset, dispose() {} }
    }
    registerDecoration(options: { x?: number; width?: number; height?: number; backgroundColor?: string; foregroundColor?: string; layer?: string }) {
      terminalHarness.xtermDecorations.push(options)
      return { dispose() {} }
    }
    select(column: number, row: number, length: number) {
      terminalHarness.xtermSelections.push([column, row, length])
    }
    selectAll() {}
    getSelection() { return terminalHarness.historySelection }
    getSelectionPosition() { return terminalHarness.historySelectionPosition }
    hasSelection() { return terminalHarness.historySelection !== '' }
    clearSelection() {}
    focus() {}
    blur() {}
    dispose() {}
  },
}))

vi.mock('./useTerminalSession', () => ({
  useTerminalSession: ({ terminalId }: { terminalId: string }) => ({
    snapshot: terminalHarness.unrelatedBanner
      ? {
          phase: 'failed',
          terminalChannels: { [terminalId]: { state: 'open' } },
          visibleError: { message: 'unrelated connection failure', recoverable: true, surface: 'banner' },
        }
      : { phase: 'connected', terminalChannels: { [terminalId]: { state: terminalHarness.channelState } } },
    inputRecoveryFailure: terminalHarness.inputRecoveryFailure,
    terminalSnapshot: terminalHarness.liveSnapshot ?? (terminalHarness.historySnapshot || terminalHarness.historyMetadata
      ? { text: 'live terminal content', cols: 80, rows: 24, alternateScreen: false, ...(terminalHarness.historyMetadata ? { history: terminalHarness.historyMetadata } : {}) }
      : null),
    terminalText: terminalHarness.liveSnapshot?.screenReplay ?? (terminalHarness.historySnapshot ? 'live terminal content' : ''),
    terminalInfo: null,
    resizeControl: { canResize: false, reason: 'follower' },
    sendInput: terminalHarness.sessionSendInput,
    cancelPendingMouseInput: terminalHarness.cancelPendingMouseInput,
    sendResize: terminalHarness.sessionSendResize,
    requestResizeOwner: terminalHarness.resizeOwnerRequest,
    releaseResizeOwner: async () => ({ canResize: false, reason: 'follower' }),
    loadScrollback: terminalHarness.historyLoad,
    searchScrollback: terminalHarness.historySearch,
    copyScrollback: terminalHarness.historyCopy,
    cancelHistorySearch: terminalHarness.historySearchCancel,
    prefetchScrollback: async () => false,
    resetScrollback: terminalHarness.historyReset,
    freezeScrollback: terminalHarness.historyFreeze,
    resumeLiveScrollback: terminalHarness.historyResume,
    markSyncLost: () => {},
    markLiveScreenSubmitted: terminalHarness.liveSubmitted,
    markLiveScreenCompleted: terminalHarness.liveCompleted,
    handleAppResume: () => {},
    reattach: terminalHarness.reattach,
    client: null,
  }),
}))

const session = {} as ProtoClientSession

describe('Terminal input modifier boundary', () => {
  beforeEach(() => {
    terminalHarness.instances.length = 0
    terminalHarness.inputRecoveryFailure = null
    terminalHarness.unrelatedBanner = false
    terminalHarness.historySnapshot = false
    terminalHarness.historyMetadata = null
    terminalHarness.historySelection = ''
    terminalHarness.historySelectionPosition = undefined
    terminalHarness.historyCopy.mockReset()
    terminalHarness.historySearch.mockReset().mockResolvedValue({ found: false, wrapped: false })
    terminalHarness.historySearchCancel.mockReset()
    terminalHarness.liveSnapshot = null
    terminalHarness.channelState = 'open'
    terminalHarness.xtermWrites.length = 0
    terminalHarness.xtermSelections.length = 0
    terminalHarness.xtermDecorations.length = 0
    terminalHarness.xtermResets = 0
    terminalHarness.scrollToBottomCalls = 0
    terminalHarness.pendingWriteCallbacks.length = 0
    terminalHarness.autoCompleteWrites = true
    terminalHarness.fitDimensions = { cols: 80, rows: 24 }
    terminalHarness.resizeObserverCallback = null
    terminalHarness.liveSubmitted.mockReset()
    terminalHarness.liveCompleted.mockReset()
    terminalHarness.sessionSendInput.mockReset().mockReturnValue(true)
    terminalHarness.sessionSendResize.mockReset().mockReturnValue(false)
    terminalHarness.resizeOwnerRequest.mockReset().mockResolvedValue({ canResize: true, reason: 'owner' })
    terminalHarness.cancelPendingMouseInput.mockReset().mockReturnValue(true)
    terminalHarness.historyLoad.mockReset().mockResolvedValue({ loadedRows: 0, totalRows: 0, hasMore: false, alternate: false })
    terminalHarness.historyReset.mockReset()
    terminalHarness.reattach.mockReset()
    terminalHarness.historyFreeze.mockReset()
    terminalHarness.historyResume.mockReset().mockReturnValue('')
    vi.stubGlobal('ResizeObserver', class {
      constructor(callback: ResizeObserverCallback) { terminalHarness.resizeObserverCallback = callback }
      observe() {}
      disconnect() {}
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('applies an ASCII modifier in the custom key handler and updates once state before the next synchronous key', async () => {
    const onInput = vi.fn(() => true)
    const onModifierStateChange = vi.fn()
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      modifierState={{ ctrl: 'once', alt: 'off' }}
      onModifierStateChange={onModifierStateChange}
      onInput={onInput}
      renderer="dom"
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const first = createEvent.keyDown(document.body, { key: 'c' }) as KeyboardEvent
    const second = createEvent.keyDown(document.body, { key: 'c' }) as KeyboardEvent

    act(() => {
      expect(xterm.emitKey(first)).toBe(false)
      expect(xterm.emitKey(second)).toBe(true)
      xterm.emitData('c')
    })

    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['\x03', 'c'])
    expect(onModifierStateChange).toHaveBeenCalledOnce()
    expect(onModifierStateChange).toHaveBeenCalledWith({ ctrl: 'off', alt: 'off' })
  })

  it('retries a rejected custom-key modifier through onData and consumes it after acceptance', async () => {
    const onInput = vi.fn()
      .mockReturnValueOnce(false)
      .mockReturnValueOnce(true)
    const onModifierStateChange = vi.fn()
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      modifierState={{ ctrl: 'once', alt: 'off' }}
      onModifierStateChange={onModifierStateChange}
      onInput={onInput}
      renderer="dom"
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const event = () => createEvent.keyDown(document.body, { key: 'c' }) as KeyboardEvent

    act(() => {
      expect(xterm.emitKey(event())).toBe(true)
      xterm.emitData('c')
    })
    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['\x03', '\x03'])
    expect(onModifierStateChange).toHaveBeenCalledOnce()
    expect(onModifierStateChange).toHaveBeenCalledWith({ ctrl: 'off', alt: 'off' })
  })

  it('applies modifiers to Android-style onData ASCII while leaving IME and paste batches raw', async () => {
    const onInput = vi.fn(() => true)
    const onModifierStateChange = vi.fn()
    const ref = createRef<TerminalHandle>()
    render(<Terminal
      ref={ref}
      machineId="studio"
      terminalId="term-shell"
      session={session}
      modifierState={{ ctrl: 'once', alt: 'once' }}
      onModifierStateChange={onModifierStateChange}
      onInput={onInput}
      renderer="dom"
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance

    act(() => {
      xterm.emitData('d')
      xterm.emitData('中')
      xterm.emitData('p')
      xterm.emitData('paste batch')
      xterm.emitData('\x1b[Z')
    })
    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['\x1b\x04', '中', 'p', 'paste batch', '\x1b[Z'])
    expect(onModifierStateChange).toHaveBeenCalledOnce()
    expect(onModifierStateChange).toHaveBeenCalledWith({ ctrl: 'off', alt: 'off' })

    expect(ref.current?.pasteText('p')).toBe(true)
    expect(ref.current?.pasteText('paste text')).toBe(true)
    expect(terminalHarness.sessionSendInput.mock.calls.map(([data]) => data)).toEqual(['p', 'paste text'])
    expect(onModifierStateChange).toHaveBeenCalledOnce()
  })

  it('keeps an onData navigation once modifier until the target accepts it', async () => {
    const onInput = vi.fn()
      .mockReturnValueOnce(false)
      .mockReturnValueOnce(true)
    const onModifierStateChange = vi.fn()
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      modifierState={{ ctrl: 'once', alt: 'off' }}
      onModifierStateChange={onModifierStateChange}
      onInput={onInput}
      renderer="dom"
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance

    act(() => {
      xterm.emitData('\x1b[C')
      xterm.emitData('\x1b[C')
    })

    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['\x1b[1;5C', '\x1b[1;5C'])
    expect(onModifierStateChange).toHaveBeenCalledOnce()
    expect(onModifierStateChange).toHaveBeenCalledWith({ ctrl: 'off', alt: 'off' })
  })

  it('keeps locked modifiers across custom ASCII keys and onData navigation', async () => {
    const onInput = vi.fn(() => true)
    const onModifierStateChange = vi.fn()
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      modifierState={{ ctrl: 'locked', alt: 'locked' }}
      onModifierStateChange={onModifierStateChange}
      onInput={onInput}
      renderer="dom"
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance

    act(() => {
      expect(xterm.emitKey(createEvent.keyDown(document.body, { key: 'c' }) as KeyboardEvent)).toBe(false)
      expect(xterm.emitKey(createEvent.keyDown(document.body, { key: 'c' }) as KeyboardEvent)).toBe(false)
      xterm.emitData('d')
      xterm.emitData('d')
      xterm.emitData('\x1b[D')
      xterm.emitData('\x1b[D')
    })

    expect(onInput.mock.calls.map(([data]) => data)).toEqual([
      '\x1b\x03',
      '\x1b\x03',
      '\x1b\x04',
      '\x1b\x04',
      '\x1b[1;7D',
      '\x1b[1;7D',
    ])
    expect(onModifierStateChange).not.toHaveBeenCalled()
  })

  it('leaves composition keyCode 229 and Unicode custom keys to onData without consuming once', async () => {
    const onInput = vi.fn(() => true)
    const onModifierStateChange = vi.fn()
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      modifierState={{ ctrl: 'once', alt: 'once' }}
      onModifierStateChange={onModifierStateChange}
      onInput={onInput}
      renderer="dom"
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance

    fireEvent.compositionStart(xterm.textarea!, { data: '' })
    expect(xterm.emitKey(createEvent.keyDown(xterm.textarea!, {
      key: 'a',
      keyCode: 65,
      isComposing: true,
    }) as KeyboardEvent)).toBe(true)
    expect(xterm.emitKey(createEvent.keyDown(xterm.textarea!, {
      key: 'Process',
      keyCode: 229,
      isComposing: false,
    }) as KeyboardEvent)).toBe(true)
    act(() => xterm.emitData('中'))
    fireEvent.compositionEnd(xterm.textarea!, { data: '中' })

    expect(xterm.emitKey(createEvent.keyDown(xterm.textarea!, { key: '中' }) as KeyboardEvent)).toBe(true)
    act(() => xterm.emitData('中'))
    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['中', '中'])
    expect(onModifierStateChange).not.toHaveBeenCalled()

    act(() => expect(xterm.emitKey(createEvent.keyDown(xterm.textarea!, { key: 'c' }) as KeyboardEvent)).toBe(false))
    expect(onInput).toHaveBeenLastCalledWith('\x1b\x03')
    expect(onModifierStateChange).toHaveBeenCalledWith({ ctrl: 'off', alt: 'off' })
  })

  it('reports rejected onData and binary sends through the same owner boundary without consuming once', async () => {
    const onInput = vi.fn(() => false)
    const onModifierStateChange = vi.fn()
    terminalHarness.inputRecoveryFailure = 'input blocked'
    render(
      <ConnectionRecoveryOverlayProvider appIntent={null}>
        <div className="relative h-96">
          <Terminal
            machineId="studio"
            terminalId="term-shell"
            session={session}
            modifierState={{ ctrl: 'once', alt: 'off' }}
            onModifierStateChange={onModifierStateChange}
            onInput={onInput}
            renderer="dom"
          />
          <ConnectionRecoveryOverlayHost />
        </div>
      </ConnectionRecoveryOverlayProvider>,
    )
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance

    act(() => {
      xterm.emitData('paste')
      xterm.emitBinary('binary')
    })

    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['paste', 'binary'])
    expect(onModifierStateChange).not.toHaveBeenCalled()
    const overlay = await screen.findByTestId('anytty-connection-recovery-overlay')
    expect(overlay.getAttribute('role')).toBe('status')
    expect(overlay.textContent).toContain('Input is paused until the connection recovers.')
  })

  it('shows a terminal channel failure without mislabeling it as paused input', async () => {
    terminalHarness.unrelatedBanner = true
    render(
      <ConnectionRecoveryOverlayProvider appIntent={null}>
        <div className="relative h-96">
          <Terminal
            machineId="studio"
            terminalId="term-shell"
            session={session}
            renderer="dom"
          />
          <ConnectionRecoveryOverlayHost />
        </div>
      </ConnectionRecoveryOverlayProvider>,
    )

    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const alert = screen.getByRole('alert')
    expect(alert.textContent).toContain('unrelated connection failure')
    expect(alert.textContent).not.toContain('Input is paused')
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(terminalHarness.reattach).toHaveBeenCalledWith(session, { forceTerminalChannel: true })
  })

  it('isolates the terminal application from focus and assistive technology while connecting', async () => {
    terminalHarness.channelState = 'connecting'
    render(
      <ConnectionRecoveryOverlayProvider appIntent={null}>
        <div className="relative h-96">
          <Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />
          <ConnectionRecoveryOverlayHost />
        </div>
      </ConnectionRecoveryOverlayProvider>,
    )

    const status = await screen.findByTestId('anytty-connection-recovery-overlay')
    const application = screen.getByRole('application', { hidden: true })

    expect(status.getAttribute('aria-live')).toBe('polite')
    expect(status.textContent).toContain('Connecting terminal...')
    expect(application.getAttribute('aria-hidden')).toBe('true')
    expect(application.hasAttribute('inert')).toBe(true)
    expect(application.getAttribute('tabindex')).toBe('-1')
  })

  it('shows stale history recovery and does not retry until reload is pressed', async () => {
    terminalHarness.historySnapshot = true
    terminalHarness.historyLoad
      .mockRejectedValueOnce(Object.assign(new Error('expired token'), { code: 'stale_resource' }))
      .mockResolvedValueOnce({ loadedRows: 0, totalRows: 0, hasMore: false, alternate: false })
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const output = (terminalHarness.instances[0] as FakeXTermInstance).element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    fireEvent.wheel(output, { deltaY: -1 })
    const alert = await screen.findByTestId('anytty-history-error')
    expect(alert.getAttribute('role')).toBe('alert')
    expect(screen.getByRole('button', { name: 'Reload history' }).className).toContain('min-h-11')
    expect(terminalHarness.historyLoad).toHaveBeenCalledTimes(1)

    fireEvent.wheel(output, { deltaY: -1 })
    await Promise.resolve()
    expect(terminalHarness.historyLoad).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByRole('button', { name: 'Reload history' }))
    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledTimes(2))
    expect(terminalHarness.historyReset).toHaveBeenCalledOnce()
  })

  it('returns to live when the first history request is empty', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const output = (terminalHarness.instances[0] as FakeXTermInstance).element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    fireEvent.wheel(output, { deltaY: -1 })

    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledOnce())
    await waitFor(() => expect(terminalHarness.historyResume).toHaveBeenCalledOnce())
    expect(terminalHarness.historyFreeze).toHaveBeenCalledOnce()
  })

  it('freezes and preloads history while copy selection mode is active', async () => {
    terminalHarness.historySnapshot = true
    const view = render(
      <Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />,
    )
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))

    view.rerender(
      <Terminal
        machineId="studio"
        terminalId="term-shell"
        session={session}
        renderer="dom"
        selectionMode
      />,
    )

    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledOnce())
    expect(terminalHarness.historyFreeze).toHaveBeenCalledOnce()
    expect(terminalHarness.historyResume).not.toHaveBeenCalled()

    view.rerender(
      <Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />,
    )
    await waitFor(() => expect(terminalHarness.historyResume).toHaveBeenCalledOnce())
  })

  it('resumes live after an in-flight copy-mode history page finishes applying', async () => {
    terminalHarness.historySnapshot = true
    terminalHarness.historyLoad.mockResolvedValue({
      loadedRows: 1,
      totalRows: 1,
      hasMore: false,
      alternate: false,
    })
    const view = render(
      <Terminal
        machineId="studio"
        terminalId="term-shell"
        session={session}
        renderer="dom"
        selectionMode
      />,
    )
    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledOnce())

    view.rerender(
      <Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />,
    )
    expect(terminalHarness.historyResume).not.toHaveBeenCalled()

    terminalHarness.historyMetadata = historyMetadata({ revision: 1 })
    view.rerender(
      <Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />,
    )

    await waitFor(() => expect(terminalHarness.historyResume).toHaveBeenCalledOnce())
  })

  it('loads older pages when a copy selection reaches the top and preserves it after prepend', async () => {
    terminalHarness.historySnapshot = true
    terminalHarness.historyLoad.mockResolvedValue({
      loadedRows: 1,
      totalRows: 1,
      hasMore: true,
      alternate: false,
    })
    const view = render(
      <Terminal
        machineId="studio"
        terminalId="term-shell"
        session={session}
        renderer="dom"
        selectionMode
      />,
    )
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledOnce())
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    stubTerminalFrameBounds(xterm)

    terminalHarness.historyMetadata = historyMetadata({ revision: 1, hasMore: true })
    view.rerender(
      <Terminal
        machineId="studio"
        terminalId="term-shell"
        session={session}
        renderer="dom"
        selectionMode
      />,
    )
    await waitFor(() => expect(terminalHarness.xtermResets).toBeGreaterThan(0))
    await waitFor(() => expect(terminalHarness.historyLoad.mock.calls.length).toBeGreaterThan(1))
    const historyLoadsBeforeTopSelection = terminalHarness.historyLoad.mock.calls.length

    vi.useFakeTimers()
    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 100 }] })
    fireEvent.touchEnd(output)
    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 80 }] })
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 1 }] })
    act(() => vi.advanceTimersByTime(80))

    expect(terminalHarness.historyLoad).toHaveBeenCalledTimes(historyLoadsBeforeTopSelection + 1)
    fireEvent.touchEnd(output)
    terminalHarness.historySelection = 'selected history'
    terminalHarness.historySelectionPosition = {
      start: { x: 1, y: 2 },
      end: { x: 4, y: 3 },
    }
    terminalHarness.historyMetadata = historyMetadata({
      revision: 2,
      operation: 'prepend',
      prependedRows: 2,
      loadedRows: 3,
      hasMore: false,
    })
    view.rerender(
      <Terminal
        machineId="studio"
        terminalId="term-shell"
        session={session}
        renderer="dom"
        selectionMode
      />,
    )
    act(() => vi.runOnlyPendingTimers())

    expect(terminalHarness.xtermSelections.at(-1)).toEqual([1, 4, 83])
    expect(terminalHarness.historyResume).not.toHaveBeenCalled()
  })

  it('does not show history loading when the request finishes within two seconds', async () => {
    terminalHarness.historySnapshot = true
    let resolveLoad: ((result: { loadedRows: number; totalRows: number; hasMore: boolean; alternate: boolean }) => void) | undefined
    terminalHarness.historyLoad.mockImplementationOnce(() => new Promise((resolve) => { resolveLoad = resolve }))
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const output = (terminalHarness.instances[0] as FakeXTermInstance).element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    vi.useFakeTimers()
    fireEvent.wheel(output, { deltaY: -1 })
    expect(terminalHarness.historyLoad).toHaveBeenCalledOnce()
    act(() => vi.advanceTimersByTime(1_999))
    expect(screen.queryByTestId('anytty-history-loading')).toBeNull()

    await act(async () => {
      resolveLoad?.({ loadedRows: 0, totalRows: 0, hasMore: false, alternate: false })
      await Promise.resolve()
    })
    expect(screen.queryByTestId('anytty-history-loading')).toBeNull()
  })

  it('shows a labeled spinner only after history loading exceeds two seconds', async () => {
    terminalHarness.historySnapshot = true
    let resolveLoad: ((result: { loadedRows: number; totalRows: number; hasMore: boolean; alternate: boolean }) => void) | undefined
    terminalHarness.historyLoad.mockImplementationOnce(() => new Promise((resolve) => { resolveLoad = resolve }))
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const output = (terminalHarness.instances[0] as FakeXTermInstance).element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    vi.useFakeTimers()
    fireEvent.wheel(output, { deltaY: -1 })
    act(() => vi.advanceTimersByTime(2_000))

    const loading = screen.getByTestId('anytty-history-loading')
    expect(loading.textContent).toContain('Loading terminal history')
    expect(loading.querySelector('svg')?.classList.contains('animate-spin')).toBe(true)

    await act(async () => {
      resolveLoad?.({ loadedRows: 0, totalRows: 0, hasMore: false, alternate: false })
      await Promise.resolve()
      vi.advanceTimersByTime(180)
    })
    expect(screen.queryByTestId('anytty-history-loading')).toBeNull()
  })

  it('routes both touch scroll directions to an alternate-screen application', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    xterm.bufferLine.type = 'alternate'

    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 100 }] })
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 70 }] })
    fireEvent.touchEnd(output)
    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 70 }] })
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 100 }] })
    fireEvent.touchEnd(output)

    expect(terminalHarness.sessionSendInput).toHaveBeenNthCalledWith(1, '\x1b[B', { cols: 80, rows: 24 })
    expect(terminalHarness.sessionSendInput).toHaveBeenNthCalledWith(2, '\x1b[A', { cols: 80, rows: 24 })
    expect(terminalHarness.historyLoad).not.toHaveBeenCalled()
    expect(terminalHarness.historyFreeze).not.toHaveBeenCalled()
  })

  it('handles alternate-screen touch gestures that hit the xterm root instead of its screen', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    if (!xterm.element) throw new Error('missing terminal root')
    xterm.bufferLine.type = 'alternate'

    fireEvent.touchStart(xterm.element, { touches: [{ clientX: 20, clientY: 100 }] })
    fireEvent.touchMove(xterm.element, { touches: [{ clientX: 20, clientY: 60 }] })
    fireEvent.touchEnd(xterm.element)

    expect(terminalHarness.sessionSendInput).toHaveBeenCalledWith('\x1b[B', { cols: 80, rows: 24 })
  })

  it('falls back to the xterm root when mouse mode starts before the screen element is available', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    if (!xterm.element) throw new Error('missing terminal root')
    xterm.bufferLine.type = 'alternate'
    xterm.setMouseModeActive(true)
    xterm.element.querySelector('.xterm-screen')?.remove()

    fireEvent.touchStart(xterm.element, { touches: [{ clientX: 20, clientY: 100 }] })
    fireEvent.touchMove(xterm.element, { touches: [{ clientX: 20, clientY: 60 }] })
    fireEvent.touchEnd(xterm.element)

    expect(terminalHarness.sessionSendInput).toHaveBeenCalledWith('\x1b[<65;1;1M')
  })

  it('uses xterm mouse protocol encoding for TUI touch scrolling when available', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    if (!xterm.element) throw new Error('missing terminal root')
    xterm.bufferLine.type = 'alternate'
    xterm.setMouseModeActive(true)
    const triggerMouseEvent = vi.fn((event: { action: number }) => {
      xterm.emitData(event.action === 1 ? '\x1b[<65;1;1M' : '\x1b[<64;1;1M')
      return true
    })
    xterm._core.coreMouseService.triggerMouseEvent = triggerMouseEvent

    fireEvent.touchStart(xterm.element, { touches: [{ clientX: 20, clientY: 100 }] })
    fireEvent.touchMove(xterm.element, { touches: [{ clientX: 20, clientY: 60 }] })
    fireEvent.touchEnd(xterm.element)

    expect(triggerMouseEvent).toHaveBeenCalledWith(expect.objectContaining({ button: 4, action: 1 }))
    expect(terminalHarness.sessionSendInput).toHaveBeenCalledWith('\x1b[<65;1;1M')
  })

  it('stops alternate-screen scrolling at touch release when inertia is disabled', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      renderer="dom"
      settings={{ ...DEFAULT_TERMINAL_SETTINGS, scrollInertia: 0 }}
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    xterm.bufferLine.type = 'alternate'
    xterm.setMouseModeActive(true)
    vi.useFakeTimers()
    vi.setSystemTime(1_000)

    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 120 }] })
    vi.setSystemTime(1_016)
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 60 }] })
    vi.setSystemTime(1_020)
    fireEvent.touchEnd(output)
    const callsAtRelease = terminalHarness.sessionSendInput.mock.calls.length
    act(() => vi.advanceTimersByTime(500))

    expect(callsAtRelease).toBeGreaterThan(0)
    expect(terminalHarness.sessionSendInput).toHaveBeenCalledTimes(callsAtRelease)
  })

  it('paces alternate-screen momentum after touch release when inertia is enabled', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      renderer="dom"
      settings={{ ...DEFAULT_TERMINAL_SETTINGS, scrollInertia: 100 }}
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    xterm.bufferLine.type = 'alternate'
    xterm.setMouseModeActive(true)
    vi.useFakeTimers()
    vi.setSystemTime(2_000)

    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 120 }] })
    vi.setSystemTime(2_016)
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 60 }] })
    vi.setSystemTime(2_020)
    fireEvent.touchEnd(output)
    const callsAtRelease = terminalHarness.sessionSendInput.mock.calls.length
    act(() => vi.advanceTimersByTime(500))

    expect(callsAtRelease).toBeGreaterThan(0)
    expect(terminalHarness.sessionSendInput.mock.calls.length).toBeGreaterThan(callsAtRelease)
  })

  it('keeps TUI momentum alive when the application redraws after a wheel report', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      renderer="dom"
      settings={{ ...DEFAULT_TERMINAL_SETTINGS, scrollInertia: 100 }}
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    xterm.bufferLine.type = 'alternate'
    xterm.setMouseModeActive(true)
    vi.useFakeTimers()
    vi.setSystemTime(3_000)

    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 120 }] })
    vi.setSystemTime(3_016)
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 60 }] })
    vi.setSystemTime(3_020)
    fireEvent.touchEnd(output)
    const callsAtRelease = terminalHarness.sessionSendInput.mock.calls.length

    xterm.emitBufferChange()
    act(() => vi.advanceTimersByTime(500))

    expect(callsAtRelease).toBeGreaterThan(0)
    expect(terminalHarness.sessionSendInput.mock.calls.length).toBeGreaterThan(callsAtRelease)
  })

  it('keeps TUI momentum alive when a live snapshot resets transient viewport state', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      renderer="dom"
      settings={{ ...DEFAULT_TERMINAL_SETTINGS, scrollInertia: 100 }}
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    xterm.bufferLine.type = 'alternate'
    xterm.setMouseModeActive(true)
    vi.useFakeTimers()
    vi.setSystemTime(4_000)

    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 120 }] })
    vi.setSystemTime(4_016)
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 60 }] })
    vi.setSystemTime(4_020)
    fireEvent.touchEnd(output)
    const callsAtRelease = terminalHarness.sessionSendInput.mock.calls.length

    act(() => terminalHarness.resizeObserverCallback?.([], {} as ResizeObserver))
    act(() => vi.advanceTimersByTime(500))

    expect(callsAtRelease).toBeGreaterThan(0)
    expect(terminalHarness.sessionSendInput.mock.calls.length).toBeGreaterThan(callsAtRelease)
  })

  it('stops TUI momentum and cancels queued wheel reports on a new touch', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      renderer="dom"
      settings={{ ...DEFAULT_TERMINAL_SETTINGS, scrollInertia: 100 }}
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    xterm.bufferLine.type = 'alternate'
    xterm.setMouseModeActive(true)
    vi.useFakeTimers()
    vi.setSystemTime(5_000)

    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 120 }] })
    vi.setSystemTime(5_016)
    fireEvent.touchMove(output, { touches: [{ clientX: 20, clientY: 60 }] })
    vi.setSystemTime(5_020)
    fireEvent.touchEnd(output)
    act(() => vi.advanceTimersByTime(100))
    const callsBeforeInterruption = terminalHarness.sessionSendInput.mock.calls.length

    fireEvent.touchStart(output, { touches: [{ clientX: 20, clientY: 80 }] })
    act(() => vi.advanceTimersByTime(500))

    expect(callsBeforeInterruption).toBeGreaterThan(0)
    expect(terminalHarness.cancelPendingMouseInput).toHaveBeenCalledTimes(2)
    expect(terminalHarness.sessionSendInput).toHaveBeenCalledTimes(callsBeforeInterruption)
  })

  it('does not enter history from wheel input while the alternate screen is active', async () => {
    terminalHarness.historySnapshot = true
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')
    xterm.bufferLine.type = 'alternate'

    fireEvent.wheel(output, { deltaY: -1 })
    await Promise.resolve()

    expect(terminalHarness.historyLoad).not.toHaveBeenCalled()
    expect(terminalHarness.historyFreeze).not.toHaveBeenCalled()
  })

  it('dismisses a nonretryable oversized history line without offering reload', async () => {
    terminalHarness.historySnapshot = true
    terminalHarness.historyLoad.mockRejectedValueOnce(Object.assign(new Error('bounded response exceeded'), {
      code: 'resource_exhausted',
      retryable: false,
    }))
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const output = (terminalHarness.instances[0] as FakeXTermInstance).element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    fireEvent.wheel(output, { deltaY: -1 })
    const alert = await screen.findByTestId('anytty-history-error')
    expect(alert.textContent).toContain('A terminal history line is too large to display.')
    expect(screen.queryByRole('button', { name: 'Reload history' })).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }))
    await waitFor(() => expect(screen.queryByTestId('anytty-history-error')).toBeNull())
    fireEvent.wheel(output, { deltaY: -1 })
    await Promise.resolve()
    expect(terminalHarness.historyLoad).toHaveBeenCalledTimes(1)
    expect(terminalHarness.historyReset).not.toHaveBeenCalled()
  })

  it('reloads frozen history with the new local columns after resize', async () => {
    terminalHarness.historySnapshot = true
    terminalHarness.historyLoad.mockResolvedValue({ loadedRows: 1, totalRows: 1, cols: 80, hasMore: true, alternate: false })
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const output = (terminalHarness.instances[0] as FakeXTermInstance).element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    fireEvent.wheel(output, { deltaY: -1 })
    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledTimes(1))
    terminalHarness.fitDimensions = { cols: 40, rows: 24 }
    act(() => terminalHarness.resizeObserverCallback?.([], {} as ResizeObserver))

    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledTimes(2))
    expect(terminalHarness.historyReset).toHaveBeenCalledOnce()
    expect(terminalHarness.historyLoad).toHaveBeenLastCalledWith(expect.any(Number), false, 40)
  })

  it('does not report or leave history when a paging request is superseded', async () => {
    terminalHarness.historySnapshot = true
    let rejectLoad: ((reason: unknown) => void) | undefined
    terminalHarness.historyLoad.mockImplementationOnce(() => new Promise((_, reject) => { rejectLoad = reject }))
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const output = (terminalHarness.instances[0] as FakeXTermInstance).element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    fireEvent.wheel(output, { deltaY: -1 })
    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledOnce())
    act(() => rejectLoad?.(new DOMException('superseded by search', 'AbortError')))

    await waitFor(() => expect(screen.queryByTestId('anytty-history-error')).toBeNull())
    expect(terminalHarness.historyResume).not.toHaveBeenCalled()
  })

  it('keeps a search result in history until the user scrolls back to the bottom', async () => {
    terminalHarness.historySnapshot = true
    terminalHarness.historyLoad.mockResolvedValue({ loadedRows: 1, totalRows: 1, cols: 80, hasMore: false, alternate: false })
    const view = render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const output = xterm.element?.querySelector('.xterm-screen')
    if (!output) throw new Error('missing terminal screen')

    fireEvent.wheel(output, { deltaY: -1 })
    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledOnce())
    terminalHarness.historyMetadata = historyMetadata({ revision: 1 })
    view.rerender(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.xtermResets).toBeGreaterThan(0))
    fireEvent.wheel(output, { deltaY: -1 })
    xterm.bufferLine.baseY = 2
    xterm.bufferLine.viewportY = 1
    act(() => xterm.emitRender())

    terminalHarness.historyMetadata = historyMetadata({
      revision: 2,
      operation: 'replace',
      searchMatchRow: 0,
      searchMatchRanges: [{ row: 0, startCol: 2, endCol: 7 }],
    })
    view.rerender(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.xtermResets).toBeGreaterThan(1))
    expect(terminalHarness.xtermDecorations).toContainEqual(expect.objectContaining({
      x: 2,
      width: 5,
      height: 1,
      backgroundColor: '#F59E0B',
      foregroundColor: '#111827',
      layer: 'top',
    }))
    expect(terminalHarness.historyResume).not.toHaveBeenCalled()
    act(() => xterm.emitRender())
    expect(terminalHarness.historyResume).not.toHaveBeenCalled()

    xterm.bufferLine.baseY = 0
    xterm.bufferLine.viewportY = 0
    fireEvent.wheel(output, { deltaY: 1 })
    await waitFor(() => expect(terminalHarness.historyResume).toHaveBeenCalledOnce())
  })

  it('keeps the logical history status visible across search UI rerenders', async () => {
    terminalHarness.historySnapshot = true
    terminalHarness.historyMetadata = historyMetadata({ revision: 1, loadedRows: 2, logicalTotalRows: 1 })
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    xterm.bufferLine.viewportY = 1
    act(() => xterm.emitRender())

    const status = await screen.findByTestId('anytty-history-position')
    await waitFor(() => expect(status.classList.contains('flex')).toBe(true))
    expect(status.hasAttribute('hidden')).toBe(false)
    expect(status.classList.contains('hidden')).toBe(false)
    expect(status.textContent).toContain('/ 1')
    expect(screen.queryByTestId('anytty-history-search')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: 'Search history' }))
    expect(await screen.findByTestId('anytty-history-search')).toBeTruthy()
    expect(status.classList.contains('flex')).toBe(true)
  })

  it('keeps the history search trigger visually hidden until history status is available', async () => {
    render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))

    const status = screen.getByTestId('anytty-history-position')
    expect(status.hasAttribute('hidden')).toBe(false)
    expect(status.classList.contains('hidden')).toBe(true)
    expect(status.classList.contains('flex')).toBe(false)
    expect(screen.queryByTestId('anytty-history-search')).toBeNull()
  })

  it('opens history search from the imperative handle before history metadata is loaded', async () => {
    const ref = createRef<TerminalHandle>()
    render(<Terminal ref={ref} machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(ref.current).not.toBeNull())

    act(() => ref.current?.openHistorySearch())
    const input = await screen.findByRole('textbox', { name: 'Search history' })
    expect(screen.getByRole('button', { name: 'Text' }).getAttribute('aria-pressed')).toBe('true')
    fireEvent.click(screen.getByRole('button', { name: 'Regex' }))
    fireEvent.change(input, { target: { value: 'current prompt' } })
    fireEvent.click(screen.getByRole('button', { name: 'Next match' }))

    await waitFor(() => expect(terminalHarness.historyLoad).toHaveBeenCalledOnce())
    await waitFor(() => expect(terminalHarness.historySearch).toHaveBeenCalledWith(
      'current prompt',
      'forward',
      80,
      undefined,
      'regex',
    ))
    expect(await screen.findByText('No matches')).toBeTruthy()
  })

  it('returns the underlying acceptance result from imperative input and paste handles', async () => {
    terminalHarness.sessionSendInput.mockReturnValue(false)
    const ref = createRef<TerminalHandle>()
    render(<Terminal ref={ref} machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(ref.current).not.toBeNull())

    expect(ref.current?.sendInput('key')).toBe(false)
    expect(ref.current?.pasteText('paste')).toBe(false)
  })

  it('waits for a trustworthy local viewport before acquiring resize ownership', async () => {
    const ref = createRef<TerminalHandle>()
    render(<Terminal ref={ref} machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(ref.current).not.toBeNull())

    terminalHarness.fitDimensions = { cols: 1, rows: 1 }
    let animationFrame = 0
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      animationFrame += 1
      if (animationFrame === 2) terminalHarness.fitDimensions = { cols: 91, rows: 31 }
      callback(animationFrame)
      return animationFrame
    })

    await act(async () => { await ref.current!.requestResizeOwner() })

    expect(terminalHarness.resizeOwnerRequest).toHaveBeenCalledWith({ cols: 91, rows: 31 })
  })

  it('does not acquire resize ownership with a stale remote grid', async () => {
    const ref = createRef<TerminalHandle>()
    render(<Terminal ref={ref} machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(ref.current).not.toBeNull())

    terminalHarness.fitDimensions = { cols: 1, rows: 1 }
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      callback(0)
      return 1
    })

    await expect(ref.current!.requestResizeOwner()).rejects.toThrow('terminal viewport size is unavailable')

    expect(terminalHarness.resizeOwnerRequest).not.toHaveBeenCalled()
  })

  it('copies frozen history by logical range instead of reading the rendered text buffer', async () => {
    terminalHarness.historyMetadata = {
      revision: 1,
      cols: 80,
      prependedRows: 2,
      loadedRows: 2,
      logicalTotalRows: 20,
      rowLogicalLineIds: ['10', '11'],
      rowInLogicalLines: [1, 0],
      rowLogicalStartCols: [79, 0],
      rowTimestampsUnixMs: [undefined, undefined],
      hasMore: true,
    }
    terminalHarness.historySelection = 'rendered fallback'
    terminalHarness.historySelectionPosition = { start: { x: 2, y: 0 }, end: { x: 5, y: 1 } }
    terminalHarness.historyCopy.mockResolvedValue('authoritative history text')
    const ref = createRef<TerminalHandle>()
    render(<Terminal ref={ref} machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(ref.current).not.toBeNull())

    await expect(ref.current!.getSelectionForClipboard()).resolves.toBe('authoritative history text')
    expect(terminalHarness.historyCopy).toHaveBeenCalledWith({
      startLineId: '10',
      startCol: 81,
      endLineId: '11',
      endCol: 5,
    }, 80)
  })

  it('applies live deltas without cloning the screen and stops bottom anchoring after two paints', async () => {
    terminalHarness.autoCompleteWrites = false
    const frames: FrameRequestCallback[] = []
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      frames.push(callback)
      return frames.length
    })
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation(() => {})
    const view = render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    stubTerminalFrameBounds(xterm)
    act(() => {
      while (frames.length > 0) frames.shift()?.(0)
    })
    const scrollsBeforeWrite = terminalHarness.scrollToBottomCalls

    terminalHarness.liveSnapshot = {
      text: 'canonical frame',
      screenReplay: 'canonical frame',
      liveReplay: 'incremental frame',
      liveRevision: 7n,
      liveFullReplace: false,
      cols: 80,
      rows: 24,
      alternateScreen: false,
    }
    view.rerender(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)

    await waitFor(() => expect(terminalHarness.liveSubmitted).toHaveBeenCalledWith(7n))
    expect(terminalHarness.xtermWrites.at(-1)).toBe('incremental frame')
    expect(terminalHarness.liveCompleted).not.toHaveBeenCalled()
    expect(terminalHarness.xtermResets).toBe(0)
    expect(xterm.element?.querySelector('[data-anytty-terminal-frame-hold]')).toBeNull()

    act(() => terminalHarness.pendingWriteCallbacks.shift()?.())
    expect(terminalHarness.liveCompleted).toHaveBeenCalledWith(7n)
    expect(terminalHarness.scrollToBottomCalls).toBe(scrollsBeforeWrite + 1)
    expect(frames).toHaveLength(1)

    act(() => frames.shift()?.(0))
    expect(frames).toHaveLength(1)
    act(() => frames.shift()?.(16))
    expect(frames).toHaveLength(0)
    expect(terminalHarness.scrollToBottomCalls).toBe(scrollsBeforeWrite + 3)
  })

  it('holds the previous terminal frame while applying a full live replacement', async () => {
    terminalHarness.autoCompleteWrites = false
    const view = render(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    stubTerminalFrameBounds(xterm)

    terminalHarness.liveSnapshot = {
      text: 'canonical frame',
      screenReplay: 'canonical frame',
      liveReplay: 'canonical frame',
      liveRevision: 8n,
      liveFullReplace: true,
      cols: 80,
      rows: 24,
      alternateScreen: false,
    }
    view.rerender(<Terminal machineId="studio" terminalId="term-shell" session={session} renderer="dom" />)

    await waitFor(() => expect(terminalHarness.liveSubmitted).toHaveBeenCalledWith(8n))
    expect(terminalHarness.xtermResets).toBe(1)
    expect(xterm.element?.querySelector('[data-anytty-terminal-frame-hold]')).not.toBeNull()
  })

  it('leaves unsupported key events to xterm instead of consuming a Ctrl once state', async () => {
    const onInput = vi.fn(() => true)
    const onModifierStateChange = vi.fn()
    render(<Terminal
      machineId="studio"
      terminalId="term-shell"
      session={session}
      modifierState={{ ctrl: 'once', alt: 'off' }}
      onModifierStateChange={onModifierStateChange}
      onInput={onInput}
      renderer="dom"
    />)
    await waitFor(() => expect(terminalHarness.instances).toHaveLength(1))
    const xterm = terminalHarness.instances[0] as FakeXTermInstance
    const event = createEvent.keyDown(document.body, { key: '1' }) as KeyboardEvent

    expect(xterm.emitKey(event)).toBe(true)
    expect(onInput).not.toHaveBeenCalled()
    expect(onModifierStateChange).not.toHaveBeenCalled()
  })
})

function historyMetadata(overrides: Partial<NonNullable<typeof terminalHarness.historyMetadata>> = {}): NonNullable<typeof terminalHarness.historyMetadata> {
  return {
    revision: 1,
    cols: 80,
    prependedRows: 1,
    loadedRows: 1,
    logicalTotalRows: 1,
    rowLogicalLineIds: ['1'],
    rowInLogicalLines: [0],
    rowLogicalStartCols: [0],
    rowTimestampsUnixMs: [undefined],
    hasMore: false,
    ...overrides,
  }
}

function stubTerminalFrameBounds(xterm: FakeXTermInstance): void {
  if (!xterm.element) throw new Error('missing terminal element')
  const output = xterm.element.querySelector('.xterm-screen') as HTMLElement | null
  if (!output) throw new Error('missing terminal screen')
  vi.spyOn(xterm.element, 'getBoundingClientRect').mockReturnValue(rect(0, 0, 320, 200))
  vi.spyOn(output, 'getBoundingClientRect').mockReturnValue(rect(0, 0, 320, 200))
}

function rect(x: number, y: number, width: number, height: number): DOMRect {
  return {
    x,
    y,
    width,
    height,
    top: y,
    left: x,
    right: x + width,
    bottom: y + height,
    toJSON: () => ({}),
  }
}
