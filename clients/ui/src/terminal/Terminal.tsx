import { FitAddon } from '@xterm/addon-fit'
import { CanvasAddon } from '@xterm/addon-canvas'
import { WebglAddon } from '@xterm/addon-webgl'
import { Terminal as XTerm } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from 'react'
import { ArrowDown, ArrowUp, Search, X } from 'lucide-react'
import { hapticSelection } from '../platform/haptics'
import { addNativeKeyboardListener } from '../platform/nativeKeyboard'
import {
  applyTerminalModifiers,
  type TerminalModifierState,
} from './mobileTerminalInput'
import type {
  TerminalHistoryLogicalMatch,
  TerminalHistoryMatchRange,
  TerminalHistorySearchMode,
  TerminalResizeControl,
  TerminalScrollbackLoadResult,
  TerminalSnapshotPayload,
} from './terminalClient'
import { logTerminalDiagnostic, terminalNow } from './terminalDiagnostics'
import { describeTerminalMouseInput, traceTerminalScroll } from './terminalScrollTrace'
import { holdTerminalFrame, type TerminalFrameHold } from './terminalFrameHold'
import { historyReplayWithViewportTail, historyRequestAwaitingApply, historyViewportAfterApply, terminalScrollLineDelta, terminalViewportAtBottom, TerminalHistoryViewportController } from './terminalHistoryViewport'
import { appendTerminalText } from './terminalTextWindow'
import { useTerminalSession } from './useTerminalSession'
import type { ProtoClientSession } from '../core/protoClientSession'
import type { Terminal as RemoteTerminal } from '../core/model'
import { DEFAULT_TERMINAL_SETTINGS, resolveTerminalMomentumProfile, resolveTerminalTheme, type TerminalSettings } from './terminalSettings'
import { useTranslation } from 'react-i18next'
import '../i18n'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { NativeSelect } from '../ui/native-select'
import { Spinner } from '../ui/spinner'
import { useConnectionRecoveryOverlay, type ConnectionRecoveryOverlayIntent } from '../connection/ConnectionRecoveryOverlay'

// Keep history responses comfortably below WebRTC DataChannel's 64 KiB message limit.
const historyScrollbackPageRows = 100
const historyInitialPrefetchDelayMs = 420
const historyLoadSkipLogIntervalMs = 1000
const historyLoadingDelayMs = 2000
const historyLoadingMinVisibleMs = 180
const historyApplyBatchDelayMs = 160
const historyApplyScrollIdleMs = 180
const historyApplyMaxDelayMs = 900
const historyApplyWatchdogMs = 4000
const historyPullMaxOffsetPx = 44
const historyPullReleaseMs = 220
const xtermWriteStatsIntervalMs = 1000
const xtermWriteSlowMs = 500
const xtermWriteLargeChars = 128 * 1024
const liveOutputPendingSoftLimitChars = 512 * 1024
const liveOutputPendingHardLimitChars = 1024 * 1024
const liveOutputWatchdogMs = 4000
const eventLoopProbeIntervalMs = 1000
const eventLoopLagWarnMs = 250
const minimumTrustedTerminalCols = 2
const minimumTrustedTerminalRows = 2
const untrustedFitRetryMs = 120
const untrustedFitLongRetryMs = 500
const untrustedFitShortRetryLimit = 8

type TerminalDimensions = { cols: number; rows: number }

function trustedTerminalDimensions(dimensions: TerminalDimensions | undefined): TerminalDimensions | undefined {
  if (!dimensions) return undefined
  const cols = Math.floor(dimensions.cols)
  const rows = Math.floor(dimensions.rows)
  if (!Number.isFinite(cols) || !Number.isFinite(rows)) return undefined
  // localweb 初始挂载时 xterm 可能在父容器尚未完成布局前把高度夹成 1 行；
  // 这个瞬时值不能写回 core PTY，否则 latest screen 会被真实压扁。
  if (cols < minimumTrustedTerminalCols || rows < minimumTrustedTerminalRows) return undefined
  return { cols, rows }
}

async function waitForTerminalLayout(): Promise<void> {
  if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') {
    await Promise.resolve()
    return
  }
  await new Promise<void>((resolve) => {
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => resolve())
    })
  })
}

function terminalHistoryLoadedRowsLimit(settings: TerminalSettings): number {
  return Math.max(0, Math.trunc(settings.scrollback))
}

export function terminalHistoryPrefetchThresholdRows(configuredRows: number, viewportRows: number): number {
  return Math.max(0, Math.trunc(configuredRows), Math.max(1, Math.trunc(viewportRows)) * 3)
}

export function createTerminalOptions(settings: TerminalSettings): ConstructorParameters<typeof XTerm>[0] {
  return {
    allowProposedApi: false,
    cursorBlink: settings.cursorBlink,
    convertEol: false,
    fontFamily: settings.fontFamily,
    fontSize: settings.fontSize,
    screenReaderMode: true,
    scrollback: settings.scrollback,
    theme: resolveTerminalTheme(settings.themeId),
  }
}

function prefersReducedTerminalMotion(): boolean {
  return typeof window !== 'undefined' && window.matchMedia?.('(prefers-reduced-motion: reduce)').matches === true
}

function isNativeAndroidWebView(): boolean {
  if (typeof navigator === 'undefined') return false
  const userAgent = navigator.userAgent || ''
  const cap = globalThis as typeof globalThis & { Capacitor?: { isNativePlatform?: () => boolean } }
  return /Android/i.test(userAgent) && cap.Capacitor?.isNativePlatform?.() === true
}

function effectiveRendererMode(configuredRenderer: TerminalRenderer): TerminalRenderer {
  if (configuredRenderer === 'auto' && isNativeAndroidWebView()) {
    return 'canvas'
  }
  return configuredRenderer
}

function historyLogicalMatchesToVisualRanges(
  history: NonNullable<TerminalSnapshotPayload['history']>,
  matches: TerminalHistoryLogicalMatch[],
): TerminalHistoryMatchRange[] {
  const lineIds = history.rowLogicalLineIds ?? []
  const rowStarts = history.rowLogicalStartCols ?? []
  const ranges: TerminalHistoryMatchRange[] = []
  for (const match of matches) {
    for (let row = 0; row < lineIds.length; row += 1) {
      const lineId = lineIds[row]
      if (!lineId || !historyLineWithinMatch(lineId, match)) continue
      const rowStart = Math.max(0, rowStarts[row] ?? 0)
      const nextStart = lineIds[row + 1] === lineId ? Math.max(rowStart, rowStarts[row + 1] ?? rowStart + history.cols) : rowStart + history.cols
      const matchStart = lineId === match.startLineId ? match.startCol : rowStart
      const matchEnd = lineId === match.endLineId ? match.endCol : nextStart
      const startCol = Math.max(rowStart, matchStart) - rowStart
      const endCol = Math.min(nextStart, matchEnd) - rowStart
      if (endCol > startCol) ranges.push({ row, startCol, endCol })
    }
  }
  return ranges
}

function historyLineWithinMatch(lineId: string, match: TerminalHistoryLogicalMatch): boolean {
  try {
    const line = BigInt(lineId)
    return line >= BigInt(match.startLineId) && line <= BigInt(match.endLineId)
  } catch {
    return lineId === match.startLineId || lineId === match.endLineId
  }
}

function sameHistoryLogicalMatch(left: TerminalHistoryLogicalMatch, right: TerminalHistoryLogicalMatch): boolean {
  return left.startLineId === right.startLineId && left.startCol === right.startCol &&
    left.endLineId === right.endLineId && left.endCol === right.endCol
}

function historySearchRailOffset(match: TerminalHistoryLogicalMatch, totalLines: number): string {
  const line = Number.parseInt(match.startLineId, 10)
  if (!Number.isFinite(line) || totalLines <= 1) return '0%'
  return `${Math.max(0, Math.min(100, ((line - 1) / (totalLines - 1)) * 100))}%`
}

interface PendingHistoryApply {
  revision: number
  cols: number
  loadedRows: number
  prependedRows: number
  operation: 'replace' | 'prepend'
  restoreViewportY: number | null
  viewportTop?: number | undefined
  absoluteViewportY?: number | undefined
  searchMatchRanges?: TerminalHistoryMatchRange[] | undefined
  text: string
}

export type TerminalRenderer = 'auto' | 'webgl' | 'canvas' | 'dom'

export interface TerminalProps {
  machineId: string
  terminalId: string
  session: ProtoClientSession
  className?: string
  onReady?: () => void
  onInput?: ((data: string) => boolean) | undefined
  onCursorMove?: (() => void) | undefined
  onBufferChange?: ((isAlternate: boolean) => void) | undefined
  onResizeControl?: ((control: TerminalResizeControl) => void) | undefined
  onTerminalInfoChange?: ((terminal: RemoteTerminal) => void) | undefined
  onHistorySearchOpenChange?: ((open: boolean) => void) | undefined
  modifierState?: TerminalModifierState | undefined
  onModifierStateChange?: ((state: TerminalModifierState) => void) | undefined
  selectionMode?: boolean | undefined
  renderer?: TerminalRenderer | undefined
  settings?: TerminalSettings | undefined
  preventFocus?: boolean | undefined
  suppressConnectingOverlay?: boolean | undefined
  historyOnly?: boolean | undefined
}

export interface TerminalHandle {
  sendInput(data: string): boolean
  sendResize(cols: number, rows: number): void
  requestResizeOwner(): Promise<TerminalResizeControl>
  releaseResizeOwner(): Promise<TerminalResizeControl>
  setResizeLock(locked: boolean): Promise<TerminalResizeControl>
  reattach(session: ProtoClientSession, options?: { forceTerminalChannel?: boolean }): void
  focus(): void
  blur(): void
  fit(): void
  openHistorySearch(): void
  pasteText(text: string): boolean
  selectAll(): void
  selectVisible(): void
  getSelection(): string
  getSelectionForClipboard(): Promise<string>
  hasSelection(): boolean
  clearSelection(): void
  getCursorInfo(): { cursorY: number; rows: number; lineHeight: number } | null
  adjustInputPosition(bottomOffset: number): void
  getBufferType(): 'normal' | 'alternate'
  updateOptions(opts: { fontSize?: number; cursorBlink?: boolean; fontFamily?: string; scrollback?: number }): void
}

