import type { CoreV2HistorySource } from './coreV2HistorySource'
import { coreV2ReflowHistoryRows } from './coreV2HistoryANSI'
import type {
  CoreV2HistoryCursor,
  CoreV2HistoryRow,
  CoreV2HistorySearchResult,
  CoreV2HistoryWindow,
} from './coreV2TerminalProtocol'

export interface CoreV2ScrollbackPage {
  rows: CoreV2HistoryRow[]
  operation: 'replace' | 'prepend'
  loadedRows: number
  totalLoadedRows: number
  committedTotalRows: number
  logicalTotalRows: number
  historyGeneration: string
  firstRowId?: string | undefined
  lastRowId?: string | undefined
  viewportTop?: number | undefined
  hasMore: boolean
}

export interface CoreV2ScrollbackSearchPage {
  found: boolean
  wrapped: boolean
  page?: CoreV2ScrollbackPage | undefined
  match?: { startLineId: string; startCol: number; endLineId: string; endCol: number } | undefined
  matchRow?: number | undefined
}

interface CoreV2ScrollbackState {
  cols: number
  token: string
  generation: string
  firstCursor?: CoreV2HistoryCursor | undefined
  firstLineId?: string | undefined
  lastLineId?: string | undefined
  totalLoadedRows: number
  viewportTop?: number | undefined
  hasMore: boolean
}

/** Bridges the legacy xterm scrollback consumer onto frozen cursor-based history windows. */
export class CoreV2ScrollbackPager {
  private readonly stateByTerminal = new Map<string, CoreV2ScrollbackState>()

  constructor(private readonly source: CoreV2HistorySource) {}

  async load(input: {
    terminalId: string
    offset: number
    limit: number
    cols: number
    signal?: AbortSignal
  }): Promise<CoreV2ScrollbackPage> {
    const current = this.stateByTerminal.get(input.terminalId)
    const loadLatest = input.offset <= 0 || !current || current.cols !== input.cols
    if (!loadLatest && !current.hasMore) {
      return emptyPage(current)
    }
    const sourceOptions = input.signal ? { signal: input.signal } : undefined

    let window: CoreV2HistoryWindow
    try {
      window = loadLatest
        ? await this.source.window({
            terminalId: input.terminalId,
            mode: 'latest',
            limit: input.limit,
            cols: input.cols,
          }, sourceOptions)
        : await this.source.window({
            terminalId: input.terminalId,
            mode: 'older',
            limit: input.limit,
            cols: input.cols,
            token: current.token,
            generation: current.generation,
            beforeCursor: requireFirstCursor(current),
            boundaryFirstLineId: current.firstLineId,
            boundaryLastLineId: current.lastLineId,
          }, sourceOptions)
    } catch (error) {
      if (current && isTerminalHistoryControlError(error)) {
        this.stateByTerminal.delete(input.terminalId)
        this.release(input.terminalId, current)
      }
      throw error
    }

    if (!window.token.trim() || !window.generation.trim()) {
      throw new Error('history scrollback requires a frozen token and generation')
    }
    if (!loadLatest && (window.token !== current.token || window.generation !== current.generation)) {
      this.stateByTerminal.delete(input.terminalId)
      throw new Error('history scrollback window changed token or generation')
    }

    const visualRows = coreV2ReflowHistoryRows(window.renderRows, input.cols)
    const next = stateFromWindow(window, visualRows, input.cols, loadLatest ? undefined : current)
    this.stateByTerminal.set(input.terminalId, next)
    if (loadLatest && current && current.token !== next.token) {
      this.release(input.terminalId, current)
    }
    return pageFromWindow(window, visualRows, next)
  }

  async search(input: {
    terminalId: string
    query: string
    direction: 'forward' | 'backward'
    cols: number
    limit: number
    start?: { lineId: string; col: number } | undefined
    signal?: AbortSignal | undefined
  }): Promise<CoreV2ScrollbackSearchPage> {
    const current = this.stateByTerminal.get(input.terminalId)
    if (!current || current.cols !== input.cols) throw new Error('history search requires a loaded frozen window')
    let result: CoreV2HistorySearchResult
    try {
      result = await this.source.search({
        terminalId: input.terminalId,
        token: current.token,
        generation: current.generation,
        query: input.query,
        direction: input.direction,
        cols: input.cols,
        limit: input.limit,
        ...(input.start ? { start: input.start } : {}),
      }, input.signal ? { signal: input.signal } : undefined)
    } catch (error) {
      if (isTerminalHistoryControlError(error)) {
        this.stateByTerminal.delete(input.terminalId)
        this.release(input.terminalId, current)
      }
      throw error
    }
    if (!result.found) return { found: false, wrapped: false }
    if (result.window.token !== current.token || result.window.generation !== current.generation) {
      this.stateByTerminal.delete(input.terminalId)
      this.release(input.terminalId, current)
      throw new Error('history search window changed token or generation')
    }
    const visualRows = coreV2ReflowHistoryRows(result.window.renderRows, input.cols)
    const next = stateFromWindow(result.window, visualRows, input.cols, undefined)
    this.stateByTerminal.set(input.terminalId, next)
    const matchRow = historyMatchVisualRow(visualRows, result.match.startLineId, result.match.startCol)
    return {
      found: true,
      wrapped: result.wrapped,
      page: pageFromWindow(result.window, visualRows, next),
      match: result.match,
      matchRow: matchRow < 0 ? 0 : matchRow,
    }
  }

