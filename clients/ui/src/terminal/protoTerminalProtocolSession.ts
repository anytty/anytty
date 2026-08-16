import { create } from '@bufbuild/protobuf'
import type { ProtoClientSession, ProtoClientSubscription } from '../core/protoClientSession'
import { openProtoEventSubscription } from '../core/protoEventSubscription'
import { CommandEnvelopeSchema } from '../generated/apipb/application_pb'
import { ApplicationEventType, EventSubscribeCommandSchema } from '../generated/apipb/events_pb'
import {
  CursorShape,
  LiveScreenNextCommandSchema,
  type CellStyle,
  type ScreenRow,
  type TerminalCursor,
  type TerminalModes,
} from '../generated/apipb/history_pb'
import {
  AttachmentMode,
  ResizeControlReason,
  ResizePolicy,
  TerminalAttachCommandSchema,
  TerminalDetachCommandSchema,
  TerminalGetCommandSchema,
  TerminalInputCommandSchema,
  TerminalRefSchema,
  TerminalResizeCommandSchema,
  TerminalSizeSchema,
  TerminalState,
  type AttachmentHandle,
  type ResizeControl,
  type TerminalInfo,
} from '../generated/apipb/terminal_pb'
import type {
  TerminalInputSize,
  TerminalHistorySearchResult,
  TerminalProtocolChannel,
  TerminalProtocolEvent,
  TerminalProtocolSession,
  TerminalResizeControl,
  TerminalScrollbackPage,
  TerminalSnapshotPayload,
} from './terminalClient'
import { coreV2HistoryRowsANSI } from './coreV2HistoryANSI'
import { createCoreV2HistorySource } from './coreV2HistorySource'
import { CoreV2ScrollbackPager } from './coreV2ScrollbackPager'
import {
  mergeLiveScreenResult,
  type CanonicalLiveScreen,
  type LiveScreenDamage,
} from './liveScreenCache'

type ProtoAttachment = {
  handle: AttachmentHandle
  channel: ProtoTerminalChannel
}

type LiveScreenDeliveryState = {
  received: CanonicalLiveScreen | undefined
  damage: LiveScreenDamage | undefined
  publishedRevision: bigint | undefined
  renderingRevision: bigint | undefined
  requestController: AbortController | undefined
  demand: boolean
  reason: TerminalSnapshotPayload['refreshReason']
}

/** createProtoTerminalProtocolSession projects generated apipb into the existing UI-only TerminalClient contract. */
export function createProtoTerminalProtocolSession(session: ProtoClientSession): TerminalProtocolSession {
  return new ProtoTerminalProtocolSession(session)
}

class ProtoTerminalProtocolSession implements TerminalProtocolSession {
  private readonly attachments = new Map<string, ProtoAttachment>()
  private readonly subscribers = new Map<string, Set<(event: TerminalProtocolEvent) => void>>()
  private readonly terminalSizes = new Map<string, TerminalInputSize>()
  private readonly inputTails = new Map<string, Promise<void>>()
  private readonly eventSubscriptions = new Map<string, Promise<ProtoClientSubscription>>()
  private readonly liveScreens = new Map<string, LiveScreenDeliveryState>()
  private readonly scrollbackPager: CoreV2ScrollbackPager
  private documentVisible = typeof document === 'undefined' || document.visibilityState !== 'hidden'
  private readonly handleVisibilityChange = () => {
    const visible = typeof document === 'undefined' || document.visibilityState !== 'hidden'
    if (visible === this.documentVisible) return
    this.documentVisible = visible
    for (const [terminalId, state] of this.liveScreens) {
      if (!visible) {
        this.pauseLiveScreen(state, 'terminal view hidden')
        continue
      }
      this.resumeLiveScreen(terminalId, state)
    }
  }

  constructor(private readonly session: ProtoClientSession) {
    this.scrollbackPager = new CoreV2ScrollbackPager(createCoreV2HistorySource(session, session.stamp.endpointId))
  }

