import {
  HistoryWindowOperation,
  RowOwnership,
  type CellStyle,
  type HistoryLineSpan,
  type HistoryRange,
  type HistoryRow,
  type HistorySearchResult,
  type HistoryWindowResult,
  type ScreenCell,
} from '../generated/apipb/history_pb'

export type CoreV2HistoryWindowOp = 'replace' | 'prepend' | 'append'
export type CoreV2HistoryWindowMode = 'latest' | 'older' | 'newer' | 'oldest' | 'range'

export interface CoreV2HistoryCursor {
  lineId: string
  rowInLine: number
}

export interface CoreV2HistoryRange {
  startLineId: string
  startCol: number
  endLineId: string
  endCol: number
}

export interface CoreV2HistoryWindowRequest {
  terminalId: string
  mode: CoreV2HistoryWindowMode
  limit: number
  cols: number
  token?: string | undefined
  generation?: string | number | bigint | undefined
  beforeCursor?: CoreV2HistoryCursor | undefined
  afterCursor?: CoreV2HistoryCursor | undefined
  boundaryFirstLineId?: string | undefined
  boundaryLastLineId?: string | undefined
  range?: CoreV2HistoryRange | undefined
}

export interface CoreV2HistoryCellStyle {
  fg?: string | undefined
  bg?: string | undefined
  bold?: boolean | undefined
  italic?: boolean | undefined
  underline?: boolean | undefined
  blink?: boolean | undefined
  reverse?: boolean | undefined
  strikethrough?: boolean | undefined
}

export interface CoreV2HistoryCell {
  text: string
  width: number
  style?: CoreV2HistoryCellStyle | undefined
  linkUrl?: string | undefined
  linkParams?: string | undefined
}

export interface CoreV2HistoryRow {
  index: number
  cells: CoreV2HistoryCell[]
  tailFillStyle?: CoreV2HistoryCellStyle | undefined
  kind?: string | undefined
  wrapped?: boolean | undefined
  ownership?: string | undefined
  timestampUnixMs?: number | undefined
  fixedGrid?: boolean | undefined
  screenCols?: number | undefined
  screenRow?: number | undefined
  screenRowSet?: boolean | undefined
  logicalLineId?: string | undefined
  rowInLine?: number | undefined
  logicalStartCol?: number | undefined
}

export interface CoreV2HistoryLineSpan {
  startRow: number
  endRow: number
  rowKind?: string | undefined
  logicalLineId: string
  timestampStartUnixMs?: number | undefined
  timestampEndUnixMs?: number | undefined
  clippedBefore: boolean
  clippedAfter: boolean
}

export interface CoreV2HistoryWindow {
  terminalId: string
  token: string
  op: CoreV2HistoryWindowOp
  cols: number
  renderRows: CoreV2HistoryRow[]
  lines: CoreV2HistoryLineSpan[]
  totalRows: number
  logicalTotal: number
  hasMore: boolean
  generation: string
  viewportAnchor?: {
    topLineId: string
    topCellOffset: number
    atEnd: boolean
    screenCols: number
    screenRows: number
  } | undefined
  firstRowId?: string | undefined
  lastRowId?: string | undefined
  firstLineId?: string | undefined
  lastLineId?: string | undefined
}

export interface CoreV2HistoryCopyRequest {
  terminalId: string
  token: string
  cols: number
  generation?: string | number | bigint | undefined
  boundaryFirstLineId?: string | undefined
  boundaryLastLineId?: string | undefined
  range: CoreV2HistoryRange
}

export type CoreV2HistorySearchDirection = 'forward' | 'backward'
export type CoreV2HistorySearchMode = 'text' | 'glob' | 'regex'

export interface CoreV2HistoryTextPosition {
  lineId: string
  col: number
}

export interface CoreV2HistorySearchRequest {
  terminalId: string
  token: string
  generation?: string | number | bigint | undefined
  query: string
  mode?: CoreV2HistorySearchMode | undefined
  direction: CoreV2HistorySearchDirection
  cols: number
  limit: number
  start?: CoreV2HistoryTextPosition | undefined
}

export type CoreV2HistorySearchResult =
  | { found: false; wrapped: false }
  | { found: true; match: CoreV2HistoryRange; window: CoreV2HistoryWindow; wrapped: boolean }

export interface CoreV2HistorySearchScanRequest {
  terminalId: string
  token: string
  generation?: string | number | bigint | undefined
  query: string
  mode?: CoreV2HistorySearchMode | undefined
  cols: number
  maxMatches: number
  start?: CoreV2HistoryTextPosition | undefined
}

export interface CoreV2HistorySearchScanResult {
  matches: CoreV2HistoryRange[]
  next?: CoreV2HistoryTextPosition | undefined
  done: boolean
}

export interface CoreV2HistoryReleaseRequest {
  terminalId: string
  token: string
}

export function coreV2HistorySearchFromAPI(value: HistorySearchResult): CoreV2HistorySearchResult {
  if (!value.found) return { found: false, wrapped: false }
  if (!value.match || !value.window) throw new Error('history search returned an incomplete match')
  return {
    found: true,
    match: historyRangeFromAPI(value.match),
    window: coreV2HistoryWindowFromAPI(value.window),
    wrapped: value.wrapped,
  }
}

