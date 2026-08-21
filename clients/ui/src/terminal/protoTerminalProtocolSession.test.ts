import { create } from '@bufbuild/protobuf'
import { describe, expect, it, vi } from 'vitest'
import { EventEnvelopeSchema } from '../generated/apipb/application_pb'
import { ResourceHandleSchema, ResourceKind } from '../generated/apipb/common_pb'
import { ApplicationEventType } from '../generated/apipb/events_pb'
import {
  CellStyleSchema,
  HistoryCursorSchema,
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
  ResizeControlReason,
  ResizePolicy,
  TerminalResizeResultSchema,
  TerminalAttachResultSchema,
  TerminalGetResultSchema,
  TerminalInfoSchema,
  TerminalLifecycleEventSchema,
  TerminalRefSchema,
  TerminalSizeSchema,
} from '../generated/apipb/terminal_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { createProtoTerminalProtocolSession } from './protoTerminalProtocolSession'

describe('ProtoTerminalProtocolSession input ordering', () => {
  it('distinguishes explicit owner acquisition from owner release', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-resize-owner', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-1'),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-1') }),
          }))
        case 'terminalResize':
          return protoResult('terminalResize', create(TerminalResizeResultSchema, {
            resizeControl: { canResize: command.command.value.takeOwnership, reason: command.command.value.takeOwnership ? ResizeControlReason.OWNER : ResizeControlReason.FOLLOWER },
          }))
        case 'terminalResizeLock':
          return protoResult('terminalResize', create(TerminalResizeResultSchema, {
            resizeControl: { canResize: false, reason: ResizeControlReason.SIZE_LOCKED, sizeLocked: command.command.value.locked },
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screenResult(session, 'terminal-1'))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    await protocol.openTerminal('terminal-1')

    expect(session.commands.find((entry) => entry.command.case === 'terminalAttach')?.command).toMatchObject({
      case: 'terminalAttach',
      value: { resizePolicy: ResizePolicy.OWNER },
    })

    await protocol.requestResizeOwner?.('terminal-1', { cols: 90, rows: 25 })
    await protocol.releaseResizeOwner?.('terminal-1')
    await protocol.setResizeLock?.('terminal-1', true)

    const resizeCommands = session.commands.filter((entry) => entry.command.case === 'terminalResize')
    expect(resizeCommands.map((entry) => entry.command.case === 'terminalResize' ? entry.command.value.takeOwnership : null)).toEqual([true, false])
    expect(session.commands.find((entry) => entry.command.case === 'terminalResizeLock')?.command).toMatchObject({
      case: 'terminalResizeLock',
      value: { locked: true },
    })
  })

  it('filters terminal events to the terminal being opened', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-scoped-events', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-scoped'),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-scoped') }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screenResult(session, 'terminal-scoped'))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)

    await protocol.openTerminal('terminal-scoped')

    const subscription = session.commands.find((entry) => entry.command.case === 'eventSubscribe')
    expect(subscription?.command).toMatchObject({
      case: 'eventSubscribe',
      value: {
        terminal: { endpointId: 'machine-scoped-events', terminalId: 'terminal-scoped' },
        types: [ApplicationEventType.TERMINAL_LIFECYCLE],
      },
    })
  })

  it('projects lifecycle resize ownership onto the local attachment identity', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-owner-events', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-owner-events'),
              surfaceId: 'mobile-terminal-owner-events',
              viewId: 'mobile-view',
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-owner-events') }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screenResult(session, 'terminal-owner-events'))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const controls: Array<{ reason: string; canResize: boolean; sizeLocked?: boolean }> = []
    protocol.subscribeTerminal('terminal-owner-events', (event) => {
      if (event.type === 'resizeControl') controls.push(event.control)
    })
    await protocol.openTerminal('terminal-owner-events')
    controls.length = 0

    session.emit(create(EventEnvelopeSchema, {
      subscription: resource(ResourceKind.SUBSCRIPTION, 1, session),
      event: { case: 'terminalLifecycle', value: create(TerminalLifecycleEventSchema, {
        terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-owner-events') }),
        attachmentProjection: true,
        resizeEpoch: 5n,
        resizeControl: {
          canResize: true,
          reason: ResizeControlReason.OWNER,
          ownerSurfaceId: 'desktop-terminal-owner-events',
          ownerViewId: 'desktop-view',
        },
      }) },
    }))
    expect(controls.at(-1)).toMatchObject({ reason: 'follower', canResize: false, sizeLocked: false })

    session.emit(create(EventEnvelopeSchema, {
      subscription: resource(ResourceKind.SUBSCRIPTION, 1, session),
      event: { case: 'terminalLifecycle', value: create(TerminalLifecycleEventSchema, {
        terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-owner-events') }),
        attachmentProjection: true,
        resizeEpoch: 6n,
        resizeControl: {
          canResize: false,
          reason: ResizeControlReason.SIZE_LOCKED,
          sizeLocked: true,
          ownerSurfaceId: 'mobile-terminal-owner-events',
          ownerViewId: 'mobile-view',
        },
      }) },
    }))
    expect(controls.at(-1)).toMatchObject({ reason: 'size_locked', canResize: false, sizeLocked: true })

    const controlCount = controls.length
    session.emit(create(EventEnvelopeSchema, {
      subscription: resource(ResourceKind.SUBSCRIPTION, 1, session),
      event: { case: 'terminalLifecycle', value: create(TerminalLifecycleEventSchema, {
        terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-owner-events') }),
        attachmentProjection: true,
        resizeEpoch: 5n,
        resizeControl: {
          canResize: true,
          reason: ResizeControlReason.OWNER,
          ownerSurfaceId: 'desktop-terminal-owner-events',
          ownerViewId: 'desktop-view',
        },
      }) },
    }))
    expect(controls).toHaveLength(controlCount)
    expect(controls.at(-1)).toMatchObject({ reason: 'size_locked', canResize: false, sizeLocked: true })

    session.emit(create(EventEnvelopeSchema, {
      subscription: resource(ResourceKind.SUBSCRIPTION, 1, session),
      event: { case: 'terminalLifecycle', value: create(TerminalLifecycleEventSchema, {
        terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-owner-events') }),
        attachmentProjection: true,
        resizeEpoch: 7n,
        resizeControl: { reason: ResizeControlReason.FOLLOWER },
      }) },
    }))
    expect(controls.at(-1)).toMatchObject({ reason: 'follower', canResize: false, sizeLocked: false })
  })

  it('publishes endpoint identity with terminal metadata', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-info', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, {
              ref: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
              name: 'zsh',
            }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screenResult(session, 'terminal-1'))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const info: Array<Record<string, unknown>> = []
    protocol.subscribeTerminal('terminal-1', (event) => {
      if (event.type === 'info') info.push(event.info)
    })

    await protocol.openTerminal('terminal-1')

    expect(info).toContainEqual(expect.objectContaining({
      terminal_id: 'terminal-1',
      machine_id: 'machine-info',
      name: 'zsh',
    }))
  })

  it('preserves prefixed indexed colors in live screen replay', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-live-color', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-color'),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-color') }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', create(NativeScreenResultSchema, {
            terminal: terminalRef(session, 'terminal-color'),
            liveRevision: 1n,
            size: create(TerminalSizeSchema, { cols: 5, rows: 1 }),
            fullReplace: true,
            rowReplacements: [create(ScreenRowReplaceSchema, {
              rowIndex: 0,
              row: create(ScreenRowSchema, {
                cells: [create(ScreenCellSchema, {
                  content: 'COLOR',
                  width: 5,
                  style: create(CellStyleSchema, {
                    foreground: 'ansi:2',
                    background: 'idx:24',
                    bold: true,
                  }),
                })],
              }),
            })],
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const replays: string[] = []
    protocol.subscribeTerminal('terminal-color', (event) => {
      if (event.type === 'snapshot') replays.push(event.snapshot.screenReplay ?? '')
    })

    await protocol.openTerminal('terminal-color')

    expect(replays.at(-1)).toContain('\u001b[1;38;5;2;48;5;24mCOLOR')
  })

  it('does not render wide-character continuation cells as black gaps', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-live-wide', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-wide'),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-wide') }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', create(NativeScreenResultSchema, {
            terminal: terminalRef(session, 'terminal-wide'),
            liveRevision: 1n,
            size: create(TerminalSizeSchema, { cols: 4, rows: 1 }),
            fullReplace: true,
            rowReplacements: [create(ScreenRowReplaceSchema, {
              rowIndex: 0,
              row: create(ScreenRowSchema, {
                cells: [
                  create(ScreenCellSchema, {
                    content: '现',
                    width: 2,
                    style: create(CellStyleSchema, { background: '#222222' }),
                  }),
                  create(ScreenCellSchema, { content: '', width: 0 }),
                  create(ScreenCellSchema, {
                    content: '在',
                    width: 2,
                    style: create(CellStyleSchema, { background: '#222222' }),
                  }),
                  create(ScreenCellSchema, { content: '', width: 0 }),
                ],
              }),
            })],
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const replays: string[] = []
    protocol.subscribeTerminal('terminal-wide', (event) => {
      if (event.type === 'snapshot') replays.push(event.snapshot.screenReplay ?? '')
    })

    await protocol.openTerminal('terminal-wide')

    expect(replays.at(-1)).toContain('\u001b[48;2;34;34;34m现在')
    expect(replays.at(-1)).not.toContain('现\u001b[0m ')
    expect(replays.at(-1)).not.toContain('在\u001b[0m ')
  })

  it('paginates visual scrollback with the frozen history cursor', async () => {
    let historyCall = 0
    let session: MockProtoSession
    session = new MockProtoSession('machine-history', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'historyWindow': {
          historyCall += 1
          const lineId = historyCall === 1 ? 42n : 41n
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            token: 'history-token',
            operation: historyCall === 1 ? HistoryWindowOperation.REPLACE : HistoryWindowOperation.PREPEND,
            rows: [create(HistoryRowSchema, {
              row: create(ScreenRowSchema, { cells: [create(ScreenCellSchema, { content: `line-${lineId}`, width: 7 })] }),
              logicalLineId: lineId,
              rowInLine: 0,
            })],
            loadedRows: 1,
            totalRows: 42,
            logicalTotal: 42,
            historyGeneration: 7n,
            firstLineId: lineId,
            lastLineId: lineId,
            hasMore: historyCall === 1,
            cursor: create(HistoryCursorSchema, { lineId }),
          }))
        }
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)

    const latest = await protocol.loadScrollback('terminal-1', 0, 1)
    const older = await protocol.loadScrollback('terminal-1', latest.rows, 1)

    expect(latest.replay).toContain('line-42')
    expect(older.replay).toContain('line-41')
    const historyCommands = session.commands.filter((entry) => entry.command.case === 'historyWindow')
    expect(historyCommands[1]?.command).toMatchObject({
      case: 'historyWindow',
      value: {
        mode: 2,
        token: 'history-token',
        historyGeneration: 7n,
        beforeCursor: { lineId: 42n, rowInLine: 0 },
        boundaryFirstLineId: 42n,
        boundaryLastLineId: 42n,
      },
    })
  })

  it('loads frozen alternate-screen history through the same pager', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-alt-history', (command) => {
      if (command.command.case === 'historyWindow') {
        return protoResult('historyWindow', create(HistoryWindowResultSchema, {
          terminal: terminalRef(session, 'terminal-alt'),
          token: 'alt-token',
          operation: HistoryWindowOperation.REPLACE,
          rows: [create(HistoryRowSchema, {
            row: create(ScreenRowSchema, { cells: [create(ScreenCellSchema, { content: 'abcdefghijkl', width: 12 })] }),
            logicalLineId: 1n,
            fixedGrid: true,
            screenCols: 12,
            screenRowSet: true,
          })],
          historyGeneration: 1n,
        }))
      }
      return protoResult('acknowledge', {})
    })
    const protocol = createProtoTerminalProtocolSession(session)

    const page = await protocol.loadScrollback('terminal-alt', 0, 20, true, { cols: 6 })

    expect(page.alternate).toBe(true)
    expect(page.cols).toBe(6)
    expect(page.rows).toBe(1)
    expect(page.replay).toContain('abcdef')
    expect(page.replay).not.toContain('ghijkl')
    expect(session.commands.some((entry) => entry.command.case === 'historyWindow')).toBe(true)
  })

  it('uses the local xterm columns for history instead of the remote PTY size', async () => {
    let session: MockProtoSession
    session = new MockProtoSession('machine-local-cols', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-1'),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, {
              ref: terminalRef(session, 'terminal-1'),
              size: create(TerminalSizeSchema, { cols: 211, rows: 57 }),
            }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', create(NativeScreenResultSchema, {
            terminal: terminalRef(session, 'terminal-1'),
            liveRevision: 1n,
            size: create(TerminalSizeSchema, { cols: 211, rows: 57 }),
            fullReplace: true,
          }))
        case 'historyWindow':
          return protoResult('historyWindow', create(HistoryWindowResultSchema, {
            terminal: terminalRef(session, 'terminal-1'),
            token: 'local-cols-token',
            operation: HistoryWindowOperation.REPLACE,
            rows: [create(HistoryRowSchema, {
              row: create(ScreenRowSchema, { cells: [create(ScreenCellSchema, { content: 'mobile', width: 6 })] }),
              logicalLineId: 1n,
            })],
            loadedRows: 1,
            historyGeneration: 1n,
            hasMore: false,
          }))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    await protocol.openTerminal('terminal-1')

    await protocol.loadScrollback('terminal-1', 0, 250, false, { cols: 50 })

    const historyCommand = session.commands.find((entry) => entry.command.case === 'historyWindow')
    expect(historyCommand?.command).toMatchObject({
      case: 'historyWindow',
      value: { cols: 50 },
    })
  })

  it('owns an asynchronous live screen refresh failure', async () => {
    let liveScreenCalls = 0
    let session: MockProtoSession
    session = new MockProtoSession('machine-live-refresh', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, {
              ref: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            }),
          }))
        case 'liveScreenNext':
          liveScreenCalls += 1
          if (liveScreenCalls > 1) throw new Error('Go binding bridge disconnected')
          return protoResult('liveScreen', screenResult(session, 'terminal-1'))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    await protocol.openTerminal('terminal-1')
    const events: string[] = []
    protocol.subscribeTerminal('terminal-1', (event) => {
      if (event.type === 'closed') events.push(event.reason ?? '')
    })

    protocol.markSyncLost('terminal-1')

    await vi.waitFor(() => expect(events).toEqual(['Go binding bridge disconnected']))
  })

  it('keeps one request in flight and one latest frame pending behind the renderer', async () => {
    let liveScreenCalls = 0
    let completeSecond: (() => void) | undefined
    let thirdRequestStarted: (() => void) | undefined
    let session: MockProtoSession
    session = new MockProtoSession('machine-live-coalescing', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-1'),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-1') }),
          }))
        case 'liveScreenNext': {
          liveScreenCalls += 1
          if (liveScreenCalls === 1) {
            return protoResult('liveScreen', screenResult(session, 'terminal-1', 'one', 1n))
          }
          if (liveScreenCalls === 2) {
            return new Promise((resolve) => {
              completeSecond = () => resolve(protoResult('liveScreen', screenResult(session, 'terminal-1', 'two', 2n)))
            })
          }
          thirdRequestStarted?.()
          return new Promise(() => {})
        }
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const revisions: bigint[] = []
    protocol.subscribeTerminal('terminal-1', (event) => {
      if (event.type === 'snapshot' && event.snapshot.liveRevision !== undefined) {
        revisions.push(event.snapshot.liveRevision)
      }
    })
    await protocol.openTerminal('terminal-1')

    expect(revisions).toEqual([1n])
    protocol.markLiveScreenSubmitted?.('terminal-1', 1n)
    expect(liveScreenCalls).toBe(2)
    completeSecond?.()
    await vi.waitFor(() => expect(revisions).toEqual([1n]))
    protocol.markLiveScreenCompleted?.('terminal-1', 1n)
    await vi.waitFor(() => expect(revisions).toEqual([1n, 2n]))
    expect(liveScreenCalls).toBe(2)

    const thirdStarted = new Promise<void>((resolve) => { thirdRequestStarted = resolve })
    protocol.markLiveScreenSubmitted?.('terminal-1', 2n)
    await thirdStarted
    expect(liveScreenCalls).toBe(3)
    const observed = session.commands
      .filter((entry) => entry.command.case === 'liveScreenNext')
      .map((entry) => entry.command.case === 'liveScreenNext' ? entry.command.value.observedRevision : -1n)
    expect(observed).toEqual([0n, 1n, 2n])
  })

  it('cancels the long poll while hidden and resumes from the canonical revision', async () => {
    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible')
    let liveScreenCalls = 0
    let session: MockProtoSession
    session = new MockProtoSession('machine-visibility', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: terminalRef(session, 'terminal-1'),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, { ref: terminalRef(session, 'terminal-1') }),
          }))
        case 'liveScreenNext':
          liveScreenCalls += 1
          if (liveScreenCalls === 1) {
            return protoResult('liveScreen', screenResult(session, 'terminal-1', 'one', 1n))
          }
          return new Promise(() => {})
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const channel = await protocol.openTerminal('terminal-1')
    protocol.markLiveScreenSubmitted?.('terminal-1', 1n)
    await vi.waitFor(() => expect(liveScreenCalls).toBe(2))
    const secondIndex = session.commands.findLastIndex((entry) => entry.command.case === 'liveScreenNext')

    visibility.mockReturnValue('hidden')
    document.dispatchEvent(new Event('visibilitychange'))
    expect(session.executeSignals[secondIndex]?.aborted).toBe(true)

    visibility.mockReturnValue('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.waitFor(() => expect(liveScreenCalls).toBe(3))
    const liveCommands = session.commands.filter((entry) => entry.command.case === 'liveScreenNext')
    expect(liveCommands.at(-1)?.command).toMatchObject({
      case: 'liveScreenNext',
      value: { observedRevision: 1n },
    })

    channel.close()
    visibility.mockRestore()
  })

  it('waits for each terminal input acknowledgement before sending the next input', async () => {
    const sent: string[] = []
    const acknowledge: Array<() => void> = []
    let session: MockProtoSession
    session = new MockProtoSession('machine-input-order', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, {
              ref: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screenResult(session, 'terminal-1'))
        case 'terminalInput':
          sent.push(new TextDecoder().decode(command.command.value.data))
          return new Promise((resolve) => acknowledge.push(() => resolve(protoResult('acknowledge', {}))))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const channel = await protocol.openTerminal('terminal-1')

    channel.sendInput?.('a')
    channel.sendInput?.('b')
    channel.sendInput?.('c')

    await vi.waitFor(() => expect(sent).toEqual(['a']))
    acknowledge.shift()?.()
    await vi.waitFor(() => expect(sent).toEqual(['a', 'b']))
    acknowledge.shift()?.()
    await vi.waitFor(() => expect(sent).toEqual(['a', 'b', 'c']))
    acknowledge.shift()?.()
  })

  it('drops queued mouse input from an older gesture after cancellation', async () => {
    const sent: string[] = []
    const acknowledge: Array<() => void> = []
    let session: MockProtoSession
    session = new MockProtoSession('machine-input-cancel', (command) => {
      switch (command.command.case) {
        case 'eventSubscribe':
          return protoResult('eventSubscription', { subscription: resource(ResourceKind.SUBSCRIPTION, 1, session) })
        case 'terminalAttach':
          return protoResult('terminalAttach', create(TerminalAttachResultSchema, {
            attachment: create(AttachmentHandleSchema, {
              resource: resource(ResourceKind.ATTACHMENT, 2, session),
              terminal: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            }),
          }))
        case 'terminalGet':
          return protoResult('terminalGet', create(TerminalGetResultSchema, {
            terminal: create(TerminalInfoSchema, {
              ref: create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId: 'terminal-1' }),
            }),
          }))
        case 'liveScreenNext':
          return protoResult('liveScreen', screenResult(session, 'terminal-1'))
        case 'terminalInput':
          sent.push(new TextDecoder().decode(command.command.value.data))
          return new Promise((resolve) => acknowledge.push(() => resolve(protoResult('acknowledge', {}))))
        default:
          return protoResult('acknowledge', {})
      }
    })
    const protocol = createProtoTerminalProtocolSession(session)
    const channel = await protocol.openTerminal('terminal-1')
    const down = '\x1b[<65;1;1M'

    channel.sendInput?.(down)
    channel.sendInput?.(down)
    await vi.waitFor(() => expect(sent).toEqual([down]))
    channel.cancelPendingMouseInput?.()
    channel.sendInput?.(down)

    acknowledge.shift()?.()
    await vi.waitFor(() => expect(sent).toEqual([down, down]))
    acknowledge.shift()?.()
    await Promise.resolve()
    expect(sent).toEqual([down, down])
  })
})

function resource(kind: ResourceKind, token: number, session: MockProtoSession) {
  return create(ResourceHandleSchema, {
    kind,
    opaqueToken: new Uint8Array([token]),
    session: session.stamp,
    generation: 1n,
  })
}

function terminalRef(session: MockProtoSession, terminalId: string) {
  return create(TerminalRefSchema, { endpointId: session.stamp.endpointId, terminalId })
}

function screenResult(
  session: MockProtoSession,
  terminalId: string,
  text = '',
  liveRevision = 1n,
) {
  const cols = Math.max(1, text.length)
  return create(NativeScreenResultSchema, {
    terminal: terminalRef(session, terminalId),
    liveRevision,
    size: create(TerminalSizeSchema, { cols, rows: 1 }),
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
