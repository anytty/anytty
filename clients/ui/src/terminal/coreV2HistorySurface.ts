import type { CoreV2HistorySource } from './coreV2HistorySource'
import type {
  CoreV2HistoryCursor,
  CoreV2HistoryLineSpan,
  CoreV2HistoryRow,
  CoreV2HistoryWindow,
  CoreV2HistoryWindowRequest,
} from './coreV2TerminalProtocol'

const DEFAULT_REQUEST_ROWS = 160
const DEFAULT_RENDER_OVERSCAN_ROWS = 80
const DEFAULT_EDGE_PREFETCH_ROWS = 32
const DEFAULT_CACHE_RETAIN_ROWS = 2400

export interface CoreV2HistorySurfaceOptions {
  terminalId: string
  cols: number
  viewportRows: number
  requestRows?: number | undefined
  renderOverscanRows?: number | undefined
  edgePrefetchRows?: number | undefined
  cacheRetainRows?: number | undefined
}

export interface CoreV2HistorySurfaceLoadOptions {
  limit?: number | undefined
  cols?: number | undefined
  viewportRows?: number | undefined
}

export interface CoreV2HistoryRenderWindow {
  startIndex: number
  endIndex: number
  viewportStartIndex: number
  viewportEndIndex: number
  viewportOffsetRows: number
  overscanRows: number
  edgePrefetchRows: number
  shouldPrefetchOlder: boolean
  shouldPrefetchNewer: boolean
}

export interface CoreV2HistorySurfaceSnapshot {
  terminalId: string
  cols: number
  viewportRows: number
  revision: number
  token: string | null
  generation: string | null
  stale: boolean
  staleReason: string | null
  rows: CoreV2HistoryRow[]
  renderRows: CoreV2HistoryRow[]
  lines: CoreV2HistoryLineSpan[]
  loadedRows: number
  totalRows: number
  logicalTotal: number
  hasOlder: boolean
  hasNewer: boolean
  renderWindow: CoreV2HistoryRenderWindow
  firstCursor?: CoreV2HistoryCursor | undefined
  lastCursor?: CoreV2HistoryCursor | undefined
  firstLineId?: string | undefined
  lastLineId?: string | undefined
}

export interface CoreV2HistorySurface {
  loadLatest(options?: CoreV2HistorySurfaceLoadOptions): Promise<CoreV2HistorySurfaceSnapshot>
  loadOlder(limit?: number): Promise<CoreV2HistorySurfaceSnapshot>
  loadNewer(limit?: number): Promise<CoreV2HistorySurfaceSnapshot>
  scrollByRows(deltaRows: number): CoreV2HistorySurfaceSnapshot
  scrollToCachedRow(rowIndex: number): CoreV2HistorySurfaceSnapshot
  setViewportRows(rows: number): CoreV2HistorySurfaceSnapshot
  resetForCols(cols: number): CoreV2HistorySurfaceSnapshot
  invalidate(reason?: string): CoreV2HistorySurfaceSnapshot
  snapshot(): CoreV2HistorySurfaceSnapshot
}

export class CoreV2HistorySurfaceStaleError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'CoreV2HistorySurfaceStaleError'
  }
}

interface CachedLineSpan {
  line: CoreV2HistoryLineSpan
  startKey: string
  endKey: string
}

class CoreV2HistorySurfaceStore implements CoreV2HistorySurface {
  private readonly source: CoreV2HistorySource
  private readonly terminalId: string
  private readonly requestRows: number
  private readonly renderOverscanRows: number
  private readonly edgePrefetchRows: number
  private readonly cacheRetainRows: number
  private readonly pending = new Map<string, Promise<CoreV2HistorySurfaceSnapshot>>()
  private rows: CoreV2HistoryRow[] = []
  private lines = new Map<string, CachedLineSpan>()
  private cols: number
  private viewportRows: number
  private viewportStartIndex = 0
  private revision = 0
  private token: string | null = null
  private generation: string | null = null
  private totalRows = 0
  private logicalTotal = 0
  private hasOlder = false
  private hasNewer = false
  private stale = false
  private staleReason: string | null = null

