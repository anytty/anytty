import type { ConnectionInfo, ConnectionPolicy, ConnectionPolicyState, MachineConnectionSnapshot, ProtoClientSession, ProtoClientSubscription, ProtoResourceStream, RtcConnectOptions, RtcConnectionStateSnapshot } from '@anytty/ui'
import type { CommandEnvelope, EventEnvelope, ResultEnvelope } from '../../ui/src/generated/apipb/application_pb'
import type { EndpointSessionStamp, ResourceHandle } from '../../ui/src/generated/apipb/common_pb'
import { ConnectionObservedPath, ConnectionRouteKind } from '../../ui/src/generated/bindingpb/client_binding_pb'

type ProtoClientSessionCloseHandler = Parameters<ProtoClientSession['subscribeClosed']>[0]
type ProtoClientSessionCloseError = Parameters<ProtoClientSessionCloseHandler>[0]

// Android 总窗口覆盖 route planning、signaling/answer、ICE、鉴权和 Hello；
// Go peer 仍用独立 deadline 约束 answer 之后的 ICE/DataChannel，不能由 UI 提前截断。
const NATIVE_SESSION_READY_TIMEOUT_MS = 45_000
const NATIVE_SESSION_DEFAULT_PROBE_TIMEOUT_MS = 3_000
const NATIVE_SESSION_INVALIDATION_TIMEOUT_MS = 8_000
const NATIVE_RECONNECT_BACKOFF_CAPS_MS = [0, 500, 2_000, 4_000, 8_000, 15_000] as const

/** NativeSessionConnector 是 Android UI 到 Go binding session 的窄连接入口，不拥有 route 或 generation 真值。 */
export type NativeSessionConnector = {
  connect(input: { machineId: string }, options?: RtcConnectOptions): Promise<ProtoClientSession>
  getConnectionPolicy?(signal?: AbortSignal): Promise<ConnectionPolicyState>
  applyConnectionPolicy?(policy: ConnectionPolicy, signal?: AbortSignal): Promise<void>
  verify?(session: ProtoClientSession, signal: AbortSignal): Promise<void>
  disconnect?(machineId: string): Promise<void>
  release?(machineId: string): Promise<void>
  setActive?(machineId: string, active: boolean): Promise<void>
  isGoManaged?(machineId: string): boolean
  requestGoRecovery?(): Promise<void>
}

export type NativeSessionManagerOptions = {
  verifyOnFirstAcquire?: boolean
  initiallyConnected?: boolean
  random?: () => number
  waitForForeground?: (signal?: AbortSignal) => Promise<void>
}

export type NativeNetworkChangeReason = 'available' | 'offline' | 'network_replaced' | 'path_changed'

type PendingNativeNetworkChange = {
  connected: boolean
  reason: NativeNetworkChangeReason
  revision: number
}

/**
 * NativeSessionManager 是单个 Endpoint 在 Android UI generation 内的 session lease owner。
 *
 * Go Client Engine 仍拥有 route、PeerSession 与 generation 真值；这里仅保证 workspace、inventory
 * 和文件传输复用同一个 binding session handle，避免 UI 消费者各自 openSession 后互相使 generation
 * 失效。workspace、任务和临时 lease 共同持有按需连接；最后一个持有者释放后关闭底层 session。
 * reset 仍用于 Android generation 更换、显式重连或 Endpoint 配置失效。
 */
export class NativeSessionManager {
  private session: ProtoClientSession | null = null
  private pending: Promise<ProtoClientSession> | null = null
  private pendingController: AbortController | null = null
  private invalidationBarrier: Promise<void> | null = null
  private sessionClosedSubscription: ProtoClientSubscription | null = null
  private epoch = 0
  private reconnectAttempt = 0
  private reconnectBackoffIndex = 0
  private reconnectTimer: ReturnType<typeof globalThis.setTimeout> | null = null
  private readonly demandOwners = new Set<symbol>()
  private demandRevision = 0
  private networkChangeRevision = 0
  private networkChangeWorker: Promise<void> | null = null
  private activeNetworkChange: PendingNativeNetworkChange | null = null
  private pendingNetworkChange: PendingNativeNetworkChange | null = null
  private keepAliveRequested = false
  private keepAliveQueue: Promise<void> = Promise.resolve()
  private lastKeepAliveUpdate: Promise<void> = Promise.resolve()
  private networkConnected: boolean
  private verifyNextOpenedSession: boolean
  private verification: {
    session: ProtoClientSession
    controller: AbortController
    promise: Promise<void>
  } | null = null
  private snapshot: MachineConnectionSnapshot
  private readonly stateListeners = new Set<() => void>()
  private readonly random: () => number
  private readonly waitForForeground: ((signal?: AbortSignal) => Promise<void>) | undefined

  readonly connectionState = {
    getSnapshot: (): MachineConnectionSnapshot => this.snapshot,
    subscribe: (listener: () => void): (() => void) => {
      this.stateListeners.add(listener)
      return () => this.stateListeners.delete(listener)
    },
  }

  constructor(
    private readonly machineId: string,
    private readonly connector: NativeSessionConnector,
    options: NativeSessionManagerOptions = {},
  ) {
    this.networkConnected = options.initiallyConnected !== false
    this.verifyNextOpenedSession = options.verifyOnFirstAcquire === true
    this.random = options.random ?? Math.random
    this.waitForForeground = options.waitForForeground
    this.snapshot = this.networkConnected
      ? idleMachineConnectionSnapshot(machineId)
      : waitingNetworkMachineConnectionSnapshot(machineId)
  }

  /** machineID 仅供 generation owner 释放同一 Endpoint 的 connector 资源。 */
  machineID(): string { return this.machineId }

  /** Prevents an acquired endpoint session from becoming unused until this owner is released. */
  retainConnectionDemand(): () => void {
    const owner = Symbol(this.machineId)
    this.demandOwners.add(owner)
    this.demandRevision += 1
    let released = false
    return () => {
      if (released) return
      released = true
      if (!this.demandOwners.delete(owner)) return
      const revision = ++this.demandRevision
      if (this.demandOwners.size === 0) {
        queueMicrotask(() => { void this.disconnectWhenUnused(revision) })
      }
    }
  }

