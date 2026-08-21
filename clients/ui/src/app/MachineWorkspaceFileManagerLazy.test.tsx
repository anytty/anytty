import { create } from '@bufbuild/protobuf'
import { act, cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Machine } from '../core/model'
import type { FileManagerComponent } from '../files/loadFileManager'
import { AcknowledgeResultSchema } from '../generated/apipb/application_pb'
import { anyttyI18n } from '../i18n'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import type { RtcConnectionStateSnapshot } from '../core/transport'
import { dispatchNativeBack, NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import { useNativeBackHandler } from '../platform/useNativeBackHandler'
import { MachineWorkspace } from './MachineWorkspace'

const fileManagerLoader = vi.hoisted(() => ({
  load: vi.fn<() => Promise<unknown>>(),
  reload: vi.fn<() => void>(),
}))

vi.mock('../terminal/Terminal', () => ({ Terminal: () => null }))
vi.mock('../files/loadFileManager', () => ({
  loadFileManager: fileManagerLoader.load,
  reloadAfterFileManagerLoadFailure: fileManagerLoader.reload,
}))

function StatefulFileManager({ active }: { active?: boolean }) {
  const [value, setValue] = useState('')
  return (
    <div data-active={String(active)} data-testid="mock-file-manager">
      <input
        aria-label="Lazy file manager state"
        value={value}
        onChange={(event) => setValue(event.currentTarget.value)}
      />
    </div>
  )
}

function FileManagerWithWorkspaceBackHandler({ active }: { active?: boolean }) {
  useNativeBackHandler(
    fileManagerWorkspaceBackHandler,
    NATIVE_BACK_PRIORITY.WORKSPACE,
    active,
  )
  return <div data-active={String(active)} data-testid="mock-file-manager" />
}

const fileManagerWorkspaceBackHandler = vi.fn()

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, reject, resolve }
}

