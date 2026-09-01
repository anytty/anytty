import {
  NativeConnection,
  type NativeDisconnectAllRequestedEvent,
  type NativeSessionDemandInput,
  type NativeSessionDemandLease,
  type NativeSessionDemandResumeInput,
  type NativeSessionDemandResumeResult,
} from './plugins/nativeConnection'

const NATIVE_SESSION_DEMAND_TIMEOUT_MS = 5_000
const NATIVE_SESSION_ANCHOR_ATTEMPT_TIMEOUT_MS = 1_200
const NATIVE_SESSION_ANCHOR_ATTEMPTS = 3
const NATIVE_SESSION_DEMAND_RETRY_CAPS_MS = [250, 500, 1_000, 2_000, 5_000, 15_000] as const

export type { NativeSessionDemandInput, NativeSessionDemandLease, NativeSessionDemandResumeResult }
export type NativeSessionDemandSink = (input: NativeSessionDemandInput) => Promise<NativeSessionDemandLease>
export type NativeSessionDemandLeaseSource = () => Promise<NativeSessionDemandLease>
export type NativeSessionDemandResumeSink = (input: NativeSessionDemandResumeInput) => Promise<NativeSessionDemandResumeResult>
export type NativeSessionDemandOwner = symbol
export type NativeSessionResumeIntent = object
export interface NativeDisconnectAllCleanupRequest {
  stopEpoch: string
  protectedEndpointIds: string[]
}

/** Owns the current renderer's complete native connection demand projection. */
export class NativeSessionDemandCoordinator {
  private desiredEndpointOwners = new Map<string, Set<NativeSessionDemandOwner>>()
  private readonly rendererRestoreOwner = Symbol('renderer-restore')
  private rendererRestoreStarted = false
  private queue: Promise<void> = Promise.resolve()
  private revision = 0
  private commandGeneration = 0
  private dirtyRevision: number | null = null
  private retryAttempt = 0
  private retryTimer: ReturnType<typeof globalThis.setTimeout> | null = null
  private lease: NativeSessionDemandLease | null = null
  private nativeAttempt = 0
  private userStopped = false
  private observedStopEpoch: bigint | null = null
  private resumedStopEpoch: bigint | null = null
  private appliedStopEpoch: bigint | null = null
  private committedNotificationEpoch: bigint | null = null
  private stopGeneration = 0
  private resumeActionGeneration = 0
  private readonly activeResumeOperations = new Map<Promise<void>, {
    actionGeneration: number
    baseStopEpoch: bigint | null
  }>()
  private readonly resumeIntentReservations = new WeakMap<NativeSessionResumeIntent, {
    actionGeneration: number
    baseStopEpoch: bigint | null
    retryPending: boolean
  }>()
  private readonly retryPendingResumeActions = new Set<{
    actionGeneration: number
    baseStopEpoch: bigint | null
    retryPending: boolean
  }>()
  private readonly resumeIntentStopGenerations = new WeakMap<NativeSessionResumeIntent, number>()
  private readonly resumeIntentIds = new WeakMap<NativeSessionResumeIntent, string>()
  private readonly resumeIntentStopEpochs = new WeakMap<NativeSessionResumeIntent, string>()
  private readonly resumeIntentAnchors = new WeakMap<NativeSessionResumeIntent, Promise<NativeSessionDemandLease>>()
  private readonly resumeIntentOperations = new WeakMap<NativeSessionResumeIntent, Promise<void>>()
  private readonly blockedResumeIntents = new WeakSet<NativeSessionResumeIntent>()

  constructor(
    private readonly replaceDemand: NativeSessionDemandSink = (input) => NativeConnection.replaceSessionDemand(input),
    private readonly getDemandLease: NativeSessionDemandLeaseSource = () => NativeConnection.getSessionDemandLease(),
    private readonly resumeDemand: NativeSessionDemandResumeSink = (input) => NativeConnection.resumeSessionDemand(input),
  ) {}

  reconcileRenderer(): Promise<void> {
    const revision = this.beginRevision()
    return this.enqueueSnapshot(this.snapshot(), revision, this.commandGeneration)
  }