  constructor(source: CoreV2HistorySource, options: CoreV2HistorySurfaceOptions) {
    if (!options.terminalId.trim()) throw new Error('history surface requires terminalId')
    this.source = source
    this.terminalId = options.terminalId
    this.cols = positiveInt(options.cols, 'history surface requires positive cols')
    this.viewportRows = positiveInt(options.viewportRows, 'history surface requires positive viewportRows')
    this.requestRows = positiveInt(options.requestRows ?? DEFAULT_REQUEST_ROWS, 'history surface requires positive requestRows')
    this.renderOverscanRows = nonNegativeInt(options.renderOverscanRows ?? DEFAULT_RENDER_OVERSCAN_ROWS)
    this.edgePrefetchRows = nonNegativeInt(options.edgePrefetchRows ?? DEFAULT_EDGE_PREFETCH_ROWS)
    this.cacheRetainRows = nonNegativeInt(options.cacheRetainRows ?? DEFAULT_CACHE_RETAIN_ROWS)
  }

  loadLatest(options: CoreV2HistorySurfaceLoadOptions = {}): Promise<CoreV2HistorySurfaceSnapshot> {
    if (options.viewportRows !== undefined) {
      this.viewportRows = positiveInt(options.viewportRows, 'history surface requires positive viewportRows')
    }
    if (options.cols !== undefined && positiveInt(options.cols, 'history surface requires positive cols') !== this.cols) {
      this.cols = positiveInt(options.cols, 'history surface requires positive cols')
      this.clearCache('history surface cols changed')
    }

    const limit = positiveInt(options.limit ?? this.requestRows, 'history surface latest requires positive limit')
    const key = `latest:${this.terminalId}:${this.cols}:${limit}`
    return this.withPending(key, async () => {
      const window = await this.source.window({
        terminalId: this.terminalId,
        mode: 'latest',
        limit,
        cols: this.cols,
      })
      this.applyReplace(window)
      return this.snapshot()
    })
  }

  loadOlder(limit = this.requestRows): Promise<CoreV2HistorySurfaceSnapshot> {
    const token = this.requireToken('older')
    const beforeCursor = this.requireFirstCursor('older')
    const request = this.frozenRequest('older', positiveInt(limit, 'older history window requires positive limit'))
    request.beforeCursor = beforeCursor

    const firstLineId = this.firstLineId()
    const lastLineId = this.lastLineId()
    if (firstLineId) request.boundaryFirstLineId = firstLineId
    if (lastLineId) request.boundaryLastLineId = lastLineId

    const key = `older:${token}:${this.generation ?? ''}:${beforeCursor.lineId}:${beforeCursor.rowInLine}:${request.limit}`
    return this.withPending(key, async () => {
      const previousViewportStart = this.viewportStartIndex
      const previousKeys = new Set(this.rows.map(rowKey))
      const window = await this.source.window(request)
      this.assertCompatibleWindow(window, 'older')
      const inserted = this.mergeWindow(window, 'prepend', previousKeys)
      this.viewportStartIndex = previousViewportStart + inserted
      this.hasOlder = window.hasMore
      this.finishWindowUpdate(window)
      return this.snapshot()
    })
  }

  loadNewer(limit = this.requestRows): Promise<CoreV2HistorySurfaceSnapshot> {
    const token = this.requireToken('newer')
    const afterCursor = this.requireLastCursor('newer')
    const request = this.frozenRequest('newer', positiveInt(limit, 'newer history window requires positive limit'))
    request.afterCursor = afterCursor

    const firstLineId = this.firstLineId()
    const lastLineId = this.lastLineId()
    if (firstLineId) request.boundaryFirstLineId = firstLineId
    if (lastLineId) request.boundaryLastLineId = lastLineId

    const wasPinnedToBottom = this.viewportStartIndex >= this.maxViewportStart()
    const key = `newer:${token}:${this.generation ?? ''}:${afterCursor.lineId}:${afterCursor.rowInLine}:${request.limit}`
    return this.withPending(key, async () => {
      const previousViewportStart = this.viewportStartIndex
      const previousKeys = new Set(this.rows.map(rowKey))
      const window = await this.source.window(request)
      this.assertCompatibleWindow(window, 'newer')
      this.mergeWindow(window, 'append', previousKeys)
      this.viewportStartIndex = wasPinnedToBottom ? this.maxViewportStart() : previousViewportStart
      this.hasNewer = window.hasMore
      this.finishWindowUpdate(window)
      return this.snapshot()
    })
  }

  scrollByRows(deltaRows: number): CoreV2HistorySurfaceSnapshot {
    this.viewportStartIndex = this.clampViewportStart(this.viewportStartIndex + Math.trunc(deltaRows))
    this.trimCacheAroundViewport()
    return this.snapshot()
  }

