import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TerminalList, terminalOutputActivityLabel, terminalOutputActivityTone, terminalProgramPresentation, type TerminalListProps } from './TerminalList'
import type { Terminal } from '../core/model'
import type { TFunction } from 'i18next'

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
    expect(screen.getByText('logs')).toBeTruthy()
    expect(screen.getByRole('list', { name: 'Terminals' }).className).toContain('gap-2')
    expect(screen.getByText('zsh').closest('li')?.className).toContain('rounded-lg')
    expect(screen.getByText('logs').closest('li')?.className).toContain('rounded-lg')

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
            foregroundProcess: 'codex',
            lastOutputAt: new Date(Date.now() - 12_000).toISOString(),
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
    expect(screen.getByRole('button', { name: /open dev shell/i }).className).toContain('h-auto')
    expect(screen.getByText('/Users/lozzow/project')).toBeTruthy()
    expect(screen.getByText('prod')).toBeTruthy()
    expect(screen.getByText('prod').className).toContain('rounded-full')
    expect(screen.queryByText('132 × 43')).toBeNull()
    expect(screen.getByText('Running')).toBeTruthy()
    expect(screen.getByText('Codex')).toBeTruthy()
    expect(screen.getByText('12s')).toBeTruthy()
    expect(screen.getByText('stopped worker')).toBeTruthy()
    expect(screen.getByText('Exited')).toBeTruthy()
    expect(screen.getByTestId('anytty-terminal-list').textContent).not.toMatch(/workspace|tab|window|pane|session/i)
  })

  it('maps known agent programs and formats output quiet time', () => {
    expect(terminalProgramPresentation('gemini').label).toBe('Gemini CLI')
    expect(terminalProgramPresentation('/usr/local/bin/claude').label).toBe('Claude Code')
    expect(terminalProgramPresentation('codex').brandAsset).toBeTruthy()
    expect(terminalProgramPresentation('github-copilot').label).toBe('GitHub Copilot')
    expect(terminalProgramPresentation('cursor-agent').brandAsset).toBeTruthy()
    expect(terminalProgramPresentation('qwen-code').label).toBe('Qwen Code')
    expect(terminalProgramPresentation('zsh').label).toBe('zsh')
    expect(terminalProgramPresentation('pwsh').label).toBe('PowerShell')
    expect(terminalProgramPresentation('opencode').label).toBe('OpenCode')
    expect(terminalProgramPresentation('nvim').icon).not.toBe(terminalProgramPresentation(undefined).icon)
    expect(terminalProgramPresentation('/usr/bin/htop').icon).not.toBe(terminalProgramPresentation(undefined).icon)
    expect(terminalProgramPresentation('tmux').label).toBe('tmux')
    expect(terminalProgramPresentation('kubectl').icon).not.toBe(terminalProgramPresentation(undefined).icon)
    expect(terminalProgramPresentation('redis-server').label).toBe('Redis')
    expect(terminalProgramPresentation('custom-agent').label).toBe('custom-agent')
    expect(terminalProgramPresentation('custom-agent').brandAsset).toBeUndefined()
    expect(terminalProgramPresentation('custom-agent').icon).toBe(terminalProgramPresentation(undefined).icon)
    const translate = ((key: string, options?: Record<string, unknown>) => `${key}:${String(options?.count ?? '')}`) as TFunction
    expect(terminalOutputActivityLabel('2026-08-13T00:00:00Z', Date.parse('2026-08-13T00:00:43Z'), translate)).toBe('terminal.outputActivity.seconds:43')
    expect(terminalOutputActivityLabel('2026-08-13T00:00:00Z', Date.parse('2026-08-13T00:02:10Z'), translate)).toBe('terminal.outputActivity.minutes:2')
    expect(terminalOutputActivityLabel('2026-08-13T00:00:00Z', Date.parse('2026-08-13T03:00:00Z'), translate)).toBe('terminal.outputActivity.hours:3')
    expect(terminalOutputActivityTone('2026-08-13T00:00:00Z', Date.parse('2026-08-13T00:00:02Z'))).toBe('fresh')
    expect(terminalOutputActivityTone('2026-08-13T00:00:00Z', Date.parse('2026-08-13T00:00:43Z'))).toBe('recent')
    expect(terminalOutputActivityTone('2026-08-13T00:00:00Z', Date.parse('2026-08-13T00:30:00Z'))).toBe('idle')
    expect(terminalOutputActivityTone('2026-08-13T00:00:00Z', Date.parse('2026-08-13T03:00:00Z'))).toBe('stale')
    expect(terminalOutputActivityTone(undefined, Date.now())).toBe('none')
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

  it('shows a compact loading status instead of placeholder terminal rows', () => {
    render(
      <TerminalList
        machineId="machine-local"
        terminals={[]}
        loading
        loadingLabel="Connecting to device..."
        onOpenTerminal={vi.fn()}
      />,
    )

    const status = screen.getByRole('status')
    expect(status.textContent).toBe('Connecting to device...')
    expect(status.getAttribute('aria-busy')).toBe('true')
    expect(screen.queryByRole('list')).toBeNull()
  })

  it('keeps cached terminals visible but disables remote actions while disconnected', () => {
    render(
      <TerminalList
        machineId="machine-local"
        terminals={[terminal({ terminalId: 'terminal-1', title: 'zsh' })]}
        interactive={false}
        onOpenTerminal={vi.fn()}
        onManageTerminal={vi.fn()}
      />,
    )

    expect(screen.getByText('zsh')).toBeTruthy()
    expect((screen.getByRole('button', { name: /open zsh/i }) as HTMLButtonElement).disabled).toBe(true)
    expect((screen.getByRole('button', { name: /manage zsh/i }) as HTMLButtonElement).disabled).toBe(true)
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
    foregroundProcess: overrides.foregroundProcess,
    lastOutputAt: overrides.lastOutputAt,
    sizeLocked: overrides.sizeLocked,
    sizeLockMode: overrides.sizeLockMode,
    environment: overrides.environment,
  }
}