  /** Seeds the first complete renderer projection before lazy workspace code mounts. */
  restoreRenderer(endpointIds: readonly string[]): Promise<void> {
    if (this.rendererRestoreStarted) return this.reconcileRenderer()
    this.rendererRestoreStarted = true
    for (const endpointId of normalizedEndpointIds(endpointIds)) {
      this.desiredEndpointOwners.set(endpointId, new Set([this.rendererRestoreOwner]))
    }
    const revision = this.beginRevision()
    return this.enqueueSnapshot(this.snapshot(), revision, this.commandGeneration)
  }

  setWorkspaceEndpoint(machineId: string | null): Promise<void> {
    const before = this.snapshot()
    for (const [endpointId, owners] of this.desiredEndpointOwners) {
      owners.delete(this.rendererRestoreOwner)
      if (owners.size === 0) this.desiredEndpointOwners.delete(endpointId)
    }
    const endpointId = machineId?.trim() ?? ''
    if (endpointId) {
      if (this.userStopped) return Promise.reject(userStoppedError())
      const owners = this.desiredEndpointOwners.get(endpointId) ?? new Set<NativeSessionDemandOwner>()
      owners.add(this.rendererRestoreOwner)
      this.desiredEndpointOwners.set(endpointId, owners)
    }
    if (sameEndpointIds(before, this.snapshot())) return this.queue
    const revision = this.beginRevision()
    return this.enqueueSnapshot(this.snapshot(), revision, this.commandGeneration)
  }

  createResumeIntent(): NativeSessionResumeIntent {
    const intent = {}
    this.prepareResumeIntent(intent)
    // Creation is the user-action boundary. Resume the native Stop gate now so
    // React mounting and foreground barriers cannot leave a retained Stop event
    // a window in which to invalidate the first click. The manager repeats this
    // exact intent idempotently before projecting nonempty demand.
    void this.resumeForUserIntent(intent).catch(() => undefined)
    return intent
  }

  /** Confirms a retained Stop and returns the cleanup scope whose native ACK is still pending. */
  async handleDisconnectAllRequested(event: NativeDisconnectAllRequestedEvent): Promise<NativeDisconnectAllCleanupRequest | null> {
    const eventStopEpoch = normalizeDisconnectAllRequestedEvent(event)
    let refreshFailures = 0

    while (true) {
      const resumeActionGeneration = this.resumeActionGeneration
      let lease: NativeSessionDemandLease
      try {
        lease = normalizeDemandLease(await withDemandDeadline(this.getDemandLease()))
      } catch (failure) {
        refreshFailures += 1
        if (refreshFailures >= 3) throw failure
        await demandRefreshRetryDelay(refreshFailures)
        continue
      }

      // A user action that overlaps this read wins over the retained notification.
      // Read again after it settles so an old stopped=true response cannot re-latch
      // an epoch that native has already resumed.
      if (resumeActionGeneration !== this.resumeActionGeneration) {
        const newerResumes = [...this.activeResumeOperations.entries()]
          .filter(([, state]) => state.actionGeneration > resumeActionGeneration)
          .map(([operation]) => operation)
        if (newerResumes.length > 0) await Promise.allSettled(newerResumes)
        continue
      }

      const leaseStopEpoch = BigInt(lease.stopEpoch)
      if (
        leaseStopEpoch < eventStopEpoch ||
        (this.observedStopEpoch !== null && leaseStopEpoch < this.observedStopEpoch)
      ) {
        refreshFailures += 1
        if (refreshFailures >= 3) {
          throw new Error('Native session demand Stop notification is ahead of its canonical lease')
        }
        continue
      }

      const sameEpochResumes = [...this.activeResumeOperations.entries()]
        .filter(([, state]) => state.baseStopEpoch === null || state.baseStopEpoch >= leaseStopEpoch)
        .map(([operation]) => operation)
      if (sameEpochResumes.length > 0) {
        await Promise.allSettled(sameEpochResumes)
        continue
      }

      let retryingCurrentEpoch = false
      for (const reservation of [...this.retryPendingResumeActions]) {
        const baseStopEpoch = reservation.baseStopEpoch
        if (baseStopEpoch !== null && baseStopEpoch < leaseStopEpoch) {
          reservation.retryPending = false
          this.retryPendingResumeActions.delete(reservation)
          continue
        }
        if (reservation.retryPending && baseStopEpoch !== null && baseStopEpoch >= leaseStopEpoch) {
          retryingCurrentEpoch = true
        }
      }
      const stopped = this.observeNativeStop(lease)
      this.adoptResumeLease(lease)
      if (stopped && !retryingCurrentEpoch) this.resetProjectionForObservedStop(lease)
      if (this.committedNotificationEpoch !== null && leaseStopEpoch <= this.committedNotificationEpoch) {
        return null
      }
      return {
        stopEpoch: lease.stopEpoch,
        protectedEndpointIds: lease.stopped ? [] : [...lease.endpointIds],
      }
    }
  }

