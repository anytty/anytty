import { create } from '@bufbuild/protobuf'
import type { ConnectionInfo, ConnectionPolicy, ConnectionPolicyState, MachineConnectionSnapshot, ProtoClientSession, ProtoClientSubscription, ProtoResourceStream, RtcConnectOptions, RtcConnectionStateSnapshot } from '@anytty/ui'
import { type CommandEnvelope, type EventEnvelope, type ResultEnvelope } from '../../ui/src/generated/apipb/application_pb'
import type { EndpointSessionStamp, ResourceHandle } from '../../ui/src/generated/apipb/common_pb'
import { ConnectionObservedPath, ConnectionRouteKind } from '../../ui/src/generated/bindingpb/client_binding_pb'

type ProtoClientSessionCloseHandler = Parameters<ProtoClientSession['subscribeClosed']>[0]
type ProtoClientSessionCloseError = Parameters<ProtoClientSessionCloseHandler>[0]

// This bounds only the renderer-to-Go binding operation. Go owns endpoint dialing and readiness.
const NATIVE_SESSION_READY_TIMEOUT_MS = 45_000
const NATIVE_SESSION_DISCONNECT_TIMEOUT_MS = 8_000
const NATIVE_SESSION_DEMAND_TIMEOUT_MS = 5_000
const NATIVE_SESSION_RECOVERY_HINT_TIMEOUT_MS = 1_500
const NATIVE_SESSION_CLEANUP_GRACE_MS = 5_000
const NATIVE_DEMAND_RETRY_CAPS_MS = [250, 500, 1_000, 2_000, 5_000, 15_000] as const
const NATIVE_RECONNECT_BACKOFF_CAPS_MS = [0, 500, 2_000, 4_000, 8_000, 15_000] as const
let nativeSessionDiagnosticId = 0

/** Narrow renderer entry point into the process-owned Go Endpoint Supervisor. */
export type NativeSessionConnector = {
  connect(input: { machineId: string }, options?: RtcConnectOptions): Promise<ProtoClientSession>
  getConnectionPolicy?(signal?: AbortSignal): Promise<ConnectionPolicyState>
  applyConnectionPolicy?(policy: ConnectionPolicy, signal?: AbortSignal): Promise<void>
  disconnect?(machineId: string): Promise<void>
  release?(machineId: string): Promise<void>
  setActive?(machineId: string, active: boolean): Promise<void>
  createResumeIntent?(): object
  resumeDemand?(intent: object): Promise<void>
  requestRecovery?(): Promise<void>
}

export type NativeSessionManagerOptions = {
  initiallyConnected?: boolean
  random?: () => number
  waitForForeground?: (signal?: AbortSignal) => Promise<void>
  writeDiagnostic?: (value: string) => void
  onUserResumeAccepted?: (intent: object) => void
}

export type NativeNetworkChangeReason = 'available' | 'offline' | 'network_replaced' | 'path_changed'

/**
 * Owns renderer leases for one endpoint. Go owns physical transport, route selection, winner
 * replacement, and transport generations; foreground preserves a live renderer lease and follows
 * only authoritative network, supervisor, or session-close transitions.
 */
