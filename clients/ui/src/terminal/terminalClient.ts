import { normalizeTerminal, type Terminal } from '../core/model'
import type { ConnectionInfo } from '../core/transport'
import type { ConnectionMessage } from '../connection/connectionMessageReducer'
import { logTerminalDiagnostic } from './terminalDiagnostics'

/**
 * TerminalSnapshotPayload 是客户端消费 owning daemon 权威屏幕的只读投影。
 * `refreshReason` 只描述本次客户端拉取链路，不能作为 terminal lifecycle、history 或屏幕内容真值；
 * live revision 只表示 daemon canonical screen 版本；renderer completion 通过独立方法回传。
 */
export interface TerminalSnapshotPayload {
  text: string
  cols: number
  rows: number
  replay?: string
  screenReplay?: string
  screenText?: string
  liveReplay?: string
  liveRevision?: bigint
  liveFullReplace?: boolean
  raw?: unknown
  scrollbackRows?: unknown[]
  alternateScreen?: boolean
  refreshReason?: 'open' | 'live_screen' | 'sync_lost' | 'manual_sync_lost'
  recovery?: {
    revision: number
    reason: string
    syncLostCount?: number
    droppedBytes?: number
  }
  history?: {
    revision: number
    cols: number
    prependedRows: number
    loadedRows: number
    operation?: 'replace' | 'prepend'
    committedTotalRows?: number
    logicalTotalRows?: number
    historyGeneration?: number
    firstRowId?: number
    lastRowId?: number
    viewportTop?: number
    rowTimestampsUnixMs?: Array<number | undefined>
    rowLogicalLineIds?: Array<string | undefined>
    rowInLogicalLines?: Array<number | undefined>
    rowLogicalStartCols?: Array<number | undefined>
    searchMatchRow?: number
    searchMatchRanges?: TerminalHistoryMatchRange[]
    hasMore: boolean
    alternate?: boolean
    prefetched?: boolean
  }
}

export interface TerminalScrollbackPage {
  beforeOffset: number
  limit: number
  rows: number
  cols: number
  replay: string
  operation?: 'replace' | 'prepend'
  committedTotalRows?: number
  logicalTotalRows?: number
  historyGeneration?: number
  firstRowId?: number
  lastRowId?: number
  viewportTop?: number
  rowTimestampsUnixMs?: Array<number | undefined>
  rowLogicalLineIds?: Array<string | undefined>
  rowInLogicalLines?: Array<number | undefined>
  rowLogicalStartCols?: Array<number | undefined>
  hasMore: boolean
  alternate: boolean
}

export interface TerminalScrollbackLoadResult {
  loadedRows: number
  totalRows: number
  cols?: number
  operation?: 'replace' | 'prepend'
  committedTotalRows?: number
  logicalTotalRows?: number
  historyGeneration?: number
  firstRowId?: number
  lastRowId?: number
  viewportTop?: number
  rowTimestampsUnixMs?: Array<number | undefined>
  rowLogicalLineIds?: Array<string | undefined>
  rowInLogicalLines?: Array<number | undefined>
  rowLogicalStartCols?: Array<number | undefined>
  hasMore: boolean
  alternate: boolean
  prefetched?: boolean
}

export interface TerminalHistorySearchResult {
  found: boolean
  wrapped: boolean
  page?: TerminalScrollbackPage | undefined
  match?: { startLineId: string; startCol: number; endLineId: string; endCol: number } | undefined
  matchRow?: number | undefined
  matchRanges?: TerminalHistoryMatchRange[] | undefined
}

export interface TerminalHistoryMatchRange {
  row: number
  startCol: number
  endCol: number
}

export type TerminalHistorySearchMode = 'text' | 'glob' | 'regex'

export type TerminalInfoPayload = Record<string, unknown>

export type TerminalResizePolicy = 'owner' | 'follower'
export type TerminalResizeControlReason = 'owner' | 'follower' | 'observer' | 'size_locked' | 'unknown'

export interface TerminalResizeControl {
  canResize: boolean
  reason: TerminalResizeControlReason
  sizeLocked?: boolean
  surfaceId?: string
  ownerSurfaceId?: string
  ownerViewId?: string
}

