export type NativeRecoveryTrigger =
  | 'app_resume'
  | 'page_visible'
  | 'renderer_stall'
  | 'binding_closed'
  | 'manual_retry'
  | 'automatic_retry'
export type NativeRecoveryIntent = 'ensure_ready' | 'repair'

export interface NativeRecoveryRequest {
  intent: NativeRecoveryIntent
  trigger: NativeRecoveryTrigger
}

export interface NativeRecoveryWork extends NativeRecoveryRequest {
  attempt: number
  signal: AbortSignal
}

export interface NativeForegroundResumeTarget {
  endpointId: string
  resume: () => Promise<void>
}

export interface NativeForegroundResumeResult {
  total: number
  resumed: number
  failures: Array<{ endpointId: string; failure: unknown }>
}

export interface NativeForegroundResumeBatch {
  total: number
  settled: Promise<NativeForegroundResumeResult>
}

export interface NativeForegroundWork {
  demand: Promise<void>
  endpoints: NativeForegroundResumeBatch
}

interface NativeRecoverySchedule {
  beginAttempt: () => number
  isCurrent: (attempt: number) => boolean
  execute: (work: NativeRecoveryWork) => Promise<void>
  retryDelay?: (failure: unknown, retryAttempt: number, work: NativeRecoveryWork) => number | null
  onRetryScheduled?: (failure: unknown, delay: number, retryAttempt: number, work: NativeRecoveryWork) => void
  onTerminalFailure?: (failure: unknown, work: NativeRecoveryWork) => void
}

interface ScheduledRecovery {
  request: NativeRecoveryRequest
  attempt: number
  controller: AbortController
  retryAttempt: number
  schedule: NativeRecoverySchedule
}

const NO_FAILURE = Symbol('no recovery failure')

/** Serializes native recovery, including its one process-local automatic retry owner. */
export class NativeRecoveryCoordinator {
  private current: ScheduledRecovery | null = null
  private pending: ScheduledRecovery | null = null
  private inFlight: Promise<void> | null = null
  private retryWait: { timer: ReturnType<typeof globalThis.setTimeout>; finish: () => void } | null = null
  private workerId = 0

  request(request: NativeRecoveryRequest, schedule: NativeRecoverySchedule): Promise<void> {
    const running = this.inFlight
    if (running) {
      if (
        this.retryWait !== null &&
        this.current &&
        this.current.schedule.isCurrent(this.current.attempt) &&
        request.trigger !== 'manual_retry' &&
        recoveryIntentPriority(request.intent) <= recoveryIntentPriority(this.current.request.intent)
      ) {
        // Lifecycle/callback storms are evidence for the already scheduled retry, not a
        // new lineage. Preserve both its delay and retryAttempt; only manual intent or a
        // strictly stronger repair may wake the worker early.
        return running
      }
      if (
        this.retryWait === null &&
        this.pending && schedule.isCurrent(this.pending.attempt) && covers(this.pending.request, request)
      ) {
        return running
      }
      if (
        this.retryWait === null &&
        !this.pending && this.current &&
        schedule.isCurrent(this.current.attempt) && covers(this.current.request, request)
      ) {
        return running
      }

      this.pending?.controller.abort(new DOMException('Superseded by a newer native recovery', 'AbortError'))
      const attempt = schedule.beginAttempt()
      this.pending = {
        request: mergeRequests(this.current?.request, this.pending?.request, request),
        attempt,
        controller: new AbortController(),
        retryAttempt: 0,
        schedule,
      }
      this.current?.controller.abort(new DOMException('Superseded by a newer native recovery', 'AbortError'))
      this.finishRetryWait()
      return running
    }

    const work: ScheduledRecovery = {
      request,
      attempt: schedule.beginAttempt(),
      controller: new AbortController(),
      retryAttempt: 0,
      schedule,
    }
    const workerId = ++this.workerId
    this.current = work
    // Defer execution until inFlight is published so a synchronous second request can join it.
    const worker = Promise.resolve().then(() => this.drain(work, workerId))
    this.inFlight = worker
    return worker
  }

  /** Cancels active and delayed recovery without letting the abandoned attempt publish failure. */
  cancel(reason: unknown = new DOMException('Native recovery was cancelled', 'AbortError')): void {
    this.current?.controller.abort(reason)
    this.pending?.controller.abort(reason)
    this.current = null
    this.pending = null
    this.finishRetryWait()
  }

