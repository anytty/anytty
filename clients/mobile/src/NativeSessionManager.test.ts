import { create } from '@bufbuild/protobuf'
import { describe, expect, it, vi } from 'vitest'
import type { ProtoClientSession, ProtoClientSubscription, ProtoResourceStream } from '@anytty/ui'
import { EndpointSessionStampSchema, type ResourceHandle } from '../../ui/src/generated/apipb/common_pb'
import { CommandEnvelopeSchema, type CommandEnvelope, type EventEnvelope, type ResultEnvelope } from '../../ui/src/generated/apipb/application_pb'
import { ConnectionCandidateType, ConnectionObservedPath, ConnectionRouteKind, ConnectionSnapshotSchema, type ConnectionSnapshot } from '../../ui/src/generated/bindingpb/client_binding_pb'
import { NativeSessionManager } from './NativeSessionManager'

type ProtoClientSessionCloseHandler = Parameters<ProtoClientSession['subscribeClosed']>[0]
type ProtoClientSessionCloseError = Parameters<ProtoClientSessionCloseHandler>[0]

describe('NativeSessionManager', () => {
  it('starts idle and retains a session only after an explicit probe', async () => {
    const session = fakeSession()
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect })

    expect(manager.connectionState.getSnapshot()).toMatchObject({
      machineId: 'daemon-a',
      phase: 'idle',
      connectionInfo: null,
    })

    await manager.probe()

    expect(connect).toHaveBeenCalledOnce()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    expect(session.close).not.toHaveBeenCalled()
  })

  it('seeds cold-start offline state without refreshing the first online session', async () => {
    const session = fakeSession()
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect })

    await manager.initializeNetworkState(false)
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'waiting_network' })
    await expect(manager.get()).rejects.toMatchObject({ code: 'network_offline' })

    await manager.initializeNetworkState(true)
    await manager.get()
    expect(connect).toHaveBeenCalledOnce()
  })

  it('keeps a remotely verified pooled Go session on first acquire', async () => {
    const pooled = fakeSession()
    const connect = vi.fn(async () => pooled)
    const verify = vi.fn(async () => {})
    const manager = new NativeSessionManager('daemon-a', { connect, verify }, { verifyOnFirstAcquire: true })

    const lease = await manager.get()

    expect(connect).toHaveBeenCalledOnce()
    expect(verify).toHaveBeenCalledWith(pooled, expect.any(AbortSignal))
    expect(pooled.invalidate).not.toHaveBeenCalled()
    expect(lease.stamp.generation).toBe(1n)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 1n },
    })
    await manager.reset()
  })

  it('replaces a pooled Go session only when first-acquire verification fails', async () => {
    const stale = fakeSession()
    const current = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockResolvedValueOnce(current)
    const verify = vi.fn(async () => { throw new Error('stale network path') })
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    await manager.networkChanged()
    const lease = await manager.get()

    expect(connect).toHaveBeenCalledTimes(2)
    expect(verify).toHaveBeenCalledOnce()
    expect(stale.invalidate).toHaveBeenCalledTimes(1)
    expect(lease.stamp.generation).toBe(2n)
    await manager.reset()
  })

  it('shares one Go session across independent UI leases', async () => {
    const session = fakeSession()
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect })

    const workspace = await manager.lease()
    const transfer = await manager.get()

    expect(connect).toHaveBeenCalledTimes(1)
    await transfer.close()
    expect(workspace.isAlive()).toBe(true)
    expect(session.close).not.toHaveBeenCalled()

    await manager.reset()
    expect(workspace.isAlive()).toBe(false)
    expect(session.close).toHaveBeenCalledTimes(1)
  })

  it('reuses the device DataChannel after every terminal UI lease has closed', async () => {
    const session = fakeSession()
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect })

    const inventory = await manager.get()
    const firstTerminal = await manager.lease()
    await inventory.close()
    await firstTerminal.close()

    const reopenedTerminal = await manager.lease()

    expect(connect).toHaveBeenCalledTimes(1)
    expect(reopenedTerminal.isAlive()).toBe(true)
    expect(session.close).not.toHaveBeenCalled()
  })

  it('projects a pooled relay connection until it is explicitly reset', async () => {
    const session = fakeSession(1n, create(ConnectionSnapshotSchema, {
      routeId: 'cloud-primary',
      routeKind: ConnectionRouteKind.CLOUD,
      observedPath: ConnectionObservedPath.SINGLE_RELAY,
      localCandidateType: ConnectionCandidateType.RELAY,
      connected: true,
    }))
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })
    const listener = vi.fn()
    manager.connectionState.subscribe(listener)

    const workspace = await manager.lease()
    await workspace.close()

    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      relayInUse: true,
      connectionInfo: { path: 'hub', observedPath: 'single_relay' },
    })
    expect(session.close).not.toHaveBeenCalled()

    await manager.reset()

    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'idle', relayInUse: false })
    expect(session.close).toHaveBeenCalledTimes(1)
    expect(listener).toHaveBeenCalled()
  })

  it('does not report relay merely because a non-selected relay candidate exists', async () => {
    const session = fakeSession(1n, create(ConnectionSnapshotSchema, {
      routeKind: ConnectionRouteKind.DIRECT,
      observedPath: ConnectionObservedPath.DIRECT,
      localCandidateType: ConnectionCandidateType.RELAY,
      connected: true,
    }))
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })

    await manager.get({ forceRelay: true })

    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      forceRelay: true,
      relayInUse: false,
      connectionInfo: { observedPath: 'direct', relayInUse: false, type: 'p2p' },
    })
  })

  it('does not report a reconnect phase when a new UI lease reuses the pooled session', async () => {
    const session = fakeSession(1n, create(ConnectionSnapshotSchema, {
      routeId: 'direct-primary',
      routeKind: ConnectionRouteKind.DIRECT,
      observedPath: ConnectionObservedPath.DIRECT,
      connected: true,
    }))
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const initial = await manager.get()
    await initial.close()
    const phases: string[] = []

    const reopened = await manager.get({ onConnectionState: (snapshot) => phases.push(snapshot.phase) })

    expect(connect).toHaveBeenCalledTimes(1)
    expect(reopened.isAlive()).toBe(true)
    expect(phases.length).toBeGreaterThan(0)
    expect(phases.every((phase) => phase === 'connected')).toBe(true)
  })

  it('opens a fresh Go session after the owned session becomes stale', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const manager = new NativeSessionManager('daemon-a', { connect })

    await manager.get()
    first.markDead()
    const current = await manager.get()

    expect(connect).toHaveBeenCalledTimes(2)
    expect(current.stamp.generation).toBe(2n)
  })

  it('evicts an asynchronously closed session and preserves its structured failure', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const manager = new NativeSessionManager('daemon-a', { connect })
    await manager.get()
    const failure = Object.assign(new Error('daemon blocked'), {
      code: 'daemon_blocked',
      retryable: true,
    })

    first.fail(failure)

    expect(connect).toHaveBeenCalledTimes(1)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'failed',
      statusText: 'daemon blocked',
      error: { message: 'daemon blocked', code: 'daemon_blocked', retryable: true },
    })
    expect(manager.connectionState.getSnapshot().error).toBe(failure)

    const current = await manager.get()
    expect(connect).toHaveBeenCalledTimes(2)
    expect(current.stamp.generation).toBe(2n)
  })

  it('automatically reconnects a retryable asynchronously closed session', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const manager = new NativeSessionManager('daemon-a', { connect })

    try {
      await manager.get()
      first.fail(Object.assign(new Error('network handoff'), { retryable: true }))
      await vi.advanceTimersByTimeAsync(0)

      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('reconnects immediately once, then jitters repeated failures', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const recovered = fakeSession(3n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockRejectedValueOnce(Object.assign(new Error('route still unavailable'), { retryable: true }))
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect }, { random: () => 0 })

    try {
      await manager.get()
      first.fail(Object.assign(new Error('transport unavailable'), { retryable: true }))
      await vi.advanceTimersByTimeAsync(0)

      expect(connect).toHaveBeenCalledTimes(2)
      await vi.advanceTimersByTimeAsync(249)
      expect(connect).toHaveBeenCalledTimes(2)
      await vi.advanceTimersByTimeAsync(1)
      expect(connect).toHaveBeenCalledTimes(3)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('rebuilds only the endpoint session after a confirmed network change', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const setActive = vi.fn(async () => {})
    const manager = new NativeSessionManager('daemon-a', { connect, setActive })

    const lease = await manager.get()
    await lease.close()
    await manager.networkChanged()

    expect(connect).toHaveBeenCalledTimes(2)
    expect(first.invalidate).toHaveBeenCalledTimes(1)
    expect(first.close).toHaveBeenCalledTimes(1)
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    expect(setActive).toHaveBeenCalledTimes(1)
    expect(setActive).toHaveBeenCalledWith('daemon-a', true)
    await manager.reset()
  })

  it('publishes a reconnecting state before replacing a session on network replacement', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const manager = new NativeSessionManager('daemon-a', { connect })

    await manager.get()
    const phases: string[] = []
    manager.connectionState.subscribe(() => phases.push(manager.connectionState.getSnapshot().phase))
    await manager.networkChanged(true, 'network_replaced')

    expect(connect).toHaveBeenCalledTimes(2)
    expect(phases).toContain('reconnecting')
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    await manager.reset()
  })

  it('keeps the same generation when a changed network still carries the application session', async () => {
    const session = fakeSession()
    const verify = vi.fn(async () => {})
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    await manager.get()
    const phases: string[] = []
    manager.connectionState.subscribe(() => phases.push(manager.connectionState.getSnapshot().phase))
    await manager.networkChanged(true, 'path_changed')

    expect(verify).toHaveBeenCalledWith(session, expect.any(AbortSignal))
    expect(connect).toHaveBeenCalledOnce()
    expect(session.invalidate).not.toHaveBeenCalled()
    expect(phases).toContain('verifying')
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 1n },
    })
    await manager.reset()
  })

  it('invalidates offline, waits without reconnecting, and recovers once online', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const recovered = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect })

    try {
      const lease = await manager.get()
      await lease.close()
      await manager.networkChanged(false)

      expect(first.invalidate).toHaveBeenCalledTimes(1)
      expect(first.close).toHaveBeenCalledTimes(1)
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'waiting_network',
        statusText: 'Waiting for network...',
        connectionInfo: null,
        error: null,
      })

      await expect(manager.get()).rejects.toMatchObject({
        code: 'network_offline',
        retryable: true,
      })
      await manager.foregroundResume()
      await vi.advanceTimersByTimeAsync(60_000)
      expect(connect).toHaveBeenCalledTimes(1)

      await manager.networkChanged(true)

      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'connected',
        connectionInfo: { generation: 2n },
      })
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('coalesces rapid offline and available callbacks into one final recovery', async () => {
    const first = fakeSession()
    const recovered = fakeSession(2n)
    let finishInvalidation!: () => void
    const invalidate = vi.fn(() => new Promise<void>((resolve) => { finishInvalidation = resolve }))
    first.invalidate = invalidate
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect })

    await manager.get()
    const recovery = manager.networkChanged(false, 'offline')
    for (let turn = 0; turn < 10 && invalidate.mock.calls.length === 0; turn += 1) await Promise.resolve()
    expect(invalidate).toHaveBeenCalledOnce()

    manager.networkChanged(true, 'available')
    manager.networkChanged(false, 'offline')
    manager.networkChanged(true, 'available')
    finishInvalidation()
    await recovery

    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })
    await manager.reset()
  })

  it('aborts an in-flight acquire offline without publishing failure or arming a retry', async () => {
    vi.useFakeTimers()
    const recovered = fakeSession(2n)
    let connectingSignal: AbortSignal | undefined
    const connect = vi.fn()
      .mockImplementationOnce((_input, options) => {
        connectingSignal = options?.signal
        return new Promise<ProtoClientSession>((_resolve, reject) => {
          options?.signal?.addEventListener('abort', () => reject(options.signal?.reason), { once: true })
        })
      })
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect })
    const pending = manager.get()

    try {
      await manager.networkChanged(false)
      await expect(pending).rejects.toMatchObject({ message: 'native session generation changed while connecting' })

      expect(connectingSignal?.aborted).toBe(true)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'waiting_network', error: null })
      await vi.advanceTimersByTimeAsync(60_000)
      expect(connect).toHaveBeenCalledTimes(1)

      await manager.networkChanged(true)

      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('cancels an already queued reconnect when the phone goes offline', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const recovered = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect })

    try {
      await manager.get()
      first.fail(Object.assign(new Error('transport unavailable'), { retryable: true }))
      await manager.networkChanged(false)
      await vi.advanceTimersByTimeAsync(60_000)

      expect(connect).toHaveBeenCalledTimes(1)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'waiting_network', error: null })

      await manager.networkChanged(true)
      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('blocks the first native acquire when constructed offline', async () => {
    const connect = vi.fn(async () => fakeSession())
    const manager = new NativeSessionManager(
      'daemon-a',
      { connect },
      { initiallyConnected: false },
    )

    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'waiting_network' })
    await expect(manager.get()).rejects.toMatchObject({ code: 'network_offline' })
    expect(connect).not.toHaveBeenCalled()

    await manager.networkChanged(true)

    expect(connect).toHaveBeenCalledOnce()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    await manager.reset()
  })

  it('keeps a remotely verified session on foreground resume', async () => {
    const session = fakeSession()
    const verify = vi.fn(async () => {})
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    await manager.get()
    const phases: string[] = []
    manager.connectionState.subscribe(() => phases.push(manager.connectionState.getSnapshot().phase))
    await manager.foregroundResume()

    expect(verify).toHaveBeenCalledWith(session, expect.any(AbortSignal))
    expect(connect).toHaveBeenCalledTimes(1)
    expect(phases).not.toContain('verifying')
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    await manager.reset()
  })

  it('keeps a responsive session when the verification RPC returns a typed business error', async () => {
    const session = fakeSession()
    const verify = vi.fn(async () => {
      throw Object.assign(new Error('machine events are not allowed'), { code: 'forbidden' })
    })
    const connect = vi.fn(async () => session)
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    await manager.get()
    await manager.foregroundResume()

    expect(connect).toHaveBeenCalledOnce()
    expect(session.invalidate).not.toHaveBeenCalled()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    await manager.reset()
  })

  it('replaces an alive session when verification reports transport unavailable', async () => {
    const stale = fakeSession()
    const recovered = fakeSession(2n)
    const verify = vi.fn(async () => {
      throw Object.assign(new Error('transport EOF'), { code: 'unavailable' })
    })
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    await manager.get()
    await manager.foregroundResume()

    expect(stale.invalidate).toHaveBeenCalledOnce()
    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })
    await manager.reset()
  })

  it('replaces a session whose remote foreground verification fails', async () => {
    const stale = fakeSession()
    const recovered = fakeSession(2n)
    const verify = vi.fn(async () => { throw new Error('stale remote transport') })
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    await manager.get()
    await manager.foregroundResume()

    expect(stale.invalidate).toHaveBeenCalledTimes(1)
    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })
    await manager.reset()
  })

  it('bounds foreground verification before replacing an unresponsive session', async () => {
    vi.useFakeTimers()
    const stale = fakeSession()
    const recovered = fakeSession(2n)
    let probeSignal: AbortSignal | undefined
    const verify = vi.fn((_session: ProtoClientSession, signal: AbortSignal) => {
      probeSignal = signal
      return new Promise<void>((_resolve, reject) => {
        signal.addEventListener('abort', () => reject(signal.reason), { once: true })
      })
    })
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    try {
      await manager.get()
      const resume = manager.foregroundResume()
      await vi.advanceTimersByTimeAsync(2_999)
      expect(probeSignal?.aborted).toBe(false)
      await vi.advanceTimersByTimeAsync(1)
      await resume

      expect(probeSignal?.aborted).toBe(true)
      expect(stale.invalidate).toHaveBeenCalledTimes(1)
      expect(connect).toHaveBeenCalledTimes(2)
      expect(manager.connectionState.getSnapshot()).toMatchObject({
        phase: 'connected',
        connectionInfo: { generation: 2n },
      })
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('aborts a superseded network acquire and reports the coalesced latest recovery', async () => {
    vi.useFakeTimers()
    const first = fakeSession()
    const recovered = fakeSession(2n)
    const latestFailure = Object.assign(new Error('new network is still settling'), { retryable: true })
    let supersededSignal: AbortSignal | undefined
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockImplementationOnce((_input, options) => {
        supersededSignal = options?.signal
        return new Promise<ProtoClientSession>((_resolve, reject) => {
          options?.signal?.addEventListener('abort', () => reject(options.signal?.reason), { once: true })
        })
      })
      .mockRejectedValueOnce(latestFailure)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect })

    try {
      await manager.get()
      const superseded = manager.networkChanged(true, 'network_replaced')
      for (let turn = 0; turn < 10 && connect.mock.calls.length < 2; turn += 1) await Promise.resolve()
      expect(connect).toHaveBeenCalledTimes(2)

      const latest = manager.networkChanged(true, 'network_replaced')
      await expect(superseded).rejects.toBe(latestFailure)
      expect(supersededSignal?.aborted).toBe(true)
      await expect(latest).rejects.toBe(latestFailure)

      await vi.advanceTimersByTimeAsync(0)
      expect(connect).toHaveBeenCalledTimes(4)
      expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    } finally {
      await manager.reset()
      vi.useRealTimers()
    }
  })

  it('coalesces path metadata changes while a connection is pending and verifies before publishing it', async () => {
    let resolveConnect!: (session: ProtoClientSession) => void
    let connectSignal: AbortSignal | undefined
    const session = fakeSession()
    const connect = vi.fn((_input, options) => {
      connectSignal = options?.signal
      return new Promise<ProtoClientSession>((resolve) => { resolveConnect = resolve })
    })
    const verify = vi.fn(async () => {})
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    const acquiring = manager.get()
    const changed = manager.networkChanged(true, 'path_changed')
    resolveConnect(session)

    await expect(acquiring).resolves.toMatchObject({ stamp: { generation: 1n } })
    await changed
    expect(connectSignal?.aborted).toBe(false)
    expect(connect).toHaveBeenCalledOnce()
    expect(verify).toHaveBeenCalledOnce()
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })
    await manager.reset()
  })

  it('waits for exact-generation invalidation before a concurrent acquire can reconnect', async () => {
    const first = fakeSession()
    const recovered = fakeSession(2n)
    let finishInvalidation!: () => void
    const invalidation = new Promise<void>((resolve) => { finishInvalidation = resolve })
    const invalidate = vi.fn(() => invalidation)
    first.invalidate = invalidate
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect })

    await manager.get()
    const replacement = manager.networkChanged(true, 'network_replaced')
    for (let turn = 0; turn < 10 && invalidate.mock.calls.length === 0; turn += 1) await Promise.resolve()
    expect(invalidate).toHaveBeenCalledOnce()

    const concurrent = manager.get()
    await Promise.resolve()
    expect(connect).toHaveBeenCalledOnce()

    finishInvalidation()
    await replacement
    await expect(concurrent).resolves.toMatchObject({ stamp: { generation: 2n } })
    expect(connect).toHaveBeenCalledTimes(2)
    await manager.reset()
  })

  it('does not revive a replacement after reset wins during invalidation', async () => {
    const first = fakeSession()
    let finishInvalidation!: () => void
    const invalidation = new Promise<void>((resolve) => { finishInvalidation = resolve })
    const invalidate = vi.fn(() => invalidation)
    first.invalidate = invalidate
    const connect = vi.fn(async () => first)
    const setActive = vi.fn(async () => {})
    const manager = new NativeSessionManager('daemon-a', { connect, setActive })

    await manager.get()
    const replacement = manager.networkChanged(true, 'network_replaced')
    for (let turn = 0; turn < 10 && invalidate.mock.calls.length === 0; turn += 1) await Promise.resolve()
    expect(invalidate).toHaveBeenCalledOnce()

    await manager.reset()
    finishInvalidation()
    await replacement

    expect(connect).toHaveBeenCalledOnce()
    expect(setActive).toHaveBeenLastCalledWith('daemon-a', false)
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'idle' })
  })

  it('supersedes an old path verification when the network identity changes', async () => {
    const stale = fakeSession()
    const recovered = fakeSession(2n)
    let finishOldVerification!: () => void
    const oldVerification = new Promise<void>((resolve) => { finishOldVerification = resolve })
    let oldVerificationSignal: AbortSignal | undefined
    const verify = vi.fn((_session: ProtoClientSession, signal: AbortSignal) => {
      oldVerificationSignal = signal
      return oldVerification
    })
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockResolvedValueOnce(recovered)
    const manager = new NativeSessionManager('daemon-a', { connect, verify })

    await manager.get()
    const checkingOldPath = manager.networkChanged(true, 'path_changed')
    await Promise.resolve()
    const replaced = manager.networkChanged(true, 'network_replaced')
    expect(oldVerificationSignal?.aborted).toBe(true)

    finishOldVerification()
    await checkingOldPath
    await replaced
    expect(stale.invalidate).toHaveBeenCalledOnce()
    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })
    await manager.reset()
  })

  it('does not publish a pooled session when its first verification is interrupted by going offline', async () => {
    const pooled = fakeSession()
    const recovered = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(pooled)
      .mockResolvedValueOnce(recovered)
    let firstVerifySignal: AbortSignal | undefined
    const verify = vi.fn()
      .mockImplementationOnce((_session: ProtoClientSession, signal: AbortSignal) => {
        firstVerifySignal = signal
        return new Promise<void>((_resolve, reject) => {
          signal.addEventListener('abort', () => reject(signal.reason), { once: true })
        })
      })
      .mockResolvedValueOnce(undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, verify }, { verifyOnFirstAcquire: true })
    const acquiring = manager.get()

    await Promise.resolve()
    await manager.networkChanged(false)
    await expect(acquiring).rejects.toMatchObject({ message: 'native session generation changed while connecting' })
    expect(firstVerifySignal?.aborted).toBe(true)
    expect(pooled.invalidate).toHaveBeenCalledTimes(1)

    await manager.networkChanged(true)
    expect(connect).toHaveBeenCalledTimes(2)
    expect(verify).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })
    await manager.reset()
  })

  it('does not let delayed stale-session cleanup replace a recovered network session', async () => {
    const stale = fakeSession()
    const recovered = fakeSession(2n)
    let finishStaleClose!: () => void
    const staleClose = new Promise<void>((resolve) => { finishStaleClose = resolve })
    stale.close = vi.fn(() => staleClose)
    const connect = vi.fn()
      .mockResolvedValueOnce(stale)
      .mockResolvedValueOnce(recovered)
      .mockResolvedValueOnce(fakeSession(3n))
    const manager = new NativeSessionManager('daemon-a', { connect })

    await manager.get()
    stale.markDead()
    const waitingForCleanup = manager.get()
    const networkRecovery = manager.networkChanged()
    await networkRecovery
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'connected' })

    finishStaleClose()
    await expect(waitingForCleanup).rejects.toMatchObject({ message: 'native session generation changed while connecting' })

    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot()).toMatchObject({
      phase: 'connected',
      connectionInfo: { generation: 2n },
    })
    await manager.reset()
  })

  it('keeps the foreground service active until an explicit reset', async () => {
    const setActive = vi.fn(async () => {})
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => fakeSession()),
      setActive,
    })

    await manager.get()
    expect(setActive).toHaveBeenCalledWith('daemon-a', true)

    await manager.reset()
    expect(setActive).toHaveBeenLastCalledWith('daemon-a', false)
  })

  it('keeps the foreground service active while an explicit reconnect replaces the session', async () => {
    const session = fakeSession()
    const setActive = vi.fn(async () => {})
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => session),
      setActive,
    })

    await manager.get()
    await manager.resetClientOnly()

    expect(session.invalidate).toHaveBeenCalledTimes(1)
    expect(setActive).toHaveBeenCalledTimes(1)
    expect(setActive).toHaveBeenCalledWith('daemon-a', true)
    await manager.reset()
  })

  it('keeps structured connection failures in connection-state callbacks', async () => {
    const failure = Object.assign(new Error('daemon deleted'), {
      code: 'daemon_deleted',
      retryable: false,
    })
    const connect = vi.fn(async (_input, options) => {
      options?.onConnectionState?.({
        machineId: 'daemon-a',
        phase: 'failed',
        statusText: failure.message,
        relayInUse: false,
        error: failure,
      })
      throw failure
    })
    const manager = new NativeSessionManager('daemon-a', { connect })
    const snapshots: Array<{ error?: Error }> = []

    await expect(manager.get({ onConnectionState: (snapshot) => snapshots.push(snapshot) })).rejects.toBe(failure)

    expect(snapshots.some((snapshot) => snapshot.error === failure)).toBe(true)
    expect(manager.connectionState.getSnapshot().error).toBe(failure)
  })

  it('does not publish an asynchronous failure for an explicit reset', async () => {
    const session = fakeSession()
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })
    const phases: string[] = []
    manager.connectionState.subscribe(() => phases.push(manager.connectionState.getSnapshot().phase))
    await manager.get()

    await manager.reset()
    session.fail(Object.assign(new Error('late close'), { code: 'unavailable' }))

    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'idle', error: null })
    expect(phases).not.toContain('failed')
  })

  it('lets one lease cancel its wait without cancelling the shared connect', async () => {
    let resolveConnect!: (session: ProtoClientSession) => void
    let managerSignal: AbortSignal | undefined
    const connect = vi.fn((_input, options) => {
      managerSignal = options?.signal
      return new Promise<ProtoClientSession>((resolve) => { resolveConnect = resolve })
    })
    const manager = new NativeSessionManager('daemon-a', { connect })
    const controller = new AbortController()

    const cancelled = manager.get({ signal: controller.signal })
    const workspace = manager.lease()
    controller.abort(new DOMException('Aborted', 'AbortError'))
    resolveConnect(fakeSession())

    await expect(cancelled).rejects.toMatchObject({ name: 'AbortError' })
    await expect(workspace).resolves.toMatchObject({ stamp: { endpointId: 'daemon-a' } })
    expect(connect).toHaveBeenCalledTimes(1)
    expect(managerSignal?.aborted).toBe(false)
  })

  it('does not block generation reset on an old pending connect', async () => {
    let managerSignal: AbortSignal | undefined
    const connect = vi.fn((_input, options) => {
      managerSignal = options?.signal
      return new Promise<ProtoClientSession>(() => {})
    })
    const manager = new NativeSessionManager('daemon-a', { connect })
    void manager.get()

    await expect(manager.reset()).resolves.toBeUndefined()
    expect(managerSignal?.aborted).toBe(true)
  })

  it('keeps the full managed connection window bounded without truncating the Go ICE deadline', async () => {
    vi.useFakeTimers()
    let managerSignal: AbortSignal | undefined
    const connect = vi.fn((_input, options) => {
      managerSignal = options?.signal
      return new Promise<ProtoClientSession>((_resolve, reject) => {
        options?.signal?.addEventListener('abort', () => reject(options.signal?.reason), { once: true })
      })
    })
    const manager = new NativeSessionManager('daemon-a', { connect })
    const pending = manager.get()
    const settled = pending.then(() => null, (error: unknown) => error)

    try {
      await vi.advanceTimersByTimeAsync(44_999)
      expect(managerSignal?.aborted).toBe(false)
      await vi.advanceTimersByTimeAsync(1)
      expect(await settled).toMatchObject({ message: 'client session timed out' })
      expect(managerSignal?.aborted).toBe(true)
    } finally {
      vi.useRealTimers()
    }
  })

  it('does not block generation reset on an unresponsive old session close', async () => {
    const session = fakeSession()
    session.close = vi.fn(() => new Promise<void>(() => {}))
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })
    await manager.get()

    await expect(manager.reset()).resolves.toBeUndefined()
    expect(session.close).toHaveBeenCalledTimes(1)
  })

  it('explicitly disconnects the pooled physical generation without reconnecting', async () => {
    const first = fakeSession()
    const second = fakeSession(2n)
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const disconnect = vi.fn(async () => { await first.invalidate?.() })
    const manager = new NativeSessionManager('daemon-a', { connect, disconnect })
    await manager.get()

    await manager.disconnect()

    expect(first.invalidate).toHaveBeenCalledTimes(1)
    expect(disconnect).toHaveBeenCalledWith('daemon-a')
    expect(first.close).toHaveBeenCalledTimes(1)
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'idle' })
    expect(connect).toHaveBeenCalledTimes(1)

    await manager.get()
    expect(connect).toHaveBeenCalledTimes(2)
    expect(manager.connectionState.getSnapshot().connectionInfo).toMatchObject({ generation: 2n })
  })

  it('does not open a replacement until explicit disconnect invalidation finishes', async () => {
    let finishInvalidation!: () => void
    const first = fakeSession()
    const second = fakeSession(2n)
    const disconnect = vi.fn(() => new Promise<void>((resolve) => { finishInvalidation = resolve }))
    const connect = vi.fn()
      .mockResolvedValueOnce(first)
      .mockResolvedValueOnce(second)
    const manager = new NativeSessionManager('daemon-a', { connect, disconnect })
    await manager.get()

    const disconnecting = manager.disconnect()
    const reopening = manager.get()
    await vi.waitFor(() => expect(disconnect).toHaveBeenCalledTimes(1))
    expect(connect).toHaveBeenCalledTimes(1)

    finishInvalidation()
    await disconnecting
    await reopening
    expect(connect).toHaveBeenCalledTimes(2)
    expect(first.close).toHaveBeenCalledTimes(1)
  })

  it('invalidates a late session when disconnect wins an in-flight connect', async () => {
    let resolveConnect!: (session: ProtoClientSession) => void
    const late = fakeSession()
    const connect = vi.fn(() => new Promise<ProtoClientSession>((resolve) => { resolveConnect = resolve }))
    const disconnect = vi.fn(async () => undefined)
    const manager = new NativeSessionManager('daemon-a', { connect, disconnect })
    const opening = manager.get()

    const disconnecting = manager.disconnect()
    resolveConnect(late)

    await expect(opening).rejects.toThrow('native session generation changed')
    await disconnecting
    expect(disconnect).toHaveBeenCalledWith('daemon-a')
    expect(late.close).toHaveBeenCalledTimes(1)
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'idle' })
    expect(connect).toHaveBeenCalledTimes(1)
  })

  it('reports an explicit disconnect when physical invalidation fails', async () => {
    const session = fakeSession()
    const failure = Object.assign(new Error('binding unavailable'), { code: 'unavailable' })
    const manager = new NativeSessionManager('daemon-a', {
      connect: vi.fn(async () => session),
      disconnect: vi.fn(async () => { throw failure }),
    })
    await manager.get()

    await expect(manager.disconnect()).rejects.toBe(failure)
    expect(session.close).toHaveBeenCalledTimes(1)
    expect(manager.connectionState.getSnapshot()).toMatchObject({ phase: 'idle' })
  })

  it('treats an already invalidated generation as a successful disconnect', async () => {
    const session = fakeSession()
    session.invalidate = vi.fn(async () => { throw Object.assign(new Error('stale'), { code: 'stale_session' }) })
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })
    await manager.get()

    await expect(manager.disconnect()).resolves.toBeUndefined()
    expect(session.close).toHaveBeenCalledTimes(1)
  })

  it('reports a closed lease through rejected async operations', async () => {
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => fakeSession()) })
    const lease = await manager.get()
    await manager.reset()

    await expect(lease.execute(create(CommandEnvelopeSchema))).rejects.toThrow('Proto session lease is closed')
  })

  it('forwards live connection snapshot sampling through the UI lease', async () => {
    const session = fakeSession()
    session.getConnectionSnapshot = vi.fn()
      .mockResolvedValueOnce(create(ConnectionSnapshotSchema, { sampledAtUnixNano: 10n, bytesSent: 20n }))
      .mockResolvedValueOnce(create(ConnectionSnapshotSchema, { sampledAtUnixNano: 30n, bytesSent: 40n }))
    const manager = new NativeSessionManager('daemon-a', { connect: vi.fn(async () => session) })
    const lease = await manager.get()

    await expect(lease.getConnectionSnapshot?.()).resolves.toMatchObject({ sampledAtUnixNano: 10n, bytesSent: 20n })
    await expect(lease.getConnectionSnapshot?.()).resolves.toMatchObject({ sampledAtUnixNano: 30n, bytesSent: 40n })
    expect(session.getConnectionSnapshot).toHaveBeenCalledTimes(2)
  })
})

function fakeSession(generation = 1n, connection?: ConnectionSnapshot): ProtoClientSession & {
  markDead(): void
  fail(error: ProtoClientSessionCloseError): void
} {
  let alive = true
  const closeHandlers = new Set<ProtoClientSessionCloseHandler>()
  return {
    stamp: create(EndpointSessionStampSchema, { endpointId: 'daemon-a', routeId: 'direct', generation }),
    ...(connection ? { connection } : {}),
    execute: vi.fn(async (_command: CommandEnvelope) => ({} as ResultEnvelope)),
    subscribeEvents: vi.fn((_handler: (event: EventEnvelope) => void): ProtoClientSubscription => ({ close() {} })),
    subscribeClosed: vi.fn((handler): ProtoClientSubscription => {
      closeHandlers.add(handler)
      return { close: () => closeHandlers.delete(handler) }
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
