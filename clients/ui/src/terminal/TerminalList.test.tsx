import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TerminalList, type TerminalListProps } from './TerminalList'
import type { Terminal } from '../core/model'

describe('TerminalList', () => {
  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('renders terminals for one machine and opens a terminal by terminalId', async () => {
    const onOpenTerminal = vi.fn()
    const onManageTerminal = vi.fn()

    render(
      <TerminalList
        machineId="machine-local"
        terminals={[
          terminal({ terminalId: 'terminal-1', title: 'zsh', command: '/bin/zsh', cols: 120, rows: 36 }),
          terminal({ terminalId: 'terminal-2', title: 'logs', command: 'tail -f app.log', cols: 100, rows: 24 }),
        ]}
        onOpenTerminal={onOpenTerminal}
        onManageTerminal={onManageTerminal}
      />,
    )

    expect(screen.getByText('zsh')).toBeTruthy()
    expect(screen.getByText('/bin/zsh')).toBeTruthy()
    expect(screen.getByText('120 × 36')).toBeTruthy()
    expect(screen.getByText('logs')).toBeTruthy()

    await userEvent.click(screen.getByRole('button', { name: /open zsh/i }))

    expect(onOpenTerminal).toHaveBeenCalledWith({
      machineId: 'machine-local',
      terminalId: 'terminal-1',
    })

    await userEvent.click(screen.getByRole('button', { name: /manage zsh/i }))
    expect(onManageTerminal).toHaveBeenCalledWith({
      machineId: 'machine-local',
      terminalId: 'terminal-1',
    })
  })

  it('renders terminal metadata needed for choosing a local environment', () => {
    render(
      <TerminalList
        machineId="machine-local"
        terminals={[
          terminal({
            terminalId: 'terminal-1',
            title: 'dev shell',
            command: '/bin/zsh -l',
            cols: 132,
            rows: 43,
            cwd: '/Users/lozzow/project',
            state: 'running',
            sizeLocked: true,
            sizeLockMode: 'lock',
            environment: 'prod',
            lastActiveAt: '2026-05-02T07:01:02Z',
          }),
          terminal({
            terminalId: 'terminal-2',
            title: 'stopped worker',
            state: 'exited',
            cols: 80,
            rows: 24,
            sizeLocked: false,
            sizeLockMode: 'off',
          }),
        ]}
        onOpenTerminal={vi.fn()}
        onManageTerminal={vi.fn()}
      />,
    )

    expect(screen.getByText('dev shell')).toBeTruthy()
    expect(screen.getByText('/Users/lozzow/project')).toBeTruthy()
    expect(screen.getByText('prod')).toBeTruthy()
    expect(screen.getByText('132 × 43')).toBeTruthy()
    expect(screen.getByText('Running')).toBeTruthy()
    expect(screen.getByText('stopped worker')).toBeTruthy()
    expect(screen.getByText('Exited')).toBeTruthy()
    expect(screen.getByTestId('anytty-terminal-list').textContent).not.toMatch(/workspace|tab|window|pane|session/i)
  })

  it('shows an empty state without mentioning sessions, windows, panes, tabs, or workspaces', () => {
    render(
      <TerminalList
        machineId="machine-local"
        terminals={[]}
        onOpenTerminal={vi.fn()}
        onManageTerminal={vi.fn()}
      />,
    )

    expect(screen.getByText('No active terminals')).toBeTruthy()
    expect(screen.getByTestId('anytty-terminal-list').textContent).not.toMatch(/session|window|pane|workspace|tab/i)
  })

  it('renders duplicate terminal ids without duplicate React keys', () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})

    render(
      <TerminalList
        machineId="machine-local"
        terminals={[
          terminal({ terminalId: '3', title: 'topic a' }),
          terminal({ terminalId: '3', title: 'topic b' }),
        ]}
        onOpenTerminal={vi.fn()}
        onManageTerminal={vi.fn()}
      />,
    )

    expect(screen.getByText('topic a')).toBeTruthy()
    expect(screen.getByText('topic b')).toBeTruthy()
    expect(consoleError.mock.calls.some((call) => call.some((arg) => String(arg).includes('Encountered two children with the same key')))).toBe(false)
  })

  it('keeps the public props machine/terminal only', () => {
    const propKeys = Object.keys({
      machineId: 'machine-local',
      terminals: [],
      onOpenTerminal: vi.fn(),
    } satisfies TerminalListProps)

    expect(propKeys).not.toContain('sessions')
    expect(propKeys).not.toContain('windows')
    expect(propKeys).not.toContain('panes')
    expect(propKeys).not.toContain('paneId')
    expect(propKeys).not.toContain('sessionId')
    expect(propKeys).not.toContain('windowId')
  })
})

function terminal(overrides: Partial<Terminal>): Terminal {
  return {
    terminalId: overrides.terminalId ?? 'terminal-1',
    machineId: overrides.machineId ?? 'machine-local',
    title: overrides.title ?? 'zsh',
    state: overrides.state ?? 'running',
    command: overrides.command,
    cols: overrides.cols,
    rows: overrides.rows,
    cwd: overrides.cwd,
    lastActiveAt: overrides.lastActiveAt,
    sizeLocked: overrides.sizeLocked,
    sizeLockMode: overrides.sizeLockMode,
    environment: overrides.environment,
  }
}
