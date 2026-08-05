import { describe, expect, it } from 'vitest'
import {
  CoreV2HistorySurfaceStaleError,
  createCoreV2HistorySurface,
  type CoreV2HistorySurfaceSnapshot,
} from './coreV2HistorySurface'
import type { CoreV2HistorySource } from './coreV2HistorySource'
import type {
  CoreV2HistoryCopyRequest,
  CoreV2HistoryCell,
  CoreV2HistoryLineSpan,
  CoreV2HistoryRow,
  CoreV2HistorySearchResult,
  CoreV2HistoryWindow,
  CoreV2HistoryWindowRequest,
} from './coreV2TerminalProtocol'

describe('CoreV2HistorySurface', () => {
  it('loads latest history and prepends older windows through logical cursors', async () => {
    const source = new MockHistorySource([
      historyWindow({ startLine: 42, count: 3, token: 'token-1', generation: '7', hasMore: true }),
      historyWindow({ startLine: 40, count: 2, token: 'token-1', generation: '7', op: 'prepend', hasMore: false }),
    ])
    const surface = createCoreV2HistorySurface(source, {
      terminalId: 'terminal-1',
      cols: 80,
      viewportRows: 2,
      requestRows: 3,
      renderOverscanRows: 1,
      edgePrefetchRows: 0,
    })

    const latest = await surface.loadLatest()
    expect(rowTexts(latest)).toEqual(['line-42', 'line-43', 'line-44'])
    expect(latest.token).toBe('token-1')
    expect(latest.generation).toBe('7')
    expect(latest.firstCursor).toEqual({ lineId: '42', rowInLine: 0 })
    expect(latest.lastCursor).toEqual({ lineId: '44', rowInLine: 0 })
    expect(latest.renderRows.map((row) => row.logicalLineId)).toEqual(['42', '43', '44'])
    expect(latest.renderWindow).toEqual(expect.objectContaining({
      startIndex: 0,
      endIndex: 3,
      viewportStartIndex: 1,
      viewportEndIndex: 3,
      viewportOffsetRows: 1,
    }))

    const older = await surface.loadOlder(2)
    expect(rowTexts(older)).toEqual(['line-40', 'line-41', 'line-42', 'line-43', 'line-44'])
    expect(older.renderRows.map((row) => row.logicalLineId)).toEqual(['42', '43', '44'])
    expect(older.renderWindow.viewportStartIndex).toBe(3)
    expect(older.hasOlder).toBe(false)
    expect(older.hasNewer).toBe(false)
    expect(older.lines.map((line) => line.logicalLineId)).toEqual(['40', '41', '42', '43', '44'])
    expect(older.rows[2]?.cells[0]?.style).toEqual(expect.objectContaining({ fg: 'ansi:42', bold: true }))
    expect(source.requests).toEqual([
      expect.objectContaining({
        terminalId: 'terminal-1',
        mode: 'latest',
        limit: 3,
        cols: 80,
      }),
      expect.objectContaining({
        terminalId: 'terminal-1',
        mode: 'older',
        limit: 2,
        cols: 80,
        token: 'token-1',
        generation: '7',
        beforeCursor: { lineId: '42', rowInLine: 0 },
        boundaryFirstLineId: '42',
        boundaryLastLineId: '44',
      }),
    ])
  })

  it('keeps a render window around the viewport and marks trimmed sides reloadable', async () => {
    const source = new MockHistorySource([
      historyWindow({ startLine: 90, count: 8, token: 'token-2', generation: '8', hasMore: true, cols: 100 }),
    ])
    const surface = createCoreV2HistorySurface(source, {
      terminalId: 'terminal-1',
      cols: 100,
      viewportRows: 3,
      requestRows: 8,
      renderOverscanRows: 2,
      edgePrefetchRows: 1,
      cacheRetainRows: 1,
    })

    const latest = await surface.loadLatest()
    expect(rowTexts(latest)).toEqual(['line-94', 'line-95', 'line-96', 'line-97'])
    expect(latest.hasOlder).toBe(true)
    expect(latest.hasNewer).toBe(false)
    expect(latest.renderWindow.shouldPrefetchOlder).toBe(true)

    const moved = surface.scrollToCachedRow(0)
    expect(rowTexts(moved)).toEqual(['line-94', 'line-95', 'line-96', 'line-97'])
    expect(moved.renderRows.map((row) => row.logicalLineId)).toEqual(['94', '95', '96', '97'])
    expect(moved.renderWindow).toEqual(expect.objectContaining({
      startIndex: 0,
      viewportStartIndex: 0,
      viewportEndIndex: 3,
      viewportOffsetRows: 0,
      shouldPrefetchOlder: true,
    }))
  })

  it('invalidates the App cache when a follow-up window changes token or generation', async () => {
    const source = new MockHistorySource([
      historyWindow({ startLine: 42, count: 2, token: 'token-1', generation: '7', hasMore: true }),
      historyWindow({ startLine: 40, count: 1, token: 'token-2', generation: '8', op: 'prepend', hasMore: false }),
    ])
    const surface = createCoreV2HistorySurface(source, {
      terminalId: 'terminal-1',
      cols: 80,
      viewportRows: 2,
    })

    await surface.loadLatest()
    await expect(surface.loadOlder(1)).rejects.toBeInstanceOf(CoreV2HistorySurfaceStaleError)
    expect(surface.snapshot()).toEqual(expect.objectContaining({
      stale: true,
      rows: [],
      renderRows: [],
      token: null,
      generation: null,
      staleReason: expect.stringContaining('token/generation'),
    }))
  })

  it('deduplicates concurrent identical window requests', async () => {
    const source = new MockHistorySource([
      historyWindow({ startLine: 1, count: 2, token: 'token-3', generation: '9', hasMore: false }),
    ])
    const surface = createCoreV2HistorySurface(source, {
      terminalId: 'terminal-1',
      cols: 80,
      viewportRows: 2,
    })

    await Promise.all([surface.loadLatest({ limit: 2 }), surface.loadLatest({ limit: 2 })])
    expect(source.requests).toHaveLength(1)
  })
})

