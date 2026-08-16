import { create } from '@bufbuild/protobuf'
import { act, render, waitFor } from '@testing-library/react'
import { useEffect } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { ResourceHandleSchema, ResourceKind } from '../generated/apipb/common_pb'
import {
  HistoryRowSchema,
  HistoryWindowOperation,
  HistoryWindowResultSchema,
  NativeScreenResultSchema,
  ScreenCellSchema,
  ScreenRowReplaceSchema,
  ScreenRowSchema,
} from '../generated/apipb/history_pb'
import {
  AttachmentHandleSchema,
  TerminalAttachResultSchema,
  TerminalGetResultSchema,
  TerminalInfoSchema,
  TerminalRefSchema,
  TerminalSizeSchema,
} from '../generated/apipb/terminal_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { useTerminalSession, type UseTerminalSessionResult } from './useTerminalSession'

describe('useTerminalSession scrollback lifecycle', () => {
  it('cancels an in-flight history operation when the terminal unmounts', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen('', 1n))
        case 'historyWindow':
          return new Promise(() => {})
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalSnapshot).not.toBeNull())

    let pending: Promise<unknown> | undefined
    act(() => {
      pending = current?.loadScrollback(25)
    })
    await waitFor(() => {
      const historyIndex = session.commands.findIndex((entry) => entry.command.case === 'historyWindow')
      expect(session.executeSignals[historyIndex]?.aborted).toBe(false)
    })

    view.unmount()

    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    const historyIndex = session.commands.findIndex((entry) => entry.command.case === 'historyWindow')
    expect(session.executeSignals[historyIndex]?.aborted).toBe(true)
  })

  it('keeps the frozen cursor across live snapshots and releases it on resume', async () => {
    let liveRevision = 1n
    let liveText = 'LIVE-1'
    let historyCalls = 0
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen(liveText, liveRevision))
        case 'historyWindow': {
          historyCalls += 1
          const older = historyCalls > 1
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session),
            token: 'frozen-token',
            operation: older ? HistoryWindowOperation.PREPEND : HistoryWindowOperation.REPLACE,
            rows: [historyRow(older ? 'HISTORY-OLDER' : 'HISTORY-LATEST', older ? 9n : 10n)],
            loadedRows: 1,
            totalRows: 10,
            logicalTotal: 10,
            historyGeneration: 7n,
            firstLineId: older ? 9n : 10n,
            lastLineId: older ? 9n : 10n,
            hasMore: !older,
          }))
        }
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalText).toContain('LIVE-1'))

    await act(async () => {
      await current?.loadScrollback(1, false, 50)
    })
    await waitFor(() => {
      expect(current?.terminalText).toContain('HISTORY-LATEST')
      expect(current?.terminalSnapshot?.history?.loadedRows).toBe(1)
    })

    liveText = 'LIVE-10'
    liveRevision = 10n
    expect(session.commands.filter((entry) => entry.command.case === 'liveScreenNext')).toHaveLength(1)
    expect(current?.terminalText).toContain('HISTORY-LATEST')
    expect(current?.terminalText).not.toContain('LIVE-10')
    expect(current?.terminalSnapshot?.history).toMatchObject({ revision: 1, loadedRows: 1 })

    await act(async () => {
      await current?.loadScrollback(1, false, 50)
    })
    const historyCommands = session.commands.filter((entry) => entry.command.case === 'historyWindow')
    expect(historyCommands).toHaveLength(2)
    expect(historyCommands[1]?.command).toMatchObject({
      case: 'historyWindow',
      value: { token: 'frozen-token', historyGeneration: 7n },
    })

    liveText = 'LIVE-11'
    liveRevision = 11n
    expect(current?.terminalText).toContain('HISTORY-OLDER')
    expect(current?.terminalText).not.toContain('LIVE-11')
    await act(async () => {
      const exhausted = await current?.loadScrollback(1, false, 50)
      expect(exhausted?.hasMore).toBe(false)
    })
    expect(session.commands.filter((entry) => entry.command.case === 'historyWindow')).toHaveLength(2)

    let resumed = ''
    act(() => {
      resumed = current?.resumeLiveScrollback() ?? ''
    })
    expect(resumed).toContain('LIVE-1')
    expect(resumed).not.toContain('HISTORY-LATEST')
    await waitFor(() => expect(current?.terminalSnapshot?.history).toBeUndefined())
    act(() => {
      current?.markLiveScreenSubmitted(1n)
      current?.markLiveScreenCompleted(1n)
    })
    await waitFor(() => expect(current?.terminalText).toContain('LIVE-11'))
    await waitFor(() => {
      expect(session.commands.some((entry) => entry.command.case === 'historyRelease')).toBe(true)
    })
    view.unmount()
  })

  it('does not append the live screen when latest history already contains it', async () => {
    const boundaryText = 'LATEST-INCLUDES-LIVE-SCREEN'
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen(boundaryText, 1n))
        case 'historyWindow':
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session),
            token: 'boundary-token',
            operation: HistoryWindowOperation.REPLACE,
            rows: [historyRow(boundaryText, 1n)],
            loadedRows: 1,
            totalRows: 1,
            logicalTotal: 1,
            historyGeneration: 1n,
            firstLineId: 1n,
            lastLineId: 1n,
            hasMore: false,
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalText).toContain(boundaryText))

    await act(async () => {
      await current?.loadScrollback(1, false, 50)
    })

    await waitFor(() => {
      expect(current?.terminalText.split(boundaryText)).toHaveLength(2)
    })
    view.unmount()
  })

  it('keeps the live screen usable when the latest history window is empty', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen('EMPTY-HISTORY-LIVE', 1n))
        case 'historyWindow':
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session),
            token: 'empty-token',
            operation: HistoryWindowOperation.REPLACE,
            loadedRows: 0,
            totalRows: 0,
            logicalTotal: 0,
            historyGeneration: 1n,
            hasMore: false,
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalText).toContain('EMPTY-HISTORY-LIVE'))

    let result
    await act(async () => {
      result = await current?.loadScrollback(250, false, 50)
    })

    expect(result).toMatchObject({ loadedRows: 0, totalRows: 0, hasMore: false })
    expect(current?.terminalText).toContain('EMPTY-HISTORY-LIVE')
    expect(current?.terminalSnapshot?.history).toMatchObject({ loadedRows: 0, hasMore: false })
    act(() => { current?.resumeLiveScrollback() })
    await waitFor(() => expect(current?.terminalSnapshot?.history).toBeUndefined())
    await waitFor(() => expect(session.commands.some((entry) => entry.command.case === 'historyRelease')).toBe(true))
    view.unmount()
  })

  it('prefetches the first history page without changing the live screen and consumes it on demand', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen('PREFETCH-LIVE', 1n))
        case 'historyWindow':
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session),
            token: 'prefetch-token',
            operation: HistoryWindowOperation.REPLACE,
            rows: [historyRow('PREFETCH-HISTORY', 10n)],
            loadedRows: 1,
            totalRows: 10,
            logicalTotal: 10,
            historyGeneration: 3n,
            firstLineId: 10n,
            lastLineId: 10n,
            hasMore: true,
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalText).toContain('PREFETCH-LIVE'))

    await act(async () => {
      expect(await current?.prefetchScrollback(1, false, 50)).toBe(true)
    })
    expect(current?.terminalText).toContain('PREFETCH-LIVE')
    expect(current?.terminalText).not.toContain('PREFETCH-HISTORY')
    expect(current?.terminalSnapshot?.history).toBeUndefined()

    let result
    await act(async () => {
      current?.freezeScrollback()
      result = await current?.loadScrollback(1, false, 50)
    })
    expect(result).toMatchObject({ loadedRows: 1, prefetched: true })
    await waitFor(() => expect(current?.terminalText).toContain('PREFETCH-HISTORY'))
    expect(session.commands.filter((entry) => entry.command.case === 'historyWindow')).toHaveLength(1)
    view.unmount()
  })

  it('invalidates a prefetched page when the authoritative live screen changes', async () => {
    let liveText = 'LIVE-BEFORE-PREFETCH'
    let liveRevision = 1n
    let historyCalls = 0
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen(liveText, liveRevision))
        case 'historyWindow':
          historyCalls += 1
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session),
            token: `prefetch-token-${historyCalls}`,
            operation: HistoryWindowOperation.REPLACE,
            rows: [historyRow(`HISTORY-${historyCalls}`, BigInt(historyCalls))],
            loadedRows: 1,
            totalRows: 10,
            logicalTotal: 10,
            historyGeneration: BigInt(historyCalls),
            firstLineId: BigInt(historyCalls),
            lastLineId: BigInt(historyCalls),
            hasMore: true,
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalText).toContain('LIVE-BEFORE-PREFETCH'))
    await act(async () => { await current?.prefetchScrollback(1, false, 50) })

    liveText = 'LIVE-AFTER-PREFETCH'
    liveRevision = 2n
    act(() => {
      current?.markLiveScreenSubmitted(1n)
      current?.markLiveScreenCompleted(1n)
    })
    await waitFor(() => expect(current?.terminalText).toContain('LIVE-AFTER-PREFETCH'))

    let result
    await act(async () => {
      current?.freezeScrollback()
      result = await current?.loadScrollback(1, false, 50)
    })
    expect(result).toMatchObject({ loadedRows: 1, prefetched: false })
    expect(historyCalls).toBe(2)
    await waitFor(() => expect(current?.terminalText).toContain('HISTORY-2'))
    view.unmount()
  })

  it('starts a new latest history generation after leaving and re-entering the terminal', async () => {
    let historyCalls = 0
    let attachmentToken = 1
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          attachmentToken += 1
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, attachmentToken, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen(`REENTRY-LIVE-${historyCalls}`, BigInt(historyCalls + 1)))
        case 'historyWindow':
          historyCalls += 1
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session),
            token: `reentry-token-${historyCalls}`,
            operation: HistoryWindowOperation.REPLACE,
            rows: [historyRow(`REENTRY-HISTORY-${historyCalls}`, BigInt(historyCalls))],
            loadedRows: 1,
            totalRows: 100000,
            logicalTotal: 100000,
            historyGeneration: BigInt(historyCalls),
            firstLineId: BigInt(historyCalls),
            lastLineId: BigInt(historyCalls),
            hasMore: true,
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const first = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalSnapshot).not.toBeNull())
    await act(async () => { await current?.loadScrollback(250, false, 50) })
    first.unmount()

    current = undefined
    const second = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalSnapshot).not.toBeNull())
    await act(async () => { await current?.loadScrollback(250, false, 50) })

    const requests = session.commands.filter((entry) => entry.command.case === 'historyWindow')
    expect(requests).toHaveLength(2)
    expect(requests[0]?.command).toMatchObject({ case: 'historyWindow', value: { mode: 1, token: '', historyGeneration: 0n, cols: 50 } })
    expect(requests[1]?.command).toMatchObject({ case: 'historyWindow', value: { mode: 1, token: '', historyGeneration: 0n, cols: 50 } })
    expect(historyCalls).toBe(2)
    second.unmount()
  })

  it('clears a timed-out load so history can be retried', async () => {
    let historyCalls = 0
    let session: MockProtoSession
    session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session), name: 'zsh' }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screen('LIVE', 1n))
        case 'historyWindow':
          historyCalls += 1
          if (historyCalls === 1) return new Promise(() => {})
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session),
            token: 'retry-token',
            operation: HistoryWindowOperation.REPLACE,
            rows: [historyRow('RETRY-HISTORY', 1n)],
            loadedRows: 1,
            totalRows: 1,
            logicalTotal: 1,
            historyGeneration: 2n,
            firstLineId: 1n,
            lastLineId: 1n,
            hasMore: false,
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitFor(() => expect(current?.terminalSnapshot).not.toBeNull())

    vi.useFakeTimers()
    try {
      const timedOut = current!.loadScrollback(1, false, 50)
      const timeoutAssertion = expect(timedOut).rejects.toThrow('timed out')
      await vi.advanceTimersByTimeAsync(10_000)
      await timeoutAssertion

      const retried = await current!.loadScrollback(1, false, 50)
      expect(retried.loadedRows).toBe(1)
      expect(historyCalls).toBe(2)
    } finally {
      vi.useRealTimers()
      view.unmount()
    }
  })
})

