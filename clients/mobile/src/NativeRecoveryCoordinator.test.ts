import { describe, expect, it, vi } from 'vitest'
import { NativeGenerationRecoveryFence } from './NativeGenerationRecoveryFence'
import {
  NativeRecoveryCoordinator,
  resumeNativeForegroundTargets,
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
})

describe('resumeNativeForegroundTargets', () => {
  it('waits for every target and rejects a single endpoint failure unchanged', async () => {
    const failure = new Error('endpoint-a failed')
    const second = deferred<void>()
    const resumed = resumeNativeForegroundTargets([
      { endpointId: 'endpoint-a', resume: async () => { throw failure } },
      { endpointId: 'endpoint-b', resume: async () => await second.promise },
    ])
    let settled = false
    void resumed.finally(() => { settled = true }).catch(() => undefined)

    await Promise.resolve()
    expect(settled).toBe(false)
    second.resolve()

    await expect(resumed).rejects.toBe(failure)
  })

  it('aggregates multiple endpoint failures with endpoint context', async () => {
    const failureA = new Error('endpoint-a failed')
    const failureB = new Error('endpoint-b failed')

    await expect(resumeNativeForegroundTargets([
      { endpointId: 'endpoint-a', resume: async () => { throw failureA } },
      { endpointId: 'endpoint-b', resume: async () => { throw failureB } },
    ])).rejects.toMatchObject({
      message: 'Native foreground resume failed for endpoints: endpoint-a, endpoint-b',
      errors: [failureA, failureB],
    })
  })
})

function recoveryHarness() {
  const coordinator = new NativeRecoveryCoordinator()
  const fence = new NativeGenerationRecoveryFence()
  const execute = vi.fn(async (_work: NativeRecoveryWork): Promise<void> => undefined)
  return {
    execute,
    fence,
    request(intent: 'ensure_ready' | 'repair', trigger: 'app_resume' | 'page_visible' | 'binding_closed' = 'app_resume') {
      return coordinator.request({
        intent,
        trigger,
      }, {
        beginAttempt: () => fence.beginAttempt(),
        isCurrent: (attempt) => fence.isCurrent(attempt),
        execute,
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