export class NativeSessionManager {
  private readonly diagnosticId = ++nativeSessionDiagnosticId
  private session: ProtoClientSession | null = null
  private pending: Promise<ProtoClientSession> | null = null
  private pendingController: AbortController | null = null
  private disconnectBarrier: Promise<void> | null = null
  private sessionClosedSubscription: ProtoClientSubscription | null = null
  private readonly issuedLeases = new Set<NativeSessionLease>()
  private epoch = 0
  private reconnectAttempt = 0
  private reconnectBackoffIndex = 0
  private reconnectTimer: ReturnType<typeof globalThis.setTimeout> | null = null
  private readonly demandOwners = new Set<symbol>()
  private readonly consumerDemandOwners = new Set<symbol>()
  private demandRevision = 0
  private consumerGeneration = 0
  private userStopped = false
  private connectionIntentGeneration = 0
  private userResumeIntent: object | null = null
  private latestResumeIntent: object | null = null
  private userResumeBarrier: { intent: object; operation: Promise<void> } | null = null
  private keepAliveRequested = false
  private keepAliveSynchronized = true
  private keepAliveQueue: Promise<void> = Promise.resolve()
  private keepAlivePendingState: boolean | null = null
  private keepAlivePending: Promise<void> | null = null
  private lastKeepAliveUpdate: Promise<void> = Promise.resolve()
  private demandRetryAttempt = 0
  private demandRetryBarrier: { operation: Promise<void>; release: () => void } | null = null
  private networkConnected: boolean
  private snapshot: MachineConnectionSnapshot
  private readonly stateListeners = new Set<() => void>()
  private readonly random: () => number
  private readonly waitForForeground: ((signal?: AbortSignal) => Promise<void>) | undefined
  private readonly writeDiagnostic: ((value: string) => void) | undefined
  private readonly onUserResumeAccepted: ((intent: object) => void) | undefined

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
    this.random = options.random ?? Math.random
    this.waitForForeground = options.waitForForeground
    this.writeDiagnostic = options.writeDiagnostic
    this.onUserResumeAccepted = options.onUserResumeAccepted
    this.snapshot = this.networkConnected
      ? idleMachineConnectionSnapshot(machineId)
      : waitingNetworkMachineConnectionSnapshot(machineId)
    this.diagnostic('created', { connected: this.networkConnected })
  }

  machineID(): string { return this.machineId }

  /** Retains endpoint demand independently from the current renderer binding generation. */
  retainConnectionDemand(resumeIntent?: object | null): () => void {
    if (resumeIntent != null) this.beginUserConnectionIntent(resumeIntent)
    return this.retainDemandOwner(false)
  }

  /** Marks an explicit endpoint action without retaining a passive workspace owner. */
  beginUserConnectionIntent(intent: object = this.connector.createResumeIntent?.() ?? {}): object {
    if (this.latestResumeIntent === intent) return intent
    this.connectionIntentGeneration += 1
    this.userResumeIntent = intent
    this.latestResumeIntent = intent
    this.resetDemandRetryBackoff()
    return intent
  }

  latestUserResumeIntent(): object | null { return this.latestResumeIntent }

  private retainDemandOwner(consumer: boolean): () => void {
    const owner = Symbol(this.machineId)
    this.demandOwners.add(owner)
    if (consumer) this.consumerDemandOwners.add(owner)
    this.demandRevision += 1
    let released = false
    return () => {
      if (released) return
      released = true
      if (!this.demandOwners.delete(owner)) return
      this.consumerDemandOwners.delete(owner)
      const revision = ++this.demandRevision
      if (this.demandOwners.size === 0) queueMicrotask(() => { void this.releaseWhenUnused(revision) })
    }
  }

  hasConnectionDemand(): boolean { return this.demandOwners.size > 0 }

  get(options?: RtcConnectOptions): Promise<ProtoClientSession> { return this.acquireForConsumer(options) }

  async probe(): Promise<ConnectionInfo | null> {
    const lease = await this.acquireForConsumer()
    try {
      return this.connectionState.getSnapshot().connectionInfo
    } finally {
      await lease.close()
    }
  }

  lease(options?: RtcConnectOptions): Promise<ProtoClientSession> { return this.acquireForConsumer(options) }

  /** Drops renderer resources and native demand without invalidating a Go transport winner. */
  async reset(): Promise<void> {
    this.consumerGeneration += 1
    this.resetDemandRetryBackoff()
    this.releaseConsumerDemandOwners()
    this.fenceRendererSession(false)
    await this.lastKeepAliveUpdate
  }

  /** Fences the old renderer attachment while preserving process-owned endpoint demand. */
  async resetBindingGeneration(): Promise<void> {
    const preserveDemand = this.shouldFollowEndpoint()
    this.fenceRendererSession(
      preserveDemand,
      preserveDemand
        ? reconnectingMachineConnectionSnapshot(this.machineId, this.snapshot.forceRelay, this.reconnectAttempt)
        : this.inactiveConnectionSnapshot(),
    )
    if (preserveDemand) {
      const activation = this.setKeepAliveRequested(true)
      if (activation) await activation
    } else {
      await this.lastKeepAliveUpdate
    }
  }

  /** Explicit user disconnect is the only manager path allowed to stop the Go endpoint winner. */
  async disconnect(): Promise<void> {
    const disconnectGeneration = ++this.connectionIntentGeneration
    this.consumerGeneration += 1
    this.resetDemandRetryBackoff()
    this.releaseConsumerDemandOwners()
    this.userStopped = true
    this.userResumeIntent = null
    this.latestResumeIntent = null
    this.userResumeBarrier = null
    const failure = userStoppedError()
    this.fenceRendererSession(
      false,
      failedMachineConnectionSnapshot(
        this.machineId,
        failure,
        this.snapshot.forceRelay,
        this.snapshot.relayInUse,
        this.reconnectAttempt,
      ),
      failure,
    )
    const keepAliveUpdate = this.lastKeepAliveUpdate
    const previous = this.disconnectBarrier ?? Promise.resolve()
    const barrier = previous.catch(() => undefined).then(async () => {
      const demand = await keepAliveUpdate.then(
        () => ({ failure: null }),
        (failure: unknown) => ({ failure }),
      )
      if (disconnectGeneration !== this.connectionIntentGeneration) {
        this.diagnostic('disconnect_superseded', {})
        return
      }
      if (demand.failure !== null) {
        this.diagnostic('disconnect_demand_deferred', {
          failure: nativeSessionFailureCode(demand.failure),
        })
        if (!this.connector.disconnect) throw demand.failure
      }
      if (this.connector.disconnect) {
        await withDeadline(
          this.connector.disconnect(this.machineId),
          NATIVE_SESSION_DISCONNECT_TIMEOUT_MS,
          'endpoint disconnect timed out',
        )
      }
    })
    this.disconnectBarrier = barrier
    void barrier.finally(() => {
      if (this.disconnectBarrier === barrier) this.disconnectBarrier = null
    }).catch(() => undefined)
    await barrier
  }

  /** Applies a process-level Stop latch to this renderer manager. */
  adoptUserStop(): Promise<void> { return this.disconnect() }

  /** Requests endpoint repair as an acceleration hint, then follows the supervisor independently. */
  async resetClientOnly(options?: { forceRelay?: boolean }): Promise<void> {
    const generation = this.consumerGeneration
    this.beginUserConnectionIntent()
    const intentGeneration = this.connectionIntentGeneration
    const forceRelay = options?.forceRelay === true || this.snapshot.forceRelay
    this.fenceRendererSession(
      true,
      this.networkConnected
        ? reconnectingMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt)
        : waitingNetworkMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt),
    )
    await this.synchronizeUserIntent(generation, intentGeneration, forceRelay)
    this.requireUserIntentCurrent(generation, intentGeneration)
    try {
      const recovery = this.connector.requestRecovery?.()
      if (recovery) {
        await withDeadline(
          recovery,
          NATIVE_SESSION_RECOVERY_HINT_TIMEOUT_MS,
          'Native endpoint recovery hint timed out',
        )
      }
      this.requireUserIntentCurrent(generation, intentGeneration)
    } catch (failure) {
      this.requireUserIntentCurrent(generation, intentGeneration)
      const error = connectionFailure(failure)
      if (!shouldRecoverWithoutFailure(error)) throw failure
      this.diagnostic('repair_deferred', { failure: nativeSessionFailureCode(error) })
    }

    while (true) {
      this.requireUserIntentCurrent(generation, intentGeneration)
      if (!this.shouldFollowEndpoint() || !this.networkConnected) return
      const epoch = this.epoch
      try {
        await this.bindFollower(epoch, { forceRelay })
        this.requireUserIntentCurrent(generation, intentGeneration)
        if (epoch !== this.epoch) continue
        return
      } catch (failure) {
        this.requireUserIntentCurrent(generation, intentGeneration)
        if (epoch !== this.epoch) continue
        const error = connectionFailure(failure)
        if (!shouldRecoverWithoutFailure(error)) throw failure
        // startBindingSession also schedules in its finally path; this makes the
        // one-click recovery contract explicit even if that ordering changes.
        this.scheduleReconnect(epoch, forceRelay, error)
        return
      }
    }
  }

  /** Every host-network revision fences the old renderer lease; Go decides transport recovery. */
  async networkChanged(
    connected = true,
    reason: NativeNetworkChangeReason = connected ? 'path_changed' : 'offline',
  ): Promise<void> {
    this.diagnostic('network_changed', {
      connected,
      reason,
      has_session: this.session !== null,
      pending: this.pending !== null,
      keep_alive: this.keepAliveRequested,
    })
    this.networkConnected = connected
    const follow = this.shouldFollowEndpoint()
    const forceRelay = this.snapshot.forceRelay
    const epoch = this.fenceRendererSession(
      true,
      connected
        ? follow
          ? reconnectingMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt)
          : this.inactiveConnectionSnapshot(forceRelay)
        : waitingNetworkMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt),
    )
    if (connected && follow) await this.bindFollower(epoch, { forceRelay })
  }

  /** Cold-start sampling does not manufacture a renderer binding generation. */
  async initializeNetworkState(connected: boolean): Promise<void> {
    if (this.session || this.pending || this.shouldFollowEndpoint()) {
      await this.networkChanged(connected)
      return
    }
    this.networkConnected = connected
    this.publish(connected
      ? this.inactiveConnectionSnapshot()
      : waitingNetworkMachineConnectionSnapshot(this.machineId, this.snapshot.forceRelay, this.reconnectAttempt))
  }

  /** Foreground entry preserves a healthy renderer binding and repairs only missing or stale state. */
  async foregroundResume(signal?: AbortSignal): Promise<void> {
    const startedAt = globalThis.performance.now()
    this.diagnostic('foreground_resume_start', {
      phase: this.snapshot.phase,
      connected: this.networkConnected,
      keep_alive: this.keepAliveRequested,
      has_session: this.session !== null,
      pending: this.pending !== null,
    })
    while (true) {
      if (signal?.aborted) throw abortError(signal)
      if (!this.shouldFollowEndpoint()) {
        this.diagnostic('foreground_resume_skipped', { reason: 'no_demand' })
        return
      }
      const forceRelay = this.snapshot.forceRelay
      if (!this.networkConnected) {
        const waiting = waitingNetworkMachineConnectionSnapshot(
          this.machineId,
          forceRelay,
          this.reconnectAttempt,
        )
        if (this.session !== null || this.pending !== null) {
          this.fenceRendererSession(true, waiting)
        } else {
          this.publish(waiting)
        }
        this.diagnostic('foreground_resume_skipped', { reason: 'offline' })
        return
      }
      if (this.session?.isAlive()) {
        this.diagnostic('foreground_resume_preserved', {
          duration_ms: Math.round(globalThis.performance.now() - startedAt),
          phase: this.snapshot.phase,
        })
        return
      }
      const epoch = this.epoch
      try {
        await this.bindFollower(epoch, { forceRelay, signal })
      } catch (failure) {
        if (signal?.aborted) throw abortError(signal)
        if (epoch !== this.epoch) continue
        this.diagnostic('foreground_resume_failed', {
          duration_ms: Math.round(globalThis.performance.now() - startedAt),
          phase: this.snapshot.phase,
          failure: nativeSessionFailureCode(failure),
        })
        throw failure
      }
      if (this.session?.isAlive()) {
        this.diagnostic('foreground_resume_done', {
          duration_ms: Math.round(globalThis.performance.now() - startedAt),
          phase: this.snapshot.phase,
        })
        return
      }
      if (epoch === this.epoch) {
        const failure = bindingUnavailableError()
        this.diagnostic('foreground_resume_failed', {
          duration_ms: Math.round(globalThis.performance.now() - startedAt),
          phase: this.snapshot.phase,
          failure: nativeSessionFailureCode(failure),
        })
        throw failure
      }
    }
  }

  private fenceRendererSession(
    preserveKeepAlive: boolean,
    nextSnapshot?: MachineConnectionSnapshot,
    fenceFailure: Error = generationChangedError(),
  ): number {
    const epoch = ++this.epoch
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
    pendingController?.abort(fenceFailure)
    if (session) this.fenceIssuedLeases(session, fenceFailure)
    void session?.close().catch(() => undefined)
    if (pending) void pending.then((late) => late.close(), () => undefined).catch(() => undefined)
    return epoch
  }

  /** Business consumers survive process-demand reconciliation and transient binding turnover. */
  private async acquireForConsumer(options?: RtcConnectOptions): Promise<ProtoClientSession> {
    const consumerGeneration = this.consumerGeneration
    if (this.userStopped && this.userResumeIntent === null) throw userStoppedError()
    const releaseDemand = this.retainDemandOwner(true)
    try {
      while (true) {
        if (this.waitForForeground) await this.waitForForeground(options?.signal)
        if (consumerGeneration !== this.consumerGeneration) throw generationChangedError()
        try {
          await this.resumeAfterUserStop(consumerGeneration)
          if (consumerGeneration !== this.consumerGeneration) throw generationChangedError()
          return await this.acquire(options, releaseDemand)
        } catch (failure) {
          if (options?.signal?.aborted) throw abortError(options.signal)
          if (consumerGeneration !== this.consumerGeneration) throw generationChangedError()
          const error = connectionFailure(failure)
          const resumingUserIntent = this.userResumeIntent !== null && this.demandOwners.size > 0
          if ((!this.shouldFollowEndpoint() && !resumingUserIntent) || !this.networkConnected || !shouldRetryBinding(error)) throw failure
          await this.waitForDemandRetry(options?.signal)
        }
      }
    } catch (failure) {
      releaseDemand()
      throw failure
    }
  }

  /** Rebinds internal renderer projection; demand-sync timeouts keep retrying automatically. */
  private async bindFollower(epoch: number, options?: RtcConnectOptions): Promise<void> {
    while (epoch === this.epoch && this.shouldFollowEndpoint() && this.networkConnected) {
      if (options?.signal?.aborted) throw abortError(options.signal)
      try {
        const lease = await this.acquire(options)
        await lease.close()
        return
      } catch (failure) {
        if (options?.signal?.aborted) throw abortError(options.signal)
        if (epoch !== this.epoch || !this.shouldFollowEndpoint() || !this.networkConnected) return
        const error = connectionFailure(failure)
        if (nativeSessionFailureCode(error) !== 'demand_sync_pending') throw failure
        await this.waitForDemandRetry(options?.signal)
      }
    }
  }

  private async acquire(options?: RtcConnectOptions, releaseDemand?: () => void): Promise<ProtoClientSession> {
    const acquireEpoch = this.epoch
    this.clearReconnectTimer()
    const disconnect = this.disconnectBarrier
    if (disconnect) await waitForAbortableOperation(disconnect, options?.signal)
    if (acquireEpoch !== this.epoch) throw generationChangedError()
    const activation = this.setKeepAliveRequested(true)
    if (activation) await waitForAbortableOperation(activation, options?.signal)
    if (acquireEpoch !== this.epoch) throw generationChangedError()
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
      if (this.session?.isAlive()) {
        this.publish(connectedMachineConnectionSnapshot(this.machineId, this.session, this.snapshot.forceRelay, this.reconnectAttempt))
        return this.issueLease(this.session, releaseDemand)
      }
      const staleSession = this.session
      if (staleSession) {
        this.sessionClosedSubscription?.close()
        this.sessionClosedSubscription = null
        this.session = null
        this.fenceIssuedLeases(staleSession, bindingUnavailableError())
        await staleSession.close().catch(() => undefined)
        if (acquireEpoch !== this.epoch) throw generationChangedError()
      }
      if (!this.pending) this.startBindingSession(options)
      const opened = await awaitSessionLease(this.pending!, options?.signal)
      if (!opened.isAlive()) throw bindingUnavailableError()
      return this.issueLease(opened, releaseDemand)
    } finally {
      stopForwarding()
    }
  }

  private startBindingSession(options?: RtcConnectOptions): void {
    const epoch = this.epoch
    this.reconnectAttempt += 1
    this.publish(connectingMachineConnectionSnapshot(this.machineId, options?.forceRelay === true, this.reconnectAttempt))
    const controller = new AbortController()
    this.pendingController = controller
    const timeout = globalThis.setTimeout(
      () => controller.abort(Object.assign(new Error('client session timed out'), { code: 'unavailable', retryable: true })),
      NATIVE_SESSION_READY_TIMEOUT_MS,
    )
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
        // OpenSession resolution is the renderer binding readiness boundary.
        if (projected.phase === 'connected') {
          this.publish(reconnectingMachineConnectionSnapshot(this.machineId, options?.forceRelay === true, this.reconnectAttempt))
          return
        }
        if (
          projected.phase === 'failed' && projected.error && this.shouldFollowEndpoint() &&
          this.networkConnected && shouldRecoverWithoutFailure(projected.error)
        ) {
          this.publish(reconnectingMachineConnectionSnapshot(this.machineId, options?.forceRelay === true, this.reconnectAttempt))
          return
        }
        this.publish(projected)
      },
    }
    let reconnectFailure: Error | null = null
    const pending = this.connector.connect({ machineId: this.machineId }, connectOptions).then(async (opened) => {
      if (epoch !== this.epoch) {
        await opened.close().catch(() => undefined)
        throw generationChangedError()
      }
      if (!opened.isAlive()) throw bindingUnavailableError()
      this.session = opened
      this.sessionClosedSubscription = opened.subscribeClosed((error) => this.handleSessionClosed(epoch, opened, error))
      this.reconnectBackoffIndex = 0
      this.publish(connectedMachineConnectionSnapshot(
        this.machineId,
        opened,
        options?.forceRelay === true,
        this.reconnectAttempt,
      ))
      return opened
    }).catch((error: unknown) => {
      if (epoch === this.epoch) {
        const failure = connectionFailure(error)
        reconnectFailure = failure
        this.publish(shouldRecoverWithoutFailure(failure) && this.shouldFollowEndpoint() && this.networkConnected
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
    const previous = this.snapshot
    this.snapshot = snapshot
    if (
      previous.phase !== snapshot.phase || previous.reconnectAttempt !== snapshot.reconnectAttempt ||
      nativeSessionFailureCode(previous.error) !== nativeSessionFailureCode(snapshot.error)
    ) {
      this.diagnostic('state', {
        from: previous.phase,
        to: snapshot.phase,
        attempt: snapshot.reconnectAttempt,
        epoch: this.epoch,
        failure: nativeSessionFailureCode(snapshot.error),
      })
    }
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
    this.fenceIssuedLeases(session, error ?? bindingUnavailableError())
    if (!this.networkConnected) {
      this.publish(waitingNetworkMachineConnectionSnapshot(this.machineId, this.snapshot.forceRelay, this.reconnectAttempt))
      return
    }
    const failure = connectionFailure(error)
    this.publish(shouldRecoverWithoutFailure(failure) && this.shouldFollowEndpoint()
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
      epoch !== this.epoch || !this.shouldFollowEndpoint() || !this.networkConnected ||
      this.reconnectTimer !== null || this.pending !== null || this.session !== null ||
      !shouldRecoverWithoutFailure(failure)
    ) return
    const cap = NATIVE_RECONNECT_BACKOFF_CAPS_MS[
      Math.min(this.reconnectBackoffIndex, NATIVE_RECONNECT_BACKOFF_CAPS_MS.length - 1)
    ] ?? 0
    const delay = cap === 0 ? 0 : equalJitterDelay(cap, this.random())
    this.reconnectBackoffIndex += 1
    this.diagnostic('reconnect_scheduled', {
      epoch,
      delay_ms: delay,
      backoff_index: this.reconnectBackoffIndex,
      failure: nativeSessionFailureCode(failure),
    })
    this.reconnectTimer = globalThis.setTimeout(() => {
      this.reconnectTimer = null
      if (epoch !== this.epoch || !this.shouldFollowEndpoint() || !this.networkConnected) return
      void (this.waitForForeground?.() ?? Promise.resolve())
        .then(() => this.bindFollower(epoch, { forceRelay }))
        .catch(() => undefined)
    }, delay)
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer === null) return
    globalThis.clearTimeout(this.reconnectTimer)
    this.reconnectTimer = null
  }

  private shouldFollowEndpoint(): boolean {
    return !this.userStopped && this.keepAliveRequested && this.demandOwners.size > 0
  }

  private inactiveConnectionSnapshot(forceRelay = this.snapshot.forceRelay): MachineConnectionSnapshot {
    if (!this.userStopped) return idleMachineConnectionSnapshot(this.machineId, this.reconnectAttempt)
    return failedMachineConnectionSnapshot(
      this.machineId,
      userStoppedError(),
      forceRelay,
      this.snapshot.relayInUse,
      this.reconnectAttempt,
    )
  }

  private async waitForDemandRetry(signal?: AbortSignal): Promise<void> {
    let barrier = this.demandRetryBarrier
    if (barrier === null) {
      const delay = nativeDemandRetryDelay(this.demandRetryAttempt)
      this.demandRetryAttempt += 1
      let release!: () => void
      const pending = new Promise<void>((resolve) => {
        const timeout = globalThis.setTimeout(resolve, delay)
        release = () => {
          globalThis.clearTimeout(timeout)
          resolve()
        }
      })
      const created = { operation: pending, release }
      void pending.finally(() => {
        if (this.demandRetryBarrier === created) this.demandRetryBarrier = null
      })
      this.demandRetryBarrier = created
      barrier = created
    }
    await waitForAbortableOperation(barrier.operation, signal)
  }

  private resetDemandRetryBackoff(): void {
    this.demandRetryAttempt = 0
    const barrier = this.demandRetryBarrier
    this.demandRetryBarrier = null
    barrier?.release()
  }

  private async synchronizeUserIntent(
    generation: number,
    intentGeneration: number,
    forceRelay: boolean,
  ): Promise<void> {
    while (true) {
      try {
        const disconnect = this.disconnectBarrier
        if (disconnect) await disconnect.catch(() => undefined)
        this.requireUserIntentCurrent(generation, intentGeneration)
        if (this.waitForForeground) await this.waitForForeground()
        this.requireUserIntentCurrent(generation, intentGeneration)
        await this.resumeAfterUserStop(generation)
        this.requireUserIntentCurrent(generation, intentGeneration)
        const activation = this.setKeepAliveRequested(true)
        if (activation) await activation
        this.requireUserIntentCurrent(generation, intentGeneration)
        return
      } catch (failure) {
        const error = connectionFailure(failure)
        if (nativeSessionFailureCode(error).trim().toLowerCase() === 'user_stopped') throw error
        this.requireUserIntentCurrent(generation, intentGeneration)
        if (!shouldRetryBinding(error)) throw failure
        this.publish(this.networkConnected
          ? reconnectingMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt)
          : waitingNetworkMachineConnectionSnapshot(this.machineId, forceRelay, this.reconnectAttempt))
        await this.waitForDemandRetry()
      }
    }
  }

  private requireUserIntentCurrent(generation: number, intentGeneration: number): void {
    if (
      generation !== this.consumerGeneration || intentGeneration !== this.connectionIntentGeneration ||
      this.demandOwners.size === 0 || (this.userStopped && this.userResumeIntent === null)
    ) throw generationChangedError()
  }

  private async resumeAfterUserStop(expectedGeneration = this.consumerGeneration): Promise<void> {
    const intent = this.userResumeIntent
    if (intent === null) {
      if (this.userStopped) throw userStoppedError()
      return
    }
    const existing = this.userResumeBarrier
    if (existing?.intent === intent) {
      await existing.operation
      if (expectedGeneration !== this.consumerGeneration) throw generationChangedError()
      return
    }
    const generation = expectedGeneration
    const intentGeneration = this.connectionIntentGeneration
    let operation: Promise<void>
    operation = (this.connector.resumeDemand?.(intent) ?? Promise.resolve())
      .catch((failure: unknown) => {
        if (
          generation !== this.consumerGeneration ||
          intentGeneration !== this.connectionIntentGeneration ||
          this.userResumeIntent !== intent
        ) throw generationChangedError()
        throw this.adoptNativeDemandFailure(failure)
      })
      .then(() => {
        if (generation !== this.consumerGeneration || this.userResumeIntent !== intent) {
          throw generationChangedError()
        }
        this.userStopped = false
        this.userResumeIntent = null
        this.resetDemandRetryBackoff()
        if (this.keepAliveRequested) this.keepAliveSynchronized = false
        try {
          this.onUserResumeAccepted?.(intent)
        } catch (failure) {
          this.diagnostic('resume_accepted_callback_failed', {
            failure: nativeSessionFailureCode(failure),
          })
        }
      }).finally(() => {
        if (this.userResumeBarrier?.operation === operation) this.userResumeBarrier = null
      })
    this.userResumeBarrier = { intent, operation }
    await operation
  }

  private issueLease(session: ProtoClientSession, releaseDemand?: () => void): NativeSessionLease {
    const lease = new NativeSessionLease(
      session,
      this.waitForForeground,
      releaseDemand,
      (released) => this.issuedLeases.delete(released),
    )
    this.issuedLeases.add(lease)
    if (!lease.isAlive()) this.issuedLeases.delete(lease)
    return lease
  }

  private fenceIssuedLeases(session: ProtoClientSession, failure: ProtoClientSessionCloseError): void {
    for (const lease of [...this.issuedLeases]) {
      if (lease.belongsTo(session)) lease.fence(failure)
    }
  }

  private setKeepAliveRequested(active: boolean): Promise<void> | null {
    if (!this.connector.setActive) {
      this.keepAliveRequested = active
      this.keepAliveSynchronized = true
      this.keepAlivePendingState = null
      this.keepAlivePending = null
      this.lastKeepAliveUpdate = Promise.resolve()
      return null
    }
    if (this.keepAliveRequested === active && this.keepAliveSynchronized) {
      this.lastKeepAliveUpdate = this.keepAliveQueue
      return this.lastKeepAliveUpdate
    }
    if (
      this.keepAliveRequested === active &&
      this.keepAlivePendingState === active &&
      this.keepAlivePending !== null
    ) return this.keepAlivePending
    this.keepAliveRequested = active
    this.keepAliveSynchronized = false
    const intentGeneration = this.connectionIntentGeneration
    const operation = this.keepAliveQueue.then(async () => {
      await withDeadline(
        this.connector.setActive!(this.machineId, active),
        NATIVE_SESSION_DEMAND_TIMEOUT_MS,
        'Native session demand reconciliation timed out',
      )
    })
    let update: Promise<void>
    update = operation.then(
      () => {
        if (
          this.keepAliveRequested !== active ||
          this.connectionIntentGeneration !== intentGeneration
        ) throw generationChangedError()
        this.keepAliveSynchronized = true
        this.resetDemandRetryBackoff()
      },
      (failure: unknown) => {
        if (this.keepAliveRequested === active) this.keepAliveSynchronized = false
        if (
          this.keepAliveRequested !== active ||
          this.connectionIntentGeneration !== intentGeneration
        ) throw generationChangedError()
        throw this.adoptNativeDemandFailure(failure)
      },
    ).finally(() => {
      if (this.keepAlivePending === update) {
        this.keepAlivePending = null
        this.keepAlivePendingState = null
      }
    })
    this.keepAliveQueue = operation.catch(() => undefined)
    this.keepAlivePendingState = active
    this.keepAlivePending = update
    this.lastKeepAliveUpdate = update
    return update
  }

  private adoptNativeDemandFailure(failure: unknown): Error {
    const error = nativeDemandReconciliationError(failure)
    if (nativeSessionFailureCode(error).trim().toLowerCase() !== 'user_stopped') return error

    this.userStopped = true
    this.userResumeIntent = null
    this.latestResumeIntent = null
    this.keepAliveSynchronized = false
    this.resetDemandRetryBackoff()
    const stopped = userStoppedError()
    this.publish(failedMachineConnectionSnapshot(
      this.machineId,
      stopped,
      this.snapshot.forceRelay,
      this.snapshot.relayInUse,
      this.reconnectAttempt,
    ))
    return stopped
  }

  private diagnostic(event: string, fields: Record<string, string | number | boolean>): void {
    const details = Object.entries(fields)
      .map(([key, value]) => `${nativeSessionDiagnosticToken(key)}=${nativeSessionDiagnosticToken(String(value))}`)
      .join(' ')
    const value = `event=session_${nativeSessionDiagnosticToken(event)} session=${this.diagnosticId}${details ? ` ${details}` : ''}`
    console.info(`[anytty:diagnostic] ${value}`)
    this.writeDiagnostic?.(value)
  }

  private async releaseWhenUnused(revision: number): Promise<void> {
    if (revision !== this.demandRevision || this.demandOwners.size !== 0) return
    if (!this.keepAliveRequested && !this.session && !this.pending) return
    this.fenceRendererSession(false)
    try {
      await this.lastKeepAliveUpdate
    } catch {
      // Passive UI cleanup must not surface native demand synchronization failures.
    }
  }

  private releaseConsumerDemandOwners(): void {
    if (this.consumerDemandOwners.size === 0) return
    for (const owner of this.consumerDemandOwners) this.demandOwners.delete(owner)
    this.consumerDemandOwners.clear()
    this.demandRevision += 1
  }
}

