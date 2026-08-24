import { create } from '@bufbuild/protobuf'
import { act, cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { forwardRef, useImperativeHandle } from 'react'
import { AcknowledgeResultSchema } from '../generated/apipb/application_pb'
import {
  TerminalCreateResultSchema,
  TerminalDefaultsResultSchema,
  TerminalDefaultsSchema,
  TerminalInfoSchema,
  TerminalRefSchema,
} from '../generated/apipb/terminal_pb'
import { anyttyI18n } from '../i18n'
import { dispatchNativeBack } from '../platform/nativeBack'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { DEFAULT_TERMINAL_SETTINGS } from '../terminal/terminalSettings'
import type { RtcConnectionStateSnapshot } from '../core/transport'
import { MachineWorkspace } from './MachineWorkspace'
import { ConnectionRecoveryOverlayProvider } from '../connection/ConnectionRecoveryOverlay'

const terminalRender = vi.hoisted(() => vi.fn())
const terminalSendInput = vi.hoisted(() => vi.fn())
const terminalPasteText = vi.hoisted(() => vi.fn())
const terminalFocus = vi.hoisted(() => vi.fn())
const terminalBlur = vi.hoisted(() => vi.fn())
const terminalFit = vi.hoisted(() => vi.fn())
const terminalReattach = vi.hoisted(() => vi.fn())
const terminalRequestResizeOwner = vi.hoisted(() => vi.fn())
const terminalReleaseResizeOwner = vi.hoisted(() => vi.fn())
const terminalSetResizeLock = vi.hoisted(() => vi.fn())
const terminalOpenHistorySearch = vi.hoisted(() => vi.fn())
const terminalHarness = vi.hoisted(() => ({ exposeHandle: true, selection: '' }))
const originalInnerWidth = window.innerWidth

vi.mock('../terminal/Terminal', () => ({
  Terminal: forwardRef(function MockTerminal(props: unknown, ref) {
    terminalRender(props)
    const terminalId = (props as { terminalId: string }).terminalId
    useImperativeHandle(ref, () => terminalHarness.exposeHandle ? ({
      sendInput: (data: string) => terminalSendInput(terminalId, data),
      sendResize: () => {},
      requestResizeOwner: () => terminalRequestResizeOwner(terminalId),
      releaseResizeOwner: () => terminalReleaseResizeOwner(terminalId),
      setResizeLock: async (locked: boolean) => {
        terminalSetResizeLock(terminalId, locked)
        return { canResize: !locked, reason: locked ? 'size_locked' : 'owner', sizeLocked: locked }
      },
      focus: () => terminalFocus(terminalId),
      blur: () => terminalBlur(terminalId),
      fit: () => terminalFit(terminalId),
      openHistorySearch: () => terminalOpenHistorySearch(terminalId),
      reattach: (...args: unknown[]) => terminalReattach(terminalId, ...args),
      selectAll: () => { terminalHarness.selection = 'selected terminal text' },
      selectVisible: () => { terminalHarness.selection = 'selected terminal text' },
      getSelection: () => terminalHarness.selection,
      getSelectionForClipboard: async () => terminalHarness.selection,
      hasSelection: () => terminalHarness.selection !== '',
      clearSelection: () => { terminalHarness.selection = '' },
      pasteText: (text: string) => terminalPasteText(terminalId, text),
      getCursorInfo: () => null,
      adjustInputPosition: () => {},
      getBufferType: () => 'normal',
      updateOptions: () => {},
    }) : null)
    return <div data-history-only={(props as { historyOnly?: boolean }).historyOnly ? 'true' : undefined} data-terminal-id={terminalId} data-testid="mock-terminal" />
  }),
}))

async function openTerminalTools() {
  await userEvent.click(screen.getByTestId('anytty-terminal-tools-button'))
}

describe('MachineWorkspace terminal creation', () => {
  beforeEach(async () => {
    terminalRender.mockReset()
    terminalSendInput.mockReset().mockReturnValue(true)
    terminalPasteText.mockReset().mockReturnValue(true)
    terminalFocus.mockReset()
    terminalBlur.mockReset()
    terminalFit.mockReset()
    terminalReattach.mockReset()
    terminalRequestResizeOwner.mockReset().mockResolvedValue({ canResize: true, reason: 'owner' })
    terminalReleaseResizeOwner.mockReset().mockResolvedValue({ canResize: false, reason: 'follower' })
    terminalSetResizeLock.mockReset()
    terminalOpenHistorySearch.mockReset()
    terminalHarness.exposeHandle = true
    terminalHarness.selection = ''
    await anyttyI18n.changeLanguage('en')
  })
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: originalInnerWidth })
  })

  it('retains native connection demand only while the workspace is mounted', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const releaseDemand = vi.fn()
    const retainConnectionDemand = vi.fn(() => releaseDemand)
    const view = render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => []),
      }}
      connector={{ connect: vi.fn(async () => new MockProtoSession('studio')) }}
      initialMachine={machine}
      retainConnectionDemand={retainConnectionDemand}
    />)

    await waitFor(() => expect(retainConnectionDemand).toHaveBeenCalledOnce())
    expect(releaseDemand).not.toHaveBeenCalled()

    view.unmount()
    expect(releaseDemand).toHaveBeenCalledOnce()
  })

  it('keeps native single-pane navigation at landscape widths', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 1024 })
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio')
    const view = render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      singlePane
    />)

    const workspace = view.container.querySelector('[data-machine-id="studio"]')
    const terminalList = await screen.findByTestId('anytty-terminal-list-page')
    expect(workspace?.className).not.toContain('md:flex-row')
    expect(terminalList.className).not.toContain('md:flex')

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))

    expect(terminalList.className.split(/\s+/)).toContain('hidden')
    expect(terminalList.className).not.toContain('md:flex')
    expect((await screen.findByTestId('anytty-terminal-header')).className).not.toContain('md:hidden')
    expect(screen.getByTestId('anytty-terminal-body').className).not.toContain('md:row-start-1')
  })

  it('keeps Web terminals as reorderable view tabs without stopping daemon terminals', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = [
      { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, command: '/bin/zsh', cwd: '/work/shell', cols: 80, rows: 24 },
      { terminalId: 'term-logs', machineId: 'studio', title: 'Logs', state: 'running' as const, command: 'tail -f app.log', cwd: '/work/logs', cols: 80, rows: 24 },
    ]
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      webLayout
    />)

    expect(await screen.findByTestId('anytty-web-workbench-bar')).toBeTruthy()
    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Open Logs' }))
    expect(screen.getAllByRole('tab')).toHaveLength(2)
    expect(screen.getByRole('tab', { name: 'Logs' }).getAttribute('aria-selected')).toBe('true')

    await userEvent.click(screen.getByRole('button', { name: 'Close Logs' }))
    expect(screen.getByRole('tab', { name: 'Shell' }).getAttribute('aria-selected')).toBe('true')
    expect(session.commands.some((command) => command.command.case === 'terminalRemove')).toBe(false)

    await userEvent.click(screen.getAllByRole('button', { name: 'Settings' })[0]!)
    expect(await screen.findByRole('dialog', { name: 'Settings' })).toBeTruthy()
  })

  it('presents Web terminal workflows as full-viewport dialogs and submits the editor with Enter', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, command: '/bin/zsh', cwd: '/work/shell', cols: 80, rows: 24 }
    const session = new MockProtoSession('studio', (command) => {
      if (command.command.case === 'terminalDefaults') {
        return protoResult('terminalDefaults', create(TerminalDefaultsResultSchema, {
          defaults: create(TerminalDefaultsSchema, { defaultCommand: ['/bin/zsh'], defaultCwd: '/work' }),
        }))
      }
      if (command.command.case === 'terminalCreate') {
        return protoResult('terminalCreate', create(TerminalCreateResultSchema, {
          terminal: create(TerminalInfoSchema, {
            ref: create(TerminalRefSchema, { endpointId: 'studio', terminalId: 'term-created' }),
          }),
        }))
      }
      return protoResult('acknowledge', create(AcknowledgeResultSchema))
    })
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      webLayout
    />)

    await userEvent.click((await screen.findAllByRole('button', { name: 'Create terminal' }))[0]!)
    const editor = await screen.findByTestId('anytty-terminal-editor-sheet')
    expect(editor.dataset.presentation).toBe('dialog')
    expect(editor.parentElement).toBe(document.body)
    expect(within(editor).getByRole('dialog', { name: 'New terminal' }).className).toContain('md:max-w-2xl')
    const name = within(editor).getByLabelText('Name') as HTMLInputElement
    await waitFor(() => expect(document.activeElement).toBe(name))
    await waitFor(() => expect((within(editor).getByLabelText('Command') as HTMLInputElement).value).toBe('/bin/zsh'))
    await userEvent.type(name, 'Web shell{enter}')
    await waitFor(() => expect(session.commands.some((command) => command.command.case === 'terminalCreate')).toBe(true))
  })

  it('turns a Web terminal drag into a directional resizable split', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = [
      { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, command: '/bin/zsh', cols: 80, rows: 24 },
      { terminalId: 'term-logs', machineId: 'studio', title: 'Logs', state: 'running' as const, command: 'tail -f app.log', cols: 80, rows: 24 },
    ]
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      webLayout
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const logsItem = (await screen.findByRole('button', { name: 'Open Logs' })).closest('[data-terminal-id]')!
    const dragValues = new Map<string, string>()
    const dataTransfer = {
      effectAllowed: 'all', dropEffect: 'move', files: [], items: [], types: [],
      clearData: vi.fn(), getData: (type: string) => dragValues.get(type) ?? '',
      setData: (type: string, value: string) => { dragValues.set(type, value) }, setDragImage: vi.fn(),
    }
    fireEvent.dragStart(logsItem, { dataTransfer })
    const splitRight = await screen.findByRole('button', { name: 'Split right' })
    fireEvent.drop(splitRight, { dataTransfer })

    expect(await screen.findByTestId('anytty-split-terminal-panel')).toBeTruthy()
    expect(screen.getByRole('separator', { name: 'Resize terminal panes' }).getAttribute('aria-orientation')).toBe('vertical')
    expect(screen.queryByRole('button', { name: 'Open here' })).toBeNull()
  })

  it('adds three, four, and five panes below without opening a chooser', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = ['Shell', 'Logs', 'Server', 'Tests', 'Watch'].map((title, index) => ({
      terminalId: `term-${title.toLowerCase()}`,
      machineId: 'studio',
      title,
      state: 'running' as const,
      command: index === 0 ? '/bin/zsh' : title.toLowerCase(),
      cols: 80,
      rows: 24,
    }))
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      webLayout
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const splitBelow = screen.getByTestId('anytty-terminal-split-button')
    await userEvent.click(splitBelow)
    await userEvent.click(splitBelow)
    await userEvent.click(splitBelow)
    await userEvent.click(splitBelow)

    await waitFor(() => expect(screen.getAllByTestId('mock-terminal')).toHaveLength(5))
    expect(screen.getAllByTestId('anytty-split-terminal-panel')).toHaveLength(4)
    expect(screen.queryByTestId('anytty-split-terminal-sheet')).toBeNull()
    expect(screen.getAllByRole('separator', { name: 'Resize terminal panes' })).toHaveLength(4)
    expect(document.querySelectorAll('[data-split-direction="rows"]')).toHaveLength(4)
  })

  it('combines four split directions into nested mobile pane layouts', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = ['Shell', 'Logs', 'Server', 'Tests', 'Watch'].map((title) => ({
      terminalId: `term-${title.toLowerCase()}`,
      machineId: 'studio',
      title,
      state: 'running' as const,
      command: title.toLowerCase(),
      cols: 80,
      rows: 24,
    }))
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    for (const direction of ['Split right', 'Split below', 'Split left', 'Split above']) {
      await userEvent.click(screen.getByTestId('anytty-terminal-tools-button'))
      const toolbar = await screen.findByTestId('anytty-terminal-action-toolbar')
      await userEvent.click(within(toolbar).getByRole('button', { name: direction }))
    }

    await waitFor(() => expect(screen.getAllByTestId('mock-terminal')).toHaveLength(5))
    expect(screen.getAllByTestId('anytty-split-terminal-panel')).toHaveLength(4)
    expect(document.querySelectorAll('[data-split-direction="columns"]')).toHaveLength(2)
    expect(document.querySelectorAll('[data-split-direction="rows"]')).toHaveLength(2)
  })

  it('prefills daemon defaults and submits a complete generated Proto create command', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const session = new MockProtoSession('studio', (command) => {
      if (command.command.case === 'terminalDefaults') {
        return protoResult('terminalDefaults', create(TerminalDefaultsResultSchema, {
          defaults: create(TerminalDefaultsSchema, { defaultCommand: ['/bin/fish'], defaultCwd: '/home/ada' }),
        }))
      }
      if (command.command.case === 'terminalCreate') {
        return protoResult('terminalCreate', create(TerminalCreateResultSchema, {
          terminal: create(TerminalInfoSchema, {
            ref: create(TerminalRefSchema, { endpointId: 'studio', terminalId: 'term-created' }),
          }),
        }))
      }
      return protoResult('acknowledge', create(AcknowledgeResultSchema))
    })
    const listTerminals = vi.fn(async () => [])

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals,
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Create terminal' }))
    const sheet = await screen.findByTestId('anytty-terminal-editor-sheet')
    await waitFor(() => expect((within(sheet).getByLabelText('Command') as HTMLInputElement).value).toBe('/bin/fish'))
    expect((within(sheet).getByLabelText('Working directory') as HTMLInputElement).value).toBe('/home/ada')

    await userEvent.click(within(sheet).getByRole('button', { name: 'Add variable' }))
    await userEvent.type(within(sheet).getByLabelText('Key'), 'MODE')
    await userEvent.type(within(sheet).getByLabelText('Value'), 'mobile')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Create terminal' }))

    await waitFor(() => expect(session.commands.some((command) => command.command.case === 'terminalCreate')).toBe(true))
    const createCommand = session.commands.find((command) => command.command.case === 'terminalCreate')
    expect(createCommand?.command.value).toMatchObject({
      terminal: {
        terminalId: expect.stringMatching(/^term-/),
        command: ['/bin/fish'],
        cwd: '/home/ada',
        env: ['MODE=mobile'],
        size: { cols: 80, rows: 24 },
      },
    })
  })

  it('keeps terminal actions and a compact two-line identity visible in the header', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cwd: '/home/ada', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const header = await screen.findByTestId('anytty-terminal-header')
    const terminalSwitcher = within(header).getByRole('button', { name: 'Switch terminal: Studio · Shell · /home/ada' })
    const terminalSummary = within(terminalSwitcher).getByTestId('anytty-terminal-title')
    expect(terminalSwitcher.getAttribute('title')).toBe('Studio · Shell · /home/ada')
    expect(terminalSwitcher.classList.contains('flex-col')).toBe(true)
    expect(terminalSummary.textContent).toBe('Studio/Shell')
    expect(terminalSummary.querySelectorAll('.truncate')).toHaveLength(2)
    expect(within(terminalSwitcher).getByTestId('anytty-terminal-path').textContent).toBe('/home/ada')
    expect(within(header).getByRole('button', { name: 'Open files' })).toBeTruthy()
    expect(within(header).getByRole('button', { name: 'Split below' })).toBeTruthy()
    expect(within(header).queryByRole('button', { name: 'Control resize' })).toBeNull()
    expect(within(header).getByRole('button', { name: 'Terminal tools' })).toBeTruthy()
    expect(within(header).queryByTestId('anytty-terminal-menu-button')).toBeNull()
    expect(header.className).toContain('min-h-11')
    expect(Array.from(header.querySelectorAll('button')).every((button) => (
      button.className.includes('h-9') || button.className.includes('min-h-9') || button.className.includes('h-11')
    ))).toBe(true)

    await userEvent.click(terminalSwitcher)
    const switcherSheet = await screen.findByTestId('anytty-terminal-switcher-sheet')
    const dialog = within(switcherSheet).getByRole('dialog')
    const handle = within(switcherSheet).getByRole('button', { name: 'Expand panel' })
    expect(dialog.className).toContain('h-[60dvh]')
    await userEvent.click(handle)
    expect(dialog.className).toContain('h-[85dvh]')
    expect(handle.getAttribute('aria-expanded')).toBe('true')
    fireEvent.pointerDown(handle, { pointerId: 1, clientY: 100 })
    fireEvent.pointerMove(handle, { pointerId: 1, clientY: 150 })
    fireEvent.pointerUp(handle, { pointerId: 1, clientY: 150 })
    expect(dialog.className).toContain('h-[60dvh]')
  })

  it('loads other machines only when expanded and switches directly to their terminal', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const remoteTerminal = {
      terminalId: 'term-logs', machineId: 'server', title: 'Logs', state: 'running' as const,
      command: 'tail -f app.log', cols: 80, rows: 24,
    }
    const loadMachineTerminals = vi.fn(async () => [remoteTerminal])
    const onSwitchTerminal = vi.fn()
    const session = new MockProtoSession('studio')

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      terminalSwitcherMachines={[
        { machineId: 'studio', name: 'Studio', terminalCount: 1 },
        { machineId: 'server', name: 'Server', terminalCount: 1 },
      ]}
      loadMachineTerminals={loadMachineTerminals}
      onSwitchTerminal={onSwitchTerminal}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await userEvent.click(screen.getByRole('button', { name: /Switch terminal:/ }))
    const groups = await screen.findByTestId('anytty-terminal-machine-groups')
    expect(within(groups).queryByRole('button', { name: 'Open Logs' })).toBeNull()
    const serverGroup = groups.querySelector('[data-switcher-machine-id="server"]') as HTMLElement
    await userEvent.click(within(serverGroup).getByRole('button', { name: /Server/ }))

    expect(loadMachineTerminals).toHaveBeenCalledTimes(1)
    await userEvent.click(await within(serverGroup).findByRole('button', { name: 'Open Logs' }))
    expect(onSwitchTerminal).toHaveBeenCalledWith({ machineId: 'server', terminalId: 'term-logs' })
  })

  it('persists keyboard layout mode for the same terminal', async () => {
    const values = new Map<string, string>()
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => { values.set(key, value) },
      removeItem: (key: string) => { values.delete(key) },
    }
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, cols: 80, rows: 24 }
    const renderWorkspace = () => render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => new MockProtoSession('studio')) }}
      initialMachine={machine}
      storage={storage}
    />)

    const first = renderWorkspace()
    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(screen.getByRole('button', { name: 'Always Shift' }))
    expect(Array.from(values.entries())).toContainEqual([
      'anytty.terminal.keyboard-mode.v1:studio:term-shell',
      'shift',
    ])
    first.unmount()

    renderWorkspace()
    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    expect(screen.getByRole('button', { name: 'Always Shift' }).getAttribute('aria-pressed')).toBe('true')
  })

  it('opens history search for the active terminal from the tools toolbar', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, cols: 80, rows: 24 }
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => new MockProtoSession('studio')) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(screen.getByRole('button', { name: 'Search history' }))

    expect(terminalOpenHistorySearch).toHaveBeenCalledOnce()
    expect(terminalOpenHistorySearch).toHaveBeenCalledWith('term-shell')
    expect(screen.queryByTestId('anytty-terminal-action-toolbar')).toBeNull()
  })

  it('pastes from the injected system clipboard without reading remote clipboard history', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, cols: 80, rows: 24 }
    const readText = vi.fn(async () => 'from phone clipboard')
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => new MockProtoSession('studio')) }}
      initialMachine={machine}
      systemClipboard={{ readText, writeText: vi.fn() }}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(screen.getByRole('button', { name: 'Paste' }))

    await waitFor(() => expect(terminalPasteText).toHaveBeenCalledWith('term-shell', 'from phone clipboard'))
    expect(readText).toHaveBeenCalledOnce()
  })

  it('shows a localized paste failure instead of the native clipboard exception', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, cols: 80, rows: 24 }
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => new MockProtoSession('studio')) }}
      initialMachine={machine}
      systemClipboard={{
        readText: vi.fn(async () => { throw new Error("Failed to execute 'readText' on 'Clipboard': Read permission denied.") }),
        writeText: vi.fn(),
      }}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(screen.getByRole('button', { name: 'Paste' }))

    expect((await screen.findAllByText('Unable to read the system clipboard')).length).toBeGreaterThan(0)
    expect(screen.queryByText(/Read permission denied/)).toBeNull()
  })

  it('locks terminal size independently from owner permission', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, cwd: '/srv/shell', cols: 80, rows: 24 }
    const session = new MockProtoSession('studio')
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(screen.getByRole('button', { name: /Acquire owner permission/ }))
    await waitFor(() => expect(screen.getByRole('button', { name: /Release owner permission/ })).toBeTruthy())
    terminalFocus.mockClear()
    await userEvent.click(screen.getByRole('button', { name: 'Lock terminal size' }))

    await waitFor(() => expect(terminalSetResizeLock).toHaveBeenCalledWith('term-shell', true))
    await waitFor(() => expect(session.commands.some((command) => command.command.case === 'terminalSetTags')).toBe(true))
    const command = session.commands.find((item) => item.command.case === 'terminalSetTags')
    expect(command?.command.value).toMatchObject({ tags: { 'anytty.size_lock': 'lock', cwd: '/srv/shell' } })
    expect(session.commands.some((item) => item.command.case === 'terminalSetMetadata')).toBe(false)
    await waitFor(() => expect(terminalFit).toHaveBeenCalledWith('term-shell'))
    expect(terminalFocus).not.toHaveBeenCalled()
  })

  it('lets a locked follower take owner permission and then unlock the terminal size', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, cols: 80, rows: 24 }
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => new MockProtoSession('studio')) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const terminalProps = terminalRender.mock.calls.at(-1)?.[0] as {
      onResizeControl: (control: { canResize: boolean; reason: string; sizeLocked?: boolean }) => void
    }
    act(() => terminalProps.onResizeControl({ canResize: false, reason: 'follower', sizeLocked: true }))
    terminalRequestResizeOwner.mockResolvedValueOnce({ canResize: false, reason: 'size_locked', sizeLocked: true })

    await openTerminalTools()
    const acquireOwnerButton = screen.getByRole('button', { name: /Acquire owner permission/ })
    expect(acquireOwnerButton.hasAttribute('disabled')).toBe(false)
    expect(screen.getByRole('button', { name: 'Unlock terminal size' }).hasAttribute('disabled')).toBe(true)

    await userEvent.click(acquireOwnerButton)
    await waitFor(() => expect(screen.getByRole('button', { name: /Release owner permission/ })).toBeTruthy())
    const unlockButton = screen.getByRole('button', { name: 'Unlock terminal size' })
    expect(unlockButton.hasAttribute('disabled')).toBe(false)
    await userEvent.click(unlockButton)

    await waitFor(() => expect(terminalSetResizeLock).toHaveBeenCalledWith('term-shell', false))
  })

  it('keeps resize-control failures local instead of reporting a network failure', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = { terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const, cols: 80, rows: 24 }
    const session = new MockProtoSession('studio')
    render(<ConnectionRecoveryOverlayProvider><MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    /></ConnectionRecoveryOverlayProvider>)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(screen.getByRole('button', { name: /Acquire owner permission/ }))
    await waitFor(() => expect(screen.getByRole('button', { name: /Release owner permission/ })).toBeTruthy())
    terminalSetResizeLock.mockImplementationOnce(() => { throw new Error('terminal resize lock is not available') })
    await userEvent.click(screen.getByRole('button', { name: 'Lock terminal size' }))

    expect(await screen.findByText('Could not update the terminal size lock')).toBeTruthy()
    expect(screen.queryByTestId('anytty-connection-recovery-overlay')).toBeNull()
  })

  it('opens an exited split terminal as read-only history', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = [
      {
        terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
        command: '/bin/zsh', cols: 80, rows: 24,
      },
      {
        terminalId: 'term-logs', machineId: 'studio', title: 'Logs', state: 'exited' as const,
        command: 'tail -f app.log', cols: 80, rows: 24,
      },
    ]
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    await userEvent.click(screen.getByTestId('anytty-terminal-split-button'))

    const splitPanel = await screen.findByTestId('anytty-split-terminal-panel')
    expect(splitPanel.querySelector('[data-testid="mock-terminal"][data-terminal-id="term-logs"][data-history-only="true"]')).toBeTruthy()
    expect(screen.queryByText('Terminal exited')).toBeNull()
  })

  it('keeps a displayed split terminal mounted in history mode when its process exits', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const runningTerminals = [
      {
        terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
        command: '/bin/zsh', cols: 80, rows: 24,
      },
      {
        terminalId: 'term-logs', machineId: 'studio', title: 'Logs', state: 'running' as const,
        command: 'tail -f app.log', cols: 80, rows: 24,
      },
    ]
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    const listTerminals = vi.fn(async () => runningTerminals)

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals,
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    await userEvent.click(screen.getByTestId('anytty-terminal-split-button'))
    const splitPanel = await screen.findByTestId('anytty-split-terminal-panel')
    await waitFor(() => expect(splitPanel.querySelector('[data-testid="mock-terminal"][data-terminal-id="term-logs"]')).toBeTruthy())

    const splitProps = [...terminalRender.mock.calls]
      .reverse()
      .map(([props]) => props as { terminalId: string; onTerminalInfoChange?: (terminal: typeof runningTerminals[number]) => void })
      .find((props) => props.terminalId === 'term-logs')
    act(() => {
      splitProps?.onTerminalInfoChange?.({ ...runningTerminals[1]!, state: 'exited' })
    })

    await waitFor(() => expect(splitPanel.querySelector('[data-testid="mock-terminal"][data-terminal-id="term-logs"][data-history-only="true"]')).toBeTruthy())
    expect(screen.queryByText('Terminal exited')).toBeNull()
  })

  it('keeps moved session actions in the unified toolbar while split', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 320 })
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = [
      {
        terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
        command: '/bin/zsh', cols: 80, rows: 24,
      },
      {
        terminalId: 'term-logs', machineId: 'studio', title: 'Logs', state: 'running' as const,
        command: 'tail -f app.log', cols: 80, rows: 24,
      },
    ]
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      terminalSettings={DEFAULT_TERMINAL_SETTINGS}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const header = await screen.findByTestId('anytty-terminal-header')
    expect(window.innerWidth).toBe(320)

    await userEvent.click(within(header).getByTestId('anytty-terminal-split-button'))
    expect(await screen.findByTestId('anytty-split-terminal-panel')).toBeTruthy()

    const toolsButton = within(header).getByTestId('anytty-terminal-tools-button')
    await userEvent.click(toolsButton)
    const toolbar = screen.getByTestId('anytty-terminal-action-toolbar')
    expect(screen.getAllByTestId('anytty-terminal-action-toolbar')).toHaveLength(1)
    expect(within(toolbar).getByRole('button', { name: 'Decrease terminal font size' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Increase terminal font size' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Renderer: Auto' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Paste' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Clipboard' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Snippets' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Connection' })).toBeTruthy()
    expect(within(toolbar).getByRole('button', { name: 'Sync input' })).toBeTruthy()
    expect(within(toolbar).getByRole('button', { name: 'Close split' })).toBeTruthy()
    expect(Array.from(toolbar.querySelectorAll('button')).every((button) => (
      button.classList.contains('h-11') || button.classList.contains('min-h-11')
    ))).toBe(true)

    await userEvent.click(within(toolbar).getByRole('button', { name: 'Select' }))
    const selectionToolbar = screen.getByTestId('anytty-terminal-action-toolbar')
    const copyButton = within(selectionToolbar).getByRole('button', { name: 'Copy' })
    expect(copyButton.classList.contains('min-h-11')).toBe(true)
    expect((copyButton as HTMLButtonElement).disabled).toBe(true)
    await userEvent.click(within(selectionToolbar).getByRole('button', { name: 'Select all' }))
    expect((copyButton as HTMLButtonElement).disabled).toBe(false)
    expect(Array.from(selectionToolbar.querySelectorAll('button')).every((button) => (
      button.classList.contains('h-11') || button.classList.contains('min-h-11')
    ))).toBe(true)

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByTestId('anytty-terminal-action-toolbar')).toBeNull()
    expect(screen.getByTestId('anytty-split-terminal-panel')).toBeTruthy()
    expect(document.activeElement).toBe(toolsButton)
  })

  it('restores the unified tools trigger after Escape from the default toolbar', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      terminalSettings={DEFAULT_TERMINAL_SETTINGS}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const header = await screen.findByTestId('anytty-terminal-header')
    const toolsButton = within(header).getByTestId('anytty-terminal-tools-button')
    expect(window.innerWidth).toBe(390)

    await openTerminalTools()
    const toolbar = screen.getByTestId('anytty-terminal-action-toolbar')
    expect(document.activeElement).toBe(toolsButton)
    await userEvent.tab()
    await userEvent.tab()
    expect(toolbar.contains(document.activeElement)).toBe(true)
    await userEvent.keyboard('{Escape}')

    expect(screen.queryByTestId('anytty-terminal-action-toolbar')).toBeNull()
    expect(document.activeElement).toBe(toolsButton)
  })

  it('keeps the unified toolbar available when the viewport narrows', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      terminalSettings={DEFAULT_TERMINAL_SETTINGS}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const toolsButton = screen.getByTestId('anytty-terminal-tools-button')
    await openTerminalTools()
    const toolbar = screen.getByTestId('anytty-terminal-action-toolbar')
    await userEvent.tab()
    await userEvent.tab()
    expect(toolbar.contains(document.activeElement)).toBe(true)

    act(() => {
      Object.defineProperty(window, 'innerWidth', { configurable: true, value: 320 })
      window.dispatchEvent(new Event('resize'))
    })

    expect(screen.getByTestId('anytty-terminal-action-toolbar')).toBeTruthy()
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByTestId('anytty-terminal-action-toolbar')).toBeNull()
    expect(document.activeElement).toBe(toolsButton)
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.getByTestId('anytty-terminal-list-page')).toBeTruthy()
  })

  it('clears toolbar state and its opener across a terminal-list round trip', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      terminalSettings={DEFAULT_TERMINAL_SETTINGS}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(within(screen.getByTestId('anytty-terminal-action-toolbar')).getByRole('button', { name: 'Select' }))
    expect(screen.getByRole('button', { name: 'Cancel selection' })).toBeTruthy()

    await userEvent.click(screen.getByRole('button', { name: 'Back to terminal list' }))
    expect(screen.getByTestId('anytty-terminal-list-page')).toBeTruthy()
    expect(screen.queryByTestId('anytty-terminal-action-toolbar')).toBeNull()
    expect(dispatchNativeBack()).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'Open Shell' }))
    expect(screen.queryByTestId('anytty-terminal-action-toolbar')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Cancel selection' })).toBeNull()
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.getByTestId('anytty-terminal-list-page')).toBeTruthy()
  })

  it('never mounts terminal resources with a stale generation while reconnecting', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const staleSession = new MockProtoSession('studio')
    const freshSession = new MockProtoSession('studio')
    const connect = vi.fn()
      .mockResolvedValueOnce(staleSession)
      .mockResolvedValueOnce(freshSession)

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    expect(terminalRender.mock.calls.some(([props]) => (props as { session: MockProtoSession }).session === staleSession)).toBe(true)

    await userEvent.click(screen.getByRole('button', { name: 'Back to terminal list' }))
    await staleSession.close()
    terminalRender.mockClear()
    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))

    await waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(terminalRender).toHaveBeenCalled())
    expect(terminalRender.mock.calls.every(([props]) => (props as { session: MockProtoSession }).session === freshSession)).toBe(true)
  })

  it('pauses terminal input during native recovery without rendering workspace banners', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    const api = {
      getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
      listTerminals: vi.fn(async () => [terminal]),
    }
    const connector = { connect: vi.fn(async () => session), reconnect: vi.fn(async () => undefined) }
    const view = render(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline connectionState="checking" />)
    expect(screen.queryByText(/Restoring the app connection/)).toBeNull()
    expect((terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }).onInput('blocked')).toBe(false)
    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline />)
    await waitFor(() => expect(connector.reconnect).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(connector.connect).toHaveBeenCalledTimes(2))
    expect(screen.queryByText('Connection restored')).toBeNull()

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline={false} />)
    expect(screen.queryByText('Your phone is offline')).toBeNull()
    expect(screen.queryByTestId('anytty-connection-recovery-overlay')).toBeNull()

    const latestProps = terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }
    expect(latestProps.onInput('whoami\n')).toBe(false)
    expect(terminalSendInput).not.toHaveBeenCalled()

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline connectionState="recovering" />)
    expect(screen.queryByText(/Restoring the app connection/)).toBeNull()
    expect((terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }).onInput('blocked')).toBe(false)
    expect(terminalSendInput).not.toHaveBeenCalled()
    expect(connector.reconnect).toHaveBeenCalledTimes(1)
    expect(connector.connect).toHaveBeenCalledTimes(2)

    view.rerender(<MachineWorkspace
      api={api}
      connector={connector}
      initialMachine={machine}
      phoneOnline
      connectionState="failed"
    />)
    expect(screen.queryByText('The app connection could not be restored.')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Retry' })).toBeNull()
    expect(connector.connect).toHaveBeenCalledTimes(2)

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline />)
    await waitFor(() => expect(connector.reconnect).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(connector.connect).toHaveBeenCalledTimes(3))
    expect(screen.queryByText('Connection restored')).toBeNull()

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline={false} />)
    expect(screen.queryByText('Your phone is offline')).toBeNull()
    expect(screen.queryByText('Connection restored')).toBeNull()
  })

  it('keeps the cached terminal list visible but inert while the app layer is unavailable', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const api = {
      getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
      listTerminals: vi.fn(async () => [terminal]),
    }
    const connector = { connect: vi.fn(async () => new MockProtoSession('studio')), reconnect: vi.fn(async () => undefined) }
    const view = render(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline />)

    await screen.findByRole('button', { name: 'Open Shell' })
    expect(screen.getByTestId('anytty-terminal-list-scroll')).toBeTruthy()

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline={false} />)
    expect(screen.queryByText('Your phone is offline')).toBeNull()
    expect(screen.getByTestId('anytty-terminal-list-scroll')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Open Shell' }) as HTMLButtonElement).disabled).toBe(true)

    view.rerender(<MachineWorkspace
      api={api}
      connector={connector}
      initialMachine={machine}
      phoneOnline
      connectionState="failed"
    />)
    expect(screen.queryByText('The app connection could not be restored.')).toBeNull()
    expect(screen.getByTestId('anytty-terminal-list-scroll')).toBeTruthy()
    expect((screen.getByRole('button', { name: 'Open Shell' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('lets the native session manager own phone network recovery', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio')
    const connect = vi.fn(async () => session)
    const reconnect = vi.fn(async () => undefined)
    let publishConnectionState: ((snapshot: RtcConnectionStateSnapshot) => void) | undefined
    const connectionStateEvents = {
      subscribe(_machineId: string, handler: (snapshot: RtcConnectionStateSnapshot) => void) {
        publishConnectionState = handler
        return { close() {} }
      },
    }
    const props = {
      api: {
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      },
      connector: { connect, reconnect },
      connectionStateEvents,
      initialMachine: machine,
    }
    const view = render(<MachineWorkspace {...props} phoneOnline />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    expect(connect).toHaveBeenCalledTimes(1)

    view.rerender(<MachineWorkspace {...props} phoneOnline={false} />)
    expect(screen.queryByText('Your phone is offline')).toBeNull()
    view.rerender(<MachineWorkspace {...props} phoneOnline />)

    expect(reconnect).not.toHaveBeenCalled()
    act(() => publishConnectionState?.({
      machineId: machine.machineId,
      phase: 'connected',
      statusText: 'Connected',
      relayInUse: false,
    }))
    await waitFor(() => expect(connect).toHaveBeenCalledTimes(1))
    expect(reconnect).not.toHaveBeenCalled()

    view.rerender(<MachineWorkspace {...props} phoneOnline connectionState="recovering" />)
    expect(screen.queryByText(/Restoring the app connection/)).toBeNull()
    view.rerender(<MachineWorkspace {...props} phoneOnline />)
    await Promise.resolve()
    expect(connect).toHaveBeenCalledTimes(1)
    expect(reconnect).not.toHaveBeenCalled()
  })

  it('shows re-pairing after the active daemon enrollment is deleted', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    const onNeedsReauthorization = vi.fn()
    const onBack = vi.fn()

    render(
      <ConnectionRecoveryOverlayProvider appIntent={null}>
        <MachineWorkspace
          api={{
            getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
            listTerminals: vi.fn(async () => [terminal]),
          }}
          connector={{ connect: vi.fn(async () => session) }}
          initialMachine={machine}
          onNeedsReauthorization={onNeedsReauthorization}
          onBack={onBack}
        />
      </ConnectionRecoveryOverlayProvider>,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    act(() => {
      session.emitClosed(Object.assign(new Error('cloud enrollment deleted'), {
        code: 'daemon_deleted',
        retryable: false,
      }))
    })

    expect(await screen.findByText('Cloud enrollment was deleted')).toBeTruthy()
    const overlay = screen.getByTestId('anytty-connection-recovery-overlay')
    const root = overlay.closest<HTMLElement>('[data-anytty-connection-overlay-root]')
    const terminalHeader = screen.getByTestId('anytty-terminal-header')
    const backToList = screen.getByRole('button', { name: 'Back to terminal list' })
    expect(screen.getAllByTestId('anytty-connection-recovery-overlay')).toHaveLength(1)
    expect(screen.getByTestId('anytty-terminal-page').contains(root)).toBe(true)
    expect(terminalHeader.contains(root)).toBe(false)
    expect(root?.contains(backToList)).toBe(false)
    expect(screen.queryByRole('button', { name: 'Retry other routes' })).toBeNull()
    const scan = screen.getByRole('button', { name: 'Open device pairing' })
    expect(scan.className).toContain('min-h-11')
    await userEvent.click(scan)
    expect(onNeedsReauthorization).toHaveBeenCalledWith('studio')

    await userEvent.click(backToList)
    const backToMachines = await screen.findByRole('button', { name: 'Back to devices' })
    expect(screen.getAllByTestId('anytty-connection-recovery-overlay')).toHaveLength(1)
    expect(screen.getByTestId('anytty-terminal-list-page').contains(screen.getByTestId('anytty-connection-recovery-overlay'))).toBe(true)
    expect(screen.getByTestId('anytty-terminal-list-page').querySelector('header')?.contains(screen.getByTestId('anytty-connection-recovery-overlay'))).toBe(false)
    await userEvent.click(backToMachines)
    expect(onBack).toHaveBeenCalledOnce()
  })

  it('returns false without a terminal handle and for a rejected single-target send', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))

    terminalHarness.exposeHandle = false
    const view = render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    let onInput = (terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }).onInput
    expect(onInput('no-handle')).toBe(false)
    expect(terminalSendInput).not.toHaveBeenCalled()

    view.unmount()
    terminalHarness.exposeHandle = true
    terminalSendInput.mockReturnValue(false)
    const rejectedSession = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => rejectedSession) }}
      initialMachine={machine}
    />)
    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    onInput = (terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }).onInput
    expect(onInput('rejected')).toBe(false)
    expect(terminalSendInput).toHaveBeenLastCalledWith('term-shell', 'rejected')
  })

  it('accepts synchronized split input when either target succeeds and never resends to a successful target', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = [
      {
        terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
        command: '/bin/zsh', cols: 80, rows: 24,
      },
      {
        terminalId: 'term-logs', machineId: 'studio', title: 'Logs', state: 'running' as const,
        command: 'tail -f app.log', cols: 80, rows: 24,
      },
    ]
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await userEvent.click(screen.getByTestId('anytty-terminal-split-button'))
    await waitFor(() => expect(screen.getAllByTestId('mock-terminal')).toHaveLength(2))
    await userEvent.click(screen.getByTestId('anytty-terminal-tools-button'))
    await userEvent.click(await screen.findByRole('button', { name: 'Sync input' }))

    const onInput = (terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }).onInput
    const assertOneAttemptPerTarget = (data: string) => {
      expect(terminalSendInput.mock.calls.filter(([, sent]) => sent === data)).toEqual([
        ['term-shell', data],
        ['term-logs', data],
      ])
    }

    terminalSendInput.mockClear()
    terminalSendInput.mockReturnValue(false)
    expect(onInput('all-fail')).toBe(false)
    assertOneAttemptPerTarget('all-fail')

    terminalSendInput.mockClear()
    terminalSendInput.mockImplementation((terminalId: string) => terminalId === 'term-shell')
    expect(onInput('partial-success')).toBe(true)
    assertOneAttemptPerTarget('partial-success')

    terminalSendInput.mockClear()
    terminalSendInput.mockReturnValue(true)
    expect(onInput('all-success')).toBe(true)
    assertOneAttemptPerTarget('all-success')
  })

  it('uses keyboard focus lock only to prevent soft-keyboard focus while shortcuts remain sendable', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    const keyboard = screen.getByRole('button', { name: 'Show system keyboard' })
    vi.useFakeTimers()
    fireEvent.pointerDown(keyboard)
    act(() => vi.advanceTimersByTime(400))
    fireEvent.pointerUp(keyboard)
    fireEvent.click(keyboard)
    vi.useRealTimers()
    await waitFor(() => {
      const props = terminalRender.mock.calls.at(-1)?.[0] as { preventFocus: boolean }
      expect(props.preventFocus).toBe(true)
    })
    expect(terminalBlur).toHaveBeenCalledWith('term-shell')

    const onInput = (terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }).onInput
    expect(onInput('shortcut')).toBe(true)
    expect(terminalSendInput).toHaveBeenLastCalledWith('term-shell', 'shortcut')
  })

  it('closes a nested workspace sheet before navigating the workspace itself', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', (command) => {
      if (command.command.case === 'terminalDefaults') {
        return protoResult('terminalDefaults', create(TerminalDefaultsResultSchema, {
          defaults: create(TerminalDefaultsSchema, { defaultCommand: ['/bin/zsh'], defaultCwd: '/tmp' }),
        }))
      }
      return protoResult('acknowledge', create(AcknowledgeResultSchema))
    })
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Create terminal' }))
    expect(await screen.findByTestId('anytty-terminal-editor-sheet')).toBeTruthy()
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-terminal-editor-sheet')).toBeNull()
    expect(screen.getByTestId('anytty-terminal-list-page')).toBeTruthy()

    await userEvent.click(screen.getByRole('button', { name: 'Open Shell' }))
    expect(await screen.findByTestId('anytty-terminal-page')).toBeTruthy()
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.getByTestId('anytty-terminal-list-page')).toBeTruthy()
    expect(dispatchNativeBack()).toBe(false)
  })

  it('returns from Files to the terminal that opened it before returning to the terminal list', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cwd: '/', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    expect(await screen.findByTestId('anytty-terminal-header')).toBeTruthy()
    expect(screen.getByTestId('mock-terminal').dataset.terminalId).toBe('term-shell')

    await userEvent.click(screen.getByTestId('anytty-terminal-files-button'))
    expect(screen.getByTestId('anytty-machine-files-overlay').classList.contains('visible')).toBe(true)

    await userEvent.click(screen.getByRole('button', { name: 'Close files' }))
    expect(screen.getByTestId('anytty-machine-files-overlay').classList.contains('invisible')).toBe(true)
    expect(screen.queryByTestId('anytty-terminal-list-page')).toBeNull()
    expect(screen.getByTestId('anytty-terminal-header')).toBeTruthy()
    expect(screen.getByTestId('mock-terminal').dataset.terminalId).toBe('term-shell')

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.getByTestId('anytty-terminal-list-page')).toBeTruthy()
  })

  it('closes Files before a split terminal hidden underneath it', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminals = [
      {
        terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
        command: '/bin/zsh', cwd: '/', cols: 80, rows: 24,
      },
      {
        terminalId: 'term-logs', machineId: 'studio', title: 'Logs', state: 'running' as const,
        command: 'tail -f app.log', cwd: '/', cols: 80, rows: 24,
      },
    ]
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => terminals),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await userEvent.click(screen.getByTestId('anytty-terminal-split-button'))
    expect(await screen.findByTestId('anytty-split-terminal-panel')).toBeTruthy()

    await userEvent.click(screen.getByTestId('anytty-terminal-files-button'))
    expect(screen.getByTestId('anytty-machine-files-overlay').classList.contains('visible')).toBe(true)
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.getByTestId('anytty-machine-files-overlay').classList.contains('invisible')).toBe(true)
    expect(screen.getByTestId('anytty-split-terminal-panel')).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-split-terminal-panel')).toBeNull()
  })

  it('closes Files before a terminal selection toolbar hidden underneath it', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cwd: '/', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await openTerminalTools()
    await userEvent.click(within(screen.getByTestId('anytty-terminal-body')).getByRole('button', { name: 'Select' }))
    expect(screen.getByRole('button', { name: 'Cancel selection' })).toBeTruthy()

    await userEvent.click(screen.getByTestId('anytty-terminal-files-button'))
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.getByTestId('anytty-machine-files-overlay').classList.contains('invisible')).toBe(true)
    expect(screen.getByRole('button', { name: 'Cancel selection' })).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByRole('button', { name: 'Cancel selection' })).toBeNull()
  })

  it('returns terminal path picker and bookmarks to their editor before closing it', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const session = new MockProtoSession('studio', (command) => {
      if (command.command.case === 'terminalDefaults') {
        return protoResult('terminalDefaults', create(TerminalDefaultsResultSchema, {
          defaults: create(TerminalDefaultsSchema, { defaultCommand: ['/bin/zsh'], defaultCwd: '/tmp' }),
        }))
      }
      return protoResult('acknowledge', create(AcknowledgeResultSchema))
    })
    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => []),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    const openCreateEditor = async () => {
      await userEvent.click(await screen.findByRole('button', { name: 'Create terminal' }))
      return screen.findByTestId('anytty-terminal-editor-sheet')
    }

    let editor = await openCreateEditor()
    await userEvent.click(within(editor).getByRole('button', { name: 'Browse' }))
    expect(await screen.findByTestId('anytty-terminal-path-picker-sheet')).toBeTruthy()
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-terminal-path-picker-sheet')).toBeNull()
    expect(screen.getByTestId('anytty-terminal-editor-sheet')).toBeTruthy()
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-terminal-editor-sheet')).toBeNull()

    editor = await openCreateEditor()
    await userEvent.click(within(editor).getByRole('button', { name: 'Path bookmarks' }))
    expect(await screen.findByTestId('anytty-terminal-path-bookmarks-sheet')).toBeTruthy()
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-terminal-path-bookmarks-sheet')).toBeNull()
    expect(screen.getByTestId('anytty-terminal-editor-sheet')).toBeTruthy()
    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-terminal-editor-sheet')).toBeNull()
  })

  it('refreshes daemon inventory after a list-page manual reconnect', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const staleSession = new MockProtoSession('studio')
    const freshSession = new MockProtoSession('studio')
    const connect = vi.fn()
      .mockResolvedValueOnce(staleSession)
      .mockResolvedValueOnce(freshSession)
    const reconnect = vi.fn(async () => undefined)
    const applyConnectionPolicy = vi.fn(async () => undefined)
    const listTerminals = vi.fn(async () => [terminal])
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals,
      }}
      connector={{
        connect,
        reconnect,
        getConnectionPolicy: vi.fn(async () => ({
          policy: { route: 'auto', cloud: 'auto', relayTransport: 'auto' },
          available: { direct: true, ssh: true, cloud: true },
          unavailableReasons: {},
        })),
        applyConnectionPolicy,
      }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    await userEvent.click(screen.getByRole('button', { name: 'Back to terminal list' }))
    await staleSession.close()
    await userEvent.click(screen.getByRole('button', { name: 'Connection info' }))
    await userEvent.click(await screen.findByRole('radio', { name: 'Direct' }))
    await userEvent.click(screen.getByRole('button', { name: 'Apply & reconnect' }))

    await waitFor(() => expect(applyConnectionPolicy).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(reconnect).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(listTerminals.mock.calls.length).toBeGreaterThan(1))
  })

  it('rebuilds the workspace session before refreshing files after a native generation resume', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const staleSession = new MockProtoSession('studio')
    const freshSession = new MockProtoSession('studio')
    const connect = vi.fn()
      .mockResolvedValueOnce(staleSession)
      .mockResolvedValueOnce(freshSession)
    const listTerminals = vi.fn(async () => [])
    const api = {
      getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
      listTerminals,
    }

    const connectionStateEvents = { subscribe: vi.fn(() => ({ close() {} })) }
    const view = render(<MachineWorkspace
      api={api}
      connector={{ connect }}
      connectionStateEvents={connectionStateEvents}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open files' }))
    await waitFor(() => expect(connect).toHaveBeenCalledTimes(1))
    view.rerender(<MachineWorkspace
      api={api}
      connector={{ connect }}
      connectionState="checking"
      connectionStateEvents={connectionStateEvents}
      initialMachine={machine}
    />)
    await staleSession.close()
    document.dispatchEvent(new Event('anytty:resume'))
    await Promise.resolve()
    expect(connect).toHaveBeenCalledTimes(1)

    view.rerender(<MachineWorkspace
      api={api}
      connector={{ connect }}
      connectionState="ready"
      connectionStateEvents={connectionStateEvents}
      initialMachine={machine}
    />)
    document.dispatchEvent(new CustomEvent('anytty:resume', { detail: { revision: 1 } }))

    await waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(listTerminals.mock.calls.length).toBeGreaterThan(1))
    act(() => {
      document.dispatchEvent(new CustomEvent('anytty:resume', { detail: { revision: 1 } }))
    })
    expect(connect).toHaveBeenCalledTimes(2)
  })

  it('keeps a healthy terminal channel attached across a foreground resume', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio')

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    terminalFit.mockClear()
    terminalReattach.mockClear()

    act(() => { document.dispatchEvent(new Event('anytty:resume')) })

    expect(terminalFit).toHaveBeenCalledWith('term-shell')
    expect(terminalReattach).not.toHaveBeenCalled()

    act(() => { document.dispatchEvent(new Event('anytty:binding-closed')) })

    expect(terminalReattach).toHaveBeenCalledWith('term-shell', session, { forceTerminalChannel: true })
  })
})
