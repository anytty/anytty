import { describe, expect, it, vi } from 'vitest'
import { NativeGenerationRecoveryFence } from './NativeGenerationRecoveryFence'
import {
  NativeRecoveryCoordinator,
  startNativeForegroundTargets,
  startNativeForegroundWork,
  type NativeRecoveryWork,
} from './NativeRecoveryCoordinator'

describe('NativeRecoveryCoordinator', () => {
  it('coalesces repeated work into one active recovery', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => await gate.promise)

    const first = harness.request('repair')
    const repeated = harness.request('repair')

    expect(repeated).toBe(first)
    gate.resolve()
    await first
    expect(harness.execute).toHaveBeenCalledOnce()
  })

  it('drains one upgraded request after invalidating the active attempt', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => await gate.promise)

    const recovery = harness.request('ensure_ready')
    harness.request('repair')
    harness.request('ensure_ready')
    gate.resolve()
    await recovery

    expect(harness.execute).toHaveBeenCalledTimes(2)
    expect(harness.execute.mock.calls[1]?.[0]).toMatchObject({
      intent: 'repair',
    })
  })

  it('queues the same work when the active attempt was invalidated in the background', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => await gate.promise)

    const recovery = harness.request('ensure_ready')
    harness.fence.invalidate()
    harness.request('ensure_ready')
    gate.resolve()
    await recovery

    expect(harness.execute).toHaveBeenCalledTimes(2)
    expect(harness.execute.mock.calls[1]?.[0]).toMatchObject({
      intent: 'ensure_ready',
    })
  })

  it('continues with queued recovery after the superseded attempt fails', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => {
      await gate.promise
      throw new Error('stale bridge failed')
    })

    const recovery = harness.request('ensure_ready')
    harness.request('repair')
    gate.resolve()

    await expect(recovery).resolves.toBeUndefined()
    expect(harness.execute).toHaveBeenCalledTimes(2)
  })

  it('does not downgrade a pending repair request', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => await gate.promise)

    const recovery = harness.request('ensure_ready')
    harness.request('repair', 'binding_closed')
    harness.request('ensure_ready', 'page_visible')
    gate.resolve()
    await recovery

    expect(harness.execute).toHaveBeenCalledTimes(2)
    expect(harness.execute.mock.calls[1]?.[0]).toMatchObject({
      intent: 'repair',
      trigger: 'binding_closed',
    })
  })

  it('automatically retries one failed recovery without another lifecycle event', async () => {
    vi.useFakeTimers()
    const harness = recoveryHarness({ retryDelay: () => 250 })
    harness.execute
      .mockRejectedValueOnce(new Error('native bridge is rebuilding'))
      .mockResolvedValueOnce(undefined)
    try {
      const recovery = harness.request('repair', 'binding_closed')
      await vi.advanceTimersByTimeAsync(0)
      expect(harness.onRetryScheduled).toHaveBeenCalledOnce()

      await vi.advanceTimersByTimeAsync(249)
      expect(harness.execute).toHaveBeenCalledOnce()
      await vi.advanceTimersByTimeAsync(1)

      await expect(recovery).resolves.toBeUndefined()
      expect(harness.execute).toHaveBeenCalledTimes(2)
      expect(harness.execute.mock.calls[1]?.[0]).toMatchObject({
        intent: 'repair',
        trigger: 'automatic_retry',
        attempt: 2,
      })
      await vi.advanceTimersByTimeAsync(60_000)
      expect(harness.execute).toHaveBeenCalledTimes(2)
    } finally {
      vi.useRealTimers()
    }
  })

  it('cancels a delayed automatic retry when user Stop wins', async () => {
    vi.useFakeTimers()
    const harness = recoveryHarness({ retryDelay: () => 250 })
    harness.execute.mockRejectedValue(new Error('native bridge is rebuilding'))
    try {
      const recovery = harness.request('repair', 'binding_closed')
      await vi.waitFor(() => expect(harness.onRetryScheduled).toHaveBeenCalledOnce())

      harness.fence.invalidate()
      harness.coordinator.cancel(Object.assign(new Error('Stopped by user'), { code: 'user_stopped' }))
      await expect(recovery).resolves.toBeUndefined()
      await vi.advanceTimersByTimeAsync(30_000)

      expect(harness.execute).toHaveBeenCalledOnce()
    } finally {
      vi.useRealTimers()
    }
  })

  it('lets a new generation wake and replace an automatic retry delay', async () => {
    vi.useFakeTimers()
    const harness = recoveryHarness({ retryDelay: () => 15_000 })
    harness.execute
      .mockRejectedValueOnce(new Error('old generation failed'))
      .mockResolvedValueOnce(undefined)
    try {
      const recovery = harness.request('ensure_ready')
      await vi.waitFor(() => expect(harness.onRetryScheduled).toHaveBeenCalledOnce())

      const replacement = harness.request('repair', 'manual_retry')

      await expect(replacement).resolves.toBeUndefined()
      await expect(recovery).resolves.toBeUndefined()
      expect(harness.execute).toHaveBeenCalledTimes(2)
      expect(harness.execute.mock.calls[1]?.[0]).toMatchObject({
        intent: 'repair',
        trigger: 'manual_retry',
      })
      expect(harness.execute.mock.calls[0]?.[0].signal.aborted).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })

  it('coalesces automatic lifecycle storms without shortening or resetting retry backoff', async () => {
    vi.useFakeTimers()
    const harness = recoveryHarness({ retryDelay: () => 15_000 })
    harness.execute
      .mockRejectedValueOnce(new Error('first repair failed'))
      .mockRejectedValueOnce(new Error('second repair failed'))
      .mockResolvedValueOnce(undefined)
    try {
      const recovery = harness.request('repair', 'binding_closed')
      await vi.advanceTimersByTimeAsync(0)
      expect(harness.onRetryScheduled).toHaveBeenCalledTimes(1)

      expect(harness.request('repair', 'binding_closed')).toBe(recovery)
      expect(harness.request('ensure_ready', 'page_visible')).toBe(recovery)
      expect(harness.request('ensure_ready', 'app_resume')).toBe(recovery)
      await vi.advanceTimersByTimeAsync(14_999)
      expect(harness.execute).toHaveBeenCalledTimes(1)

      await vi.advanceTimersByTimeAsync(1)
      expect(harness.execute).toHaveBeenCalledTimes(2)
      expect(harness.onRetryScheduled).toHaveBeenCalledTimes(2)
      expect(harness.onRetryScheduled.mock.calls.map((call) => call[2])).toEqual([1, 2])

      expect(harness.request('repair', 'binding_closed')).toBe(recovery)
      expect(harness.request('ensure_ready', 'page_visible')).toBe(recovery)
      await vi.advanceTimersByTimeAsync(14_999)
      expect(harness.execute).toHaveBeenCalledTimes(2)

      await vi.advanceTimersByTimeAsync(1)
      await expect(recovery).resolves.toBeUndefined()
      expect(harness.execute).toHaveBeenCalledTimes(3)
      expect(harness.execute.mock.calls[2]?.[0]).toMatchObject({
        intent: 'repair',
        trigger: 'automatic_retry',
        attempt: 3,
      })
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('startNativeForegroundTargets', () => {
  it('starts every target synchronously, then retains failed endpoint identity', async () => {
    const failure = new Error('endpoint-a failed')
    const second = deferred<void>()
    const started: string[] = []
    const batch = startNativeForegroundTargets([
      { endpointId: 'endpoint-a', resume: async () => { started.push('endpoint-a'); throw failure } },
      { endpointId: 'endpoint-b', resume: async () => { started.push('endpoint-b'); await second.promise } },
    ])
    expect(started).toEqual(['endpoint-a', 'endpoint-b'])
    expect(batch.total).toBe(2)
    let settled = false
    void batch.settled.finally(() => { settled = true })

    await Promise.resolve()
    expect(settled).toBe(false)
    second.resolve()

    await expect(batch.settled).resolves.toEqual({
      total: 2,
      resumed: 1,
      failures: [{ endpointId: 'endpoint-a', failure }],
    })
  })

  it('reports all endpoint failures to per-endpoint diagnostics', async () => {
    const failureA = new Error('endpoint-a failed')
    const failureB = new Error('endpoint-b failed')

    const batch = startNativeForegroundTargets([
      { endpointId: 'endpoint-a', resume: async () => { throw failureA } },
      { endpointId: 'endpoint-b', resume: async () => { throw failureB } },
    ])
    await expect(batch.settled).resolves.toEqual({
      total: 2,
      resumed: 0,
      failures: [
        { endpointId: 'endpoint-a', failure: failureA },
        { endpointId: 'endpoint-b', failure: failureB },
      ],
    })
  })
})

describe('startNativeForegroundWork', () => {
  it('starts endpoint fences and lets the foreground caller resolve while demand reconciliation hangs', async () => {
    const resume = vi.fn(async () => undefined)
    const foregroundResume = async () => {
      const work = startNativeForegroundWork(
        () => new Promise<void>(() => {}),
        [{ endpointId: 'endpoint-a', resume }],
      )
      void work.demand.catch(() => undefined)
      void work.endpoints.settled
    }

    await expect(foregroundResume()).resolves.toBeUndefined()
    expect(resume).toHaveBeenCalledOnce()
  })

  it('keeps demand failure observable without suppressing endpoint recovery', async () => {
    const failure = new Error('demand attachment changed')
    const resume = vi.fn(async () => undefined)
    const work = startNativeForegroundWork(
      async () => { throw failure },
      [{ endpointId: 'endpoint-a', resume }],
    )

    await expect(work.demand).rejects.toBe(failure)
    await expect(work.endpoints.settled).resolves.toMatchObject({ resumed: 1, failures: [] })
    expect(resume).toHaveBeenCalledOnce()
  })
})

function recoveryHarness(options: { retryDelay?: () => number | null } = {}) {
  const coordinator = new NativeRecoveryCoordinator()
  const fence = new NativeGenerationRecoveryFence()
  const execute = vi.fn(async (_work: NativeRecoveryWork): Promise<void> => undefined)
  const onRetryScheduled = vi.fn()
  return {
    coordinator,
    execute,
    fence,
    onRetryScheduled,
    request(
      intent: 'ensure_ready' | 'repair',
      trigger: 'app_resume' | 'page_visible' | 'binding_closed' | 'manual_retry' = 'app_resume',
    ) {
      return coordinator.request({
        intent,
        trigger,
      }, {
        beginAttempt: () => fence.beginAttempt(),
        isCurrent: (attempt) => fence.isCurrent(attempt),
        execute,
        retryDelay: options.retryDelay
          ? (_failure, _retryAttempt, _work) => options.retryDelay!()
          : undefined,
        onRetryScheduled,
      })
    },
  }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}