  hasConnectionDemand(): boolean { return this.demandOwners.size > 0 }

  /** get 为 inventory 和文件传输取得当前 Endpoint 的独立 UI lease。 */
  get(options?: RtcConnectOptions): Promise<ProtoClientSession> {
    return this.acquireForConsumer(options)
  }

  /** probe temporarily verifies an endpoint and releases both its UI lease and connection demand. */
  async probe(): Promise<ConnectionInfo | null> {
    const lease = await this.acquireForConsumer()
    try {
      return this.connectionState.getSnapshot().connectionInfo
    } finally {
      await lease.close()
    }
  }

  /** lease 为 workspace 取得当前 Endpoint 的独立 UI lease。 */
  lease(options?: RtcConnectOptions): Promise<ProtoClientSession> {
    return this.acquireForConsumer(options)
  }

  /** reset 在 native generation 更换、显式重连或 Endpoint 配置失效时关闭唯一底层 binding session。 */
  async reset(): Promise<void> {
    this.resetOwnedSession(false)
    await this.lastKeepAliveUpdate
  }

  /** disconnect invalidates the pooled physical generation after explicit or last-owner release. */
  async disconnect(): Promise<void> {
    const session = this.resetOwnedSession(false, false)
    const keepAliveUpdate = this.lastKeepAliveUpdate
    const previous = this.invalidationBarrier ?? Promise.resolve()
    const barrier = previous.catch(() => undefined).then(async () => {
      try {
        await keepAliveUpdate
        if (this.connector.disconnect) {
          await withDeadline(
            this.connector.disconnect(this.machineId),
            NATIVE_SESSION_INVALIDATION_TIMEOUT_MS,
            'endpoint disconnect timed out',
          )
        } else if (session?.invalidate) {
          await withDeadline(
            session.invalidate(),
            NATIVE_SESSION_INVALIDATION_TIMEOUT_MS,
            'session invalidation timed out',
          )
        }
      } catch (error) {
        if (!isAlreadyInvalidatedError(error)) throw error
      } finally {
        void session?.close().catch(() => undefined)
      }
    })
    this.invalidationBarrier = barrier
    void barrier.finally(() => {
      if (this.invalidationBarrier === barrier) this.invalidationBarrier = null
    }).catch(() => undefined)
    await barrier
  }

  /** resetClientOnly 响应 UI 重连请求，但仍由 Go 在下一次 openSession 时重新选择 route。 */
  async resetClientOnly(_options?: { forceRelay?: boolean }): Promise<void> {
    if (this.waitForForeground) await this.waitForForeground()
    if (this.goOwnsMaintenance()) {
      this.publish(networkTransitionMachineConnectionSnapshot(
        this.machineId,
        this.snapshot.forceRelay,
        this.reconnectAttempt,
        'reconnecting',
        'Reconnecting...',
      ))
      await this.connector.requestGoRecovery?.()
      this.resetOwnedSession(true)
      if (this.networkConnected && this.keepAliveRequested) {
        await this.acquire({ forceRelay: this.snapshot.forceRelay }).then((lease) => lease.close())
      }
      return
    }
    const session = this.resetOwnedSession(
      true,
      false,
      networkTransitionMachineConnectionSnapshot(
        this.machineId,
        this.snapshot.forceRelay,
        this.reconnectAttempt,
        'reconnecting',
        'Reconnecting...',
      ),
    )
    if (!session) return
    await this.invalidateDetachedSession(session)
  }

  /** Native network callbacks are route hints; the live application path decides whether to reconnect. */
  async networkChanged(
    connected = true,
    reason: NativeNetworkChangeReason = connected ? 'path_changed' : 'offline',
  ): Promise<void> {
    this.networkConnected = connected
    if (this.goOwnsMaintenance()) {
      this.verifyNextOpenedSession = false
      await this.applyGoManagedNetworkChange(connected)
      return
    }
    this.verifyNextOpenedSession = true
    const change: PendingNativeNetworkChange = {
      connected,
      reason,
      revision: ++this.networkChangeRevision,
    }
    if (this.networkChangeWorker) {
      this.pendingNetworkChange = mergeNativeNetworkChanges(
        this.activeNetworkChange,
        this.pendingNetworkChange,
        change,
      )
      if (!connected || reason === 'network_replaced') {
        this.verification?.controller.abort(new Error('native network identity changed'))
        this.pendingController?.abort(new Error('native network identity changed while connecting'))
      }
      return this.networkChangeWorker
    }

    const worker = Promise.resolve().then(() => this.drainNetworkChanges(change))
    this.networkChangeWorker = worker
    return worker
  }

  private async applyGoManagedNetworkChange(connected: boolean): Promise<void> {
    if (!connected) {
      this.publish(waitingNetworkMachineConnectionSnapshot(
        this.machineId,
        this.snapshot.forceRelay,
        this.reconnectAttempt,
      ))
      return
    }
    if (this.session?.isAlive()) {
      this.publish(connectedMachineConnectionSnapshot(
        this.machineId,
        this.session,
        this.snapshot.forceRelay,
        this.reconnectAttempt,
      ))
      return
    }
    if (!this.keepAliveRequested) {
      this.publish(idleMachineConnectionSnapshot(this.machineId, this.reconnectAttempt))
      return
    }
    this.publish(reconnectingMachineConnectionSnapshot(
      this.machineId,
      this.snapshot.forceRelay,
      this.reconnectAttempt,
    ))
    await this.acquire({ forceRelay: this.snapshot.forceRelay }).then((lease) => lease.close())
  }

  private async drainNetworkChanges(initial: PendingNativeNetworkChange): Promise<void> {
    let change: PendingNativeNetworkChange | null = initial
    let lastFailure: unknown
    try {
      while (change) {
        this.activeNetworkChange = change
        try {
          await this.applyNetworkChange(change)
          lastFailure = undefined
        } catch (failure) {
          lastFailure = failure
        }
        change = this.pendingNetworkChange
        this.pendingNetworkChange = null
      }
      if (lastFailure !== undefined) throw lastFailure
    } finally {
      this.activeNetworkChange = null
      this.pendingNetworkChange = null
      this.networkChangeWorker = null
    }
  }

