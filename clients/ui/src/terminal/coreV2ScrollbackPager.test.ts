import { describe, expect, it } from 'vitest'
import { Terminal } from '@xterm/xterm'
import { coreV2HistoryRowsANSI, coreV2ReflowHistoryRows } from './coreV2HistoryANSI'
import type { CoreV2HistorySource } from './coreV2HistorySource'
import { CoreV2ScrollbackPager } from './coreV2ScrollbackPager'
import type {
  CoreV2HistoryCopyRequest,
  CoreV2HistorySearchRequest,
  CoreV2HistorySearchResult,
  CoreV2HistorySearchScanRequest,
  CoreV2HistorySearchScanResult,
  CoreV2HistoryWindow,
  CoreV2HistoryWindowRequest,
} from './coreV2TerminalProtocol'

describe('CoreV2ScrollbackPager', () => {
  it('continues older pages with the frozen token, generation, cursor, and boundaries', async () => {
    const source = new MockSource([
      window({ lineId: '42', token: 'token-1', generation: '7', hasMore: true }),
      window({ lineId: '41', token: 'token-1', generation: '7', op: 'prepend', hasMore: false }),
    ])
    const pager = new CoreV2ScrollbackPager(source)

    const latest = await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })
    const older = await pager.load({ terminalId: 'terminal-1', offset: latest.totalLoadedRows, limit: 1, cols: 80 })

    expect(older.totalLoadedRows).toBe(2)
    expect(older.rows[0]?.logicalLineId).toBe('41')
    expect(source.requests[1]).toEqual(expect.objectContaining({
      mode: 'older',
      token: 'token-1',
      generation: '7',
      beforeCursor: { lineId: '42', rowInLine: 0 },
      boundaryFirstLineId: '42',
      boundaryLastLineId: '42',
    }))
    expect(latest.operation).toBe('replace')
    expect(older.operation).toBe('prepend')
  })

  it('replaces a stale frozen window when the caller resets to offset zero', async () => {
    const source = new MockSource([
      window({ lineId: '10', token: 'token-1', generation: '1', hasMore: false }),
      window({ lineId: '20', token: 'token-2', generation: '2', hasMore: true }),
    ])
    const pager = new CoreV2ScrollbackPager(source)

    await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })
    const refreshed = await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })

    expect(refreshed).toMatchObject({
      operation: 'replace',
      totalLoadedRows: 1,
      historyGeneration: '2',
    })
    expect(source.releases).toEqual([{ terminalId: 'terminal-1', token: 'token-1', generation: '1' }])
  })

  it('starts a new frozen generation when the local viewport columns change', async () => {
    const source = new MockSource([
      window({ lineId: '10', token: 'wide-token', generation: '1', hasMore: true }),
      window({ lineId: '20', token: 'narrow-token', generation: '2', hasMore: true }),
    ])
    const pager = new CoreV2ScrollbackPager(source)

    await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })
    const resized = await pager.load({ terminalId: 'terminal-1', offset: 1, limit: 1, cols: 50 })

    expect(resized.operation).toBe('replace')
    expect(source.requests[1]).toEqual(expect.objectContaining({ mode: 'latest', cols: 50 }))
    expect(source.releases).toEqual([{ terminalId: 'terminal-1', token: 'wide-token', generation: '1' }])
  })

  it('drops a stale token and only reloads after an explicit latest request', async () => {
    const stale = Object.assign(new Error('history expired'), { code: 'stale_resource' })
    const source = new MockSource([
      window({ lineId: '10', token: 'token-1', generation: '1', hasMore: true }),
      stale,
      window({ lineId: '20', token: 'token-2', generation: '2', hasMore: false }),
    ])
    const pager = new CoreV2ScrollbackPager(source)

    const latest = await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })
    await expect(pager.load({ terminalId: 'terminal-1', offset: latest.totalLoadedRows, limit: 1, cols: 80 })).rejects.toBe(stale)
    expect(source.releases).toEqual([{ terminalId: 'terminal-1', token: 'token-1', generation: '1' }])

    const reloaded = await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })
    expect(reloaded.historyGeneration).toBe('2')
    expect(source.requests.map((request) => request.mode)).toEqual(['latest', 'older', 'latest'])
  })

  it('serializes structured history styles and tail fill into replay ANSI', () => {
    const replay = coreV2HistoryRowsANSI([{
      index: 0,
      logicalLineId: '1',
      rowInLine: 0,
      cells: [{ text: 'ERR', width: 3, style: { fg: 'ansi:1', bold: true } }],
      tailFillStyle: { bg: '#102030' },
    }], 5)

    expect(replay).toContain('\u001b[1;38;5;1mERR')
    expect(replay).toContain('\u001b[48;2;16;32;48m  ')
  })

  it('preserves soft-wrap identity when replaying a reflowed logical line', async () => {
    const rows = coreV2ReflowHistoryRows([{
      index: 0,
      logicalLineId: '1',
      rowInLine: 0,
      cells: [{ text: 'abcdefghijkl', width: 12 }],
    }], 6)
    const terminal = new Terminal({ cols: 6, rows: 2, scrollback: 20 })

    await new Promise<void>((resolve) => terminal.write(coreV2HistoryRowsANSI(rows, 6), resolve))

    const first = terminal.buffer.active.getLine(0)!
    const second = terminal.buffer.active.getLine(1)!
    expect(second.isWrapped).toBe(true)
    expect(`${first.translateToString(true)}${second.isWrapped ? '' : '\n'}${second.translateToString(true)}`).toBe('abcdefghijkl')
    terminal.dispose()
  })

  it('tracks the exact logical column after an early wide-character wrap', () => {
    const rows = coreV2ReflowHistoryRows([{
      index: 0,
      logicalLineId: '1',
      rowInLine: 0,
      cells: [
        { text: 'a', width: 1 },
        { text: 'b', width: 1 },
        { text: 'c', width: 1 },
        { text: 'd', width: 1 },
        { text: '中', width: 2 },
        { text: 'e', width: 1 },
      ],
    }], 5)

    expect(rows.map((row) => row.cells.map((cell) => cell.text).join(''))).toEqual(['abcd', '中e'])
    expect(rows.map((row) => row.logicalStartCol)).toEqual([0, 4])
  })

  it('crops a fixed grid row to one local xterm row', async () => {
    const rows = coreV2ReflowHistoryRows([{
      index: 0,
      fixedGrid: true,
      cells: [{ text: 'abcdefghijkl', width: 12 }],
    }], 6)
    const terminal = new Terminal({ cols: 6, rows: 2, scrollback: 20 })

    expect(rows).toHaveLength(1)
    expect(rows[0]?.cells.map((cell) => cell.text).join('')).toBe('abcdef')
    await new Promise<void>((resolve) => terminal.write(coreV2HistoryRowsANSI(rows, 6), resolve))
    expect(terminal.buffer.active.length).toBe(2)
    expect(terminal.buffer.active.getLine(1)?.isWrapped).toBe(false)
    terminal.dispose()
  })

  it('maps the frozen live viewport anchor and shifts it across older prepends', async () => {
    const latestWindow = window({ lineId: '42', token: 'token-1', generation: '7', hasMore: true })
    latestWindow.renderRows = [
      historyRow('41', 0, 'older', 5),
      historyRow('42', 0, 'abcdefghij', 10),
    ]
    latestWindow.viewportAnchor = {
      topLineId: '42',
      topCellOffset: 5,
      atEnd: false,
      screenCols: 5,
      screenRows: 3,
    }
    const olderWindow = window({ lineId: '40', token: 'token-1', generation: '7', op: 'prepend', hasMore: false })
    const pager = new CoreV2ScrollbackPager(new MockSource([latestWindow, olderWindow]))

    const latest = await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 3, cols: 5 })
    const older = await pager.load({ terminalId: 'terminal-1', offset: latest.totalLoadedRows, limit: 1, cols: 5 })

    expect(latest.viewportTop).toBe(2)
    expect(older.viewportTop).toBe(4)
  })

  it('counts production logical rows after local reflow', async () => {
    const latestWindow = window({ lineId: '42', token: 'token-1', generation: '7', hasMore: false })
    latestWindow.renderRows = [historyRow('42', 0, 'abcdefghijklmnopqr', 18)]
    latestWindow.viewportAnchor = {
      topLineId: '42',
      topCellOffset: 6,
      atEnd: false,
      screenCols: 6,
      screenRows: 2,
    }
    const pager = new CoreV2ScrollbackPager(new MockSource([latestWindow]))

    const latest = await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 100, cols: 6 })

    expect(latest.rows.map((row) => row.cells.map((cell) => cell.text).join(''))).toEqual([
      'abcdef',
      'ghijkl',
      'mnopqr',
    ])
    expect(latest.loadedRows).toBe(3)
    expect(latest.viewportTop).toBe(1)
  })

  it('searches the current frozen generation and replaces the paging anchor with the match window', async () => {
    const matchWindow = window({ lineId: '20', token: 'token-1', generation: '7', hasMore: true })
    matchWindow.renderRows = [historyRow('20', 0, 'abcdefghijkl', 12)]
    const source = new MockSource(
      [window({ lineId: '42', token: 'token-1', generation: '7', hasMore: true })],
      { found: true, wrapped: false, match: { startLineId: '20', startCol: 7, endLineId: '20', endCol: 10 }, window: matchWindow },
    )
    const pager = new CoreV2ScrollbackPager(source)
    await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 5 })

    const result = await pager.search({ terminalId: 'terminal-1', query: '*hij*', mode: 'glob', direction: 'backward', cols: 5, limit: 100 })

    expect(result).toMatchObject({
      found: true,
      wrapped: false,
      matchRow: 1,
      matchRanges: [
        { row: 1, startCol: 2, endCol: 5 },
      ],
    })
    expect(result.page?.rows[0]?.logicalLineId).toBe('20')
    expect(source.searchRequests[0]).toMatchObject({ token: 'token-1', generation: '7', query: '*hij*', mode: 'glob', direction: 'backward' })
  })

  it('releases an expired frozen token after search fails', async () => {
    const stale = Object.assign(new Error('history expired'), { code: 'stale_resource' })
    const source = new MockSource(
      [window({ lineId: '42', token: 'token-1', generation: '7', hasMore: true })],
      stale,
    )
    const pager = new CoreV2ScrollbackPager(source)
    await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })

    await expect(pager.search({ terminalId: 'terminal-1', query: 'needle', direction: 'forward', cols: 80, limit: 100 })).rejects.toBe(stale)

    expect(source.releases).toEqual([{ terminalId: 'terminal-1', token: 'token-1', generation: '7' }])
    await expect(pager.copy('terminal-1', 80, { startLineId: '42', startCol: 0, endLineId: '42', endCol: 1 })).rejects.toThrow('loaded frozen window')
  })

  it('streams scan batches from the frozen generation without replacing the paging state', async () => {
    const source = new MockSource(
      [window({ lineId: '42', token: 'token-1', generation: '7', hasMore: true })],
      undefined,
      'selected',
      [
        { matches: [{ startLineId: '4', startCol: 1, endLineId: '4', endCol: 7 }], next: { lineId: '4', col: 7 }, done: false },
        { matches: [{ startLineId: '9', startCol: 2, endLineId: '9', endCol: 8 }], done: true },
      ],
    )
    const pager = new CoreV2ScrollbackPager(source)
    await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })
    const batches: CoreV2HistorySearchScanResult[] = []

    await pager.scan({ terminalId: 'terminal-1', query: 'needle', mode: 'text', cols: 80 }, (batch) => batches.push(batch))

    expect(batches).toEqual([
      { matches: [{ startLineId: '4', startCol: 1, endLineId: '4', endCol: 7 }], done: false },
      { matches: [{ startLineId: '9', startCol: 2, endLineId: '9', endCol: 8 }], done: true },
    ])
    expect(source.scanRequests).toEqual([
      expect.objectContaining({ token: 'token-1', generation: '7', maxMatches: 64 }),
      expect.objectContaining({ token: 'token-1', generation: '7', start: { lineId: '4', col: 7 } }),
    ])
    await expect(pager.copy('terminal-1', 80, { startLineId: '42', startCol: 0, endLineId: '42', endCol: 1 })).resolves.toBe('selected')
  })

  it('copies a logical range from the current frozen generation', async () => {
    const source = new MockSource([window({ lineId: '42', token: 'token-1', generation: '7', hasMore: true })], undefined, 'selected')
    const pager = new CoreV2ScrollbackPager(source)
    await pager.load({ terminalId: 'terminal-1', offset: 0, limit: 1, cols: 80 })

    await expect(pager.copy('terminal-1', 80, { startLineId: '42', startCol: 1, endLineId: '42', endCol: 4 })).resolves.toBe('selected')
    expect(source.copyRequests[0]).toMatchObject({
      token: 'token-1',
      generation: '7',
      boundaryFirstLineId: '42',
      boundaryLastLineId: '42',
    })
  })
})