export function coreV2HistorySearchScanFromAPI(value: HistorySearchResult): CoreV2HistorySearchScanResult {
  const matches = value.scanMatches.map(historyRangeFromAPI)
  const next = value.scanNext && value.scanNext.lineId !== 0n
    ? { lineId: value.scanNext.lineId.toString(), col: value.scanNext.col }
    : undefined
  if (!value.scanDone && (matches.length === 0 || !next)) {
    throw new Error('history search scan returned an incomplete batch')
  }
  return { matches, next, done: value.scanDone }
}

export function coreV2HistoryWindowFromAPI(value: HistoryWindowResult): CoreV2HistoryWindow {
  return {
    terminalId: value.terminal?.terminalId ?? '',
    token: value.token,
    op: historyWindowOp(value.operation),
    cols: value.size?.cols ?? 0,
    renderRows: value.rows.map(historyRowFromAPI),
    lines: value.lines.map(historyLineFromAPI),
    totalRows: value.totalRows,
    logicalTotal: value.logicalTotal,
    hasMore: value.hasMore,
    generation: value.historyGeneration.toString(),
    viewportAnchor: value.viewportAnchor ? {
      topLineId: value.viewportAnchor.topLineId.toString(),
      topCellOffset: value.viewportAnchor.topCellOffset,
      atEnd: value.viewportAnchor.atEnd,
      screenCols: value.viewportAnchor.screenCols,
      screenRows: value.viewportAnchor.screenRows,
    } : undefined,
    firstRowId: optionalID(value.firstRowId),
    lastRowId: optionalID(value.lastRowId),
    firstLineId: optionalID(value.firstLineId),
    lastLineId: optionalID(value.lastLineId),
  }
}

function historyRowFromAPI(row: HistoryRow, index: number): CoreV2HistoryRow {
  return {
    index,
    cells: (row.row?.cells ?? []).map(historyCellFromAPI),
    tailFillStyle: historyStyleFromAPI(row.row?.tailFill),
    kind: optionalText(row.rowKind),
    wrapped: row.wrapped,
    ownership: historyOwnership(row.ownership),
    timestampUnixMs: unixNanoToMs(row.timestampUnixNano),
    fixedGrid: row.fixedGrid,
    screenCols: row.screenCols,
    screenRow: row.screenRows,
    screenRowSet: row.screenRowSet,
    logicalLineId: optionalID(row.logicalLineId),
    rowInLine: row.rowInLine,
  }
}

function historyCellFromAPI(cell: ScreenCell): CoreV2HistoryCell {
  return {
    text: cell.content,
    width: cell.width || 1,
    style: historyStyleFromAPI(cell.style),
    linkUrl: optionalText(cell.linkUrl),
    linkParams: optionalText(cell.linkParams),
  }
}

function historyLineFromAPI(line: HistoryLineSpan): CoreV2HistoryLineSpan {
  return {
    startRow: line.startRow,
    endRow: line.endRow,
    rowKind: optionalText(line.rowKind),
    logicalLineId: line.logicalLineId.toString(),
    timestampStartUnixMs: unixNanoToMs(line.timestampStartUnixNano),
    timestampEndUnixMs: unixNanoToMs(line.timestampEndUnixNano),
    clippedBefore: line.clippedBefore,
    clippedAfter: line.clippedAfter,
  }
}

function historyRangeFromAPI(value: HistoryRange): CoreV2HistoryRange {
  return {
    startLineId: value.startLineId.toString(),
    startCol: value.startCol,
    endLineId: value.endLineId.toString(),
    endCol: value.endCol,
  }
}

function historyStyleFromAPI(style: CellStyle | undefined): CoreV2HistoryCellStyle | undefined {
  if (!style) return undefined
  return cleanStyle({
    fg: optionalText(style.foreground),
    bg: optionalText(style.background),
    bold: style.bold,
    italic: style.italic,
    underline: style.underline,
    blink: style.blink,
    reverse: style.reverse,
    strikethrough: style.strikethrough,
  })
}

function historyOwnership(value: RowOwnership): string | undefined {
  switch (value) {
    case RowOwnership.PERSISTED: return 'persisted'
    case RowOwnership.LIVE_TAIL_RECLAIMED: return 'live_tail_reclaimed'
    case RowOwnership.LIVE_TAIL_LIVE: return 'live_tail_live'
    case RowOwnership.SCREEN: return 'screen'
    case RowOwnership.UNSPECIFIED: return undefined
    default: throw new Error(`unsupported history row ownership ${value}`)
  }
}

function historyWindowOp(value: HistoryWindowOperation): CoreV2HistoryWindowOp {
  switch (value) {
    case HistoryWindowOperation.REPLACE: return 'replace'
    case HistoryWindowOperation.PREPEND: return 'prepend'
    case HistoryWindowOperation.APPEND: return 'append'
    default: throw new Error(`unsupported history window operation ${value}`)
  }
}

function cleanStyle(style: CoreV2HistoryCellStyle): CoreV2HistoryCellStyle | undefined {
  return Object.values(style).some((value) => value !== undefined && value !== false) ? style : undefined
}

function optionalText(value: string): string | undefined {
  return value || undefined
}

function optionalID(value: bigint): string | undefined {
  return value === 0n ? undefined : value.toString()
}

function unixNanoToMs(value: bigint): number | undefined {
  return value === 0n ? undefined : Number(value / 1_000_000n)
}