export const defaultTerminalResizeControl: TerminalResizeControl = {
  canResize: false,
  reason: 'unknown',
}

export function terminalResizeControlOwnsResize(control: TerminalResizeControl | undefined): boolean {
  return control?.canResize === true || control?.reason === 'size_locked'
}

export type TerminalProtocolEvent =
  | { type: 'output'; data: Uint8Array }
  | { type: 'snapshot'; snapshot: TerminalSnapshotPayload }
  | { type: 'info'; info: TerminalInfoPayload }
  | { type: 'resizeControl'; control: TerminalResizeControl }
  | { type: 'closed'; reason?: string }

export interface TerminalProtocolChannel {
  readonly label: string
  readonly readyState: 'connecting' | 'open' | 'closing' | 'closed'
  send(data: Uint8Array): void
  sendInput?(data: string, size?: TerminalInputSize): void
  cancelPendingMouseInput?(): void
  sendResize?(cols: number, rows: number): void
  close(): void
}

export interface TerminalInputSize {
  cols: number
  rows: number
}

export interface TerminalProtocolSession {
  openTerminal(terminalId: string): Promise<TerminalProtocolChannel>
  getConnectionInfo(): Promise<ConnectionInfo>
  subscribeTerminal(terminalId: string, handler: (event: TerminalProtocolEvent) => void): () => void
  loadScrollback(
    terminalId: string,
    offset: number,
    limit: number,
    alternate?: boolean,
    options?: { signal?: AbortSignal; cols?: number },
  ): Promise<TerminalScrollbackPage>
  searchScrollback?(
    terminalId: string,
    query: string,
    direction: 'forward' | 'backward',
    cols: number,
    limit: number,
    start?: { lineId: string; col: number } | undefined,
    options?: { signal?: AbortSignal | undefined; mode?: TerminalHistorySearchMode | undefined },
  ): Promise<TerminalHistorySearchResult>
  copyScrollback?(
    terminalId: string,
    range: { startLineId: string; startCol: number; endLineId: string; endCol: number },
    cols: number,
    options?: { signal?: AbortSignal | undefined },
  ): Promise<string>
  resetScrollback?(terminalId: string): void
  closeTerminalChannel(terminalId: string): void
  markSyncLost?(terminalId: string, reason?: string): void
  setLiveScreenDemand?(terminalId: string, enabled: boolean): void
  markLiveScreenSubmitted?(terminalId: string, revision: bigint): void
  markLiveScreenCompleted?(terminalId: string, revision: bigint): void
  requestResizeOwner?(terminalId: string, size?: TerminalInputSize): Promise<TerminalResizeControl>
  releaseResizeOwner?(terminalId: string): Promise<TerminalResizeControl>
  setResizeLock?(terminalId: string, locked: boolean): Promise<TerminalResizeControl>
}

export interface TerminalClientCallbacks {
  onOutput: (data: Uint8Array) => void
  onSnapshot: (snapshot: TerminalSnapshotPayload) => void
  onTerminalInfo?: (info: Terminal) => void
  onResizeControl?: (control: TerminalResizeControl) => void
  onLifecycle?: (message: Extract<ConnectionMessage, { type: 'terminal.channelOpen' | 'terminal.channelClosed' }>) => void
  onError: (message: string) => void
  onClose: (reason?: string) => void
  onOpen?: () => void
  onInputDropped?: () => void
  onInputSendFailed?: (message: string) => void
}

export class TerminalClient {
  private callbacks: TerminalClientCallbacks
  private session: TerminalProtocolSession | null = null
  private terminalId = ''
  private channel: TerminalProtocolChannel | null = null
  private unsubscribe: (() => void) | null = null
  private machineId = ''
  private resizeControl: TerminalResizeControl = defaultTerminalResizeControl
  private bindingGeneration = 0
  private historyOnly = false
  private readonly maxOpenAttempts = 2

  constructor(callbacks: TerminalClientCallbacks) {
    this.callbacks = callbacks
  }

  connect(terminalId: string, session: TerminalProtocolSession): void {
    this.historyOnly = false
    this.log('connect_start', { terminalId })
    void this.bind(terminalId, session, { closePrevious: true }).catch(() => {})
  }

