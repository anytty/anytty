import { describe, expect, it, vi } from 'vitest'
import {
  NativeSessionDemandCoordinator,
  type NativeSessionDemandInput,
  type NativeSessionDemandLease,
  type NativeSessionDemandResumeResult,
} from './NativeSessionDemand'

const demandOwner = Symbol('test-demand-owner')

function createDemandHarness(behavior: (input: NativeSessionDemandInput) => Promise<void> = async () => {}) {
  let demandRevision = 0
  let stopEpoch = 0
  let stopped = false
  let nativeEndpointIds: string[] = []
  const resumeIntents = new Map<string, { stopEpoch: number; accepted: boolean }>()
  const attachmentId = 'renderer-a'
  const getDemandLease = vi.fn(async (): Promise<NativeSessionDemandLease> => ({
    attachmentId,
    demandRevision: String(demandRevision),
    stopEpoch: String(stopEpoch),
    endpointIds: [...nativeEndpointIds],
    stopped,
  }))
  const replaceDemand = vi.fn(async (input: NativeSessionDemandInput): Promise<NativeSessionDemandLease> => {
    await behavior(input)
    if (input.attachmentId !== attachmentId || input.baseDemandRevision !== String(demandRevision)) {
      throw new Error('stale native demand lease')
    }
    if (stopped && input.endpointIds.length > 0) {
      throw Object.assign(new Error('native demand was stopped by the user'), {
        code: 'user_stopped',
        retryable: false,
      })
    }
    demandRevision += 1
    nativeEndpointIds = [...input.endpointIds]
    return {
      attachmentId,
      demandRevision: String(demandRevision),
      stopEpoch: String(stopEpoch),
      endpointIds: [...nativeEndpointIds],
      stopped,
    }
  })
  const resumeNatively = async (input: { intentId: string; baseStopEpoch: string }): Promise<NativeSessionDemandResumeResult> => {
    const existing = resumeIntents.get(input.intentId)
    if (existing !== undefined) {
      return {
        attachmentId,
        demandRevision: String(demandRevision),
        stopEpoch: String(stopEpoch),
        endpointIds: [...nativeEndpointIds],
        stopped,
        outcome: existing.accepted && existing.stopEpoch === stopEpoch ? 'resumed' : 'stopped',
      }
    }
    const requestedStopEpoch = Number(input.baseStopEpoch)
    if (requestedStopEpoch !== stopEpoch) {
      resumeIntents.set(input.intentId, { stopEpoch: requestedStopEpoch, accepted: false })
      return {
        attachmentId,
        demandRevision: String(demandRevision),
        stopEpoch: String(stopEpoch),
        endpointIds: [...nativeEndpointIds],
        stopped,
        outcome: 'stopped',
      }
    }
    if (stopped) {
      demandRevision += 1
      stopped = false
    }
    resumeIntents.set(input.intentId, { stopEpoch, accepted: true })
    return {
      attachmentId,
      demandRevision: String(demandRevision),
      stopEpoch: String(stopEpoch),
      endpointIds: [...nativeEndpointIds],
      stopped,
      outcome: 'resumed',
    }
  }
  const resumeDemand = vi.fn(resumeNatively)
  const demand = new NativeSessionDemandCoordinator(replaceDemand, getDemandLease, resumeDemand)
  return {
    demand,
    getDemandLease,
    replaceDemand,
    resumeDemand,
    resumeNatively,
    stopNatively() {
      demandRevision += 1
      stopEpoch = demandRevision
      stopped = true
      nativeEndpointIds = []
    },
    notifyStop: async () => {
      const request = await demand.handleDisconnectAllRequested({ stopEpoch: String(stopEpoch), stopped: true })
      if (request !== null) demand.commitDisconnectAllCleanup(request.stopEpoch)
      return request
    },
    currentLease: (): NativeSessionDemandLease => ({
      attachmentId,
      demandRevision: String(demandRevision),
      stopEpoch: String(stopEpoch),
      endpointIds: [...nativeEndpointIds],
      stopped,
    }),
    nativeEndpointIds: () => [...nativeEndpointIds],
  }
}