class MockSource implements CoreV2HistorySource {
  readonly requests: CoreV2HistoryWindowRequest[] = []
  readonly copyRequests: CoreV2HistoryCopyRequest[] = []
  readonly searchRequests: CoreV2HistorySearchRequest[] = []
  readonly scanRequests: CoreV2HistorySearchScanRequest[] = []
  readonly releases: Array<{ terminalId: string; token: string; generation?: string | number | bigint }> = []

  constructor(
    private readonly windows: Array<CoreV2HistoryWindow | Error>,
    private readonly searchResult?: CoreV2HistorySearchResult | Error,
    private readonly copyText = '',
    private readonly scanResults: CoreV2HistorySearchScanResult[] = [],
  ) {}

  async window(request: CoreV2HistoryWindowRequest): Promise<CoreV2HistoryWindow> {
    this.requests.push({ ...request })
    const result = this.windows.shift()
    if (!result) throw new Error('unexpected history window request')
    if (result instanceof Error) throw result
    return result
  }

  async copy(request: CoreV2HistoryCopyRequest): Promise<string> {
    this.copyRequests.push(request)
    return this.copyText
  }

  async search(request: CoreV2HistorySearchRequest): Promise<CoreV2HistorySearchResult> {
    this.searchRequests.push(request)
    if (!this.searchResult) throw new Error('not used')
    if (this.searchResult instanceof Error) throw this.searchResult
    return this.searchResult
  }