  scrollToCachedRow(rowIndex: number): CoreV2HistorySurfaceSnapshot {
    this.viewportStartIndex = this.clampViewportStart(Math.trunc(rowIndex))
    this.trimCacheAroundViewport()
    return this.snapshot()
  }

  setViewportRows(rows: number): CoreV2HistorySurfaceSnapshot {
    this.viewportRows = positiveInt(rows, 'history surface requires positive viewportRows')
    this.viewportStartIndex = this.clampViewportStart(this.viewportStartIndex)
    this.trimCacheAroundViewport()
    return this.snapshot()
  }

  resetForCols(cols: number): CoreV2HistorySurfaceSnapshot {
    const nextCols = positiveInt(cols, 'history surface requires positive cols')
    if (nextCols !== this.cols) {
      this.cols = nextCols
      this.invalidate('history surface cols changed')
    }
    return this.snapshot()
  }

  invalidate(reason = 'history surface invalidated'): CoreV2HistorySurfaceSnapshot {
    this.clearCache(reason)
    this.stale = true
    this.staleReason = reason
    this.revision += 1
    return this.snapshot()
  }

  snapshot(): CoreV2HistorySurfaceSnapshot {
    this.viewportStartIndex = this.clampViewportStart(this.viewportStartIndex)
    const rows = this.rows.map((row, index) => cloneRow(row, index))
    const renderWindow = this.renderWindow(rows.length)
    const snapshot: CoreV2HistorySurfaceSnapshot = {
      terminalId: this.terminalId,
      cols: this.cols,
      viewportRows: this.viewportRows,
      revision: this.revision,
      token: this.token,
      generation: this.generation,
      stale: this.stale,
      staleReason: this.staleReason,
      rows,
      renderRows: rows.slice(renderWindow.startIndex, renderWindow.endIndex),
      lines: this.snapshotLines(rows),
      loadedRows: rows.length,
      totalRows: this.totalRows,
      logicalTotal: this.logicalTotal,
      hasOlder: this.hasOlder,
      hasNewer: this.hasNewer,
      renderWindow,
    }
    const firstCursor = cursorFromRow(rows[0])
    const lastCursor = cursorFromRow(rows.at(-1))
    const firstLineId = this.firstLineId()
    const lastLineId = this.lastLineId()
    if (firstCursor) snapshot.firstCursor = firstCursor
    if (lastCursor) snapshot.lastCursor = lastCursor
    if (firstLineId) snapshot.firstLineId = firstLineId
    if (lastLineId) snapshot.lastLineId = lastLineId
    return snapshot
  }

  private frozenRequest(mode: 'older' | 'newer', limit: number): CoreV2HistoryWindowRequest {
    return {
      terminalId: this.terminalId,
      mode,
      limit,
      cols: this.cols,
      token: this.requireToken(mode),
      generation: this.generation ?? undefined,
    }
  }

  private requireToken(mode: string): string {
    if (!this.token) throw new Error(`${mode} history window requires a loaded core-v2 history token`)
    return this.token
  }

  private requireFirstCursor(mode: string): CoreV2HistoryCursor {
    const cursor = cursorFromRow(this.rows[0])
    if (!cursor) throw new Error(`${mode} history window requires a logical first cursor`)
    return cursor
  }

  private requireLastCursor(mode: string): CoreV2HistoryCursor {
    const cursor = cursorFromRow(this.rows.at(-1))
    if (!cursor) throw new Error(`${mode} history window requires a logical last cursor`)
    return cursor
  }

  private applyReplace(window: CoreV2HistoryWindow): void {
    this.assertWindowShape(window)
    if (!window.token.trim()) throw new Error('history surface requires a frozen core-v2 history token')
    if (!window.generation.trim()) throw new Error('history surface requires core-v2 history generation')

    // latest window 是 App history surface 的冻结边界，后续分页必须沿用同一 token/generation。
    this.rows = normalizeWindowRows(window)
    this.lines.clear()
    this.registerWindowLines(window)
    this.token = window.token
    this.generation = window.generation
    this.totalRows = window.totalRows
    this.logicalTotal = window.logicalTotal
    this.hasOlder = window.hasMore
    this.hasNewer = false
    this.stale = false
    this.staleReason = null
    this.viewportStartIndex = this.maxViewportStart()
    this.finishWindowUpdate(window)
  }