  async connectHistory(terminalId: string, session: TerminalProtocolSession): Promise<void> {
    if (!terminalId) throw new Error('terminalId is required')
    this.cleanupSubscription()
    if (this.session && this.terminalId && !this.historyOnly) {
      this.session.closeTerminalChannel(this.terminalId)
    }
    this.session = session
    this.terminalId = terminalId
    this.channel = null
    this.historyOnly = true
    const generation = ++this.bindingGeneration
    const machineId = await this.resolveMachineId(session)
    if (!this.isCurrentBinding(session, terminalId, generation)) return
    this.machineId = machineId
    this.log('history_bound', { machineId, terminalId })
  }

  reattach(session: TerminalProtocolSession): Promise<void> {
    if (!this.terminalId) {
      this.callbacks.onError('terminal client is not connected')
      return Promise.resolve()
    }
    return this.bind(this.terminalId, session, { closePrevious: false })
  }

  disconnect(): void {
    this.bindingGeneration += 1
    const terminalId = this.terminalId
    this.cleanupSubscription()
    if (this.session && terminalId && !this.historyOnly) {
      this.session.closeTerminalChannel(terminalId)
    }
    this.channel = null
    this.session = null
    this.terminalId = ''
    this.historyOnly = false
  }

  sendInput(data: string, size?: TerminalInputSize): boolean {
    const sent = this.sendMessage(
      { type: 'input', data, ...(size ? { cols: size.cols, rows: size.rows } : {}) },
      { reportInputFailure: true },
    )
    this.log('send_input', {
      terminalId: this.terminalId,
      details: {
        sent,
        chars: data.length,
        size,
      },
    })
    return sent
  }

  cancelPendingMouseInput(): boolean {
    const channel = this.channel
    if (!channel || channel.readyState !== 'open' || !channel.cancelPendingMouseInput) return false
    channel.cancelPendingMouseInput()
    return true
  }

  sendResize(cols: number, rows: number): boolean {
    if (!this.resizeControl.canResize) {
      this.log('send_resize_skipped', {
        terminalId: this.terminalId,
        details: { cols, rows, resizeControl: this.resizeControl },
      })
      return false
    }
    const sent = this.sendMessage({ type: 'resize', cols, rows }, { reportInputFailure: false })
    this.log('send_resize', {
      terminalId: this.terminalId,
      details: { sent, cols, rows, resizeControl: this.resizeControl },
    })
    return sent
  }

  async requestResizeOwner(size?: TerminalInputSize): Promise<TerminalResizeControl> {
    if (!this.session || !this.terminalId || !this.session.requestResizeOwner) {
      throw new Error('terminal resize ownership is not available')
    }
    const control = await this.session.requestResizeOwner(this.terminalId, size)
    this.resizeControl = control
    this.callbacks.onResizeControl?.(control)
    return control
  }

  async releaseResizeOwner(): Promise<TerminalResizeControl> {
    if (!this.session || !this.terminalId || !this.session.releaseResizeOwner) {
      throw new Error('terminal resize ownership is not available')
    }
    const control = await this.session.releaseResizeOwner(this.terminalId)
    this.resizeControl = control
    this.callbacks.onResizeControl?.(control)
    return control
  }

  async setResizeLock(locked: boolean): Promise<TerminalResizeControl> {
    if (!this.session || !this.terminalId || !this.session.setResizeLock) {
      throw new Error('terminal resize lock is not available')
    }
    const control = await this.session.setResizeLock(this.terminalId, locked)
    this.resizeControl = control
    this.callbacks.onResizeControl?.(control)
    return control
  }

  loadScrollback(
    offset: number,
    limit: number,
    alternate = false,
    options?: { signal?: AbortSignal; cols?: number },
  ): Promise<TerminalScrollbackPage> {
    if (!this.session || !this.terminalId) {
      return Promise.reject(new Error('terminal client is not connected'))
    }
    return this.session.loadScrollback(this.terminalId, offset, limit, alternate, options)
  }