  private eventSubscription(terminalId: string): Promise<ProtoClientSubscription> {
    const existing = this.eventSubscriptions.get(terminalId)
    if (existing) return existing
    const opening = openProtoEventSubscription(this.session, create(EventSubscribeCommandSchema, {
      terminal: this.terminalRef(terminalId),
      types: [ApplicationEventType.TERMINAL_LIFECYCLE],
    }), (event) => {
      if (event.event.case === 'terminalLifecycle') {
        const terminal = event.event.value.terminal
        const terminalId = terminal?.ref?.terminalId
        if (terminal && terminalId) {
          this.rememberSize(terminalId, terminal.size)
          this.publish(terminalId, { type: 'info', info: terminalInfoView(terminal) })
        }
      }
    })
    this.eventSubscriptions.set(terminalId, opening)
    void opening.catch(() => {
      if (this.eventSubscriptions.get(terminalId) === opening) this.eventSubscriptions.delete(terminalId)
    })
    return opening
  }

  private closeEventSubscription(terminalId: string): void {
    const opening = this.eventSubscriptions.get(terminalId)
    if (!opening) return
    this.eventSubscriptions.delete(terminalId)
    void opening.then((subscription) => subscription.close()).catch(() => undefined)
  }

  async openTerminal(terminalId: string): Promise<TerminalProtocolChannel> {
    await this.eventSubscription(terminalId)
    const existing = this.attachments.get(terminalId)
    if (existing) return existing.channel
    const terminal = this.terminalRef(terminalId)
    const attach = create(TerminalAttachCommandSchema, {
      terminal,
      mode: AttachmentMode.COLLABORATOR,
      resizePolicy: ResizePolicy.OWNER,
      surfaceId: `mobile-${terminalId}`,
      viewId: crypto.randomUUID(),
    })
    const result = await this.session.execute(command('terminalAttach', attach))
    if (result.result.case !== 'terminalAttach' || !result.result.value.attachment?.resource) {
      throw new Error('terminal attach returned no attachment resource')
    }
    const channel = new ProtoTerminalChannel(this, terminalId)
    const firstAttachment = this.attachments.size === 0
    this.attachments.set(terminalId, { handle: result.result.value.attachment, channel })
    if (firstAttachment && typeof document !== 'undefined') {
      this.documentVisible = document.visibilityState !== 'hidden'
      document.addEventListener('visibilitychange', this.handleVisibilityChange)
    }
    const liveState: LiveScreenDeliveryState = {
      received: undefined,
      damage: undefined,
      publishedRevision: undefined,
      renderingRevision: undefined,
      requestController: undefined,
      demand: true,
      reason: 'open',
    }
    this.liveScreens.set(terminalId, liveState)
    this.publish(terminalId, { type: 'resizeControl', control: resizeControlView(result.result.value.resizeControl) })
    await Promise.all([this.publishTerminalInfo(terminalId), this.startLiveScreenRequest(terminalId, liveState)])
    return channel
  }

  async getConnectionInfo() {
    return {
      path: 'hub' as const,
      connectionId: `${this.session.stamp.endpointId}:${this.session.stamp.generation}`,
      machineId: this.session.stamp.endpointId,
      relayInUse: false,
    }
  }

  subscribeTerminal(terminalId: string, handler: (event: TerminalProtocolEvent) => void): () => void {
    const handlers = this.subscribers.get(terminalId) ?? new Set()
    handlers.add(handler)
    this.subscribers.set(terminalId, handlers)
    return () => {
      handlers.delete(handler)
      if (handlers.size === 0) this.subscribers.delete(terminalId)
    }
  }

  async loadScrollback(
    terminalId: string,
    offset: number,
    limit: number,
    alternate = false,
    options?: { signal?: AbortSignal; cols?: number },
  ): Promise<TerminalScrollbackPage> {
    const requestedCols = Math.trunc(options?.cols ?? 0)
    const cols = requestedCols > 0 ? requestedCols : this.terminalSizes.get(terminalId)?.cols ?? 80
    const page = await this.scrollbackPager.load({
      terminalId,
      offset,
      limit,
      cols,
      ...(options?.signal ? { signal: options.signal } : {}),
    })
    return {
      beforeOffset: offset,
      limit,
      rows: page.loadedRows,
      cols,
      replay: coreV2HistoryRowsANSI(page.rows, cols),
      operation: page.operation,
      committedTotalRows: page.committedTotalRows,
      logicalTotalRows: page.logicalTotalRows,
      historyGeneration: Number(page.historyGeneration),
      ...(page.firstRowId ? { firstRowId: Number(page.firstRowId) } : {}),
      ...(page.lastRowId ? { lastRowId: Number(page.lastRowId) } : {}),
      ...(page.viewportTop !== undefined ? { viewportTop: page.viewportTop } : {}),
      rowTimestampsUnixMs: page.rows.map((row) => row.timestampUnixMs),
      rowLogicalLineIds: page.rows.map((row) => row.logicalLineId),
      rowInLogicalLines: page.rows.map((row) => row.rowInLine),
      rowLogicalStartCols: page.rows.map((row) => row.logicalStartCol),
      hasMore: page.hasMore,
      alternate,
    }
  }