  async copy(
    terminalId: string,
    cols: number,
    range: { startLineId: string; startCol: number; endLineId: string; endCol: number },
    signal?: AbortSignal,
  ): Promise<string> {
    const current = this.stateByTerminal.get(terminalId)
    if (!current || current.cols !== cols) throw new Error('history copy requires a loaded frozen window')
    try {
      return await this.source.copy({
        terminalId,
        token: current.token,
        generation: current.generation,
        cols,
        boundaryFirstLineId: current.firstLineId,
        boundaryLastLineId: current.lastLineId,
        range,
      }, signal ? { signal } : undefined)
    } catch (error) {
      if (isTerminalHistoryControlError(error)) {
        this.stateByTerminal.delete(terminalId)
        this.release(terminalId, current)
      }
      throw error
    }
  }

  forget(terminalId: string): void {
    const state = this.stateByTerminal.get(terminalId)
    this.stateByTerminal.delete(terminalId)
    if (state) this.release(terminalId, state)
  }

  private release(terminalId: string, state: CoreV2ScrollbackState): void {
    void this.source.release?.({
      terminalId,
      token: state.token,
      generation: state.generation,
    }).catch(() => undefined)
  }
}

function historyMatchVisualRow(rows: CoreV2HistoryRow[], lineId: string, column: number): number {
  let remaining = Math.max(0, Math.trunc(column))
  let fallback = -1
  for (let index = 0; index < rows.length; index += 1) {
    const row = rows[index]!
    if (row.logicalLineId !== lineId) continue
    fallback = index
    const width = row.cells.reduce((total, cell) => total + Math.max(1, cell.width), 0)
    if (remaining < width || !row.wrapped) return index
    remaining -= width
  }
  return fallback < 0 ? 0 : fallback
}

function isTerminalHistoryControlError(error: unknown): boolean {
  if (!(error instanceof Error)) return false
  const code = (error as Error & { code?: string }).code
  return code === 'stale_resource' || code === 'resource_exhausted'
}

function stateFromWindow(window: CoreV2HistoryWindow, visualRows: CoreV2HistoryRow[], cols: number, previous: CoreV2ScrollbackState | undefined): CoreV2ScrollbackState {
  const loadedRows = visualRows.length
  return {
    cols,
    token: window.token,
    generation: window.generation,
    firstCursor: cursorFromRow(window.renderRows[0]),
    firstLineId: window.firstLineId ?? window.renderRows[0]?.logicalLineId,
    lastLineId: window.lastLineId ?? window.renderRows.at(-1)?.logicalLineId,
    totalLoadedRows: (previous?.totalLoadedRows ?? 0) + loadedRows,
    viewportTop: previous?.viewportTop === undefined
      ? historyViewportTop(window, visualRows)
      : previous.viewportTop + loadedRows,
    hasMore: window.hasMore,
  }
}

function pageFromWindow(window: CoreV2HistoryWindow, visualRows: CoreV2HistoryRow[], state: CoreV2ScrollbackState): CoreV2ScrollbackPage {
  return {
    rows: visualRows,
    operation: window.op === 'prepend' ? 'prepend' : 'replace',
    loadedRows: visualRows.length,
    totalLoadedRows: state.totalLoadedRows,
    committedTotalRows: window.totalRows,
    logicalTotalRows: window.logicalTotal,
    historyGeneration: window.generation,
    firstRowId: window.firstRowId,
    lastRowId: window.lastRowId,
    viewportTop: state.viewportTop,
    hasMore: window.hasMore,
  }
}

function emptyPage(state: CoreV2ScrollbackState): CoreV2ScrollbackPage {
  return {
    rows: [],
    operation: 'prepend',
    loadedRows: 0,
    totalLoadedRows: state.totalLoadedRows,
    committedTotalRows: state.totalLoadedRows,
    logicalTotalRows: state.totalLoadedRows,
    historyGeneration: state.generation,
    viewportTop: state.viewportTop,
    hasMore: false,
  }
}

function historyViewportTop(window: CoreV2HistoryWindow, visualRows: CoreV2HistoryRow[]): number | undefined {
  const anchor = window.viewportAnchor
  if (!anchor) return undefined
  if (anchor.atEnd) return visualRows.length

  let remaining = Math.max(0, Math.trunc(anchor.topCellOffset))
  let found = false
  for (let index = 0; index < visualRows.length; index += 1) {
    const row = visualRows[index]!
    if (row.logicalLineId !== anchor.topLineId) {
      if (found) break
      continue
    }
    found = true
    if (remaining === 0) return index
    const width = row.cells.reduce((total, cell) => total + Math.max(1, cell.width), 0)
    if (remaining < width) return index
    remaining -= width
  }
  return undefined
}

function requireFirstCursor(state: CoreV2ScrollbackState): CoreV2HistoryCursor {
  if (!state.firstCursor) throw new Error('older history scrollback requires a logical first cursor')
  return state.firstCursor
}

function cursorFromRow(row: CoreV2HistoryRow | undefined): CoreV2HistoryCursor | undefined {
  if (!row?.logicalLineId || row.rowInLine === undefined) return undefined
  return { lineId: row.logicalLineId, rowInLine: row.rowInLine }
}