export const Terminal = forwardRef<TerminalHandle, TerminalProps>(function Terminal(
  {
    machineId,
    terminalId,
    session,
    className,
    onReady,
    onInput,
    onCursorMove,
    onBufferChange,
    onResizeControl,
    onTerminalInfoChange,
    onHistorySearchOpenChange,
    modifierState,
    onModifierStateChange,
    selectionMode = false,
    renderer,
    settings = DEFAULT_TERMINAL_SETTINGS,
    preventFocus = false,
    suppressConnectingOverlay = false,
    historyOnly = false,
  },
  ref,
) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement | null>(null)
  const historyStatusRef = useRef<HTMLDivElement | null>(null)
  const historyStatusTextRef = useRef<HTMLSpanElement | null>(null)
  const historySearchBottomResumeSuppressedRef = useRef(false)
  const historyMetadataRef = useRef<NonNullable<TerminalSnapshotPayload['history']> | undefined>(undefined)
  const updateHistoryStatusRef = useRef<() => void>(() => {})
  const formatHistoryStatusRef = useRef((current: number, total: number, timestamp?: number) => (
    t('terminal.tools.historyPosition', {
      current,
      total,
      time: timestamp === undefined ? '' : formatHistoryTimestamp(timestamp),
    })
  ))
  const xtermRef = useRef<XTerm | null>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const lastWrittenTextRef = useRef('')
  const latestTerminalTextRef = useRef('')
  const lastSentResizeRef = useRef<{ cols: number; rows: number } | null>(null)
  const isOpenRef = useRef(false)
  const pendingLiveOutputRef = useRef('')
  const liveOutputInFlightRef = useRef(false)
  const liveOutputGenerationRef = useRef(0)
  const liveOutputDroppedCharsRef = useRef(0)
  const liveOutputSyncLostRef = useRef(false)
  const liveOutputWatchdogRef = useRef<number | null>(null)
  const markLiveOutputSyncLostRef = useRef<(reason?: string) => void>(() => {})
  const surfaceReadyRef = useRef(false)
  const initialViewportReadyRef = useRef(false)
  const canSendResizeRef = useRef(false)
  const historyLoadingRef = useRef(false)
  const historyApplyingRef = useRef(false)
  const historyRestoreViewportOnLoadRef = useRef(false)
  const pullingHistoryRef = useRef(false)
  const historyRevisionAppliedRef = useRef(0)
  const historyLoadedRowsAppliedRef = useRef(0)
  const historyLoadedRowsRequestedRef = useRef(0)
  const historyHasMoreRef = useRef(true)
  const lastHistoryLoadSkipLogRef = useRef<{ reason: string; at: number } | null>(null)
  const historyLoadingShownAtRef = useRef(0)
  const historyLoadingShowTimerRef = useRef<number | null>(null)
  const historyLoadingHideTimerRef = useRef<number | null>(null)
  const lastSnapshotTextRef = useRef('')
  const lastLiveScreenSubmittedRevisionRef = useRef<bigint | undefined>(undefined)
  const latestLiveScreenRevisionRef = useRef<bigint | undefined>(undefined)
  const markLiveScreenSubmittedRef = useRef<(revision: bigint) => void>(() => {})
  const markLiveScreenCompletedRef = useRef<(revision: bigint) => void>(() => {})
  const recoveryRevisionAppliedRef = useRef(0)
  const pendingHistoryViewportRef = useRef<number | null>(null)
  const pendingHistoryApplyRef = useRef<PendingHistoryApply | null>(null)
  const historySearchHighlightLayerRef = useRef<HTMLDivElement | null>(null)
  const historyApplyQueuedAtRef = useRef(0)
  const historyApplyTimerRef = useRef<number | null>(null)
  const historyLoadArmedByUserRef = useRef(false)
  const historyViewportControllerRef = useRef(new TerminalHistoryViewportController())
  const primedHistoryFrameRef = useRef<TerminalFrameHold | null>(null)
  const historyRequestColsRef = useRef(0)
  const historyProjectionReloadPendingRef = useRef(false)
  const lastHistoryScrollActivityAtRef = useRef(0)
  const loadScrollbackRef = useRef<(limit?: number, alternate?: boolean, cols?: number) => Promise<TerminalScrollbackLoadResult>>(async () => ({
    loadedRows: 0,
    totalRows: 0,
    hasMore: false,
    alternate: false,
  }))
  const resetScrollbackRef = useRef<() => void>(() => {})
  const retryHistoryLoadRef = useRef<() => void>(() => {})
  const freezeScrollbackRef = useRef<() => void>(() => {})
  const resumeLiveScrollbackRef = useRef<() => string>(() => '')
  const resumeFrozenHistoryAtBottomRef = useRef<(includePrimed?: boolean) => boolean>(() => false)
  const maybePrefetchScrollbackRef = useRef<() => void>(() => {})
  const enterSelectionModeRef = useRef<() => void>(() => {})
  const exitSelectionModeRef = useRef<() => void>(() => {})
  const selectionModeExitPendingRef = useRef(false)
  const scheduleHistoryApplyRef = useRef<(immediate?: boolean) => void>(() => {})
  const resetTransientViewportOffsetRef = useRef<() => void>(() => {})
  const hasTerminalSnapshotRef = useRef(false)
  const snapshotAlternateScreenRef = useRef(false)
  const fitFrameRef = useRef<number | null>(null)
  const fitRetryTimerRef = useRef<number | null>(null)
  const fitRetryCountRef = useRef(0)
  const preventFocusRef = useRef(preventFocus)
  const bottomAnchorFrameRef = useRef<number | null>(null)
  const xtermWriteStatsRef = useRef({
    writes: 0,
    chars: 0,
    pendingCallbacks: 0,
    lastLogAt: terminalNow(),
    lastChars: 0,
    lastWrites: 0,
  })

  useEffect(() => {
    preventFocusRef.current = preventFocus
  }, [preventFocus])
  const terminalDisposedRef = useRef(true)
  const terminalGenerationRef = useRef(0)
  const settingsRef = useRef(settings)
  const modifierStateRef = useRef<TerminalModifierState | undefined>(undefined)
  const onModifierStateChangeRef = useRef<((state: TerminalModifierState) => void) | undefined>(undefined)
  const onInputRef = useRef<((data: string) => boolean) | undefined>(undefined)
  const onCursorMoveRef = useRef<(() => void) | undefined>(undefined)
  const onBufferChangeRef = useRef<((isAlternate: boolean) => void) | undefined>(undefined)
  const onResizeControlRef = useRef<((control: TerminalResizeControl) => void) | undefined>(undefined)
  const selectionModeRef = useRef(false)
  const selectionResetHandlersRef = useRef(new Set<() => void>())
  const [surfaceReady, setSurfaceReady] = useState(false)
  const [initialViewportReady, setInitialViewportReady] = useState(false)
  const [xtermReady, setXtermReady] = useState(false)
  const [historyLoadingVisible, setHistoryLoadingVisible] = useState(false)
  const [historyStatusVisible, setHistoryStatusVisible] = useState(false)
  const [historyLoadFailure, setHistoryLoadFailure] = useState<'none' | 'reloadable' | 'line-too-large'>('none')
  const [historySearchOpen, setHistorySearchOpen] = useState(false)
  const [historySearchQuery, setHistorySearchQuery] = useState('')
  const [historySearchMode, setHistorySearchMode] = useState<TerminalHistorySearchMode>('text')
  const [historySearchBusy, setHistorySearchBusy] = useState(false)
  const [historySearchScanBusy, setHistorySearchScanBusy] = useState(false)
  const [historySearchScanComplete, setHistorySearchScanComplete] = useState(false)
  const [historySearchMatches, setHistorySearchMatches] = useState<TerminalHistoryLogicalMatch[]>([])
  const [historySearchActiveMatch, setHistorySearchActiveMatch] = useState<TerminalHistoryLogicalMatch | null>(null)
  const [historySearchWrapped, setHistorySearchWrapped] = useState(false)
  const [historySearchMessage, setHistorySearchMessage] = useState('')
  const historySearchOpenRef = useRef(false)
  const historySearchMatchesRef = useRef<TerminalHistoryLogicalMatch[]>([])
  const historySearchScanKeyRef = useRef('')
  const historySearchLastMatchRef = useRef<{
    query: string
    mode: TerminalHistorySearchMode
    startLineId: string
    startCol: number
    endLineId: string
    endCol: number
  } | null>(null)

  const clearHistorySearchHighlight = useCallback(() => {
    historySearchHighlightLayerRef.current?.replaceChildren()
  }, [])

  const paintHistorySearchHighlights = useCallback((term: XTerm, activeRanges?: TerminalHistoryMatchRange[]) => {
    clearHistorySearchHighlight()
    const history = historyMetadataRef.current
    const active = historySearchLastMatchRef.current
    const layer = historySearchHighlightLayerRef.current
    if (!history || !layer || term.cols <= 0 || term.rows <= 0) return
    const weakMatches = active
      ? historySearchMatchesRef.current.filter((match) => !sameHistoryLogicalMatch(match, active))
      : historySearchMatchesRef.current
    const weakRanges = historyLogicalMatchesToVisualRanges(history, weakMatches)
    const currentRanges = activeRanges ?? (active ? historyLogicalMatchesToVisualRanges(history, [active]) : [])
    const screen = layer.parentElement
    const screenRect = screen?.getBoundingClientRect()
    const screenWidth = screen?.clientWidth || screenRect?.width || term.cols
    const screenHeight = screen?.clientHeight || screenRect?.height || term.rows
    const cellWidth = screenWidth / term.cols
    const cellHeight = screenHeight / term.rows
    const viewportY = term.buffer.active.viewportY
    const fragment = document.createDocumentFragment()
    const decorate = (range: TerminalHistoryMatchRange, current: boolean) => {
      const startCol = Math.max(0, Math.min(term.cols - 1, Math.trunc(range.startCol)))
      const endCol = Math.max(startCol + 1, Math.min(term.cols, Math.trunc(range.endCol)))
      const visibleRow = Math.trunc(range.row) - viewportY
      if (visibleRow < 0 || visibleRow >= term.rows) return
      const highlight = document.createElement('span')
      highlight.className = current
        ? 'anytty-history-search-highlight anytty-history-search-highlight-current'
        : 'anytty-history-search-highlight'
      highlight.dataset.current = current ? 'true' : 'false'
      highlight.style.transform = `translate3d(${startCol * cellWidth}px, ${visibleRow * cellHeight}px, 0)`
      highlight.style.width = `${(endCol - startCol) * cellWidth}px`
      highlight.style.height = `${cellHeight}px`
      fragment.append(highlight)
    }
    weakRanges.forEach((range) => decorate(range, false))
    currentRanges.forEach((range) => decorate(range, true))
    layer.append(fragment)
  }, [clearHistorySearchHighlight])

  useEffect(() => {
    historySearchOpenRef.current = historySearchOpen
    onHistorySearchOpenChange?.(historySearchOpen)
  }, [historySearchOpen, onHistorySearchOpenChange])

  useEffect(() => () => onHistorySearchOpenChange?.(false), [onHistorySearchOpenChange])

  const showHistoryLoading = useCallback(() => {
    if (historyLoadingHideTimerRef.current !== null) {
      window.clearTimeout(historyLoadingHideTimerRef.current)
      historyLoadingHideTimerRef.current = null
    }
    if (historyLoadingShownAtRef.current > 0) {
      setHistoryLoadingVisible(true)
      return
    }
    if (historyLoadingShowTimerRef.current !== null) return
    historyLoadingShowTimerRef.current = window.setTimeout(() => {
      historyLoadingShowTimerRef.current = null
      historyLoadingShownAtRef.current = terminalNow()
      setHistoryLoadingVisible(true)
    }, historyLoadingDelayMs)
  }, [])

  const hideHistoryLoading = useCallback((force = false) => {
    if (historyLoadingShowTimerRef.current !== null) {
      window.clearTimeout(historyLoadingShowTimerRef.current)
      historyLoadingShowTimerRef.current = null
    }
    if (historyLoadingHideTimerRef.current !== null) {
      window.clearTimeout(historyLoadingHideTimerRef.current)
      historyLoadingHideTimerRef.current = null
    }
    if (force || historyLoadingShownAtRef.current === 0) {
      historyLoadingShownAtRef.current = 0
      setHistoryLoadingVisible(false)
      return
    }
    const elapsed = terminalNow() - historyLoadingShownAtRef.current
    const remaining = historyLoadingMinVisibleMs - elapsed
    if (remaining <= 0) {
      historyLoadingShownAtRef.current = 0
      setHistoryLoadingVisible(false)
      return
    }
    historyLoadingHideTimerRef.current = window.setTimeout(() => {
      historyLoadingHideTimerRef.current = null
      historyLoadingShownAtRef.current = 0
      setHistoryLoadingVisible(false)
    }, remaining)
  }, [])

  const logTerminal = useCallback((event: string, input: {
    level?: 'debug' | 'info' | 'warn' | 'error'
    details?: Record<string, unknown> | undefined
  } = {}) => {
    logTerminalDiagnostic(`xterm.${event}`, {
      level: input.level,
      machineId,
      terminalId,
      details: input.details,
    })
  }, [machineId, terminalId])

  const writeToXterm = useCallback((
    term: XTerm,
    text: string,
    reason: string,
    onDone?: () => void,
  ) => {
    const stats = xtermWriteStatsRef.current
    const startedAt = terminalNow()
    stats.writes += 1
    stats.chars += text.length
    stats.pendingCallbacks += 1
    const writeId = stats.writes
    const now = terminalNow()
    logTerminal('write_enqueue', {
      details: {
        writeId,
        reason,
        chars: text.length,
        pendingCallbacks: stats.pendingCallbacks,
      },
    })
    if (text.length >= xtermWriteLargeChars || now - stats.lastLogAt >= xtermWriteStatsIntervalMs) {
      const elapsedSeconds = Math.max(0.001, (now - stats.lastLogAt) / 1000)
      const intervalChars = stats.chars - stats.lastChars
      const intervalWrites = stats.writes - stats.lastWrites
      logTerminal('write_start', {
        level: text.length >= xtermWriteLargeChars ? 'warn' : 'debug',
        details: {
          writeId,
          reason,
          chars: text.length,
          writes: stats.writes,
          totalChars: stats.chars,
          intervalWrites,
          intervalChars,
          charsPerSecond: Math.round(intervalChars / elapsedSeconds),
          pendingCallbacks: stats.pendingCallbacks,
        },
      })
      stats.lastLogAt = now
      stats.lastChars = stats.chars
      stats.lastWrites = stats.writes
    }
    let done = false
    const finish = () => {
      if (done) return
      done = true
      stats.pendingCallbacks = Math.max(0, stats.pendingCallbacks - 1)
      const elapsedMs = Math.round(terminalNow() - startedAt)
      if (elapsedMs >= xtermWriteSlowMs || text.length >= xtermWriteLargeChars) {
        logTerminal('write_done', {
          level: elapsedMs >= xtermWriteSlowMs ? 'warn' : 'debug',
          details: {
            writeId,
            reason,
            chars: text.length,
            elapsedMs,
            pendingCallbacks: stats.pendingCallbacks,
            bufferLength: term.buffer.active.length,
            viewportY: term.buffer.active.viewportY,
            rows: term.rows,
            cols: term.cols,
          },
        })
      }
      onDone?.()
    }
    try {
      term.write(text, finish)
    } catch (error) {
      stats.pendingCallbacks = Math.max(0, stats.pendingCallbacks - 1)
      logTerminal('write_error', {
        level: 'error',
        details: {
          writeId,
          reason,
          chars: text.length,
          message: error instanceof Error ? error.message : String(error),
        },
      })
      throw error
    }
  }, [logTerminal])

  const clearLiveOutputWatchdog = useCallback(() => {
    if (liveOutputWatchdogRef.current === null) return
    window.clearTimeout(liveOutputWatchdogRef.current)
    liveOutputWatchdogRef.current = null
  }, [])

  const markSurfaceReady = useCallback(() => {
    if (surfaceReadyRef.current) return
    surfaceReadyRef.current = true
    setSurfaceReady(true)
  }, [])

  const flushPendingTerminalOutput = useCallback(() => {
    if (terminalDisposedRef.current) return
    if (liveOutputInFlightRef.current) return
    const term = xtermRef.current
    const pending = pendingLiveOutputRef.current
    if (!term || !pending) return
    pendingLiveOutputRef.current = ''
    liveOutputInFlightRef.current = true
    liveOutputSyncLostRef.current = false
    const writeGeneration = liveOutputGenerationRef.current + 1
    liveOutputGenerationRef.current = writeGeneration
    clearLiveOutputWatchdog()
    liveOutputWatchdogRef.current = window.setTimeout(() => {
      liveOutputWatchdogRef.current = null
      if (!liveOutputInFlightRef.current || terminalDisposedRef.current || liveOutputGenerationRef.current !== writeGeneration) return
      const droppedChars = pendingLiveOutputRef.current.length + liveOutputDroppedCharsRef.current
      pendingLiveOutputRef.current = ''
      liveOutputDroppedCharsRef.current = 0
      liveOutputSyncLostRef.current = true
      liveOutputInFlightRef.current = false
      liveOutputGenerationRef.current += 1
      logTerminal('live_output_write_stalled', {
        level: 'warn',
        details: {
          inFlightChars: pending.length,
          droppedChars,
          watchdogMs: liveOutputWatchdogMs,
        },
      })
      markLiveOutputSyncLostRef.current('xterm renderer stalled while applying live output')
    }, liveOutputWatchdogMs)
    writeToXterm(term, pending, 'stream_output', () => {
      if (liveOutputGenerationRef.current !== writeGeneration) return
      clearLiveOutputWatchdog()
      const syncLost = liveOutputSyncLostRef.current
      liveOutputSyncLostRef.current = false
      liveOutputInFlightRef.current = false
      markSurfaceReady()
      if (!syncLost) {
        flushPendingTerminalOutput()
      }
    })
    lastWrittenTextRef.current = appendTerminalText(lastWrittenTextRef.current, pending)
  }, [clearLiveOutputWatchdog, logTerminal, markSurfaceReady, writeToXterm])

  const handleTerminalOutput = useCallback((text: string) => {
    if (terminalDisposedRef.current) return
    if (liveOutputSyncLostRef.current) return
    const historyViewport = historyViewportControllerRef.current
    const liveUpdateAlreadyDeferred = historyViewport.hasDeferredLiveUpdate
    if (!historyViewport.shouldRenderLiveUpdate()) {
      if (!liveUpdateAlreadyDeferred) {
        logTerminal('history_live_update_deferred', {
          details: { source: 'stream_output', chars: text.length },
        })
      }
      return
    }
    historyHasMoreRef.current = true
    const nextLength = pendingLiveOutputRef.current.length + text.length
    if (liveOutputInFlightRef.current && nextLength > liveOutputPendingSoftLimitChars) {
      const droppedChars = nextLength + liveOutputDroppedCharsRef.current
      pendingLiveOutputRef.current = ''
      liveOutputDroppedCharsRef.current = 0
      liveOutputSyncLostRef.current = true
      liveOutputGenerationRef.current += 1
      const hardLimitExceeded = nextLength > liveOutputPendingHardLimitChars
      logTerminal(hardLimitExceeded ? 'live_output_pending_hard_limit' : 'live_output_pending_soft_limit', {
        level: 'warn',
        details: {
          incomingChars: text.length,
          droppedChars,
          softLimitChars: liveOutputPendingSoftLimitChars,
          hardLimitChars: liveOutputPendingHardLimitChars,
        },
      })
      markLiveOutputSyncLostRef.current(hardLimitExceeded
        ? 'xterm live output buffer exceeded hard limit'
        : 'xterm live output buffer exceeded soft limit')
      return
    }
    pendingLiveOutputRef.current += text
    flushPendingTerminalOutput()
  }, [flushPendingTerminalOutput, logTerminal])

  const terminalSession = useTerminalSession({
    machineId,
    terminalId,
    session,
    onOutput: handleTerminalOutput,
    historyOnly,
  })

  useEffect(() => {
    if (terminalSession.terminalInfo) onTerminalInfoChange?.(terminalSession.terminalInfo)
  }, [onTerminalInfoChange, terminalSession.terminalInfo])
  markLiveOutputSyncLostRef.current = terminalSession.markSyncLost
  freezeScrollbackRef.current = terminalSession.freezeScrollback
  resumeLiveScrollbackRef.current = terminalSession.resumeLiveScrollback
  latestLiveScreenRevisionRef.current = terminalSession.terminalSnapshot?.liveRevision
  markLiveScreenSubmittedRef.current = terminalSession.markLiveScreenSubmitted
  markLiveScreenCompletedRef.current = terminalSession.markLiveScreenCompleted
  historyMetadataRef.current = terminalSession.terminalSnapshot?.history
  formatHistoryStatusRef.current = (current, total, timestamp) => t('terminal.tools.historyPosition', {
    current,
    total,
    time: timestamp === undefined ? '' : formatHistoryTimestamp(timestamp),
  })

  latestTerminalTextRef.current = terminalSession.terminalText
  settingsRef.current = settings
  hasTerminalSnapshotRef.current = terminalSession.terminalSnapshot !== null
  snapshotAlternateScreenRef.current = terminalSession.terminalSnapshot?.alternateScreen === true

  useEffect(() => {
    loadScrollbackRef.current = terminalSession.loadScrollback
  }, [terminalSession.loadScrollback])

  useEffect(() => {
    resetScrollbackRef.current = terminalSession.resetScrollback
  }, [terminalSession.resetScrollback])

  useEffect(() => {
    updateHistoryStatusRef.current()
    if (!terminalSession.terminalSnapshot?.history) {
      terminalSession.cancelHistorySearch()
      setHistorySearchOpen(false)
      setHistorySearchMessage('')
      historySearchLastMatchRef.current = null
      historySearchMatchesRef.current = []
      historySearchScanKeyRef.current = ''
      setHistorySearchMatches([])
      setHistorySearchActiveMatch(null)
      setHistorySearchScanBusy(false)
      setHistorySearchScanComplete(false)
      setHistorySearchWrapped(false)
      clearHistorySearchHighlight()
    }
  }, [clearHistorySearchHighlight, terminalSession.cancelHistorySearch, terminalSession.terminalSnapshot?.history?.revision])

  const runHistorySearchScan = useCallback(async (query: string, mode: TerminalHistorySearchMode, cols: number) => {
    const key = `${query}\u0000${mode}\u0000${cols}`
    if (historySearchScanKeyRef.current === key && (historySearchScanBusy || historySearchScanComplete)) return
    historySearchScanKeyRef.current = key
    historySearchMatchesRef.current = []
    setHistorySearchMatches([])
    setHistorySearchScanBusy(true)
    setHistorySearchScanComplete(false)
    try {
      await terminalSession.scanScrollback(query, cols, (batch) => {
        if (historySearchScanKeyRef.current !== key) return
        setHistorySearchMatches((current) => {
          const next = [...current]
          for (const match of batch.matches) {
            if (!next.some((candidate) => sameHistoryLogicalMatch(candidate, match))) next.push(match)
          }
          historySearchMatchesRef.current = next
          return next
        })
        if (batch.done) setHistorySearchScanComplete(true)
      }, mode)
    } catch (error) {
      if (!(error instanceof DOMException && error.name === 'AbortError') && historySearchScanKeyRef.current === key) {
        setHistorySearchMessage(error instanceof Error && error.message.trim()
          ? error.message
          : t('terminal.tools.searchUnavailable'))
      }
    } finally {
      if (historySearchScanKeyRef.current === key) setHistorySearchScanBusy(false)
    }
  }, [historySearchScanBusy, historySearchScanComplete, t, terminalSession.scanScrollback])

  const runHistorySearch = useCallback(async (direction: 'forward' | 'backward') => {
    const query = historySearchQuery
    const term = xtermRef.current
    if (!query.trim() || !term || historySearchBusy) return
    setHistorySearchBusy(true)
    setHistorySearchMessage('')
    try {
      if (!historyMetadataRef.current) {
        await loadScrollbackRef.current(historyScrollbackPageRows, false, term.cols)
      }
      void runHistorySearchScan(query, historySearchMode, term.cols)
      const previous = historySearchLastMatchRef.current?.query === query &&
        historySearchLastMatchRef.current?.mode === historySearchMode
        ? historySearchLastMatchRef.current
        : null
      const start = previous
        ? direction === 'forward'
          ? { lineId: previous.endLineId, col: previous.endCol }
          : { lineId: previous.startLineId, col: previous.startCol }
        : undefined
      const result = await terminalSession.searchScrollback(query, direction, term.cols, start, historySearchMode)
      if (!result.found || !result.match) {
        historySearchLastMatchRef.current = null
        setHistorySearchActiveMatch(null)
        setHistorySearchWrapped(false)
        paintHistorySearchHighlights(term)
        setHistorySearchMessage(t('terminal.tools.searchNoResults'))
        return
      }
      historySearchLastMatchRef.current = { query, mode: historySearchMode, ...result.match }
      setHistorySearchActiveMatch(result.match)
      setHistorySearchWrapped(result.wrapped)
      setHistorySearchMessage(result.wrapped
        ? t('terminal.tools.searchWrapped')
        : t('terminal.tools.searchMatch', { line: result.match.startLineId }))
    } catch (error) {
      if (!(error instanceof DOMException && error.name === 'AbortError')) {
        clearHistorySearchHighlight()
        setHistorySearchMessage(error instanceof Error && error.message.trim()
          ? error.message
          : t('terminal.tools.searchUnavailable'))
      }
    } finally {
      setHistorySearchBusy(false)
    }
  }, [historySearchBusy, historySearchMode, historySearchQuery, paintHistorySearchHighlights, runHistorySearchScan, t, terminalSession.searchScrollback])

  useEffect(() => {
    const term = xtermRef.current
    if (!term || !historySearchOpen) return
    paintHistorySearchHighlights(term)
  }, [historySearchActiveMatch, historySearchMatches, historySearchOpen, paintHistorySearchHighlights, terminalSession.terminalSnapshot?.history?.revision])

  useEffect(() => {
    surfaceReadyRef.current = false
    initialViewportReadyRef.current = false
    lastLiveScreenSubmittedRevisionRef.current = undefined
    setSurfaceReady(false)
    setInitialViewportReady(false)
  }, [historyOnly, machineId, renderer, session, settings.renderer, terminalId])

  const isOpen = historyOnly
    ? terminalSession.historyReady
    : terminalSession.snapshot.terminalChannels[terminalId]?.state === 'open'
  const showConnectingOverlay = !suppressConnectingOverlay && (!isOpen || !surfaceReady)
  const terminalSurfaceVisible = initialViewportReady && !showConnectingOverlay
  const [showDelayedConnectingOverlay, setShowDelayedConnectingOverlay] = useState(false)
  const inputFailureVisible = Boolean(terminalSession.inputRecoveryFailure)
  const channelFailure = terminalSession.snapshot.visibleError
  const reattachTerminal = terminalSession.reattach
  useEffect(() => {
    if (!showConnectingOverlay) {
      setShowDelayedConnectingOverlay(false)
      return
    }
    const timer = window.setTimeout(() => setShowDelayedConnectingOverlay(true), 300)
    return () => window.clearTimeout(timer)
  }, [showConnectingOverlay])
  const terminalConnectionOverlayIntent = useMemo<ConnectionRecoveryOverlayIntent | null>(() => {
    if (channelFailure) {
      return {
        kind: 'failed',
        title: t('errors.connectionProblemTitle'),
        description: channelFailure.message,
        ...(channelFailure.recoverable ? {
          action: {
            label: t('workspace.connection.retry'),
            onClick: () => reattachTerminal(session, { forceTerminalChannel: true }),
          },
        } : {}),
      }
    }
    if (inputFailureVisible) {
      return {
        kind: 'recovering',
        title: t('connectionStatus.recovering'),
        description: t('connectionStatus.inputPaused'),
      }
    }
    if (showDelayedConnectingOverlay) {
      return {
        kind: 'recovering',
        title: t('workspace.connectingTerminal'),
      }
    }
    return null
  }, [channelFailure, inputFailureVisible, reattachTerminal, session, showDelayedConnectingOverlay, t])
  useConnectionRecoveryOverlay(terminalConnectionOverlayIntent)

  useEffect(() => {
    if (!isOpen || !surfaceReady) return
    if (terminalSession.terminalSnapshot?.alternateScreen === true) return
    const timer = window.setTimeout(() => {
      if (terminalDisposedRef.current || historyViewportControllerRef.current.isLiveUpdateDeferred) return
      if (historyLoadingRef.current || historyApplyingRef.current || pendingHistoryApplyRef.current) return
      const term = xtermRef.current
      if (!term || term.buffer.active.type !== 'normal' || term.cols < minimumTrustedTerminalCols) return
      const limit = Math.min(historyScrollbackPageRows, terminalHistoryLoadedRowsLimit(settingsRef.current))
      if (limit <= 0) return
      void terminalSession.prefetchScrollback(limit, false, term.cols)
    }, historyInitialPrefetchDelayMs)
    return () => window.clearTimeout(timer)
  }, [
    isOpen,
    surfaceReady,
    terminalSession.prefetchScrollback,
    terminalSession.terminalSnapshot?.alternateScreen,
    terminalSession.terminalText,
  ])

  useEffect(() => {
    if (!historyOnly || !terminalSession.historyReady || !xtermReady || surfaceReady) return
    const term = xtermRef.current
    if (!term || term.cols < minimumTrustedTerminalCols) return
    const limit = Math.min(historyScrollbackPageRows, terminalHistoryLoadedRowsLimit(settingsRef.current))
    if (limit <= 0) {
      markSurfaceReady()
      return
    }
    let cancelled = false
    void terminalSession.loadScrollback(limit, false, term.cols).then((result) => {
      if (!cancelled && result.loadedRows === 0) markSurfaceReady()
    }).catch(() => {
      if (!cancelled) markSurfaceReady()
    })
    return () => { cancelled = true }
  }, [historyOnly, markSurfaceReady, surfaceReady, terminalSession.historyReady, terminalSession.loadScrollback, xtermReady])

  const isScrolledToBottom = useCallback((term: XTerm) => {
    const buffer = term.buffer.active
    return terminalViewportAtBottom(buffer.viewportY, buffer.baseY)
  }, [])

  const reloadHistoryProjectionWhenIdle = useCallback(() => {
    if (!historyProjectionReloadPendingRef.current || historyLoadingRef.current || historyApplyingRef.current) return
    historyProjectionReloadPendingRef.current = false
    pendingHistoryApplyRef.current = null
    pendingHistoryViewportRef.current = null
    retryHistoryLoadRef.current()
  }, [])

  const fitAndMaybeSendResize = useCallback((allowInitialReveal = false) => {
    if (terminalDisposedRef.current) return
    const term = xtermRef.current
    const fitAddon = fitAddonRef.current
    const container = containerRef.current
    if (!term || !fitAddon || !container) return
    const shouldKeepBottom = isScrolledToBottom(term)

    let dimensions: TerminalDimensions | undefined
    try {
      const proposedDimensions = fitAddon.proposeDimensions()
      dimensions = trustedTerminalDimensions(proposedDimensions)
      if (!dimensions) {
        if (fitRetryTimerRef.current === null && typeof window !== 'undefined') {
          const retryCount = fitRetryCountRef.current
          fitRetryCountRef.current += 1
          fitRetryTimerRef.current = window.setTimeout(() => {
            fitRetryTimerRef.current = null
            fitAndMaybeSendResize(allowInitialReveal)
          }, retryCount < untrustedFitShortRetryLimit ? untrustedFitRetryMs : untrustedFitLongRetryMs)
          if (retryCount === 0) {
            logTerminal('fit_untrusted_dimensions', {
              level: 'warn',
              details: {
                proposedCols: proposedDimensions?.cols,
                proposedRows: proposedDimensions?.rows,
                containerWidth: container.clientWidth,
                containerHeight: container.clientHeight,
              },
            })
          }
        }
        return
      }
      if (terminalDisposedRef.current || xtermRef.current !== term) return
      if (fitRetryTimerRef.current !== null && typeof window !== 'undefined') {
        window.clearTimeout(fitRetryTimerRef.current)
        fitRetryTimerRef.current = null
      }
      fitRetryCountRef.current = 0
      const previousCols = term.cols
      term.resize(dimensions.cols, dimensions.rows)
      if (allowInitialReveal && !initialViewportReadyRef.current) {
        initialViewportReadyRef.current = true
        setInitialViewportReady(true)
        logTerminal('initial_viewport_ready', {
          details: {
            cols: dimensions.cols,
            rows: dimensions.rows,
            containerWidth: container.clientWidth,
            containerHeight: container.clientHeight,
          },
        })
      }
      if (shouldKeepBottom) term.scrollToBottom()
      if (previousCols !== dimensions.cols && (
        historyViewportControllerRef.current.isLiveUpdateDeferred ||
        historyLoadedRowsAppliedRef.current > 0 ||
        historyLoadedRowsRequestedRef.current > 0
      )) {
        historyProjectionReloadPendingRef.current = true
        reloadHistoryProjectionWhenIdle()
      }
    } catch {
      // xterm can leave delayed viewport/fit work behind while React is unmounting.
      // Treat those races as stale lifecycle work instead of crashing the UI.
      return
    }

    if (!canSendResizeRef.current) return
    if (!isOpenRef.current) return
    const last = lastSentResizeRef.current
    if (last?.cols === dimensions.cols && last.rows === dimensions.rows) return
    lastSentResizeRef.current = dimensions
    terminalSession.sendResize(dimensions.cols, dimensions.rows)
  }, [isScrolledToBottom, logTerminal, reloadHistoryProjectionWhenIdle, terminalSession.sendResize])

  const scrollToBottomIfCurrent = useCallback((term: XTerm) => {
    if (terminalDisposedRef.current || xtermRef.current !== term) return
    try {
      term.scrollToBottom()
    } catch (error) {
      if (!terminalDisposedRef.current) throw error
    }
  }, [])

  const cancelBottomAnchor = useCallback(() => {
    if (bottomAnchorFrameRef.current !== null && typeof window !== 'undefined') {
      window.cancelAnimationFrame(bottomAnchorFrameRef.current)
      bottomAnchorFrameRef.current = null
    }
  }, [])

  const keepBottomAnchored = useCallback(() => {
    if (terminalDisposedRef.current) return
    const term = xtermRef.current
    if (!term) return
    scrollToBottomIfCurrent(term)
    if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') return
    if (bottomAnchorFrameRef.current !== null) {
      window.cancelAnimationFrame(bottomAnchorFrameRef.current)
      bottomAnchorFrameRef.current = null
    }
    const generation = terminalGenerationRef.current
    const scrollAfterPaint = (remainingFrames: number) => {
      bottomAnchorFrameRef.current = window.requestAnimationFrame(() => {
        if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) {
          bottomAnchorFrameRef.current = null
          return
        }
        const current = xtermRef.current
        if (current) scrollToBottomIfCurrent(current)
        if (remainingFrames > 1) {
          scrollAfterPaint(remainingFrames - 1)
        } else {
          bottomAnchorFrameRef.current = null
        }
      })
    }
    scrollAfterPaint(2)
  }, [scrollToBottomIfCurrent])

  const resumeFrozenHistoryAtBottom = useCallback((includePrimed = false) => {
    const term = xtermRef.current
    if (!term || terminalDisposedRef.current) return false
    if (selectionModeRef.current) return false
    if (historySearchBottomResumeSuppressedRef.current && !includePrimed) return false
    const busy = historyLoadingRef.current ||
      historyApplyingRef.current ||
      pendingHistoryApplyRef.current !== null ||
      historyRequestAwaitingApply(historyLoadedRowsRequestedRef.current, historyLoadedRowsAppliedRef.current) ||
      pullingHistoryRef.current ||
      historyProjectionReloadPendingRef.current
    if (!historyViewportControllerRef.current.resumeAtBottom(isScrolledToBottom(term), busy, includePrimed)) return false

    primedHistoryFrameRef.current?.remove()
    primedHistoryFrameRef.current = null
    clearLiveOutputWatchdog()
    liveOutputInFlightRef.current = false
    liveOutputGenerationRef.current += 1
    pendingLiveOutputRef.current = ''
    liveOutputDroppedCharsRef.current = 0
    liveOutputSyncLostRef.current = false
    historyLoadArmedByUserRef.current = false
    historyLoadedRowsAppliedRef.current = 0
    historyLoadedRowsRequestedRef.current = 0
    historyHasMoreRef.current = true
    historySearchBottomResumeSuppressedRef.current = false
    historyRequestColsRef.current = 0
    pendingHistoryViewportRef.current = null
    pullingHistoryRef.current = false
    selectionModeExitPendingRef.current = false
    const liveText = resumeLiveScrollbackRef.current()
    latestTerminalTextRef.current = liveText
    resetTransientViewportOffsetRef.current()
    const screen = containerRef.current?.querySelector('.xterm-screen') as HTMLElement | null
    const heldFrame = containerRef.current && screen ? holdTerminalFrame(containerRef.current, screen) : null
    try {
      term.reset()
      const liveRevision = latestLiveScreenRevisionRef.current
      if (liveRevision !== undefined) {
        lastLiveScreenSubmittedRevisionRef.current = liveRevision
        markLiveScreenSubmittedRef.current(liveRevision)
      }
      writeToXterm(term, liveText, 'history_resume_live', () => {
        if (liveRevision !== undefined) markLiveScreenCompletedRef.current(liveRevision)
        markSurfaceReady()
        keepBottomAnchored()
        heldFrame?.releaseAfterPaint()
      })
      lastWrittenTextRef.current = liveText
      logTerminal('history_resume_live', {
        details: {
          chars: liveText.length,
          cols: term.cols,
          rows: term.rows,
        },
      })
      return true
    } catch (error) {
      heldFrame?.remove()
      throw error
    }
  }, [clearLiveOutputWatchdog, isScrolledToBottom, keepBottomAnchored, logTerminal, markSurfaceReady, writeToXterm])
  resumeFrozenHistoryAtBottomRef.current = resumeFrozenHistoryAtBottom

  const proposedTerminalSize = useCallback(() => {
    if (terminalDisposedRef.current) return undefined
    const term = xtermRef.current
    const fitAddon = fitAddonRef.current
    if (!term || !fitAddon) return undefined
    try {
      const dimensions = trustedTerminalDimensions(fitAddon.proposeDimensions())
      if (!dimensions || terminalDisposedRef.current || xtermRef.current !== term) return undefined
      term.resize(dimensions.cols, dimensions.rows)
      return dimensions
    } catch {
      return undefined
    }
  }, [])

  const currentTerminalSize = useCallback(() => {
    return proposedTerminalSize() ?? trustedTerminalDimensions({
      cols: xtermRef.current?.cols ?? 0,
      rows: xtermRef.current?.rows ?? 0,
    })
  }, [proposedTerminalSize])

  const requestResizeOwnerAtLocalSize = useCallback(async () => {
    let size = proposedTerminalSize()
    if (!size) {
      await waitForTerminalLayout()
      size = proposedTerminalSize()
    }
    if (!size) throw new Error('terminal viewport size is unavailable')
    return await terminalSession.requestResizeOwner(size)
  }, [proposedTerminalSize, terminalSession.requestResizeOwner])

  const sendInputAtCurrentSize = useCallback((data: string): boolean => {
    return terminalSession.sendInput(data, currentTerminalSize())
  }, [currentTerminalSize, terminalSession.sendInput])

  const sendUserInput = useCallback((data: string): boolean => {
    if (historyOnly) return false
    const delegateInput = onInputRef.current
    if (delegateInput) {
      return delegateInput(data)
    }
    // Mouse reports are already tied to the current xterm grid. ResizeObserver
    // owns size synchronization; attaching a resize RPC to every wheel report
    // serializes scrolling behind two remote acknowledgements per event.
    if (describeTerminalMouseInput(data)) return terminalSession.sendInput(data)
    return sendInputAtCurrentSize(data)
  }, [historyOnly, sendInputAtCurrentSize, terminalSession.sendInput])

  const scheduleFit = useCallback(() => {
    if (terminalDisposedRef.current) return
    const generation = terminalGenerationRef.current
    if (typeof window === 'undefined' || typeof window.requestAnimationFrame !== 'function') {
      fitAndMaybeSendResize(true)
      return
    }
    if (fitFrameRef.current !== null) {
      window.cancelAnimationFrame(fitFrameRef.current)
      fitFrameRef.current = null
    }
    fitFrameRef.current = window.requestAnimationFrame(() => {
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) {
        fitFrameRef.current = null
        return
      }
      fitFrameRef.current = window.requestAnimationFrame(() => {
        fitFrameRef.current = null
        if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
        fitAndMaybeSendResize(true)
      })
    })
  }, [fitAndMaybeSendResize])

  useImperativeHandle(ref, () => ({
    sendInput: sendInputAtCurrentSize,
    sendResize: terminalSession.sendResize,
    requestResizeOwner: requestResizeOwnerAtLocalSize,
    releaseResizeOwner: terminalSession.releaseResizeOwner,
    setResizeLock: terminalSession.setResizeLock,
    reattach: terminalSession.reattach,
    focus: () => {
      if (terminalDisposedRef.current || preventFocusRef.current) return
      try {
        xtermRef.current?.focus()
      } catch {
        // Ignore stale focus calls after xterm has been disposed.
      }
    },
    blur: () => {
      if (terminalDisposedRef.current) return
      try {
        xtermRef.current?.blur()
      } catch {
        // Ignore stale blur calls after xterm has been disposed.
      }
      containerRef.current?.querySelector('textarea')?.blur()
    },
    fit: () => {
      fitAndMaybeSendResize()
      scheduleFit()
    },
    openHistorySearch: () => {
      if (terminalDisposedRef.current) return
      setHistorySearchMessage('')
      setHistorySearchOpen(true)
    },
    pasteText: (text: string) => {
      const isMultiline = text.includes('\n') || text.includes('\r')
      return sendInputAtCurrentSize(isMultiline ? `\x1b[200~${text}\x1b[201~` : text)
    },
    selectAll: () => {
      if (terminalDisposedRef.current) return
      xtermRef.current?.selectAll()
    },
    selectVisible: () => {
      if (terminalDisposedRef.current) return
      const term = xtermRef.current
      if (!term) return
      const startRow = term.buffer.active.viewportY
      term.select(0, startRow, term.cols * term.rows)
    },
    getSelection: () => terminalDisposedRef.current ? '' : xtermRef.current?.getSelection() ?? '',
    getSelectionForClipboard: async () => {
      if (terminalDisposedRef.current) return ''
      const term = xtermRef.current
      if (!term) return ''
      const localSelection = term.getSelection()
      const history = historyMetadataRef.current
      const selection = term.getSelectionPosition()
      if (!localSelection || !history || !selection) return localSelection
      const startRow = selection.start.y
      const endRow = selection.end.y
      const startLineId = history.rowLogicalLineIds?.[startRow]
      const endLineId = history.rowLogicalLineIds?.[endRow]
      const startLogicalCol = history.rowLogicalStartCols?.[startRow]
      const endLogicalCol = history.rowLogicalStartCols?.[endRow]
      if (!startLineId || !endLineId || startLogicalCol === undefined || endLogicalCol === undefined) return localSelection
      return terminalSession.copyScrollback({
        startLineId,
        startCol: startLogicalCol + Math.max(0, selection.start.x),
        endLineId,
        endCol: endLogicalCol + Math.max(0, selection.end.x),
      }, term.cols)
    },
    hasSelection: () => terminalDisposedRef.current ? false : xtermRef.current?.hasSelection() ?? false,
    clearSelection: () => {
      if (terminalDisposedRef.current) return
      xtermRef.current?.clearSelection()
      for (const resetSelection of selectionResetHandlersRef.current) resetSelection()
    },
    getCursorInfo: () => {
      if (terminalDisposedRef.current) return null
      const term = xtermRef.current
      if (!term) return null
      const lineHeight = Math.ceil((term.element?.clientHeight ?? 0) / term.rows) || 20
      return {
        cursorY: term.buffer.active.cursorY,
        rows: term.rows,
        lineHeight,
      }
    },
    adjustInputPosition: (bottomOffset: number) => {
      if (terminalDisposedRef.current) return
      const element = xtermRef.current?.element
      if (!element) return
      const compositionOverlay = containerRef.current?.querySelector('.comp-overlay') as HTMLElement | null
      if (bottomOffset > 0) {
        element.classList.add('input-adjusted')
        element.style.setProperty('--kb-input-bottom', `${bottomOffset}px`)
        if (compositionOverlay) compositionOverlay.style.bottom = `${bottomOffset}px`
      } else {
        element.classList.remove('input-adjusted')
        element.style.removeProperty('--kb-input-bottom')
        if (compositionOverlay) compositionOverlay.style.bottom = ''
      }
    },
    getBufferType: () => terminalDisposedRef.current ? 'normal' : xtermRef.current?.buffer.active.type ?? 'normal',
    updateOptions: (opts) => {
      if (terminalDisposedRef.current || !xtermRef.current) return
      if (opts.fontSize !== undefined) xtermRef.current.options.fontSize = opts.fontSize
      if (opts.cursorBlink !== undefined) xtermRef.current.options.cursorBlink = opts.cursorBlink
      if (opts.fontFamily !== undefined) xtermRef.current.options.fontFamily = opts.fontFamily
      if (opts.scrollback !== undefined) xtermRef.current.options.scrollback = opts.scrollback
      settingsRef.current = {
        ...settingsRef.current,
        ...(opts.fontSize !== undefined ? { fontSize: opts.fontSize } : {}),
        ...(opts.cursorBlink !== undefined ? { cursorBlink: opts.cursorBlink } : {}),
        ...(opts.fontFamily !== undefined ? { fontFamily: opts.fontFamily } : {}),
        ...(opts.scrollback !== undefined ? { scrollback: opts.scrollback } : {}),
      }
      fitAndMaybeSendResize()
    },
  }), [fitAndMaybeSendResize, requestResizeOwnerAtLocalSize, scheduleFit, sendInputAtCurrentSize, terminalSession.copyScrollback, terminalSession.reattach, terminalSession.releaseResizeOwner, terminalSession.sendResize, terminalSession.setResizeLock])

  useEffect(() => {
    modifierStateRef.current = modifierState
  }, [modifierState])

  useEffect(() => {
    const term = xtermRef.current
    if (!term) return
    term.options.fontSize = settings.fontSize
    term.options.fontFamily = settings.fontFamily
    term.options.cursorBlink = settings.cursorBlink
    term.options.scrollback = settings.scrollback
    term.options.theme = resolveTerminalTheme(settings.themeId)
    fitAndMaybeSendResize()
    const fonts = typeof document === 'undefined' ? undefined : document.fonts
    if (!fonts) return
    let cancelled = false
    void fonts.load(`${settings.fontSize}px ${settings.fontFamily}`, 'AaBb0123\uE0B0\uF07C\uF120').then(() => {
      if (cancelled || terminalDisposedRef.current || xtermRef.current !== term) return
      term.options.fontFamily = settings.fontFamily
      fitAndMaybeSendResize()
    }).catch(() => undefined)
    return () => { cancelled = true }
  }, [fitAndMaybeSendResize, settings])

  useEffect(() => {
    onModifierStateChangeRef.current = onModifierStateChange
  }, [onModifierStateChange])

  useEffect(() => {
    onInputRef.current = onInput
  }, [onInput])

  useEffect(() => {
    onCursorMoveRef.current = onCursorMove
  }, [onCursorMove])

  useEffect(() => {
    onBufferChangeRef.current = onBufferChange
  }, [onBufferChange])

  useEffect(() => {
    onResizeControlRef.current = onResizeControl
  }, [onResizeControl])

  useEffect(() => {
    const wasSelectionMode = selectionModeRef.current
    selectionModeRef.current = selectionMode
    if (selectionMode && !wasSelectionMode) {
      enterSelectionModeRef.current()
    } else if (!selectionMode) {
      for (const resetSelection of selectionResetHandlersRef.current) resetSelection()
      if (wasSelectionMode) exitSelectionModeRef.current()
    }
  }, [selectionMode])

  useEffect(() => {
    canSendResizeRef.current = terminalSession.resizeControl.canResize
    onResizeControlRef.current?.(terminalSession.resizeControl)
    if (terminalSession.resizeControl.canResize) {
      fitAndMaybeSendResize()
    }
  }, [fitAndMaybeSendResize, terminalSession.resizeControl])

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const generation = terminalGenerationRef.current + 1
    terminalGenerationRef.current = generation
    terminalDisposedRef.current = false
    initialViewportReadyRef.current = false
    setInitialViewportReady(false)
    setHistoryLoadFailure('none')
    if (historyApplyTimerRef.current !== null) {
      window.clearTimeout(historyApplyTimerRef.current)
      historyApplyTimerRef.current = null
    }
    pendingHistoryApplyRef.current = null
    historyApplyQueuedAtRef.current = 0
    historyLoadArmedByUserRef.current = false
    primedHistoryFrameRef.current?.remove()
    primedHistoryFrameRef.current = null
    historyViewportControllerRef.current.reset()
    historyRequestColsRef.current = 0
    historyProjectionReloadPendingRef.current = false
    lastHistoryScrollActivityAtRef.current = 0
    historyApplyingRef.current = false
    historyRevisionAppliedRef.current = 0
    historyLoadedRowsAppliedRef.current = 0
    historyLoadedRowsRequestedRef.current = 0
    historyHasMoreRef.current = true
    hideHistoryLoading(true)
    lastSnapshotTextRef.current = ''
    recoveryRevisionAppliedRef.current = 0

    const term = new XTerm(createTerminalOptions(settingsRef.current))
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(container)
    if (term.element) term.element.style.overflow = 'hidden'
    const coreWithViewport = term as XTerm & { _core?: { viewport?: { scrollBarWidth?: number } } }
    if (coreWithViewport._core?.viewport) {
      coreWithViewport._core.viewport.scrollBarWidth = 0
    }
    const isMobile = typeof window !== 'undefined' && window.innerWidth < 768
    const textarea = container.querySelector('.xterm-helper-textarea') as HTMLTextAreaElement | null
    const compositionOverlay = document.createElement('div')
    compositionOverlay.className = 'comp-overlay'
    term.element?.append(compositionOverlay)
    const handleCompositionStart = () => {
      if (!isMobile) return
      compositionOverlay.style.display = 'block'
      compositionOverlay.textContent = ''
    }
    const handleCompositionUpdate = (event: CompositionEvent) => {
      if (!isMobile) return
      compositionOverlay.textContent = event.data
    }
    const handleCompositionEnd = () => {
      compositionOverlay.style.display = 'none'
      compositionOverlay.textContent = ''
    }
    textarea?.addEventListener('compositionstart', handleCompositionStart)
    textarea?.addEventListener('compositionupdate', handleCompositionUpdate)
    textarea?.addEventListener('compositionend', handleCompositionEnd)
    const onFocus = () => { if (preventFocusRef.current && textarea) textarea.blur() }
    textarea?.addEventListener('focus', onFocus)
    let activeRenderer: TerminalRenderer | 'canvas-fallback' = 'dom'
    const loadCanvasRenderer = (reason: 'requested' | 'fallback') => {
      try {
        term.loadAddon(new CanvasAddon())
        activeRenderer = reason === 'requested' ? 'canvas' : 'canvas-fallback'
      } catch {
        // Fall back to xterm's DOM renderer when canvas is unavailable.
        activeRenderer = 'dom'
      }
    }
    const isTestDom = typeof navigator !== 'undefined' && navigator.userAgent.includes('jsdom')
    if (!isTestDom) {
      const configuredRenderer = renderer ?? settingsRef.current.renderer
      const rendererMode = effectiveRendererMode(configuredRenderer)
      if (rendererMode === 'canvas') {
        loadCanvasRenderer('requested')
      } else if (rendererMode !== 'dom') {
        // auto or webgl: try WebGL first, fall back to canvas
        try {
          const webglAddon = new WebglAddon(true)
          webglAddon.onContextLoss(() => {
            webglAddon.dispose()
            if (rendererMode !== 'webgl') loadCanvasRenderer('fallback')
            term.refresh(0, term.rows - 1)
            logTerminal('renderer_context_loss', {
              level: 'warn',
              details: {
                configuredRenderer,
                effectiveRenderer: rendererMode,
                fallbackRenderer: activeRenderer,
              },
            })
          })
          term.loadAddon(webglAddon)
          activeRenderer = 'webgl'
        } catch {
          if (rendererMode !== 'webgl') {
            loadCanvasRenderer('fallback')
          } else {
            activeRenderer = 'dom'
          }
        }
      }
      logTerminal('renderer_selected', {
        level: 'info',
        details: {
          configuredRenderer,
          effectiveRenderer: rendererMode,
          activeRenderer,
          nativeAndroidWebView: isNativeAndroidWebView(),
        },
      })
    }
    xtermRef.current = term
    setXtermReady(true)
    fitAddonRef.current = fitAddon

    // Write existing session text so the terminal isn't blank after renderer switch
    const initialText = latestTerminalTextRef.current
    if (initialText) {
      try {
        writeToXterm(term, initialText, 'initial_text', () => {
          markSurfaceReady()
          keepBottomAnchored()
        })
      } catch {}
      lastWrittenTextRef.current = initialText
    }

    const dataDisposable = term.onData((data) => {
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
      const mouseInput = describeTerminalMouseInput(data)
      if (mouseInput) {
        traceTerminalScroll('xterm.data', {
          machineId,
          terminalId,
          details: { ...mouseInput, chars: data.length },
        })
      }
      const currentModifiers = modifierStateRef.current
      if (currentModifiers && (currentModifiers.ctrl !== 'off' || currentModifiers.alt !== 'off')) {
        // Android IMEs often emit xterm data without a usable keydown event.
        // Apply modifiers here as the authoritative fallback; multi-character
        // paste/composition payloads remain unchanged by applyTerminalModifiers.
        const result = applyTerminalModifiers(data, currentModifiers)
        const accepted = sendUserInput(result.data)
        if (accepted && (result.ctrl !== currentModifiers.ctrl || result.alt !== currentModifiers.alt)) {
          const nextModifiers = { ctrl: result.ctrl, alt: result.alt }
          modifierStateRef.current = nextModifiers
          onModifierStateChangeRef.current?.(nextModifiers)
        }
        return
      }
      sendUserInput(data)
    })
    const binaryDisposable = term.onBinary((data) => {
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
      sendUserInput(data)
    })
    term.attachCustomKeyEventHandler((event) => {
      if (event.type !== 'keydown') return true
      const currentModifiers = modifierStateRef.current
      if (!currentModifiers || (currentModifiers.ctrl === 'off' && currentModifiers.alt === 'off')) return true
      if (event.isComposing || event.keyCode === 229) return true
      if (['Control', 'Alt', 'Shift', 'Meta'].includes(event.key)) return true
      if (event.key.length !== 1 || event.key.charCodeAt(0) < 0x20 || event.key.charCodeAt(0) > 0x7e) return true

      const result = applyTerminalModifiers(event.key, currentModifiers)
      if (result.data === event.key) return true
      const accepted = sendUserInput(result.data)
      if (!accepted) return true
      if (result.ctrl !== currentModifiers.ctrl || result.alt !== currentModifiers.alt) {
        const nextModifiers = { ctrl: result.ctrl, alt: result.alt }
        modifierStateRef.current = nextModifiers
        onModifierStateChangeRef.current?.(nextModifiers)
      }
      return false
    })
    const cursorDisposable = term.onCursorMove(() => {
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
      onCursorMoveRef.current?.()
    })
    const coreTerminal = term as XTerm & {
      _core?: {
        coreMouseService?: {
          areMouseEventsActive?: boolean
          triggerMouseEvent?: (event: {
            col: number
            row: number
            x: number
            y: number
            button: number
            action: number
            ctrl: boolean
            alt: boolean
            shift: boolean
          }) => boolean
        }
        coreService?: { decPrivateModes?: { applicationCursorKeys?: boolean } }
      }
    }
    let screenElement = container.querySelector('.xterm-screen') as HTMLElement | null
    const resolveScreenElement = () => {
      if (!screenElement?.isConnected) {
        screenElement = term.element?.querySelector('.xterm-screen') as HTMLElement | null
      }
      return screenElement
    }
    const historySearchHighlightLayer = document.createElement('div')
    historySearchHighlightLayer.className = 'anytty-history-search-highlights'
    historySearchHighlightLayer.setAttribute('aria-hidden', 'true')
    resolveScreenElement()?.append(historySearchHighlightLayer)
    historySearchHighlightLayerRef.current = historySearchHighlightLayer
    const getLineHeight = () => Math.ceil((term.element?.clientHeight ?? 0) / term.rows) || 20
    let lineHeightPx = getLineHeight()
    let touchAccum = 0
    let touchLastY = Number.NaN
    let touchStartX = Number.NaN
    let touchStartY = Number.NaN
    let touchStartTime = 0
    let touchMoved = false
    let touchActive = false
    let velocityY = 0
    let lastTouchTime = 0
    let touchScrollRoute: 'viewport' | 'tui' = 'viewport'
    let touchTuiEvents = 0
    let momentumFrame = 0
    let smoothActive = false
    let pendingScrollPx = 0
    let scrollFrame = 0
    const cancelScrollFrame = () => {
      if (!scrollFrame) return
      window.cancelAnimationFrame(scrollFrame)
      scrollFrame = 0
    }
    let totalPxOffset = 0
    let baseViewportY = 0
    let renderedViewportY = 0
    let scrollbarHideTimer = 0
    let scrollbarDragging = false
    let scrollbarDragStartY = 0
    let scrollbarDragStartRatio = 0
    let historyPrefetchPending = false
    let historyPullOffsetPx = 0
    let historyPullReleaseTimer = 0
    let historyPullReleasing = false
    let historyPullLoading = false
    let nudgeHistoryPullVisual = (_pixels: number) => {}
    let releaseHistoryPullVisual = () => {}
    let setHistoryPullLoading = (_loading: boolean) => {}
    let historyEntryGestureReady = false
    let selectionAnchorCol = 0
    let selectionAnchorRow = 0
    let selectionEndCol = 0
    let selectionEndRow = 0
    let selectionTouchActive = false
    let selectionModeWasActive = false
    let selectionAutoScrollId: number | null = null
    let selectionLastClientX = 0
    let selectionLastClientY = 0
    let selectionPhase: 'idle' | 'anchor-set' = 'idle'
    let selectionExtending = false

    const noteHistoryScrollActivity = () => {
      lastHistoryScrollActivityAtRef.current = Date.now()
    }

    const noteUserScrollActivity = () => {
      cancelBottomAnchor()
      historySearchBottomResumeSuppressedRef.current = false
      noteHistoryScrollActivity()
    }

    const armHistoryScrollbackLoad = () => {
      const historyViewport = historyViewportControllerRef.current
      if (!historyViewport.isLiveUpdateDeferred) {
        historyViewport.prime()
        freezeScrollbackRef.current()
        clearLiveOutputWatchdog()
        liveOutputInFlightRef.current = false
        liveOutputGenerationRef.current += 1
        pendingLiveOutputRef.current = ''
        liveOutputDroppedCharsRef.current = 0
        liveOutputSyncLostRef.current = false
      } else if (
        historyEntryGestureReady &&
        historyViewport.isPrimed &&
        historyLoadedRowsAppliedRef.current > 0 &&
        !historyLoadingRef.current &&
        !historyApplyingRef.current &&
        pendingHistoryApplyRef.current === null
      ) {
        historyViewport.enterHistory()
        historyEntryGestureReady = false
        logTerminal('history_enter', {
          details: { loadedRows: historyLoadedRowsAppliedRef.current },
        })
      }
      historyLoadArmedByUserRef.current = true
      if (snapshotAlternateScreenRef.current && term.buffer.active.type === 'alternate') {
        term.reset()
      }
      noteUserScrollActivity()
    }

    const clearMomentum = () => {
      cancelScrollFrame()
      if (!momentumFrame) return
      window.cancelAnimationFrame(momentumFrame)
      momentumFrame = 0
    }

    const scrollbarTrack = document.createElement('div')
    scrollbarTrack.className = 'term-scrollbar-track'
    const scrollbarThumb = document.createElement('div')
    scrollbarThumb.className = 'term-scrollbar-thumb'
    scrollbarTrack.append(scrollbarThumb)
    container.append(scrollbarTrack)

    const historyPullIndicator = document.createElement('div')
    historyPullIndicator.className = 'term-history-pull-indicator'
    historyPullIndicator.setAttribute('aria-hidden', 'true')
    container.append(historyPullIndicator)

    const magnifier = document.createElement('div')
    magnifier.className = 'sel-magnifier'
    const magnifierCanvas = document.createElement('canvas')
    const magnifierWidth = 160
    const magnifierHeight = 60
    const pixelRatio = typeof window === 'undefined' ? 1 : window.devicePixelRatio || 1
    magnifierCanvas.width = magnifierWidth * pixelRatio
    magnifierCanvas.height = magnifierHeight * pixelRatio
    magnifierCanvas.style.width = `${magnifierWidth}px`
    magnifierCanvas.style.height = `${magnifierHeight}px`
    magnifierCanvas.style.display = 'block'
    magnifier.append(magnifierCanvas)
    container.append(magnifier)

    const selectionAnchorMarker = document.createElement('div')
    selectionAnchorMarker.className = 'sel-anchor-marker'
    container.append(selectionAnchorMarker)

    const hideSelectionAnchor = () => {
      selectionAnchorMarker.style.display = 'none'
      selectionAnchorMarker.style.opacity = '0'
      selectionAnchorMarker.style.left = '-9999px'
      selectionAnchorMarker.style.top = '-9999px'
      selectionAnchorMarker.style.height = '0px'
    }
    hideSelectionAnchor()

    const showScrollbar = () => {
      scrollbarTrack.classList.add('visible')
      window.clearTimeout(scrollbarHideTimer)
    }

    const hideScrollbarDelayed = () => {
      window.clearTimeout(scrollbarHideTimer)
      if (scrollbarDragging) return
      scrollbarHideTimer = window.setTimeout(() => {
        scrollbarTrack.classList.remove('visible')
      }, 1200)
    }

    const updateScrollbar = () => {
      const buffer = term.buffer.active
      const totalLines = buffer.length
      const viewportRows = term.rows
      if (totalLines <= viewportRows) {
        scrollbarTrack.classList.remove('visible')
        return
      }
      const maxScroll = totalLines - viewportRows
      const ratio = buffer.viewportY / maxScroll
      const thumbRatio = Math.max(0.06, viewportRows / totalLines)
      const trackHeight = scrollbarTrack.clientHeight
      const thumbHeight = Math.max(24, trackHeight * thumbRatio)
      const thumbTop = ratio * (trackHeight - thumbHeight)
      scrollbarThumb.style.height = `${thumbHeight}px`
      scrollbarThumb.style.transform = `translateY(${thumbTop}px)`
    }

    const updateHistoryStatus = () => {
      const element = historyStatusRef.current
      const text = historyStatusTextRef.current
      if (!element || !text) return
      const history = historyMetadataRef.current
      if (!history || history.loadedRows <= 0) {
        text.textContent = ''
        setHistoryStatusVisible(false)
        return
      }
      const total = Math.max(1, history.logicalTotalRows ?? history.committedTotalRows ?? 1)
      const visibleBottom = Math.max(1, Math.min(history.loadedRows, term.buffer.active.viewportY + term.rows))
      const logicalLineId = Number(history.rowLogicalLineIds?.[visibleBottom - 1])
      const current = Number.isSafeInteger(logicalLineId) && logicalLineId > 0
        ? Math.min(total, logicalLineId)
        : total
      const timestamp = history.rowTimestampsUnixMs?.[visibleBottom - 1]
      text.textContent = formatHistoryStatusRef.current(current, total, timestamp)
      setHistoryStatusVisible(true)
    }
    updateHistoryStatusRef.current = updateHistoryStatus

    const historyLoadedRowsLimit = () => terminalHistoryLoadedRowsLimit(settingsRef.current)

    const scrollbackLoadBlockedReason = () => {
      if (historyLoadingRef.current) return 'loading'
      if (historyApplyingRef.current) return 'applying'
      if (pendingHistoryApplyRef.current) return 'pending_apply'
      if (historyRequestAwaitingApply(historyLoadedRowsRequestedRef.current, historyLoadedRowsAppliedRef.current)) return 'awaiting_apply'
      if (!historyHasMoreRef.current) return 'no_more'
      if (historyLoadedRowsRequestedRef.current >= historyLoadedRowsLimit()) return 'capacity_limit'
      if (!hasTerminalSnapshotRef.current && !snapshotAlternateScreenRef.current && term.buffer.active.type !== 'alternate') return 'no_snapshot'
      if (term.buffer.active.type !== 'normal' && !snapshotAlternateScreenRef.current) return 'alternate_buffer'
      return null
    }

    const loadScrollbackPage = async (restoreViewport: boolean): Promise<TerminalScrollbackLoadResult> => {
      const alternate = snapshotAlternateScreenRef.current || term.buffer.active.type === 'alternate'
      const emptyResult = () => ({
        loadedRows: 0,
        totalRows: historyLoadedRowsRequestedRef.current,
        hasMore: historyHasMoreRef.current,
        alternate,
      })
      if (alternate && term.buffer.active.type === 'alternate') {
        term.reset()
      }
      const blockedReason = scrollbackLoadBlockedReason()
      if (blockedReason) {
        const now = terminalNow()
        const lastSkipLog = lastHistoryLoadSkipLogRef.current
        if (!lastSkipLog || lastSkipLog.reason !== blockedReason || now - lastSkipLog.at >= historyLoadSkipLogIntervalMs) {
          lastHistoryLoadSkipLogRef.current = { reason: blockedReason, at: now }
          logTerminal('history_load_skip', {
            level: 'debug',
            details: {
              reason: blockedReason,
              loadedRows: historyLoadedRowsRequestedRef.current,
              hasMore: historyHasMoreRef.current,
              rowsLimit: historyLoadedRowsLimit(),
              xtermScrollback: term.options.scrollback,
            },
          })
        }
        return emptyResult()
      }
      lastHistoryLoadSkipLogRef.current = null
      if (historyRequestColsRef.current > 0 && historyRequestColsRef.current !== term.cols) {
        historyLoadedRowsRequestedRef.current = 0
        historyLoadedRowsAppliedRef.current = 0
      }
      historyRequestColsRef.current = term.cols
      const remainingRows = Math.max(0, historyLoadedRowsLimit() - historyLoadedRowsRequestedRef.current)
      const requestRows = Math.min(historyScrollbackPageRows, remainingRows)
      if (requestRows <= 0) return emptyResult()
      historyLoadingRef.current = true
      showHistoryLoading()
      if (restoreViewport) {
        pullingHistoryRef.current = true
        setHistoryPullLoading(true)
      }
      historyRestoreViewportOnLoadRef.current = restoreViewport
      pendingHistoryViewportRef.current = restoreViewport ? term.buffer.active.viewportY : null
      let keepVisibleForApply = false
      let loadFailed = false
      let loadCancelled = false
      try {
        const result = await loadScrollbackRef.current(requestRows, alternate, term.cols)
        setHistoryLoadFailure('none')
        historyLoadedRowsRequestedRef.current = result.operation === 'replace'
          ? result.totalRows
          : Math.max(historyLoadedRowsRequestedRef.current, result.totalRows)
        historyHasMoreRef.current = result.hasMore
        if (result.hasMore && historyLoadedRowsRequestedRef.current >= historyLoadedRowsLimit()) {
          logTerminal('history_load_capacity_limit', {
            level: 'warn',
            details: {
              loadedRows: historyLoadedRowsRequestedRef.current,
              rowsLimit: historyLoadedRowsLimit(),
              xtermScrollback: term.options.scrollback,
            },
          })
        }
        if (result.loadedRows <= 0) {
          pendingHistoryViewportRef.current = null
        } else {
          keepVisibleForApply = true
        }
        return result
      } catch (error) {
        if (typeof error === 'object' && error !== null && 'name' in error && error.name === 'AbortError') {
          loadCancelled = true
          pendingHistoryViewportRef.current = null
          pullingHistoryRef.current = false
          return emptyResult()
        }
        loadFailed = true
        const historyError = error as Error & { code?: string, retryable?: boolean }
        const lineTooLarge = historyError instanceof Error
          && historyError.code === 'resource_exhausted'
          && historyError.retryable === false
        pendingHistoryViewportRef.current = null
        pullingHistoryRef.current = false
        logTerminal('history_load_failed', {
          level: 'warn',
          details: {
            reason: error instanceof Error ? error.message : String(error),
            loadedRows: historyLoadedRowsRequestedRef.current,
          },
        })
        historyHasMoreRef.current = false
        historyLoadArmedByUserRef.current = false
        setHistoryLoadFailure(lineTooLarge ? 'line-too-large' : 'reloadable')
        return emptyResult()
      } finally {
        historyLoadingRef.current = false
        historyRestoreViewportOnLoadRef.current = false
        if (!keepVisibleForApply) {
          pullingHistoryRef.current = false
          setHistoryPullLoading(false)
          hideHistoryLoading()
          if (!loadCancelled || selectionModeExitPendingRef.current) {
            resumeFrozenHistoryAtBottomRef.current(true)
          }
          if (loadFailed) {
            historyHasMoreRef.current = false
            historyLoadArmedByUserRef.current = false
          }
        }
        reloadHistoryProjectionWhenIdle()
      }
    }

    retryHistoryLoadRef.current = () => {
      resetScrollbackRef.current()
      historyLoadedRowsRequestedRef.current = 0
      historyLoadedRowsAppliedRef.current = 0
      historyHasMoreRef.current = true
      historyLoadArmedByUserRef.current = true
      setHistoryLoadFailure('none')
      void loadScrollbackPage(true)
    }

    const maybePrefetchScrollback = () => {
      if (historyPrefetchPending || !historyLoadArmedByUserRef.current) return
      const thresholdRows = terminalHistoryPrefetchThresholdRows(
        settingsRef.current.scrollbackPrefetchThresholdRows,
        term.rows,
      )
      if (term.buffer.active.viewportY > thresholdRows) return
      historyPrefetchPending = true
      void loadScrollbackPage(true).finally(() => {
        historyPrefetchPending = false
      })
    }
    maybePrefetchScrollbackRef.current = maybePrefetchScrollback

    const shouldDelayHistoryApply = (pending: PendingHistoryApply) => {
      if (pending.restoreViewportY === null) return false
      if (momentumFrame || smoothActive || scrollbarDragging) return true
      const elapsed = Date.now() - historyApplyQueuedAtRef.current
      if (elapsed >= historyApplyMaxDelayMs) return false
      return Date.now() - lastHistoryScrollActivityAtRef.current < historyApplyScrollIdleMs
    }

    const applyPendingHistory = () => {
      historyApplyTimerRef.current = null
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
      const pending = pendingHistoryApplyRef.current
      if (!pending) return
      if (pending.cols !== term.cols) {
        pendingHistoryApplyRef.current = null
        historyApplyQueuedAtRef.current = 0
        hideHistoryLoading()
        historyProjectionReloadPendingRef.current = true
        reloadHistoryProjectionWhenIdle()
        return
      }
      if (shouldDelayHistoryApply(pending)) {
        historyApplyTimerRef.current = window.setTimeout(applyPendingHistory, historyApplyScrollIdleMs)
        return
      }
      pendingHistoryApplyRef.current = null
      historyApplyQueuedAtRef.current = 0
      let heldFrame: TerminalFrameHold | null = null
      let historyApplyReleased = false
      let historyApplyWatchdog: number | null = null
      const abandonHistoryApply = () => {
        if (historyApplyReleased) return
        historyApplyReleased = true
        if (historyApplyWatchdog !== null) {
          window.clearTimeout(historyApplyWatchdog)
          historyApplyWatchdog = null
        }
        heldFrame?.remove()
        historyApplyingRef.current = false
        pullingHistoryRef.current = false
        setHistoryPullLoading(false)
        historyProjectionReloadPendingRef.current = false
        historyLoadedRowsRequestedRef.current = historyLoadedRowsAppliedRef.current
        hideHistoryLoading()
        resumeFrozenHistoryAtBottomRef.current(true)
      }
      try {
        historyApplyingRef.current = true
        historyApplyWatchdog = window.setTimeout(abandonHistoryApply, historyApplyWatchdogMs)
        const finishHistoryApply = () => {
          if (historyApplyReleased) return
          historyApplyReleased = true
          if (historyApplyWatchdog !== null) {
            window.clearTimeout(historyApplyWatchdog)
            historyApplyWatchdog = null
          }
          historyLoadedRowsAppliedRef.current = pending.loadedRows
          historyApplyingRef.current = false
          setHistoryPullLoading(false)
          hideHistoryLoading()
          if (pending.restoreViewportY !== null) {
            maybePrefetchScrollbackRef.current()
          }
          reloadHistoryProjectionWhenIdle()
          resumeFrozenHistoryAtBottomRef.current(selectionModeExitPendingRef.current)
        }
        const previouslyAppliedRows = historyLoadedRowsAppliedRef.current
        const rowsAddedSinceLastApply = Math.max(0, pending.loadedRows - previouslyAppliedRows)
        const currentViewportY = term.buffer.active.viewportY
        const shouldKeepBottom = pending.restoreViewportY === null &&
          !pullingHistoryRef.current &&
          isScrolledToBottom(term)
        if (typeof term.options.scrollback === 'number' && rowsAddedSinceLastApply > 0) {
          term.options.scrollback = Math.max(
            term.options.scrollback,
            Math.min(pending.loadedRows, historyLoadedRowsLimit()),
          )
        }
        const latestText = latestTerminalTextRef.current
        const historyText = latestText.startsWith(pending.text) ? latestText : pending.text
        const textToApply = historyReplayWithViewportTail(
          historyText,
          pending.loadedRows,
          pending.viewportTop,
          term.rows,
        )
        const selectionBeforeApply = selectionModeRef.current && term.hasSelection()
          ? term.getSelectionPosition()
          : undefined
        const bufferLengthBeforeApply = term.buffer.active.length
        const currentScreenElement = resolveScreenElement()
        if (currentScreenElement) heldFrame = holdTerminalFrame(container, currentScreenElement)
        clearHistorySearchHighlight()
        term.reset()
        writeToXterm(term, textToApply, 'history_apply', () => {
          markSurfaceReady()
          const bufferLengthAfterApply = term.buffer.active.length
          const actualPrependedRows = Math.max(0, bufferLengthAfterApply - bufferLengthBeforeApply)
          const viewportOffsetRows = actualPrependedRows > 0 ? actualPrependedRows : pending.prependedRows
          if (pending.absoluteViewportY !== undefined || pending.restoreViewportY !== null || pullingHistoryRef.current) {
            cancelBottomAnchor()
            term.scrollToLine(pending.absoluteViewportY ?? historyViewportAfterApply({
              operation: pending.operation,
              previouslyAppliedRows,
              restoreViewportY: pending.restoreViewportY ?? currentViewportY,
              actualPrependedRows,
              fallbackPrependedRows: pending.prependedRows,
              bufferLength: bufferLengthAfterApply,
              viewportRows: term.rows,
              initialViewportTop: pending.viewportTop,
            }))
          } else if (shouldKeepBottom) {
            keepBottomAnchored()
          } else {
            cancelBottomAnchor()
            term.scrollToLine(currentViewportY + viewportOffsetRows)
          }
          paintHistorySearchHighlights(term, pending.searchMatchRanges)
          if (selectionModeRef.current) {
            if (selectionPhase !== 'idle' && viewportOffsetRows > 0) {
              selectionAnchorRow += viewportOffsetRows
              selectionEndRow += viewportOffsetRows
            }
            if (selectionBeforeApply) {
              const length = Math.max(
                0,
                (selectionBeforeApply.end.y - selectionBeforeApply.start.y) * term.cols +
                  selectionBeforeApply.end.x - selectionBeforeApply.start.x,
              )
              term.select(
                selectionBeforeApply.start.x,
                selectionBeforeApply.start.y + viewportOffsetRows,
                length,
              )
            }
          }
          if (pending.operation === 'replace' && previouslyAppliedRows === 0 && !historyOnly) {
            primedHistoryFrameRef.current?.remove()
            primedHistoryFrameRef.current = heldFrame
            heldFrame = null
            logTerminal('history_first_page_staged', {
              details: {
                loadedRows: pending.loadedRows,
                viewportY: term.buffer.active.viewportY,
                bufferLength: term.buffer.active.length,
                viewportRows: term.rows,
                atBottom: isScrolledToBottom(term),
              },
            })
          }
          pullingHistoryRef.current = false
          heldFrame?.releaseAfterPaint()
          if (selectionModeExitPendingRef.current) term.scrollToBottom()
          finishHistoryApply()
        })
        lastWrittenTextRef.current = textToApply
      } catch (error) {
        abandonHistoryApply()
        if (!terminalDisposedRef.current) throw error
      }
    }

    const scheduleHistoryApply = (immediate = false) => {
      if (historyApplyQueuedAtRef.current === 0) {
        historyApplyQueuedAtRef.current = Date.now()
      }
      if (historyApplyTimerRef.current !== null) {
        if (!immediate) return
        window.clearTimeout(historyApplyTimerRef.current)
      }
      historyApplyTimerRef.current = window.setTimeout(
        applyPendingHistory,
        immediate ? 0 : historyApplyBatchDelayMs,
      )
    }
    scheduleHistoryApplyRef.current = scheduleHistoryApply

    const isMouseModeActive = () => {
      try {
        return term.element?.classList.contains('enable-mouse-events') === true ||
          Boolean(coreTerminal._core?.coreMouseService?.areMouseEventsActive)
      } catch {
        return false
      }
    }
    const isAlternateScreen = () => snapshotAlternateScreenRef.current || term.buffer.active.type === 'alternate'
    const isApplicationCursor = () => {
      try {
        return Boolean(coreTerminal._core?.coreService?.decPrivateModes?.applicationCursorKeys)
      } catch {
        return false
      }
    }

    const syncTransform = () => {
      const currentScreenElement = resolveScreenElement()
      if (!currentScreenElement) return
      const renderedPx = smoothActive ? (renderedViewportY - baseViewportY) * lineHeightPx : 0
      const subLinePx = smoothActive ? totalPxOffset - renderedPx : 0
      const translateY = historyPullOffsetPx - subLinePx
      if (!smoothActive && !historyPullReleasing && Math.abs(translateY) < 0.01) {
        currentScreenElement.style.transform = ''
        primedHistoryFrameRef.current?.setTransform('')
        return
      }
      const transform = `translateY(${translateY}px)`
      currentScreenElement.style.transform = transform
      primedHistoryFrameRef.current?.setTransform(transform)
    }

    const updateHistoryPullIndicator = () => {
      const progress = Math.min(1, historyPullOffsetPx / historyPullMaxOffsetPx)
      historyPullIndicator.style.setProperty('--term-history-pull-scale', `${0.25 + progress * 0.75}`)
      historyPullIndicator.style.setProperty('--term-history-pull-opacity', `${0.35 + progress * 0.65}`)
      historyPullIndicator.classList.toggle('visible', progress > 0 || historyPullLoading)
      historyPullIndicator.classList.toggle('loading', historyPullLoading)
    }

    const clearHistoryPullReleaseTimer = () => {
      if (!historyPullReleaseTimer) return
      window.clearTimeout(historyPullReleaseTimer)
      historyPullReleaseTimer = 0
    }

    releaseHistoryPullVisual = () => {
      clearHistoryPullReleaseTimer()
      if (historyPullOffsetPx <= 0) {
        updateHistoryPullIndicator()
        return
      }
      historyPullReleasing = true
      const currentScreenElement = resolveScreenElement()
      if (currentScreenElement) {
        currentScreenElement.style.transition = prefersReducedTerminalMotion()
          ? ''
          : `transform ${historyPullReleaseMs}ms cubic-bezier(0.22, 1.35, 0.36, 1)`
      }
      historyPullOffsetPx = 0
      syncTransform()
      updateHistoryPullIndicator()
      historyPullReleaseTimer = window.setTimeout(() => {
        historyPullReleaseTimer = 0
        historyPullReleasing = false
        const screen = resolveScreenElement()
        if (screen) {
          screen.style.transition = ''
          if (!smoothActive) {
            screen.style.transform = ''
            screen.style.willChange = ''
          }
        }
        if (!smoothActive) primedHistoryFrameRef.current?.setTransform('')
        updateHistoryPullIndicator()
      }, prefersReducedTerminalMotion() ? 0 : historyPullReleaseMs)
    }

    nudgeHistoryPullVisual = (pixels: number) => {
      if (pixels <= 0) return
      clearHistoryPullReleaseTimer()
      historyPullReleasing = false
      const currentScreenElement = resolveScreenElement()
      if (currentScreenElement) {
        currentScreenElement.style.transition = ''
        currentScreenElement.style.willChange = 'transform'
      }
      if (!prefersReducedTerminalMotion()) {
        const remaining = Math.max(0, historyPullMaxOffsetPx - historyPullOffsetPx)
        const resisted = Math.max(1.5, pixels * 0.24) * (remaining / historyPullMaxOffsetPx)
        historyPullOffsetPx = Math.min(historyPullMaxOffsetPx, historyPullOffsetPx + resisted)
        syncTransform()
      }
      updateHistoryPullIndicator()
      if (!touchActive) {
        historyPullReleaseTimer = window.setTimeout(releaseHistoryPullVisual, 70)
      }
    }

    setHistoryPullLoading = (loading: boolean) => {
      historyPullLoading = loading
      updateHistoryPullIndicator()
    }

    const smoothBegin = () => {
      if (smoothActive) return
      smoothActive = true
      lineHeightPx = getLineHeight()
      baseViewportY = term.buffer.active.viewportY
      renderedViewportY = baseViewportY
      totalPxOffset = 0
      const currentScreenElement = resolveScreenElement()
      if (currentScreenElement) currentScreenElement.style.willChange = 'transform'
    }

    const smoothEnd = () => {
      if (!smoothActive) return
      smoothActive = false
      const currentScreenElement = resolveScreenElement()
      if (currentScreenElement) {
        if (historyPullOffsetPx > 0 || historyPullReleasing) {
          syncTransform()
        } else {
          currentScreenElement.style.transform = ''
          currentScreenElement.style.willChange = ''
        }
      }
      if (historyPullOffsetPx <= 0 && !historyPullReleasing) {
        primedHistoryFrameRef.current?.setTransform('')
      }
    }

    const resetTransientViewportOffset = () => {
      const preserveTuiGesture = (touchActive || momentumFrame !== 0) && (isMouseModeActive() || isAlternateScreen())
      traceTerminalScroll('xterm.transient_reset', {
        machineId,
        terminalId,
        details: {
          alternateScreen: isAlternateScreen(),
          momentumActive: momentumFrame !== 0,
          mouseMode: isMouseModeActive(),
          preserveTuiGesture,
          touchActive,
        },
      })
      if (preserveTuiGesture) return
      clearMomentum()
      clearHistoryPullReleaseTimer()
      historyPullOffsetPx = 0
      historyPullReleasing = false
      updateHistoryPullIndicator()
      const currentScreenElement = resolveScreenElement()
      if (currentScreenElement) {
        currentScreenElement.style.transition = ''
        currentScreenElement.style.transform = ''
        currentScreenElement.style.willChange = ''
      }
      primedHistoryFrameRef.current?.setTransform('')
      smoothActive = false
      touchAccum = 0
      velocityY = 0
      touchLastY = Number.NaN
    }
    resetTransientViewportOffsetRef.current = resetTransientViewportOffset

    const scrollPixels = (px: number): boolean => {
      if (px > 0) noteUserScrollActivity()
      totalPxOffset += px
      const desiredViewportY = baseViewportY + Math.trunc(totalPxOffset / lineHeightPx)
      const currentViewportY = term.buffer.active.viewportY
      const lineDelta = terminalScrollLineDelta(desiredViewportY, currentViewportY)
      let clamped = false
      if (lineDelta !== 0) {
        if (px < 0) {
          armHistoryScrollbackLoad()
        }
        term.scrollLines(lineDelta)
        if (historyViewportControllerRef.current.confirmHistoryMovement(isScrolledToBottom(term))) {
          const primedFrame = primedHistoryFrameRef.current
          primedHistoryFrameRef.current = null
          primedFrame?.releaseAfterPaint()
        }
        const actualViewportY = Math.round(term.buffer.active.viewportY)
        if (actualViewportY !== Math.round(desiredViewportY)) {
          totalPxOffset = (actualViewportY - baseViewportY) * lineHeightPx
          clamped = true
          if (px < 0) {
            nudgeHistoryPullVisual(Math.abs(px))
            maybePrefetchScrollback()
          }
        }
      }
      syncTransform()
      const resumedLive = px > 0 && resumeFrozenHistoryAtBottomRef.current()
      return clamped || resumedLive
    }

    const flushScrollFrame = () => {
      if (scrollFrame) {
        window.cancelAnimationFrame(scrollFrame)
        scrollFrame = 0
      }
      if (pendingScrollPx === 0) return
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) {
        pendingScrollPx = 0
        return
      }
      const px = pendingScrollPx
      pendingScrollPx = 0
      scrollPixels(px)
    }

    const queueScrollPixels = (px: number) => {
      pendingScrollPx += px
      if (scrollFrame) return
      scrollFrame = window.requestAnimationFrame(() => {
        scrollFrame = 0
        if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
        if (pendingScrollPx === 0) return
        const queued = pendingScrollPx
        pendingScrollPx = 0
        scrollPixels(queued)
      })
    }

    const sendScrollInput = (down: boolean) => {
      if (isMouseModeActive()) {
        const targetElement = resolveScreenElement() ?? term.element
        if (!targetElement) return
        const rect = targetElement.getBoundingClientRect()
        const mouseService = coreTerminal._core?.coreMouseService
        if (mouseService?.triggerMouseEvent) {
          const width = Math.max(1, rect.width)
          const height = Math.max(1, rect.height)
          const x = Math.floor(width / 2)
          const y = Math.floor(height / 2)
          const col = Math.min(term.cols - 1, Math.max(0, Math.floor(x / (width / Math.max(1, term.cols)))))
          const row = Math.min(term.rows - 1, Math.max(0, Math.floor(y / (height / Math.max(1, term.rows)))))
          const accepted = mouseService.triggerMouseEvent({
            col,
            row,
            x,
            y,
            button: 4,
            action: down ? 1 : 0,
            ctrl: false,
            alt: false,
            shift: false,
          })
          traceTerminalScroll('xterm.mouse.dispatch', {
            machineId,
            terminalId,
            details: { accepted, direction: down ? 'down' : 'up', path: 'coreMouseService', col, row },
          })
          if (accepted) return
        }
        traceTerminalScroll('xterm.mouse.dispatch', {
          machineId,
          terminalId,
          details: { direction: down ? 'down' : 'up', path: 'wheelEventFallback' },
        })
        targetElement.dispatchEvent(new WheelEvent('wheel', {
          bubbles: true,
          cancelable: true,
          clientX: rect.left + rect.width / 2,
          clientY: rect.top + rect.height / 2,
          deltaMode: WheelEvent.DOM_DELTA_LINE,
          deltaY: down ? 1 : -1,
        }))
        return
      }
      const applicationCursor = isApplicationCursor()
      traceTerminalScroll('xterm.cursor.dispatch', {
        machineId,
        terminalId,
        details: { applicationCursor, direction: down ? 'down' : 'up' },
      })
      sendInputAtCurrentSize(down
        ? (applicationCursor ? '\x1bOB' : '\x1b[B')
        : (applicationCursor ? '\x1bOA' : '\x1b[A'))
    }

    const sendTuiScrollPixels = (pixelDelta: number, maxEvents = Number.POSITIVE_INFINITY) => {
      const accumulatedBefore = touchAccum
      touchAccum += pixelDelta
      const pendingLines = Math.trunc(touchAccum / lineHeightPx)
      if (pendingLines === 0) {
        traceTerminalScroll('tui.quantize', {
          machineId,
          terminalId,
          details: { accumulatedBefore, accumulatedAfter: touchAccum, lineHeightPx, maxEvents, pendingLines, pixelDelta },
        })
        return
      }
      const eventCount = Math.min(Math.abs(pendingLines), maxEvents)
      if (eventCount < Math.abs(pendingLines)) {
        touchAccum %= lineHeightPx
      } else {
        touchAccum -= pendingLines * lineHeightPx
      }
      traceTerminalScroll('tui.quantize', {
        machineId,
        terminalId,
        details: { accumulatedBefore, accumulatedAfter: touchAccum, eventCount, lineHeightPx, maxEvents, pendingLines, pixelDelta },
      })
      for (let index = 0; index < eventCount; index += 1) {
        sendScrollInput(pendingLines > 0)
        touchTuiEvents += 1
      }
    }

    const initTouchScroll = (event: TouchEvent) => {
      clearMomentum()
      terminalSession.cancelPendingMouseInput()
      smoothEnd()
      historyEntryGestureReady = historyViewportControllerRef.current.isPrimed &&
        historyLoadedRowsAppliedRef.current > 0
      let sumX = 0
      let sumY = 0
      for (let index = 0; index < event.touches.length; index += 1) {
        const touch = event.touches[index]!
        sumX += touch.clientX
        sumY += touch.clientY
      }
      const x = sumX / event.touches.length
      const y = sumY / event.touches.length
      touchLastY = y
      touchAccum = 0
      touchStartX = x
      touchStartY = y
      touchStartTime = Date.now()
      touchMoved = false
      touchActive = true
      velocityY = 0
      lastTouchTime = Date.now()
      touchScrollRoute = 'viewport'
      touchTuiEvents = 0
      traceTerminalScroll('touch.start', {
        machineId,
        terminalId,
        details: {
          alternateScreen: isAlternateScreen(),
          mouseMode: isMouseModeActive(),
          touches: event.touches.length,
          x,
          y,
        },
      })
    }

    const stopSelectionAutoScroll = () => {
      if (selectionAutoScrollId === null) return
      window.clearInterval(selectionAutoScrollId)
      selectionAutoScrollId = null
    }

    const hideMagnifier = () => {
      magnifier.style.display = 'none'
    }

    const resetSelectionTouchState = () => {
      selectionPhase = 'idle'
      selectionExtending = false
      selectionTouchActive = false
      stopSelectionAutoScroll()
      hideSelectionAnchor()
      hideMagnifier()
    }
    selectionResetHandlersRef.current.add(resetSelectionTouchState)

    const touchToBufferPosition = (clientX: number, clientY: number) => {
      const targetElement = resolveScreenElement() || container
      const rect = targetElement.getBoundingClientRect()
      const cellWidth = rect.width / term.cols
      const col = Math.max(0, Math.min(term.cols - 1, Math.floor((clientX - rect.left) / cellWidth)))
      const row = Math.floor((clientY - rect.top) / lineHeightPx)
      const bufferRow = Math.max(0, Math.min(term.buffer.active.length - 1, row + term.buffer.active.viewportY))
      return { col, row: bufferRow }
    }

    const applySelection = () => {
      let row1 = selectionAnchorRow
      let col1 = selectionAnchorCol
      let row2 = selectionEndRow
      let col2 = selectionEndCol
      if (row1 > row2 || (row1 === row2 && col1 > col2)) {
        ;[row1, col1, row2, col2] = [row2, col2, row1, col1]
      }
      const length = (row2 - row1) * term.cols + (col2 - col1) + 1
      term.select(col1, row1, length)
    }

    const showSelectionAnchor = (col: number, row: number) => {
      if (!selectionModeRef.current) {
        hideSelectionAnchor()
        return
      }
      const targetElement = resolveScreenElement() || container
      const rect = targetElement.getBoundingClientRect()
      const containerRect = container.getBoundingClientRect()
      const cellWidth = rect.width / term.cols
      const viewportRow = row - term.buffer.active.viewportY
      const x = rect.left - containerRect.left + col * cellWidth
      const y = rect.top - containerRect.top + viewportRow * lineHeightPx
      if (!Number.isFinite(x) || !Number.isFinite(y)) {
        hideSelectionAnchor()
        return
      }
      selectionAnchorMarker.style.left = `${x}px`
      selectionAnchorMarker.style.top = `${y}px`
      selectionAnchorMarker.style.height = `${lineHeightPx}px`
      selectionAnchorMarker.style.display = 'block'
      selectionAnchorMarker.style.opacity = '1'
    }

    const updateMagnifier = (col: number, bufferRow: number, clientX: number, clientY: number) => {
      const canvases = container.querySelectorAll('.xterm-screen canvas') as NodeListOf<HTMLCanvasElement>
      const context = magnifierCanvas.getContext('2d')
      if (!context || canvases.length === 0) return

      const referenceCanvas = canvases[0]!
      const viewRow = bufferRow - term.buffer.active.viewportY
      const centerX = ((col + 0.5) / term.cols) * referenceCanvas.width
      const centerY = ((viewRow + 0.5) / term.rows) * referenceCanvas.height
      const rect = (resolveScreenElement() || container).getBoundingClientRect()
      const scaleX = referenceCanvas.width / Math.max(1, rect.width)
      const scaleY = referenceCanvas.height / Math.max(1, rect.height)
      const sampleWidth = (magnifierWidth * scaleX) / 2
      const sampleHeight = (magnifierHeight * scaleY) / 2

      context.clearRect(0, 0, magnifierCanvas.width, magnifierCanvas.height)
      context.fillStyle = '#000'
      context.fillRect(0, 0, magnifierCanvas.width, magnifierCanvas.height)
      for (const canvas of canvases) {
        try {
          context.drawImage(
            canvas,
            centerX - sampleWidth / 2,
            centerY - sampleHeight / 2,
            sampleWidth,
            sampleHeight,
            0,
            0,
            magnifierCanvas.width,
            magnifierCanvas.height,
          )
        } catch {
          // Canvas can be temporarily unavailable while xterm swaps render layers.
        }
      }

      context.strokeStyle = 'rgba(255,255,255,0.5)'
      context.lineWidth = 1
      const midX = magnifierCanvas.width / 2
      const midY = magnifierCanvas.height / 2
      context.beginPath()
      context.moveTo(midX, 0)
      context.lineTo(midX, magnifierCanvas.height)
      context.moveTo(0, midY)
      context.lineTo(magnifierCanvas.width, midY)
      context.stroke()

      const containerRect = container.getBoundingClientRect()
      let x = clientX - containerRect.left - magnifierWidth / 2
      let y = clientY - containerRect.top - 70 - magnifierHeight
      if (x < 4) x = 4
      if (x + magnifierWidth > containerRect.width - 4) x = containerRect.width - magnifierWidth - 4
      if (y < 4) y = clientY - containerRect.top + 30
      magnifier.style.left = `${x}px`
      magnifier.style.top = `${y}px`
      magnifier.style.display = 'block'
    }

    const checkSelectionAutoScroll = (clientY: number) => {
      const targetElement = resolveScreenElement() || container
      const rect = targetElement.getBoundingClientRect()
      const zone = 40
      const atTop = clientY < rect.top + zone
      const atBottom = clientY > rect.bottom - zone
      if (!atTop && !atBottom) {
        stopSelectionAutoScroll()
        return
      }
      if (selectionAutoScrollId !== null) return
      selectionAutoScrollId = window.setInterval(() => {
        if (atTop) {
          historyEntryGestureReady = historyViewportControllerRef.current.isPrimed &&
            historyLoadedRowsAppliedRef.current > 0
          armHistoryScrollbackLoad()
        }
        term.scrollLines(atBottom ? 1 : -1)
        if (historyViewportControllerRef.current.confirmHistoryMovement(isScrolledToBottom(term))) {
          const primedFrame = primedHistoryFrameRef.current
          primedHistoryFrameRef.current = null
          primedFrame?.releaseAfterPaint()
        }
        if (atTop) maybePrefetchScrollback()
        if (selectionExtending) {
          const position = touchToBufferPosition(selectionLastClientX, selectionLastClientY)
          selectionEndCol = position.col
          selectionEndRow = position.row
          applySelection()
        }
      }, 80)
    }

    const enterSelectionMode = () => {
      selectionModeExitPendingRef.current = false
      resetSelectionTouchState()
      selectionModeWasActive = true
      armHistoryScrollbackLoad()
      void loadScrollbackPage(true)
    }

    const exitSelectionMode = () => {
      resetSelectionTouchState()
      selectionModeWasActive = false
      if (!historyViewportControllerRef.current.isLiveUpdateDeferred) return
      selectionModeExitPendingRef.current = true
      term.scrollToBottom()
      resumeFrozenHistoryAtBottomRef.current(true)
    }
    enterSelectionModeRef.current = enterSelectionMode
    exitSelectionModeRef.current = exitSelectionMode
    if (selectionModeRef.current) enterSelectionMode()

    const handleTouchStart = (event: TouchEvent) => {
      if (event.touches.length < 1) return
      cancelBottomAnchor()
      if (selectionModeRef.current && !selectionModeWasActive) {
        resetSelectionTouchState()
        selectionModeWasActive = true
      }
      if (!selectionModeRef.current && selectionModeWasActive) {
        resetSelectionTouchState()
        selectionModeWasActive = false
      }
      if (selectionModeRef.current && event.touches.length === 1) {
        event.preventDefault()
        event.stopPropagation()
        selectionTouchActive = true
        selectionExtending = false
        const touch = event.touches[0]
        if (!touch) return
        const position = touchToBufferPosition(touch.clientX, touch.clientY)
        if (selectionPhase === 'idle') {
          term.clearSelection()
          selectionAnchorCol = position.col
          selectionAnchorRow = position.row
          selectionPhase = 'anchor-set'
          showSelectionAnchor(position.col, position.row)
          updateMagnifier(position.col, position.row, touch.clientX, touch.clientY)
        } else {
          selectionEndCol = position.col
          selectionEndRow = position.row
          selectionExtending = true
          applySelection()
          hideSelectionAnchor()
          updateMagnifier(position.col, position.row, touch.clientX, touch.clientY)
        }
        return
      }
      if (selectionModeRef.current && event.touches.length >= 2) {
        selectionTouchActive = false
        selectionExtending = false
        stopSelectionAutoScroll()
        hideMagnifier()
        initTouchScroll(event)
        event.preventDefault()
        event.stopPropagation()
        return
      }
      event.preventDefault()
      event.stopPropagation()
      initTouchScroll(event)
    }

    const handleTouchMove = (event: TouchEvent) => {
      if (event.touches.length < 1) return
      if (selectionModeRef.current) {
        event.preventDefault()
        event.stopPropagation()
        if (event.touches.length >= 2 && selectionTouchActive) {
          selectionTouchActive = false
          selectionExtending = false
          stopSelectionAutoScroll()
          hideMagnifier()
          let sumY = 0
          for (let index = 0; index < event.touches.length; index += 1) {
            sumY += event.touches[index]!.clientY
          }
          touchLastY = sumY / event.touches.length
          touchAccum = 0
          velocityY = 0
          lastTouchTime = Date.now()
          touchMoved = false
          touchStartY = touchLastY
          touchStartTime = Date.now()
        }
        if (selectionTouchActive && event.touches.length === 1) {
          const touch = event.touches[0]
          if (!touch) return
          selectionLastClientX = touch.clientX
          selectionLastClientY = touch.clientY
          const position = touchToBufferPosition(touch.clientX, touch.clientY)
          if (selectionExtending) {
            selectionEndCol = position.col
            selectionEndRow = position.row
            applySelection()
            updateMagnifier(position.col, position.row, touch.clientX, touch.clientY)
            checkSelectionAutoScroll(touch.clientY)
            return
          }
          selectionAnchorCol = position.col
          selectionAnchorRow = position.row
          showSelectionAnchor(position.col, position.row)
          updateMagnifier(position.col, position.row, touch.clientX, touch.clientY)
          return
        }
        if (event.touches.length >= 2) {
          let sumY = 0
          for (let index = 0; index < event.touches.length; index += 1) {
            sumY += event.touches[index]!.clientY
          }
          const y = sumY / event.touches.length
          if (Number.isNaN(touchLastY)) {
            touchLastY = y
            lastTouchTime = Date.now()
            return
          }
          if (Math.abs(y - touchStartY) > 10) touchMoved = true
          const now = Date.now()
          const dt = now - lastTouchTime
          const dy = touchLastY - y
          if (dt > 0) {
            const instantVelocity = (dy / dt) * 1000
            velocityY = velocityY * 0.3 + instantVelocity * 0.7
          }
          lastTouchTime = now
          touchLastY = y
          smoothBegin()
          queueScrollPixels(dy)
          return
        }
        return
      }
      event.preventDefault()
      event.stopPropagation()
      let sumY = 0
      for (let index = 0; index < event.touches.length; index += 1) {
        sumY += event.touches[index]!.clientY
      }
      const y = sumY / event.touches.length
      if (Number.isNaN(touchLastY)) {
        touchLastY = y
        lastTouchTime = Date.now()
        return
      }
      if (Math.abs(y - touchStartY) > 10) touchMoved = true

      const now = Date.now()
      const dt = now - lastTouchTime
      const dy = touchLastY - y
      if (dt > 0) {
        const instantVelocity = (dy / dt) * 1000
        velocityY = velocityY * 0.3 + instantVelocity * 0.7
      }
      lastTouchTime = now
      touchLastY = y

      traceTerminalScroll('touch.move', {
        machineId,
        terminalId,
        details: {
          alternateScreen: isAlternateScreen(),
          dt,
          dy,
          mouseMode: isMouseModeActive(),
          velocityY,
          y,
        },
      })

      if (isMouseModeActive() || isAlternateScreen()) {
        touchScrollRoute = 'tui'
        if (smoothActive) smoothEnd()
        sendTuiScrollPixels(dy)
        return
      }

      touchScrollRoute = 'viewport'
      smoothBegin()
      queueScrollPixels(dy)
    }

    const handleTouchEnd = () => {
      flushScrollFrame()
      touchActive = false
      if (selectionModeRef.current) {
        if (selectionTouchActive) {
          selectionTouchActive = false
          stopSelectionAutoScroll()
          hideMagnifier()
          if (selectionExtending) {
            selectionAnchorCol = selectionEndCol
            selectionAnchorRow = selectionEndRow
            selectionExtending = false
          }
          return
        }
        smoothEnd()
        releaseHistoryPullVisual()
        touchLastY = Number.NaN
        return
      }
      const touchEndNow = Date.now()
      const gestureDt = Math.max(1, touchEndNow - touchStartTime)
      const gestureVelocity = Number.isFinite(touchLastY)
        ? ((touchStartY - touchLastY) / gestureDt) * 1000
        : 0
      if (Number.isFinite(gestureVelocity) && Math.abs(gestureVelocity) > Math.abs(velocityY)) {
        velocityY = gestureVelocity
      }
      traceTerminalScroll('touch.end', {
        machineId,
        terminalId,
        details: {
          ageSinceMove: touchEndNow - lastTouchTime,
          alternateScreen: isAlternateScreen(),
          gestureDuration: gestureDt,
          gestureVelocity,
          inertia: settingsRef.current.scrollInertia,
          mouseMode: isMouseModeActive(),
          moved: touchMoved,
          route: touchScrollRoute,
          tuiEvents: touchTuiEvents,
          velocityY,
        },
      })
      if (touchMoved) {
        logTerminal('touch_scroll_end', {
          details: {
            inertia: settingsRef.current.scrollInertia,
            lineHeightPx,
            mouseMode: isMouseModeActive(),
            alternateScreen: isAlternateScreen(),
            route: touchScrollRoute,
            tuiEvents: touchTuiEvents,
            velocityY: Math.round(velocityY),
          },
        })
      }
      if (historyPullOffsetPx > 0) {
        velocityY = 0
        smoothEnd()
        releaseHistoryPullVisual()
      } else if (!touchMoved && touchEndNow - touchStartTime < 500) {
        const currentScreenElement = resolveScreenElement()
        if (isMouseModeActive() && currentScreenElement) {
          const mouseOptions: MouseEventInit = {
            bubbles: true,
            button: 0,
            cancelable: true,
            clientX: touchStartX,
            clientY: touchStartY,
          }
          currentScreenElement.dispatchEvent(new MouseEvent('mousedown', { ...mouseOptions, buttons: 1 }))
          currentScreenElement.dispatchEvent(new MouseEvent('mouseup', { ...mouseOptions, buttons: 0 }))
        }
        try {
          if (!preventFocusRef.current) xtermRef.current?.focus()
        } catch {
          // Ignore stale focus work while unmounting.
        }
        velocityY = 0
        smoothEnd()
      } else if (!prefersReducedTerminalMotion() && resolveTerminalMomentumProfile(settingsRef.current.scrollInertia).enabled && Math.abs(velocityY) > 60) {
        if (touchEndNow - lastTouchTime > 80) {
          traceTerminalScroll('momentum.stop', {
            machineId,
            terminalId,
            details: { reason: 'stale_touch', ageSinceMove: touchEndNow - lastTouchTime },
          })
          velocityY = 0
          smoothEnd()
        } else {
          let lastFrameTime = performance.now()
          const step = (now: number) => {
            if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) {
              traceTerminalScroll('momentum.stop', { machineId, terminalId, details: { reason: 'terminal_disposed' } })
              momentumFrame = 0
              smoothEnd()
              return
            }
            const currentProfile = resolveTerminalMomentumProfile(settingsRef.current.scrollInertia)
            if (!currentProfile.enabled) {
              traceTerminalScroll('momentum.stop', { machineId, terminalId, details: { reason: 'disabled' } })
              momentumFrame = 0
              smoothEnd()
              return
            }
            const frameDt = Math.min(0.032, Math.max(0, (now - lastFrameTime) / 1000))
            lastFrameTime = now
            velocityY *= Math.pow(currentProfile.deceleration, frameDt * 60)
            if (Math.abs(velocityY) < currentProfile.minimumVelocity) {
              traceTerminalScroll('momentum.stop', {
                machineId,
                terminalId,
                details: { reason: 'minimum_velocity', velocityY },
              })
              momentumFrame = 0
              smoothEnd()
              return
            }
            const pixelDelta = velocityY * frameDt
            traceTerminalScroll('momentum.frame', {
              machineId,
              terminalId,
              details: {
                alternateScreen: isAlternateScreen(),
                frameDt,
                mouseMode: isMouseModeActive(),
                pixelDelta,
                velocityY,
              },
            })
            if (isMouseModeActive() || isAlternateScreen()) {
              if (smoothActive) smoothEnd()
              sendTuiScrollPixels(pixelDelta, 1)
            } else if (scrollPixels(pixelDelta)) {
              momentumFrame = 0
              smoothEnd()
              return
            }
            momentumFrame = window.requestAnimationFrame(step)
          }
          touchAccum = 0
          traceTerminalScroll('momentum.start', {
            machineId,
            terminalId,
            details: { route: touchScrollRoute, velocityY },
          })
          momentumFrame = window.requestAnimationFrame(step)
        }
      } else {
        smoothEnd()
      }
      touchLastY = Number.NaN
      if (historyApplyTimerRef.current === null && pendingHistoryApplyRef.current) {
        scheduleHistoryApply()
      }
    }

    const handleRenderDisposable = typeof term.onRender === 'function'
      ? term.onRender(() => {
        const viewportY = term.buffer.active.viewportY
        if (viewportY !== renderedViewportY) {
          const movingTowardOlderHistory = viewportY < renderedViewportY
          if (historyViewportControllerRef.current.confirmHistoryMovement(isScrolledToBottom(term))) {
            const primedFrame = primedHistoryFrameRef.current
            primedHistoryFrameRef.current = null
            primedFrame?.releaseAfterPaint()
          }
          renderedViewportY = viewportY
          if (smoothActive) syncTransform()
          updateScrollbar()
          if (historyLoadingRef.current) {
            if (historyRestoreViewportOnLoadRef.current) {
              pendingHistoryViewportRef.current = term.buffer.active.viewportY
            }
          } else {
            if (historyLoadArmedByUserRef.current && movingTowardOlderHistory) {
              maybePrefetchScrollback()
            }
          }
          const pendingHistoryApply = pendingHistoryApplyRef.current
          if (pendingHistoryApply !== null && pendingHistoryApply.restoreViewportY !== null) {
            pendingHistoryApplyRef.current = {
              ...pendingHistoryApply,
              restoreViewportY: term.buffer.active.viewportY,
            }
          }
          noteHistoryScrollActivity()
          resumeFrozenHistoryAtBottomRef.current()
          updateHistoryStatus()
          showScrollbar()
          hideScrollbarDelayed()
        }
        if (historySearchOpenRef.current) paintHistorySearchHighlights(term)
      })
      : { dispose() {} }
    const bufferDisposable = term.buffer.onBufferChange?.(() => {
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
      const preserveTuiMomentum = momentumFrame !== 0 && (isMouseModeActive() || isAlternateScreen())
      traceTerminalScroll('xterm.buffer_change', {
        machineId,
        terminalId,
        details: {
          alternateScreen: isAlternateScreen(),
          bufferType: term.buffer.active.type,
          momentumActive: momentumFrame !== 0,
          mouseMode: isMouseModeActive(),
          preserveTuiMomentum,
        },
      })
      if (!preserveTuiMomentum) {
        clearMomentum()
        smoothEnd()
      }
      updateScrollbar()
      onBufferChangeRef.current?.(term.buffer.active.type === 'alternate')
    }) ?? { dispose() {} }

    const scrollTouchTarget = term.element || container
    scrollTouchTarget.addEventListener('touchstart', handleTouchStart, { capture: true, passive: false })
    scrollTouchTarget.addEventListener('touchmove', handleTouchMove, { capture: true, passive: false })
    scrollTouchTarget.addEventListener('touchend', handleTouchEnd, { capture: true, passive: true })
    scrollTouchTarget.addEventListener('touchcancel', handleTouchEnd, { capture: true, passive: true })

    const handleWheelTrace = (event: WheelEvent) => {
      traceTerminalScroll('wheel.raw', {
        machineId,
        terminalId,
        details: {
          alternateScreen: isAlternateScreen(),
          deltaMode: event.deltaMode,
          deltaX: event.deltaX,
          deltaY: event.deltaY,
          mouseMode: isMouseModeActive(),
          timeStamp: event.timeStamp,
        },
      })
    }
    scrollTouchTarget.addEventListener('wheel', handleWheelTrace, { capture: true, passive: true })

    const handleWheel = (event: WheelEvent) => {
      if (isMouseModeActive() || isAlternateScreen()) return
      const thresholdRows = terminalHistoryPrefetchThresholdRows(
        settingsRef.current.scrollbackPrefetchThresholdRows,
        term.rows,
      )
      if (event.deltaY < 0) {
        historyEntryGestureReady = historyViewportControllerRef.current.isPrimed &&
          historyLoadedRowsAppliedRef.current > 0
        armHistoryScrollbackLoad()
        if (term.buffer.active.viewportY <= thresholdRows) {
          if (term.buffer.active.viewportY <= 0) {
            nudgeHistoryPullVisual(Math.max(8, Math.abs(event.deltaY)))
          }
          maybePrefetchScrollback()
        }
      } else if (event.deltaY !== 0) {
        noteUserScrollActivity()
        resumeFrozenHistoryAtBottomRef.current()
      }
    }
    scrollTouchTarget.addEventListener('wheel', handleWheel, { passive: true })

    const handleScrollbarTouchStart = (event: TouchEvent) => {
      event.preventDefault()
      event.stopPropagation()
      hapticSelection()
      historyEntryGestureReady = historyViewportControllerRef.current.isPrimed &&
        historyLoadedRowsAppliedRef.current > 0
      scrollbarDragging = true
      scrollbarTrack.classList.add('active')
      showScrollbar()
      const touch = event.touches[0]
      if (!touch) return
      scrollbarDragStartY = touch.clientY
      const buffer = term.buffer.active
      const maxScroll = buffer.length - term.rows
      scrollbarDragStartRatio = maxScroll > 0 ? buffer.viewportY / maxScroll : 0
    }
    const handleScrollbarTouchMove = (event: TouchEvent) => {
      if (!scrollbarDragging) return
      event.preventDefault()
      event.stopPropagation()
      const touch = event.touches[0]
      if (!touch) return
      const scrollableTrack = scrollbarTrack.clientHeight - scrollbarThumb.clientHeight
      if (scrollableTrack <= 0) return
      const deltaRatio = (touch.clientY - scrollbarDragStartY) / scrollableTrack
      const nextRatio = Math.max(0, Math.min(1, scrollbarDragStartRatio + deltaRatio))
      const buffer = term.buffer.active
      const maxScroll = buffer.length - term.rows
      const targetY = Math.round(nextRatio * maxScroll)
      const diff = targetY - buffer.viewportY
      if (diff !== 0) {
        if (diff < 0) {
          armHistoryScrollbackLoad()
        } else {
          noteUserScrollActivity()
        }
        term.scrollLines(diff)
      }
      updateScrollbar()
    }
    const handleScrollbarTouchEnd = (event: TouchEvent) => {
      if (!scrollbarDragging) return
      event.preventDefault()
      scrollbarDragging = false
      scrollbarTrack.classList.remove('active')
      hideScrollbarDelayed()
      if (historyApplyTimerRef.current === null && pendingHistoryApplyRef.current) {
        scheduleHistoryApply()
      }
    }
    const handleScrollbarTrackTouchStart = (event: TouchEvent) => {
      if (event.target !== scrollbarTrack) return
      event.preventDefault()
      event.stopPropagation()
      hapticSelection()
      historyEntryGestureReady = historyViewportControllerRef.current.isPrimed &&
        historyLoadedRowsAppliedRef.current > 0
      const touch = event.touches[0]
      if (!touch) return
      const rect = scrollbarTrack.getBoundingClientRect()
      const ratio = Math.max(0, Math.min(1, (touch.clientY - rect.top) / rect.height))
      const buffer = term.buffer.active
      const maxScroll = buffer.length - term.rows
      const targetY = Math.round(ratio * maxScroll)
      const diff = targetY - buffer.viewportY
      if (diff !== 0) {
        if (diff < 0) {
          armHistoryScrollbackLoad()
        } else {
          noteUserScrollActivity()
        }
        term.scrollLines(diff)
      }
      updateScrollbar()
      scrollbarDragging = true
      scrollbarTrack.classList.add('active')
      showScrollbar()
      scrollbarDragStartY = touch.clientY
      scrollbarDragStartRatio = maxScroll > 0 ? buffer.viewportY / maxScroll : 0
    }

    scrollbarThumb.addEventListener('touchstart', handleScrollbarTouchStart, { passive: false })
    scrollbarTrack.addEventListener('touchstart', handleScrollbarTrackTouchStart, { passive: false })
    document.addEventListener('touchmove', handleScrollbarTouchMove, { passive: false })
    document.addEventListener('touchend', handleScrollbarTouchEnd, { passive: false })
    document.addEventListener('touchcancel', handleScrollbarTouchEnd, { passive: false })

    fitAndMaybeSendResize()
    scheduleFit()
    let resizeObserver: ResizeObserver | null = null
    if (typeof ResizeObserver !== 'undefined') {
      resizeObserver = new ResizeObserver(() => {
        if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
        resetTransientViewportOffset()
        fitAndMaybeSendResize()
        lineHeightPx = getLineHeight()
      })
      resizeObserver.observe(container)
    }

    const clearViewportOffsetForKeyboard = () => {
      if (terminalDisposedRef.current || terminalGenerationRef.current !== generation) return
      resetTransientViewportOffset()
      smoothEnd()
    }

    const removeNativeKeyboardListener = addNativeKeyboardListener(clearViewportOffsetForKeyboard)
    document.addEventListener('anytty:resume', clearViewportOffsetForKeyboard)
    let expectedProbeAt = terminalNow() + eventLoopProbeIntervalMs
    const eventLoopProbe = window.setInterval(() => {
      const now = terminalNow()
      const lagMs = now - expectedProbeAt
      expectedProbeAt = now + eventLoopProbeIntervalMs
      if (lagMs >= eventLoopLagWarnMs) {
        logTerminal('event_loop_lag', {
          level: 'warn',
          details: {
            lagMs: Math.round(lagMs),
            pendingWriteCallbacks: xtermWriteStatsRef.current.pendingCallbacks,
            terminalTextChars: latestTerminalTextRef.current.length,
            lastWrittenChars: lastWrittenTextRef.current.length,
            historyApplying: historyApplyingRef.current,
            historyLoading: historyLoadingRef.current,
          },
        })
      }
    }, eventLoopProbeIntervalMs)

    if (window.visualViewport) {
      window.visualViewport.addEventListener('resize', clearViewportOffsetForKeyboard)
      window.visualViewport.addEventListener('scroll', clearViewportOffsetForKeyboard)
    }

    return () => {
      terminalDisposedRef.current = true
      terminalGenerationRef.current += 1
      if (window.visualViewport) {
        window.visualViewport.removeEventListener('resize', clearViewportOffsetForKeyboard)
        window.visualViewport.removeEventListener('scroll', clearViewportOffsetForKeyboard)
      }
      removeNativeKeyboardListener()
      document.removeEventListener('anytty:resume', clearViewportOffsetForKeyboard)
      window.clearInterval(eventLoopProbe)
      if (fitFrameRef.current !== null) {
        window.cancelAnimationFrame(fitFrameRef.current)
        fitFrameRef.current = null
      }
      if (fitRetryTimerRef.current !== null) {
        window.clearTimeout(fitRetryTimerRef.current)
        fitRetryTimerRef.current = null
      }
      fitRetryCountRef.current = 0
      cancelBottomAnchor()
      clearLiveOutputWatchdog()
      liveOutputInFlightRef.current = false
      liveOutputGenerationRef.current += 1
      pendingLiveOutputRef.current = ''
      liveOutputDroppedCharsRef.current = 0
      liveOutputSyncLostRef.current = false
      if (historyApplyTimerRef.current !== null) {
        window.clearTimeout(historyApplyTimerRef.current)
        historyApplyTimerRef.current = null
      }
      pendingHistoryApplyRef.current = null
      historyPrefetchPending = false
      clearHistoryPullReleaseTimer()
      historyPullOffsetPx = 0
      historyPullLoading = false
      historyPullReleasing = false
      clearHistorySearchHighlight()
      historyLoadArmedByUserRef.current = false
      primedHistoryFrameRef.current?.remove()
      primedHistoryFrameRef.current = null
      historyViewportControllerRef.current.reset()
      historyRequestColsRef.current = 0
      historyProjectionReloadPendingRef.current = false
      historyLoadedRowsRequestedRef.current = 0
      historyHasMoreRef.current = true
      retryHistoryLoadRef.current = () => {}
      hideHistoryLoading(true)
      container.querySelectorAll('[data-anytty-terminal-frame-hold]').forEach((element) => element.remove())
      maybePrefetchScrollbackRef.current = () => {}
      if (enterSelectionModeRef.current === enterSelectionMode) enterSelectionModeRef.current = () => {}
      if (exitSelectionModeRef.current === exitSelectionMode) exitSelectionModeRef.current = () => {}
      selectionModeExitPendingRef.current = false
      scheduleHistoryApplyRef.current = () => {}
      resetTransientViewportOffsetRef.current = () => {}
      resizeObserver?.disconnect()
      dataDisposable.dispose()
      binaryDisposable.dispose()
      cursorDisposable.dispose()
      handleRenderDisposable.dispose()
      bufferDisposable.dispose()
      textarea?.removeEventListener('compositionstart', handleCompositionStart)
      textarea?.removeEventListener('compositionupdate', handleCompositionUpdate)
      textarea?.removeEventListener('compositionend', handleCompositionEnd)
      textarea?.removeEventListener('focus', onFocus)
      compositionOverlay.remove()
      clearMomentum()
      smoothEnd()
      selectionResetHandlersRef.current.delete(resetSelectionTouchState)
      resetSelectionTouchState()
      window.clearTimeout(scrollbarHideTimer)
      scrollTouchTarget.removeEventListener('touchstart', handleTouchStart, true)
      scrollTouchTarget.removeEventListener('touchmove', handleTouchMove, true)
      scrollTouchTarget.removeEventListener('touchend', handleTouchEnd, true)
      scrollTouchTarget.removeEventListener('touchcancel', handleTouchEnd, true)
      scrollTouchTarget.removeEventListener('wheel', handleWheelTrace, true)
      scrollTouchTarget.removeEventListener('wheel', handleWheel)
      scrollbarThumb.removeEventListener('touchstart', handleScrollbarTouchStart)
      scrollbarTrack.removeEventListener('touchstart', handleScrollbarTrackTouchStart)
      document.removeEventListener('touchmove', handleScrollbarTouchMove)
      document.removeEventListener('touchend', handleScrollbarTouchEnd)
      document.removeEventListener('touchcancel', handleScrollbarTouchEnd)
      scrollbarTrack.remove()
      historyPullIndicator.remove()
      historySearchHighlightLayer.remove()
      if (historySearchHighlightLayerRef.current === historySearchHighlightLayer) {
        historySearchHighlightLayerRef.current = null
      }
      magnifier.remove()
      selectionAnchorMarker.remove()
      if (xtermRef.current === term) xtermRef.current = null
      setXtermReady(false)
      if (fitAddonRef.current === fitAddon) fitAddonRef.current = null
      try {
        term.dispose()
      } catch {
        // Ignore dispose races from xterm internals during React teardown.
      }
      lastWrittenTextRef.current = ''
      lastSentResizeRef.current = null
      isOpenRef.current = false
      updateHistoryStatusRef.current = () => {}
      if (historyStatusRef.current) {
        if (historyStatusTextRef.current) historyStatusTextRef.current.textContent = ''
      }
      setHistoryStatusVisible(false)
    }
  }, [cancelBottomAnchor, clearHistorySearchHighlight, clearLiveOutputWatchdog, fitAndMaybeSendResize, historyOnly, isScrolledToBottom, keepBottomAnchored, logTerminal, markSurfaceReady, paintHistorySearchHighlights, reloadHistoryProjectionWhenIdle, renderer, scheduleFit, sendInputAtCurrentSize, sendUserInput, settings.renderer, writeToXterm])

  useEffect(() => {
    isOpenRef.current = isOpen
    if (!isOpen) {
      primedHistoryFrameRef.current?.remove()
      primedHistoryFrameRef.current = null
      if (historyViewportControllerRef.current.isLiveUpdateDeferred) {
        resumeLiveScrollbackRef.current()
      }
      historyViewportControllerRef.current.reset()
      historyRequestColsRef.current = 0
      clearLiveOutputWatchdog()
      liveOutputInFlightRef.current = false
      liveOutputGenerationRef.current += 1
      pendingLiveOutputRef.current = ''
      liveOutputDroppedCharsRef.current = 0
      liveOutputSyncLostRef.current = false
      lastWrittenTextRef.current = ''
      lastSnapshotTextRef.current = ''
      recoveryRevisionAppliedRef.current = 0
      surfaceReadyRef.current = false
      initialViewportReadyRef.current = false
      setSurfaceReady(false)
      setInitialViewportReady(false)
      return
    }
    fitAndMaybeSendResize()
    scheduleFit()
    onReady?.()
  }, [clearLiveOutputWatchdog, fitAndMaybeSendResize, isOpen, onReady, scheduleFit])

  useEffect(() => {
    if (terminalDisposedRef.current) return
    const term = xtermRef.current
    if (!term) return

    const snapshot = terminalSession.terminalSnapshot
    const history = snapshot?.history
    const snapshotText = snapshot ? (snapshot.screenReplay ?? snapshot.replay ?? snapshot.text) : ''
    const authoritativeSnapshotChanged = Boolean(
      snapshot && !history && snapshotText !== lastSnapshotTextRef.current,
    )
    if (snapshot) {
      hasTerminalSnapshotRef.current = true
      snapshotAlternateScreenRef.current = snapshot.alternateScreen === true
      if (authoritativeSnapshotChanged && !historyViewportControllerRef.current.isLiveUpdateDeferred) {
        historyLoadedRowsRequestedRef.current = 0
        historyHasMoreRef.current = true
      }
    }
    const recoveryRevision = snapshot?.recovery?.revision ?? 0
    const liveRevision = snapshot?.liveRevision
    const liveSnapshotCandidate = Boolean(
      snapshot &&
      liveRevision !== undefined &&
      snapshot.liveReplay !== undefined &&
      liveRevision !== lastLiveScreenSubmittedRevisionRef.current,
    )
    const snapshotReplayCandidate = liveSnapshotCandidate || Boolean(
      snapshot &&
      liveRevision === undefined &&
      snapshotText &&
      (snapshotText !== lastSnapshotTextRef.current || recoveryRevision > recoveryRevisionAppliedRef.current) &&
      (lastWrittenTextRef.current === '' || recoveryRevision > recoveryRevisionAppliedRef.current),
    )
    const liveUpdateAlreadyDeferred = historyViewportControllerRef.current.hasDeferredLiveUpdate
    const shouldReplaySnapshot = snapshotReplayCandidate && historyViewportControllerRef.current.shouldRenderLiveUpdate()
    if (snapshotReplayCandidate && !shouldReplaySnapshot && !liveUpdateAlreadyDeferred) {
      logTerminal('history_live_update_deferred', {
        details: { source: 'snapshot', refreshReason: snapshot?.refreshReason },
      })
    }

    if (shouldReplaySnapshot) {
      resetTransientViewportOffsetRef.current()
      const replacesLiveFrame = !liveSnapshotCandidate || snapshot!.liveFullReplace
      const screen = replacesLiveFrame
        ? containerRef.current?.querySelector('.xterm-screen') as HTMLElement | null
        : null
      const heldFrame = replacesLiveFrame && containerRef.current && screen
        ? holdTerminalFrame(containerRef.current, screen)
        : null
      try {
        clearLiveOutputWatchdog()
        liveOutputInFlightRef.current = false
        liveOutputGenerationRef.current += 1
        pendingLiveOutputRef.current = ''
        liveOutputDroppedCharsRef.current = 0
        liveOutputSyncLostRef.current = false
        const liveReplay = liveSnapshotCandidate ? snapshot!.liveReplay! : terminalSession.terminalText
        if (replacesLiveFrame) term.reset()
        if (liveRevision !== undefined) {
          lastLiveScreenSubmittedRevisionRef.current = liveRevision
          terminalSession.markLiveScreenSubmitted(liveRevision)
        }
        writeToXterm(term, liveReplay, liveSnapshotCandidate
          ? (snapshot!.liveFullReplace ? 'live_screen_full' : 'live_screen_delta')
          : recoveryRevision > recoveryRevisionAppliedRef.current ? 'snapshot_recovery' : 'snapshot_full_text', () => {
          if (liveRevision !== undefined) terminalSession.markLiveScreenCompleted(liveRevision)
          markSurfaceReady()
          keepBottomAnchored()
          heldFrame?.releaseAfterPaint()
        })
        lastWrittenTextRef.current = terminalSession.terminalText
        if (recoveryRevision > recoveryRevisionAppliedRef.current) {
          recoveryRevisionAppliedRef.current = recoveryRevision
        }
      } catch (error) {
        if (liveRevision !== undefined) terminalSession.markLiveScreenCompleted(liveRevision)
        heldFrame?.remove()
        if (!terminalDisposedRef.current) throw error
      }
    }
    lastSnapshotTextRef.current = snapshotText

    if (history && history.revision > historyRevisionAppliedRef.current) {
      historyRevisionAppliedRef.current = history.revision
      if (history.cols !== term.cols) {
        pendingHistoryApplyRef.current = null
        pendingHistoryViewportRef.current = null
        hideHistoryLoading()
        historyProjectionReloadPendingRef.current = true
        reloadHistoryProjectionWhenIdle()
        return
      }
      const restoreViewportY = pendingHistoryViewportRef.current
      pendingHistoryViewportRef.current = null
      const replacesHistory = history.operation === 'replace'
      const previousPending = pendingHistoryApplyRef.current
      const previousPendingRows = replacesHistory ? 0 : previousPending?.loadedRows ?? historyLoadedRowsAppliedRef.current
      const pendingPrependedRows = Math.max(0, history.loadedRows - previousPendingRows)
      if (history.alternate) {
        term.options.scrollback = Math.max(
          typeof term.options.scrollback === 'number' ? term.options.scrollback : 0,
          Math.min(history.loadedRows + term.rows, terminalHistoryLoadedRowsLimit(settingsRef.current)),
        )
        term.reset()
      }
      pendingHistoryApplyRef.current = {
        revision: history.revision,
        cols: history.cols,
        loadedRows: history.loadedRows,
        prependedRows: (replacesHistory ? 0 : previousPending?.prependedRows ?? 0) + pendingPrependedRows,
        operation: replacesHistory ? 'replace' : 'prepend',
        restoreViewportY: restoreViewportY ?? previousPending?.restoreViewportY ?? null,
        viewportTop: historyOnly ? undefined : history.viewportTop ?? previousPending?.viewportTop,
        absoluteViewportY: history.searchMatchRow ?? previousPending?.absoluteViewportY,
        searchMatchRanges: history.searchMatchRanges ??
          (history.searchMatchRow !== undefined ? previousPending?.searchMatchRanges : undefined),
        text: terminalSession.terminalText,
      }
      if (history.searchMatchRow !== undefined) {
        historySearchBottomResumeSuppressedRef.current = true
      }
      historyLoadedRowsRequestedRef.current = replacesHistory
        ? history.loadedRows
        : Math.max(historyLoadedRowsRequestedRef.current, history.loadedRows)
      historyHasMoreRef.current = history.hasMore
      showHistoryLoading()
      scheduleHistoryApplyRef.current(history.prefetched === true)
      return
    }

    const pendingHistoryApply = pendingHistoryApplyRef.current
    if (pendingHistoryApply) {
      pendingHistoryApplyRef.current = {
        ...pendingHistoryApply,
        text: terminalSession.terminalText,
      }
      return
    }
  }, [clearLiveOutputWatchdog, historyOnly, keepBottomAnchored, logTerminal, markSurfaceReady, reloadHistoryProjectionWhenIdle, terminalSession.markLiveScreenCompleted, terminalSession.markLiveScreenSubmitted, terminalSession.terminalSnapshot, terminalSession.terminalText, writeToXterm])

  const historySearchActiveIndex = historySearchActiveMatch
    ? historySearchMatches.findIndex((match) => sameHistoryLogicalMatch(match, historySearchActiveMatch)) + 1
    : 0
  const historySearchCountLabel = !historySearchQuery.trim()
    ? '0 / 0'
    : historySearchScanBusy
      ? `${historySearchActiveIndex > 0 ? historySearchActiveIndex : '–'} / ${historySearchMatches.length}+`
      : historySearchScanComplete
        ? `${historySearchActiveIndex > 0 ? historySearchActiveIndex : historySearchMatches.length === 0 ? 0 : '–'} / ${historySearchMatches.length}`
        : historySearchActiveMatch
          ? '– / ?'
          : '0 / ?'
  const historySearchRailMatches = historySearchMatches.length <= 120
    ? historySearchMatches
    : historySearchMatches.filter((_, index) => index % Math.ceil(historySearchMatches.length / 120) === 0)

  return (
    <section
      className={`relative isolate flex h-full min-h-0 w-full flex-col overflow-hidden ${className || ''}`}
      data-machine-id={machineId}
      data-terminal-id={terminalId}
      data-phase={terminalSession.snapshot.phase}
      data-channel-state={terminalSession.snapshot.terminalChannels[terminalId]?.state ?? 'closed'}
      data-history-only={historyOnly ? 'true' : undefined}
      data-viewport-ready={initialViewportReady ? 'true' : 'false'}
      data-testid="anytty-terminal"
    >
      <div
        ref={containerRef}
        aria-label={t('terminal.tools.output')}
        aria-hidden={!terminalSurfaceVisible ? true : undefined}
        className="absolute inset-0 min-h-0 overflow-hidden xterm-wrapper outline-none"
        inert={!terminalSurfaceVisible ? true : undefined}
        style={{
          opacity: terminalSurfaceVisible ? 1 : 0,
          overscrollBehavior: 'contain',
          touchAction: 'none',
        }}
        role="application"
        tabIndex={terminalSurfaceVisible ? 0 : -1}
      />
      {historyLoadingVisible && !showConnectingOverlay ? (
        <div
          aria-live="polite"
          className="pointer-events-none absolute left-1/2 top-3 z-[60] flex max-w-[calc(100%_-_2rem)] -translate-x-1/2 items-center gap-2 whitespace-nowrap rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)]/95 px-3 py-2 text-xs font-medium text-[var(--anytty-text)] shadow-sm backdrop-blur-md"
          data-testid="anytty-history-loading"
          role="status"
        >
          <Spinner className="text-[var(--anytty-accent)]" aria-hidden="true" />
          <span className="truncate">{t('terminal.tools.loadingHistory')}</span>
        </div>
      ) : null}
      <div
        ref={historyStatusRef}
        className={`absolute z-[55] min-h-11 max-w-[calc(100%_-_2rem)] items-center rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)]/90 pl-2 font-mono text-[11px] tabular-nums text-[var(--anytty-muted)] backdrop-blur ${historyStatusVisible ? 'flex' : 'hidden'}`}
        data-testid="anytty-history-position"
        style={{
          bottom: 'max(0.75rem, env(safe-area-inset-bottom))',
          right: 'max(1rem, env(safe-area-inset-right))',
        }}
      >
        <span ref={historyStatusTextRef} className="truncate" />
        <Button variant="ghost"
          type="button"
          aria-label={t('terminal.tools.searchHistory')}
          className="ml-1 grid h-11 w-11 shrink-0 place-items-center text-[var(--anytty-text)] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-[-3px]"
          onClick={() => {
            void hapticSelection()
            setHistorySearchOpen(true)
          }}
        >
          <Search className="h-4 w-4" />
        </Button>
      </div>
      {historySearchOpen && historySearchRailMatches.length > 0 ? (
        <div
          aria-hidden="true"
          className="pointer-events-none absolute right-0 top-2 z-[58] w-1"
          style={{ bottom: 'calc(env(safe-area-inset-bottom) + 6.75rem)' }}
        >
          {historySearchRailMatches.map((match, index) => {
            const current = historySearchActiveMatch ? sameHistoryLogicalMatch(match, historySearchActiveMatch) : false
            return <span
              key={`${match.startLineId}:${match.startCol}:${index}`}
              className={`absolute right-0 block h-0.5 ${current ? 'w-1 bg-amber-300' : 'w-0.5 bg-amber-500/55'}`}
              style={{ top: historySearchRailOffset(match, terminalSession.terminalSnapshot?.history?.logicalTotalRows ?? 0) }}
            />
          })}
        </div>
      ) : null}
      {historySearchOpen ? (
        <form
          aria-busy={historySearchBusy}
          className="absolute inset-x-0 bottom-0 z-[60] border-t border-[var(--anytty-border)] bg-[var(--anytty-surface)] px-1 py-1 text-[var(--anytty-text)] shadow-[0_-6px_20px_rgba(0,0,0,0.28)]"
          data-testid="anytty-history-search"
          onSubmit={(event) => { event.preventDefault(); void runHistorySearch('forward') }}
          style={{ paddingBottom: 'max(0.25rem, env(safe-area-inset-bottom))' }}
        >
          <div className="flex h-11 min-w-0 items-center gap-1">
            <NativeSelect
              aria-label={t('terminal.searchModes.label')}
              className="h-11 w-11 shrink-0 border-[var(--anytty-border-subtle)] bg-[var(--anytty-bg)] px-1 pr-6 font-mono text-xs font-semibold min-[360px]:w-[4.75rem] min-[360px]:px-2 min-[360px]:pr-7"
              value={historySearchMode}
              onChange={(event) => {
                terminalSession.cancelHistorySearch()
                historySearchScanKeyRef.current = ''
                historySearchLastMatchRef.current = null
                historySearchMatchesRef.current = []
                clearHistorySearchHighlight()
                setHistorySearchMode(event.target.value as TerminalHistorySearchMode)
                setHistorySearchMatches([])
                setHistorySearchActiveMatch(null)
                setHistorySearchScanBusy(false)
                setHistorySearchScanComplete(false)
                setHistorySearchWrapped(false)
                setHistorySearchMessage('')
              }}
            >
              {(['text', 'glob', 'regex'] as const).map((mode) => (
                <option key={mode} value={mode}>{t(`terminal.searchModes.${mode}`)}</option>
              ))}
            </NativeSelect>
            <label className="flex h-11 min-w-8 flex-1 items-center gap-1 rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-bg)] px-2 focus-within:ring-2 focus-within:ring-[var(--anytty-accent)] min-[360px]:min-w-[3.5rem]">
              <Search className="h-4 w-4 shrink-0 text-[var(--anytty-muted)]" aria-hidden="true" />
              <Input
                autoFocus
                aria-label={t('terminal.tools.searchHistory')}
                className="h-10 min-w-0 flex-1 border-0 bg-transparent px-0 font-mono shadow-none focus-visible:ring-0"
                value={historySearchQuery}
                onChange={(event) => {
                  terminalSession.cancelHistorySearch()
                  historySearchScanKeyRef.current = ''
                  historySearchLastMatchRef.current = null
                  historySearchMatchesRef.current = []
                  clearHistorySearchHighlight()
                  setHistorySearchQuery(event.target.value)
                  setHistorySearchMatches([])
                  setHistorySearchActiveMatch(null)
                  setHistorySearchScanBusy(false)
                  setHistorySearchScanComplete(false)
                  setHistorySearchWrapped(false)
                  setHistorySearchMessage('')
                }}
              />
            </label>
            <span
              aria-label={t('terminal.tools.searchResultCount', { current: historySearchActiveIndex, total: historySearchMatches.length })}
              className={`flex min-w-[3.25rem] shrink-0 items-center justify-end gap-1 whitespace-nowrap font-mono text-[11px] tabular-nums ${historySearchMessage && !historySearchBusy ? 'text-amber-400' : 'text-[var(--anytty-muted)]'}`}
              role="status"
            >
              {historySearchWrapped ? <span aria-hidden="true">↻</span> : null}
              {historySearchScanBusy ? <Spinner className="h-3 w-3" aria-hidden="true" /> : null}
              {historySearchCountLabel}
            </span>
            <Button variant="ghost" type="button" aria-label={t('terminal.tools.searchPrevious')} className="grid h-11 w-11 shrink-0 place-items-center rounded-md border border-[var(--anytty-border-subtle)]" disabled={historySearchBusy || !historySearchQuery.trim()} onClick={() => { void hapticSelection(); void runHistorySearch('backward') }}>
              <ArrowUp className="h-4 w-4" />
            </Button>
            <Button variant="ghost" type="submit" aria-label={t('terminal.tools.searchNext')} className="grid h-11 w-11 shrink-0 place-items-center rounded-md border border-[var(--anytty-border-subtle)]" disabled={historySearchBusy || !historySearchQuery.trim()} onClick={() => { void hapticSelection() }}>
              {historySearchBusy ? <Spinner aria-hidden="true" /> : <ArrowDown className="h-4 w-4" />}
            </Button>
            <Button variant="ghost" type="button" aria-label={t('common.close')} className="grid h-11 w-11 shrink-0 place-items-center" onClick={() => {
              terminalSession.cancelHistorySearch()
              historySearchScanKeyRef.current = ''
              historySearchLastMatchRef.current = null
              historySearchMatchesRef.current = []
              clearHistorySearchHighlight()
              setHistorySearchMatches([])
              setHistorySearchActiveMatch(null)
              setHistorySearchScanBusy(false)
              setHistorySearchScanComplete(false)
              setHistorySearchWrapped(false)
              setHistorySearchMessage('')
              setHistorySearchOpen(false)
            }}>
              <X className="h-4 w-4" />
            </Button>
          </div>
          {historySearchMessage ? <span className="sr-only" role="status">{historySearchMessage}</span> : null}
        </form>
      ) : null}
      {historyLoadFailure !== 'none' && !showConnectingOverlay ? (
        <div
          className={`absolute inset-x-2 z-[60] flex min-h-11 items-center justify-between gap-3 rounded-md border border-amber-500/40 bg-[var(--anytty-surface)] px-3 py-2 text-sm text-[var(--anytty-text)] ${inputFailureVisible ? 'top-14' : 'top-3'}`}
          data-testid="anytty-history-error"
          role="alert"
        >
          <span>{t(historyLoadFailure === 'line-too-large' ? 'terminal.tools.historyLineTooLarge' : 'terminal.tools.historyUnavailable')}</span>
          <Button variant="ghost"
            type="button"
            className="min-h-11 shrink-0 rounded-md border border-[var(--anytty-border)] px-3 font-semibold focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2"
            onClick={historyLoadFailure === 'line-too-large'
              ? () => setHistoryLoadFailure('none')
              : () => retryHistoryLoadRef.current()}
          >
            {t(historyLoadFailure === 'line-too-large' ? 'terminal.tools.dismissHistoryError' : 'terminal.tools.reloadHistory')}
          </Button>
        </div>
      ) : null}
    </section>
  )
})

function formatHistoryTimestamp(timestampUnixMs: number): string {
  return new Date(timestampUnixMs).toLocaleString(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
