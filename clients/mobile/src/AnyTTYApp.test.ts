import { describe, expect, it, vi } from 'vitest'

vi.stubGlobal('self', globalThis)
const {
  NativeDeferredDisconnectFence,
  NativeDisconnectAllRequestProcessor,
  NativeEndpointStopRegistry,
  NativeTransferIntentCoordinator,
  NativeWorkspaceResumeIntentRegistry,
  runNativeEndpointStopCleanup,
  withNativeRecoveryTimeout,
} = await import('./AnyTTYApp')

describe('withNativeRecoveryTimeout', () => {
  it('aborts the step fence so a late operation cannot publish success', async () => {
    vi.useFakeTimers()
    const completion = deferred<void>()
    let stepSignal!: AbortSignal
    let lateSuccessPublished = false
    try {
      const recovery = withNativeRecoveryTimeout(async (signal) => {
        stepSignal = signal
        await completion.promise
        if (!signal.aborted) lateSuccessPublished = true
      }, 'Native binding replacement', undefined, 100)
      const outcome = expect(recovery).rejects.toMatchObject({
        message: 'Native binding replacement timed out',
        code: 'unavailable',
      })

      await vi.advanceTimersByTimeAsync(100)
      await outcome
      expect(stepSignal.aborted).toBe(true)

      completion.resolve()
      await Promise.resolve()
      expect(lateSuccessPublished).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('runNativeEndpointStopCleanup', () => {
  it('preserves an endpoint whose fresh intent already covers the delayed Stop event', async () => {
    const freshIntent = {}
    const disconnectA = vi.fn(async () => undefined)
    const disconnectB = vi.fn(async () => undefined)
    const protectedAtSuspend: string[] = []
    const endpointStops = new NativeEndpointStopRegistry(
      (intent, stopEpoch) => intent === freshIntent && stopEpoch === '7',
    )

    await runNativeEndpointStopCleanup(
      '7',
      [
        { machineId: 'machine-a', resumeIntent: freshIntent, isCurrent: () => true, adoptUserStop: disconnectA },
        { machineId: 'machine-b', resumeIntent: null, isCurrent: () => true, adoptUserStop: disconnectB },
      ],
      endpointStops,
      async (protectedMachineIds) => { protectedAtSuspend.push(...protectedMachineIds) },
      new NativeDeferredDisconnectFence(),
    )

    expect(protectedAtSuspend).toEqual(['machine-a'])
    expect(disconnectA).not.toHaveBeenCalled()
    expect(disconnectB).toHaveBeenCalledOnce()
  })

  it('latches an unprotected endpoint across manager replacement and later recreation', async () => {
    const freshIntent = {}
    const endpointStops = new NativeEndpointStopRegistry(
      (intent, stopEpoch) => intent === freshIntent && stopEpoch === '7',
    )
    const suspend = deferred<void>()
    let oldBIsCurrent = true
    const oldB = stoppedManager()
    const replacementB = stoppedManager()
    const lateB = stoppedManager()
    const protectedA = stoppedManager()

    const stopping = runNativeEndpointStopCleanup(
      '7',
      [
        { machineId: 'machine-a', resumeIntent: freshIntent, isCurrent: () => true, adoptUserStop: protectedA.adoptUserStop },
        { machineId: 'machine-b', resumeIntent: null, isCurrent: () => oldBIsCurrent, adoptUserStop: oldB.adoptUserStop },
      ],
      endpointStops,
      async () => await suspend.promise,
      new NativeDeferredDisconnectFence(),
    )

    expect(() => oldB.passiveGet()).toThrow(expect.objectContaining({ code: 'user_stopped' }))
    expect(endpointStops.isStopped('machine-a')).toBe(false)
    expect(endpointStops.isStopped('machine-b')).toBe(true)
    oldBIsCurrent = false
    endpointStops.adoptIfStopped('machine-b', replacementB.adoptUserStop)
    expect(() => replacementB.passiveGet()).toThrow(expect.objectContaining({ code: 'user_stopped' }))
    expect(protectedA.adoptUserStop).not.toHaveBeenCalled()

    suspend.resolve()
    await stopping
    endpointStops.adoptIfStopped('machine-b', lateB.adoptUserStop)
    expect(() => lateB.passiveGet()).toThrow(expect.objectContaining({ code: 'user_stopped' }))
  })

  it('protects current canonical demand while a replayed Stop cleans older owners', async () => {
    const endpointStops = new NativeEndpointStopRegistry(() => false)
    const managerA = stoppedManager()
    const managerB = stoppedManager()
    const protectedAtSuspend: string[] = []
    endpointStops.latchStop('7', [{ machineId: 'machine-a', resumeIntent: null }])

    await runNativeEndpointStopCleanup(
      '7',
      [
        { machineId: 'machine-a', resumeIntent: null, isCurrent: () => true, adoptUserStop: managerA.adoptUserStop },
        { machineId: 'machine-b', resumeIntent: null, isCurrent: () => true, adoptUserStop: managerB.adoptUserStop },
      ],
      endpointStops,
      async (protectedMachineIds) => { protectedAtSuspend.push(...protectedMachineIds) },
      new NativeDeferredDisconnectFence(),
      new Set(['machine-a']),
    )

    expect(protectedAtSuspend).toEqual(['machine-a'])
    expect(managerA.adoptUserStop).not.toHaveBeenCalled()
    expect(endpointStops.isStopped('machine-a')).toBe(false)
    expect(managerB.adoptUserStop).toHaveBeenCalledOnce()
    expect(endpointStops.isStopped('machine-b')).toBe(true)
  })

  it('keeps an endpoint latch until an accepted intent covers that Stop epoch', () => {
    const coveringIntent = {}
    const endpointStops = new NativeEndpointStopRegistry(
      (intent, stopEpoch) => intent === coveringIntent && stopEpoch === '9',
    )
    endpointStops.latchStop('9', [{ machineId: 'machine-b', resumeIntent: null }])

    endpointStops.acceptUserResume('machine-b', {})
    expect(endpointStops.isStopped('machine-b')).toBe(true)

    endpointStops.acceptUserResume('machine-b', coveringIntent)
    expect(endpointStops.isStopped('machine-b')).toBe(false)
  })

  it('lets a pending action covering a newer Stop supersede an older endpoint latch', () => {
    const intent = {}
    const endpointStops = new NativeEndpointStopRegistry(
      (candidate, stopEpoch) => candidate === intent && BigInt(stopEpoch) <= 2n,
    )
    endpointStops.latchStop('1', [{ machineId: 'machine-b', resumeIntent: null }])

    const protectedMachineIds = endpointStops.latchStop('2', [{ machineId: 'machine-b', resumeIntent: intent }])

    expect([...protectedMachineIds]).toEqual(['machine-b'])
    expect(endpointStops.isStopped('machine-b')).toBe(false)
  })

  it('retains a newer endpoint latch when an intent only covers an older Stop', () => {
    const intent = {}
    const endpointStops = new NativeEndpointStopRegistry(
      (candidate, stopEpoch) => candidate === intent && stopEpoch === '1',
    )
    endpointStops.latchStop('2', [{ machineId: 'machine-b', resumeIntent: null }])

    const protectedMachineIds = endpointStops.latchStop('1', [{ machineId: 'machine-b', resumeIntent: intent }])

    expect([...protectedMachineIds]).toEqual([])
    expect(endpointStops.isStopped('machine-b')).toBe(true)
  })

  it('protects an eager workspace intent while the lazy manager is still pending', async () => {
    const resumeIntent = {}
    const markFreshConnectionIntent = vi.fn()
    const workspaceIntents = new NativeWorkspaceResumeIntentRegistry(markFreshConnectionIntent)
    const endpointStops = new NativeEndpointStopRegistry(
      (intent, stopEpoch) => intent === resumeIntent && stopEpoch === '11',
    )
    const protectedAtSuspend: string[] = []

    workspaceIntents.register('machine-a', resumeIntent)
    await runNativeEndpointStopCleanup(
      '11',
      workspaceIntents.entries().map((pending) => ({
        ...pending,
        isCurrent: () => workspaceIntents.isPending(pending.machineId, pending.resumeIntent),
        adoptUserStop: async () => undefined,
      })),
      endpointStops,
      async (protectedMachineIds) => { protectedAtSuspend.push(...protectedMachineIds) },
      new NativeDeferredDisconnectFence(),
    )

    expect(markFreshConnectionIntent).toHaveBeenCalledOnce()
    expect(markFreshConnectionIntent).toHaveBeenCalledWith('machine-a')
    expect(protectedAtSuspend).toEqual(['machine-a'])
    const beginUserConnectionIntent = vi.fn()
    workspaceIntents.attachManager('machine-a', { beginUserConnectionIntent })
    expect(beginUserConnectionIntent).toHaveBeenCalledWith(resumeIntent)
    workspaceIntents.consume('machine-a', resumeIntent)
    expect(workspaceIntents.entries()).toEqual([])
  })

  it('does not mint a second generation when the same consumed intent is reattached', () => {
    const markFreshConnectionIntent = vi.fn()
    const workspaceIntents = new NativeWorkspaceResumeIntentRegistry(markFreshConnectionIntent)
    const intent = {}

    workspaceIntents.register('machine-a', intent)
    workspaceIntents.consume('machine-a', intent)
    workspaceIntents.register('machine-a', intent)

    expect(markFreshConnectionIntent).toHaveBeenCalledOnce()
  })

  it('adopts an old endpoint latch before attaching a pending intent covering the new Stop', async () => {
    const intent = {}
    const workspaceIntents = new NativeWorkspaceResumeIntentRegistry(vi.fn())
    const endpointStops = new NativeEndpointStopRegistry(
      (candidate, stopEpoch) => candidate === intent && stopEpoch === '2',
    )
    let latestIntent: object | null = null
    let stopped = false
    const manager = {
      beginUserConnectionIntent(candidate: object) {
        latestIntent = candidate
        return candidate
      },
      adoptUserStop: vi.fn(async () => {
        stopped = true
        latestIntent = null
      }),
    }
    endpointStops.latchStop('1', [{ machineId: 'machine-a', resumeIntent: null }])
    workspaceIntents.register('machine-a', intent)

    endpointStops.adoptIfStopped('machine-a', manager.adoptUserStop)
    workspaceIntents.attachManager('machine-a', manager)
    const protectedMachineIds = endpointStops.latchStop('2', [{
      machineId: 'machine-a',
      resumeIntent: latestIntent ?? workspaceIntents.currentIntent('machine-a'),
    }])

    expect(stopped).toBe(true)
    expect([...protectedMachineIds]).toEqual(['machine-a'])
    expect(endpointStops.isStopped('machine-a')).toBe(false)
  })
})

describe('NativeDeferredDisconnectFence', () => {
  it('drops a deferred stop continuation after fresh connection intent', async () => {
    const fence = new NativeDeferredDisconnectFence()
    const cleanup = deferred<void>()
    const disconnect = vi.fn(async (_endpointId: string) => undefined)

    const stopped = fence.run(['machine-a'], () => cleanup.promise, disconnect)
    fence.markFreshIntent('machine-a')
    cleanup.resolve()

    await expect(stopped).resolves.toBe(0)
    expect(disconnect).not.toHaveBeenCalled()
  })

  it('disconnects unaffected endpoints when one endpoint receives fresh intent', async () => {
    const fence = new NativeDeferredDisconnectFence()
    const cleanup = deferred<void>()
    const disconnect = vi.fn(async (_endpointId: string) => undefined)

    const stopped = fence.run(['machine-a', 'machine-b'], () => cleanup.promise, disconnect)
    fence.markFreshIntent('machine-a')
    cleanup.resolve()

    await expect(stopped).resolves.toBe(1)
    expect(disconnect).toHaveBeenCalledOnce()
    expect(disconnect).toHaveBeenCalledWith('machine-b')
  })

  it('waits for every manager cleanup and rejects when any disconnect fails', async () => {
    const fence = new NativeDeferredDisconnectFence()
    const secondCleanup = deferred<void>()
    const disconnect = vi.fn(async (endpointId: string) => {
      if (endpointId === 'machine-a') throw new Error('manager disconnect failed')
      await secondCleanup.promise
    })

    const stopping = fence.run(['machine-a', 'machine-b'], async () => undefined, disconnect)
    const outcome = stopping.then(() => 'fulfilled' as const, (failure: unknown) => failure)
    await vi.waitFor(() => expect(disconnect).toHaveBeenCalledTimes(2))
    let settled = false
    void outcome.then(() => { settled = true })
    await Promise.resolve()
    expect(settled).toBe(false)

    secondCleanup.resolve()
    await expect(outcome).resolves.toMatchObject({ message: 'manager disconnect failed' })
  })
})

describe('NativeDisconnectAllRequestProcessor', () => {
  it('commits an epoch only after failed cleanup is retried successfully', async () => {
    vi.useFakeTimers()
    try {
      const canonicalize = vi.fn(async () => cleanupRequest('7'))
      const cleanupStarted = deferred<void>()
      const cleanup = vi.fn()
        .mockImplementationOnce(async () => {
          cleanupStarted.resolve()
          throw new Error('renderer bridge is rebuilding')
        })
        .mockResolvedValueOnce(undefined)
      const commit = vi.fn(async () => undefined)
      const processor = new NativeDisconnectAllRequestProcessor(canonicalize, commit)
      processor.setCleanup(cleanup)

      processor.enqueue({ stopEpoch: '7', stopped: true })
      await cleanupStarted.promise
      expect(commit).not.toHaveBeenCalled()

      await vi.advanceTimersByTimeAsync(250)
      await vi.waitFor(() => expect(cleanup).toHaveBeenCalledTimes(2))
      expect(commit).toHaveBeenCalledOnce()
      expect(commit).toHaveBeenCalledWith('7')
    } finally {
      vi.useRealTimers()
    }
  })

  it('lets a newer Stop supersede an older epoch waiting for retry', async () => {
    vi.useFakeTimers()
    try {
      const firstCleanup = deferred<void>()
      const canonicalize = vi.fn(async (event: { stopEpoch: string }) => cleanupRequest(event.stopEpoch))
      const cleanup = vi.fn(async (request: { stopEpoch: string }) => {
        if (request.stopEpoch === '7') {
          firstCleanup.resolve()
          throw new Error('cleanup failed')
        }
      })
      const commit = vi.fn(async () => undefined)
      const processor = new NativeDisconnectAllRequestProcessor(canonicalize, commit)
      processor.setCleanup(cleanup)

      processor.enqueue({ stopEpoch: '7', stopped: true })
      await firstCleanup.promise
      processor.enqueue({ stopEpoch: '9', stopped: true })
      await vi.advanceTimersByTimeAsync(250)
      await vi.waitFor(() => expect(commit).toHaveBeenCalledWith('9'))

      expect(canonicalize.mock.calls.map(([event]) => event.stopEpoch)).toEqual(['7', '9'])
      expect(commit).not.toHaveBeenCalledWith('7')
    } finally {
      vi.useRealTimers()
    }
  })

  it('pauses retries without a mounted cleanup target and resumes when reattached', async () => {
    vi.useFakeTimers()
    try {
      const firstCleanup = deferred<void>()
      const cleanup = vi.fn(async () => {
        firstCleanup.resolve()
        throw new Error('cleanup failed')
      })
      const processor = new NativeDisconnectAllRequestProcessor(
        async () => cleanupRequest('3'),
        vi.fn(async () => undefined),
      )
      processor.setCleanup(cleanup)
      processor.enqueue({ stopEpoch: '3', stopped: true })
      await firstCleanup.promise

      processor.setCleanup(null)
      await vi.advanceTimersByTimeAsync(30_000)
      expect(cleanup).toHaveBeenCalledOnce()

      const replacementCleanup = vi.fn(async () => undefined)
      processor.setCleanup(replacementCleanup)
      await vi.waitFor(() => expect(replacementCleanup).toHaveBeenCalledOnce())
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not locally commit until the native acknowledgement succeeds', async () => {
    vi.useFakeTimers()
    try {
      const cleanup = vi.fn(async () => undefined)
      const nativeAcknowledge = vi.fn()
        .mockRejectedValueOnce(new Error('newer Stop won'))
        .mockResolvedValueOnce(undefined)
      const localCommit = vi.fn()
      const commit = vi.fn(async (stopEpoch: string) => {
        await nativeAcknowledge(stopEpoch)
        localCommit(stopEpoch)
      })
      const processor = new NativeDisconnectAllRequestProcessor(
        async () => cleanupRequest('7'),
        commit,
      )
      processor.setCleanup(cleanup)

      processor.enqueue({ stopEpoch: '7', stopped: true })
      await vi.waitFor(() => expect(nativeAcknowledge).toHaveBeenCalledOnce())
      expect(localCommit).not.toHaveBeenCalled()

      await vi.advanceTimersByTimeAsync(250)
      await vi.waitFor(() => expect(localCommit).toHaveBeenCalledWith('7'))
      expect(cleanup).toHaveBeenCalledTimes(2)
      expect(nativeAcknowledge).toHaveBeenCalledTimes(2)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('NativeTransferIntentCoordinator', () => {
  it('does not commit an async transfer continuation rejected by the exact native intent', async () => {
    const intent = {}
    const confirm = vi.fn(async () => {
      throw Object.assign(new Error('Stopped by user'), { code: 'user_stopped' })
    })
    const finish = vi.fn()
    const coordinator = new NativeTransferIntentCoordinator(() => intent, confirm, finish)
    const commit = vi.fn()

    const action = coordinator.begin('machine-a')
    await expect(coordinator.run('machine-a', action, commit)).rejects.toMatchObject({ code: 'user_stopped' })

    expect(confirm).toHaveBeenCalledWith('machine-a', intent)
    expect(commit).not.toHaveBeenCalled()
    expect(finish).toHaveBeenCalledWith('machine-a', intent)
  })

  it('reattaches the same confirmed token before committing after manager replacement', async () => {
    const intent = {}
    const order: string[] = []
    const coordinator = new NativeTransferIntentCoordinator(
      () => intent,
      async () => { order.push('confirm') },
      () => { order.push('finish') },
    )

    await coordinator.run('machine-a', coordinator.begin('machine-a'), () => { order.push('commit') })

    expect(order).toEqual(['confirm', 'commit', 'finish'])
  })
})

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((accept, decline) => {
    resolve = accept
    reject = decline
  })
  return { promise, resolve, reject }
}

function cleanupRequest(stopEpoch: string, protectedEndpointIds: string[] = []) {
  return { stopEpoch, protectedEndpointIds }
}

function stoppedManager() {
  let stopped = false
  return {
    adoptUserStop: vi.fn(async () => { stopped = true }),
    passiveGet() {
      if (stopped) throw Object.assign(new Error('Stopped by user'), { code: 'user_stopped' })
      return 'connected'
    },
  }
}
