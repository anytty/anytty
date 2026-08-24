import { create } from '@bufbuild/protobuf'
import { lazy, Suspense, useEffect, useMemo, useState, type CSSProperties } from 'react'
import { AnyTTYApiApplication, AnyTTYApiTerminal, ProtoBindingClient, ProtoBindingConnector, createBrowserRemoteNetworkRuntime, normalizeTerminalInventory } from '@anytty/ui'
import type { LocalStatus, Machine, MachineWorkspaceProps } from '@anytty/ui'
import { WebSocketBindingBackend, type BindingBridgeEndpoint } from './GoBindingClient'

const LocalMachineWorkspace = lazy(() => import('@anytty/ui/machine-workspace').then((module) => ({ default: module.MachineWorkspace })))

export interface LocalWebBootstrap {
  bridge: BindingBridgeEndpoint
  machine: {
    id: 'local'
    name: string
    platform: string
  }
}

export function LocalWebAnyTTYApp({ bootstrap, initialAppThemeStyle }: { bootstrap: LocalWebBootstrap; initialAppThemeStyle: CSSProperties }) {
  const [connectionError, setConnectionError] = useState<string | null>(null)
  const networkRuntime = useMemo(() => createBrowserRemoteNetworkRuntime(), [])
  const storage = networkRuntime.storage
  if (!storage) throw new Error('browser storage is unavailable')

  const client = useMemo(() => new ProtoBindingClient(new WebSocketBindingBackend(async () => bootstrap.bridge)), [bootstrap.bridge])
  useEffect(
    () => () => {
      void client.close()
    },
    [client],
  )
  const connector = useMemo(
    () =>
      new ProtoBindingConnector(() => client, {
        endpointId: bootstrap.machine.id,
      }),
    [bootstrap.machine.id, client],
  )
  const machine = useMemo<Machine>(
    () => ({
      machineId: bootstrap.machine.id,
      name: bootstrap.machine.name,
      state: 'online',
    }),
    [bootstrap.machine.id, bootstrap.machine.name],
  )
  const api = useMemo(() => createLocalMachineAPI(machine, connector, setConnectionError), [connector, machine])
  const workspaceConnector = useMemo<MachineWorkspaceProps['connector']>(
    () => ({
      connect(input, options) {
        if (input.machineId !== machine.machineId) return Promise.reject(new Error('local web cannot connect to another machine'))
        return connectLocal(connector, machine.machineId, options, setConnectionError)
      },
    }),
    [connector, machine.machineId],
  )

  return (
    <section className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex h-full w-full flex-col overflow-hidden antialiased" style={initialAppThemeStyle}>
      <Suspense fallback={<div className="h-full w-full bg-[var(--anytty-app-bg)]" />}>
        <LocalMachineWorkspace
          api={api}
          connector={workspaceConnector}
          initialMachine={machine}
          storage={storage}
          phoneOnline
          connectionState="ready"
          webLayout
        />
      </Suspense>
      {connectionError ? (
        <div className="fixed bottom-4 left-1/2 z-50 max-w-[calc(100%-2rem)] -translate-x-1/2 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800 shadow" role="alert">
          {connectionError}
        </div>
      ) : null}
    </section>
  )
}

function createLocalMachineAPI(machine: Machine, connector: ProtoBindingConnector, onConnectionError: (message: string | null) => void): MachineWorkspaceProps['api'] {
  return {
    async getStatus(): Promise<LocalStatus> {
      return {
        machine,
        localWeb: { httpUrl: globalThis.location.origin, rtcOfferUrl: '' },
      }
    },
    async listTerminals(options) {
      try {
        const session = await connectLocal(connector, machine.machineId, options, onConnectionError)
        try {
          const response = await session.execute(
            create(AnyTTYApiApplication.CommandEnvelopeSchema, {
              command: {
                case: 'terminalList',
                value: create(AnyTTYApiTerminal.TerminalListCommandSchema),
              },
            }),
          )
          if (response.result.case !== 'terminalList') throw new Error(`terminal list returned ${response.result.case || 'no result'}`)
          return normalizeTerminalInventory({
            machine_id: machine.machineId,
            terminals: response.result.value.terminals.map((terminal) => ({
              terminal_id: terminal.ref?.terminalId ?? '',
              name: terminal.name,
              state: terminal.state === AnyTTYApiTerminal.TerminalState.RUNNING ? 'running' : terminal.state === AnyTTYApiTerminal.TerminalState.EXITED ? 'exited' : 'unknown',
              command: terminal.command,
              cwd: terminal.cwd,
              live_cwd: terminal.liveCwd,
              cols: terminal.size?.cols ?? 0,
              rows: terminal.size?.rows ?? 0,
              foreground_process: terminal.foregroundProcess || undefined,
              last_output_at: unixNanoISOString(terminal.lastOutputAtUnixNano),
              size_locked: terminal.tags['anytty.size_lock'] === 'lock',
              size_lock_mode: terminal.tags['anytty.size_lock'],
            })),
          }).terminals
        } finally {
          await session.close()
        }
      } catch (error) {
        onConnectionError(error instanceof Error ? error.message : String(error))
        throw error
      }
    },
  }
}

async function connectLocal(connector: ProtoBindingConnector, machineId: string, options: Parameters<ProtoBindingConnector['connect']>[1], onConnectionError: (message: string | null) => void) {
  try {
    const session = await connector.connect({ machineId }, options)
    onConnectionError(null)
    return session
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    onConnectionError(message)
    throw error
  }
}

function unixNanoISOString(value: bigint): string | undefined {
  if (value <= 0n) return undefined
  return new Date(Number(value / 1_000_000n)).toISOString()
}

export function parseLocalWebBootstrap(value: unknown): LocalWebBootstrap | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  const record = value as Record<string, unknown>
  if (!record.bridge || typeof record.bridge !== 'object' || Array.isArray(record.bridge)) return null
  if (!record.machine || typeof record.machine !== 'object' || Array.isArray(record.machine)) return null
  const bridge = record.bridge as Record<string, unknown>
  const machine = record.machine as Record<string, unknown>
  const hasLocalPath = bridge.path === '/api/bridge'
  const hasLoopbackPort = Number.isInteger(bridge.port) && Number(bridge.port) >= 1 && Number(bridge.port) <= 65535
  if (!hasLocalPath && !hasLoopbackPort) return null
  if (typeof bridge.token !== 'string' || !/^[A-Za-z0-9_-]{43}$/.test(bridge.token)) return null
  if (machine.id !== 'local' || typeof machine.name !== 'string' || machine.name.trim() === '') return null
  if (typeof machine.platform !== 'string' || !/^[a-z0-9_-]{1,32}$/.test(machine.platform)) return null
  return {
    bridge: hasLocalPath ? { path: '/api/bridge', token: bridge.token } : { port: Number(bridge.port), token: bridge.token },
    machine: {
      id: 'local',
      name: machine.name.trim(),
      platform: machine.platform,
    },
  }
}
