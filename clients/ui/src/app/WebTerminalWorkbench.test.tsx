import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Terminal } from '../core/model'
import { anyttyI18n } from '../i18n'
import { DEFAULT_TERMINAL_SETTINGS } from '../terminal/terminalSettings'
import { WebTerminalSettingsDialog } from './WebTerminalSettingsDialog'
import { ANYTTY_TERMINAL_DRAG_TYPE, WebTerminalDropOverlay } from './WebTerminalDropOverlay'
import { WebSplitDivider, WebTerminalWorkbench } from './WebTerminalWorkbench'

afterEach(cleanup)

describe('WebTerminalWorkbench', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  it('renders layout tabs with keyboard navigation and view-only close actions', async () => {
    const onActivateTab = vi.fn()
    const onCloseTab = vi.fn()
    renderWorkbench({ onActivateTab, onCloseTab })

    const shell = screen.getByRole('tab', { name: 'Shell' })
    expect(shell.getAttribute('aria-selected')).toBe('true')
    fireEvent.keyDown(shell, { key: 'ArrowRight' })
    expect(onActivateTab).toHaveBeenCalledWith('logs')

    await userEvent.click(screen.getByRole('button', { name: 'Close Logs' }))
    expect(onCloseTab).toHaveBeenCalledWith('logs')
  })

  it('toggles the terminal sidebar from the persistent workbench control', async () => {
    const onToggleSidebar = vi.fn()
    const view = renderWorkbench({ onToggleSidebar, sidebarOpen: true })

    await userEvent.click(screen.getByRole('button', { name: 'Hide terminal sidebar' }))
    expect(onToggleSidebar).toHaveBeenCalledOnce()

    view.rerender(<WebTerminalWorkbench {...workbenchProps({ onToggleSidebar, sidebarOpen: false })} />)
    expect(screen.getByRole('button', { name: 'Show terminal sidebar' })).toBeTruthy()
  })

  it('opens the terminal picker from the workbench search action', async () => {
    const onOpenTerminalPicker = vi.fn()
    renderWorkbench({ onOpenTerminalPicker })

    await userEvent.click(screen.getByRole('button', { name: 'Find terminal' }))
    expect(onOpenTerminalPicker).toHaveBeenCalledOnce()
  })

  it('reorders tabs through native drag and adds a split below directly', async () => {
    const onReorderTabs = vi.fn()
    const onOpenSplit = vi.fn()
    const onTerminalDragChange = vi.fn()
    renderWorkbench({ onOpenSplit, onReorderTabs, onTerminalDragChange, splitTabTerminalIds: ['logs'] })
    const transfer = dataTransfer('logs')
    const shellContainer = screen.getByRole('tab', { name: 'Shell' }).parentElement!
    const logsContainer = screen.getByRole('tab', { name: 'Logs' }).parentElement!
    vi.spyOn(shellContainer, 'getBoundingClientRect').mockReturnValue(rect(100))

    fireEvent.dragStart(logsContainer, { dataTransfer: transfer })
    fireEvent.dragOver(shellContainer, { clientX: 10, dataTransfer: transfer })
    fireEvent.drop(shellContainer, { clientX: 10, dataTransfer: transfer })

    expect(onTerminalDragChange).toHaveBeenCalledWith('logs')
    expect(onReorderTabs).toHaveBeenCalledWith('logs', 'shell', 'after')
    await userEvent.click(screen.getByRole('button', { name: 'Split below' }))
    expect(onOpenSplit).toHaveBeenCalledOnce()
  })

  it('shows one final-layout preview inside the hovered leaf pane and keeps the divider keyboard-resizable', () => {
    const onDrop = vi.fn()
    const onRatioChange = vi.fn()
    const view = render(
      <div>
        <div data-pane-key="primary" data-pane-terminal-id="shell" />
        <div data-pane-key="terminal:logs" data-pane-terminal-id="logs" />
        <WebTerminalDropOverlay canSplit draggedTerminalId="incoming" onDrop={onDrop} />
      </div>,
    )
    const overlay = screen.getByTestId('anytty-web-terminal-drop-overlay')
    const [shellPane, logsPane] = overlay.parentElement!.querySelectorAll<HTMLElement>('[data-pane-key]')
    vi.spyOn(overlay, 'getBoundingClientRect').mockReturnValue(rectAt(0, 0, 400, 200))
    vi.spyOn(shellPane!, 'getBoundingClientRect').mockReturnValue(rectAt(0, 0, 200, 200))
    vi.spyOn(logsPane!, 'getBoundingClientRect').mockReturnValue(rectAt(200, 0, 200, 200))
    expect(screen.queryByTestId('anytty-web-terminal-drop-preview')).toBeNull()
    expect(screen.queryAllByRole('button')).toHaveLength(0)

    const transfer = dataTransfer('incoming')
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 10, 100))
    expect(overlay.getAttribute('data-preview-target')).toBe('left')
    expect(overlay.getAttribute('data-preview-pane-key')).toBe('primary')
    expect(screen.getAllByTestId('anytty-web-terminal-drop-preview')).toHaveLength(1)
    expect(previewLayout(overlay)).toEqual({ axis: 'columns', panes: ['incoming', 'existing'] })
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 100, 10))
    expect(overlay.getAttribute('data-preview-target')).toBe('top')
    expect(previewLayout(overlay)).toEqual({ axis: 'rows', panes: ['incoming', 'existing'] })
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 190, 100))
    expect(overlay.getAttribute('data-preview-target')).toBe('right')
    expect(previewLayout(overlay)).toEqual({ axis: 'columns', panes: ['existing', 'incoming'] })
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 100, 190))
    expect(overlay.getAttribute('data-preview-target')).toBe('bottom')
    expect(previewLayout(overlay)).toEqual({ axis: 'rows', panes: ['existing', 'incoming'] })
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 390, 100))
    expect(overlay.getAttribute('data-preview-pane-key')).toBe('terminal:logs')
    expect(screen.getByTestId('anytty-web-terminal-drop-preview').parentElement?.style.left).toBe('200px')
    fireEvent(overlay, pointerDragEvent('drop', transfer, 390, 100))
    expect(onDrop).toHaveBeenCalledWith('incoming', 'terminal:logs', 'right')

    fireEvent(overlay, pointerDragEvent('dragover', dataTransfer('logs'), 300, 100))
    expect(overlay.getAttribute('data-preview-pane-key')).toBeNull()
    expect(screen.queryByTestId('anytty-web-terminal-drop-preview')).toBeNull()

    view.rerender(<WebSplitDivider direction="columns" ratio={50} onRatioChange={onRatioChange} onResizeEnd={vi.fn()} />)
    const divider = screen.getByRole('separator', { name: 'Resize terminal panes' })
    fireEvent.keyDown(divider, { key: 'ArrowRight' })
    expect(onRatioChange).toHaveBeenCalledWith(55)
  })
})