  /** Advances the notification fence only after endpoint and transfer cleanup has completed. */
  commitDisconnectAllCleanup(stopEpoch: string): void {
    if (!/^\d+$/.test(stopEpoch)) throw new Error('Native session demand Stop cleanup epoch is invalid')
    const committed = BigInt(stopEpoch)
    if (this.committedNotificationEpoch === null || committed > this.committedNotificationEpoch) {
      this.committedNotificationEpoch = committed
    }
  }

  resumeIntentCoversStopEpoch(intent: NativeSessionResumeIntent | null, stopEpoch: string): boolean {
    if (intent === null || !/^\d+$/.test(stopEpoch)) return false
    const intentStopEpoch = this.resumeIntentStopEpochs.get(intent)
    return intentStopEpoch !== undefined && BigInt(intentStopEpoch) >= BigInt(stopEpoch)
  }

  resumeForUserIntent(intent: NativeSessionResumeIntent): Promise<void> {
    this.prepareResumeIntent(intent)
    const intentStopGeneration = this.bindIntentStopGeneration(intent)
    if (intentStopGeneration !== this.stopGeneration || this.blockedResumeIntents.has(intent)) {
      return Promise.reject(userStoppedError())
    }

    const existingIntentOperation = this.resumeIntentOperations.get(intent)
    if (existingIntentOperation) return existingIntentOperation

    this.resumeActionGeneration += 1
    const actionGeneration = this.resumeActionGeneration
    const operationState = this.resumeIntentReservations.get(intent) ?? {
      actionGeneration,
      baseStopEpoch: null,
      retryPending: false,
    }
    operationState.actionGeneration = actionGeneration
    operationState.retryPending = false
    this.retryPendingResumeActions.delete(operationState)
    this.resumeIntentReservations.set(intent, operationState)
    this.cancelRetry()
    this.retryAttempt = 0
    this.commandGeneration += 1
    this.nativeAttempt += 1
    this.revision += 1
    this.dirtyRevision = null
    this.lease = null
    this.queue = Promise.resolve()
    let operationStopGeneration = intentStopGeneration
    let operation: Promise<void>
    operation = withDemandDeadline((async () => {
      const prepared = await this.resumeInput(intent, intentStopGeneration)
      operationStopGeneration = prepared.stopGeneration
      operationState.baseStopEpoch = BigInt(prepared.input.baseStopEpoch)
      return this.resumeDemand(prepared.input)
    })()).then((value) => {
      if (this.stopGeneration !== operationStopGeneration) throw userStoppedError()
      const next = normalizeDemandResumeResult(value)
      const nextStopEpoch = BigInt(next.stopEpoch)
      if (this.observedStopEpoch !== null && nextStopEpoch < this.observedStopEpoch) {
        this.blockedResumeIntents.add(intent)
        throw userStoppedError()
      }
      if (this.observeNativeStop(next)) this.resetProjectionForObservedStop(next)
      if (next.outcome === 'stopped') {
        this.blockedResumeIntents.add(intent)
        if (this.resumedStopEpoch === null || nextStopEpoch > this.resumedStopEpoch) {
          this.userStopped = true
        }
        throw userStoppedError()
      }
      this.resumedStopEpoch = nextStopEpoch
      this.adoptResumeLease(next)
      this.userStopped = false
    }).then(() => {
      this.retryPendingResumeActions.delete(operationState)
      this.resumeIntentReservations.delete(intent)
    }, (failure: unknown) => {
      const reservationIsStale = operationState.baseStopEpoch !== null &&
        this.observedStopEpoch !== null && operationState.baseStopEpoch < this.observedStopEpoch
      if (resumeFailureKeepsIntent(failure) && !reservationIsStale) {
        operationState.retryPending = true
        this.retryPendingResumeActions.add(operationState)
      } else {
        operationState.retryPending = false
        this.retryPendingResumeActions.delete(operationState)
        this.resumeIntentReservations.delete(intent)
      }
      throw failure
    }).finally(() => {
      if (this.resumeIntentOperations.get(intent) === operation) {
        this.resumeIntentOperations.delete(intent)
      }
      this.activeResumeOperations.delete(operation)
    })
    this.resumeIntentOperations.set(intent, operation)
    this.activeResumeOperations.set(operation, operationState)
    return operation
  }