  async scan(request: CoreV2HistorySearchScanRequest): Promise<CoreV2HistorySearchScanResult> {
    this.scanRequests.push(request)
    const result = this.scanResults.shift()
    if (!result) throw new Error('unexpected history scan request')
    return result
  }

  async release(request: { terminalId: string; token: string; generation?: string | number | bigint }): Promise<void> {
    this.releases.push({ ...request })
  }
}

function historyRow(lineId: string, rowInLine: number, text: string, width: number) {
  return {
    index: rowInLine,
    cells: [{ text, width }],
    logicalLineId: lineId,
    rowInLine,
  }
}

function window(input: {
  lineId: string
  token: string
  generation: string
  op?: 'replace' | 'prepend'
  hasMore: boolean
}): CoreV2HistoryWindow {
  return {
    terminalId: 'terminal-1',
    token: input.token,
    op: input.op ?? 'replace',
    cols: 80,
    renderRows: [{
      index: 0,
      cells: [{ text: `line-${input.lineId}`, width: 7 }],
      logicalLineId: input.lineId,
      rowInLine: 0,
    }],
    lines: [],
    totalRows: 42,
    logicalTotal: 42,
    hasMore: input.hasMore,
    generation: input.generation,
    firstLineId: input.lineId,
    lastLineId: input.lineId,
  }
}