function nativeSessionFailureCode(failure: unknown): string {
  if (failure && typeof failure === 'object') {
    const code = 'code' in failure && typeof failure.code === 'string' ? failure.code.trim() : ''
    if (code) return code
    if ('name' in failure && typeof failure.name === 'string' && failure.name.trim()) return failure.name
  }
  return failure == null ? 'none' : typeof failure
}

function nativeDemandReconciliationError(failure: unknown): Error {
  const cause = failure instanceof Error ? failure : new Error(String(failure))
  if ((cause as Error & { retryable?: boolean }).retryable === false) return cause
  return Object.assign(new Error(cause.message || 'Native session demand reconciliation failed', { cause }), {
    code: 'demand_sync_pending',
    retryable: true,
  })
}

function nativeSessionDiagnosticToken(value: string): string {
  return value.trim().replace(/[^A-Za-z0-9_.-]+/g, '_').slice(0, 80) || 'none'
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
  const code = (error as Error & { code?: unknown }).code
  const normalized = typeof code === 'string' ? code.trim().toLowerCase() : ''
  // stale_session is a generation fence, not a terminal user/action failure. Go
  // intentionally reports the old stamp as non-retryable to prevent replaying the
  // same operation; the follower must still acquire the supervisor's newer winner.
  if (normalized === 'stale_session') return true
  if (isNonRetryableFailure(error)) return false
  return normalized === '' || !userActionFailureCodes.has(normalized)
}