  /**
   * Revalidates an async user continuation against native Stop truth using the
   * exact intent created at the synchronous action boundary.
   */
  async confirmResumeIntent(intent: NativeSessionResumeIntent): Promise<void> {
    const inFlight = this.resumeIntentOperations.get(intent)
    if (inFlight) {
      try {
        await inFlight
      } catch (failure) {
        if (!resumeFailureKeepsIntent(failure)) throw failure
      }
    }
    await this.resumeForUserIntent(intent)
  }

  setActive(machineId: string, active: boolean, owner: NativeSessionDemandOwner): Promise<void> {
    const endpointId = machineId.trim()
    if (!endpointId) return Promise.reject(new Error('machineId is required'))
    if (active && this.userStopped) return Promise.reject(userStoppedError())

    const currentOwners = this.desiredEndpointOwners.get(endpointId)
    const alreadyActive = currentOwners?.has(owner) === true
    if (alreadyActive === active) return active ? this.reconcileRenderer() : this.queue

    const wasDemanded = (currentOwners?.size ?? 0) > 0
    if (active) {
      const owners = currentOwners ?? new Set<NativeSessionDemandOwner>()
      owners.add(owner)
      this.desiredEndpointOwners.set(endpointId, owners)
    } else if (currentOwners) {
      currentOwners.delete(owner)
      if (currentOwners.size === 0) this.desiredEndpointOwners.delete(endpointId)
    }
    const isDemanded = (this.desiredEndpointOwners.get(endpointId)?.size ?? 0) > 0
    if (wasDemanded === isDemanded) return this.queue
    const revision = this.beginRevision()
    return this.enqueueSnapshot(this.snapshot(), revision, this.commandGeneration)
  }

  private snapshot(): string[] {
    return [...this.desiredEndpointOwners.keys()].sort()
  }

  private beginRevision(): number {
    this.cancelRetry()
    this.retryAttempt = 0
    this.nativeAttempt += 1
    this.revision += 1
    this.dirtyRevision = this.revision
    return this.revision
  }

  private enqueueSnapshot(
    endpointIds: string[],
    revision: number,
    commandGeneration: number,
  ): Promise<void> {
    const operation = this.queue.then(async () => {
      if (!this.isCurrent(revision, commandGeneration)) return
      const nativeAttempt = ++this.nativeAttempt
      try {
        await withDemandDeadline(this.submitSnapshot(endpointIds, revision, commandGeneration, nativeAttempt))
        if (!this.isCurrentAttempt(revision, commandGeneration, nativeAttempt)) return
        this.dirtyRevision = null
        this.retryAttempt = 0
        this.cancelRetry()
      } catch (failure) {
        if (this.isCurrentAttempt(revision, commandGeneration, nativeAttempt)) {
          this.nativeAttempt += 1
          this.lease = null
          this.scheduleRetry(revision, commandGeneration)
        }
        throw failure
      }
    })
    this.queue = operation.catch(() => undefined)
    return operation
  }

