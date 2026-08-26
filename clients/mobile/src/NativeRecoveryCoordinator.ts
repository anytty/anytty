export type NativeRecoveryTrigger = 'app_resume' | 'page_visible' | 'renderer_stall' | 'binding_closed' | 'manual_retry'
export type NativeRecoveryIntent = 'ensure_ready' | 'repair'

export interface NativeRecoveryRequest {
  intent: NativeRecoveryIntent
  trigger: NativeRecoveryTrigger
}

export interface NativeRecoveryWork extends NativeRecoveryRequest {
  attempt: number
}

export interface NativeForegroundResumeTarget {
  endpointId: string
  resume: () => Promise<void>
}

export interface NativeForegroundResumeResult {
  total: number
  resumed: number
  failures: unknown[]
}

interface NativeRecoverySchedule {
  beginAttempt: () => number
  isCurrent: (attempt: number) => boolean
  execute: (work: NativeRecoveryWork) => Promise<void>
}

interface ScheduledRecovery {
  request: NativeRecoveryRequest
  attempt: number
  execute: NativeRecoverySchedule['execute']
}

const NO_FAILURE = Symbol('no recovery failure')

/** Serializes native recovery while retaining one upgraded request behind the active attempt. */
export class NativeRecoveryCoordinator {
  private current: ScheduledRecovery | null = null
  private pending: ScheduledRecovery | null = null
  private inFlight: Promise<void> | null = null
  private workerId = 0

  request(request: NativeRecoveryRequest, schedule: NativeRecoverySchedule): Promise<void> {
    const running = this.inFlight
    if (running) {
      if (this.pending && schedule.isCurrent(this.pending.attempt) && covers(this.pending.request, request)) {
        return running
      }
      if (!this.pending && this.current && schedule.isCurrent(this.current.attempt) && covers(this.current.request, request)) {
        return running
      }

      const attempt = schedule.beginAttempt()
      this.pending = {
        request: mergeRequests(this.current?.request, this.pending?.request, request),
        attempt,
        execute: schedule.execute,
      }
      return running
    }

    const work: ScheduledRecovery = {
      request,
      attempt: schedule.beginAttempt(),
      execute: schedule.execute,
    }
    const workerId = ++this.workerId
    this.current = work
    // Defer execution until inFlight is published so a synchronous second request can join it.
    const worker = Promise.resolve().then(() => this.drain(work, workerId))
    this.inFlight = worker
    return worker
  }

  private async drain(initial: ScheduledRecovery, workerId: number): Promise<void> {
    let work: ScheduledRecovery | null = initial
    let failure: unknown = NO_FAILURE
    try {
      while (work) {
        this.current = work
        try {
          await work.execute({ ...work.request, attempt: work.attempt })
          failure = NO_FAILURE
        } catch (error) {
          failure = error
        }

        if (this.current === work) this.current = null
        work = this.pending
        this.pending = null
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
}

/** Waits for every demanded endpoint to settle without making one endpoint a global APP failure. */
export async function resumeNativeForegroundTargets(
  targets: Iterable<NativeForegroundResumeTarget>,
): Promise<NativeForegroundResumeResult> {
  const pending = [...targets]
  const results = await Promise.allSettled(pending.map((target) => target.resume()))
  const failures = results.flatMap((result) => result.status === 'rejected' ? [result.reason] : [])
  return {
    total: pending.length,
    resumed: pending.length - failures.length,
    failures,
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