  async searchScrollback(
    terminalId: string,
    query: string,
    direction: 'forward' | 'backward',
    cols: number,
    limit: number,
    start?: { lineId: string; col: number } | undefined,
    options?: { signal?: AbortSignal | undefined },
  ): Promise<TerminalHistorySearchResult> {
    const result = await this.scrollbackPager.search({
      terminalId,
      query,
      direction,
      cols,
      limit,
      ...(start ? { start } : {}),
      ...(options?.signal ? { signal: options.signal } : {}),
    })
    if (!result.found || !result.page) return { found: false, wrapped: false }
    return {
      found: true,
      wrapped: result.wrapped,
      match: result.match,
      matchRow: result.matchRow,
      page: {
        beforeOffset: 0,
        limit,
        rows: result.page.loadedRows,
        cols,
        replay: coreV2HistoryRowsANSI(result.page.rows, cols),
        operation: 'replace',
        committedTotalRows: result.page.committedTotalRows,
        logicalTotalRows: result.page.logicalTotalRows,
        historyGeneration: Number(result.page.historyGeneration),
        ...(result.page.firstRowId ? { firstRowId: Number(result.page.firstRowId) } : {}),
        ...(result.page.lastRowId ? { lastRowId: Number(result.page.lastRowId) } : {}),
        ...(result.page.viewportTop !== undefined ? { viewportTop: result.page.viewportTop } : {}),
        rowTimestampsUnixMs: result.page.rows.map((row) => row.timestampUnixMs),
        rowLogicalLineIds: result.page.rows.map((row) => row.logicalLineId),
        rowInLogicalLines: result.page.rows.map((row) => row.rowInLine),
        rowLogicalStartCols: result.page.rows.map((row) => row.logicalStartCol),
        hasMore: result.page.hasMore,
        alternate: false,
      },
    }
  }

  copyScrollback(
    terminalId: string,
    range: { startLineId: string; startCol: number; endLineId: string; endCol: number },
    cols: number,
    options?: { signal?: AbortSignal | undefined },
  ): Promise<string> {
    return this.scrollbackPager.copy(terminalId, cols, range, options?.signal)
  }

  resetScrollback(terminalId: string): void {
    this.scrollbackPager.forget(terminalId)
  }

  closeTerminalChannel(terminalId: string): void {
    this.scrollbackPager.forget(terminalId)
    this.closeEventSubscription(terminalId)
    const liveState = this.liveScreens.get(terminalId)
    if (liveState) {
      this.pauseLiveScreen(liveState, 'terminal channel closed')
      this.liveScreens.delete(terminalId)
    }
    const attachment = this.attachments.get(terminalId)
    if (!attachment) return
    this.attachments.delete(terminalId)
    attachment.channel.markClosed()
    void this.session.execute(command('terminalDetach', create(TerminalDetachCommandSchema, {
      attachment: attachment.handle.resource,
    }))).catch(() => undefined)
    if (this.attachments.size === 0 && typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', this.handleVisibilityChange)
    }
  }

  markSyncLost(terminalId: string): void {
    const state = this.liveScreens.get(terminalId)
    if (!state) return
    this.pauseLiveScreen(state, 'terminal live screen sync lost')
    state.received = undefined
    state.damage = undefined
    state.publishedRevision = undefined
    state.reason = 'manual_sync_lost'
    void this.startLiveScreenRequest(terminalId, state)
  }

