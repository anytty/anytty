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

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, reject, resolve }
}

function renderWorkspace(initialMachine: Machine = { machineId: 'studio', name: 'Studio', state: 'online' }) {
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
    <MachineWorkspace api={api} connector={connector} initialMachine={currentMachine} />,
  )
  return {
    ...workspace,
    rerenderMachine(machine: Machine) {
      currentMachine = machine
      workspace.rerender(
        <MachineWorkspace api={api} connector={connector} initialMachine={currentMachine} />,
      )
    },
  }
}

describe('MachineWorkspace FileManager loading', () => {
  beforeEach(async () => {
    fileManagerLoader.load.mockReset()
    fileManagerLoader.reload.mockReset()
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
})
