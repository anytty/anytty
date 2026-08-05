import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import {
  CellStyleSchema,
  HistoryCopyResultSchema,
  HistoryCursorSchema,
  HistoryLineSpanSchema,
  HistoryRangeSchema,
  HistoryRowSchema,
  HistorySearchDirection,
  HistorySearchResultSchema,
  HistoryTextPositionSchema,
  HistoryViewportAnchorSchema,
  HistoryWindowOperation,
  HistoryWindowResultSchema,
  RowOwnership,
  ScreenCellSchema,
  ScreenRowSchema,
} from '../generated/apipb/history_pb'
import { TerminalRefSchema, TerminalSizeSchema } from '../generated/apipb/terminal_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { createCoreV2HistorySource } from './coreV2HistorySource'

describe('CoreV2HistorySource generated Proto API', () => {
  it('dispatches endpoint-aware history windows and projects the result', async () => {
    const session = new MockProtoSession('machine-local', () => protoResult('historyWindow', create(HistoryWindowResultSchema, {
      token: 'hist-token', operation: HistoryWindowOperation.REPLACE, size: create(TerminalSizeSchema, { cols: 80, rows: 24 }),
      loadedRows: 0, totalRows: 3, loadedLines: 0, logicalTotal: 3, hasMore: true, historyGeneration: 7n,
      firstLineId: 42n, lastLineId: 43n, cursor: create(HistoryCursorSchema, { lineId: 42n }),
      viewportAnchor: create(HistoryViewportAnchorSchema, {
        topLineId: 42n, topCellOffset: 17, screenCols: 80, screenRows: 24,
      }),
    })))
    const source = createCoreV2HistorySource(session, 'machine-local')

    const result = await source.window({ terminalId: 'terminal-1', mode: 'latest', limit: 20, cols: 80 })

    expect(result).toMatchObject({
      token: 'hist-token', generation: '7', totalRows: 3, logicalTotal: 3, hasMore: true,
      viewportAnchor: { topLineId: '42', topCellOffset: 17, screenCols: 80, screenRows: 24 },
    })
    expect(session.commands[0]?.command).toMatchObject({
      case: 'historyWindow',
      value: { terminal: { endpointId: 'machine-local', terminalId: 'terminal-1' }, limit: 20, cols: 80 },
    })
  })

  it('rejects endpoint mismatches before dispatch', () => {
    const session = new MockProtoSession('machine-local')
    expect(() => createCoreV2HistorySource(session, 'machine-other')).toThrow(/machine-local.*machine-other/)
    expect(session.commands).toEqual([])
  })

  it('forwards cancellation to the generated Proto execute operation', async () => {
    const session = new MockProtoSession('machine-local', () => protoResult('historyWindow', create(HistoryWindowResultSchema, {
      token: 'hist-token', operation: HistoryWindowOperation.REPLACE, historyGeneration: 1n,
    })))
    const source = createCoreV2HistorySource(session, 'machine-local')
    const controller = new AbortController()

    await source.window(
      { terminalId: 'terminal-1', mode: 'latest', limit: 20, cols: 80 },
      { signal: controller.signal },
    )

    const historyIndex = session.commands.findIndex((entry) => entry.command.case === 'historyWindow')
    expect(session.executeSignals[historyIndex]).toBe(controller.signal)
  })

  it('copies a frozen range in bounded chunks and restores inter-chunk newlines', async () => {
    let chunk = 0
    const session = new MockProtoSession('machine-local', (command) => {
      expect(command.command.case).toBe('historyCopy')
      chunk += 1
      return chunk === 1
        ? protoResult('historyCopy', create(HistoryCopyResultSchema, {
            text: 'first\nsecond',
            next: create(HistoryTextPositionSchema, { lineId: 43n }),
            done: false,
          }))
        : protoResult('historyCopy', create(HistoryCopyResultSchema, { text: 'third', done: true }))
    })
    const source = createCoreV2HistorySource(session, 'machine-local')

    const copied = await source.copy({
      terminalId: 'terminal-1', token: 'hist-token', generation: '7', cols: 80,
      boundaryFirstLineId: '41', boundaryLastLineId: '43',
      range: { startLineId: '41', startCol: 2, endLineId: '43', endCol: 5 },
    })

    expect(copied).toBe('first\nsecond\nthird')
    expect(session.commands).toHaveLength(2)
    expect(session.commands[0]?.command).toMatchObject({
      case: 'historyCopy',
      value: {
        maxLines: 8192, maxBytes: 524288,
        window: { range: { startLineId: 41n, startCol: 2, endLineId: 43n, endCol: 5 } },
      },
    })
    expect(session.commands[1]?.command).toMatchObject({
      case: 'historyCopy',
      value: { window: { range: { startLineId: 43n, startCol: 0, endLineId: 43n, endCol: 5 } } },
    })
  })

  it('searches a frozen snapshot forward and backward and projects match windows', async () => {
    const session = new MockProtoSession('machine-local', (command) => {
      if (command.command.case !== 'historySearch') throw new Error('unexpected command')
      const backward = command.command.value.direction === HistorySearchDirection.BACKWARD
      const lineId = backward ? 4n : 18n
      return protoResult('historySearch', create(HistorySearchResultSchema, {
        found: true,
        match: create(HistoryRangeSchema, { startLineId: lineId, startCol: 8, endLineId: lineId, endCol: 14 }),
        window: create(HistoryWindowResultSchema, {
          terminal: create(TerminalRefSchema, { endpointId: 'machine-local', terminalId: 'terminal-1' }),
          token: 'hist-token', operation: HistoryWindowOperation.REPLACE,
          size: create(TerminalSizeSchema, { cols: 80 }), historyGeneration: 7n,
          firstLineId: lineId, lastLineId: lineId,
        }),
        wrapped: backward,
      }))
    })
    const source = createCoreV2HistorySource(session, 'machine-local')
    const common = { terminalId: 'terminal-1', token: 'hist-token', generation: 7n, query: 'needle', cols: 80, limit: 24 }

    const forward = await source.search({ ...common, direction: 'forward', start: { lineId: '5', col: 0 } })
    const backward = await source.search({ ...common, direction: 'backward', start: { lineId: '18', col: 8 } })

    expect(forward).toMatchObject({ found: true, match: { startLineId: '18' }, window: { token: 'hist-token' }, wrapped: false })
    expect(backward).toMatchObject({ found: true, match: { startLineId: '4' }, window: { token: 'hist-token' }, wrapped: true })
    expect(session.commands.map((entry) => entry.command)).toMatchObject([
      { case: 'historySearch', value: { historyGeneration: 7n, direction: HistorySearchDirection.FORWARD, start: { lineId: 5n, col: 0 } } },
      { case: 'historySearch', value: { historyGeneration: 7n, direction: HistorySearchDirection.BACKWARD, start: { lineId: 18n, col: 8 } } },
    ])
  })

  it('returns an explicit no-match result and rejects wrong result cases', async () => {
    const noMatchSession = new MockProtoSession('machine-local', () => protoResult(
      'historySearch',
      create(HistorySearchResultSchema, { found: false }),
    ))
    const noMatchSource = createCoreV2HistorySource(noMatchSession, 'machine-local')
    await expect(noMatchSource.search({
      terminalId: 'terminal-1', token: 'hist-token', query: 'missing', direction: 'forward', cols: 80, limit: 24,
    })).resolves.toEqual({ found: false, wrapped: false })

    const wrongResultSource = createCoreV2HistorySource(new MockProtoSession('machine-local'), 'machine-local')
    await expect(wrongResultSource.search({
      terminalId: 'terminal-1', token: 'hist-token', query: 'needle', direction: 'forward', cols: 80, limit: 24,
    })).rejects.toThrow(/search returned no result/)
    await expect(wrongResultSource.copy({
      terminalId: 'terminal-1', token: 'hist-token', cols: 80,
      range: { startLineId: '1', startCol: 0, endLineId: '1', endCol: 1 },
    })).rejects.toThrow(/copy returned no result/)
  })

  it('projects generated Proto history rows, styles, and line spans', async () => {
    const session = new MockProtoSession('machine-local', () => protoResult('historyWindow', create(HistoryWindowResultSchema, {
      terminal: create(TerminalRefSchema, { endpointId: 'machine-local', terminalId: 'terminal-1' }),
      token: 'hist-token',
      operation: HistoryWindowOperation.PREPEND,
      size: create(TerminalSizeSchema, { cols: 80, rows: 24 }),
      rows: [create(HistoryRowSchema, {
        row: create(ScreenRowSchema, {
          cells: [create(ScreenCellSchema, {
            content: 'ERR',
            width: 3,
            style: create(CellStyleSchema, { foreground: 'ansi:1', background: '#102030', bold: true }),
          })],
          tailFill: create(CellStyleSchema, { background: 'idx:24' }),
        }),
        rowKind: 'output',
        ownership: RowOwnership.PERSISTED,
        logicalLineId: 42n,
        rowInLine: 1,
      })],
      lines: [create(HistoryLineSpanSchema, {
        startRow: 0,
        endRow: 0,
        rowKind: 'output',
        logicalLineId: 42n,
        clippedBefore: true,
      })],
      loadedRows: 1,
      totalRows: 10,
      loadedLines: 1,
      logicalTotal: 10,
      hasMore: true,
      historyGeneration: 7n,
      firstLineId: 42n,
      lastLineId: 42n,
      cursor: create(HistoryCursorSchema, { lineId: 42n, rowInLine: 1 }),
    })))
    const source = createCoreV2HistorySource(session, 'machine-local')

    const result = await source.window({ terminalId: 'terminal-1', mode: 'older', limit: 20, cols: 80, token: 'hist-token', beforeCursor: { lineId: '43', rowInLine: 0 } })

    expect(result).toMatchObject({
      terminalId: 'terminal-1',
      op: 'prepend',
      renderRows: [{
        logicalLineId: '42',
        rowInLine: 1,
        ownership: 'persisted',
        cells: [{ text: 'ERR', width: 3, style: { fg: 'ansi:1', bg: '#102030', bold: true } }],
        tailFillStyle: { bg: 'idx:24' },
      }],
      lines: [{ logicalLineId: '42', startRow: 0, endRow: 0, clippedBefore: true }],
    })
  })
})
