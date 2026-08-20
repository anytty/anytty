import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from 'react'
import {
  initialConnectionSnapshot,
  reduceConnectionMessage,
  type ConnectionMessage,
  type ConnectionSnapshot,
} from '../connection/connectionMessageReducer'
import {
  defaultTerminalResizeControl,
  TerminalClient,
  type TerminalClientCallbacks,
  type TerminalHistorySearchResult,
  type TerminalProtocolSession,
  type TerminalResizeControl,
  type TerminalScrollbackLoadResult,
  type TerminalScrollbackPage,
  type TerminalSnapshotPayload,
} from './terminalClient'
import { createProtoTerminalProtocolSession } from './protoTerminalProtocolSession'
import type { ProtoClientSession } from '../core/protoClientSession'
import { logTerminalDiagnostic, terminalNow } from './terminalDiagnostics'
import { appendTerminalText } from './terminalTextWindow'
import type { Terminal } from '../core/model'
import type {
  ConnectionInfo,
} from '../core/transport'

const recentInputRecoveryWindowMs = 1500
const inputRecoveryMaxEntries = 64
const inputRecoveryMaxBytes = 64 * 1024
const inputRecoveryOverflowReason = 'Terminal input is blocked because the recovery buffer is full'
const outputStatsIntervalMs = 1000
const largeOutputChunkBytes = 64 * 1024
const liveOutputPublishDelayMs = 120
const scrollbackLoadTimeoutMs = 10_000

interface TerminalChannelRecoveryOptions {
  session?: TerminalSession | undefined
  forceTerminalChannel?: boolean | undefined
  supersede?: boolean | undefined
  showClosedReason?: boolean | undefined
}

export interface UseTerminalSessionOptions {
  machineId: string
  terminalId: string
  session: TerminalSession
  onOutput?: ((text: string) => void) | undefined
  historyOnly?: boolean | undefined
}

export interface UseTerminalSessionResult {
  snapshot: ConnectionSnapshot
  inputRecoveryFailure: string | null
  terminalSnapshot: TerminalSnapshotPayload | null
  terminalText: string
  terminalInfo: Terminal | null
  resizeControl: TerminalResizeControl
  historyReady: boolean
  /** True once the channel accepts locally or recovery owns replay; this is not a remote PTY acknowledgement. */
  sendInput(data: string, size?: { cols: number; rows: number }): boolean
  sendResize(cols: number, rows: number): boolean
  requestResizeOwner(size?: { cols: number; rows: number }): Promise<TerminalResizeControl>
  releaseResizeOwner(): Promise<TerminalResizeControl>
  loadScrollback(limit?: number, alternate?: boolean, cols?: number): Promise<TerminalScrollbackLoadResult>
  searchScrollback(
    query: string,
    direction: 'forward' | 'backward',
    cols: number,
    start?: { lineId: string; col: number } | undefined,
  ): Promise<TerminalHistorySearchResult>
  copyScrollback(
    range: { startLineId: string; startCol: number; endLineId: string; endCol: number },
    cols: number,
  ): Promise<string>
  cancelHistorySearch(): void
  prefetchScrollback(limit?: number, alternate?: boolean, cols?: number): Promise<boolean>
  resetScrollback(): void
  freezeScrollback(): void
  resumeLiveScrollback(): string
  markSyncLost(reason?: string): void
  markLiveScreenSubmitted(revision: bigint): void
  markLiveScreenCompleted(revision: bigint): void
  handleAppResume(resumeKind: 'quick' | 'cold' | 'frozen'): void
  reattach(session: TerminalSession, options?: { forceTerminalChannel?: boolean }): void
  client: TerminalClient | null
}

type TerminalSession = ProtoClientSession

interface InputRecoveryMessage {
  data: string
  size?: { cols: number; rows: number } | undefined
  byteLength: number
}

interface InputRecoveryOwner {
  readonly session: TerminalSession
  readonly sequence: number
  phase: 'recent' | 'recovering' | 'draining'
  messages: InputRecoveryMessage[]
  nextIndex: number
  queuedBytes: number
  overflowed: boolean
  expiryTimer: ReturnType<typeof setTimeout> | null
}

interface InputRecoveryFailure {
  readonly session: TerminalSession
  readonly sequence: number
  readonly reason: string
}

interface ScrollbackPrefetchEntry {
  key: string
  revision: number
  page: TerminalScrollbackPage
}

interface ScrollbackPrefetchPending {
  key: string
  revision: number
  controller: AbortController
  promise: Promise<TerminalScrollbackPage | null>
}

function scrollbackPrefetchKey(limit: number, alternate: boolean, cols?: number): string {
  return `${alternate ? 'alternate' : 'normal'}:${Math.max(0, Math.trunc(cols ?? 0))}:${Math.max(0, Math.trunc(limit))}`
}