  setLiveScreenDemand(terminalId: string, enabled: boolean): void {
    const state = this.liveScreens.get(terminalId)
    if (!state || state.demand === enabled) return
    state.demand = enabled
    if (!enabled) {
      this.pauseLiveScreen(state, 'terminal live screen demand disabled')
      return
    }
    this.resumeLiveScreen(terminalId, state)
  }

  markLiveScreenSubmitted(terminalId: string, revision: bigint): void {
    const state = this.liveScreens.get(terminalId)
    if (!state || state.publishedRevision !== revision || state.renderingRevision !== undefined) return
    state.publishedRevision = undefined
    state.renderingRevision = revision
    state.reason = 'live_screen'
    void this.startLiveScreenRequest(terminalId, state)
  }

  markLiveScreenCompleted(terminalId: string, revision: bigint): void {
    const state = this.liveScreens.get(terminalId)
    if (!state || state.renderingRevision !== revision) return
    state.renderingRevision = undefined
    if (state.damage) this.publishReceivedScreen(terminalId, state)
  }

  private pauseLiveScreen(state: LiveScreenDeliveryState, reason: string): void {
    state.requestController?.abort(new DOMException(reason, 'AbortError'))
    state.requestController = undefined
    if (state.publishedRevision !== undefined && state.renderingRevision === undefined && state.received) {
      state.publishedRevision = undefined
      state.damage = fullScreenDamage(state.received.rows)
    }
  }

  private resumeLiveScreen(terminalId: string, state: LiveScreenDeliveryState): void {
    if (!state.demand || !this.documentVisible) return
    if (state.damage && state.publishedRevision === undefined && state.renderingRevision === undefined) {
      this.publishReceivedScreen(terminalId, state)
      return
    }
    void this.startLiveScreenRequest(terminalId, state)
  }

  requestResizeOwner(terminalId: string, size?: TerminalInputSize): Promise<TerminalResizeControl> {
    return this.resize(terminalId, size ?? { cols: 80, rows: 24 }, ResizePolicy.OWNER)
  }

  releaseResizeOwner(terminalId: string): Promise<TerminalResizeControl> {
    return this.resize(terminalId, this.terminalSizes.get(terminalId) ?? { cols: 80, rows: 24 }, ResizePolicy.FOLLOWER)
  }

  sendInput(terminalId: string, data: string, size?: TerminalInputSize): void {
    const attachment = this.attachments.get(terminalId)
    const resource = attachment?.handle.resource
    if (!attachment || !resource) throw new Error('terminal attachment is unavailable')
    // PTY input 的真值顺序是用户事件顺序；同一 terminal 必须串行等待 ACK，不能按并发 RPC 完成顺序写入。
    const send = async () => {
      if (this.attachments.get(terminalId) !== attachment) return
      if (size) await this.resize(terminalId, size, ResizePolicy.OWNER)
      if (this.attachments.get(terminalId) !== attachment) return
      const result = await this.session.execute(command('terminalInput', create(TerminalInputCommandSchema, {
        attachment: resource,
        data: new TextEncoder().encode(data),
      })))
      if (result.result.case !== 'acknowledge') throw new Error('terminal input was not acknowledged')
    }
    const previous = this.inputTails.get(terminalId) ?? Promise.resolve()
    const next = previous.then(send).catch((error) => {
      if (this.attachments.get(terminalId) === attachment) {
        this.publish(terminalId, { type: 'closed', reason: errorMessage(error) })
      }
    })
    this.inputTails.set(terminalId, next)
    void next.finally(() => {
      if (this.inputTails.get(terminalId) === next) this.inputTails.delete(terminalId)
    })
  }

  sendResize(terminalId: string, cols: number, rows: number): void {
    void this.resize(terminalId, { cols, rows }, ResizePolicy.OWNER).catch((error) => {
      this.publish(terminalId, { type: 'closed', reason: errorMessage(error) })
    })
  }