  private async applyNetworkChange(change: PendingNativeNetworkChange): Promise<void> {
    const { connected, reason, revision } = change
    const reconnect = this.keepAliveRequested
    const forceRelay = this.snapshot.forceRelay
    if (!this.session && !this.pending && !reconnect) {
      // OpenSession may reuse a Go-owned pooled session. Verify it once instead of
      // always throwing away a healthy first connection.
      this.publish(connected
        ? idleMachineConnectionSnapshot(this.machineId, this.reconnectAttempt)
        : waitingNetworkMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt))
      return
    }
    if (connected && this.pending && reason !== 'network_replaced') {
      // Native path metadata often arrives in several callbacks. Let the in-flight
      // connection finish once, then verify the resulting pooled generation.
      const pending = this.pending
      const opened = await pending.catch(() => null)
      if (opened && this.verifyNextOpenedSession && this.session === opened && opened.isAlive() && this.connector.verify) {
        await this.verifyCurrentSession(opened, revision)
      }
      return
    }
    if (connected && reason === 'network_replaced' && this.session?.isAlive()) {
      this.verification?.controller.abort(new Error('native network identity changed'))
      this.publish(networkTransitionMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt, 'reconnecting', 'Network changed. Reconnecting...'))
      await this.replaceSession(this.session, forceRelay)
      return
    }
    if (connected && this.session?.isAlive() && this.connector.verify) {
      this.publish(networkTransitionMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt, 'verifying', 'Verifying connection...'))
      await this.verifyCurrentSession(this.session, revision)
      return
    }
    if (connected && this.session?.isAlive() && reason !== 'available') {
      this.publish(networkTransitionMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt, 'reconnecting', 'Reconnecting...'))
      await this.replaceSession(this.session, forceRelay)
      return
    }
    const session = this.resetOwnedSession(
      reconnect,
      false,
      connected
        ? idleMachineConnectionSnapshot(this.machineId, this.reconnectAttempt)
        : waitingNetworkMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt),
    )
    const resetEpoch = this.epoch
    if (session) {
      await this.invalidateDetachedSession(session)
    }
    if (
      !connected ||
      resetEpoch !== this.epoch ||
      !this.keepAliveRequested ||
      !this.networkConnected
    ) return
    await this.acquire({ forceRelay }).then((lease) => lease.close())
  }

  /** Cold-start sampling seeds connectivity without pretending that a network generation changed. */
  async initializeNetworkState(connected: boolean): Promise<void> {
    if (this.session || this.pending || this.keepAliveRequested) {
      await this.networkChanged(connected)
      return
    }
    this.networkConnected = connected
    this.publish(connected
      ? idleMachineConnectionSnapshot(this.machineId, this.reconnectAttempt)
      : waitingNetworkMachineConnectionSnapshot(this.machineId, this.snapshot.forceRelay, this.reconnectAttempt))
  }

  /** foregroundResume verifies the remote application protocol, not merely the local JS-to-Go bridge. */
  async foregroundResume(): Promise<void> {
    if (!this.keepAliveRequested) return
    if (!this.networkConnected) {
      this.publish(waitingNetworkMachineConnectionSnapshot(
        this.machineId,
        this.snapshot.forceRelay,
        this.reconnectAttempt,
      ))
      return
    }
    if (this.goOwnsMaintenance()) {
      this.verifyNextOpenedSession = false
      this.resetOwnedSession(
        true,
        true,
        networkTransitionMachineConnectionSnapshot(
          this.machineId,
          this.snapshot.forceRelay,
          this.reconnectAttempt,
          'reconnecting',
          'Restoring session...',
        ),
      )
      await this.acquire({ forceRelay: this.snapshot.forceRelay }).then((lease) => lease.close())
      return
    }
    const session = this.session
    if (!session?.isAlive()) {
      await this.acquire({ forceRelay: this.snapshot.forceRelay }).then((lease) => lease.close())
      return
    }
    if (!this.connector.verify) return
    await this.verifyCurrentSession(session)
  }

  private verifyCurrentSession(session: ProtoClientSession, networkRevision?: number): Promise<void> {
    if (this.verification?.session === session) return this.verification.promise
    const epoch = this.epoch
    const forceRelay = this.snapshot.forceRelay
    const controller = new AbortController()
    const promise = this.runSessionVerification(session, controller, probeTimeoutMs(session)).then(async () => {
      if (networkRevision !== undefined && networkRevision !== this.networkChangeRevision) return
      if (epoch !== this.epoch || this.session !== session || !session.isAlive()) return
      this.verifyNextOpenedSession = false
      this.publish(connectedMachineConnectionSnapshot(this.machineId, session, forceRelay, this.reconnectAttempt))
    }).catch(async (failure: unknown) => {
      if (networkRevision !== undefined && networkRevision !== this.networkChangeRevision) return
      if (epoch !== this.epoch || this.session !== session) return
      if (!shouldReplaceAfterVerification(failure, session, controller.signal)) {
        if (session.isAlive()) {
          this.verifyNextOpenedSession = false
          this.publish(connectedMachineConnectionSnapshot(this.machineId, session, forceRelay, this.reconnectAttempt))
        }
        return
      }
      await this.replaceSession(session, forceRelay).catch((reconnectFailure) => {
        throw reconnectFailure ?? failure
      })
    }).finally(() => {
      if (this.verification?.promise === promise) this.verification = null
    })
    this.verification = { session, controller, promise }
    return promise
  }

  private async runSessionVerification(
    session: ProtoClientSession,
    controller: AbortController,
    timeoutMs: number,
  ): Promise<void> {
    const timeout = globalThis.setTimeout(() => controller.abort(new Error('session verification timed out')), timeoutMs)
    try {
      await this.connector.verify?.(session, controller.signal)
    } finally {
      globalThis.clearTimeout(timeout)
    }
  }

  private async replaceSession(session: ProtoClientSession, forceRelay: boolean): Promise<void> {
    if (this.session !== session) return
    const reconnect = this.keepAliveRequested
    const detached = this.resetOwnedSession(
      reconnect,
      false,
      networkTransitionMachineConnectionSnapshot(
        this.machineId,
        forceRelay,
        this.reconnectAttempt,
        'reconnecting',
        'Reconnecting...',
      ),
    )
    const replacementEpoch = this.epoch
    if (detached) {
      await this.invalidateDetachedSession(detached)
    }
    if (
      replacementEpoch === this.epoch &&
      this.keepAliveRequested &&
      this.networkConnected
    ) {
      await this.acquire({ forceRelay }).then((lease) => lease.close())
    }
  }

  private resetOwnedSession(
    preserveKeepAlive: boolean,
    closeSession = true,
    nextSnapshot?: MachineConnectionSnapshot,
  ): ProtoClientSession | null {
    this.epoch += 1
    this.verification?.controller.abort(new Error('native session generation changed'))
    this.verification = null
    this.clearReconnectTimer()
    this.reconnectBackoffIndex = 0
    if (!preserveKeepAlive) {
      const deactivation = this.setKeepAliveRequested(false)
      if (deactivation) void deactivation.catch(() => undefined)
    }
    const session = this.session
    const pending = this.pending
    const pendingController = this.pendingController
    const sessionClosedSubscription = this.sessionClosedSubscription
    this.session = null
    this.pending = null
    this.pendingController = null
    this.sessionClosedSubscription = null
    sessionClosedSubscription?.close()
    this.publish(nextSnapshot ?? idleMachineConnectionSnapshot(this.machineId, this.reconnectAttempt))
    pendingController?.abort(new Error('native session generation changed while connecting'))
    // Generation replacement cannot wait for a transport on the old network. The epoch fence
    // closes late results, while the new peer is free to use the current network snapshot.
    if (closeSession) void session?.close().catch(() => undefined)
    if (pending) {
      void pending.then((late) => late.close(), () => undefined).catch(() => undefined)
    }
    return session
  }

  private invalidateDetachedSession(session: ProtoClientSession, strict = false): Promise<void> {
    const previous = this.invalidationBarrier ?? Promise.resolve()
    const barrier = previous.catch(() => undefined).then(async () => {
      try {
        if (session.invalidate) {
          await withDeadline(
            session.invalidate(),
            NATIVE_SESSION_INVALIDATION_TIMEOUT_MS,
            'session invalidation timed out',
          )
        }
      } catch (error) {
        if (strict && !isAlreadyInvalidatedError(error)) throw error
      } finally {
        void session.close().catch(() => undefined)
      }
    })
    this.invalidationBarrier = barrier
    void barrier.finally(() => {
      if (this.invalidationBarrier === barrier) this.invalidationBarrier = null
    }).catch(() => undefined)
    return barrier
  }

  /** Business consumers wait for the foreground generation and survive manager-owned transient retries. */
  private async acquireForConsumer(options?: RtcConnectOptions): Promise<ProtoClientSession> {
    const releaseDemand = this.retainConnectionDemand()
    try {
      while (true) {
        if (this.waitForForeground) await this.waitForForeground(options?.signal)
        try {
          return await this.acquire(options, releaseDemand)
        } catch (failure) {
          if (options?.signal?.aborted) throw abortError(options.signal)
          const error = connectionFailure(failure)
          if (
            this.goOwnsMaintenance() &&
            this.keepAliveRequested &&
            this.networkConnected &&
            shouldKeepGoConsumerPending(error)
          ) {
            await waitForRetryWindow(options?.signal)
            continue
          }
          if (!this.keepAliveRequested || !this.networkConnected || !shouldKeepConsumerPending(error)) throw failure
          await this.waitForRecoverableSession(options?.signal)
        }
      }
    } catch (failure) {
      releaseDemand()
      throw failure
    }
  }

  private waitForRecoverableSession(signal?: AbortSignal): Promise<void> {
    if (signal?.aborted) return Promise.reject(abortError(signal))
    return new Promise<void>((resolve, reject) => {
      let settled = false
      let unsubscribe = () => {}
      const finish = (failure?: unknown) => {
        if (settled) return
        settled = true
        unsubscribe()
        signal?.removeEventListener('abort', abort)
        if (failure) reject(failure)
        else resolve()
      }
      const evaluate = () => {
        if (signal?.aborted) {
          finish(abortError(signal))
          return
        }
        if (this.snapshot.phase === 'connected') {
          finish()
          return
        }
        if (this.snapshot.phase === 'waiting_network') {
          finish(networkUnavailableError())
          return
        }
        if (this.snapshot.phase === 'failed') {
          finish(this.snapshot.error ?? new Error(this.snapshot.statusText || 'Connection failed'))
          return
        }
        if (this.snapshot.phase === 'idle' && !this.keepAliveRequested) {
          finish(Object.assign(new Error('native session acquisition was superseded'), { code: 'cancelled' }))
        }
      }
      const abort = () => finish(signal ? abortError(signal) : new DOMException('Aborted', 'AbortError'))
      unsubscribe = this.connectionState.subscribe(evaluate)
      signal?.addEventListener('abort', abort, { once: true })
      evaluate()
    })
  }

  private async acquire(options?: RtcConnectOptions, releaseDemand?: () => void): Promise<ProtoClientSession> {
    const acquireEpoch = this.epoch
    this.clearReconnectTimer()
    const activation = this.setKeepAliveRequested(true)
    if (activation) await activation
    if (this.goOwnsMaintenance()) this.verifyNextOpenedSession = false
    const stopForwarding = this.forwardState(options)
    try {
      if (!this.networkConnected) {
        this.publish(waitingNetworkMachineConnectionSnapshot(
          this.machineId,
          options?.forceRelay === true || this.snapshot.forceRelay,
          this.reconnectAttempt,
        ))
        throw networkUnavailableError()
      }
      const invalidation = this.invalidationBarrier
      if (invalidation) await invalidation
      if (acquireEpoch !== this.epoch) {
        throw new Error('native session generation changed while connecting')
      }
      if (!this.networkConnected) throw networkUnavailableError()
      if (this.session?.isAlive()) {
        this.publish(connectedMachineConnectionSnapshot(this.machineId, this.session, this.snapshot.forceRelay, this.reconnectAttempt))
        return new NativeSessionLease(this.session, this.waitForForeground, releaseDemand)
      }
      const staleSession = this.session
      if (staleSession) {
        this.sessionClosedSubscription?.close()
        this.sessionClosedSubscription = null
        await staleSession.close().catch(() => undefined)
        if (this.session === staleSession) this.session = null
        if (acquireEpoch !== this.epoch) {
          throw new Error('native session generation changed while connecting')
        }
        if (this.session?.isAlive()) {
          this.publish(connectedMachineConnectionSnapshot(this.machineId, this.session, this.snapshot.forceRelay, this.reconnectAttempt))
          return new NativeSessionLease(this.session, this.waitForForeground, releaseDemand)
        }
      }
      if (!this.pending) {
        const epoch = this.epoch
        this.reconnectAttempt += 1
        this.publish({
          machineId: this.machineId,
          phase: 'connecting',
          statusText: 'Connecting...',
          connectionInfo: null,
          forceRelay: options?.forceRelay === true,
          relayInUse: false,
          reconnectAttempt: this.reconnectAttempt,
          error: null,
        })
        // 底层 connect 属于 manager，而不是任一 UI lease；单个 consumer 只能取消自己的等待。
        // manager-owned signal 同时约束完整 binding operation，并在 generation reset 时主动释放旧 Go attempt。
        const controller = new AbortController()
        this.pendingController = controller
        const timeout = globalThis.setTimeout(() => controller.abort(new Error('client session timed out')), NATIVE_SESSION_READY_TIMEOUT_MS)
        const connectOptions: RtcConnectOptions = {
          forceRelay: options?.forceRelay,
          signal: controller.signal,
          onStatus: (status) => {
            if (epoch !== this.epoch) return
            this.publish({ ...this.snapshot, statusText: status })
          },
          onConnectionState: (snapshot) => {
            if (epoch !== this.epoch) return
            const projected = connectionStateSnapshot(this.machineId, snapshot, options?.forceRelay === true, this.reconnectAttempt)
            if (this.goOwnsMaintenance() && projected.phase === 'connected') {
              // SessionOwner publishes its transport winner before the supervisor's
              // application probe completes. Only the resolved OpenSession may expose
              // READY to renderer state.
              this.publish(reconnectingMachineConnectionSnapshot(
                this.machineId,
                options?.forceRelay === true,
                this.reconnectAttempt,
              ))
              return
            }
            if (
              projected.phase === 'failed' &&
              projected.error &&
              this.keepAliveRequested &&
              this.networkConnected &&
              shouldRecoverWithoutFailure(projected.error)
            ) {
              this.publish(reconnectingMachineConnectionSnapshot(
                this.machineId,
                options?.forceRelay === true,
                this.reconnectAttempt,
              ))
              return
            }
            this.publish(projected)
          },
        }
        const connect = async (): Promise<ProtoClientSession> => {
          let opened = await this.connector.connect({ machineId: this.machineId }, connectOptions)
          if (!this.verifyNextOpenedSession || !this.connector.verify) return opened
          const verification = new AbortController()
          const abortVerification = () => verification.abort(controller.signal.reason)
          controller.signal.addEventListener('abort', abortVerification, { once: true })
          try {
            try {
              await this.runSessionVerification(opened, verification, probeTimeoutMs(opened))
            } catch (failure) {
              if (epoch !== this.epoch || controller.signal.aborted) {
                this.verifyNextOpenedSession = true
                await this.invalidateDetachedSession(opened)
                throw new Error('native session generation changed while connecting', { cause: failure })
              }
              if (!shouldReplaceAfterVerification(failure, opened, verification.signal)) {
                this.verifyNextOpenedSession = false
                return opened
              }
              await this.invalidateDetachedSession(opened)
              if (epoch !== this.epoch) throw new Error('native session generation changed while connecting', { cause: failure })
              opened = await this.connector.connect({ machineId: this.machineId }, connectOptions)
            }
            this.verifyNextOpenedSession = false
          } finally {
            controller.signal.removeEventListener('abort', abortVerification)
          }
          return opened
        }
        let reconnectFailure: Error | null = null
        const pending = connect().then(async (opened) => {
          if (epoch !== this.epoch) {
            await this.invalidateDetachedSession(opened)
            throw new Error('native session generation changed while connecting')
          }
          if (!opened.isAlive()) throw new Error('Go client session is unavailable')
          this.session = opened
          this.sessionClosedSubscription = opened.subscribeClosed((error) => {
            this.handleSessionClosed(epoch, opened, error)
          })
          this.reconnectBackoffIndex = 0
          this.publish(connectedMachineConnectionSnapshot(this.machineId, opened, options?.forceRelay === true, this.reconnectAttempt))
          return opened
        }).catch((error: unknown) => {
          if (epoch === this.epoch) {
            const failure = connectionFailure(error)
            reconnectFailure = failure
            this.publish(shouldRecoverWithoutFailure(failure) && this.keepAliveRequested && this.networkConnected
              ? reconnectingMachineConnectionSnapshot(this.machineId, options?.forceRelay === true, this.reconnectAttempt)
              : failedMachineConnectionSnapshot(this.machineId, failure, options?.forceRelay === true, false, this.reconnectAttempt))
          }
          throw error
        })
        this.pending = pending
        void pending.finally(() => {
          globalThis.clearTimeout(timeout)
          if (this.pendingController === controller) this.pendingController = null
          if (this.pending === pending) this.pending = null
          if (epoch === this.epoch && reconnectFailure) {
            this.scheduleReconnect(epoch, options?.forceRelay === true, reconnectFailure)
          }
        }).catch(() => undefined)
      }
      const opened = await awaitSessionLease(this.pending, options?.signal)
      if (!opened.isAlive()) throw new Error('Go client session is unavailable')
      return new NativeSessionLease(opened, this.waitForForeground, releaseDemand)
    } finally {
      stopForwarding()
    }
  }

  private forwardState(options: RtcConnectOptions | undefined): () => void {
    if (!options?.onConnectionState && !options?.onStatus) return () => {}
    const forward = () => {
      const snapshot = this.snapshot
      options.onStatus?.(snapshot.statusText)
      options.onConnectionState?.({
        machineId: snapshot.machineId,
        phase: snapshot.phase,
        ...(snapshot.connectionInfo?.path ? { path: snapshot.connectionInfo.path } : {}),
        ...(snapshot.connectionInfo?.observedPath ? { observedPath: snapshot.connectionInfo.observedPath } : {}),
        ...(snapshot.connectionInfo?.routeSelectionReason ? { routeSelectionReason: snapshot.connectionInfo.routeSelectionReason } : {}),
        statusText: snapshot.statusText,
        relayInUse: snapshot.relayInUse,
        ...(snapshot.error ? { error: snapshot.error } : {}),
      })
    }
    if (this.snapshot.phase !== 'idle') forward()
    this.stateListeners.add(forward)
    return () => this.stateListeners.delete(forward)
  }

  private publish(snapshot: MachineConnectionSnapshot): void {
    this.snapshot = snapshot
    for (const listener of this.stateListeners) listener()
  }

  private handleSessionClosed(
    epoch: number,
    session: ProtoClientSession,
    error: ProtoClientSessionCloseError,
  ): void {
    if (epoch !== this.epoch || this.session !== session) return
    this.sessionClosedSubscription?.close()
    this.sessionClosedSubscription = null
    this.session = null
    if (!this.networkConnected) {
      this.publish(waitingNetworkMachineConnectionSnapshot(
        this.machineId,
        this.snapshot.forceRelay,
        this.reconnectAttempt,
      ))
      return
    }
    const failure = connectionFailure(error)
    this.publish(shouldRecoverWithoutFailure(failure) && this.keepAliveRequested
      ? reconnectingMachineConnectionSnapshot(this.machineId, this.snapshot.forceRelay, this.reconnectAttempt)
      : failedMachineConnectionSnapshot(
        this.machineId,
        failure,
        this.snapshot.forceRelay,
        this.snapshot.relayInUse,
        this.reconnectAttempt,
      ))
    this.scheduleReconnect(epoch, this.snapshot.forceRelay, failure)
  }

  private scheduleReconnect(epoch: number, forceRelay: boolean, failure: Error): void {
    if (
      epoch !== this.epoch ||
      !this.keepAliveRequested ||
      !this.networkConnected ||
      this.reconnectTimer !== null ||
      this.pending !== null ||
      this.session !== null ||
      !shouldRecoverWithoutFailure(failure) ||
      this.goOwnsMaintenance()
    ) return
    const cap = NATIVE_RECONNECT_BACKOFF_CAPS_MS[Math.min(this.reconnectBackoffIndex, NATIVE_RECONNECT_BACKOFF_CAPS_MS.length - 1)] ?? 0
    const delay = cap === 0 ? 0 : equalJitterDelay(cap, this.random())
    this.reconnectBackoffIndex += 1
    this.reconnectTimer = globalThis.setTimeout(() => {
      this.reconnectTimer = null
      if (epoch !== this.epoch || !this.keepAliveRequested || !this.networkConnected) return
      void (this.waitForForeground?.() ?? Promise.resolve()).then(() => this.acquire({ forceRelay })).then(
        (lease) => lease.close(),
        () => undefined,
      ).catch(() => undefined)
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer === null) return
    globalThis.clearTimeout(this.reconnectTimer)
    this.reconnectTimer = null
  }

  private setKeepAliveRequested(active: boolean): Promise<void> | null {
    if (!this.connector.setActive) {
      this.keepAliveRequested = active
      this.lastKeepAliveUpdate = Promise.resolve()
      return null
    }
    if (this.keepAliveRequested === active) {
      this.lastKeepAliveUpdate = this.keepAliveQueue
      return this.lastKeepAliveUpdate
    }
    this.keepAliveRequested = active
    const operation = this.keepAliveQueue.then(async () => {
      await this.connector.setActive!(this.machineId, active)
    })
    const update = operation.catch((failure: unknown) => {
      if (this.keepAliveRequested === active) this.keepAliveRequested = !active
      throw failure
    })
    this.keepAliveQueue = operation.catch(() => undefined)
    this.lastKeepAliveUpdate = update
    return update
  }

  private goOwnsMaintenance(): boolean {
    return this.connector.isGoManaged?.(this.machineId) === true
  }

  private async disconnectWhenUnused(revision: number): Promise<void> {
    if (revision !== this.demandRevision || this.demandOwners.size !== 0) return
    if (!this.keepAliveRequested && !this.session && !this.pending) return
    try {
      await this.disconnect()
    } catch {
      // Passive demand release must not turn a completed UI/task cleanup into an error.
    }
  }
}

