import { create } from '@bufbuild/protobuf'
import { describe, expect, it, vi } from 'vitest'
import type { ProtoClientSession, ProtoClientSubscription, ProtoResourceStream, RtcConnectOptions } from '@anytty/ui'
import { EndpointSessionStampSchema, ResourceHandleSchema, type ResourceHandle } from '../../ui/src/generated/apipb/common_pb'
import { CommandEnvelopeSchema, ReleaseResourceCommandSchema, ResultEnvelopeSchema, type CommandEnvelope, type EventEnvelope, type ResultEnvelope } from '../../ui/src/generated/apipb/application_pb'
import { ConnectionObservedPath, ConnectionRouteKind, ConnectionSnapshotSchema, type ConnectionSnapshot } from '../../ui/src/generated/bindingpb/client_binding_pb'
import { NativeSessionManager } from './NativeSessionManager'

type ProtoClientSessionCloseHandler = Parameters<ProtoClientSession['subscribeClosed']>[0]
type ProtoClientSessionCloseError = Parameters<ProtoClientSessionCloseHandler>[0]

describe('NativeSessionManager', () => {
  it('emits lifecycle diagnostics without exposing the endpoint identifier', async () => {
    const writeDiagnostic = vi.fn()
    const manager = new NativeSessionManager('private-endpoint-id', {
      connect: vi.fn(async () => fakeSession()),
    }, { writeDiagnostic })

    await manager.foregroundResume()

    expect(writeDiagnostic).toHaveBeenCalledWith(expect.stringContaining('event=session_created'))
    expect(writeDiagnostic).toHaveBeenCalledWith(expect.stringContaining('event=session_foreground_resume_skipped'))
    expect(writeDiagnostic.mock.calls.flat().join(' ')).not.toContain('private-endpoint-id')
  })

  it('waits for foreground readiness before opening the first renderer binding', async () => {
    const gate = deferred<void>()
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager('daemon-a', { connect }, {
      waitForForeground: async () => await gate.promise,
    })

    const opening = manager.get()
    await Promise.resolve()
    expect(connect).not.toHaveBeenCalled()

    gate.resolve()
    await expect(opening).resolves.toMatchObject({ stamp: { generation: 1n } })
    expect(connect).toHaveBeenCalledOnce()
    await manager.reset()
  })

  it('shares one renderer binding and releases passive demand without disconnecting the Go winner', async () => {
    const session = fakeSession()
    const connect = vi.fn(async () => session)
    const disconnect = vi.fn(async () => undefined)
    const setActive = vi.fn(async (_machineId: string, _active: boolean) => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, disconnect, setActive })

    const workspace = await manager.lease()
    const transfer = await manager.get()
    expect(connect).toHaveBeenCalledOnce()
    expect(setActive).toHaveBeenCalledTimes(1)
    expect(setActive).toHaveBeenLastCalledWith('daemon-a', true)

    await workspace.close()
    await Promise.resolve()
    expect(session.close).not.toHaveBeenCalled()

    await transfer.close()
    await vi.waitFor(() => expect(session.close).toHaveBeenCalledOnce())
    expect(setActive).toHaveBeenLastCalledWith('daemon-a', false)
    expect(disconnect).not.toHaveBeenCalled()
    expect(session.invalidate).not.toHaveBeenCalled()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'idle' })
  })

  it('preserves demand across a binding generation and rebinds on foreground', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const setActive = vi.fn(async () => undefined)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, setActive, disconnect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const oldLease = await manager.get()

    await manager.resetBindingGeneration()

    expect(manager.hasConnectionDemand()).toBe(true)
    expect(oldLease.isAlive()).toBe(false)
    expect(first.close).toHaveBeenCalledOnce()
    expect(first.invalidate).not.toHaveBeenCalled()
    expect(setActive).not.toHaveBeenCalledWith('daemon-a', false)
    expect(disconnect).not.toHaveBeenCalled()

    await manager.foregroundResume()
    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })

    await oldLease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('fences the renderer lease offline and reacquires it online without invalidating Go', async () => {
    const retained = fakeSession()
    const recovered = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(retained)
      .mockResolvedValueOnce(recovered)
    const setActive = vi.fn(async () => undefined)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, setActive, disconnect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const rendererLease = await manager.get()

    await manager.networkChanged(false, 'offline')

    expect(rendererLease.isAlive()).toBe(false)
    expect(retained.close).toHaveBeenCalledOnce()
    expect(retained.invalidate).not.toHaveBeenCalled()
    expect(setActive).not.toHaveBeenCalledWith('daemon-a', false)
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'waiting_network' })

    await manager.networkChanged(true, 'available')

    expect(connect).toHaveBeenCalledTimes(2)
    expect(disconnect).not.toHaveBeenCalled()
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })

    await rendererLease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('preserves a healthy renderer lease on foreground resume', async () => {
    const retained = fakeSession()
    const connect = vi.fn(async () => retained)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const oldLease = await manager.get()

    await manager.foregroundResume()

    expect(oldLease.isAlive()).toBe(true)
    expect(retained.close).not.toHaveBeenCalled()
    expect(retained.invalidate).not.toHaveBeenCalled()
    expect(retained.execute).not.toHaveBeenCalled()
    expect(connect).toHaveBeenCalledOnce()
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 1n },
    })

    await oldLease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('joins an in-flight renderer binding on foreground resume', async () => {
    const openingGate = deferred<ProtoClientSession>()
    const session = fakeSession()
    const connect = vi.fn(async () => await openingGate.promise)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const opening = manager.get()
    await vi.waitFor(() => expect(connect).toHaveBeenCalledOnce())

    const resuming = manager.foregroundResume()
    await Promise.resolve()
    expect(connect).toHaveBeenCalledOnce()

    openingGate.resolve(session)
    const lease = await opening
    await resuming

    expect(connect).toHaveBeenCalledOnce()
    expect(lease.isAlive()).toBe(true)
    expect(session.close).not.toHaveBeenCalled()

    await lease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('cancels only the old foreground waiter and lets the next foreground join its binding', async () => {
    const openingGate = deferred<ProtoClientSession>()
    const stale = fakeSession()
    const session = fakeSession()
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockImplementationOnce(async () => await openingGate.promise)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const oldLease = await manager.get()
    const oldLifecycle = new AbortController()
    stale.markDead()

    const backgrounded = manager.foregroundResume(oldLifecycle.signal)
    await vi.waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    oldLifecycle.abort(new Error('app backgrounded'))
    await expect(backgrounded).rejects.toThrow('app backgrounded')

    const resumed = manager.foregroundResume()
    await Promise.resolve()
    expect(connect).toHaveBeenCalledTimes(2)
    openingGate.resolve(session)
    await resumed

    expect(connect).toHaveBeenCalledTimes(2)
    expect(session.isAlive()).toBe(true)
    expect(session.close).not.toHaveBeenCalled()
    expect(session.invalidate).not.toHaveBeenCalled()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })

    await oldLease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('cancels foreground recovery while native demand retry is backing off', async () => {
    const stale = fakeSession()
    const rejectedBinding = deferred<ProtoClientSession>()
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockImplementationOnce(() => rejectedBinding.promise)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const oldLease = await manager.get()
    const lifecycle = new AbortController()
    stale.markDead()

    const backgrounded = manager.foregroundResume(lifecycle.signal)
    await vi.waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    rejectedBinding.reject(Object.assign(new Error('native runtime unavailable'), {
      code: 'demand_sync_pending',
      retryable: true,
    }))
    await Promise.resolve()
    lifecycle.abort(new Error('app backgrounded'))

    await expect(backgrounded).rejects.toThrow('app backgrounded')

    await oldLease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('waits for the current binding when a network revision supersedes foreground recovery', async () => {
    const firstOpening = deferred<ProtoClientSession>()
    const currentOpening = deferred<ProtoClientSession>()
    const current = fakeSession(2n)
    const connect = vi.fn()
      .mockImplementationOnce(async (_input: { machineId: string }, options?: RtcConnectOptions) => {
        return await new Promise<ProtoClientSession>((resolve, reject) => {
          const aborted = () => reject(options?.signal?.reason ?? new Error('aborted'))
          options?.signal?.addEventListener('abort', aborted, { once: true })
          void firstOpening.promise.then(resolve, reject)
        })
      })
      .mockImplementationOnce(async () => await currentOpening.promise)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const consumer = manager.get()
    await vi.waitFor(() => expect(connect).toHaveBeenCalledOnce())

    let foregroundSettled = false
    const foreground = manager.foregroundResume().finally(() => { foregroundSettled = true })
    const networkChange = manager.networkChanged(true, 'network_replaced')
    await vi.waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    await Promise.resolve()
    expect(foregroundSettled).toBe(false)

    currentOpening.resolve(current)
    await networkChange
    await foreground
    const lease = await consumer

    expect(lease.isAlive()).toBe(true)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })

    await lease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('replaces a stale renderer binding on foreground resume', async () => {
    const stale = fakeSession()
    const recovered = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const oldLease = await manager.get()
    stale.markDead()

    await manager.foregroundResume()

    expect(oldLease.isAlive()).toBe(false)
    expect(stale.close).toHaveBeenCalledOnce()
    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })

    await oldLease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('closes a late stale binding and lets its consumer follow the current generation', async () => {
    vi.useFakeTimers()
    const staleOpen = deferred<ProtoClientSession>()
    const currentOpen = deferred<ProtoClientSession>()
    const stale = fakeSession()
    const current = fakeSession(2n)
    const connect = vi.fn()
      .mockReturnValueOnce(staleOpen.promise)
      .mockReturnValueOnce(currentOpen.promise)
    const manager = new NativeSessionManager('daemon-a', { connect })

    try {
      const opening = manager.get()
      await vi.waitFor(() => expect(connect).toHaveBeenCalledOnce())
      const changed = manager.networkChanged(true, 'network_replaced')
      await vi.waitFor(() => expect(connect).toHaveBeenCalledTimes(2))

      currentOpen.resolve(current)
      await changed
      staleOpen.resolve(stale)
      await Promise.resolve()
      await vi.advanceTimersByTimeAsync(250)

      await expect(opening).resolves.toMatchObject({ stamp: { generation: 2n } })
      expect(stale.close).toHaveBeenCalledOnce()
      expect(stale.invalidate).not.toHaveBeenCalled()
      expect(current.invalidate).not.toHaveBeenCalled()
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('automatically reacquires a follower binding after an asynchronous close', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand({})
    const lease = await manager.get()

    try {
      first.fail(Object.assign(new Error('binding closed'), { code: 'unavailable', retryable: true }))
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'reconnecting' })

      await vi.advanceTimersByTimeAsync(0)

      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'connected',
        connectionInfo: { generation: 2n },
      })
    } finally {
      await lease.close()
      releaseWorkspace()
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('keeps a cold renderer follower recovering across stale generations until native ready', async () => {
    vi.useFakeTimers()
    const stale = Object.assign(new Error('endpoint session generation was replaced'), {
      code: 'stale_session',
      retryable: false,
    })
    const old = fakeSession()
    const current = fakeSession(2n)
    const nativeReady = deferred<ProtoClientSession>()
    const connect = vi.fn()
      .mockResolvedValueOnce(old)
      .mockReturnValueOnce(nativeReady.promise)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand({})
    const lease = await manager.get()
    const phases: string[] = []
    const unsubscribe = manager.connectionState.subscribe(() => {
      phases.push(manager.connectionState.getSnapshot().phase)
    })

    try {
      old.fail(stale)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'reconnecting' })

      await vi.advanceTimersByTimeAsync(0)
      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connecting' })

      nativeReady.resolve(current)
      await Promise.resolve()
      await Promise.resolve()
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'connected',
        connectionInfo: { generation: 2n },
      })
      expect(phases).not.toContain('failed')
    } finally {
      unsubscribe()
      await lease.close()
      releaseWorkspace()
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('releases a consumer demand owner when the underlying binding closes', async () => {
    vi.useFakeTimers()
    const sessions = [fakeSession(), fakeSession(2n), fakeSession(3n), fakeSession(4n)]
    const connect = vi.fn()
    for (const session of sessions) connect.mockResolvedValueOnce(session)
    const setActive = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, setActive })
    const releaseWorkspace = manager.retainConnectionDemand()
    let lease = await manager.get()

    try {
      for (let index = 0; index < 3; index += 1) {
        sessions[index]!.fail(Object.assign(new Error('binding closed'), { code: 'unavailable', retryable: true }))
        await vi.advanceTimersByTimeAsync(0)
        expect(lease.isAlive()).toBe(false)
        expect(connect).toHaveBeenCalledTimes(index + 2)
        lease = await manager.get()
      }

      await lease.close()
      releaseWorkspace()
      await vi.waitFor(() => expect(setActive).toHaveBeenLastCalledWith('daemon-a', false))
      expect(manager.hasConnectionDemand()).toBe(false)
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('releases every consumer owner across repeated manager-driven binding fences', async () => {
    const sessions = [fakeSession(), fakeSession(2n), fakeSession(3n)]
    const connect = vi.fn()
    for (const session of sessions) connect.mockResolvedValueOnce(session)
    const setActive = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, setActive })
    const releaseWorkspace = manager.retainConnectionDemand()

    for (let index = 0; index < sessions.length; index += 1) {
      const lease = await manager.get()
      const closed = vi.fn()
      lease.subscribeClosed(closed)

      await manager.resetBindingGeneration()

      expect(lease.isAlive()).toBe(false)
      expect(closed).toHaveBeenCalledOnce()
    }

    releaseWorkspace()
    await vi.waitFor(() => expect(setActive).toHaveBeenLastCalledWith('daemon-a', false))
    expect(manager.hasConnectionDemand()).toBe(false)
    await manager.reset()
  })

  it('retries transient demand synchronization without requiring another user action', async () => {
    vi.useFakeTimers()
    const setActive = vi.fn()
      .mockRejectedValueOnce(new Error('renderer attachment changed'))
      .mockResolvedValue(undefined)
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager('daemon-a', { connect, setActive })

    try {
      const opening = manager.get()
      await Promise.resolve()
      expect(connect).not.toHaveBeenCalled()

      await vi.advanceTimersByTimeAsync(250)

      const lease = await opening
      expect(setActive).toHaveBeenNthCalledWith(1, 'daemon-a', true)
      expect(setActive).toHaveBeenNthCalledWith(2, 'daemon-a', true)
      expect(connect).toHaveBeenCalledOnce()
      await lease.close()
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('coalesces concurrent activation retries and backs off under sustained failure', async () => {
    vi.useFakeTimers()
    const setActive = vi.fn(async (_machineId: string, _active: boolean): Promise<void> => {
      throw new Error('native runtime unavailable')
    })
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager('daemon-a', { connect, setActive })
    const controllers = [new AbortController(), new AbortController(), new AbortController()]
    const openings = controllers.map(({ signal }) => manager.get({ signal }))

    try {
      await vi.advanceTimersByTimeAsync(0)
      expect(setActive).toHaveBeenCalledTimes(1)

      await vi.advanceTimersByTimeAsync(30_000)

      expect(setActive.mock.calls.filter(([, active]) => active === true).length).toBeLessThanOrEqual(7)
      expect(connect).not.toHaveBeenCalled()
    } finally {
      setActive.mockImplementation(async () => undefined)
      for (const controller of controllers) controller.abort(new Error('test complete'))
      await Promise.allSettled(openings)
      await manager.reset().catch(() => undefined)
      vi.useRealTimers()
    }
  })

  it('lets fresh user intent supersede an old capped demand retry window', async () => {
    vi.useFakeTimers()
    const setActive = vi.fn(async (_machineId: string, _active: boolean): Promise<void> => {
      throw new Error('native runtime unavailable')
    })
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => fakeSession()),
      setActive,
    })
    const oldController = new AbortController()
    const freshController = new AbortController()
    const oldOpening = manager.get({ signal: oldController.signal })
    let freshOpening: Promise<ProtoClientSession> | undefined
    let releaseWorkspace: (() => void) | undefined

    try {
      // Reach the capped 15 second retry window: 250 + 500 + 1000 + 2000 + 5000.
      await vi.advanceTimersByTimeAsync(8_750)
      const attemptsBeforeFreshIntent = setActive.mock.calls.filter(([, active]) => active === true).length
      expect(attemptsBeforeFreshIntent).toBeGreaterThanOrEqual(6)

      releaseWorkspace = manager.retainConnectionDemand({})
      freshOpening = manager.get({ signal: freshController.signal })
      await vi.advanceTimersByTimeAsync(0)

      const immediateAttempts = setActive.mock.calls.filter(([, active]) => active === true).length
      expect(immediateAttempts).toBeGreaterThan(attemptsBeforeFreshIntent)
      await vi.advanceTimersByTimeAsync(249)
      expect(setActive.mock.calls.filter(([, active]) => active === true)).toHaveLength(immediateAttempts)
      await vi.advanceTimersByTimeAsync(1)
      expect(setActive.mock.calls.filter(([, active]) => active === true).length).toBeGreaterThan(immediateAttempts)
    } finally {
      oldController.abort(new Error('test complete'))
      freshController.abort(new Error('test complete'))
      await Promise.allSettled([oldOpening, ...(freshOpening ? [freshOpening] : [])])
      releaseWorkspace?.()
      setActive.mockImplementation(async () => undefined)
      await manager.reset().catch(() => undefined)
      vi.useRealTimers()
    }
  })

  it('retries a transient explicit-resume failure without requiring a second click', async () => {
    vi.useFakeTimers()
    const resumeDemand = vi.fn()
      .mockRejectedValueOnce(new Error('native plugin is rebuilding'))
      .mockResolvedValue(undefined)
    const preparedIntent = {}
    const createResumeIntent = vi.fn(() => preparedIntent)
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager('daemon-a', { connect, createResumeIntent, resumeDemand })
    const releaseWorkspace = manager.retainConnectionDemand(preparedIntent)

    try {
      const opening = manager.get()
      await Promise.resolve()
      expect(connect).not.toHaveBeenCalled()

      await vi.advanceTimersByTimeAsync(250)

      const lease = await opening
      expect(createResumeIntent).not.toHaveBeenCalled()
      expect(resumeDemand).toHaveBeenCalledTimes(2)
      expect(resumeDemand.mock.calls[0]?.[0]).toBe(resumeDemand.mock.calls[1]?.[0])
      expect(resumeDemand.mock.calls[0]?.[0]).toBe(preparedIntent)
      expect(connect).toHaveBeenCalledOnce()
      await lease.close()
    } finally {
      releaseWorkspace()
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it.each([
    {
      lifecycleEvent: 'a network path change',
      advanceEpoch: async (manager: NativeSessionManager) => {
        await manager.networkChanged(true, 'path_changed')
      },
    },
    {
      lifecycleEvent: 'a foreground resume',
      advanceEpoch: async (manager: NativeSessionManager) => {
        await manager.foregroundResume()
      },
    },
  ])('keeps one manual reconnect alive across $lifecycleEvent during native resume', async ({ advanceEpoch }) => {
    const harness = await pendingManualRecoveryHarness()

    try {
      await advanceEpoch(harness.manager)
      harness.resumeGate.resolve()
      await expect(harness.recovery).resolves.toBeUndefined()

      expect(harness.requestRecovery).toHaveBeenCalledOnce()
      expect(harness.connect).toHaveBeenCalledTimes(2)
      expect(harness.manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'connected',
        connectionInfo: { generation: 2n },
      })
    } finally {
      harness.resumeGate.resolve()
      harness.releaseWorkspace()
      await harness.manager.disconnect().catch(() => undefined)
    }
  })

  it.each([
    {
      cancellation: 'an explicit disconnect',
      cancel: async (manager: NativeSessionManager) => {
        await manager.disconnect()
      },
    },
    {
      cancellation: 'a newer user intent',
      cancel: async (manager: NativeSessionManager) => {
        manager.beginUserConnectionIntent()
      },
    },
    {
      cancellation: 'consumer cancellation',
      cancel: async (manager: NativeSessionManager) => {
        await manager.reset()
      },
    },
  ])('fences a pending manual reconnect after $cancellation', async ({ cancel }) => {
    const harness = await pendingManualRecoveryHarness()

    try {
      await cancel(harness.manager)
      const rejected = expect(harness.recovery).rejects.toMatchObject({ code: 'cancelled' })
      harness.resumeGate.resolve()
      await rejected

      expect(harness.requestRecovery).not.toHaveBeenCalled()
      expect(harness.connect).toHaveBeenCalledOnce()
    } finally {
      harness.resumeGate.resolve()
      harness.releaseWorkspace()
      await harness.manager.disconnect().catch(() => undefined)
    }
  })

  it('keeps one manual retry alive across transient resume and activation failures', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const recovered = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(recovered)
    const resumeDemand = vi.fn()
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error('native resume is rebuilding'))
      .mockResolvedValue(undefined)
    const setActive = vi.fn()
      .mockResolvedValueOnce(undefined)
      .mockResolvedValueOnce(undefined)
      .mockRejectedValueOnce(new Error('native demand handoff is rebuilding'))
      .mockResolvedValue(undefined)
    const requestRecovery = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      resumeDemand,
      setActive,
      requestRecovery,
      disconnect: vi.fn(async () => undefined),
    })
    const releaseWorkspace = manager.retainConnectionDemand({})
    await manager.get()
    await manager.disconnect()

    try {
      const recovery = manager.resetClientOnly()
      await Promise.resolve()
      expect(connect).toHaveBeenCalledOnce()

      await vi.advanceTimersByTimeAsync(250)
      expect(setActive).toHaveBeenCalledTimes(3)
      expect(connect).toHaveBeenCalledOnce()

      await vi.advanceTimersByTimeAsync(500)
      await recovery

      expect(resumeDemand).toHaveBeenCalledTimes(3)
      expect(setActive).toHaveBeenCalledTimes(4)
      expect(requestRecovery).toHaveBeenCalledOnce()
      expect(connect).toHaveBeenCalledTimes(2)
    } finally {
      releaseWorkspace()
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('uses endpoint recovery for manual reconnect without disconnecting or invalidating the winner', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const requestRecovery = vi.fn(async () => undefined)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, requestRecovery, disconnect })
    const releaseWorkspace = manager.retainConnectionDemand({})
    const lease = await manager.get()

    await manager.resetClientOnly()

    expect(requestRecovery).toHaveBeenCalledOnce()
    expect(disconnect).not.toHaveBeenCalled()
    expect(first.close).toHaveBeenCalledOnce()
    expect(first.invalidate).not.toHaveBeenCalled()
    expect(connect).toHaveBeenCalledTimes(2)

    await lease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('continues manual reconnect when the native recovery hint never settles', async () => {
    vi.useFakeTimers()
    const frozenRecovery = deferred<void>()
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      requestRecovery: vi.fn(() => frozenRecovery.promise),
    })
    const releaseWorkspace = manager.retainConnectionDemand()
    await manager.get()

    try {
      const recovery = manager.resetClientOnly()
      await vi.advanceTimersByTimeAsync(1_499)
      expect(connect).toHaveBeenCalledOnce()

      await vi.advanceTimersByTimeAsync(1)
      await recovery

      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'connected',
        connectionInfo: { generation: 2n },
      })
    } finally {
      frozenRecovery.resolve()
      releaseWorkspace()
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('reattaches after transient repair and binding failures without a second manual retry', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const recovered = fakeSession(3n)
    const unavailable = Object.assign(new Error('native runtime is rebuilding'), {
      code: 'unavailable',
      retryable: true,
    })
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockRejectedValueOnce(unavailable)
      .mockResolvedValueOnce(recovered)
    const requestRecovery = vi.fn(async () => { throw unavailable })
    const manager = new NativeSessionManager('daemon-a', { connect, requestRecovery })
    const releaseWorkspace = manager.retainConnectionDemand()
    const lease = await manager.get()

    try {
      await manager.resetClientOnly()
      expect(requestRecovery).toHaveBeenCalledOnce()
      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'reconnecting' })

      await vi.advanceTimersByTimeAsync(0)

      expect(connect).toHaveBeenCalledTimes(3)
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'connected',
        connectionInfo: { generation: 3n },
      })
    } finally {
      await lease.close()
      releaseWorkspace()
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('calls the physical disconnect only for explicit disconnect and does not auto-reconnect', async () => {
    vi.useFakeTimers()
    const session = fakeSession()
    const connect = vi.fn(async () => session)
    const disconnect = vi.fn(async () => undefined)
    const setActive = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, disconnect, setActive })
    const lease = await manager.get()
    const leaseClosed = vi.fn()
    lease.subscribeClosed(leaseClosed)

    try {
      await manager.disconnect()

      expect(disconnect).toHaveBeenCalledOnce()
      expect(disconnect).toHaveBeenCalledWith('daemon-a')
      expect(setActive).toHaveBeenLastCalledWith('daemon-a', false)
      expect(session.close).toHaveBeenCalledOnce()
      expect(session.invalidate).not.toHaveBeenCalled()
      await vi.runAllTimersAsync()
      expect(connect).toHaveBeenCalledOnce()
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'failed',
        error: { code: 'user_stopped', retryable: false },
      })
      expect(leaseClosed).toHaveBeenCalledWith(expect.objectContaining({
        code: 'user_stopped',
        retryable: false,
      }))
    } finally {
      await lease.close()
      vi.useRealTimers()
    }
  })

  it('still performs the physical disconnect when Demand deactivation fails', async () => {
    const session = fakeSession()
    const deactivation = deferred<void>()
    const setActive = vi.fn()
      .mockResolvedValueOnce(undefined)
      .mockImplementationOnce(async () => await deactivation.promise)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => session),
      setActive,
      disconnect,
    })
    const lease = await manager.get()

    const stopping = manager.disconnect()
    await vi.waitFor(() => expect(setActive).toHaveBeenLastCalledWith('daemon-a', false))
    expect(disconnect).not.toHaveBeenCalled()
    deactivation.reject(new Error('renderer attachment changed'))
    await expect(stopping).resolves.toBeUndefined()

    expect(disconnect).toHaveBeenCalledOnce()
    expect(session.close).toHaveBeenCalledOnce()
    await lease.close()
  })

  it('does not let a delayed physical disconnect overwrite a fresh user intent', async () => {
    const session = fakeSession()
    const deactivation = deferred<void>()
    const setActive = vi.fn()
      .mockResolvedValueOnce(undefined)
      .mockImplementationOnce(async () => await deactivation.promise)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => session),
      setActive,
      disconnect,
    })
    const lease = await manager.get()

    const stopping = manager.disconnect()
    await vi.waitFor(() => expect(setActive).toHaveBeenLastCalledWith('daemon-a', false))
    manager.beginUserConnectionIntent()
    deactivation.resolve()

    await expect(stopping).resolves.toBeUndefined()
    expect(disconnect).not.toHaveBeenCalled()
    expect(session.close).toHaveBeenCalledOnce()
    await lease.close()
  })

  it('fences a consumer that was waiting for foreground when the user stops', async () => {
    const foreground = deferred<void>()
    const connect = vi.fn(async () => fakeSession())
    const setActive = vi.fn(async () => undefined)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, setActive, disconnect }, {
      waitForForeground: async () => await foreground.promise,
    })

    const opening = manager.get()
    await Promise.resolve()
    await manager.disconnect()
    foreground.resolve()

    await expect(opening).rejects.toMatchObject({ code: 'cancelled' })
    expect(connect).not.toHaveBeenCalled()
    expect(setActive).not.toHaveBeenCalledWith('daemon-a', true)
  })

  it('does not let a manual recovery waiting for foreground cross a newer user stop', async () => {
    const foreground = deferred<void>()
    let shouldWait = false
    const setActive = vi.fn(async () => undefined)
    const requestRecovery = vi.fn(async () => undefined)
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      setActive,
      requestRecovery,
      disconnect: vi.fn(async () => undefined),
    }, {
      waitForForeground: async () => { if (shouldWait) await foreground.promise },
    })
    const releaseWorkspace = manager.retainConnectionDemand()
    await manager.get()
    shouldWait = true

    const recovery = manager.resetClientOnly()
    await Promise.resolve()
    await manager.disconnect()
    foreground.resolve()

    await expect(recovery).rejects.toMatchObject({ code: 'cancelled' })
    expect(setActive.mock.calls).toEqual([
      ['daemon-a', true],
      ['daemon-a', false],
    ])
    expect(requestRecovery).not.toHaveBeenCalled()
    expect(connect).toHaveBeenCalledOnce()
    releaseWorkspace()
  })

  it('blocks background consumers after user stop until a new workspace intent resumes demand', async () => {
    const connect = vi.fn(async () => fakeSession())
    const setActive = vi.fn(async () => undefined)
    const resumeDemand = vi.fn(async () => undefined)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      setActive,
      resumeDemand,
      disconnect,
    })

    await manager.disconnect()
    await expect(manager.get()).rejects.toMatchObject({ code: 'user_stopped', retryable: false })
    expect(resumeDemand).not.toHaveBeenCalled()

    const freshIntent = {}
    const releaseWorkspace = manager.retainConnectionDemand(freshIntent)
    const lease = await manager.get()
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(manager.latestUserResumeIntent()).toBe(freshIntent)
    expect(setActive).toHaveBeenLastCalledWith('daemon-a', true)
    expect(connect).toHaveBeenCalledOnce()

    await lease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('retains the latest successful resume intent until explicit disconnect', async () => {
    const firstIntent = {}
    const newerIntent = {}
    const onUserResumeAccepted = vi.fn()
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => fakeSession()),
      setActive: vi.fn(async () => undefined),
      resumeDemand: vi.fn(async () => undefined),
      disconnect: vi.fn(async () => undefined),
    }, {
      onUserResumeAccepted,
    })
    const releaseWorkspace = manager.retainConnectionDemand(firstIntent)

    const lease = await manager.get()
    expect(manager.latestUserResumeIntent()).toBe(firstIntent)
    expect(onUserResumeAccepted).toHaveBeenCalledWith(firstIntent)

    manager.beginUserConnectionIntent(newerIntent)
    expect(manager.latestUserResumeIntent()).toBe(newerIntent)

    await manager.disconnect()
    expect(manager.latestUserResumeIntent()).toBeNull()
    expect(lease.isAlive()).toBe(false)

    await lease.close()
    releaseWorkspace()
  })

  it('registers the same eager workspace intent only once when lazy demand mounts', async () => {
    const intent = {}
    const resumeDemand = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => fakeSession()),
      resumeDemand,
    })

    manager.beginUserConnectionIntent(intent)
    const releaseWorkspace = manager.retainConnectionDemand(intent)
    const lease = await manager.get()

    expect(manager.latestUserResumeIntent()).toBe(intent)
    expect(resumeDemand).toHaveBeenCalledTimes(1)
    expect(resumeDemand).toHaveBeenCalledWith(intent)
    await lease.close()
    releaseWorkspace()
    await manager.reset()
  })

  it('does not create resume history for passive demand without an intent', () => {
    const createResumeIntent = vi.fn(() => ({}))
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => fakeSession()),
      createResumeIntent,
    })

    const releaseDemand = manager.retainConnectionDemand()

    expect(createResumeIntent).not.toHaveBeenCalled()
    expect(manager.latestUserResumeIntent()).toBeNull()
    releaseDemand()
  })

  it('adopts a runtime Stop synchronously and only a fresh intent resumes passive demand', async () => {
    const connect = vi.fn(async () => fakeSession())
    const resumeDemand = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      resumeDemand,
      disconnect: vi.fn(async () => undefined),
    })
    const releasePassiveDemand = manager.retainConnectionDemand()

    const adoption = manager.adoptUserStop()
    await expect(manager.get()).rejects.toMatchObject({ code: 'user_stopped', retryable: false })

    const freshIntent = {}
    const releaseFreshDemand = manager.retainConnectionDemand(freshIntent)
    const lease = await manager.get()
    await adoption

    expect(resumeDemand).toHaveBeenCalledWith(freshIntent)
    expect(connect).toHaveBeenCalledOnce()
    await lease.close()
    releasePassiveDemand()
    releaseFreshDemand()
    await manager.reset()
  })

  it('keeps the user-stop recovery state across offline, foreground, and renderer generations', async () => {
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      disconnect: vi.fn(async () => undefined),
    })
    const expectUserStopped = () => expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'failed',
      error: { code: 'user_stopped', retryable: false },
    })

    await manager.disconnect()
    expectUserStopped()

    await manager.networkChanged(false, 'offline')
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'waiting_network' })
    await manager.networkChanged(true, 'available')
    expectUserStopped()

    await manager.foregroundResume()
    expectUserStopped()
    await manager.resetBindingGeneration()
    expectUserStopped()
    expect(connect).not.toHaveBeenCalled()
  })

  it('adopts a process-owned Stop in a fresh manager and resumes it with one new intent', async () => {
    let stopped = true
    const connect = vi.fn(async () => fakeSession())
    const setActive = vi.fn(async (_machineId: string, active: boolean) => {
      if (active && stopped) {
        throw Object.assign(new Error('Native demand is stopped'), {
          code: 'user_stopped',
          retryable: false,
        })
      }
    })
    const resumeDemand = vi.fn(async () => { stopped = false })
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      setActive,
      resumeDemand,
      createResumeIntent: () => ({}),
    })
    const releaseWorkspace = manager.retainConnectionDemand(null)

    await expect(manager.get()).rejects.toMatchObject({ code: 'user_stopped', retryable: false })
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'failed',
      error: { code: 'user_stopped', retryable: false },
    })

    await manager.foregroundResume()
    expect(setActive).toHaveBeenCalledTimes(1)
    expect(connect).not.toHaveBeenCalled()

    await manager.resetClientOnly()
    expect(resumeDemand).toHaveBeenCalledOnce()
    expect(setActive).toHaveBeenCalledTimes(2)
    expect(connect).toHaveBeenCalledOnce()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })

    releaseWorkspace()
    await manager.reset()
  })

  it('does not let a stale stopped resume response overwrite a newer successful intent', async () => {
    const staleResume = deferred<void>()
    const resumeDemand = vi.fn()
      .mockImplementationOnce(async () => await staleResume.promise)
      .mockResolvedValueOnce(undefined)
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      resumeDemand,
      setActive: vi.fn(async () => undefined),
      createResumeIntent: () => ({}),
    })
    const releaseWorkspace = manager.retainConnectionDemand(null)

    const staleRecovery = manager.resetClientOnly()
    await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledOnce())
    const currentRecovery = manager.resetClientOnly()
    await expect(currentRecovery).resolves.toBeUndefined()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })

    staleResume.reject(Object.assign(new Error('Superseded Stop response'), {
      code: 'user_stopped',
      retryable: false,
    }))
    await expect(staleRecovery).rejects.toMatchObject({ code: 'cancelled' })
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    expect(connect).toHaveBeenCalledOnce()

    releaseWorkspace()
    await manager.reset()
  })

  it('replays active endpoint demand after every explicit workspace resume', async () => {
    const connect = vi.fn(async () => fakeSession())
    const setActive = vi.fn(async (_machineId: string, _active: boolean) => undefined)
    const resumeDemand = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, setActive, resumeDemand })
    const releaseFirstWorkspace = manager.retainConnectionDemand({})
    const first = await manager.get()
    await first.close()

    const releaseFreshWorkspace = manager.retainConnectionDemand({})
    const second = await manager.get()

    expect(resumeDemand).toHaveBeenCalledTimes(2)
    expect(setActive.mock.calls.filter(([, active]) => active === true)).toEqual([
      ['daemon-a', true],
      ['daemon-a', true],
    ])

    await second.close()
    releaseFirstWorkspace()
    releaseFreshWorkspace()
    await manager.reset()
  })

  it('lets one manual retry resume an already mounted workspace after user stop', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const setActive = vi.fn(async () => undefined)
    const resumeDemand = vi.fn(async () => undefined)
    const requestRecovery = vi.fn(async () => undefined)
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect,
      setActive,
      resumeDemand,
      requestRecovery,
      disconnect,
    })
    const releaseWorkspace = manager.retainConnectionDemand({})
    const oldLease = await manager.get()

    await manager.disconnect()
    await manager.resetClientOnly()

    expect(oldLease.isAlive()).toBe(false)
    expect(resumeDemand).toHaveBeenCalledTimes(2)
    expect(requestRecovery).toHaveBeenCalledOnce()
    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })

    releaseWorkspace()
    await manager.reset()
  })

  it('surfaces an explicit physical disconnect failure', async () => {
    const failure = new Error('native disconnect failed')
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => fakeSession()),
      disconnect: vi.fn(async () => { throw failure }),
    })
    const lease = await manager.get()

    await expect(manager.disconnect()).rejects.toBe(failure)
    await lease.close()
  })

  it('does not expose Go transport connected until OpenSession resolves the renderer binding', async () => {
    const open = deferred<ProtoClientSession>()
    let options: RtcConnectOptions | undefined
    const connect = vi.fn((_input: { machineId: string }, inputOptions?: RtcConnectOptions) => {
      options = inputOptions
      return open.promise
    })
    const manager = new NativeSessionManager('daemon-a', { connect })
    const opening = manager.get()
    await vi.waitFor(() => expect(connect).toHaveBeenCalledOnce())

    options?.onConnectionState?.({
      machineId: 'daemon-a',
      phase: 'connected',
      statusText: 'Transport ready',
      relayInUse: false,
    })
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'reconnecting' })

    open.resolve(fakeSession())
    const lease = await opening
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    await lease.close()
  })

  it('does not reconnect a non-retryable asynchronous close', async () => {
    vi.useFakeTimers()
    const session = fakeSession()
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const releaseWorkspace = manager.retainConnectionDemand()
    const lease = await manager.get()
    const failure = Object.assign(new Error('Authorization revoked'), {
      code: 'authorization_revoked',
      retryable: false,
    })

    try {
      session.fail(failure)
      await vi.runAllTimersAsync()

      expect(connect).toHaveBeenCalledOnce()
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'failed', error: failure })
    } finally {
      await lease.close()
      releaseWorkspace()
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('lets one consumer cancel its wait without cancelling the shared binding', async () => {
    const open = deferred<ProtoClientSession>()
    const connect = vi.fn(() => open.promise)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const controller = new AbortController()

    const cancelled = manager.get({ signal: controller.signal })
    const retained = manager.get()
    await vi.waitFor(() => expect(connect).toHaveBeenCalledOnce())
    controller.abort(new Error('picker closed'))
    await expect(cancelled).rejects.toThrow('picker closed')

    open.resolve(fakeSession())
    await expect(retained).resolves.toMatchObject({ stamp: { generation: 1n } })
    expect(connect).toHaveBeenCalledOnce()
    await manager.reset()
  })

  it('does not wait for a stale pending binding during generation reset', async () => {
    const open = deferred<ProtoClientSession>()
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(() => open.promise) })
    void manager.get().catch(() => undefined)
    await Promise.resolve()

    await expect(manager.resetBindingGeneration()).resolves.toBeUndefined()

    const stale = fakeSession()
    open.resolve(stale)
    await vi.waitFor(() => expect(stale.close).toHaveBeenCalledOnce())
    expect(stale.invalidate).not.toHaveBeenCalled()
    await manager.reset()
  })

  it('keeps resource cleanup alive across a UI lease close', async () => {
    const commandGate = deferred<ResultEnvelope>()
    const session = fakeSession()
    session.execute = vi.fn(() => commandGate.promise)
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })
    const lease = await manager.get()
    const cleanup = lease.execute(create(CommandEnvelopeSchema, {
      command: {
        case: 'releaseResource',
        value: create(ReleaseResourceCommandSchema, {
          resource: create(ResourceHandleSchema, { opaqueToken: new Uint8Array([1]) }),
        }),
      },
    }))

    await lease.close()
    expect(session.close).not.toHaveBeenCalled()
    commandGate.resolve({} as ResultEnvelope)
    await cleanup
    await vi.waitFor(() => expect(session.close).toHaveBeenCalledOnce())
  })

  it('force-releases demand when a manager fence overtakes stalled cleanup', async () => {
    const commandGate = deferred<ResultEnvelope>()
    const session = fakeSession()
    session.execute = vi.fn(() => commandGate.promise)
    const setActive = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => session),
      setActive,
    })
    const lease = await manager.get()
    void lease.execute(create(CommandEnvelopeSchema, {
      command: {
        case: 'releaseResource',
        value: create(ReleaseResourceCommandSchema, {
          resource: create(ResourceHandleSchema, { opaqueToken: new Uint8Array([1]) }),
        }),
      },
    })).catch(() => undefined)

    await lease.close()
    expect(manager.hasConnectionDemand()).toBe(true)
    await manager.resetBindingGeneration()

    await vi.waitFor(() => expect(setActive).toHaveBeenLastCalledWith('daemon-a', false))
    expect(manager.hasConnectionDemand()).toBe(false)
    expect(session.close).toHaveBeenCalledOnce()
  })

  it('bounds cleanup ownership when no later lifecycle fence arrives', async () => {
    vi.useFakeTimers()
    const session = fakeSession()
    session.execute = vi.fn(() => new Promise<ResultEnvelope>(() => {}))
    const setActive = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => session),
      setActive,
    })
    const releaseWorkspace = manager.retainConnectionDemand()

    try {
      const lease = await manager.get()
      void lease.execute(create(CommandEnvelopeSchema, {
        command: {
          case: 'releaseResource',
          value: create(ReleaseResourceCommandSchema, {
            resource: create(ResourceHandleSchema, { opaqueToken: new Uint8Array([1]) }),
          }),
        },
      })).catch(() => undefined)

      await lease.close()
      releaseWorkspace()
      await vi.advanceTimersByTimeAsync(4_999)
      expect(setActive).not.toHaveBeenCalledWith('daemon-a', false)

      await vi.advanceTimersByTimeAsync(1)
      expect(setActive).toHaveBeenLastCalledWith('daemon-a', false)
      expect(manager.hasConnectionDemand()).toBe(false)
      expect(session.close).toHaveBeenCalledOnce()
    } finally {
      await manager.reset().catch(() => undefined)
      vi.useRealTimers()
    }
  })

  it('projects route facts from the resolved renderer binding', async () => {
    const connection = create(ConnectionSnapshotSchema, {
      routeId: 'relay-route',
      routeKind: ConnectionRouteKind.CLOUD,
      observedPath: ConnectionObservedPath.SINGLE_RELAY,
    })
    const session = fakeSession(7n, connection)
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })

    const lease = await manager.get()

    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      relayInUse: true,
      connectionInfo: {
        generation: 7n,
        routeKind: 'cloud',
        observedPath: 'single_relay',
        type: 'relay',
      },
    })
    await lease.close()
  })

  it('forwards live connection snapshot sampling through the renderer lease', async () => {
    const session = fakeSession()
    session.getConnectionSnapshot = vi.fn()
      .mockResolvedValueOnce(create(ConnectionSnapshotSchema, { sampledAtUnixNano: 10n, bytesSent: 20n }))
      .mockResolvedValueOnce(create(ConnectionSnapshotSchema, { sampledAtUnixNano: 30n, bytesSent: 40n }))
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })
    const lease = await manager.get()

    await expect(lease.getConnectionSnapshot?.()).resolves.toMatchObject({ sampledAtUnixNano: 10n, bytesSent: 20n })
    await expect(lease.getConnectionSnapshot?.()).resolves.toMatchObject({ sampledAtUnixNano: 30n, bytesSent: 40n })
    expect(session.getConnectionSnapshot).toHaveBeenCalledTimes(2)
    await lease.close()
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