export function useTerminalSession(options: UseTerminalSessionOptions): UseTerminalSessionResult {
  const [snapshot, dispatch] = useReducer(reduceConnectionMessage, undefined, initialConnectionSnapshot)
  const [inputRecoveryFailure, setInputRecoveryFailure] = useState<InputRecoveryFailure | null>(null)
  const [terminalSnapshot, setTerminalSnapshot] = useState<TerminalSnapshotPayload | null>(null)
  const [terminalText, setTerminalText] = useState('')
  const [terminalInfo, setTerminalInfo] = useState<Terminal | null>(null)
  const [resizeControl, setResizeControl] = useState<TerminalResizeControl>(defaultTerminalResizeControl)
  const [historyReady, setHistoryReady] = useState(false)
  const clientRef = useRef<TerminalClient | null>(null)
  const sessionRef = useRef(options.session)
  const connectionIdRef = useRef('terminal-connection')
  const protocolSessionRef = useRef<TerminalProtocolSession | null>(null)
  const scrollbackPrefixTextRef = useRef('')
  const loadedScrollbackRowsRef = useRef(0)
  const scrollbackRowTimestampsRef = useRef<Array<number | undefined>>([])
  const scrollbackRowLineIdsRef = useRef<Array<string | undefined>>([])
  const scrollbackRowInLinesRef = useRef<Array<number | undefined>>([])
  const scrollbackRowStartColsRef = useRef<Array<number | undefined>>([])
  const historyRevisionRef = useRef(0)
  const loadingScrollbackRef = useRef(false)
  const scrollbackAbortControllerRef = useRef<AbortController | null>(null)
  const hasMoreScrollbackRef = useRef(true)
  const scrollbackFrozenRef = useRef(false)
  const liveHistoryRevisionRef = useRef(0)
  const scrollbackPrefetchRef = useRef<ScrollbackPrefetchEntry | null>(null)
  const scrollbackPrefetchPendingRef = useRef<ScrollbackPrefetchPending | null>(null)
  const activeScrollbackModeRef = useRef<'normal' | 'alternate'>('normal')
  const inputRecoveryOwnerRef = useRef<InputRecoveryOwner | null>(null)
  const inputRecoverySequenceRef = useRef(0)
  const terminalRecoveryPromiseRef = useRef<Promise<boolean> | null>(null)
  const terminalRecoverySeqRef = useRef(0)
  const inputTextEncoderRef = useRef(new TextEncoder())
  const textDecoderRef = useRef(new TextDecoder())
  const onOutputRef = useRef<((text: string) => void) | undefined>(options.onOutput)
  const terminalTextRef = useRef('')
  const terminalContentTextRef = useRef('')
  const terminalTextPublishTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const outputStatsRef = useRef({
    chunks: 0,
    bytes: 0,
    chars: 0,
    trimmedChars: 0,
    lastLogAt: terminalNow(),
    lastBytes: 0,
    lastChunks: 0,
  })

  const logSession = useCallback((event: string, input: {
    level?: 'debug' | 'info' | 'warn' | 'error'
    details?: Record<string, unknown> | undefined
  } = {}) => {
    logTerminalDiagnostic(`session.${event}`, {
      level: input.level,
      machineId: options.machineId,
      terminalId: options.terminalId,
      connectionId: connectionIdRef.current,
      details: input.details,
    })
  }, [options.machineId, options.terminalId])

  const maybeLogOutputStats = useCallback((currentTextChars: number) => {
    const now = terminalNow()
    const stats = outputStatsRef.current
    if (now - stats.lastLogAt < outputStatsIntervalMs) return
    const elapsedSeconds = Math.max(0.001, (now - stats.lastLogAt) / 1000)
    const intervalBytes = stats.bytes - stats.lastBytes
    const intervalChunks = stats.chunks - stats.lastChunks
    logSession('output_stats', {
      details: {
        chunks: stats.chunks,
        bytes: stats.bytes,
        chars: stats.chars,
        currentTextChars,
        trimmedChars: stats.trimmedChars,
        intervalChunks,
        intervalBytes,
        bytesPerSecond: Math.round(intervalBytes / elapsedSeconds),
      },
    })
    stats.lastLogAt = now
    stats.lastBytes = stats.bytes
    stats.lastChunks = stats.chunks
  }, [logSession])

  const isCurrentInputRecoveryOwner = useCallback((owner: InputRecoveryOwner): boolean => {
    return inputRecoveryOwnerRef.current === owner &&
      inputRecoverySequenceRef.current === owner.sequence &&
      sessionRef.current === owner.session
  }, [])

  const clearInputRecoveryOwner = useCallback((owner: InputRecoveryOwner): boolean => {
    if (
      inputRecoveryOwnerRef.current !== owner ||
      inputRecoverySequenceRef.current !== owner.sequence ||
      sessionRef.current !== owner.session
    ) return false
    inputRecoveryOwnerRef.current = null
    if (owner.expiryTimer !== null) {
      clearTimeout(owner.expiryTimer)
      owner.expiryTimer = null
    }
    return true
  }, [])

  const revokeInputRecoveryOwner = useCallback((expectedSession?: TerminalSession): boolean => {
    if (expectedSession && sessionRef.current !== expectedSession) return false
    const owner = inputRecoveryOwnerRef.current
    if (expectedSession && owner && owner.session !== expectedSession) return false
    inputRecoverySequenceRef.current += 1
    inputRecoveryOwnerRef.current = null
    if (owner?.expiryTimer != null) {
      clearTimeout(owner.expiryTimer)
      owner.expiryTimer = null
    }
    return true
  }, [])

  const setInputRecoveryOwnerFailure = useCallback((owner: InputRecoveryOwner, reason: string): boolean => {
    if (!isCurrentInputRecoveryOwner(owner)) return false
    setInputRecoveryFailure({ session: owner.session, sequence: owner.sequence, reason })
    return true
  }, [isCurrentInputRecoveryOwner])

  const setCurrentInputRecoveryFailure = useCallback((
    session: TerminalSession,
    sequence: number,
    reason: string,
  ): boolean => {
    if (sessionRef.current !== session || inputRecoverySequenceRef.current !== sequence) return false
    setInputRecoveryFailure({ session, sequence, reason })
    return true
  }, [])

  const clearInputRecoveryFailureForOwner = useCallback((owner: InputRecoveryOwner) => {
    setInputRecoveryFailure((current) => (
      current?.session === owner.session && current.sequence === owner.sequence ? null : current
    ))
  }, [])

  const clearInputRecoveryFailureForAcceptedInput = useCallback((
    session: TerminalSession,
    sequence: number,
  ) => {
    setInputRecoveryFailure((current) => {
      if (
        sessionRef.current !== session ||
        inputRecoverySequenceRef.current !== sequence ||
        current?.session !== session ||
        current.sequence > sequence
      ) return current
      return null
    })
  }, [])

  const clearInputRecoveryFailureForSession = useCallback((
    session: TerminalSession,
    throughSequence: number,
  ) => {
    setInputRecoveryFailure((current) => (
      current?.session === session && current.sequence <= throughSequence ? null : current
    ))
  }, [])

  const clearTerminalTextPublishTimer = useCallback(() => {
    if (terminalTextPublishTimerRef.current === null) return
    clearTimeout(terminalTextPublishTimerRef.current)
    terminalTextPublishTimerRef.current = null
  }, [])

  const publishTerminalTextNow = useCallback(() => {
    clearTerminalTextPublishTimer()
    setTerminalText(terminalTextRef.current)
  }, [clearTerminalTextPublishTimer])

  const scheduleTerminalTextPublish = useCallback(() => {
    if (terminalTextPublishTimerRef.current !== null) return
    terminalTextPublishTimerRef.current = setTimeout(() => {
      terminalTextPublishTimerRef.current = null
      setTerminalText(terminalTextRef.current)
    }, liveOutputPublishDelayMs)
  }, [])

  useEffect(() => {
    onOutputRef.current = options.onOutput
  }, [options.onOutput])

  const recoverTerminalChannel = useCallback((
    reason: string,
    recoveryOptions: TerminalChannelRecoveryOptions = {},
  ): Promise<boolean> => {
    const existing = terminalRecoveryPromiseRef.current
    if (existing && recoveryOptions.supersede !== true) return existing

    const seq = terminalRecoverySeqRef.current + 1
    terminalRecoverySeqRef.current = seq
    const targetSession = recoveryOptions.session ?? sessionRef.current
    const previousSession = sessionRef.current
    const client = clientRef.current
    if (!client) return Promise.resolve(false)
    if (targetSession !== previousSession) {
      revokeInputRecoveryOwner(previousSession)
      clearInputRecoveryFailureForSession(previousSession, inputRecoverySequenceRef.current)
      sessionRef.current = targetSession
    }

    const startedAt = terminalNow()
    logSession('terminal_recovery_start', {
      level: 'warn',
      details: {
        reason,
        forceTerminalChannel: recoveryOptions.forceTerminalChannel === true,
        supersede: recoveryOptions.supersede === true,
      },
    })

    const recoveryPromise = (async (): Promise<boolean> => {
      try {
        scrollbackAbortControllerRef.current?.abort(new DOMException('Terminal channel reconnecting', 'AbortError'))
        scrollbackAbortControllerRef.current = null
        scrollbackPrefetchPendingRef.current?.controller.abort(new DOMException('Terminal channel reconnecting', 'AbortError'))
        scrollbackPrefetchPendingRef.current = null
        scrollbackPrefetchRef.current = null
        dispatch({
          type: 'terminal.channelClosed',
          machineId: options.machineId,
          terminalId: options.terminalId,
          ...(recoveryOptions.showClosedReason === false ? {} : { reason }),
        })
        protocolSessionRef.current?.closeTerminalChannel(options.terminalId)
        closeRtcTerminalDataChannel(previousSession, options.terminalId)
        if (recoveryOptions.forceTerminalChannel) {
          closeRtcTerminalDataChannel(targetSession, options.terminalId)
        }
        setResizeControl(defaultTerminalResizeControl)
        const info = await terminalSessionConnectionInfo(targetSession)
        if (terminalRecoverySeqRef.current !== seq) return false
        connectionIdRef.current = info.connectionId
        const protocolSession = createProtoTerminalProtocolSession(targetSession)
        protocolSessionRef.current = protocolSession
        await client.reattach(protocolSession)
        if (terminalRecoverySeqRef.current !== seq) return false
        dispatch({ type: 'connection.verified', connectionId: info.connectionId })
        logSession('terminal_recovery_done', {
          level: 'info',
          details: {
            reason,
            elapsedMs: Math.round(terminalNow() - startedAt),
            connectionId: info.connectionId,
          },
        })
        return true
      } catch (err) {
        if (terminalRecoverySeqRef.current === seq) {
          const message = err instanceof Error ? err.message : String(err)
          logSession('terminal_recovery_failed', {
            level: 'error',
            details: {
              reason,
              elapsedMs: Math.round(terminalNow() - startedAt),
              error: message,
            },
          })
          dispatch({
            type: 'connection.failed',
            reason: message,
            recoverable: true,
            surface: 'banner',
          })
        }
        return false
      } finally {
        if (terminalRecoverySeqRef.current === seq) {
          terminalRecoveryPromiseRef.current = null
        }
      }
    })()
    terminalRecoveryPromiseRef.current = recoveryPromise
    return recoveryPromise
  }, [clearInputRecoveryFailureForSession, logSession, options.machineId, options.terminalId, revokeInputRecoveryOwner])

  const startInputRecoveryOwner = useCallback((owner: InputRecoveryOwner, reason: string): boolean => {
    if (!isCurrentInputRecoveryOwner(owner)) return false
    if (owner.phase !== 'recent') return true
    const client = clientRef.current
    if (!client || !owner.session.isAlive()) {
      clearInputRecoveryOwner(owner)
      return false
    }

    owner.phase = 'recovering'
    if (owner.expiryTimer !== null) {
      clearTimeout(owner.expiryTimer)
      owner.expiryTimer = null
    }

    void (async () => {
      try {
        const recovered = await recoverTerminalChannel(reason, {
          session: owner.session,
          forceTerminalChannel: true,
        })
        if (!isCurrentInputRecoveryOwner(owner)) return
        if (!recovered) {
          setInputRecoveryOwnerFailure(owner, 'Terminal input recovery failed')
          return
        }

        owner.phase = 'draining'
        while (isCurrentInputRecoveryOwner(owner) && owner.nextIndex < owner.messages.length) {
          const message = owner.messages[owner.nextIndex]
          if (!message) break
          const accepted = client.sendInput(message.data, message.size)
          if (!accepted) {
            setInputRecoveryOwnerFailure(owner, 'Terminal input recovery failed while replaying queued input')
            return
          }
          owner.nextIndex += 1
          owner.queuedBytes -= message.byteLength
        }
        if (isCurrentInputRecoveryOwner(owner) && owner.nextIndex === owner.messages.length) {
          clearInputRecoveryFailureForOwner(owner)
        }
      } catch (error) {
        setInputRecoveryOwnerFailure(owner, error instanceof Error ? error.message : String(error))
      } finally {
        clearInputRecoveryOwner(owner)
      }
    })()
    return true
  }, [clearInputRecoveryFailureForOwner, clearInputRecoveryOwner, isCurrentInputRecoveryOwner, recoverTerminalChannel, setInputRecoveryOwnerFailure])

  const invalidateLiveScrollbackPrefetch = useCallback((reason: 'output' | 'snapshot') => {
    liveHistoryRevisionRef.current += 1
    const cached = scrollbackPrefetchRef.current
    const pending = scrollbackPrefetchPendingRef.current
    if (!cached && !pending) return
    scrollbackPrefetchRef.current = null
    pending?.controller.abort(new DOMException('Terminal live content changed', 'AbortError'))
    clientRef.current?.resetScrollback()
    logSession('scrollback_prefetch_invalidated', {
      level: 'debug',
      details: { reason, cached: cached !== null, pending: pending !== null },
    })
  }, [logSession])

  const callbacks = useMemo<TerminalClientCallbacks>(() => ({
    onOutput: (data) => {
      if (!scrollbackFrozenRef.current) {
        hasMoreScrollbackRef.current = true
        invalidateLiveScrollbackPrefetch('output')
      }
      const decoded = textDecoderRef.current.decode(data)
      const stats = outputStatsRef.current
      stats.chunks += 1
      stats.bytes += data.byteLength
      stats.chars += decoded.length
      if (data.byteLength >= largeOutputChunkBytes) {
        logSession('large_output_chunk', {
          level: 'warn',
          details: {
            bytes: data.byteLength,
            decodedChars: decoded.length,
            chunks: stats.chunks,
            totalBytes: stats.bytes,
          },
        })
      }
      onOutputRef.current?.(decoded)
      terminalContentTextRef.current = appendTerminalText(terminalContentTextRef.current, decoded)
      if (scrollbackFrozenRef.current) {
        maybeLogOutputStats(terminalContentTextRef.current.length)
        return
      }
      const current = terminalTextRef.current
      const next = appendTerminalText(current, decoded)
      const expectedLength = current.length + decoded.length
      if (next.length < expectedLength) {
        stats.trimmedChars += expectedLength - next.length
      }
      terminalTextRef.current = next
      maybeLogOutputStats(next.length)
      scheduleTerminalTextPublish()
    },
    onSnapshot: (nextSnapshot) => {
      if (!scrollbackFrozenRef.current) invalidateLiveScrollbackPrefetch('snapshot')
      const canPreserveHistory = scrollbackFrozenRef.current &&
        !nextSnapshot.alternateScreen &&
        activeScrollbackModeRef.current === 'normal' &&
        scrollbackPrefixTextRef.current !== ''
      const nextText = nextSnapshot.screenReplay ?? nextSnapshot.replay ?? nextSnapshot.text
      const screenText = nextSnapshot.screenReplay ?? nextSnapshot.text
      setTerminalSnapshot((current) => canPreserveHistory && current?.history
        ? { ...nextSnapshot, history: current.history }
        : nextSnapshot)
      logSession('snapshot_received', {
        details: {
          textChars: nextSnapshot.text.length,
          replayChars: nextSnapshot.replay?.length ?? 0,
          screenTextChars: nextSnapshot.screenText?.length ?? 0,
          screenReplayChars: nextSnapshot.screenReplay?.length ?? 0,
          rows: nextSnapshot.rows,
          cols: nextSnapshot.cols,
          scrollbackRows: nextSnapshot.scrollbackRows?.length ?? 0,
          alternateScreen: nextSnapshot.alternateScreen === true,
        },
      })
      terminalContentTextRef.current = screenText ?? nextText
      if (!canPreserveHistory) {
        loadedScrollbackRowsRef.current = 0
        scrollbackPrefixTextRef.current = ''
        activeScrollbackModeRef.current = nextSnapshot.alternateScreen ? 'alternate' : 'normal'
        hasMoreScrollbackRef.current = true
      }
      terminalTextRef.current = canPreserveHistory
        ? scrollbackPrefixTextRef.current
        : nextText
      publishTerminalTextNow()
    },
    onTerminalInfo: setTerminalInfo,
    onResizeControl: setResizeControl,
    onLifecycle: (message) => dispatch(message),
    onError: (message) => dispatch({ type: 'connection.failed', reason: message, recoverable: true, surface: 'banner' }),
    onClose: (reason) => {
      if (!sessionRef.current.isAlive()) {
        // native generation 已失效时，terminal 不得在旧 lease 上自建订阅或重放输入；
        // MachineWorkspace 会在新 generation 发布后重新取得 session 并 reattach。
        revokeInputRecoveryOwner(sessionRef.current)
        return
      }
      const owner = inputRecoveryOwnerRef.current
      if (owner && isCurrentInputRecoveryOwner(owner) && owner.phase !== 'recent') {
        return
      }
      if (owner && isCurrentInputRecoveryOwner(owner) && shouldRecoverRecentInput(reason)) {
        startInputRecoveryOwner(owner, reason ?? 'terminal channel closed')
      } else if (shouldRecoverTerminalChannel(sessionRef.current, reason)) {
        void recoverTerminalChannel(reason ?? 'terminal channel closed', {
          forceTerminalChannel: true,
        })
      }
    },
    onOpen: () => {},
    onInputDropped: () => {
      const owner = inputRecoveryOwnerRef.current
      if (!owner || !isCurrentInputRecoveryOwner(owner)) {
        dispatch({ type: 'connection.failed', reason: 'terminal input dropped', recoverable: true, surface: 'toast' })
      }
    },
    onInputSendFailed: (reason) => {
      if (!sessionRef.current.isAlive()) {
        revokeInputRecoveryOwner(sessionRef.current)
        return
      }
      const owner = inputRecoveryOwnerRef.current
      if (owner && isCurrentInputRecoveryOwner(owner)) {
        startInputRecoveryOwner(owner, reason)
      } else if (shouldRecoverTerminalChannel(sessionRef.current, reason)) {
        void recoverTerminalChannel(reason, { forceTerminalChannel: true })
      }
    },
  }), [invalidateLiveScrollbackPrefetch, isCurrentInputRecoveryOwner, logSession, maybeLogOutputStats, publishTerminalTextNow, recoverTerminalChannel, revokeInputRecoveryOwner, scheduleTerminalTextPublish, startInputRecoveryOwner])

  useEffect(() => {
    const previousSession = sessionRef.current
    revokeInputRecoveryOwner()
    clearInputRecoveryFailureForSession(previousSession, inputRecoverySequenceRef.current)
    sessionRef.current = options.session
    const client = new TerminalClient(callbacks)
    clientRef.current = client

    let cancelled = false
    logSession('mount', { level: 'info' })
    clearTerminalTextPublishTimer()
    terminalTextRef.current = ''
    terminalContentTextRef.current = ''
    setTerminalSnapshot(null)
    setTerminalText('')
    setTerminalInfo(null)
    setResizeControl(defaultTerminalResizeControl)
    setHistoryReady(false)
    scrollbackPrefixTextRef.current = ''
    loadedScrollbackRowsRef.current = 0
    scrollbackRowTimestampsRef.current = []
    scrollbackRowLineIdsRef.current = []
    scrollbackRowInLinesRef.current = []
    scrollbackRowStartColsRef.current = []
    historyRevisionRef.current = 0
    loadingScrollbackRef.current = false
    scrollbackAbortControllerRef.current?.abort(new DOMException('Terminal session replaced', 'AbortError'))
    scrollbackAbortControllerRef.current = null
    scrollbackPrefetchPendingRef.current?.controller.abort(new DOMException('Terminal session replaced', 'AbortError'))
    scrollbackPrefetchPendingRef.current = null
    scrollbackPrefetchRef.current = null
    hasMoreScrollbackRef.current = true
    scrollbackFrozenRef.current = false
    liveHistoryRevisionRef.current = 0
    activeScrollbackModeRef.current = 'normal'
    terminalRecoveryPromiseRef.current = null
    terminalRecoverySeqRef.current += 1
    outputStatsRef.current = {
      chunks: 0,
      bytes: 0,
      chars: 0,
      trimmedChars: 0,
      lastLogAt: terminalNow(),
      lastBytes: 0,
      lastChunks: 0,
    }
    dispatch({ type: 'user.connectMachine', machineId: options.machineId })

    terminalSessionConnectionInfo(options.session).then((info) => {
      if (cancelled) return
      connectionIdRef.current = info.connectionId
      logSession('connection_info', {
        level: 'info',
        details: {
          path: info.path,
          relayInUse: info.relayInUse === true,
          type: info.type,
          rtt: info.rtt,
        },
      })
      dispatch({ type: 'connection.connected', path: info.path, connectionId: info.connectionId })
      dispatch({ type: 'user.openTerminal', machineId: options.machineId, terminalId: options.terminalId })
      return createProtoTerminalProtocolSession(options.session)
    }).then((protocolSession) => {
      if (cancelled || !protocolSession) return
      protocolSessionRef.current = protocolSession
      if (options.historyOnly) {
        return client.connectHistory(options.terminalId, protocolSession).then(() => {
          if (cancelled) return
          setTerminalSnapshot({ text: '', cols: 0, rows: 0 })
          setHistoryReady(true)
        })
      }
      client.connect(options.terminalId, protocolSession)
    }).catch((err: unknown) => {
      if (cancelled) return
      const reason = err instanceof Error ? err.message : String(err)
      logSession('connect_failed', {
        level: 'error',
        details: { reason },
      })
      dispatch({ type: 'connection.failed', reason, recoverable: true, surface: 'banner' })
    })

    return () => {
      cancelled = true
      scrollbackAbortControllerRef.current?.abort(new DOMException('Terminal session closed', 'AbortError'))
      scrollbackAbortControllerRef.current = null
      scrollbackPrefetchPendingRef.current?.controller.abort(new DOMException('Terminal session closed', 'AbortError'))
      scrollbackPrefetchPendingRef.current = null
      scrollbackPrefetchRef.current = null
      scrollbackFrozenRef.current = false
      logSession('unmount', {
        level: 'info',
        details: {
          chunks: outputStatsRef.current.chunks,
          bytes: outputStatsRef.current.bytes,
          chars: outputStatsRef.current.chars,
          trimmedChars: outputStatsRef.current.trimmedChars,
        },
      })
      client.disconnect()
      protocolSessionRef.current = null
      clientRef.current = null
      revokeInputRecoveryOwner(options.session)
      clearInputRecoveryFailureForSession(options.session, inputRecoverySequenceRef.current)
      clearTerminalTextPublishTimer()
      dispatch({ type: 'user.release' })
    }
  }, [callbacks, clearInputRecoveryFailureForSession, clearTerminalTextPublishTimer, logSession, options.historyOnly, options.machineId, options.terminalId, options.session, revokeInputRecoveryOwner])

  const sendInput = useCallback((data: string, size?: { cols: number; rows: number }) => {
    if (options.historyOnly) return false
    const client = clientRef.current
    const session = sessionRef.current
    if (!client || !session.isAlive()) return false

    const message: InputRecoveryMessage = {
      data,
      ...(size ? { size } : {}),
      byteLength: inputTextEncoderRef.current.encode(data).byteLength,
    }
    const currentOwner = inputRecoveryOwnerRef.current
    if (currentOwner && isCurrentInputRecoveryOwner(currentOwner) && currentOwner.phase !== 'recent') {
      const queuedEntries = currentOwner.messages.length - currentOwner.nextIndex
      if (
        queuedEntries >= inputRecoveryMaxEntries ||
        currentOwner.queuedBytes + message.byteLength > inputRecoveryMaxBytes
      ) {
        currentOwner.overflowed = true
        setInputRecoveryOwnerFailure(currentOwner, inputRecoveryOverflowReason)
        return false
      }
      currentOwner.messages.push(message)
      currentOwner.queuedBytes += message.byteLength
      clearInputRecoveryFailureForAcceptedInput(session, currentOwner.sequence)
      return true
    }
    if (currentOwner) clearInputRecoveryOwner(currentOwner)

    let owner: InputRecoveryOwner | null = null
    if (message.byteLength <= inputRecoveryMaxBytes) {
      const sequence = inputRecoverySequenceRef.current + 1
      inputRecoverySequenceRef.current = sequence
      const recentOwner: InputRecoveryOwner = {
        session,
        sequence,
        phase: 'recent',
        messages: [message],
        nextIndex: 0,
        queuedBytes: message.byteLength,
        overflowed: false,
        expiryTimer: null,
      }
      owner = recentOwner
      inputRecoveryOwnerRef.current = recentOwner
      recentOwner.expiryTimer = setTimeout(() => {
        clearInputRecoveryOwner(recentOwner)
      }, recentInputRecoveryWindowMs)
    }

    const sent = client.sendInput(data, size)
    if (sent) {
      clearInputRecoveryFailureForAcceptedInput(session, owner?.sequence ?? inputRecoverySequenceRef.current)
      return true
    }
    if (!owner) {
      const sequence = inputRecoverySequenceRef.current + 1
      inputRecoverySequenceRef.current = sequence
      setCurrentInputRecoveryFailure(
        session,
        sequence,
        'Terminal input was not accepted and exceeds the recovery buffer limit',
      )
      return false
    }
    if (!session.isAlive()) {
      clearInputRecoveryOwner(owner)
      return false
    }
    if (owner.phase === 'recent') {
      startInputRecoveryOwner(owner, 'terminal input send failed')
    }
    const accepted = isCurrentInputRecoveryOwner(owner) && owner.phase !== 'recent'
    if (accepted) clearInputRecoveryFailureForAcceptedInput(session, owner.sequence)
    return accepted
  }, [clearInputRecoveryFailureForAcceptedInput, clearInputRecoveryOwner, isCurrentInputRecoveryOwner, options.historyOnly, setCurrentInputRecoveryFailure, setInputRecoveryOwnerFailure, startInputRecoveryOwner])

  const sendResize = useCallback((cols: number, rows: number) => {
    if (options.historyOnly) return false
    return clientRef.current?.sendResize(cols, rows) ?? false
  }, [logSession, options.historyOnly])

  const requestResizeOwner = useCallback((size?: { cols: number; rows: number }) => {
    return clientRef.current?.requestResizeOwner(size) ?? Promise.reject(new Error('terminal client is not connected'))
  }, [])

  const releaseResizeOwner = useCallback(() => {
    return clientRef.current?.releaseResizeOwner() ?? Promise.reject(new Error('terminal client is not connected'))
  }, [])

  const markSyncLost = useCallback((reason?: string) => {
    clientRef.current?.markSyncLost(reason)
  }, [])

  const markLiveScreenSubmitted = useCallback((revision: bigint) => {
    clientRef.current?.markLiveScreenSubmitted(revision)
  }, [])

  const markLiveScreenCompleted = useCallback((revision: bigint) => {
    clientRef.current?.markLiveScreenCompleted(revision)
  }, [])

  const prefetchScrollback = useCallback(async (limit = 100, alternate = false, cols?: number): Promise<boolean> => {
    const normalizedLimit = Math.max(0, Math.trunc(limit))
    const normalizedCols = Math.max(0, Math.trunc(cols ?? 0))
    if (normalizedLimit <= 0 || alternate || normalizedCols <= 0) return false
    if (scrollbackFrozenRef.current || loadingScrollbackRef.current) return false
    const client = clientRef.current
    if (!client) return false

    const key = scrollbackPrefetchKey(normalizedLimit, alternate, normalizedCols)
    const revision = liveHistoryRevisionRef.current
    const cached = scrollbackPrefetchRef.current
    if (cached?.key === key && cached.revision === revision) return true
    const existing = scrollbackPrefetchPendingRef.current
    if (existing?.key === key && existing.revision === revision) {
      return (await existing.promise) !== null
    }

    if (cached || existing) {
      scrollbackPrefetchRef.current = null
      existing?.controller.abort(new DOMException('Terminal history prefetch parameters changed', 'AbortError'))
      client.resetScrollback()
      if (existing) await existing.promise
    }
    if (
      scrollbackFrozenRef.current ||
      loadingScrollbackRef.current ||
      liveHistoryRevisionRef.current !== revision ||
      clientRef.current !== client
    ) return false

    const controller = new AbortController()
    const timeout = setTimeout(() => {
      controller.abort(new Error(`Terminal history prefetch timed out after ${scrollbackLoadTimeoutMs}ms`))
    }, scrollbackLoadTimeoutMs)
    const startedAt = terminalNow()
    logSession('scrollback_prefetch_start', {
      details: { limit: normalizedLimit, cols: normalizedCols, revision },
    })

    let pending!: ScrollbackPrefetchPending
    const promise = (async (): Promise<TerminalScrollbackPage | null> => {
      try {
        const page = await client.loadScrollback(0, normalizedLimit, false, {
          signal: controller.signal,
          cols: normalizedCols,
        })
        if (
          controller.signal.aborted ||
          liveHistoryRevisionRef.current !== revision ||
          clientRef.current !== client
        ) {
          client.resetScrollback()
          return null
        }
        scrollbackPrefetchRef.current = { key, revision, page }
        logSession('scrollback_prefetch_success', {
          details: {
            elapsedMs: Math.round(terminalNow() - startedAt),
            rows: page.rows,
            hasMore: page.hasMore,
            replayChars: page.replay.length,
            cols: normalizedCols,
            revision,
          },
        })
        return page
      } catch (error) {
        client.resetScrollback()
        if (!controller.signal.aborted) {
          logSession('scrollback_prefetch_failed', {
            level: 'debug',
            details: {
              elapsedMs: Math.round(terminalNow() - startedAt),
              reason: error instanceof Error ? error.message : String(error),
            },
          })
        }
        return null
      } finally {
        clearTimeout(timeout)
        if (scrollbackPrefetchPendingRef.current === pending) {
          scrollbackPrefetchPendingRef.current = null
        }
      }
    })()
    pending = { key, revision, controller, promise }
    scrollbackPrefetchPendingRef.current = pending
    return (await promise) !== null
  }, [logSession])

  const consumePrefetchedScrollback = useCallback(async (limit: number, alternate: boolean, cols?: number): Promise<TerminalScrollbackPage | null> => {
    const key = scrollbackPrefetchKey(limit, alternate, cols)
    const revision = liveHistoryRevisionRef.current
    let cached = scrollbackPrefetchRef.current
    if (cached?.key === key && cached.revision === revision) {
      scrollbackPrefetchRef.current = null
      logSession('scrollback_prefetch_consumed', {
        details: { rows: cached.page.rows, cols, revision },
      })
      return cached.page
    }

    const pending = scrollbackPrefetchPendingRef.current
    if (pending?.key === key && pending.revision === revision) {
      await pending.promise
      cached = scrollbackPrefetchRef.current
      if (cached?.key === key && cached.revision === revision) {
        scrollbackPrefetchRef.current = null
        logSession('scrollback_prefetch_consumed', {
          details: { rows: cached.page.rows, cols, revision, waited: true },
        })
        return cached.page
      }
      return null
    }

    if (cached || pending) {
      scrollbackPrefetchRef.current = null
      pending?.controller.abort(new DOMException('Terminal history prefetch no longer matches', 'AbortError'))
      clientRef.current?.resetScrollback()
      if (pending) await pending.promise
    }
    return null
  }, [logSession])

  const freezeScrollback = useCallback(() => {
    scrollbackFrozenRef.current = true
    clientRef.current?.setLiveScreenDemand(false)
  }, [])

  const resumeLiveScrollback = useCallback(() => {
    scrollbackFrozenRef.current = false
    clientRef.current?.setLiveScreenDemand(true)
    scrollbackAbortControllerRef.current?.abort(new DOMException('Terminal history view resumed live output', 'AbortError'))
    scrollbackAbortControllerRef.current = null
    scrollbackPrefetchPendingRef.current?.controller.abort(new DOMException('Terminal history view resumed live output', 'AbortError'))
    scrollbackPrefetchPendingRef.current = null
    scrollbackPrefetchRef.current = null
    clientRef.current?.resetScrollback()
    scrollbackPrefixTextRef.current = ''
    loadedScrollbackRowsRef.current = 0
    scrollbackRowTimestampsRef.current = []
    scrollbackRowLineIdsRef.current = []
    scrollbackRowInLinesRef.current = []
    scrollbackRowStartColsRef.current = []
    hasMoreScrollbackRef.current = true
    activeScrollbackModeRef.current = 'normal'
    const liveText = terminalContentTextRef.current
    terminalTextRef.current = liveText
    setTerminalSnapshot((current) => {
      if (!current?.history) return current
      const { history: _history, ...liveSnapshot } = current
      return liveSnapshot
    })
    publishTerminalTextNow()
    return liveText
  }, [publishTerminalTextNow])

  const resetScrollback = useCallback(() => {
    scrollbackAbortControllerRef.current?.abort(new DOMException('Terminal history is being reloaded', 'AbortError'))
    scrollbackAbortControllerRef.current = null
    scrollbackPrefetchPendingRef.current?.controller.abort(new DOMException('Terminal history is being reloaded', 'AbortError'))
    scrollbackPrefetchPendingRef.current = null
    scrollbackPrefetchRef.current = null
    clientRef.current?.resetScrollback()
    loadedScrollbackRowsRef.current = 0
    scrollbackRowTimestampsRef.current = []
    scrollbackRowLineIdsRef.current = []
    scrollbackRowInLinesRef.current = []
    scrollbackRowStartColsRef.current = []
    hasMoreScrollbackRef.current = true
    activeScrollbackModeRef.current = 'normal'
  }, [])

  const loadScrollback = useCallback(async (limit = 100, alternate = false, cols?: number): Promise<TerminalScrollbackLoadResult> => {
    scrollbackFrozenRef.current = true
    clientRef.current?.setLiveScreenDemand(false)
    const mode: 'normal' | 'alternate' = alternate ? 'alternate' : 'normal'
    if (activeScrollbackModeRef.current !== mode) {
      activeScrollbackModeRef.current = mode
      loadedScrollbackRowsRef.current = 0
      scrollbackRowTimestampsRef.current = []
      scrollbackRowLineIdsRef.current = []
      scrollbackRowInLinesRef.current = []
      scrollbackRowStartColsRef.current = []
      scrollbackPrefixTextRef.current = ''
      hasMoreScrollbackRef.current = true
    }
    if (loadingScrollbackRef.current || !hasMoreScrollbackRef.current) {
      return {
        loadedRows: 0,
        totalRows: loadedScrollbackRowsRef.current,
        hasMore: hasMoreScrollbackRef.current,
        alternate,
      }
    }
    const client = clientRef.current
    if (!client) {
      return {
        loadedRows: 0,
        totalRows: loadedScrollbackRowsRef.current,
        hasMore: false,
        alternate,
      }
    }
    loadingScrollbackRef.current = true
    const normalizedLimit = Math.max(0, Math.trunc(limit))
    const normalizedCols = Math.max(0, Math.trunc(cols ?? 0))
    let controller: AbortController | null = null
    let timeout: ReturnType<typeof setTimeout> | null = null
    let prefetched = false
    const startedAt = terminalNow()
    logSession('scrollback_load_start', {
      details: {
        offset: loadedScrollbackRowsRef.current,
        limit: normalizedLimit,
        cols: normalizedCols || undefined,
      },
    })
    try {
      let page = loadedScrollbackRowsRef.current === 0
        ? await consumePrefetchedScrollback(normalizedLimit, alternate, normalizedCols || undefined)
        : null
      if (page) {
        prefetched = true
      } else {
        controller = new AbortController()
        scrollbackAbortControllerRef.current = controller
        timeout = setTimeout(() => {
          controller?.abort(new Error(`Terminal history request timed out after ${scrollbackLoadTimeoutMs}ms`))
        }, scrollbackLoadTimeoutMs)
        page = await client.loadScrollback(
          loadedScrollbackRowsRef.current,
          normalizedLimit,
          alternate,
          { signal: controller.signal, ...(normalizedCols > 0 ? { cols: normalizedCols } : {}) },
        )
      }
      const historyMetadata = scrollbackResultMetadata(page)
      const loadedRows = page.rows
      const operation = page.operation ?? (loadedScrollbackRowsRef.current === 0 ? 'replace' : 'prepend')
      if (loadedRows === 0) {
        hasMoreScrollbackRef.current = false
        if (operation === 'replace') {
          loadedScrollbackRowsRef.current = 0
          scrollbackRowTimestampsRef.current = []
          scrollbackRowLineIdsRef.current = []
          scrollbackRowInLinesRef.current = []
          scrollbackRowStartColsRef.current = []
          scrollbackPrefixTextRef.current = ''
          terminalTextRef.current = terminalContentTextRef.current
          const revision = historyRevisionRef.current + 1
          historyRevisionRef.current = revision
          setTerminalSnapshot((current) => current ? {
            ...current,
            history: {
              revision,
              cols: page.cols,
              prependedRows: 0,
              loadedRows: 0,
              operation,
              ...historyMetadata,
              rowTimestampsUnixMs: [],
              rowLogicalLineIds: [],
              rowInLogicalLines: [],
              rowLogicalStartCols: [],
              hasMore: false,
              alternate: page.alternate,
              prefetched,
            },
          } : current)
          publishTerminalTextNow()
        }
        logSession('scrollback_load_empty', {
          details: {
            elapsedMs: Math.round(terminalNow() - startedAt),
            offset: loadedScrollbackRowsRef.current,
            limit: normalizedLimit,
            ...historyMetadata,
            prefetched,
          },
        })
        return {
          loadedRows: 0,
          totalRows: loadedScrollbackRowsRef.current,
          cols: page.cols,
          operation,
          ...historyMetadata,
          hasMore: false,
          alternate: page.alternate,
          prefetched,
        }
      }
      loadedScrollbackRowsRef.current = operation === 'replace'
        ? loadedRows
        : loadedScrollbackRowsRef.current + loadedRows
      const pageTimestamps = page.rowTimestampsUnixMs ?? Array.from({ length: loadedRows }, () => undefined)
      scrollbackRowTimestampsRef.current = operation === 'replace'
        ? pageTimestamps
        : [...pageTimestamps, ...scrollbackRowTimestampsRef.current]
      const pageLineIds = page.rowLogicalLineIds ?? Array.from({ length: loadedRows }, () => undefined)
      scrollbackRowLineIdsRef.current = operation === 'replace'
        ? pageLineIds
        : [...pageLineIds, ...scrollbackRowLineIdsRef.current]
      const pageRowInLines = page.rowInLogicalLines ?? Array.from({ length: loadedRows }, () => undefined)
      scrollbackRowInLinesRef.current = operation === 'replace'
        ? pageRowInLines
        : [...pageRowInLines, ...scrollbackRowInLinesRef.current]
      const pageRowStartCols = page.rowLogicalStartCols ?? Array.from({ length: loadedRows }, () => undefined)
      scrollbackRowStartColsRef.current = operation === 'replace'
        ? pageRowStartCols
        : [...pageRowStartCols, ...scrollbackRowStartColsRef.current]
      hasMoreScrollbackRef.current = page.hasMore
      const revision = historyRevisionRef.current + 1
      historyRevisionRef.current = revision
      const prefix = page.replay
      scrollbackPrefixTextRef.current = operation === 'replace'
        ? prefix
        : joinTerminalTextWithSeparator(prefix, scrollbackPrefixTextRef.current, '\r\n')
      setTerminalSnapshot((current) => current ? {
        ...current,
        history: {
          revision,
          cols: page.cols,
          prependedRows: loadedRows,
          loadedRows: loadedScrollbackRowsRef.current,
          operation,
          ...historyMetadata,
          rowTimestampsUnixMs: scrollbackRowTimestampsRef.current,
          rowLogicalLineIds: scrollbackRowLineIdsRef.current,
          rowInLogicalLines: scrollbackRowInLinesRef.current,
          rowLogicalStartCols: scrollbackRowStartColsRef.current,
          hasMore: page.hasMore,
          alternate: page.alternate,
          prefetched,
        },
      } : current)
      terminalTextRef.current = scrollbackPrefixTextRef.current
      publishTerminalTextNow()
      logSession('scrollback_load_success', {
        details: {
          elapsedMs: Math.round(terminalNow() - startedAt),
          loadedRows,
          totalRows: loadedScrollbackRowsRef.current,
          operation,
          ...historyMetadata,
          hasMore: page.hasMore,
          prefixChars: prefix.length,
          prefetched,
        },
      })
      return {
        loadedRows,
        totalRows: loadedScrollbackRowsRef.current,
        cols: page.cols,
        operation,
        ...historyMetadata,
        hasMore: page.hasMore,
        alternate: page.alternate,
        prefetched,
      }
    } catch (error) {
      logSession('scrollback_load_failed', {
        level: controller?.signal.aborted ? 'warn' : 'error',
        details: {
          elapsedMs: Math.round(terminalNow() - startedAt),
          offset: loadedScrollbackRowsRef.current,
          limit: normalizedLimit,
          reason: error instanceof Error ? error.message : String(error),
        },
      })
      throw error
    } finally {
      if (timeout !== null) clearTimeout(timeout)
      if (controller && scrollbackAbortControllerRef.current === controller) {
        scrollbackAbortControllerRef.current = null
      }
      loadingScrollbackRef.current = false
    }
  }, [consumePrefetchedScrollback, logSession, publishTerminalTextNow])

  const searchScrollback = useCallback(async (
    query: string,
    direction: 'forward' | 'backward',
    cols: number,
    start?: { lineId: string; col: number } | undefined,
  ): Promise<TerminalHistorySearchResult> => {
    const client = clientRef.current
    if (!client) throw new Error('terminal client is not connected')
    scrollbackFrozenRef.current = true
    client.setLiveScreenDemand(false)
    scrollbackAbortControllerRef.current?.abort(new DOMException('Terminal history search superseded', 'AbortError'))
    const controller = new AbortController()
    scrollbackAbortControllerRef.current = controller
    const timeout = setTimeout(() => controller.abort(new Error(`Terminal history search timed out after ${scrollbackLoadTimeoutMs}ms`)), scrollbackLoadTimeoutMs)
    try {
      const result = await client.searchScrollback(query, direction, cols, 100, start, { signal: controller.signal })
      if (!result.found || !result.page) return result
      const page = result.page
      loadedScrollbackRowsRef.current = page.rows
      scrollbackRowTimestampsRef.current = page.rowTimestampsUnixMs ?? Array.from({ length: page.rows }, () => undefined)
      scrollbackRowLineIdsRef.current = page.rowLogicalLineIds ?? Array.from({ length: page.rows }, () => undefined)
      scrollbackRowInLinesRef.current = page.rowInLogicalLines ?? Array.from({ length: page.rows }, () => undefined)
      scrollbackRowStartColsRef.current = page.rowLogicalStartCols ?? Array.from({ length: page.rows }, () => undefined)
      scrollbackPrefixTextRef.current = page.replay
      hasMoreScrollbackRef.current = page.hasMore
      const revision = historyRevisionRef.current + 1
      historyRevisionRef.current = revision
      const historyMetadata = scrollbackResultMetadata(page)
      setTerminalSnapshot((current) => current ? {
        ...current,
        history: {
          revision,
          cols: page.cols,
          prependedRows: page.rows,
          loadedRows: page.rows,
          operation: 'replace',
          ...historyMetadata,
          rowTimestampsUnixMs: scrollbackRowTimestampsRef.current,
          rowLogicalLineIds: scrollbackRowLineIdsRef.current,
          rowInLogicalLines: scrollbackRowInLinesRef.current,
          rowLogicalStartCols: scrollbackRowStartColsRef.current,
          searchMatchRow: result.matchRow ?? 0,
          hasMore: page.hasMore,
          alternate: false,
        },
      } : current)
      terminalTextRef.current = page.replay
      publishTerminalTextNow()
      return result
    } finally {
      clearTimeout(timeout)
      if (scrollbackAbortControllerRef.current === controller) scrollbackAbortControllerRef.current = null
    }
  }, [publishTerminalTextNow])

  const copyScrollback = useCallback((
    range: { startLineId: string; startCol: number; endLineId: string; endCol: number },
    cols: number,
  ): Promise<string> => {
    const client = clientRef.current
    if (!client) return Promise.reject(new Error('terminal client is not connected'))
    return client.copyScrollback(range, cols)
  }, [])

  const cancelHistorySearch = useCallback(() => {
    scrollbackAbortControllerRef.current?.abort(new DOMException('Terminal history search closed', 'AbortError'))
    scrollbackAbortControllerRef.current = null
  }, [])

  const handleAppResume = useCallback((resumeKind: 'quick' | 'cold' | 'frozen') => {
    dispatch({ type: 'app.resume', resumeKind })
  }, [])

  const reattach = useCallback((session: TerminalSession, reattachOptions: { forceTerminalChannel?: boolean } = {}) => {
    if (options.historyOnly) {
      const protocolSession = createProtoTerminalProtocolSession(session)
      protocolSessionRef.current = protocolSession
      sessionRef.current = session
      setHistoryReady(false)
      void clientRef.current?.connectHistory(options.terminalId, protocolSession).then(() => setHistoryReady(true))
      return
    }
    void recoverTerminalChannel('terminal reattach requested', {
      session,
      forceTerminalChannel: reattachOptions.forceTerminalChannel,
      supersede: true,
      showClosedReason: false,
    })
  }, [options.historyOnly, options.terminalId, recoverTerminalChannel])

  return {
    snapshot,
    inputRecoveryFailure: inputRecoveryFailure?.reason ?? null,
    terminalSnapshot,
    terminalText,
    terminalInfo,
    resizeControl,
    historyReady,
    sendInput,
    sendResize,
    requestResizeOwner,
    releaseResizeOwner,
    loadScrollback,
    searchScrollback,
    copyScrollback,
    cancelHistorySearch,
    prefetchScrollback,
    resetScrollback,
    freezeScrollback,
    resumeLiveScrollback,
    markSyncLost,
    markLiveScreenSubmitted,
    markLiveScreenCompleted,
    handleAppResume,
    reattach,
    client: clientRef.current,
  }
}