  private async submitSnapshot(
    endpointIds: string[],
    revision: number,
    commandGeneration: number,
    nativeAttempt: number,
  ): Promise<void> {
    let lease = this.lease
    if (lease === null) {
      const acquired = normalizeDemandLease(await this.getDemandLease())
      if (!this.isCurrentAttempt(revision, commandGeneration, nativeAttempt)) return
      if (this.observeNativeStop(acquired)) {
        this.resetProjectionForObservedStop(acquired)
        if (endpointIds.length > 0) throw userStoppedError()
        return
      }
      this.lease = acquired
      lease = acquired
    }
    const next = normalizeDemandLease(await this.replaceDemand({
      attachmentId: lease.attachmentId,
      baseDemandRevision: lease.demandRevision,
      endpointIds,
    }))
    if (!this.isCurrentAttempt(revision, commandGeneration, nativeAttempt)) return
    if (this.observeNativeStop(next)) {
      this.resetProjectionForObservedStop(next)
      if (endpointIds.length > 0) throw userStoppedError()
      return
    }
    this.lease = next
  }

  private isCurrent(revision: number, commandGeneration: number): boolean {
    return commandGeneration === this.commandGeneration &&
      revision === this.revision &&
      this.dirtyRevision === revision
  }

  private isCurrentAttempt(revision: number, commandGeneration: number, nativeAttempt: number): boolean {
    return nativeAttempt === this.nativeAttempt && this.isCurrent(revision, commandGeneration)
  }

  private observeNativeStop(lease: NativeSessionDemandLease): boolean {
    const stopEpoch = BigInt(lease.stopEpoch)
    const previous = this.observedStopEpoch
    if (previous !== null && stopEpoch < previous) {
      throw new Error('Native session demand stop epoch regressed')
    }
    this.observedStopEpoch = stopEpoch
    return lease.stopped && (this.resumedStopEpoch === null || stopEpoch > this.resumedStopEpoch)
  }

  private resetProjectionForObservedStop(lease: NativeSessionDemandLease): void {
    const stopEpoch = BigInt(lease.stopEpoch)
    if (this.resumedStopEpoch !== null && stopEpoch <= this.resumedStopEpoch) return
    if (this.appliedStopEpoch !== null && stopEpoch <= this.appliedStopEpoch) {
      this.userStopped = true
      this.adoptResumeLease(lease)
      return
    }

    this.appliedStopEpoch = stopEpoch
    this.desiredEndpointOwners.clear()
    this.userStopped = true
    this.stopGeneration += 1
    this.cancelRetry()
    this.retryAttempt = 0
    this.commandGeneration += 1
    this.nativeAttempt += 1
    this.revision += 1
    this.dirtyRevision = null
    this.lease = lease
    this.queue = Promise.resolve()
  }

  private bindIntentStopGeneration(intent: NativeSessionResumeIntent): number {
    const existing = this.resumeIntentStopGenerations.get(intent)
    if (existing !== undefined) return existing
    this.resumeIntentStopGenerations.set(intent, this.stopGeneration)
    return this.stopGeneration
  }

  private prepareResumeIntent(intent: NativeSessionResumeIntent): void {
    const stopGeneration = this.bindIntentStopGeneration(intent)
    this.resumeIntentId(intent)
    if (this.resumeIntentStopEpochs.has(intent) || this.resumeIntentAnchors.has(intent)) return

    // Every distinct action gets a fresh process-owned Stop anchor. The latest
    // locally observed epoch may predate a retained native notification.
    void this.startResumeIntentAnchor(intent, stopGeneration).catch(() => undefined)
  }

  private resumeIntentId(intent: NativeSessionResumeIntent): string {
    const existing = this.resumeIntentIds.get(intent)
    if (existing) return existing
    const next = createResumeIntentId()
    this.resumeIntentIds.set(intent, next)
    return next
  }