class MockHistorySource implements CoreV2HistorySource {
  readonly requests: CoreV2HistoryWindowRequest[] = []
  private readonly windows: CoreV2HistoryWindow[]

  constructor(windows: CoreV2HistoryWindow[]) {
    this.windows = windows
  }

  async window(request: CoreV2HistoryWindowRequest): Promise<CoreV2HistoryWindow> {
    this.requests.push({ ...request })
    const window = this.windows.shift()
    if (!window) throw new Error(`unexpected history.window request ${request.mode}`)
    return window
  }

  async copy(_request: CoreV2HistoryCopyRequest): Promise<string> {
    throw new Error('surface cache tests must not copy through history source')
  }

  async search(): Promise<CoreV2HistorySearchResult> {
    throw new Error('surface cache tests must not search through history source')
  }
}

function historyWindow(input: {
  startLine: number
  count: number
  token: string
  generation: string
  op?: 'replace' | 'prepend' | 'append' | undefined
  hasMore: boolean
  terminalId?: string | undefined
  cols?: number | undefined
}): CoreV2HistoryWindow {
  const terminalId = input.terminalId ?? 'terminal-1'
  const cols = input.cols ?? 80
  const renderRows: CoreV2HistoryRow[] = []
  const lines: CoreV2HistoryLineSpan[] = []
  for (let index = 0; index < input.count; index += 1) {
    const lineId = String(input.startLine + index)
    renderRows.push({
      index,
      cells: [historyCell(`line-${lineId}`, lineId)],
      logicalLineId: lineId,
      rowInLine: 0,
      kind: 'output',
      wrapped: false,
      ownership: 'terminal',
    })
    lines.push({
      startRow: index,
      endRow: index,
      logicalLineId: lineId,
      rowKind: 'output',
      clippedBefore: false,
      clippedAfter: false,
    })
  }
  return {
    terminalId,
    token: input.token,
    op: input.op ?? 'replace',
    cols,
    renderRows,
    lines,
    totalRows: input.startLine + input.count,
    logicalTotal: input.startLine + input.count,
    hasMore: input.hasMore,
    generation: input.generation,
    firstLineId: String(input.startLine),
    lastLineId: String(input.startLine + input.count - 1),
  }
}

function historyCell(text: string, lineId: string): CoreV2HistoryCell {
  return {
    text,
    width: text.length,
    style: {
      fg: `ansi:${lineId}`,
      bold: true,
    },
  }
}

function rowTexts(snapshot: CoreV2HistorySurfaceSnapshot): string[] {
  return snapshot.rows.map((row) => row.cells.map((cell) => cell.text).join(''))
}