async function withDeadline<T>(operation: Promise<T>, timeoutMs: number, message: string): Promise<T> {
  let timeout: ReturnType<typeof globalThis.setTimeout> | undefined
  try {
    return await Promise.race([
      operation,
      new Promise<never>((_, reject) => {
        timeout = globalThis.setTimeout(() => reject(new Error(message)), timeoutMs)
      }),
    ])
  } finally {
    if (timeout !== undefined) globalThis.clearTimeout(timeout)
  }
}

function isAlreadyInvalidatedError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false
  const code = 'code' in error && typeof error.code === 'string' ? error.code : ''
  return code === 'stale_session' || code === 'not_found'
}

function mergeNativeNetworkChanges(
  active: PendingNativeNetworkChange | null,
  pending: PendingNativeNetworkChange | null,
  next: PendingNativeNetworkChange,
): PendingNativeNetworkChange {
  if (!next.connected || pending?.connected === false || (!pending && active?.connected === false)) return next
  if (next.reason === 'network_replaced' || pending?.reason === 'network_replaced') {
    return { ...next, reason: 'network_replaced' }
  }
  return next
}

function isNonRetryableFailure(error: Error): boolean {
  return (error as Error & { retryable?: boolean }).retryable === false
}

const userActionFailureCodes = new Set([
  'auth',
  'unauthenticated',
  'capability_invalid',
  'capability_expired',
  'authorization_revoked',
  'scope_invalid',
  'daemon_blocked',
  'daemon_deleted',
  'device_identity_mismatch',
  'entitlement_denied',
  'relay_not_in_plan',
  'subscription_inactive',
  'relay_region_unavailable',
  'relay_quota_exhausted',
  'relay_concurrency_exhausted',
  'resource_exhausted',
])

