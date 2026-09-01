import { useEffect } from 'react'
import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { anyttyI18n } from '../i18n'
import type { MachineConnectionSnapshot } from '../connection/machineConnectionSnapshot'
import type { RemoteNetworkRuntime, RemoteRuntimeStorage } from '../core/transport'
import { createMachineStore } from '../state/machineStore'
import { dispatchNativeBack } from '../platform/nativeBack'
import { connectionPathDetail, machinePlatformIcon, machinePlatformLabel, RemoteControlApp, type ExternalPairingAdapter, type MachineRuntime } from './RemoteControlApp'

vi.mock('./MachineWorkspace', () => ({
  MachineWorkspace: function MockMachineWorkspace({
    onBack,
    retainConnectionDemand,
    connectionResumeIntent,
  }: {
    onBack: () => void
    retainConnectionDemand?: ((resumeIntent?: object | null) => () => void) | undefined
    connectionResumeIntent?: object | null | undefined
  }) {
    useEffect(() => retainConnectionDemand?.(connectionResumeIntent ?? null), [connectionResumeIntent, retainConnectionDemand])
    return <button type="button" onClick={onBack}>Back to devices</button>
  },
  ConnectionInfoDialog: ({ policyState, onApply, onRefresh }: {
    policyState: { policy: { route: string; cloud: string; relayTransport: string } } | null
    onApply: (policy: { route: 'cloud'; cloud: 'relay'; relayTransport: 'tcp' }) => void
    onRefresh: () => void
  }) => (
    <div aria-label="Connection & network" role="dialog">
      <span>{policyState ? `${policyState.policy.cloud}/${policyState.policy.relayTransport}` : 'loading'}</span>
      <button type="button" onClick={onRefresh}>Refresh connection</button>
      <button type="button" onClick={() => onApply({ route: 'cloud', cloud: 'relay', relayTransport: 'tcp' })}>Apply Relay TCP</button>
    </div>
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

  it('names Direct and SSH connection routes without calling them local', () => {
    const t = anyttyI18n.t.bind(anyttyI18n)
    expect(connectionPathDetail(connectedRouteSnapshot('direct'), t)).toBe('P2P direct')
    expect(connectionPathDetail(connectedRouteSnapshot('ssh'), t)).toBe('SSH')
  })

  it('uses explicit OS presets and keeps a generic fallback', () => {
    expect(machinePlatformLabel('darwin')).toBe('macOS')
    expect(machinePlatformLabel('win32')).toBe('Windows')
    expect(machinePlatformLabel('future-os')).toBe('future-os')
    expect(machinePlatformIcon('future-os')).toBe(machinePlatformIcon(undefined))
    expect(machinePlatformIcon('darwin')).not.toBe(machinePlatformIcon(undefined))
    expect(machinePlatformIcon('windows')).not.toBe(machinePlatformIcon(undefined))
  })

  it('does not label a route as relay without an observed relay path', () => {
    const t = anyttyI18n.t.bind(anyttyI18n)
    const snapshot = connectedRouteSnapshot('direct')
    snapshot.relayInUse = true
    if (snapshot.connectionInfo) snapshot.connectionInfo.relayInUse = true

    expect(connectionPathDetail(snapshot, t)).toBe('P2P direct')
  })

  it('shows authenticated per-device Edge presence without opening a session', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const probeConnection = vi.fn(async () => undefined)
    const snapshot = idleSnapshot()
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
        subscribe: () => () => undefined,
      },
      probeConnection,
    }

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={() => runtime}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    await waitFor(() => expect(screen.getAllByLabelText('Daemon online on Edge').length).toBeGreaterThan(0))
    expect(screen.queryByText('Not connected')).toBeNull()
    expect(screen.queryByText('0 terminals')).toBeNull()
    expect(screen.queryByText('AnyTTY daemon')).toBeNull()
    expect(screen.getByTestId('anytty-machine-list-panel').contains(screen.getByRole('list', { name: 'Devices' }))).toBe(true)
    expect(screen.getByRole('button', { name: 'Open Build host' }).className).toContain('rounded-lg')
    expect(probeConnection).not.toHaveBeenCalled()
  })

  it('does not show direct probing after a direct session is connected', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'local_cloud',
      addresses: { local: ['https://build.local:41102'], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const runtime = runtimeWithSnapshot(connectedRouteSnapshot('direct'))
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).includes('cloud.anytty.com')) return new Response('{}', { status: 401 })
      return new Response('{}', { status: 503 })
    })

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={() => runtime}
        networkRuntime={{ storage, queryParam: () => null, fetch }}
      />,
    )

    await waitFor(() => expect(screen.getAllByLabelText(/Direct access available/).length).toBeGreaterThan(0))
    expect(screen.queryByLabelText('Checking Direct TCP access')).toBeNull()
    await waitFor(() => expect(screen.getAllByLabelText(/Daemon online on Edge/).length).toBeGreaterThan(0))
    expect(screen.queryByText('Session connected')).toBeNull()
    expect(screen.getByText('P2P direct')).toBeTruthy()
    expect(screen.queryByText('0 terminals')).toBeNull()
  })

  it('shows mDNS discovery as an available Direct route before opening a session', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'offline',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'local_cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        externalPairingAdapter={authorizedAdapter()}
        directReachableMachineIds={new Set(['device-1'])}
        machineRuntimeFactory={() => runtimeWithSnapshot(idleSnapshot())}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    const localStatus = await screen.findByLabelText('Direct access available')
    expect(localStatus.getAttribute('data-tone')).toBe('positive')
    expect(localStatus.textContent).toBe('Direct')
  })

  it('does not report a configured Direct route as unreachable when mDNS does not find it', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Public Direct host',
      state: 'offline',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'local',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-08-27T00:00:00.000Z',
      updatedAt: '2026-08-27T00:00:00.000Z',
    })

    render(
      <RemoteControlApp
        externalPairingAdapter={authorizedAdapter()}
        directReachableMachineIds={new Set()}
        directCheckingMachineIds={new Set()}
        machineRuntimeFactory={() => runtimeWithSnapshot(idleSnapshot())}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    const directStatus = await screen.findByLabelText('Direct access status unknown')
    expect(directStatus.textContent).toBe('Direct')
    expect(directStatus.getAttribute('data-tone')).toBe('neutral')
    expect(screen.queryByText('Not currently reachable')).toBeNull()
  })

  it('shows connection progress only inside the affected device row', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'offline',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'local_cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const connecting = { ...idleSnapshot(), phase: 'connecting' as const, statusText: 'Connecting' }

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        externalPairingAdapter={authorizedAdapter()}
        locallyDiscoveredMachineIds={new Set(['device-1'])}
        machineRuntimeFactory={() => runtimeWithSnapshot(connecting)}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    expect((await screen.findByLabelText('Direct access available')).textContent).toBe('Direct')
    expect((await screen.findByLabelText('Daemon online on Edge')).textContent).toBe('Cloud')
    expect(await screen.findByText('Connecting to the device...')).toBeTruthy()
    expect(screen.queryByTestId('anytty-connection-recovery-overlay')).toBeNull()
    expect(screen.getByRole('button', { name: 'Open Build host' }).hasAttribute('disabled')).toBe(false)
  })

  it('shows an inline progress indicator for each reachability check in flight', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'local_cloud',
      addresses: { local: ['https://build.local:41102'], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const pendingFetch = vi.fn(() => new Promise<Response>(() => undefined))

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'checking']])}
        directReachableMachineIds={new Set()}
        directCheckingMachineIds={new Set(['device-1'])}
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={() => runtimeWithSnapshot(idleSnapshot())}
        networkRuntime={{ storage, queryParam: () => null, fetch: pendingFetch }}
      />,
    )

    const directCheck = await screen.findByLabelText('Checking Direct TCP access')
    const cloudCheck = await screen.findByLabelText('Checking daemon presence on Edge')
    expect(directCheck.getAttribute('aria-busy')).toBe('true')
    expect(cloudCheck.getAttribute('aria-busy')).toBe('true')
    expect(directCheck.querySelector('.lucide-route')?.classList.contains('animate-spin')).toBe(false)
    expect(directCheck.querySelector('.lucide-route')?.parentElement?.classList.contains('animate-pulse')).toBe(true)
    expect(directCheck.querySelector('.lucide-loader-circle')).toBeNull()
    expect(cloudCheck.querySelector('.lucide-cloud')?.classList.contains('animate-spin')).toBe(false)
    expect(cloudCheck.querySelector('.lucide-cloud')?.parentElement?.classList.contains('animate-pulse')).toBe(true)
    expect(cloudCheck.querySelector('.lucide-loader-circle')).toBeNull()
  })

  it.each([
    ['phone network', { phoneOnline: false, connectionState: 'ready' as const }],
    ['native generation', { phoneOnline: true, connectionState: 'checking' as const }],
  ])('reprobes local Hub reachability when the %s becomes ready', async (_source, unavailableProps) => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'local_cloud',
      addresses: { local: ['https://build.local:41102'], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const fetch = vi.fn(async () => new Response('{}', { status: 200 }))
    const commonProps = {
      cloudPresenceByMachineId: new Map([['device-1', 'online' as const]]),
      externalPairingAdapter: authorizedAdapter(),
      machineRuntimeFactory: () => runtimeWithSnapshot(idleSnapshot()),
      networkRuntime: { storage, queryParam: () => null, fetch },
    }
    const view = render(<RemoteControlApp {...commonProps} {...unavailableProps} />)

    await Promise.resolve()
    expect(fetch).not.toHaveBeenCalled()

    view.rerender(<RemoteControlApp {...commonProps} phoneOnline connectionState="ready" />)

    await waitFor(() => expect(fetch).toHaveBeenCalledOnce())
    expect(String(fetch.mock.calls[0]?.[0])).toBe('https://build.local:41102/api/health')
  })

  it('does not treat Controller health as daemon presence and shows the Edge offline result', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })

    const fetch = vi.fn(async () => new Response('{}', { status: 200 }))
    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'offline']])}
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={() => runtimeWithSnapshot(idleSnapshot())}
        networkRuntime={{ storage, queryParam: () => null, fetch }}
      />,
    )

    const cloud = await screen.findByLabelText('Daemon offline on Edge')
    expect(cloud.getAttribute('data-tone')).toBe('warning')
    expect(fetch).not.toHaveBeenCalled()
    expect(screen.queryByText('Not connected')).toBeNull()
    expect(screen.queryByText('Cloud offline')).toBeNull()
    expect(screen.getByRole('button', { name: 'Open Build host' }).hasAttribute('disabled')).toBe(false)
  })

  it('does not turn a missed mDNS discovery into a Direct connection failure', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'local_cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const reconnecting = { ...idleSnapshot(), phase: 'reconnecting' as const, statusText: 'Reconnecting' }

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'offline']])}
        externalPairingAdapter={authorizedAdapter()}
        locallyDiscoveredMachineIds={new Set()}
        locallyDiscoveringMachineIds={new Set()}
        machineRuntimeFactory={() => runtimeWithSnapshot(reconnecting)}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    expect((await screen.findByLabelText('Daemon offline on Edge')).getAttribute('data-tone')).toBe('warning')
    expect(screen.getByRole('button', { name: 'Open Build host' }).hasAttribute('disabled')).toBe(false)
    const status = screen.getByText('Connection interrupted. Reconnecting...').closest<HTMLElement>('[role="status"]')
    expect(status).toBeTruthy()
    expect(status.textContent).toBe('Connection interrupted. Reconnecting...')
    expect(status.getAttribute('aria-busy')).toBe('true')
    expect(status.querySelector('.lucide-loader-circle')).toBeTruthy()
    expect(screen.queryByText('Not currently reachable')).toBeNull()
    expect(screen.queryByTestId('anytty-connection-recovery-overlay')).toBeNull()
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
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={machineRuntimeFactory}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'Open Build host' }))
    act(() => { expect(dispatchNativeBack()).toBe(true) })

    await waitFor(() => expect(screen.getAllByLabelText('Daemon online on Edge').length).toBeGreaterThan(0))
    expect(screen.queryByText('Session connected')).toBeNull()
    expect(screen.queryByText(/Single relay/)).toBeNull()
    expect(machineRuntimeFactory).toHaveBeenCalledTimes(1)
    expect(dispose).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Open Build host' }))
    await userEvent.click(screen.getByRole('button', { name: 'Back to devices' }))
    expect(machineRuntimeFactory).toHaveBeenCalledTimes(1)

    await userEvent.click(screen.getByRole('button', { name: 'More actions for Build host' }))
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Disconnect from Build host' }))

    await waitFor(() => expect(disconnect).toHaveBeenCalledTimes(1))
    expect(screen.queryByText('Not connected')).toBeNull()
    expect(screen.queryByRole('menuitem', { name: 'Disconnect from Build host' })).toBeNull()
  })

  it('mints a resume intent only for explicit entry and restores the workspace journal without one', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const resumeIntent = {}
    const releaseConnectionDemand = vi.fn()
    const retainConnectionDemand = vi.fn((_intent?: object | null) => releaseConnectionDemand)
    const createWorkspaceResumeIntent = vi.fn(() => resumeIntent)
    const onWorkspaceResumeIntent = vi.fn((_machineId: string, _intent: object) => undefined)
    const onActiveWorkspaceChange = vi.fn((_machineId: string | null) => undefined)
    const runtime: MachineRuntime = {
      ...runtimeWithSnapshot(connectedRelaySnapshot()),
      retainConnectionDemand,
    }
    const adapter = authorizedAdapter()
    const renderApp = () => render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        createWorkspaceResumeIntent={createWorkspaceResumeIntent}
        externalPairingAdapter={adapter}
        machineRuntimeFactory={() => runtime}
        networkRuntime={networkRuntime(storage)}
        onActiveWorkspaceChange={onActiveWorkspaceChange}
        onWorkspaceResumeIntent={onWorkspaceResumeIntent}
      />,
    )

    const enteredView = renderApp()
    fireEvent.click(await screen.findByRole('button', { name: 'Open Build host' }))

    expect(createWorkspaceResumeIntent).toHaveBeenCalledOnce()
    expect(onWorkspaceResumeIntent).toHaveBeenCalledWith('device-1', resumeIntent)
    expect(onActiveWorkspaceChange).toHaveBeenCalledWith('device-1')
    await waitFor(() => expect(retainConnectionDemand).toHaveBeenCalledWith(resumeIntent))
    expect(createWorkspaceResumeIntent.mock.invocationCallOrder[0]).toBeLessThan(
      onWorkspaceResumeIntent.mock.invocationCallOrder[0]!,
    )
    expect(onWorkspaceResumeIntent.mock.invocationCallOrder[0]).toBeLessThan(
      retainConnectionDemand.mock.invocationCallOrder[0]!,
    )
    enteredView.unmount()
    expect(releaseConnectionDemand).toHaveBeenCalledOnce()

    createWorkspaceResumeIntent.mockClear()
    onWorkspaceResumeIntent.mockClear()
    onActiveWorkspaceChange.mockClear()
    retainConnectionDemand.mockClear()
    releaseConnectionDemand.mockClear()

    const restoredView = renderApp()
    expect(await screen.findByRole('button', { name: 'Back to devices' })).toBeTruthy()
    await waitFor(() => expect(retainConnectionDemand).toHaveBeenCalledWith(null))
    expect(createWorkspaceResumeIntent).not.toHaveBeenCalled()
    expect(onWorkspaceResumeIntent).not.toHaveBeenCalled()
    expect(onActiveWorkspaceChange).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Back to devices' }))
    expect(onActiveWorkspaceChange).toHaveBeenCalledWith(null)
    restoredView.unmount()

    retainConnectionDemand.mockClear()
    const homeView = renderApp()
    expect(await screen.findByRole('button', { name: 'Open Build host' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Back to devices' })).toBeNull()
    expect(retainConnectionDemand).not.toHaveBeenCalled()
    homeView.unmount()
  })

  it('keeps device truth visible while app recovery pauses remote actions', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        connectionState="recovering"
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={() => runtimeWithSnapshot(connectedRelaySnapshot())}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    expect(screen.queryByText('Restoring the app connection...')).toBeNull()
    expect(screen.queryByTestId('anytty-connection-recovery-overlay')).toBeNull()
    expect((screen.getByRole('button', { name: 'Open Build host' }) as HTMLButtonElement).disabled).toBe(true)
    expect(screen.getByLabelText('Daemon online on Edge')).toBeTruthy()

    await userEvent.click(screen.getByRole('button', { name: 'More actions for Build host' }))
    expect(screen.queryByRole('menuitem', { name: 'Disconnect from Build host' })).toBeNull()
    await userEvent.click(screen.getByRole('menuitem', { name: 'Device details' }))
    const dialog = screen.getByRole('dialog', { name: 'Build host' })
    expect(dialog.textContent).toContain('Connected')
    expect(dialog.textContent).toContain('Single relay')
  })

  it('edits and reconnects a device policy from the list while connection recovery is unavailable', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      state: 'online',
      terminalCount: 0,
      source: 'manual',
      accessClass: 'cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })
    const getConnectionPolicy = vi.fn(async () => ({
      policy: { route: 'auto', cloud: 'auto', relayTransport: 'auto' } as const,
      available: { direct: true, ssh: false, cloud: true },
      unavailableReasons: { ssh: 'credential_unavailable' as const },
    }))
    const applyConnectionPolicy = vi.fn(async () => undefined)
    const probeConnection = vi.fn(async () => undefined)
    const runtime: MachineRuntime = {
      api: {
        getStatus: vi.fn(async () => ({
          machine: { machineId: 'device-1', name: 'Build host', state: 'online' },
          localWeb: { httpUrl: '', rtcOfferUrl: '' },
        })),
        listTerminals: vi.fn(async () => []),
      },
      connector: {
        connect: vi.fn(async () => { throw new Error('unused') }),
        getConnectionPolicy,
        applyConnectionPolicy,
      },
      probeConnection,
    }

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        connectionState="recovering"
        externalPairingAdapter={authorizedAdapter()}
        machineRuntimeFactory={() => runtime}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'More actions for Build host' }))
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Connection & network' }))
    await waitFor(() => expect(getConnectionPolicy).toHaveBeenCalledOnce())
    expect(screen.getByRole('dialog', { name: 'Connection & network' })).toBeTruthy()
    expect(probeConnection).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Refresh connection' }))
    await waitFor(() => expect(probeConnection).toHaveBeenCalledOnce())

    await userEvent.click(screen.getByRole('button', { name: 'Apply Relay TCP' }))
    await waitFor(() => expect(applyConnectionPolicy).toHaveBeenCalledWith({ route: 'cloud', cloud: 'relay', relayTransport: 'tcp' }))
    expect(probeConnection).toHaveBeenCalledTimes(2)
  })

  it('closes device details with one native Back event and preserves the device list', async () => {
    const storage = new MemoryStorage()
    createMachineStore({ storage }).saveMachine({
      machineId: 'device-1',
      name: 'Build host',
      hostname: 'build.local',
      osInfo: 'macOS 15.5',
      hubId: 'hub-shanghai-1',
      state: 'online',
      terminalCount: 2,
      lastSeenAt: '2026-07-28T01:00:00.000Z',
      source: 'manual',
      accessClass: 'local_cloud',
      addresses: { local: [], lan: [], public: [] },
      endpoints: {},
      addedAt: '2026-07-28T00:00:00.000Z',
      updatedAt: '2026-07-28T00:00:00.000Z',
    })

    render(
      <RemoteControlApp
        cloudPresenceByMachineId={new Map([['device-1', 'online']])}
        externalPairingAdapter={authorizedAdapter()}
        networkRuntime={networkRuntime(storage)}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'More actions for Build host' }))
    await userEvent.click(await screen.findByRole('menuitem', { name: 'Device details' }))
    const dialog = screen.getByRole('dialog', { name: 'Build host' })
    expect(dialog.querySelector('dl')?.parentElement?.className).toContain('pb-[calc(env(safe-area-inset-bottom)+1rem)]')
    expect(dialog.textContent).toContain('macOS 15.5')
    expect(dialog.textContent).toContain('hub-shanghai-1')
    await waitFor(() => expect(dialog.textContent).toContain('Daemon online on Edge'))

    await userEvent.click(screen.getByRole('button', { name: 'Edit device name' }))
    const alias = screen.getByRole('textbox', { name: 'Name in this app' })
    await userEvent.type(alias, 'Studio Mac')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(screen.getByRole('dialog', { name: 'Studio Mac' })).toBeTruthy()
    expect(createMachineStore({ storage }).getMachine('device-1')).toMatchObject({
      name: 'Build host',
      alias: 'Studio Mac',
    })

    await userEvent.click(screen.getByRole('button', { name: 'Edit device icon' }))
    await userEvent.click(screen.getByRole('button', { name: 'Laptop' }))
    const upload = screen.getByLabelText('Upload local image') as HTMLInputElement
    expect(upload.accept).toBe('image/*')
    expect(screen.getByText(/stay on this client/i)).toBeTruthy()
    expect(createMachineStore({ storage }).getMachine('device-1')).toMatchObject({
      name: 'Build host',
      alias: 'Studio Mac',
      icon: 'laptop',
    })

    act(() => { expect(dispatchNativeBack()).toBe(true) })

    expect(screen.queryByRole('dialog', { name: 'Studio Mac' })).toBeNull()
    expect(screen.getByRole('button', { name: 'Open Studio Mac' })).toBeTruthy()
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
        connectionState="recovering"
        externalPairingAdapter={authorizedAdapter()}
        networkRuntime={networkRuntime(storage)}
        onRefreshMachines={onRefreshMachines}
      />,
    )

    const scroller = await screen.findByTestId('anytty-machine-list-scroller')
    expect(screen.queryByText('Restoring the app connection...')).toBeNull()
    expect(screen.queryByTestId('anytty-connection-recovery-overlay')).toBeNull()
    expect((screen.getByRole('button', { name: 'Refresh devices' }) as HTMLButtonElement).disabled).toBe(true)
    fireEvent.touchStart(scroller, { touches: [{ clientY: 10 }] })
    fireEvent.touchMove(scroller, { touches: [{ clientY: 150 }] })
    fireEvent.touchEnd(scroller)
    expect(onRefreshMachines).not.toHaveBeenCalled()

    view.rerender(
      <RemoteControlApp
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

function runtimeWithSnapshot(snapshot: MachineConnectionSnapshot): MachineRuntime {
  return {
    api: {
      getStatus: vi.fn(async () => ({
        machine: { machineId: snapshot.machineId, name: 'Build host', state: 'online' },
        localWeb: { httpUrl: '', rtcOfferUrl: '' },
      })),
      listTerminals: vi.fn(async () => []),
    },
    connector: { connect: vi.fn(async () => { throw new Error('unused') }) },
    listConnectionState: {
      getSnapshot: () => snapshot,
      subscribe: () => () => undefined,
    },
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

function connectedRouteSnapshot(routeKind: 'direct' | 'ssh'): MachineConnectionSnapshot {
  return {
    machineId: 'device-1',
    phase: 'connected',
    statusText: 'Connected',
    connectionInfo: {
      path: 'local',
      routeKind,
      connectionId: 'device-1:1',
      machineId: 'device-1',
      relayInUse: false,
      type: 'unknown',
    },
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
