import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Terminal } from '../core/model'
import { anyttyI18n } from '../i18n'
import { DEFAULT_TERMINAL_SETTINGS } from '../terminal/terminalSettings'
import { WebTerminalSettingsDialog } from './WebTerminalSettingsDialog'
import { ANYTTY_TERMINAL_DRAG_TYPE, WebSplitDivider, WebTerminalDropOverlay, WebTerminalWorkbench } from './WebTerminalWorkbench'

afterEach(cleanup)

describe('WebTerminalWorkbench', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  it('renders ordered tabs with keyboard navigation and view-only close actions', async () => {
    const onActivateTerminal = vi.fn()
    const onCloseTab = vi.fn()
    renderWorkbench({ onActivateTerminal, onCloseTab })

    const shell = screen.getByRole('tab', { name: 'Shell' })
    expect(shell.getAttribute('aria-selected')).toBe('true')
    fireEvent.keyDown(shell, { key: 'ArrowRight' })
    expect(onActivateTerminal).toHaveBeenCalledWith('logs')

    await userEvent.click(screen.getByRole('button', { name: 'Close Logs' }))
    expect(onCloseTab).toHaveBeenCalledWith('logs')
  })

  it('reorders tabs through native drag and adds a split below directly', async () => {
    const onReorderTabs = vi.fn()
    const onOpenSplit = vi.fn()
    const onTerminalDragChange = vi.fn()
    renderWorkbench({ onOpenSplit, onReorderTabs, onTerminalDragChange, splitTerminalIds: ['logs'] })
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

  it('shows one global split preview that follows the drag position and keeps the divider keyboard-resizable', () => {
    const onDrop = vi.fn()
    const onRatioChange = vi.fn()
    const view = render(<WebTerminalDropOverlay canSplit draggedTerminalId="logs" onDrop={onDrop} />)
    const overlay = screen.getByTestId('anytty-web-terminal-drop-overlay')
    vi.spyOn(overlay, 'getBoundingClientRect').mockReturnValue(rect(400, 200))
    expect(screen.getAllByTestId('anytty-web-terminal-drop-preview')).toHaveLength(1)
    expect(screen.queryAllByRole('button')).toHaveLength(0)

    const transfer = dataTransfer('logs')
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 10, 100))
    expect(overlay.getAttribute('data-preview-target')).toBe('left')
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 200, 10))
    expect(overlay.getAttribute('data-preview-target')).toBe('top')
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 390, 100))
    expect(overlay.getAttribute('data-preview-target')).toBe('right')
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 200, 190))
    expect(overlay.getAttribute('data-preview-target')).toBe('bottom')
    fireEvent(overlay, pointerDragEvent('dragover', transfer, 390, 100))
    fireEvent(overlay, pointerDragEvent('drop', transfer, 390, 100))
    expect(onDrop).toHaveBeenCalledWith('logs', 'right')

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
  const props: Parameters<typeof WebTerminalWorkbench>[0] = {
    terminals: [terminal('shell', 'Shell'), terminal('logs', 'Logs')],
    openTerminalIds: ['shell', 'logs'],
    activeTerminalId: 'shell',
    splitTerminalIds: [],
    draggedTerminalId: null,
    canCreateTerminal: true,
    canSplitTerminal: true,
    disabled: false,
    onActivateTerminal: vi.fn(),
    onCloseTab: vi.fn(),
    onCreateTerminal: vi.fn(),
    onOpenFiles: vi.fn(),
    onOpenSettings: vi.fn(),
    onOpenSplit: vi.fn(),
    onReorderTabs: vi.fn(),
    onTerminalDragChange: vi.fn(),
    ...overrides,
  }
  return render(<WebTerminalWorkbench {...props} />)
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
  return { x: 0, y: 0, width, height, top: 0, right: width, bottom: height, left: 0, toJSON: () => ({}) }
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