function shouldRecoverWithoutFailure(error: Error): boolean {
  if (isNonRetryableFailure(error)) return false
  const code = (error as Error & { code?: unknown }).code
  return typeof code !== 'string' || !userActionFailureCodes.has(code.trim().toLowerCase())
}

function shouldKeepConsumerPending(error: Error): boolean {
  if (!shouldRecoverWithoutFailure(error)) return false
  const code = (error as Error & { code?: unknown }).code
  if (typeof code !== 'string') return false
  const normalized = code.trim().toLowerCase()
  return normalized === 'unavailable' || normalized === 'stale_session' || normalized === 'temporary'
}

function shouldKeepGoConsumerPending(error: Error): boolean {
  if (!shouldRecoverWithoutFailure(error)) return false
  const code = (error as Error & { code?: unknown }).code
  if (typeof code !== 'string') return false
  const normalized = code.trim().toLowerCase()
  return normalized === 'unavailable' || normalized === 'stale_session' || normalized === 'temporary' ||
    normalized === 'cancelled' || normalized === 'canceled'
}

function waitForRetryWindow(signal?: AbortSignal): Promise<void> {
  if (signal?.aborted) return Promise.reject(abortError(signal))
  return new Promise<void>((resolve, reject) => {
    const timeout = globalThis.setTimeout(() => {
      signal?.removeEventListener('abort', abort)
      resolve()
    }, 250)
    const abort = () => {
      globalThis.clearTimeout(timeout)
      reject(signal ? abortError(signal) : new DOMException('Aborted', 'AbortError'))
    }
    signal?.addEventListener('abort', abort, { once: true })
  })
}

