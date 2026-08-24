import { describe, expect, it } from 'vitest'
import {
  PRIMARY_TERMINAL_PANE,
  removeTerminalPane,
  splitTerminalPane,
  terminalPaneKey,
  terminalPaneKeys,
  updateTerminalSplitRatio,
} from './terminalSplitLayout'

describe('terminalSplitLayout', () => {
  it('nests any number of panes below the active pane', () => {
    const two = splitTerminalPane(PRIMARY_TERMINAL_PANE, 'primary', 'logs', 'rows', 'split-1')
    const three = splitTerminalPane(two, terminalPaneKey('logs'), 'server', 'rows', 'split-2')
    const four = splitTerminalPane(three, terminalPaneKey('server'), 'tests', 'rows', 'split-3')

    expect(terminalPaneKeys(four)).toEqual([
      'primary',
      terminalPaneKey('logs'),
      terminalPaneKey('server'),
      terminalPaneKey('tests'),
    ])
  })

  it('moves an existing pane and collapses its previous branch', () => {
    const rows = splitTerminalPane(PRIMARY_TERMINAL_PANE, 'primary', 'logs', 'rows', 'split-1')
    const nested = splitTerminalPane(rows, terminalPaneKey('logs'), 'server', 'rows', 'split-2')
    const moved = splitTerminalPane(nested, 'primary', 'server', 'columns', 'split-3')

    expect(terminalPaneKeys(moved)).toEqual(['primary', terminalPaneKey('server'), terminalPaneKey('logs')])
  })

  it('removes a pane and updates one divider without changing the rest of the tree', () => {
    const rows = splitTerminalPane(PRIMARY_TERMINAL_PANE, 'primary', 'logs', 'rows', 'split-1')
    const nested = splitTerminalPane(rows, terminalPaneKey('logs'), 'server', 'rows', 'split-2')
    const resized = updateTerminalSplitRatio(nested, 'split-2', 90)
    const removed = removeTerminalPane(resized, terminalPaneKey('logs'))

    expect(resized.type === 'split' && resized.second.type === 'split' ? resized.second.ratio : null).toBe(80)
    expect(removed && terminalPaneKeys(removed)).toEqual(['primary', terminalPaneKey('server')])
  })

  it('inserts panes before the target for top and left splits', () => {
    const left = splitTerminalPane(PRIMARY_TERMINAL_PANE, 'primary', 'logs', 'columns', 'split-1', 'before')
    expect(terminalPaneKeys(left)).toEqual([terminalPaneKey('logs'), 'primary'])
  })

  it('replaces only the targeted leaf when nesting different split directions', () => {
    const columns = splitTerminalPane(PRIMARY_TERMINAL_PANE, 'primary', 'logs', 'columns', 'split-1')
    const leftRows = splitTerminalPane(columns, 'primary', 'server', 'rows', 'split-2')

    expect(leftRows).toMatchObject({
      type: 'split',
      direction: 'columns',
      first: {
        type: 'split',
        direction: 'rows',
        first: { type: 'pane', paneKey: 'primary' },
        second: { type: 'pane', paneKey: terminalPaneKey('server') },
      },
      second: { type: 'pane', paneKey: terminalPaneKey('logs') },
    })
  })
})