export type TerminalSessionMessage = ConnectionMessage

function joinTerminalTextWithSeparator(prefix: string, current: string, separator: string): string {
  if (!prefix) return current
  if (!current) return prefix
  return `${prefix}${separator}${current}`
}

function scrollbackResultMetadata(page: Pick<TerminalScrollbackLoadResult, 'committedTotalRows' | 'logicalTotalRows' | 'historyGeneration' | 'firstRowId' | 'lastRowId' | 'viewportTop'>): Partial<TerminalScrollbackLoadResult> {
  return {
    ...(page.committedTotalRows !== undefined ? { committedTotalRows: page.committedTotalRows } : {}),
    ...(page.logicalTotalRows !== undefined ? { logicalTotalRows: page.logicalTotalRows } : {}),
    ...(page.historyGeneration !== undefined ? { historyGeneration: page.historyGeneration } : {}),
    ...(page.firstRowId !== undefined ? { firstRowId: page.firstRowId } : {}),
    ...(page.lastRowId !== undefined ? { lastRowId: page.lastRowId } : {}),
    ...(page.viewportTop !== undefined ? { viewportTop: page.viewportTop } : {}),
  }
}

function terminalSessionConnectionInfo(session: TerminalSession): Promise<ConnectionInfo> {
  return createProtoTerminalProtocolSession(session).getConnectionInfo()
}

function closeRtcTerminalDataChannel(session: TerminalSession, terminalId: string): void {
  void session
  void terminalId
}

function shouldRecoverRecentInput(reason?: string): boolean {
  if (!reason) return false
  return /not open|send failed|unavailable|input send failed|not attached|readonly/i.test(reason)
}

/** shouldRecoverTerminalChannel 只允许同一存活 session 内恢复 terminal channel；generation 重建由上层 owner 负责。 */
export function shouldRecoverTerminalChannel(session: Pick<ProtoClientSession, 'isAlive'>, reason?: string): boolean {
  if (!session.isAlive() || !reason) return false
  if (/terminal (?:has )?exited|terminal removed|terminal not found/i.test(reason)) return false
  return /terminal data channel|terminal channel send failed|terminal protocol .* timed out|native bridge|not open|closed|timed out|send failed|not attached|attachment channel|ensure resize requires attachment|unavailable|interrupted|disconnected|stale|client session|transport/i.test(reason)
}