async function pendingManualRecoveryHarness() {
  const resumeGate = deferred<void>()
  const first = fakeSession()
  const recovered = fakeSession(2n)
  const connect = vi.fn()
    .mockResolvedValueOnce(first)
    .mockResolvedValueOnce(recovered)
  const resumeDemand = vi.fn()
    .mockResolvedValueOnce(undefined)
    .mockImplementationOnce(async () => await resumeGate.promise)
  const requestRecovery = vi.fn(async () => undefined)
  const manager = new NativeSessionManager('daemon-a', {
    connect,
    resumeDemand,
    requestRecovery,
    setActive: vi.fn(async () => undefined),
    disconnect: vi.fn(async () => undefined),
  })
  const releaseWorkspace = manager.retainConnectionDemand({})
  await manager.get()
  await manager.disconnect()

  const recovery = manager.resetClientOnly()
  await vi.waitFor(() => expect(resumeDemand).toHaveBeenCalledTimes(2))
  return { manager, connect, requestRecovery, recovery, releaseWorkspace, resumeGate }
}

function fakeSession(generation = 1n, connection?: ConnectionSnapshot): ProtoClientSession & {
  markDead(): void
  fail(error: ProtoClientSessionCloseError): void
} {
  let alive = true
  const closeHandlers = new Set<ProtoClientSessionCloseHandler>()
  return {
    stamp: create(EndpointSessionStampSchema, { endpointId: 'daemon-a', routeId: 'direct', generation }),
    ...(connection ? { connection } : {}),
    execute: vi.fn(async (_command: CommandEnvelope) => create(ResultEnvelopeSchema)),
    subscribeEvents: vi.fn((_handler: (event: EventEnvelope) => void): ProtoClientSubscription => ({ close() {} })),
    subscribeClosed: vi.fn((handler): ProtoClientSubscription => {
      closeHandlers.add(handler)
      return { close: () => { closeHandlers.delete(handler) } }
    }),
    openResourceStream: vi.fn(async (_resource: ResourceHandle): Promise<ProtoResourceStream> => { throw new Error('unused') }),
    isAlive: () => alive,
    invalidate: vi.fn(async () => {
      alive = false
      closeHandlers.clear()
    }),
    close: vi.fn(async () => {
      alive = false
      closeHandlers.clear()
    }),
    markDead: () => { alive = false },
    fail: (error) => {
      if (!alive) return
      alive = false
      const handlers = [...closeHandlers]
      closeHandlers.clear()
      handlers.forEach((handler) => handler(error))
    },
  }
}