  private assertCompatibleWindow(window: CoreV2HistoryWindow, mode: string): void {
    this.assertWindowShape(window)
    if (window.token !== this.token || window.generation !== this.generation) {
      // token/generation 改变说明后端历史窗口已换代，App cache 不能继续拼接旧窗口。
      const reason = `${mode} history window changed token/generation; App cache must be reloaded`
      this.clearCache(reason)
      this.stale = true
      this.staleReason = reason
      this.revision += 1
      throw new CoreV2HistorySurfaceStaleError(reason)
    }
  }

  private assertWindowShape(window: CoreV2HistoryWindow): void {
    if (window.terminalId !== this.terminalId) {
      throw new Error(`history surface terminal mismatch: ${window.terminalId} !== ${this.terminalId}`)
    }
    if (window.cols !== this.cols) {
      throw new Error(`history surface cols mismatch: ${window.cols} !== ${this.cols}`)
    }
    for (const row of window.renderRows) {
      if (!cursorFromRow(row)) {
        throw new Error('history surface rows must carry logical line cursor metadata')
      }
    }
  }

  private mergeWindow(
    window: CoreV2HistoryWindow,
    direction: 'prepend' | 'append',
    previousKeys: Set<string>,
  ): number {
    const incoming = normalizeWindowRows(window)
    const incomingKeys = new Set(incoming.map(rowKey))
    const remaining = this.rows.filter((row) => !incomingKeys.has(rowKey(row)))
    this.rows = direction === 'prepend' ? [...incoming, ...remaining] : [...remaining, ...incoming]
    this.registerWindowLines(window)
    return incoming.reduce((count, row) => count + (previousKeys.has(rowKey(row)) ? 0 : 1), 0)
  }

  private finishWindowUpdate(window: CoreV2HistoryWindow): void {
    this.totalRows = Math.max(this.totalRows, window.totalRows)
    this.logicalTotal = Math.max(this.logicalTotal, window.logicalTotal)
    this.viewportStartIndex = this.clampViewportStart(this.viewportStartIndex)
    this.trimCacheAroundViewport()
    this.revision += 1
  }

  private registerWindowLines(window: CoreV2HistoryWindow): void {
    for (const line of window.lines) {
      const startRow = window.renderRows[line.startRow]
      const endRow = window.renderRows[line.endRow]
      if (!startRow || !endRow) continue
      this.lines.set(line.logicalLineId, {
        line,
        startKey: rowKey(startRow),
        endKey: rowKey(endRow),
      })
    }
    this.registerMissingLineSpans(window)
  }

  private registerMissingLineSpans(window: CoreV2HistoryWindow): void {
    const spans = new Map<string, { start: number; end: number }>()
    window.renderRows.forEach((row, index) => {
      const cursor = cursorFromRow(row)
      if (!cursor) return
      const existing = spans.get(cursor.lineId)
      if (existing) {
        existing.end = index
      } else {
        spans.set(cursor.lineId, { start: index, end: index })
      }
    })
    for (const [lineId, span] of spans) {
      if (this.lines.has(lineId)) continue
      const startRow = window.renderRows[span.start]
      const endRow = window.renderRows[span.end]
      if (!startRow || !endRow) continue
      this.lines.set(lineId, {
        line: {
          startRow: span.start,
          endRow: span.end,
          logicalLineId: lineId,
          clippedBefore: false,
          clippedAfter: false,
        },
        startKey: rowKey(startRow),
        endKey: rowKey(endRow),
      })
    }
  }

  private snapshotLines(rows: CoreV2HistoryRow[]): CoreV2HistoryLineSpan[] {
    const keyToIndex = new Map(rows.map((row, index) => [rowKey(row), index]))
    const lines: CoreV2HistoryLineSpan[] = []
    for (const cached of this.lines.values()) {
      const startRow = keyToIndex.get(cached.startKey)
      const endRow = keyToIndex.get(cached.endKey)
      if (startRow === undefined || endRow === undefined) continue
      lines.push({
        ...cached.line,
        startRow,
        endRow,
      })
    }
    return lines.sort((a, b) => a.startRow - b.startRow)
  }

  private trimCacheAroundViewport(): void {
    if (this.cacheRetainRows <= 0 || this.rows.length === 0) return
    const viewportEnd = Math.min(this.rows.length, this.viewportStartIndex + this.viewportRows)
    const keepStart = Math.max(0, this.viewportStartIndex - this.cacheRetainRows)
    const keepEnd = Math.min(this.rows.length, viewportEnd + this.cacheRetainRows)
    if (keepStart === 0 && keepEnd === this.rows.length) return
    const droppedOlder = keepStart > 0
    const droppedNewer = keepEnd < this.rows.length
    this.rows = this.rows.slice(keepStart, keepEnd)
    this.viewportStartIndex = this.clampViewportStart(this.viewportStartIndex - keepStart)
    // trim 只丢本地缓存，不改变后端历史存在性；被裁掉的方向仍要允许按 cursor 重新加载。
    if (droppedOlder) this.hasOlder = true
    if (droppedNewer) this.hasNewer = true
    this.dropOrphanLines()
  }

