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
import { MachineWorkspace } from './MachineWorkspace'

const terminalRender = vi.hoisted(() => vi.fn())
const terminalSendInput = vi.hoisted(() => vi.fn())
const terminalPasteText = vi.hoisted(() => vi.fn())
const terminalFocus = vi.hoisted(() => vi.fn())
const terminalBlur = vi.hoisted(() => vi.fn())
const terminalFit = vi.hoisted(() => vi.fn())
const terminalReattach = vi.hoisted(() => vi.fn())
const terminalHarness = vi.hoisted(() => ({ exposeHandle: true, selection: '' }))
const originalInnerWidth = window.innerWidth

vi.mock('../terminal/Terminal', () => ({
  Terminal: forwardRef(function MockTerminal(props: unknown, ref) {
    terminalRender(props)
    const terminalId = (props as { terminalId: string }).terminalId
    useImperativeHandle(ref, () => terminalHarness.exposeHandle ? ({
      sendInput: (data: string) => terminalSendInput(terminalId, data),
      sendResize: () => {},
      requestResizeOwner: async () => ({ canResize: true, reason: 'owner' }),
      releaseResizeOwner: async () => ({ canResize: false, reason: 'follower' }),
      focus: () => terminalFocus(terminalId),
      blur: () => terminalBlur(terminalId),
      fit: () => terminalFit(terminalId),
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
    return <div data-terminal-id={terminalId} data-testid="mock-terminal" />
  }),
}))

async function openTerminalToolsFromMenu() {
  await userEvent.click(screen.getByRole('button', { name: 'Open terminal menu' }))
  const menu = await screen.findByTestId('anytty-terminal-menu-sheet')
  await userEvent.click(within(menu).getByRole('button', { name: 'Terminal tools' }))
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
    terminalHarness.exposeHandle = true
    terminalHarness.selection = ''
    await anyttyI18n.changeLanguage('en')
  })
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: originalInnerWidth })
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

  it('keeps files, split, and the three-dot menu visible in the compact terminal header', async () => {
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
    expect(terminalSwitcher.classList.contains('flex-col')).toBe(false)
    expect(terminalSummary.textContent).toBe('Studio · Shell · /home/ada')
    expect(terminalSummary.classList.contains('whitespace-nowrap')).toBe(true)
    expect(terminalSummary.classList.contains('truncate')).toBe(true)
    expect(within(header).getByRole('button', { name: 'Open files' })).toBeTruthy()
    expect(within(header).getByRole('button', { name: 'Split terminal' })).toBeTruthy()
    expect(within(header).queryByRole('button', { name: 'Control resize' })).toBeNull()
    expect(within(header).queryByRole('button', { name: 'Terminal tools' })).toBeNull()
    const menuButton = within(header).getByTestId('anytty-terminal-menu-button')
    expect(menuButton.querySelector('.lucide-ellipsis')).toBeTruthy()
    expect(header.className).toContain('min-h-10')
    expect(Array.from(header.querySelectorAll('button')).every((button) => (
      button.className.includes('h-10') || button.className.includes('min-h-10')
    ))).toBe(true)
  })

  it('closes only the 320px topmost menu before the selection toolbar while split', async () => {
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

    const menuButton = within(header).getByRole('button', { name: 'Open terminal menu' })
    await userEvent.click(within(header).getByTestId('anytty-terminal-split-button'))
    await userEvent.click(within(await screen.findByTestId('anytty-split-terminal-sheet')).getByRole('button', { name: 'Open Logs' }))
    expect(await screen.findByTestId('anytty-split-terminal-panel')).toBeTruthy()

    await userEvent.click(menuButton)
    const menuSheet = await screen.findByTestId('anytty-terminal-menu-sheet')
    expect(menuSheet.className).toContain('absolute')
    expect(menuSheet.className).toContain('origin-top-right')
    expect(within(menuSheet).getByRole('button', { name: 'Control resize' })).toBeTruthy()
    const toolbarEntry = within(menuSheet).getByRole('button', { name: 'Terminal tools' })
    expect(toolbarEntry.classList.contains('min-h-11')).toBe(true)
    await userEvent.click(toolbarEntry)

    expect(screen.queryByTestId('anytty-terminal-menu-sheet')).toBeNull()
    const toolbar = screen.getByTestId('anytty-terminal-action-toolbar')
    expect(screen.getAllByTestId('anytty-terminal-action-toolbar')).toHaveLength(1)
    expect(within(toolbar).getByRole('button', { name: 'Decrease terminal font size' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Increase terminal font size' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Renderer: Auto' }).classList.contains('h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Paste' }).classList.contains('min-h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Clipboard' }).classList.contains('min-h-11')).toBe(true)
    expect(within(toolbar).getByRole('button', { name: 'Snippets' }).classList.contains('min-h-11')).toBe(true)
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

    await userEvent.click(menuButton)
    expect(await screen.findByTestId('anytty-terminal-menu-sheet')).toBeTruthy()
    expect(screen.getByTestId('anytty-terminal-action-toolbar')).toBeTruthy()
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByTestId('anytty-terminal-menu-sheet')).toBeNull()
    expect(screen.getByTestId('anytty-terminal-action-toolbar')).toBeTruthy()
    expect(screen.getByTestId('anytty-split-terminal-panel')).toBeTruthy()
    expect(document.activeElement).toBe(menuButton)

    await userEvent.tab()
    expect(selectionToolbar.contains(document.activeElement)).toBe(true)
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByTestId('anytty-terminal-action-toolbar')).toBeNull()
    expect(screen.getByTestId('anytty-split-terminal-panel')).toBeTruthy()
    expect(document.activeElement).toBe(menuButton)
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
    const toolsButton = within(header).getByTestId('anytty-terminal-menu-button')
    expect(window.innerWidth).toBe(390)

    await openTerminalToolsFromMenu()
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
    const toolsButton = screen.getByTestId('anytty-terminal-menu-button')
    await openTerminalToolsFromMenu()
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
    await openTerminalToolsFromMenu()
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

  it('pauses terminal input while the phone is offline', async () => {
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
    const retryConnectionRecovery = vi.fn(async () => undefined)
    const view = render(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline={false} connectionReady={false} />)
    await screen.findByText('Your phone is offline')

    const latestProps = terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }
    expect(latestProps.onInput('whoami\n')).toBe(false)
    expect(terminalSendInput).not.toHaveBeenCalled()

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline connectionReady={false} />)
    await screen.findByText('Connection interrupted. Reconnecting')
    expect((terminalRender.mock.calls.at(-1)?.[0] as { onInput: (data: string) => boolean }).onInput('blocked')).toBe(false)
    expect(terminalSendInput).not.toHaveBeenCalled()
    expect(connector.reconnect).not.toHaveBeenCalled()
    expect(connector.connect).toHaveBeenCalledTimes(1)

    view.rerender(<MachineWorkspace
      api={api}
      connector={connector}
      initialMachine={machine}
      phoneOnline
      connectionReady={false}
      connectionRecoveryFailed
      onRetryConnectionRecovery={retryConnectionRecovery}
    />)
    expect(await screen.findByText('Connection service unavailable')).toBeTruthy()
    const retry = screen.getByRole('button', { name: 'Retry' })
    expect(retry.className).toContain('min-h-11')
    await userEvent.click(retry)
    expect(retryConnectionRecovery).toHaveBeenCalledOnce()
    expect(connector.connect).toHaveBeenCalledTimes(1)

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline connectionReady />)
    await waitFor(() => expect(connector.reconnect).toHaveBeenCalledTimes(1))
    await waitFor(() => expect(connector.connect).toHaveBeenCalledTimes(2))
    expect(await screen.findByText('Connection restored')).toBeTruthy()

    view.rerender(<MachineWorkspace api={api} connector={connector} initialMachine={machine} phoneOnline={false} connectionReady={false} />)
    await screen.findByText('Your phone is offline')
    expect(screen.queryByText('Connection restored')).toBeNull()
  })

  it('shows re-pairing after the active daemon enrollment is deleted', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const terminal = {
      terminalId: 'term-shell', machineId: 'studio', title: 'Shell', state: 'running' as const,
      command: '/bin/zsh', cols: 80, rows: 24,
    }
    const session = new MockProtoSession('studio', () => protoResult('acknowledge', create(AcknowledgeResultSchema)))
    const onNeedsReauthorization = vi.fn()

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals: vi.fn(async () => [terminal]),
      }}
      connector={{ connect: vi.fn(async () => session) }}
      initialMachine={machine}
      onNeedsReauthorization={onNeedsReauthorization}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open Shell' }))
    await screen.findByTestId('mock-terminal')
    act(() => {
      session.emitClosed(Object.assign(new Error('cloud enrollment deleted'), {
        code: 'daemon_deleted',
        retryable: false,
      }))
    })

    expect(await screen.findByText('Cloud enrollment was deleted')).toBeTruthy()
    const retryOtherRoutes = screen.getByRole('button', { name: 'Retry other routes' })
    expect(retryOtherRoutes.className).toContain('min-h-11')
    const scan = screen.getByRole('button', { name: 'Scan QR' })
    expect(scan.className).toContain('min-h-11')
    expect(scan.parentElement?.className).toContain('grid-cols-1')
    expect(scan.parentElement?.className).toContain('min-[360px]:grid-cols-2')
    await userEvent.click(scan)
    expect(onNeedsReauthorization).toHaveBeenCalledWith('studio')
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
    const splitSheet = await screen.findByTestId('anytty-split-terminal-sheet')
    await userEvent.click(within(splitSheet).getByRole('button', { name: 'Open Logs' }))
    await waitFor(() => expect(screen.getAllByTestId('mock-terminal')).toHaveLength(2))
    await userEvent.click(screen.getByRole('button', { name: 'Open terminal menu' }))
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
    await userEvent.click(within(await screen.findByTestId('anytty-split-terminal-sheet')).getByRole('button', { name: 'Open Logs' }))
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
    await openTerminalToolsFromMenu()
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

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({ machine, localWeb: { httpUrl: '', rtcOfferUrl: '' } })),
        listTerminals,
      }}
      connector={{ connect }}
      initialMachine={machine}
    />)

    await userEvent.click(await screen.findByRole('button', { name: 'Open files' }))
    await waitFor(() => expect(connect).toHaveBeenCalledTimes(1))
    await staleSession.close()
    document.dispatchEvent(new Event('anytty:resume'))

    await waitFor(() => expect(connect).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(listTerminals.mock.calls.length).toBeGreaterThan(1))
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