function Harness({
  session,
  onChange,
}: {
  session: MockProtoSession
  onChange: (value: UseTerminalSessionResult) => void
}) {
  const value = useTerminalSession({ machineId: 'machine-local', terminalId: 'terminal-1', session })
  useEffect(() => onChange(value), [onChange, value])
  return null
}

function terminalRef(session: MockProtoSession) {
  return create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' })
}

function resource(kind: ResourceKind, token: number, session: MockProtoSession) {
  return create(ResourceHandleSchema, {
    kind,
    opaqueToken: new Uint8Array([token]),
    session: session.stamp,
    generation: 1n,
  })
}

function historyRow(text: string, lineId: bigint) {
  return create(HistoryRowSchema, {
    row: create(ScreenRowSchema, { cells: [create(ScreenCellSchema, { content: text, width: text.length })] }),
    logicalLineId: lineId,
  })
}

function screen(text: string, liveRevision: bigint) {
  return create(NativeScreenResultSchema, {
    size: create(TerminalSizeSchema, { cols: 50, rows: 24 }),
    liveRevision,
    fullReplace: true,
    rowReplacements: text
      ? [create(ScreenRowReplaceSchema, {
          rowIndex: 0,
          row: create(ScreenRowSchema, {
            cells: [create(ScreenCellSchema, { content: text, width: text.length })],
          }),
        })]
      : [],
  })
}