  private dropOrphanLines(): void {
    const keys = new Set(this.rows.map(rowKey))
    for (const [lineId, line] of this.lines) {
      if (!keys.has(line.startKey) || !keys.has(line.endKey)) {
        this.lines.delete(lineId)
      }
    }
  }

  private renderWindow(rowCount: number): CoreV2HistoryRenderWindow {
    const viewportStartIndex = this.clampViewportStart(this.viewportStartIndex)
    const viewportEndIndex = Math.min(rowCount, viewportStartIndex + this.viewportRows)
    const startIndex = Math.max(0, viewportStartIndex - this.renderOverscanRows)
    const endIndex = Math.min(rowCount, viewportEndIndex + this.renderOverscanRows)
    return {
      startIndex,
      endIndex,
      viewportStartIndex,
      viewportEndIndex,
      viewportOffsetRows: viewportStartIndex - startIndex,
      overscanRows: this.renderOverscanRows,
      edgePrefetchRows: this.edgePrefetchRows,
      shouldPrefetchOlder: this.hasOlder && viewportStartIndex - startIndex <= this.edgePrefetchRows,
      shouldPrefetchNewer: this.hasNewer && endIndex - viewportEndIndex <= this.edgePrefetchRows,
    }
  }

  private firstLineId(): string | undefined {
    return cursorFromRow(this.rows[0])?.lineId
  }

  private lastLineId(): string | undefined {
    return cursorFromRow(this.rows.at(-1))?.lineId
  }

  private maxViewportStart(): number {
    return Math.max(0, this.rows.length - this.viewportRows)
  }

  private clampViewportStart(value: number): number {
    return clamp(Math.trunc(value), 0, this.maxViewportStart())
  }

  private clearCache(reason: string): void {
    this.rows = []
    this.lines.clear()
    this.viewportStartIndex = 0
    this.token = null
    this.generation = null
    this.totalRows = 0
    this.logicalTotal = 0
    this.hasOlder = false
    this.hasNewer = false
    this.staleReason = reason
  }

  private withPending(
    key: string,
    load: () => Promise<CoreV2HistorySurfaceSnapshot>,
  ): Promise<CoreV2HistorySurfaceSnapshot> {
    const existing = this.pending.get(key)
    if (existing) return existing
    const pending = load().finally(() => {
      this.pending.delete(key)
    })
    this.pending.set(key, pending)
    return pending
  }
}

export function createCoreV2HistorySurface(
  source: CoreV2HistorySource,
  options: CoreV2HistorySurfaceOptions,
): CoreV2HistorySurface {
  return new CoreV2HistorySurfaceStore(source, options)
}

function normalizeWindowRows(window: CoreV2HistoryWindow): CoreV2HistoryRow[] {
  return window.renderRows.map((row, index) => cloneRow(row, index))
}

function cloneRow(row: CoreV2HistoryRow, index: number): CoreV2HistoryRow {
  return {
    ...row,
    index,
    tailFillStyle: row.tailFillStyle ? { ...row.tailFillStyle } : undefined,
    cells: row.cells.map((cell) => ({
      ...cell,
      style: cell.style ? { ...cell.style } : undefined,
    })),
  }
}

function cursorFromRow(row: CoreV2HistoryRow | undefined): CoreV2HistoryCursor | undefined {
  if (!row?.logicalLineId || row.rowInLine === undefined) return undefined
  return {
    lineId: row.logicalLineId,
    rowInLine: row.rowInLine,
  }
}

function rowKey(row: CoreV2HistoryRow): string {
  const cursor = cursorFromRow(row)
  if (!cursor) throw new Error('history surface row is missing logical cursor metadata')
  return `${cursor.lineId}:${cursor.rowInLine}`
}

function positiveInt(value: number, message: string): number {
  if (!Number.isFinite(value) || value <= 0) throw new Error(message)
  return Math.max(1, Math.trunc(value))
}

function nonNegativeInt(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 0
  return Math.trunc(value)
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value))
}