  private async resize(terminalId: string, size: TerminalInputSize, policy: ResizePolicy): Promise<TerminalResizeControl> {
    const resource = this.attachments.get(terminalId)?.handle.resource
    if (!resource) throw new Error('terminal attachment is unavailable')
    const result = await this.session.execute(command('terminalResize', create(TerminalResizeCommandSchema, {
      attachment: resource,
      size: create(TerminalSizeSchema, { cols: size.cols, rows: size.rows }),
      resizePolicy: policy,
    })))
    if (result.result.case !== 'terminalResize') throw new Error('terminal resize returned no result')
    this.terminalSizes.set(terminalId, size)
    const control = resizeControlView(result.result.value.resizeControl)
    this.publish(terminalId, { type: 'resizeControl', control })
    return control
  }

  private async publishTerminalInfo(terminalId: string): Promise<void> {
    const result = await this.session.execute(command('terminalGet', create(TerminalGetCommandSchema, { terminal: this.terminalRef(terminalId) })))
    if (result.result.case === 'terminalGet' && result.result.value.terminal) {
      this.rememberSize(terminalId, result.result.value.terminal.size)
      this.publish(terminalId, { type: 'info', info: terminalInfoView(result.result.value.terminal) })
    }
  }

  private async startLiveScreenRequest(terminalId: string, state: LiveScreenDeliveryState): Promise<void> {
    const attachment = this.attachments.get(terminalId)
    if (
      !attachment ||
      this.liveScreens.get(terminalId) !== state ||
      state.requestController ||
      !state.demand ||
      !this.documentVisible ||
      state.publishedRevision !== undefined ||
      state.damage !== undefined
    ) return

    const controller = new AbortController()
    state.requestController = controller
    const observedRevision = state.received?.revision ?? 0n
    try {
      const result = await this.session.execute(command('liveScreenNext', create(LiveScreenNextCommandSchema, {
        terminal: this.terminalRef(terminalId),
        observedRevision,
      })), { signal: controller.signal })
      if (result.result.case !== 'liveScreen') throw new Error('live screen next returned no result')
      if (
        controller.signal.aborted ||
        this.attachments.get(terminalId) !== attachment ||
        this.liveScreens.get(terminalId) !== state
      ) return
      state.requestController = undefined
      const incoming = result.result.value
      if (state.received && incoming.liveRevision <= state.received.revision) return
      const merged = mergeLiveScreenResult(
        state.received,
        incoming,
        this.session.stamp.generation,
        this.terminalRef(terminalId),
      )
      if (!merged) {
        state.received = undefined
        state.damage = undefined
        state.reason = 'sync_lost'
        await this.startLiveScreenRequest(terminalId, state)
        return
      }
      state.received = merged.screen
      state.damage = merged.damage
      this.rememberSize(terminalId, { cols: merged.screen.cols, rows: merged.screen.rows })
      if (state.renderingRevision === undefined && state.publishedRevision === undefined && this.documentVisible && state.demand) {
        this.publishReceivedScreen(terminalId, state)
      }
    } catch (error) {
      if (state.requestController === controller) state.requestController = undefined
      if (controller.signal.aborted || isAbortError(error)) return
      if (!this.session.isAlive() || !this.attachments.has(terminalId)) return
      this.publish(terminalId, { type: 'closed', reason: errorMessage(error) })
    }
  }

  private publishReceivedScreen(terminalId: string, state: LiveScreenDeliveryState): void {
    const screen = state.received
    const damage = state.damage
    if (!screen || !damage || state.publishedRevision !== undefined || state.renderingRevision !== undefined) return
    state.damage = undefined
    state.publishedRevision = screen.revision
    this.publish(terminalId, { type: 'snapshot', snapshot: screenSnapshot(screen, damage, state.reason) })
  }

  private rememberSize(terminalId: string, size: { cols: number; rows: number } | undefined): void {
    if (size && size.cols > 0 && size.rows > 0) this.terminalSizes.set(terminalId, { cols: size.cols, rows: size.rows })
  }

  private terminalRef(terminalId: string) {
    return create(TerminalRefSchema, { endpointId: this.session.stamp.endpointId, terminalId })
  }

  private publish(terminalId: string, event: TerminalProtocolEvent): void {
    this.subscribers.get(terminalId)?.forEach((handler) => handler(event))
  }
}

class ProtoTerminalChannel implements TerminalProtocolChannel {
  readonly label: string
  readyState: TerminalProtocolChannel['readyState'] = 'open'

  constructor(private readonly owner: ProtoTerminalProtocolSession, private readonly terminalId: string) {
    this.label = `proto-terminal:${terminalId}`
  }