  private async resumeInput(
    intent: NativeSessionResumeIntent,
    intentStopGeneration: number,
  ): Promise<{ input: NativeSessionDemandResumeInput; stopGeneration: number }> {
    const intentId = this.resumeIntentId(intent)
    if (intentStopGeneration !== this.stopGeneration) throw userStoppedError()
    const existingStopEpoch = this.resumeIntentStopEpochs.get(intent)
    if (existingStopEpoch !== undefined) {
      return {
        input: { intentId, baseStopEpoch: existingStopEpoch },
        stopGeneration: intentStopGeneration,
      }
    }

    await this.startResumeIntentAnchor(intent, intentStopGeneration)
    const boundStopGeneration = this.resumeIntentStopGenerations.get(intent)
    const baseStopEpoch = this.resumeIntentStopEpochs.get(intent)
    if (boundStopGeneration === undefined || baseStopEpoch === undefined) {
      throw new Error('Native session demand stop epoch baseline is unavailable')
    }
    if (boundStopGeneration !== this.stopGeneration) throw userStoppedError()
    return {
      input: { intentId, baseStopEpoch },
      stopGeneration: boundStopGeneration,
    }
  }

  private startResumeIntentAnchor(
    intent: NativeSessionResumeIntent,
    generation: number,
  ): Promise<NativeSessionDemandLease> {
    const existing = this.resumeIntentAnchors.get(intent)
    if (existing) return existing

    let operation: Promise<NativeSessionDemandLease>
    operation = this.readResumeIntentAnchor().then((lease) => {
      const leaseStopEpoch = BigInt(lease.stopEpoch)
      if (generation !== this.stopGeneration) {
        // Another post-Stop action may have observed this exact epoch first and
        // advanced the local generation. An anchor at that same (or a newer)
        // canonical epoch is still a valid post-Stop action; an older anchor is
        // the pre-Stop action that must remain fenced.
        if (this.observedStopEpoch === null || leaseStopEpoch < this.observedStopEpoch) {
          throw userStoppedError()
        }
      }
      if (this.observeNativeStop(lease)) {
        this.resetProjectionForObservedStop(lease)
      } else {
        this.adoptResumeLease(lease)
      }
      this.resumeIntentStopGenerations.set(intent, this.stopGeneration)
      this.resumeIntentStopEpochs.set(intent, lease.stopEpoch)
      return lease
    }).finally(() => {
      if (this.resumeIntentAnchors.get(intent) === operation) this.resumeIntentAnchors.delete(intent)
    })
    this.resumeIntentAnchors.set(intent, operation)
    return operation
  }

  private async readResumeIntentAnchor(): Promise<NativeSessionDemandLease> {
    let lastFailure: unknown
    for (let attempt = 1; attempt <= NATIVE_SESSION_ANCHOR_ATTEMPTS; attempt += 1) {
      try {
        return normalizeDemandLease(await withDemandDeadline(
          this.getDemandLease(),
          NATIVE_SESSION_ANCHOR_ATTEMPT_TIMEOUT_MS,
        ))
      } catch (failure) {
        lastFailure = failure
        if (attempt >= NATIVE_SESSION_ANCHOR_ATTEMPTS) throw failure
        await demandRefreshRetryDelay(attempt)
      }
    }
    throw lastFailure
  }

  private adoptResumeLease(next: NativeSessionDemandLease): void {
    const current = this.lease
    if (current === null) {
      this.lease = next
      return
    }
    if (current.attachmentId !== next.attachmentId) return
    if (BigInt(current.stopEpoch) > BigInt(next.stopEpoch)) return
    if (
      current.stopEpoch === next.stopEpoch &&
      BigInt(current.demandRevision) > BigInt(next.demandRevision)
    ) return
    this.lease = next
  }

  private scheduleRetry(revision: number, commandGeneration: number): void {
    if (!this.isCurrent(revision, commandGeneration) || this.retryTimer !== null) return
    const delay = NATIVE_SESSION_DEMAND_RETRY_CAPS_MS[
      Math.min(this.retryAttempt, NATIVE_SESSION_DEMAND_RETRY_CAPS_MS.length - 1)
    ] ?? 15_000
    this.retryAttempt += 1
    this.retryTimer = globalThis.setTimeout(() => {
      this.retryTimer = null
      if (!this.isCurrent(revision, commandGeneration)) return
      void this.enqueueSnapshot(this.snapshot(), revision, commandGeneration).catch(() => undefined)
    }, delay)
  }