function renderWorkspace(
  initialMachine: Machine = { machineId: 'studio', name: 'Studio', state: 'online' },
  onBack?: () => void,
) {
  let currentMachine = initialMachine
  const sessions = new Map<string, MockProtoSession>()
  const sessionFor = (machineId: string) => {
    const existing = sessions.get(machineId)
    if (existing) return existing
    const session = new MockProtoSession(
      machineId,
      () => protoResult('acknowledge', create(AcknowledgeResultSchema)),
    )
    sessions.set(machineId, session)
    return session
  }
  const api = {
    getStatus: vi.fn(async () => ({ machine: currentMachine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
    listTerminals: vi.fn(async () => []),
  }
  const connector = {
    connect: vi.fn(async ({ machineId }: { machineId: string }) => sessionFor(machineId)),
  }
  const workspace = render(
    <MachineWorkspace api={api} connector={connector} initialMachine={currentMachine} onBack={onBack} />,
  )
  return {
    ...workspace,
    rerenderMachine(machine: Machine) {
      currentMachine = machine
      workspace.rerender(
        <MachineWorkspace api={api} connector={connector} initialMachine={currentMachine} onBack={onBack} />,
      )
    },
  }
}

describe('MachineWorkspace FileManager loading', () => {
  beforeEach(async () => {
    fileManagerLoader.load.mockReset()
    fileManagerLoader.reload.mockReset()
    fileManagerWorkspaceBackHandler.mockReset()
    fileManagerLoader.load.mockResolvedValue(StatefulFileManager)
    await anyttyI18n.changeLanguage('en')
  })

  afterEach(() => {
    cleanup()
  })

  it('loads on first open, stays mounted while closed, and reuses the loaded module', async () => {
    renderWorkspace()

    await screen.findByTestId('anytty-terminal-list-page')
    expect(fileManagerLoader.load).not.toHaveBeenCalled()
    expect(screen.queryByTestId('anytty-machine-files-overlay')).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    const stateInput = await screen.findByRole('textbox', { name: 'Lazy file manager state' })
    expect(fileManagerLoader.load).toHaveBeenCalledTimes(1)
    await userEvent.type(stateInput, '/remembered')

    await userEvent.click(screen.getByRole('button', { name: 'Close files' }))
    await waitFor(() => expect(screen.getByTestId('mock-file-manager').dataset.active).toBe('false'))
    expect(screen.getByRole('textbox', { name: 'Lazy file manager state' })).toBe(stateInput)

    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    await waitFor(() => expect(screen.getByTestId('mock-file-manager').dataset.active).toBe('true'))
    expect(screen.getByRole<HTMLInputElement>('textbox', { name: 'Lazy file manager state' }).value).toBe('/remembered')
    expect(fileManagerLoader.load).toHaveBeenCalledTimes(1)
  })

  it('uses its own left back button to return to the terminal list without leaving the device', async () => {
    const onBack = vi.fn()
    renderWorkspace(undefined, onBack)

    await screen.findByTestId('anytty-terminal-list-page')
    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    const overlay = await screen.findByTestId('anytty-machine-files-overlay')

    expect(overlay.classList.contains('z-[60]')).toBe(true)
    await userEvent.click(screen.getByRole('button', { name: 'Close files' }))

    expect(overlay.classList.contains('invisible')).toBe(true)
    expect(screen.getByTestId('anytty-terminal-list-page')).toBeTruthy()
    expect(onBack).not.toHaveBeenCalled()
  })

  it('closes files with one native back and returns to the terminal before its internal workspace handler', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cwd: '/srv/project', cols: 80, rows: 24,
    }
    const session = new MockProtoSession(
      machine.machineId,
      () => protoResult('acknowledge', create(AcknowledgeResultSchema)),
    )
    fileManagerLoader.load.mockResolvedValue(FileManagerWithWorkspaceBackHandler)

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      initialTerminalId={terminal.terminalId}
      singlePane
    />)

    await screen.findByTestId('anytty-terminal-header')
    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    await waitFor(() => expect(screen.getByTestId('mock-file-manager').dataset.active).toBe('true'))

    act(() => { expect(dispatchNativeBack()).toBe(true) })

    await waitFor(() => expect(screen.getByTestId('mock-file-manager').dataset.active).toBe('false'))
    expect(screen.getByTestId('anytty-terminal-header')).toBeTruthy()
    expect(screen.queryByTestId('anytty-terminal-list-page')).toBeNull()
    expect(fileManagerWorkspaceBackHandler).not.toHaveBeenCalled()
  })

  it('contains an import rejection and reloads the application without exposing the raw error', async () => {
    const rawError = 'chunk unavailable from https://private.invalid/file-manager.js'
    fileManagerLoader.load.mockRejectedValueOnce(new Error(rawError))
    renderWorkspace()

    await screen.findByTestId('anytty-terminal-list-page')
    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))

    const alert = await screen.findByRole('alert')
    expect(alert.textContent).toContain('Files could not be loaded.')
    expect(document.body.textContent).not.toContain(rawError)
    expect(fileManagerLoader.load).toHaveBeenCalledTimes(1)

    await userEvent.click(screen.getByRole('button', { name: 'Reload application' }))
    expect(fileManagerLoader.reload).toHaveBeenCalledTimes(1)
    expect(fileManagerLoader.load).toHaveBeenCalledTimes(1)
  })

  it('reuses one pending load across a rapid close and reopen', async () => {
    const pending = createDeferred<FileManagerComponent>()
    fileManagerLoader.load.mockReturnValueOnce(pending.promise)
    renderWorkspace()

    await screen.findByTestId('anytty-terminal-list-page')
    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    expect(fileManagerLoader.load).toHaveBeenCalledTimes(1)

    await userEvent.click(screen.getByRole('button', { name: 'Close files' }))
    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    expect(fileManagerLoader.load).toHaveBeenCalledTimes(1)

    await act(async () => pending.resolve(StatefulFileManager))
    expect(await screen.findByRole('textbox', { name: 'Lazy file manager state' })).not.toBeNull()
    expect(screen.getByTestId('mock-file-manager').dataset.active).toBe('true')
  })

  it('ignores a pending load after unmount', async () => {
    const pending = createDeferred<FileManagerComponent>()
    let renders = 0
    const CountingFileManager = () => {
      renders += 1
      return <div data-testid="counting-file-manager" />
    }
    fileManagerLoader.load.mockReturnValueOnce(pending.promise)
    const workspace = renderWorkspace()

    await screen.findByTestId('anytty-terminal-list-page')
    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    workspace.unmount()
    await act(async () => pending.resolve(CountingFileManager))

    expect(renders).toBe(0)
  })

  it('invalidates a pending load when the machine context changes', async () => {
    const pending = createDeferred<FileManagerComponent>()
    fileManagerLoader.load.mockReturnValue(pending.promise)
    const workspace = renderWorkspace()

    await screen.findByTestId('anytty-terminal-list-page')
    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))

    workspace.rerenderMachine({ machineId: 'lab', name: 'Lab', state: 'online' })
    await waitFor(() => expect(screen.queryByTestId('anytty-machine-files-overlay')).toBeNull())
    await act(async () => pending.resolve(StatefulFileManager))
    expect(screen.queryByTestId('mock-file-manager')).toBeNull()

    await userEvent.click(screen.getByRole('button', { name: 'Open files' }))
    expect(await screen.findByRole('textbox', { name: 'Lazy file manager state' })).not.toBeNull()
    expect(fileManagerLoader.load).toHaveBeenCalledTimes(2)
  })

  it('reattaches an open file manager after the native session reconnects without terminals', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const staleSession = new MockProtoSession('studio')
    const freshSession = new MockProtoSession('studio')
    const connect = vi.fn()
      .mockResolvedValueOnce(staleSession)
      .mockResolvedValueOnce(freshSession)
    let publishConnectionState: ((snapshot: RtcConnectionStateSnapshot) => void) | undefined

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => []),
      }}
      connector={{ connect }}
      connectionStateEvents={{
        subscribe(_machineId, handler) {
          publishConnectionState = handler
          return { close() {} }
        },
      }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open files' }))
    await waitFor(() => expect(connect).toHaveBeenCalledTimes(1))
    expect((await screen.findByTestId('mock-file-manager')).dataset.active).toBe('true')

    act(() => {
      publishConnectionState?.({
        machineId: machine.machineId,
        phase: 'waiting_network',
        statusText: 'Waiting for network',
        relayInUse: false,
      })
      staleSession.emitClosed(Object.assign(new Error('network changed'), {
        code: 'unavailable',
        retryable: true,
      }))
    })
    await waitFor(() => expect(screen.queryByTestId('mock-file-manager')).toBeNull())

    act(() => publishConnectionState?.({
      machineId: machine.machineId,
      phase: 'connected',
      statusText: 'Connected',
      relayInUse: false,
    }))

    await waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(screen.getByTestId('mock-file-manager').dataset.active).toBe('true'))
  })
})
