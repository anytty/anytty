import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { anyttyI18n } from '../i18n'
import type { MachineConnectionSnapshot } from '../connection/machineConnectionSnapshot'
import type { RemoteNetworkRuntime, RemoteRuntimeStorage } from '../core/transport'
import { createMachineStore } from '../state/machineStore'
import { dispatchNativeBack } from '../platform/nativeBack'
import { RemoteControlApp, type ExternalPairingAdapter, type MachineRuntime } from './RemoteControlApp'

vi.mock('./MachineWorkspace', () => ({
  MachineWorkspace: ({ onBack }: { onBack: () => void }) => (
    <button type="button" onClick={onBack}>Back to devices</button>
  ),
}))

describe('RemoteControlApp native session pool', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('keeps a relay runtime across back navigation and disconnects it explicitly from the device menu', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      hostname: 'build.local',
      state: 'online',
      terminalCount: 2,
      source: 'manual',
      accessClass: 'cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    let snapshot = connectedRelaySnapshot()
    const listeners = new Set<() => void>()
    const disconnect = vi.fn(async () => {
      snapshot = idleSnapshot()
      for (const listener of listeners) listener()
    })
    const dispose = vi.fn()
    const runtime: MachineRuntime = {
      api: {
        getStatus: vi.fn(async () => ({
          machine: { machineId: 'device-1', name: 'Build host', state: 'online' },
          localWeb: { httpUrl: '', rtcOfferUrl: '' },
        })),
        listTerminals: vi.fn(async () => []),
      },
      connector: { connect: vi.fn(async () => { throw new Error('unused') }) },
      listConnectionState: {
        getSnapshot: () => snapshot,
        subscribe(listener) {
          listeners.add(listener)
          return () => listeners.delete(listener)
        },
      },
      disconnect,
      dispose,
    }
    const machineRuntimeFactory = vi.fn(() => runtime)
    vi.spyOn(globalThis, 'confirm').mockReturnValue(true)

    render(
      <RemoteControlApp
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={machineRuntimeFactory}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'Open Build host' }))
    act(() => { expect(dispatchNativeBack()).toBe(true) })

    expect(await screen.findByText('Connected')).toBeTruthy()
    expect(screen.getByText(/Single relay/)).toBeTruthy()
    expect(machineRuntimeFactory).toHaveBeenCalledTimes(1)
    expect(dispose).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Open Build host' }))
    await userEvent.click(screen.getByRole('button', { name: 'Back to devices' }))
    expect(machineRuntimeFactory).toHaveBeenCalledTimes(1)

    await userEvent.click(screen.getByRole('button', { name: 'More actions for Build host' }))
    await userEvent.click(screen.getByRole('button', { name: 'Disconnect from Build host' }))

    await waitFor(() => expect(disconnect).toHaveBeenCalledTimes(1))
    expect(screen.getByText('Available')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Disconnect from Build host' })).toBeNull()
  })

  it('closes device details with one native Back event and preserves the device list', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      hostname: 'build.local',
      state: 'online',
      terminalCount: 2,
      source: 'manual',
      accessClass: 'local',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })

    render(
      <RemoteControlApp
        externalPairingAdapter={authorizedAdapter()}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'More actions for Build host' }))
    await userEvent.click(screen.getByRole('button', { name: 'Device details' }))
    const dialog = screen.getByRole('dialog', { name: 'Build host' })
    expect(dialog.querySelector('dl')?.className).toContain('pb-[calc(env(safe-area-inset-bottom)+1rem)]')

    act(() => { expect(dispatchNativeBack()).toBe(true) })

    expect(screen.queryByRole('dialog', { name: 'Build host' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Open Build host' })).toBeTruthy()
  })

  it('refreshes the device registry after a deliberate pull gesture', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      hostname: 'build.local',
      state: 'online',
      terminalCount: 2,
      source: 'manual',
      accessClass: 'local',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const onRefreshMachines = vi.fn(async () => undefined)

    const view = render(
      <RemoteControlApp
        connectionReady={false}
        externalPairingAdapter={authorizedAdapter()}
        networkRuntime={networkRuntime(storage)}
        onRefreshMachines={onRefreshMachines}
      />,
    )

    const scroller = await screen.findByTestId('anytty-machine-list-scroller')
    expect(screen.getByText('Network is back. Restoring secure connections...')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Refresh devices' }) as HTMLButtonElement).disabled).toBe(true)
    fireEvent.touchStart(scroller, { touches: [{ clientY: 10 }] })
    fireEvent.touchMove(scroller, { touches: [{ clientY: 150 }] })
    fireEvent.touchEnd(scroller)
    expect(onRefreshMachines).not.toHaveBeenCalled()

    view.rerender(
      <RemoteControlApp
        connectionReady
        externalPairingAdapter={authorizedAdapter()}
        networkRuntime={networkRuntime(storage)}
        onRefreshMachines={onRefreshMachines}
      />,
    )
    await waitFor(() => expect((screen.getByRole('button', { name: 'Refresh devices' }) as HTMLButtonElement).disabled).toBe(false))
    fireEvent.touchStart(scroller, { touches: [{ clientY: 10 }] })
    fireEvent.touchMove(scroller, { touches: [{ clientY: 150 }] })
    fireEvent.touchCancel(scroller)
    expect(onRefreshMachines).not.toHaveBeenCalled()

    fireEvent.touchStart(scroller, { touches: [{ clientY: 10 }] })
    fireEvent.touchMove(scroller, { touches: [{ clientY: 150 }] })
    expect(screen.getByText('Release to refresh')).toBeTruthy()
    fireEvent.touchEnd(scroller)

    await waitFor(() => expect(onRefreshMachines).toHaveBeenCalledOnce())
    expect(await screen.findByText('Device status updated')).toBeTruthy()
  })
})

function connectedRelaySnapshot(): MachineConnectionSnapshot {
  return {
    machineId: 'device-1',
    phase: 'connected',
    statusText: 'Connected',
    connectionInfo: {
      path: 'hub',
      observedPath: 'single_relay',
      connectionId: 'device-1:1',
      machineId: 'device-1',
      relayInUse: true,
      type: 'relay',
    },
    forceRelay: false,
    relayInUse: true,
    reconnectAttempt: 1,
    error: null,
  }
}

function idleSnapshot(): MachineConnectionSnapshot {
  return {
    machineId: 'device-1',
    phase: 'idle',
    statusText: 'Ready',
    connectionInfo: null,
    forceRelay: false,
    relayInUse: false,
    reconnectAttempt: 1,
    error: null,
  }
}

function authorizedAdapter(): ExternalPairingAdapter {
  return {
    import: async () => null,
    isAuthorized: () => true,
    forget: () => {},
  }
}

function networkRuntime(storage: RemoteRuntimeStorage): RemoteNetworkRuntime {
  return {
    storage,
    queryParam: () => null,
    fetch: async () => new Response('{}', { status: 200 }),
  }
}

class MemoryStorage implements RemoteRuntimeStorage {
  private readonly values = new Map<string, string>()
  getItem(key: string): string | null { return this.values.get(key) ?? null }
  removeItem(key: string): void { this.values.delete(key) }
  setItem(key: string, value: string): void { this.values.set(key, value) }
}