  send(_data: Uint8Array): void { throw new Error('Proto terminal channel accepts semantic input only') }
  sendInput(data: string, size?: TerminalInputSize): void { this.owner.sendInput(this.terminalId, data, size) }
  sendResize(cols: number, rows: number): void { this.owner.sendResize(this.terminalId, cols, rows) }
  close(): void { this.owner.closeTerminalChannel(this.terminalId) }
  markClosed(): void { this.readyState = 'closed' }
}

function command(caseName: string, value: object) {
  return create(CommandEnvelopeSchema, { command: { case: caseName, value } } as never)
}

function screenSnapshot(
  screen: CanonicalLiveScreen,
  damage: LiveScreenDamage,
  reason: TerminalSnapshotPayload['refreshReason'],
): TerminalSnapshotPayload {
  const text = fullScreenANSI(screen)
  return {
    text,
    screenReplay: text,
    screenText: rowsText(screen.screenRows),
    liveReplay: damage.fullReplace ? text : changedRowsANSI(screen, damage.changedRows),
    liveRevision: screen.revision,
    liveFullReplace: damage.fullReplace,
    cols: screen.cols,
    rows: screen.rows,
    alternateScreen: screen.alternateScreen,
    ...(reason ? { refreshReason: reason } : {}),
  }
}

function fullScreenDamage(rows: number): LiveScreenDamage {
  return { fullReplace: true, changedRows: Array.from({ length: rows }, (_, rowIndex) => rowIndex) }
}

function fullScreenANSI(screen: CanonicalLiveScreen): string {
  const alternate = screen.alternateScreen ? '\u001b[?1049h' : ''
  return `${alternate}\u001b[?7l\u001b[H\u001b[2J\u001b[H${rowsANSI(screen.screenRows, screen.cols)}${modesANSI(screen.modes)}${cursorANSI(screen.cursor)}`
}

function changedRowsANSI(screen: CanonicalLiveScreen, changedRows: number[]): string {
  let output = '\u001b[?7l'
  for (const rowIndex of changedRows) {
    output += `\u001b[${rowIndex + 1};1H${rowANSI(screen.screenRows[rowIndex]!, screen.cols)}`
  }
  return `${output}${modesANSI(screen.modes)}${cursorANSI(screen.cursor)}`
}

function rowsText(rows: ScreenRow[]): string {
  return rows.map((row) => row.cells.map((cell) => {
    if (isWideContinuationCell(cell)) return ''
    return cell.content || ' '.repeat(Math.max(1, cell.width))
  }).join('').replace(/\s+$/, '')).join('\r\n')
}

function rowsANSI(rows: ScreenRow[], cols: number): string {
  return rows.map((row) => rowANSI(row, cols)).join('\r\n')
}

function rowANSI(row: ScreenRow, cols: number): string {
  let current = ''
  let width = 0
  let output = ''
  for (const cell of row.cells) {
    if (isWideContinuationCell(cell)) continue
    const style = styleANSI(cell.style)
    if (style !== current) {
      output += `\u001b[0m${style}`
      current = style
    }
    const content = cell.content || ' '.repeat(Math.max(1, cell.width))
    output += content
    width += Math.max(1, cell.width)
  }
  if (cols > width) output += `\u001b[0m${styleANSI(row.tailFill)}${' '.repeat(cols - width)}`
  return `${output}\u001b[0m`
}

function isWideContinuationCell(cell: ScreenRow['cells'][number]): boolean {
  return cell.width <= 0 && cell.content === ''
}

function styleANSI(style: CellStyle | undefined): string {
  if (!style) return ''
  const codes: string[] = []
  if (style.bold) codes.push('1')
  if (style.italic) codes.push('3')
  if (style.underline) codes.push('4')
  if (style.blink) codes.push('5')
  if (style.reverse) codes.push('7')
  if (style.strikethrough) codes.push('9')
  const foreground = colorANSI(style.foreground, true)
  const background = colorANSI(style.background, false)
  if (foreground) codes.push(foreground)
  if (background) codes.push(background)
  return codes.length > 0 ? `\u001b[${codes.join(';')}m` : ''
}