describe('NativeSessionDemandCoordinator', () => {
  it('replaces native ownership with the complete sorted renderer demand', async () => {
    const { demand, getDemandLease, replaceDemand } = createDemandHarness()

    await demand.setActive('machine-b', true, demandOwner)
    await demand.setActive('machine-a', true, demandOwner)
    await demand.setActive('machine-b', false, demandOwner)

    expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
      ['machine-b'],
      ['machine-a', 'machine-b'],
      ['machine-a'],
    ])
    expect(replaceDemand.mock.calls.map(([input]) => input.baseDemandRevision)).toEqual(['0', '1', '2'])
    expect(getDemandLease).toHaveBeenCalledTimes(1)
  })

  it('can reconcile an empty renderer after the previous WebView disappeared', async () => {
    const { demand, replaceDemand } = createDemandHarness()

    await demand.reconcileRenderer()

    expect(replaceDemand).toHaveBeenCalledWith(expect.objectContaining({ endpointIds: [] }))
  })

  it('restores visible workspace demand before the lazy manager mounts and releases it on Back', async () => {
    const { demand, replaceDemand, nativeEndpointIds } = createDemandHarness()
    const managerOwner = Symbol('restored-manager')

    await demand.restoreRenderer(['machine-a'])
    await demand.setActive('machine-a', true, managerOwner)
    await demand.setWorkspaceEndpoint(null)
    expect(nativeEndpointIds()).toEqual(['machine-a'])

    await demand.setActive('machine-a', false, managerOwner)
    expect(nativeEndpointIds()).toEqual([])
    expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
      ['machine-a'],
      [],
    ])
  })

  it('does not mistake a historical Stop epoch for a currently latched Stop gate', async () => {
    const harness = createDemandHarness()
    harness.stopNatively()
    await harness.notifyStop()
    await harness.demand.resumeForUserIntent({})
    expect(harness.currentLease()).toMatchObject({ stopEpoch: '1', stopped: false })

    const freshRenderer = new NativeSessionDemandCoordinator(
      harness.replaceDemand,
      harness.getDemandLease,
      harness.resumeDemand,
    )
    await freshRenderer.restoreRenderer(['machine-a'])

    expect(harness.nativeEndpointIds()).toEqual(['machine-a'])
  })

  it('protects canonical demand when an unacknowledged Stop replays after renderer loss', async () => {
    const harness = createDemandHarness()
    harness.stopNatively()
    const event = { stopEpoch: harness.currentLease().stopEpoch, stopped: true }
    await harness.demand.resumeForUserIntent({})
    await harness.demand.setActive('machine-a', true, demandOwner)

    const replacementRenderer = new NativeSessionDemandCoordinator(
      harness.replaceDemand,
      harness.getDemandLease,
      harness.resumeDemand,
    )

    await expect(replacementRenderer.handleDisconnectAllRequested(event)).resolves.toEqual(
      cleanupRequest(event.stopEpoch, ['machine-a']),
    )
    expect(harness.currentLease()).toMatchObject({
      stopped: false,
      endpointIds: ['machine-a'],
    })
  })

  it('keeps renderer restoration behind the current native Stop gate', async () => {
    const { demand, resumeDemand, stopNatively } = createDemandHarness()
    stopNatively()

    await expect(demand.restoreRenderer(['machine-a'])).rejects.toMatchObject({
      code: 'user_stopped',
      retryable: false,
    })
    await expect(demand.setActive('machine-a', true, demandOwner)).rejects.toMatchObject({
      code: 'user_stopped',
    })
    expect(resumeDemand).not.toHaveBeenCalled()
  })

  it('tracks independent manager owners so a stale false cannot clear newer demand', async () => {
    const { demand, replaceDemand } = createDemandHarness()
    const oldManager = Symbol('old-manager')
    const newManager = Symbol('new-manager')

    await demand.setActive('machine-a', true, oldManager)
    await demand.setActive('machine-a', true, newManager)
    await demand.setActive('machine-a', false, oldManager)

    expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
      ['machine-a'],
    ])

    await demand.setActive('machine-a', false, newManager)
    expect(replaceDemand).toHaveBeenLastCalledWith(expect.objectContaining({ endpointIds: [] }))
  })

  it('retains the latest local intent when native rejects its replacement', async () => {
    const behavior = vi.fn()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error('stale revision'))
      .mockResolvedValueOnce(undefined)
    const { demand, replaceDemand } = createDemandHarness(behavior)

    await demand.setActive('machine-a', true, demandOwner)
    await expect(demand.setActive('machine-a', false, demandOwner)).rejects.toThrow('stale revision')
    await demand.reconcileRenderer()

    expect(replaceDemand).toHaveBeenLastCalledWith(expect.objectContaining({ endpointIds: [] }))
  })

  it('does not let an older failed update roll back a newer full snapshot', async () => {
    let rejectFirst!: (failure: Error) => void
    const first = new Promise<void>((_resolve, reject) => { rejectFirst = reject })
    const behavior = vi.fn()
      .mockReturnValueOnce(first)
      .mockResolvedValueOnce(undefined)
    const { demand, replaceDemand } = createDemandHarness(behavior)

    const firstUpdate = demand.setActive('machine-a', true, demandOwner)
    await vi.waitFor(() => expect(replaceDemand).toHaveBeenCalledTimes(1))
    const secondUpdate = demand.setActive('machine-b', true, demandOwner)
    rejectFirst(new Error('old attachment'))

    await expect(firstUpdate).rejects.toThrow('old attachment')
    await secondUpdate
    expect(replaceDemand).toHaveBeenLastCalledWith(expect.objectContaining({
      endpointIds: ['machine-a', 'machine-b'],
    }))
  })

  it('retries a dirty full snapshot without another renderer action', async () => {
    vi.useFakeTimers()
    const behavior = vi.fn()
      .mockRejectedValueOnce(new Error('native attachment unavailable'))
      .mockResolvedValueOnce(undefined)
    const { demand, replaceDemand } = createDemandHarness(behavior)
    try {
      await expect(demand.setActive('machine-a', true, demandOwner)).rejects.toThrow('native attachment unavailable')
      expect(replaceDemand).toHaveBeenCalledTimes(1)

      await vi.advanceTimersByTimeAsync(249)
      expect(replaceDemand).toHaveBeenCalledTimes(1)
      await vi.advanceTimersByTimeAsync(1)

      expect(replaceDemand).toHaveBeenCalledTimes(2)
      expect(replaceDemand).toHaveBeenLastCalledWith(expect.objectContaining({ endpointIds: ['machine-a'] }))
    } finally {
      vi.useRealTimers()
    }
  })

  it('retries a dirty full snapshot after the native call times out', async () => {
    vi.useFakeTimers()
    const behavior = vi.fn()
      .mockReturnValueOnce(new Promise<void>(() => {}))
      .mockResolvedValueOnce(undefined)
    const { demand, replaceDemand } = createDemandHarness(behavior)
    try {
      const update = demand.setActive('machine-a', true, demandOwner)
      await Promise.resolve()
      await vi.advanceTimersByTimeAsync(5_000)
      await expect(update).rejects.toMatchObject({ code: 'demand_sync_pending' })

      await vi.advanceTimersByTimeAsync(250)

      expect(replaceDemand).toHaveBeenCalledTimes(2)
      expect(replaceDemand).toHaveBeenLastCalledWith(expect.objectContaining({ endpointIds: ['machine-a'] }))
    } finally {
      vi.useRealTimers()
    }
  })

  it('cancels a stale retry when a newer full snapshot succeeds', async () => {
    vi.useFakeTimers()
    const behavior = vi.fn()
      .mockRejectedValueOnce(new Error('native attachment unavailable'))
      .mockResolvedValue(undefined)
    const { demand, replaceDemand } = createDemandHarness(behavior)
    try {
      await expect(demand.setActive('machine-a', true, demandOwner)).rejects.toThrow('native attachment unavailable')
      await demand.setActive('machine-b', true, demandOwner)
      await vi.advanceTimersByTimeAsync(15_000)

      expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
        ['machine-a'],
        ['machine-a', 'machine-b'],
      ])
    } finally {
      vi.useRealTimers()
    }
  })

  it('fences a stale retry that was queued before a newer full snapshot', async () => {
    vi.useFakeTimers()
    const behavior = vi.fn()
      .mockRejectedValueOnce(new Error('native attachment unavailable'))
      .mockResolvedValue(undefined)
    const { demand, replaceDemand } = createDemandHarness(behavior)
    try {
      await expect(demand.setActive('machine-a', true, demandOwner)).rejects.toThrow('native attachment unavailable')
      vi.advanceTimersByTime(250)

      await demand.setActive('machine-b', true, demandOwner)

      expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
        ['machine-a'],
        ['machine-a', 'machine-b'],
      ])
    } finally {
      vi.useRealTimers()
    }
  })

  it('cancels the old generation retry when the user stops all demand', async () => {
    vi.useFakeTimers()
    const behavior = vi.fn()
      .mockRejectedValueOnce(new Error('native attachment unavailable'))
      .mockResolvedValue(undefined)
    const { demand, replaceDemand, stopNatively, notifyStop } = createDemandHarness(behavior)
    try {
      await expect(demand.setActive('machine-a', true, demandOwner)).rejects.toThrow('native attachment unavailable')
      stopNatively()
      await notifyStop()
      await vi.advanceTimersByTimeAsync(15_000)

      expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
        ['machine-a'],
      ])
    } finally {
      vi.useRealTimers()
    }
  })

  it('cannot reacquire a fresh lease and resurrect demand before the native stop event is handled', async () => {
    vi.useFakeTimers()
    const { demand, replaceDemand, stopNatively, notifyStop, nativeEndpointIds } = createDemandHarness()
    try {
      await demand.setActive('machine-a', true, demandOwner)
      stopNatively()

      await expect(demand.reconcileRenderer()).rejects.toThrow('stale native demand lease')
      await vi.advanceTimersByTimeAsync(250)
      await vi.advanceTimersByTimeAsync(500)

      expect(nativeEndpointIds()).toEqual([])
      expect(replaceDemand.mock.calls.filter(([input]) => input.endpointIds.length > 0)).toHaveLength(2)

      await notifyStop()
      const callsAfterClear = replaceDemand.mock.calls.length
      await vi.advanceTimersByTimeAsync(30_000)
      expect(replaceDemand).toHaveBeenCalledTimes(callsAfterClear)
      expect(nativeEndpointIds()).toEqual([])
    } finally {
      vi.useRealTimers()
    }
  })

  it('requires an explicit user-intent resume before accepting nonempty demand after stop', async () => {
    const { demand, resumeDemand, stopNatively, notifyStop, nativeEndpointIds } = createDemandHarness()

    await demand.setActive('machine-a', true, demandOwner)
    stopNatively()
    await notifyStop()

    await expect(demand.setActive('machine-a', true, demandOwner)).rejects.toMatchObject({
      code: 'user_stopped',
      retryable: false,
    })
    await demand.resumeForUserIntent({})
    await demand.setActive('machine-a', true, demandOwner)

    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(nativeEndpointIds()).toEqual(['machine-a'])
  })

  it('keeps a Stop epoch retryable until cleanup commits it', async () => {
    const { demand, stopNatively, currentLease } = createDemandHarness()
    stopNatively()
    const event = { stopEpoch: currentLease().stopEpoch, stopped: true }

    await demand.reconcileRenderer()
    await expect(demand.handleDisconnectAllRequested(event)).resolves.toEqual(cleanupRequest(event.stopEpoch))
    await expect(demand.handleDisconnectAllRequested(event)).resolves.toEqual(cleanupRequest(event.stopEpoch))
    demand.commitDisconnectAllCleanup(event.stopEpoch)
    const freshIntent = demand.createResumeIntent()
    await expect(demand.handleDisconnectAllRequested(event)).resolves.toBeNull()
    await expect(demand.resumeForUserIntent(freshIntent)).resolves.toBeUndefined()
  })

  it('ignores an old stopped lease response after one user action already resumed that epoch', async () => {
    const { demand, getDemandLease, stopNatively, currentLease, nativeEndpointIds } = createDemandHarness()
    const delayedEventLease = deferred<NativeSessionDemandLease>()
    stopNatively()
    const stoppedLease = currentLease()
    getDemandLease.mockImplementationOnce(() => delayedEventLease.promise)

    const notification = demand.handleDisconnectAllRequested({
      stopEpoch: stoppedLease.stopEpoch,
      stopped: true,
    })
    await vi.waitFor(() => expect(getDemandLease).toHaveBeenCalledOnce())

    await demand.resumeForUserIntent(demand.createResumeIntent())
    delayedEventLease.resolve(stoppedLease)

    await expect(notification).resolves.toEqual(cleanupRequest(stoppedLease.stopEpoch))
    await expect(demand.setActive('machine-a', true, demandOwner)).resolves.toBeUndefined()
    expect(nativeEndpointIds()).toEqual(['machine-a'])
  })

  it('waits for an in-flight user resume before resolving a retained Stop notification', async () => {
    const { demand, resumeDemand, resumeNatively, stopNatively, currentLease } = createDemandHarness()
    const releaseResume = deferred<void>()
    stopNatively()
    const event = { stopEpoch: currentLease().stopEpoch, stopped: true }
    resumeDemand.mockImplementationOnce(async (input) => {
      const accepted = await resumeNatively(input)
      await releaseResume.promise
      return accepted
    })

    const resume = demand.resumeForUserIntent(demand.createResumeIntent())
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())
    const notification = demand.handleDisconnectAllRequested(event)
    releaseResume.resolve()

    await expect(resume).resolves.toBeUndefined()
    await expect(notification).resolves.toEqual(cleanupRequest(event.stopEpoch))
  })

  it('restores only the fresh endpoint intent after refreshing a native stop anchor', async () => {
    const { demand, resumeDemand, stopNatively, notifyStop, nativeEndpointIds } = createDemandHarness()
    const oldA = Symbol('old-a')
    const oldB = Symbol('old-b')
    const freshA = Symbol('fresh-a')

    await demand.setActive('machine-a', true, oldA)
    await demand.setActive('machine-b', true, oldB)
    stopNatively()
    await notifyStop()

    await demand.resumeForUserIntent({})
    expect(nativeEndpointIds()).toEqual([])
    await demand.setActive('machine-a', true, freshA)

    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(nativeEndpointIds()).toEqual(['machine-a'])
  })

  it('coalesces retries of the same intent behind one refreshed stop anchor', async () => {
    const { demand, getDemandLease, resumeDemand, stopNatively, notifyStop } = createDemandHarness()
    stopNatively()
    await notifyStop()
    const intent = {}

    const resumeA = demand.resumeForUserIntent(intent)
    const resumeB = demand.resumeForUserIntent(intent)

    expect(resumeB).toBe(resumeA)
    await Promise.all([resumeA, resumeB])
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(resumeDemand.mock.calls[0]?.[0].intentId).toBeTruthy()
    expect(getDemandLease).toHaveBeenCalledTimes(2)
  })

  it('uses a delayed initial native stop baseline so the first post-reload click resumes', async () => {
    const { demand, getDemandLease, resumeDemand, stopNatively, currentLease } = createDemandHarness()
    const initialLease = deferred<NativeSessionDemandLease>()
    stopNatively()
    getDemandLease.mockImplementationOnce(() => initialLease.promise)

    const intent = demand.createResumeIntent()
    const resumed = demand.resumeForUserIntent(intent)
    await Promise.resolve()
    expect(resumeDemand).not.toHaveBeenCalled()

    initialLease.resolve(currentLease())
    await expect(resumed).resolves.toBeUndefined()

    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(resumeDemand.mock.calls[0]?.[0].baseStopEpoch).toBe(currentLease().stopEpoch)
  })

  it('accepts concurrent post-Stop intents anchored to the same unseen epoch', async () => {
    const { demand, getDemandLease, resumeDemand, stopNatively, currentLease } = createDemandHarness()
    const anchorA = deferred<NativeSessionDemandLease>()
    const anchorB = deferred<NativeSessionDemandLease>()
    stopNatively()
    const stoppedLease = currentLease()
    getDemandLease
      .mockImplementationOnce(() => anchorA.promise)
      .mockImplementationOnce(() => anchorB.promise)

    const intentA = demand.createResumeIntent()
    const intentB = demand.createResumeIntent()
    const resumeA = demand.resumeForUserIntent(intentA)
    const resumeB = demand.resumeForUserIntent(intentB)
    await vi.waitFor(() => expect(getDemandLease).toHaveBeenCalledTimes(2))

    anchorA.resolve(stoppedLease)
    await expect(resumeA).resolves.toBeUndefined()
    anchorB.resolve(stoppedLease)
    await expect(resumeB).resolves.toBeUndefined()

    expect(resumeDemand).toHaveBeenCalledTimes(2)
    expect(resumeDemand.mock.calls.map(([input]) => input.baseStopEpoch)).toEqual([
      stoppedLease.stopEpoch,
      stoppedLease.stopEpoch,
    ])
  })

  it('refreshes a historical observed epoch at the boundary of every new user action', async () => {
    const { demand, resumeDemand, stopNatively, currentLease } = createDemandHarness()
    await demand.setActive('machine-old', true, Symbol('old'))
    stopNatively()

    const intent = demand.createResumeIntent()
    await expect(demand.resumeForUserIntent(intent)).resolves.toBeUndefined()

    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(resumeDemand.mock.calls[0]?.[0].baseStopEpoch).toBe(currentLease().stopEpoch)
    expect(currentLease().stopped).toBe(false)
  })

  it('drops every pre-stop endpoint owner before restoring only the fresh endpoint intent', async () => {
    const { demand, stopNatively, nativeEndpointIds } = createDemandHarness()
    await demand.setActive('machine-a', true, Symbol('old-a'))
    await demand.setActive('machine-b', true, Symbol('old-b'))
    stopNatively()

    const intent = demand.createResumeIntent()
    await expect(demand.resumeForUserIntent(intent)).resolves.toBeUndefined()
    await demand.setActive('machine-a', true, Symbol('fresh-a'))

    expect(nativeEndpointIds()).toEqual(['machine-a'])
  })

  it('does not let a concurrent startup reconciliation swallow the first post-reload resume', async () => {
    const { demand, getDemandLease, resumeDemand, stopNatively, currentLease } = createDemandHarness()
    const startupLease = deferred<NativeSessionDemandLease>()
    const intentLease = deferred<NativeSessionDemandLease>()
    stopNatively()
    getDemandLease
      .mockImplementationOnce(() => startupLease.promise)
      .mockImplementationOnce(() => intentLease.promise)

    const reconcile = demand.reconcileRenderer()
    await vi.waitFor(() => expect(getDemandLease).toHaveBeenCalledOnce())
    const intent = demand.createResumeIntent()
    const resumed = demand.resumeForUserIntent(intent)
    await vi.waitFor(() => expect(getDemandLease).toHaveBeenCalledTimes(2))
    expect(resumeDemand).not.toHaveBeenCalled()

    startupLease.resolve(currentLease())
    await expect(reconcile).resolves.toBeUndefined()
    expect(resumeDemand).not.toHaveBeenCalled()

    intentLease.resolve(currentLease())
    await expect(resumed).resolves.toBeUndefined()
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(resumeDemand.mock.calls[0]?.[0].baseStopEpoch).toBe(currentLease().stopEpoch)
  })

  it('does not upgrade an unknown-baseline intent across a concurrent local Stop', async () => {
    const { demand, getDemandLease, resumeDemand, stopNatively, notifyStop, currentLease } = createDemandHarness()
    const oldInitialLease = deferred<NativeSessionDemandLease>()
    const oldLease = currentLease()
    getDemandLease.mockImplementationOnce(() => oldInitialLease.promise)

    const oldIntent = demand.createResumeIntent()
    const oldResume = demand.resumeForUserIntent(oldIntent)
    await Promise.resolve()
    stopNatively()
    const stopped = notifyStop()
    oldInitialLease.resolve(oldLease)

    await stopped
    await expect(oldResume).rejects.toMatchObject({ code: 'user_stopped', retryable: false })
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(resumeDemand.mock.calls[0]?.[0].baseStopEpoch).toBe(oldLease.stopEpoch)
  })

  it('reuses the same native intent ID after a transport failure', async () => {
    const { demand, resumeDemand } = createDemandHarness()
    resumeDemand.mockRejectedValueOnce(new Error('native runtime is rebuilding'))
    const intent = {}

    await expect(demand.resumeForUserIntent(intent)).rejects.toThrow('native runtime is rebuilding')
    await expect(demand.resumeForUserIntent(intent)).resolves.toBeUndefined()

    expect(resumeDemand).toHaveBeenCalledTimes(2)
    expect(resumeDemand.mock.calls[1]?.[0].intentId).toBe(resumeDemand.mock.calls[0]?.[0].intentId)
  })

  it('keeps one post-stop intent retryable while its eager native resume is rebuilding', async () => {
    const { demand, resumeDemand, stopNatively, currentLease, nativeEndpointIds } = createDemandHarness()
    stopNatively()
    const event = { stopEpoch: currentLease().stopEpoch, stopped: true }
    resumeDemand.mockRejectedValueOnce(new Error('native runtime is rebuilding'))

    const intent = demand.createResumeIntent()
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())

    await expect(demand.handleDisconnectAllRequested(event)).resolves.toEqual(cleanupRequest(event.stopEpoch))
    await expect(demand.resumeForUserIntent(intent)).resolves.toBeUndefined()
    await expect(demand.setActive('machine-a', true, demandOwner)).resolves.toBeUndefined()

    expect(resumeDemand).toHaveBeenCalledTimes(2)
    expect(resumeDemand.mock.calls[1]?.[0].intentId).toBe(resumeDemand.mock.calls[0]?.[0].intentId)
    expect(nativeEndpointIds()).toEqual(['machine-a'])
  })

  it('keeps the first post-Stop intent alive while its epoch anchor is rebuilding', async () => {
    const { demand, getDemandLease, resumeDemand, stopNatively, currentLease } = createDemandHarness()
    stopNatively()
    const event = { stopEpoch: currentLease().stopEpoch, stopped: true }
    getDemandLease.mockRejectedValueOnce(new Error('native runtime is rebuilding'))

    const intent = demand.createResumeIntent()
    const resumed = demand.resumeForUserIntent(intent)
    await vi.waitFor(() => expect(getDemandLease).toHaveBeenCalledOnce())
    const notification = demand.handleDisconnectAllRequested(event)

    await expect(resumed).resolves.toBeUndefined()
    await expect(notification).resolves.toEqual(cleanupRequest(event.stopEpoch))
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(resumeDemand.mock.calls[0]?.[0].baseStopEpoch).toBe(event.stopEpoch)
  })

  it('lets a newer Stop epoch supersede an older transport-retry reservation', async () => {
    const { demand, resumeDemand, stopNatively, currentLease } = createDemandHarness()
    stopNatively()
    resumeDemand.mockRejectedValueOnce(new Error('native runtime is rebuilding'))
    const oldIntent = demand.createResumeIntent()
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())

    stopNatively()
    const newerEvent = { stopEpoch: currentLease().stopEpoch, stopped: true }

    await expect(demand.handleDisconnectAllRequested(newerEvent)).resolves.toEqual(cleanupRequest(newerEvent.stopEpoch))
    await expect(demand.resumeForUserIntent(oldIntent)).rejects.toMatchObject({ code: 'user_stopped' })
    await expect(demand.resumeForUserIntent(demand.createResumeIntent())).resolves.toBeUndefined()
  })

  it('never revives an accepted intent after its response is lost and a newer Stop wins', async () => {
    vi.useFakeTimers()
    const { demand, resumeDemand, resumeNatively, stopNatively } = createDemandHarness()
    try {
      resumeDemand.mockImplementationOnce(async (input) => {
        await resumeNatively(input)
        return await new Promise<NativeSessionDemandResumeResult>(() => {})
      })
      const oldIntent = {}
      const firstAttempt = demand.resumeForUserIntent(oldIntent)
      const timedOut = expect(firstAttempt).rejects.toMatchObject({ code: 'demand_sync_pending' })
      await vi.advanceTimersByTimeAsync(5_000)
      await timedOut

      stopNatively()
      await expect(demand.resumeForUserIntent(oldIntent)).rejects.toMatchObject({
        code: 'user_stopped',
        retryable: false,
      })
      await expect(demand.resumeForUserIntent({})).resolves.toBeUndefined()

      expect(resumeDemand).toHaveBeenCalledTimes(3)
      expect(resumeDemand.mock.calls[1]?.[0].intentId).toBe(resumeDemand.mock.calls[0]?.[0].intentId)
      expect(resumeDemand.mock.calls[2]?.[0].intentId).not.toBe(resumeDemand.mock.calls[0]?.[0].intentId)
    } finally {
      vi.useRealTimers()
    }
  })

  it('fails closed when Stop wins after intent creation but before native processing', async () => {
    const { demand, resumeDemand, resumeNatively, stopNatively } = createDemandHarness()
    const processIntent = deferred<void>()
    resumeDemand.mockImplementationOnce(async (input) => {
      await processIntent.promise
      return resumeNatively(input)
    })

    const attempt = demand.resumeForUserIntent({})
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())
    expect(resumeDemand.mock.calls[0]?.[0].baseStopEpoch).toBe('0')
    stopNatively()
    processIntent.resolve()

    await expect(attempt).rejects.toMatchObject({ code: 'user_stopped', retryable: false })
    expect(resumeDemand.mock.calls[0]?.[0]).toMatchObject({ baseStopEpoch: '0' })
  })

  it('freshly revalidates an in-flight transfer intent before an async continuation commits', async () => {
    const { demand, resumeDemand, resumeNatively, stopNatively } = createDemandHarness()
    const releaseOldResponse = deferred<void>()
    resumeDemand.mockImplementationOnce(async (input) => {
      const accepted = await resumeNatively(input)
      await releaseOldResponse.promise
      return accepted
    })

    const intent = demand.createResumeIntent()
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())
    stopNatively()
    const confirmation = demand.confirmResumeIntent(intent)
    releaseOldResponse.resolve()

    await expect(confirmation).rejects.toMatchObject({ code: 'user_stopped', retryable: false })
    expect(resumeDemand).toHaveBeenCalledTimes(2)
    expect(resumeDemand.mock.calls[1]?.[0].intentId).toBe(resumeDemand.mock.calls[0]?.[0].intentId)
  })

  it('rejects an intent prepared before Stop even when resume is first called afterward', async () => {
    const { demand, resumeDemand, stopNatively, notifyStop, nativeEndpointIds } = createDemandHarness()
    await demand.setActive('machine-old', true, Symbol('old'))
    const oldIntent = demand.createResumeIntent()

    stopNatively()
    await notifyStop()

    await expect(demand.resumeForUserIntent(oldIntent)).rejects.toMatchObject({
      code: 'user_stopped',
      retryable: false,
    })
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(nativeEndpointIds()).toEqual([])
    await expect(demand.setActive('machine-old', true, Symbol('stale'))).rejects.toMatchObject({ code: 'user_stopped' })

    const freshIntent = demand.createResumeIntent()
    await expect(demand.resumeForUserIntent(freshIntent)).resolves.toBeUndefined()
    expect(resumeDemand).toHaveBeenCalledTimes(2)
  })

  it('lets intent creation overtake a retained Stop read before the manager resumes it', async () => {
    const { demand, getDemandLease, resumeDemand, stopNatively, notifyStop, currentLease } = createDemandHarness()
    const refreshed = deferred<NativeSessionDemandLease>()
    stopNatively()
    const stoppedLease = currentLease()
    getDemandLease.mockImplementationOnce(() => refreshed.promise)

    const stopped = notifyStop()
    const freshIntent = demand.createResumeIntent()
    const fresh = demand.resumeForUserIntent(freshIntent)
    await expect(fresh).resolves.toBeUndefined()
    refreshed.resolve(stoppedLease)

    await expect(stopped).resolves.toEqual(cleanupRequest(stoppedLease.stopEpoch))
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(resumeDemand.mock.calls[0]?.[0].baseStopEpoch).toBe(currentLease().stopEpoch)
  })

  it('keeps native demand stopped when an old successful response arrives after Stop', async () => {
    vi.useFakeTimers()
    const { demand, resumeDemand, resumeNatively, stopNatively, nativeEndpointIds } = createDemandHarness()
    const releaseResponse = deferred<void>()
    try {
      await demand.setActive('machine-old', true, Symbol('old'))
      resumeDemand.mockImplementationOnce(async (input) => {
        const accepted = await resumeNatively(input)
        await releaseResponse.promise
        return accepted
      })
      const oldAttempt = demand.resumeForUserIntent({})
      await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())

      stopNatively()
      releaseResponse.resolve()
      await oldAttempt
      await expect(demand.setActive('machine-fresh', true, Symbol('fresh'))).rejects.toThrow('stale native demand lease')
      await vi.advanceTimersByTimeAsync(250)

      expect(nativeEndpointIds()).toEqual([])
    } finally {
      vi.useRealTimers()
    }
  })

  it('lets every distinct concurrent user action register its own native intent', async () => {
    const { demand, resumeDemand, stopNatively, notifyStop } = createDemandHarness()
    stopNatively()
    await notifyStop()

    await Promise.all([
      demand.resumeForUserIntent({}),
      demand.resumeForUserIntent({}),
    ])

    expect(resumeDemand).toHaveBeenCalledTimes(2)
    expect(resumeDemand.mock.calls[1]?.[0].intentId).not.toBe(resumeDemand.mock.calls[0]?.[0].intentId)
  })

  it('does not let a delayed same-epoch resume response regress a newer demand lease', async () => {
    const { demand, replaceDemand, resumeDemand, resumeNatively, stopNatively, notifyStop } = createDemandHarness()
    const releaseFirstResponse = deferred<void>()
    stopNatively()
    await notifyStop()
    resumeDemand.mockImplementationOnce(async (input) => {
      const accepted = await resumeNatively(input)
      await releaseFirstResponse.promise
      return accepted
    })

    const delayed = demand.resumeForUserIntent({})
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())
    await demand.resumeForUserIntent({})
    await demand.setActive('machine-a', true, Symbol('fresh-a'))
    releaseFirstResponse.resolve()
    await delayed
    await demand.setActive('machine-b', true, Symbol('fresh-b'))

    expect(replaceDemand.mock.calls.at(-1)?.[0]).toMatchObject({
      baseDemandRevision: '3',
      endpointIds: ['machine-a', 'machine-b'],
    })
  })

  it('does not let a delayed old response clobber a fresh post-stop resume', async () => {
    const { demand, resumeDemand, resumeNatively, stopNatively, notifyStop } = createDemandHarness()
    const releaseOldResponse = deferred<void>()
    resumeDemand.mockImplementationOnce(async (input) => {
      const accepted = await resumeNatively(input)
      await releaseOldResponse.promise
      return accepted
    })

    const oldAttempt = demand.resumeForUserIntent({})
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())
    stopNatively()
    await notifyStop()
    const freshAttempt = demand.resumeForUserIntent({})
    await freshAttempt
    releaseOldResponse.resolve()

    await expect(oldAttempt).rejects.toMatchObject({ code: 'user_stopped', retryable: false })
    await expect(demand.setActive('machine-fresh', true, Symbol('fresh'))).resolves.toBeUndefined()
  })

  it('fences a locally known old intent but lets one new click resume', async () => {
    const { demand, resumeDemand, stopNatively, notifyStop } = createDemandHarness()
    resumeDemand.mockRejectedValueOnce(new Error('native runtime is rebuilding'))
    const oldIntent = {}
    await expect(demand.resumeForUserIntent(oldIntent)).rejects.toThrow('native runtime is rebuilding')

    stopNatively()
    await notifyStop()
    await expect(demand.resumeForUserIntent(oldIntent)).rejects.toMatchObject({ code: 'user_stopped' })
    await expect(demand.resumeForUserIntent({})).resolves.toBeUndefined()
    expect(resumeDemand).toHaveBeenCalledTimes(2)
  })

  it('releases the queue after an unresponsive native replacement', async () => {
    vi.useFakeTimers()
    const behavior = vi.fn()
      .mockReturnValueOnce(new Promise<void>(() => {}))
      .mockResolvedValueOnce(undefined)
    const { demand, replaceDemand } = createDemandHarness(behavior)
    try {
      const first = demand.setActive('machine-a', true, demandOwner)
      await vi.waitFor(() => expect(replaceDemand).toHaveBeenCalledTimes(1))
      const reconciled = demand.reconcileRenderer()

      await vi.advanceTimersByTimeAsync(5_000)

      await expect(first).rejects.toMatchObject({ code: 'demand_sync_pending' })
      await expect(reconciled).resolves.toBeUndefined()
      expect(replaceDemand).toHaveBeenNthCalledWith(2, expect.objectContaining({ endpointIds: ['machine-a'] }))
    } finally {
      vi.useRealTimers()
    }
  })

  it('fences updates that were queued before a native user stop', async () => {
    let currentLease: NativeSessionDemandLease = {
      attachmentId: 'renderer-a', demandRevision: '0', stopEpoch: '0', endpointIds: [], stopped: false,
    }
    let finishFirst!: () => void
    const first = new Promise<NativeSessionDemandLease>((resolve) => {
      finishFirst = () => resolve({
        attachmentId: 'renderer-a', demandRevision: '1', stopEpoch: '0', endpointIds: ['machine-a'], stopped: false,
      })
    })
    const getDemandLease = vi.fn(async () => currentLease)
    const replaceDemand = vi.fn((input: NativeSessionDemandInput): Promise<NativeSessionDemandLease> => {
      if (input.endpointIds.length > 0) {
        currentLease = {
          attachmentId: 'renderer-a', demandRevision: '1', stopEpoch: '0', endpointIds: ['machine-a'], stopped: false,
        }
        return first
      }
      currentLease = {
        attachmentId: input.attachmentId, demandRevision: '3', stopEpoch: '2', endpointIds: [], stopped: true,
      }
      return Promise.resolve(currentLease)
    })
    const demand = new NativeSessionDemandCoordinator(replaceDemand, getDemandLease)

    const activeA = demand.setActive('machine-a', true, demandOwner)
    await vi.waitFor(() => expect(replaceDemand).toHaveBeenCalledTimes(1))
    const staleActiveB = demand.setActive('machine-b', true, demandOwner)
    // Native notification action clears Demand and rotates the lease before JS handles the event.
    currentLease = {
      attachmentId: 'renderer-b', demandRevision: '2', stopEpoch: '2', endpointIds: [], stopped: true,
    }
    const stopped = demand.handleDisconnectAllRequested({ stopEpoch: '2', stopped: true })
    finishFirst()

    await activeA
    await staleActiveB
    await stopped
    expect(replaceDemand.mock.calls.map(([input]) => input.endpointIds)).toEqual([
      ['machine-a'],
    ])
    expect(getDemandLease).toHaveBeenCalledTimes(2)
  })
})

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

function cleanupRequest(stopEpoch: string, protectedEndpointIds: string[] = []) {
  return { stopEpoch, protectedEndpointIds }
}
