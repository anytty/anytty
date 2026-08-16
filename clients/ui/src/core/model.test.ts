import { describe, expect, it } from 'vitest'
import {
  assertRemoteModelShape,
  normalizeMachine,
  normalizeTerminal,
  type Machine,
  type Terminal,
} from './model'

describe('remote public model', () => {
  it('normalizes machine and terminal records without tgent layout concepts', () => {
    const machine: Machine = normalizeMachine({
      machine_id: 'machine-local',
      name: 'Local Mac',
      state: 'online',
      terminal_count: 2,
      local_rtc: {
        signaling_url: 'http://127.0.0.1:18888',
        ice_tcp_url: 'tcp://127.0.0.1:18889',
      },
    })

    const terminal: Terminal = normalizeTerminal({
      terminal_id: 'term-1',
      machine_id: 'machine-local',
      title: 'zsh',
      state: 'running',
      command: '/bin/zsh',
      cols: 120,
      rows: 36,
      foreground_process: 'gemini',
      last_output_at: '2026-08-13T10:00:00Z',
    })

    expect(machine.machineId).toBe('machine-local')
    expect(machine.terminalCount).toBe(2)
    expect(terminal.terminalId).toBe('term-1')
    expect(terminal.machineId).toBe(machine.machineId)
    expect(terminal.foregroundProcess).toBe('gemini')
    expect(terminal.lastOutputAt).toBe('2026-08-13T10:00:00Z')
  })

  it('rejects workspace/tab/window/pane-shaped public records', () => {
    expect(() =>
      assertRemoteModelShape({
        machine_id: 'machine-local',
        workspace_id: 'workspace-1',
      }),
    ).toThrow(/workspace_id/)

    expect(() =>
      normalizeTerminal({
        terminal_id: 'term-1',
        machine_id: 'machine-local',
        pane_id: 'pane-1',
      }),
    ).toThrow(/pane_id/)

    expect(() =>
      assertRemoteModelShape({
        machine: {
          machine_id: 'machine-local',
          windows: [{ window_id: 'window-1' }],
        },
      }),
    ).toThrow(/windows/)
  })

  it('keeps signaling session identifiers out of the terminal model instead of rejecting all session_id usage globally', () => {
    expect(() =>
      assertRemoteModelShape({
        session_id: 'rtc-signaling-session',
        terminal_id: 'term-1',
      }),
    ).not.toThrow()
  })

  it('normalizes lifecycle command arrays so terminal exit updates remain consumable', () => {
    expect(normalizeTerminal({
      terminal_id: 'term-1',
      machine_id: 'machine-local',
      state: 'exited',
      command: ['/bin/zsh', '-l'],
    })).toMatchObject({
      state: 'exited',
      command: '/bin/zsh -l',
    })
  })
})