  private async drain(initial: ScheduledRecovery, workerId: number): Promise<void> {
    let work: ScheduledRecovery | null = initial
    let failure: unknown = NO_FAILURE
    try {
      while (work) {
        this.current = work
        const recoveryWork: NativeRecoveryWork = {
          ...work.request,
          attempt: work.attempt,
          signal: work.controller.signal,
        }
        try {
          await work.schedule.execute(recoveryWork)
          failure = NO_FAILURE
        } catch (error) {
          failure = error
        }

        if (this.current === work) this.current = null
        const pending = this.takePending()
        if (pending) {
          work = pending
          failure = NO_FAILURE
          continue
        }
        if (
          failure !== NO_FAILURE &&
          !work.controller.signal.aborted &&
          work.schedule.isCurrent(work.attempt)
        ) {
          const delay = work.schedule.retryDelay?.(failure, work.retryAttempt, recoveryWork) ?? null
          if (delay !== null) {
            const retryAttempt: number = work.retryAttempt + 1
            work.schedule.onRetryScheduled?.(failure, delay, retryAttempt, recoveryWork)
            this.current = work
            await this.waitForRetry(Math.max(0, delay))
            const requested = this.takePending()
            if (requested) {
              work = requested
              failure = NO_FAILURE
              continue
            }
            if (work.controller.signal.aborted || !work.schedule.isCurrent(work.attempt)) {
              work = null
              failure = NO_FAILURE
              continue
            }
            work.controller.abort(new DOMException('Native recovery retry advanced the attempt', 'AbortError'))
            work = {
              request: { intent: 'repair', trigger: 'automatic_retry' },
              attempt: work.schedule.beginAttempt(),
              controller: new AbortController(),
              retryAttempt,
              schedule: work.schedule,
            }
            failure = NO_FAILURE
            continue
          }
          work.schedule.onTerminalFailure?.(failure, recoveryWork)
        }
        if (work.controller.signal.aborted || !work.schedule.isCurrent(work.attempt)) {
          failure = NO_FAILURE
        }
        work = null
      }
      if (failure !== NO_FAILURE) throw failure
    } finally {
      if (this.workerId === workerId) {
        this.current = null
        this.pending = null
        this.inFlight = null
      }
    }
  }

  private takePending(): ScheduledRecovery | null {
    const pending = this.pending
    this.pending = null
    return pending
  }

  private async waitForRetry(delay: number): Promise<void> {
    await new Promise<void>((resolve) => {
      let settled = false
      const wait = {
        timer: globalThis.setTimeout(() => finish(), delay),
        finish,
      }
      function finish() {
        if (settled) return
        settled = true
        globalThis.clearTimeout(wait.timer)
        resolve()
      }
      this.retryWait = wait
    })
    this.retryWait = null
  }

  private finishRetryWait(): void {
    const wait = this.retryWait
    this.retryWait = null
    wait?.finish()
  }
}

/** Starts every endpoint fence synchronously; endpoint convergence remains independently observable. */
export function startNativeForegroundTargets(
  targets: Iterable<NativeForegroundResumeTarget>,
): NativeForegroundResumeBatch {
  const pending = [...targets]
  return {
    total: pending.length,
    settled: Promise.allSettled(pending.map((target) => target.resume())).then((results) => {
      const failures = results.flatMap((result, index) => result.status === 'rejected'
        ? [{ endpointId: pending[index]?.endpointId ?? 'unknown', failure: result.reason }]
        : [])
      return {
        total: pending.length,
        resumed: pending.length - failures.length,
        failures,
      }
    }),
  }
}

/** Starts demand reconciliation and endpoint fences independently so neither can gate the other. */
export function startNativeForegroundWork(
  reconcileDemand: () => Promise<void>,
  targets: Iterable<NativeForegroundResumeTarget>,
): NativeForegroundWork {
  let demand: Promise<void>
  try {
    demand = reconcileDemand()
  } catch (failure) {
    demand = Promise.reject(failure)
  }
  return {
    demand,
    endpoints: startNativeForegroundTargets(targets),
  }
}

function covers(current: NativeRecoveryRequest, requested: NativeRecoveryRequest): boolean {
  return recoveryIntentPriority(current.intent) >= recoveryIntentPriority(requested.intent)
}

function mergeRequests(...requests: Array<NativeRecoveryRequest | null | undefined>): NativeRecoveryRequest {
  return requests.reduce<NativeRecoveryRequest>((merged, request) => {
    if (!request) return merged
    return recoveryIntentPriority(request.intent) >= recoveryIntentPriority(merged.intent)
      ? request
      : merged
  }, {
    intent: 'ensure_ready',
    trigger: 'app_resume',
  })
}

function recoveryIntentPriority(intent: NativeRecoveryIntent): number {
  return intent === 'repair' ? 1 : 0
}
