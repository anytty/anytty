import { describe, expect, it } from 'vitest'
import {
  createTerminalInventorySnapshot,
  normalizeTerminalInventory,
  selectTerminal,
} from './terminalInventory'

describe('terminalInventory', () => {
  it('normalizes a machine scoped terminal list without tgent layout concepts', () => {
    const snapshot = normalizeTerminalInventory({
      machine_id: 'machine-local',
      terminals: [
        {
          terminal_id: 'terminal-1',
          machine_id: 'machine-local',
          name: 'zsh',
          command: '/bin/zsh',
          cols: 120,
          rows: 36,
          state: 'running',
          last_active_at: '2026-05-01T10:00:00Z',
        },
      ],
    })

    expect(snapshot.machineId).toBe('machine-local')
    expect(snapshot.terminals).toHaveLength(1)
    expect(snapshot.terminals[0]).toEqual(
      expect.objectContaining({
        terminalId: 'terminal-1',
        machineId: 'machine-local',
        title: 'zsh',
      }),
    )
    expect(JSON.stringify(snapshot)).not.toMatch(/workspace|tab|window|pane|session/i)
  })

  it('normalizes runtime terminal inventory records that use Go JSON field names', () => {
    const snapshot = normalizeTerminalInventory({
      machine_id: 'machine-cloud',
      terminals: [
        {
          ID: '1',
          Name: 'shell',
          State: 'running',
          Command: ['/bin/sh'],
          Cols: 80,
          Rows: 24,
        },
      ],
    })

    expect(snapshot.terminals).toEqual([
      expect.objectContaining({
        terminalId: '1',
        machineId: 'machine-cloud',
        title: 'shell',
        state: 'running',
        command: '/bin/sh',
        cols: 80,
        rows: 24,
      }),
    ])
  })

  it('deduplicates repeated terminal ids from runtime inventory responses', () => {
    const snapshot = normalizeTerminalInventory({
      machine_id: 'machine-cloud',
      terminals: [
        {
          terminal_id: '1',
          title: '1',
          state: 'running',
        },
        {
          terminal_id: '1',
          name: 'dev shell',
          command: ['/bin/zsh', '-l'],
          cols: 120,
          rows: 36,
          state: 'running',
        },
        {
          terminal_id: '2',
          name: 'worker',
          state: 'running',
        },
      ],
    })

    expect(snapshot.terminals).toHaveLength(2)
    expect(snapshot.terminals.map((terminal) => terminal.terminalId)).toEqual(['1', '2'])
    expect(snapshot.terminals[0]).toEqual(expect.objectContaining({
      terminalId: '1',
      title: 'dev shell',
      command: '/bin/zsh -l',
      cols: 120,
      rows: 36,
    }))
  })

  it('rejects tgent session/window/pane-shaped inventory records', () => {
    expect(() =>
      normalizeTerminalInventory({
        machine_id: 'machine-local',
        sessions: [
          {
            id: 'session-1',
            windows: [{ id: 'window-1', panes: [{ id: 'pane-1' }] }],
          },
        ],
      } as never),
    ).toThrow(/windows|panes|session/)

    expect(() =>
      normalizeTerminalInventory({
        machine_id: 'machine-local',
        terminals: [
          {
            terminal_id: 'terminal-1',
            machine_id: 'machine-local',
            title: 'zsh',
          },
        ],
        sessions: [],
      } as never),
    ).toThrow(/sessions/)
  })

  it('rejects terminals that explicitly belong to another machine', () => {
    expect(() =>
      normalizeTerminalInventory({
        machine_id: 'machine-a',
        terminals: [
          {
            terminal_id: 'terminal-1',
            machine_id: 'machine-b',
            title: 'zsh',
          },
        ],
      }),
    ).toThrow(/machine-b/)
  })

  it('updates user intent through the reducer when a terminal is selected', () => {
    const initial = createTerminalInventorySnapshot('machine-local', [])
    const selected = selectTerminal(initial, 'terminal-1')

    expect(selected.connection.userIntent).toEqual({
      kind: 'terminal',
      machineId: 'machine-local',
      terminalId: 'terminal-1',
    })
    expect(selected.connection.activeTerminalId).toBe('terminal-1')
  })
})
