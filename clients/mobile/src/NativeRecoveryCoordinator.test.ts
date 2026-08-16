import { describe, expect, it, vi } from 'vitest'
import { NativeGenerationRecoveryFence } from './NativeGenerationRecoveryFence'
import { NativeRecoveryCoordinator, type NativeRecoveryWork } from './NativeRecoveryCoordinator'

describe('NativeRecoveryCoordinator', () => {
  it('coalesces repeated work into one active recovery', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => await gate.promise)

    const first = harness.request({ replaceBinding: true, reloadRegistry: false })
    const repeated = harness.request({ replaceBinding: true, reloadRegistry: false })

    expect(repeated).toBe(first)
    gate.resolve()
    await first
    expect(harness.execute).toHaveBeenCalledOnce()
  })

  it('drains one OR-upgraded request after invalidating the active attempt', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => await gate.promise)

    const recovery = harness.request({ replaceBinding: false, reloadRegistry: true })
    harness.request({ replaceBinding: true, reloadRegistry: false })
    harness.request({ replaceBinding: false, reloadRegistry: true })
    gate.resolve()
    await recovery

    expect(harness.execute).toHaveBeenCalledTimes(2)
    expect(harness.execute.mock.calls[1]?.[0]).toMatchObject({
      replaceBinding: true,
      reloadRegistry: true,
    })
  })

  it('queues the same work when the active attempt was invalidated in the background', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => await gate.promise)

    const recovery = harness.request({ replaceBinding: false, reloadRegistry: true })
    harness.fence.invalidate()
    harness.request({ replaceBinding: false, reloadRegistry: true })
    gate.resolve()
    await recovery

    expect(harness.execute).toHaveBeenCalledTimes(2)
    expect(harness.execute.mock.calls[1]?.[0]).toMatchObject({
      replaceBinding: false,
      reloadRegistry: true,
    })
  })

  it('continues with queued recovery after the superseded attempt fails', async () => {
    const harness = recoveryHarness()
    const gate = deferred<void>()
    harness.execute.mockImplementationOnce(async () => {
      await gate.promise
      throw new Error('stale bridge failed')
    })

    const recovery = harness.request({ replaceBinding: false, reloadRegistry: true })
    harness.request({ replaceBinding: true, reloadRegistry: false })
    gate.resolve()

    await expect(recovery).resolves.toBeUndefined()
    expect(harness.execute).toHaveBeenCalledTimes(2)
  })
})

function recoveryHarness() {
  const coordinator = new NativeRecoveryCoordinator()
  const fence = new NativeGenerationRecoveryFence()
  const execute = vi.fn(async (_work: NativeRecoveryWork): Promise<void> => undefined)
  return {
    execute,
    fence,
    request(request: { replaceBinding: boolean; reloadRegistry: boolean }) {
      return coordinator.request(request, {
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