function colorANSI(value: string, foreground: boolean): string {
  const token = value.trim()
  if (!token) return ''
  const indexed = /^(?:(?:ansi|idx):)?(\d{1,3})$/i.exec(token)
  if (indexed) {
    const index = Math.min(255, Number(indexed[1]))
    return `${foreground ? 38 : 48};5;${index}`
  }
  const rgb = /^#([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(token)
  if (!rgb) return ''
  return `${foreground ? 38 : 48};2;${parseInt(rgb[1]!, 16)};${parseInt(rgb[2]!, 16)};${parseInt(rgb[3]!, 16)}`
}

function cursorANSI(cursor: TerminalCursor | undefined): string {
  if (!cursor) return ''
  const visibility = cursor.visible ? '\u001b[?25h' : '\u001b[?25l'
  const shape = cursor.shape === CursorShape.UNDERLINE ? (cursor.blink ? 3 : 4)
    : cursor.shape === CursorShape.BAR ? (cursor.blink ? 5 : 6)
      : cursor.blink ? 1 : 2
  return `\u001b[${Math.max(1, cursor.row + 1)};${Math.max(1, cursor.col + 1)}H\u001b[${shape} q${visibility}`
}

function modesANSI(modes: TerminalModes | undefined): string {
  if (!modes) return ''
  const mouseNormal = modes.mouseNormal || (modes.mouseTracking && !modes.mouseX10 && !modes.mouseButtonEvent && !modes.mouseAnyEvent)
  return [
    privateModeANSI(1, modes.applicationCursor),
    privateModeANSI(7, modes.autoWrap),
    privateModeANSI(1007, modes.alternateScroll),
    privateModeANSI(2004, modes.bracketedPaste),
    privateModeANSI(9, modes.mouseX10),
    privateModeANSI(1000, mouseNormal),
    privateModeANSI(1002, modes.mouseButtonEvent),
    privateModeANSI(1003, modes.mouseAnyEvent),
    privateModeANSI(1005, false),
    privateModeANSI(1006, modes.mouseSgr),
  ].join('')
}

function privateModeANSI(mode: number, enabled: boolean): string {
  return `\u001b[?${mode}${enabled ? 'h' : 'l'}`
}

function terminalInfoView(info: TerminalInfo): Record<string, unknown> {
  return {
    terminal_id: info.ref?.terminalId ?? '', machine_id: info.ref?.endpointId ?? '', name: info.name, command: info.command, tags: info.tags,
    state: info.state === TerminalState.RUNNING ? 'running' : info.state === TerminalState.EXITED ? 'exited' : info.state === TerminalState.REMOVED ? 'removed' : 'created', cwd: info.cwd, live_cwd: info.liveCwd,
    created_at_unix_nano: info.createdAtUnixNano, exit_code: info.exitCode, exited_at_unix_nano: info.exitedAtUnixNano,
    foreground_process: info.foregroundProcess || undefined,
    last_output_at: unixNanoISOString(info.lastOutputAtUnixNano),
  }
}

function unixNanoISOString(value: bigint): string | undefined {
  if (value <= 0n) return undefined
  return new Date(Number(value / 1_000_000n)).toISOString()
}

function resizeControlView(control: ResizeControl | undefined): TerminalResizeControl {
  if (!control) return { canResize: false, reason: 'unknown' }
  const reason = control.reason === ResizeControlReason.OWNER ? 'owner'
    : control.reason === ResizeControlReason.FOLLOWER ? 'follower'
      : control.reason === ResizeControlReason.OBSERVER ? 'observer'
        : control.reason === ResizeControlReason.SIZE_LOCKED ? 'size_locked' : 'unknown'
  return {
    canResize: control.canResize, reason, sizeLocked: control.sizeLocked,
    ...(control.surfaceId ? { surfaceId: control.surfaceId } : {}),
    ...(control.ownerSurfaceId ? { ownerSurfaceId: control.ownerSurfaceId } : {}),
    ...(control.ownerViewId ? { ownerViewId: control.ownerViewId } : {}),
  }
}

function errorMessage(error: unknown): string { return error instanceof Error ? error.message : String(error) }

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException
    ? error.name === 'AbortError'
    : error instanceof Error && error.name === 'AbortError'
}
