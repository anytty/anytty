import { afterEach, describe, expect, it, vi } from 'vitest'
import { RemoteNetworkStateManager, type NativeNetworkStatus, type NativeNetworkStatusPlugin } from './remoteNetworkState'

describe('RemoteNetworkStateManager', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('uses an explicitly injected native network plugin', async () => {
    vi.useFakeTimers()
    let networkStatusHandler: ((status: NativeNetworkStatus) => void) | undefined
    const remove = vi.fn()
    const plugin: NativeNetworkStatusPlugin = {
      getStatus: vi.fn(async () => ({ connected: false, connectionType: 'none' })),
      addListener: vi.fn(async (_eventName, handler) => {
        networkStatusHandler = handler
        return { remove }
      }),
    }
    const manager = new RemoteNetworkStateManager(plugin)
    manager.init()
    await Promise.resolve()
    await Promise.resolve()
    await vi.advanceTimersByTimeAsync(101)
    expect(manager.state.phoneOnline).toBe(false)
    expect(manager.state.connectionType).toBe('none')

    networkStatusHandler?.({ connected: true, connectionType: 'wifi' })
    await vi.advanceTimersByTimeAsync(101)
    expect(manager.state.phoneOnline).toBe(true)
    expect(manager.state.connectionType).toBe('wifi')

    manager.destroy()
    expect(remove).toHaveBeenCalledOnce()
  })

  it('does not treat an expected native WebView freeze as a network outage', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-03T06:00:00Z'))
    const plugin: NativeNetworkStatusPlugin = {
      getStatus: vi.fn(async () => ({ connected: true, connectionType: 'wifi' })),
    }
    const manager = new RemoteNetworkStateManager(plugin)
    const snapshots: Array<{ jsFrozenRecovery: boolean; networkReady: boolean }> = []
    manager.subscribe((state) => snapshots.push(state))

    manager.init()
    await Promise.resolve()
    await Promise.resolve()
    await vi.advanceTimersByTimeAsync(101)
    expect(manager.state.networkReady).toBe(true)

    vi.setSystemTime(new Date('2026-08-03T06:10:00Z'))
    await vi.advanceTimersByTimeAsync(3_100)

    expect(snapshots.some((state) => state.jsFrozenRecovery)).toBe(true)
    expect(snapshots.every((state) => state.networkReady)).toBe(true)
    expect(manager.state.networkReady).toBe(true)
    manager.destroy()
  })
})