describe('WebTerminalSettingsDialog', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  it('uses a modal surface and emits focused terminal setting patches', async () => {
    const onChange = vi.fn()
    const onOpenChange = vi.fn()
    render(<WebTerminalSettingsDialog open settings={DEFAULT_TERMINAL_SETTINGS} onChange={onChange} onOpenChange={onOpenChange} />)

    const dialog = screen.getByRole('dialog', { name: 'Settings' })
    expect(dialog.getAttribute('data-state')).toBe('open')
    await userEvent.click(screen.getByRole('button', { name: 'Increase terminal font size' }))
    expect(onChange).toHaveBeenCalledWith({ fontSize: DEFAULT_TERMINAL_SETTINGS.fontSize + 1 })

    await userEvent.click(screen.getByRole('button', { name: 'Canvas' }))
    expect(onChange).toHaveBeenCalledWith({ renderer: 'canvas' })
    await userEvent.click(screen.getByRole('button', { name: 'Close Settings' }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})

function renderWorkbench(overrides: Partial<Parameters<typeof WebTerminalWorkbench>[0]> = {}) {
  return render(<WebTerminalWorkbench {...workbenchProps(overrides)} />)
}

function workbenchProps(overrides: Partial<Parameters<typeof WebTerminalWorkbench>[0]> = {}): Parameters<typeof WebTerminalWorkbench>[0] {
  return {
    terminals: [terminal('shell', 'Shell'), terminal('logs', 'Logs')],
    tabTerminalIds: ['shell', 'logs'],
    activeTabTerminalId: 'shell',
    splitTabTerminalIds: [],
    draggedTerminalId: null,
    sidebarOpen: true,
    canCreateTerminal: true,
    canSplitTerminal: true,
    disabled: false,
    onActivateTab: vi.fn(),
    onCloseTab: vi.fn(),
    onCreateTerminal: vi.fn(),
    onOpenFiles: vi.fn(),
    onOpenTerminalPicker: vi.fn(),
    onOpenSettings: vi.fn(),
    onOpenSplit: vi.fn(),
    onReorderTabs: vi.fn(),
    onToggleSidebar: vi.fn(),
    onTerminalDragChange: vi.fn(),
    ...overrides,
  }
}

function terminal(terminalId: string, title: string): Terminal {
  return { terminalId, machineId: 'local', title, state: 'running', command: '/bin/zsh', cwd: `/work/${terminalId}` }
}

function dataTransfer(terminalId: string): DataTransfer {
  const values = new Map<string, string>([[ANYTTY_TERMINAL_DRAG_TYPE, terminalId], ['text/plain', terminalId]])
  return {
    dropEffect: 'move',
    effectAllowed: 'all',
    files: [] as unknown as FileList,
    items: [] as unknown as DataTransferItemList,
    types: [...values.keys()],
    clearData: vi.fn(),
    getData: (type: string) => values.get(type) ?? '',
    setData: (type: string, value: string) => { values.set(type, value) },
    setDragImage: vi.fn(),
  }
}

function rect(width: number, height = 40): DOMRect {
  return rectAt(0, 0, width, height)
}

function rectAt(left: number, top: number, width: number, height: number): DOMRect {
  return { x: left, y: top, width, height, top, right: left + width, bottom: top + height, left, toJSON: () => ({}) }
}

function pointerDragEvent(type: 'dragover' | 'drop', transfer: DataTransfer, clientX: number, clientY: number): Event {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.defineProperties(event, {
    clientX: { value: clientX },
    clientY: { value: clientY },
    dataTransfer: { value: transfer },
  })
  return event
}

function previewLayout(overlay: HTMLElement): { axis: string | null; panes: Array<string | null> } {
  const preview = overlay.querySelector<HTMLElement>('[data-preview-layout]')
  return {
    axis: preview?.getAttribute('data-preview-layout') ?? null,
    panes: Array.from(preview?.querySelectorAll<HTMLElement>('[data-preview-pane]') ?? []).map((pane) => pane.getAttribute('data-preview-pane')),
  }
}
