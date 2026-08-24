import { describe, expect, it } from 'vitest'
import { splitTerminalPane, terminalPaneKey } from './terminalSplitLayout'
import {
  createWebTerminalTab,
  findWebTerminalTab,
  sanitizeWebTerminalTabs,
  updateWebTerminalTab,
} from './webTerminalTabs'

describe('webTerminalTabs', () => {
  it('keeps a split tree and focused pane scoped to its owning tab', () => {
    const shell = createWebTerminalTab('shell')
    const logsRoot = splitTerminalPane(shell.root, 'primary', 'logs', 'columns', 'split-1')
    const tabs = updateWebTerminalTab([shell, createWebTerminalTab('server')], 'shell', {
      root: logsRoot,
      activePaneKey: terminalPaneKey('logs'),
    })

    expect(tabs[0]).toMatchObject({ terminalId: 'shell', root: logsRoot, activePaneKey: terminalPaneKey('logs') })
    expect(tabs[1]).toEqual(createWebTerminalTab('server'))
    expect(findWebTerminalTab(tabs, 'logs')).toEqual({ tab: tabs[0], paneKey: terminalPaneKey('logs') })
    expect(findWebTerminalTab(tabs, 'server')).toEqual({ tab: tabs[1], paneKey: 'primary' })
  })

  it('removes unavailable tabs and invalid split leaves without changing other layouts', () => {
    const shell = createWebTerminalTab('shell')
    const logsRoot = splitTerminalPane(shell.root, 'primary', 'logs', 'rows', 'split-1')
    const tabs = [
      { ...shell, root: logsRoot, activePaneKey: terminalPaneKey('logs') },
      createWebTerminalTab('server'),
    ]

    expect(sanitizeWebTerminalTabs(tabs, new Set(['shell']))).toEqual([createWebTerminalTab('shell')])
  })
})