function idleMachineConnectionSnapshot(machineId: string, reconnectAttempt = 0): MachineConnectionSnapshot {
  return {
    machineId,
    phase: 'idle',
    statusText: 'Ready',
    connectionInfo: null,
    forceRelay: false,
    relayInUse: false,
    reconnectAttempt,
    error: null,
  }
}

function waitingNetworkMachineConnectionSnapshot(
  machineId: string,
  forceRelay = false,
  reconnectAttempt = 0,
): MachineConnectionSnapshot {
  return {
    machineId,
    phase: 'waiting_network',
    statusText: 'Waiting for network...',
    connectionInfo: null,
    forceRelay,
    relayInUse: false,
    reconnectAttempt,
    error: null,
  }
}

function networkTransitionMachineConnectionSnapshot(
  machineId: string,
  forceRelay: boolean,
  reconnectAttempt: number,
  phase: 'verifying' | 'reconnecting',
  statusText: string,
): MachineConnectionSnapshot {
  return {
    machineId,
    phase,
    statusText,
    connectionInfo: null,
    forceRelay,
    relayInUse: false,
    reconnectAttempt,
    error: null,
  }
}

function reconnectingMachineConnectionSnapshot(
  machineId: string,
  forceRelay: boolean,
  reconnectAttempt: number,
): MachineConnectionSnapshot {
  return networkTransitionMachineConnectionSnapshot(
    machineId,
    forceRelay,
    reconnectAttempt,
    'reconnecting',
    'Reconnecting...',
  )
}