  searchScrollback(
    query: string,
    direction: 'forward' | 'backward',
    cols: number,
    limit: number,
    start?: { lineId: string; col: number } | undefined,
    options?: { signal?: AbortSignal | undefined; mode?: TerminalHistorySearchMode | undefined },
  ): Promise<TerminalHistorySearchResult> {
    if (!this.session || !this.terminalId || !this.session.searchScrollback) {
      return Promise.reject(new Error('terminal history search is not available'))
    }
    return this.session.searchScrollback(this.terminalId, query, direction, cols, limit, start, options)
  }

  copyScrollback(
    range: { startLineId: string; startCol: number; endLineId: string; endCol: number },
    cols: number,
    options?: { signal?: AbortSignal | undefined },
  ): Promise<string> {
    if (!this.session || !this.terminalId || !this.session.copyScrollback) {
      return Promise.reject(new Error('terminal history copy is not available'))
    }
    return this.session.copyScrollback(this.terminalId, range, cols, options)
  }

  resetScrollback(): void {
    if (!this.session || !this.terminalId) return
    this.session.resetScrollback?.(this.terminalId)
  }

  markSyncLost(reason?: string): void {
    if (!this.session || !this.terminalId) return
    this.session.markSyncLost?.(this.terminalId, reason)
  }

  setLiveScreenDemand(enabled: boolean): void {
    if (!this.session || !this.terminalId) return
    this.session.setLiveScreenDemand?.(this.terminalId, enabled)
  }

  markLiveScreenSubmitted(revision: bigint): void {
    if (!this.session || !this.terminalId) return
    this.session.markLiveScreenSubmitted?.(this.terminalId, revision)
  }

  markLiveScreenCompleted(revision: bigint): void {
    if (!this.session || !this.terminalId) return
    this.session.markLiveScreenCompleted?.(this.terminalId, revision)
  }

  private bind(
    terminalId: string,
    session: TerminalProtocolSession,
    options: { closePrevious: boolean },
  ): Promise<void> {
    if (!terminalId) {
      throw new Error('terminalId is required')
    }

    if (options.closePrevious) {
      this.cleanupSubscription()
      if (this.session && this.terminalId) {
        this.session.closeTerminalChannel(this.terminalId)
      }
    } else {
      this.cleanupSubscription()
    }

    this.session = session
    this.terminalId = terminalId
    this.historyOnly = false
    const generation = ++this.bindingGeneration

    const openPromise = this.openForBinding(session, terminalId, generation, 1).then(([machineId, channel]) => {
      if (!this.isCurrentBinding(session, terminalId, generation)) {
        channel.close()
        return
      }
      this.machineId = machineId
      if (channel.label !== `proto-terminal:${terminalId}`) {
        channel.close()
        this.callbacks.onError(`unexpected terminal channel label ${channel.label}`)
        throw new Error(`unexpected terminal channel label ${channel.label}`)
      }
      this.channel = channel
      this.log('channel_bound', {
        machineId,
        terminalId,
        channelLabel: channel.label,
        details: { readyState: channel.readyState, generation },
      })
      this.callbacks.onOpen?.()
      this.callbacks.onLifecycle?.({
        type: 'terminal.channelOpen',
        machineId,
        terminalId,
      })
      return
    }).catch((err: unknown) => {
      if (!this.isCurrentBinding(session, terminalId, generation)) return
      this.log('bind_failed', {
        terminalId,
        level: 'error',
        details: { reason: errorMessage(err), generation },
      })
      this.callbacks.onError(errorMessage(err))
      this.callbacks.onClose()
    })

    this.unsubscribe = session.subscribeTerminal(terminalId, (event) => {
      if (!this.isCurrentBinding(session, terminalId, generation)) return
      this.handleProtocolEvent(event)
    })
    return openPromise
  }

  private async openForBinding(
    session: TerminalProtocolSession,
    terminalId: string,
    generation: number,
    attempt: number,
  ): Promise<[string, TerminalProtocolChannel]> {
    try {
      return await Promise.all([
        this.resolveMachineId(session),
        session.openTerminal(terminalId),
      ])
    } catch (err) {
      if (!this.isCurrentBinding(session, terminalId, generation) || attempt >= this.maxOpenAttempts) throw err
      return this.openForBinding(session, terminalId, generation, attempt + 1)
    }
  }

  private isCurrentBinding(session: TerminalProtocolSession, terminalId: string, generation: number): boolean {
    return this.session === session && this.terminalId === terminalId && this.bindingGeneration === generation
  }

