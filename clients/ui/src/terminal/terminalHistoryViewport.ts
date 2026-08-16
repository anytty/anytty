export type TerminalHistoryViewportPhase = 'live' | 'primed' | 'entering' | 'frozen'

export function terminalViewportAtBottom(viewportY: number, baseY: number): boolean {
  return Number.isFinite(viewportY) && Number.isFinite(baseY) && viewportY >= baseY
}

/** Keeps live transport updates separate from the xterm viewport while history is visible. */
export class TerminalHistoryViewportController {
  private phase: TerminalHistoryViewportPhase = 'live'
  private deferredLiveUpdate = false

  get isFrozen(): boolean {
    return this.phase === 'entering' || this.phase === 'frozen'
  }

  get isPrimed(): boolean {
    return this.phase === 'primed'
  }

  get isLiveUpdateDeferred(): boolean {
    return this.phase !== 'live'
  }

  get hasDeferredLiveUpdate(): boolean {
    return this.deferredLiveUpdate
  }

  prime(): void {
    if (this.phase === 'live') this.phase = 'primed'
  }

  enterHistory(): boolean {
    if (this.phase !== 'primed') return false
    this.phase = 'entering'
    return true
  }

  confirmHistoryMovement(atBottom: boolean): boolean {
    if (this.phase !== 'entering' || atBottom) return false
    this.phase = 'frozen'
    return true
  }

  shouldRenderLiveUpdate(): boolean {
    if (this.phase === 'live') return true
    this.deferredLiveUpdate = true
    return false
  }

  resumeAtBottom(atBottom: boolean, busy: boolean, includePrimed = false): boolean {
    const canResume = this.phase === 'frozen' || (includePrimed && this.phase !== 'live')
    if (!canResume || !atBottom || busy) return false
    this.phase = 'live'
    this.deferredLiveUpdate = false
    return true
  }

  reset(): void {
    this.phase = 'live'
    this.deferredLiveUpdate = false
  }
}

export function historyViewportAfterApply(input: {
  operation: 'replace' | 'prepend'
  previouslyAppliedRows: number
  restoreViewportY: number
  actualPrependedRows: number
  fallbackPrependedRows: number
  bufferLength: number
  viewportRows: number
  initialViewportTop?: number | undefined
}): number {
  const rowDelta = input.actualPrependedRows > 0
    ? input.actualPrependedRows
    : input.fallbackPrependedRows
  if (input.operation === 'replace' && input.previouslyAppliedRows === 0) {
    if (input.initialViewportTop !== undefined) {
      return Math.max(0, Math.min(input.initialViewportTop, input.bufferLength - input.viewportRows))
    }
    return Math.max(0, input.bufferLength - input.viewportRows)
  }
  return Math.max(0, input.restoreViewportY + rowDelta)
}

export function historyReplayWithViewportTail(
  text: string,
  loadedRows: number,
  viewportTop: number | undefined,
  viewportRows: number,
): string {
  if (viewportTop === undefined || viewportRows <= 0) return text
  const rowsBelowTop = Math.max(0, loadedRows - viewportTop)
  const tailRows = Math.max(0, viewportRows - rowsBelowTop)
  return tailRows > 0 ? `${text}${'\r\n'.repeat(tailRows)}` : text
}

export function terminalScrollLineDelta(desiredViewportY: number, currentViewportY: number): number {
  if (!Number.isFinite(desiredViewportY) || !Number.isFinite(currentViewportY)) return 0
  return Math.round(desiredViewportY) - Math.round(currentViewportY)
}

export function historyRequestAwaitingApply(requestedRows: number, appliedRows: number): boolean {
  return requestedRows > appliedRows
}