function failedMachineConnectionSnapshot(
  machineId: string,
  failure: Error,
  forceRelay: boolean,
  relayInUse: boolean,
  reconnectAttempt: number,
): MachineConnectionSnapshot {
  return {
    machineId,
    phase: 'failed',
    statusText: failure.message,
    connectionInfo: null,
    forceRelay,
    relayInUse,
    reconnectAttempt,
    error: failure,
  }
}

function networkUnavailableError(): Error {
  return Object.assign(new Error('Your phone is offline.'), {
    code: 'network_offline',
    retryable: true,
  })
}

function probeTimeoutMs(session: ProtoClientSession): number {
  const nanos = session.connection?.roundTripNanos ?? 0n
  if (nanos <= 0n) return NATIVE_SESSION_DEFAULT_PROBE_TIMEOUT_MS
  const rttMs = Number(nanos / 1_000_000n)
  return Math.min(5_000, Math.max(1_500, rttMs * 3))
}

function shouldReplaceAfterVerification(
  failure: unknown,
  session: ProtoClientSession,
  signal: AbortSignal,
): boolean {
  if (!session.isAlive() || signal.aborted) return true
  const code = (failure as { code?: string } | null)?.code
  if (code) return code === 'stale_session' || code === 'unavailable'
  return true
}

function equalJitterDelay(cap: number, sample: number): number {
  const normalized = Number.isFinite(sample) ? Math.min(1, Math.max(0, sample)) : 0.5
  return Math.floor(cap / 2 + normalized * cap / 2)
}

function connectionStateSnapshot(
  machineId: string,
  snapshot: RtcConnectionStateSnapshot,
  forceRelay: boolean,
  reconnectAttempt: number,
): MachineConnectionSnapshot {
  return {
    machineId,
    phase: snapshot.phase,
    statusText: snapshot.statusText,
    connectionInfo: null,
    forceRelay,
    relayInUse: snapshot.relayInUse,
    reconnectAttempt,
    error: snapshot.phase === 'failed' ? connectionFailure(snapshot.error ?? snapshot.statusText) : null,
  }
}

