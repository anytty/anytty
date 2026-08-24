import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Terminal } from '../core/model'
import { anyttyI18n } from '../i18n'
import {
  filterWebTerminalPickerTerminals,
  terminalPickerMatchIndexes,
  WebTerminalPickerDialog,
} from './WebTerminalPickerDialog'

afterEach(cleanup)

describe('WebTerminalPickerDialog', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  it('fuzzy-filters terminals and opens the keyboard-selected result', async () => {
    const onOpenChange = vi.fn()
    const onSelectTerminal = vi.fn()
    renderPicker({ onOpenChange, onSelectTerminal })

    const input = screen.getByRole('combobox', { name: 'Find terminal' })
    await waitFor(() => expect(document.activeElement).toBe(input))
    await userEvent.type(input, 'lgs')

    expect(screen.getByRole('option', { name: /Logs/ })).toBeTruthy()
    expect(screen.queryByRole('option', { name: /Shell/ })).toBeNull()
    await userEvent.keyboard('{Enter}')
    expect(onOpenChange).toHaveBeenCalledWith(false)
    expect(onSelectTerminal).toHaveBeenCalledWith('logs')
  })

  it('keeps create first and supports arrow-key selection', async () => {
    const onCreateTerminal = vi.fn()
    const onSelectTerminal = vi.fn()
    renderPicker({ onCreateTerminal, onSelectTerminal })

    const options = screen.getAllByRole('option')
    expect(options[0]?.textContent).toContain('New terminal')
    expect(options[1]?.textContent).toContain('Logs')
    expect(options[2]?.textContent).toContain('Shell')

    await userEvent.keyboard('{ArrowDown}{Enter}')
    expect(onSelectTerminal).toHaveBeenCalledWith('logs')
    expect(onCreateTerminal).not.toHaveBeenCalled()
  })

  it('matches the TUI subsequence search across terminal metadata', () => {
    expect(terminalPickerMatchIndexes('term-pool', 'trpl')).toEqual([0, 2, 5, 8])
    expect(terminalPickerMatchIndexes('term-pool', 'logs')).toBeNull()
    expect(filterWebTerminalPickerTerminals(pickerTerminals, '80x24').map((terminal) => terminal.terminalId)).toEqual(['shell'])
    expect(filterWebTerminalPickerTerminals(pickerTerminals, '/wrk/lgs').map((terminal) => terminal.terminalId)).toEqual(['logs'])
  })
})

const pickerTerminals: Terminal[] = [
  { terminalId: 'shell', machineId: 'local', title: 'Shell', state: 'running', command: '/bin/zsh', cwd: '/work/shell', cols: 80, rows: 24 },
  { terminalId: 'logs', machineId: 'local', title: 'Logs', state: 'running', command: 'tail -f app.log', cwd: '/work/logs', cols: 120, rows: 40 },
]

function renderPicker(overrides: Partial<Parameters<typeof WebTerminalPickerDialog>[0]> = {}) {
  return render(<WebTerminalPickerDialog
    activeTerminalId="shell"
    canCreateTerminal
    disabled={false}
    open
    terminals={pickerTerminals}
    onCreateTerminal={vi.fn()}
    onOpenChange={vi.fn()}
    onSelectTerminal={vi.fn()}
    {...overrides}
  />)
}