  private handleProtocolEvent(event: TerminalProtocolEvent): void {
    this.log('protocol_event', {
      terminalId: this.terminalId,
      details: protocolEventDetails(event),
    })
    switch (event.type) {
      case 'output':
        this.callbacks.onOutput(event.data)
        return
      case 'snapshot':
        this.callbacks.onSnapshot(event.snapshot)
        return
      case 'info':
        try {
          this.callbacks.onTerminalInfo?.(normalizeTerminal(event.info))
        } catch (err) {
          this.callbacks.onError(errorMessage(err))
        }
        return
      case 'resizeControl':
        this.resizeControl = event.control
        this.callbacks.onResizeControl?.(event.control)
        return
      case 'closed':
        this.callbacks.onLifecycle?.({
          type: 'terminal.channelClosed',
          machineId: this.machineId || 'machine-local',
          terminalId: this.terminalId,
          ...(event.reason ? { reason: event.reason } : {}),
        })
        this.callbacks.onClose(event.reason)
        return
    }
  }

  private sendMessage(
    message: { type: 'input'; data: string; cols?: number; rows?: number } | { type: 'resize'; cols: number; rows: number },
    options: { reportInputFailure: boolean },
  ): boolean {
    const channel = this.channel
    if (!channel || channel.readyState !== 'open') {
      this.log('send_message_no_channel', {
        terminalId: this.terminalId,
        level: options.reportInputFailure ? 'warn' : 'debug',
        details: {
          type: message.type,
          readyState: channel?.readyState,
        },
      })
      if (options.reportInputFailure) {
        this.callbacks.onInputDropped?.()
        this.callbacks.onInputSendFailed?.('terminal channel is not open')
      }
      return false
    }

    try {
	  if (message.type === 'input' && channel.sendInput) channel.sendInput(message.data, message.cols && message.rows ? { cols: message.cols, rows: message.rows } : undefined)
	  else if (message.type === 'resize' && channel.sendResize) channel.sendResize(message.cols, message.rows)
	  else channel.send(new TextEncoder().encode(JSON.stringify(message)))
      return true
    } catch (err) {
      this.log('send_message_failed', {
        terminalId: this.terminalId,
        level: options.reportInputFailure ? 'warn' : 'debug',
        details: {
          type: message.type,
          reason: errorMessage(err),
        },
      })
      if (options.reportInputFailure) {
        this.callbacks.onInputDropped?.()
        this.callbacks.onInputSendFailed?.(errorMessage(err))
      }
      return false
    }
  }

  private cleanupSubscription(): void {
    this.unsubscribe?.()
    this.unsubscribe = null
  }

  private async resolveMachineId(session: TerminalProtocolSession): Promise<string> {
    const info: ConnectionInfo = await session.getConnectionInfo()
    return info.machineId
  }

  private log(event: string, input: {
    level?: 'debug' | 'info' | 'warn' | 'error'
    machineId?: string
    terminalId?: string
    channelLabel?: string
    details?: Record<string, unknown>
  } = {}): void {
    logTerminalDiagnostic(`client.${event}`, {
      level: input.level,
      machineId: input.machineId,
      terminalId: input.terminalId,
      channelLabel: input.channelLabel,
      details: input.details,
    })
  }
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err)
}

function protocolEventDetails(event: TerminalProtocolEvent): Record<string, unknown> {
  switch (event.type) {
    case 'output':
      return { type: event.type, bytes: event.data.byteLength }
    case 'snapshot':
      return {
        type: event.type,
        textChars: event.snapshot.text.length,
        replayChars: event.snapshot.replay?.length ?? 0,
        screenTextChars: event.snapshot.screenText?.length ?? 0,
        screenReplayChars: event.snapshot.screenReplay?.length ?? 0,
        rows: event.snapshot.rows,
        cols: event.snapshot.cols,
        alternateScreen: event.snapshot.alternateScreen === true,
      }
    case 'info':
      return { type: event.type, terminalId: event.info.id, status: event.info.status }
    case 'resizeControl':
      return { type: event.type, control: event.control }
    case 'closed':
      return { type: event.type, reason: event.reason }
  }
}