function shouldRetryBinding(error: Error): boolean {
  if (!shouldRecoverWithoutFailure(error)) return false
  const code = (error as Error & { code?: unknown }).code
  if (typeof code !== 'string') return false
  const normalized = code.trim().toLowerCase()
  return normalized === 'demand_sync_pending' || normalized === 'unavailable' ||
    normalized === 'stale_session' || normalized === 'temporary' ||
    normalized === 'cancelled' || normalized === 'canceled'
}

function nativeDemandRetryDelay(attempt: number): number {
  return NATIVE_DEMAND_RETRY_CAPS_MS[
    Math.min(Math.max(attempt, 0), NATIVE_DEMAND_RETRY_CAPS_MS.length - 1)
  ] ?? 15_000
}

async function waitForAbortableOperation<T>(operation: Promise<T>, signal?: AbortSignal): Promise<T> {
  if (!signal) return await operation
  if (signal.aborted) throw abortError(signal)
  return await new Promise<T>((resolve, reject) => {
    const abort = () => {
      signal.removeEventListener('abort', abort)
      reject(abortError(signal))
    }
    signal.addEventListener('abort', abort, { once: true })
    void operation.then(
      (value) => {
        signal.removeEventListener('abort', abort)
        resolve(value)
      },
      (failure) => {
        signal.removeEventListener('abort', abort)
        reject(failure)
      },
    )
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

function connectingMachineConnectionSnapshot(
  machineId: string,
  forceRelay: boolean,
  reconnectAttempt: number,
): MachineConnectionSnapshot {
  return {
    machineId,
    phase: 'connecting',
    statusText: 'Connecting...',
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
  statusText = 'Reconnecting...',
): MachineConnectionSnapshot {
  return {
    machineId,
    phase: 'reconnecting',
    statusText,
    connectionInfo: null,
    forceRelay,
    relayInUse: false,
    reconnectAttempt,
    error: null,
  }
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
  return Object.assign(new Error('Your phone is offline.'), { code: 'network_offline', retryable: true })
}

function bindingUnavailableError(): Error {
  return Object.assign(new Error('Go client session is unavailable'), { code: 'unavailable', retryable: true })
}

function generationChangedError(): Error {
  return Object.assign(new Error('native renderer binding generation changed'), {
    code: 'cancelled',
    retryable: true,
  })
}

function userStoppedError(): Error {
  return Object.assign(new Error('Connection was stopped by the user'), {
    code: 'user_stopped',
    retryable: false,
  })
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
  private readonly closeListeners = new Set<{
    handler: ProtoClientSessionCloseHandler
    closed: boolean
  }>()
  private readonly cleanupOperations = new Set<Promise<unknown>>()
  private cleanupReleaseTimer: ReturnType<typeof globalThis.setTimeout> | null = null
  private underlyingClosedSubscription: ProtoClientSubscription | null = null
  private demandReleased = false

  constructor(
    private readonly session: ProtoClientSession,
    private readonly waitForForeground?: (signal?: AbortSignal) => Promise<void>,
    private readonly releaseDemand?: () => void,
    private readonly onDemandReleased?: (lease: NativeSessionLease) => void,
  ) {
    this.underlyingClosedSubscription = session.subscribeClosed((failure) => {
      this.terminate(failure, true)
    })
    if (!session.isAlive()) this.terminate(bindingUnavailableError(), true)
  }

  get stamp(): EndpointSessionStamp { return this.session.stamp }
  get connection() { return this.session.connection }
  get resourcePoolOwner(): ProtoClientSession { return this.session.resourcePoolOwner ?? this.session }

  belongsTo(session: ProtoClientSession): boolean { return this.session === session }

  fence(failure: ProtoClientSessionCloseError): void {
    if (!this.alive) {
      this.releaseDemandOnce()
      return
    }
    this.terminate(failure, true)
  }

  async execute(command: CommandEnvelope, options?: { signal?: AbortSignal }): Promise<ResultEnvelope> {
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
    this.requireAlive()
    const listener = { handler, closed: false }
    this.closeListeners.add(listener)
    const leaseSubscription: ProtoClientSubscription = {
      close: () => {
        if (listener.closed) return
        listener.closed = true
        this.closeListeners.delete(listener)
      },
    }
    return leaseSubscription
  }

  async openResourceStream(
    resource: ResourceHandle,
    options?: { initialUploadOffset?: bigint; signal?: AbortSignal },
  ): Promise<ProtoResourceStream> {
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
    this.terminate(undefined, false)
  }

  private terminate(failure: ProtoClientSessionCloseError | undefined, notifyClosed: boolean): void {
    if (!this.alive) return
    this.alive = false
    this.underlyingClosedSubscription?.close()
    this.underlyingClosedSubscription = null
    for (const subscription of [...this.subscriptions]) subscription.close()
    this.subscriptions.clear()
    if (notifyClosed || this.cleanupOperations.size === 0) {
      this.releaseDemandOnce()
    } else if (this.cleanupReleaseTimer === null) {
      this.cleanupReleaseTimer = globalThis.setTimeout(() => {
        this.cleanupReleaseTimer = null
        this.releaseDemandOnce()
      }, NATIVE_SESSION_CLEANUP_GRACE_MS)
    }
    const listeners = [...this.closeListeners]
    this.closeListeners.clear()
    for (const listener of listeners) {
      listener.closed = true
      if (!notifyClosed || failure === undefined) continue
      try {
        listener.handler(failure)
      } catch {
        // A consumer callback cannot prevent lease ownership cleanup.
      }
    }
  }

  private finishCleanupOperation(operation: Promise<unknown>): void {
    this.cleanupOperations.delete(operation)
    if (!this.alive && this.cleanupOperations.size === 0) this.releaseDemandOnce()
  }

  private releaseDemandOnce(): void {
    if (this.demandReleased) return
    this.demandReleased = true
    if (this.cleanupReleaseTimer !== null) {
      globalThis.clearTimeout(this.cleanupReleaseTimer)
      this.cleanupReleaseTimer = null
    }
    try {
      this.releaseDemand?.()
    } finally {
      this.onDemandReleased?.(this)
    }
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

async function awaitSessionLease(
  pending: Promise<ProtoClientSession>,
  signal: AbortSignal | undefined,
): Promise<ProtoClientSession> {
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