function connectedMachineConnectionSnapshot(
  machineId: string,
  session: ProtoClientSession,
  forceRelay: boolean,
  reconnectAttempt: number,
): MachineConnectionSnapshot {
  const connection = session.connection
  const observedPath = connection?.observedPath === ConnectionObservedPath.DIRECT
    ? 'direct'
    : connection?.observedPath === ConnectionObservedPath.SINGLE_RELAY
      ? 'single_relay'
      : undefined
  const routeKind: ConnectionInfo['routeKind'] = connection?.routeKind === ConnectionRouteKind.DIRECT
    ? 'direct'
    : connection?.routeKind === ConnectionRouteKind.SSH
      ? 'ssh'
      : connection?.routeKind === ConnectionRouteKind.CLOUD
        ? 'cloud'
        : connection?.routeKind === ConnectionRouteKind.LOCAL
          ? 'local'
          : undefined
  const relayInUse = observedPath === 'single_relay'
  const connectionInfo: ConnectionInfo = {
    path: routeKind === 'cloud' ? 'hub' : 'local',
    ...(connection?.routeId || session.stamp.routeId ? { routeId: connection?.routeId || session.stamp.routeId } : {}),
    ...(routeKind ? { routeKind } : {}),
    ...(observedPath ? { observedPath } : {}),
    connectionId: `${session.stamp.endpointId}:${session.stamp.generation}`,
    machineId,
    relayInUse,
    type: relayInUse ? 'relay' : observedPath === 'direct' ? 'p2p' : 'unknown',
    generation: session.stamp.generation,
  }
  return {
    machineId,
    phase: 'connected',
    statusText: 'Connected',
    connectionInfo,
    forceRelay,
    relayInUse,
    reconnectAttempt,
    error: null,
  }
}

class NativeSessionLease implements ProtoClientSession {
  private alive = true
  private readonly subscriptions = new Set<ProtoClientSubscription>()
  private readonly cleanupOperations = new Set<Promise<unknown>>()
  private demandReleased = false

  constructor(
    private readonly session: ProtoClientSession,
    private readonly waitForForeground?: (signal?: AbortSignal) => Promise<void>,
    private readonly releaseDemand?: () => void,
  ) {}

  get stamp(): EndpointSessionStamp { return this.session.stamp }
  get connection() { return this.session.connection }
  get resourcePoolOwner(): ProtoClientSession { return this.session.resourcePoolOwner ?? this.session }

  async execute(command: CommandEnvelope, options?: { signal?: AbortSignal }): Promise<ResultEnvelope> {
    // Cleanup belongs to the resource-owning physical session. Capture it before
    // the foreground wait so an immediately closed UI lease cannot suppress the release.
    const cleanupSession = releasesSessionResource(command) ? this.requireAlive() : null
    const operation = (async () => {
      if (this.waitForForeground) await this.waitForForeground(options?.signal)
      return await (cleanupSession ?? this.requireAlive()).execute(command, options)
    })()
    if (cleanupSession) {
      this.cleanupOperations.add(operation)
      void operation.then(
        () => this.finishCleanupOperation(operation),
        () => this.finishCleanupOperation(operation),
      )
    }
    return await operation
  }

  subscribeEvents(handler: (event: EventEnvelope) => void): ProtoClientSubscription {
    const subscription = this.requireAlive().subscribeEvents(handler)
    const leaseSubscription: ProtoClientSubscription = {
      close: () => {
        subscription.close()
        this.subscriptions.delete(leaseSubscription)
      },
    }
    this.subscriptions.add(leaseSubscription)
    return leaseSubscription
  }

  subscribeClosed(handler: (error: ProtoClientSessionCloseError) => void): ProtoClientSubscription {
    const subscription = this.requireAlive().subscribeClosed(handler)
    const leaseSubscription: ProtoClientSubscription = {
      close: () => {
        subscription.close()
        this.subscriptions.delete(leaseSubscription)
      },
    }
    this.subscriptions.add(leaseSubscription)
    return leaseSubscription
  }

  async openResourceStream(resource: ResourceHandle, options?: { initialUploadOffset?: bigint; signal?: AbortSignal }): Promise<ProtoResourceStream> {
    if (this.waitForForeground) await this.waitForForeground(options?.signal)
    return await this.requireAlive().openResourceStream(resource, options)
  }

  isAlive(): boolean { return this.alive && this.session.isAlive() }

  async getConnectionSnapshot() {
    if (this.waitForForeground) await this.waitForForeground()
    const session = this.requireAlive()
    return await (session.getConnectionSnapshot?.() ?? Promise.resolve(session.connection))
  }

  async close(): Promise<void> {
    if (!this.alive) return
    this.alive = false
    for (const subscription of [...this.subscriptions]) subscription.close()
    this.subscriptions.clear()
    if (this.cleanupOperations.size === 0) {
      this.releaseDemandOnce()
    }
  }

  private finishCleanupOperation(operation: Promise<unknown>): void {
    this.cleanupOperations.delete(operation)
    if (!this.alive && this.cleanupOperations.size === 0) this.releaseDemandOnce()
  }

  private releaseDemandOnce(): void {
    if (this.demandReleased) return
    this.demandReleased = true
    this.releaseDemand?.()
  }

  private requireAlive(): ProtoClientSession {
    if (!this.isAlive()) throw new Error('Proto session lease is closed')
    return this.session
  }
}

function releasesSessionResource(command: CommandEnvelope): boolean {
  switch (command.command.case) {
    case 'releaseResource':
    case 'terminalDetach':
    case 'historyRelease':
    case 'fileTransferCancel':
      return true
    default:
      return false
  }
}

async function awaitSessionLease(pending: Promise<ProtoClientSession>, signal: AbortSignal | undefined): Promise<ProtoClientSession> {
  if (!signal) return await pending
  if (signal.aborted) throw abortError(signal)
  return await new Promise<ProtoClientSession>((resolve, reject) => {
    let settled = false
    const abort = () => {
      if (settled) return
      settled = true
      signal.removeEventListener('abort', abort)
      reject(abortError(signal))
    }
    signal.addEventListener('abort', abort, { once: true })
    void pending.then(
      (session) => {
        if (settled) return
        settled = true
        signal.removeEventListener('abort', abort)
        resolve(session)
      },
      (error) => {
        if (settled) return
        settled = true
        signal.removeEventListener('abort', abort)
        reject(error)
      },
    )
  })
}

function abortError(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new DOMException('Aborted', 'AbortError')
}

function connectionFailure(error: unknown): ProtoClientSessionCloseError {
  if (error instanceof Error) return error as ProtoClientSessionCloseError
  return new Error(typeof error === 'string' && error.trim() ? error : 'Connection failed')
}