  private cancelRetry(): void {
    if (this.retryTimer === null) return
    globalThis.clearTimeout(this.retryTimer)
    this.retryTimer = null
  }
}

function userStoppedError(): Error {
  return Object.assign(new Error('Native session demand was stopped by the user'), {
    code: 'user_stopped',
    retryable: false,
  })
}

function resumeFailureKeepsIntent(failure: unknown): boolean {
  const value = failure as { code?: unknown; retryable?: unknown } | null
  const code = typeof value?.code === 'string' ? value.code.trim().toLowerCase() : ''
  return code !== 'user_stopped' && code !== 'cancelled' && value?.retryable !== false
}

function normalizeDemandLease(value: NativeSessionDemandLease): NativeSessionDemandLease {
  const attachmentId = value?.attachmentId?.trim()
  const demandRevision = value?.demandRevision?.trim()
  const stopEpoch = value?.stopEpoch?.trim()
  if (
    !attachmentId || !demandRevision || !stopEpoch || !/^\d+$/.test(demandRevision) || !/^\d+$/.test(stopEpoch) ||
    !Array.isArray(value?.endpointIds) || value.endpointIds.some((endpointId) => typeof endpointId !== 'string' || !endpointId.trim()) ||
    typeof value?.stopped !== 'boolean'
  ) {
    throw new Error('Native session demand lease is invalid')
  }
  return {
    attachmentId,
    demandRevision,
    stopEpoch,
    endpointIds: normalizedEndpointIds(value.endpointIds),
    stopped: value.stopped,
  }
}

function normalizeDemandResumeResult(value: NativeSessionDemandResumeResult): NativeSessionDemandResumeResult {
  const lease = normalizeDemandLease(value)
  if (value?.outcome !== 'resumed' && value?.outcome !== 'stopped') {
    throw new Error('Native session demand resume outcome is invalid')
  }
  return { ...lease, outcome: value.outcome }
}

function normalizeDisconnectAllRequestedEvent(value: NativeDisconnectAllRequestedEvent): bigint {
  const stopEpoch = value?.stopEpoch?.trim()
  if (!stopEpoch || !/^\d+$/.test(stopEpoch) || value?.stopped !== true) {
    throw new Error('Native disconnect-all notification is invalid')
  }
  return BigInt(stopEpoch)
}

function createResumeIntentId(): string {
  if (typeof globalThis.crypto?.randomUUID === 'function') return globalThis.crypto.randomUUID()
  const random = new Uint8Array(16)
  globalThis.crypto?.getRandomValues?.(random)
  const hasEntropy = random.some((value) => value !== 0)
  if (hasEntropy) return [...random].map((value) => value.toString(16).padStart(2, '0')).join('')
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`
}

function normalizedEndpointIds(endpointIds: readonly string[]): string[] {
  const normalized = new Set<string>()
  for (const value of endpointIds) {
    const endpointId = value.trim()
    if (!endpointId) throw new Error('Native session demand endpoint ID is invalid')
    normalized.add(endpointId)
  }
  return [...normalized].sort()
}

function sameEndpointIds(left: readonly string[], right: readonly string[]): boolean {
  return left.length === right.length && left.every((endpointId, index) => endpointId === right[index])
}

async function withDemandDeadline<T>(
  operation: Promise<T>,
  timeoutMs = NATIVE_SESSION_DEMAND_TIMEOUT_MS,
): Promise<T> {
  let timeout: ReturnType<typeof globalThis.setTimeout> | undefined
  try {
    return await Promise.race([
      operation,
      new Promise<never>((_, reject) => {
        timeout = globalThis.setTimeout(() => {
          reject(Object.assign(new Error('Native session demand reconciliation timed out'), {
            code: 'demand_sync_pending',
            retryable: true,
          }))
        }, timeoutMs)
      }),
    ])
  } finally {
    if (timeout !== undefined) globalThis.clearTimeout(timeout)
  }
}

async function demandRefreshRetryDelay(attempt: number): Promise<void> {
  const delay = attempt <= 1 ? 50 : 150
  await new Promise<void>((resolve) => globalThis.setTimeout(resolve, delay))
}

export const nativeSessionDemand = new NativeSessionDemandCoordinator()
