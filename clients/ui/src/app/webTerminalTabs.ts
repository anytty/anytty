import {
  PRIMARY_TERMINAL_PANE,
  removeTerminalPane,
  terminalPaneKey,
  terminalPaneKeys,
  type TerminalPaneKey,
  type TerminalSplitNode,
} from './terminalSplitLayout'

export interface WebTerminalTabLayout {
  terminalId: string
  root: TerminalSplitNode
  activePaneKey: TerminalPaneKey
}

export function createWebTerminalTab(terminalId: string): WebTerminalTabLayout {
  return {
    terminalId,
    root: PRIMARY_TERMINAL_PANE,
    activePaneKey: 'primary',
  }
}

export function webTerminalTabPaneKey(tab: WebTerminalTabLayout, terminalId: string): TerminalPaneKey | null {
  if (tab.terminalId === terminalId) return 'primary'
  const paneKey = terminalPaneKey(terminalId)
  return terminalPaneKeys(tab.root).includes(paneKey) ? paneKey : null
}

export function findWebTerminalTab(tabs: readonly WebTerminalTabLayout[], terminalId: string): { tab: WebTerminalTabLayout; paneKey: TerminalPaneKey } | null {
  for (const tab of tabs) {
    const paneKey = webTerminalTabPaneKey(tab, terminalId)
    if (paneKey) return { tab, paneKey }
  }
  return null
}

export function updateWebTerminalTab(
  tabs: WebTerminalTabLayout[],
  terminalId: string,
  update: Pick<WebTerminalTabLayout, 'root' | 'activePaneKey'>,
): WebTerminalTabLayout[] {
  let changed = false
  const next = tabs.map((tab) => {
    if (tab.terminalId !== terminalId || (tab.root === update.root && tab.activePaneKey === update.activePaneKey)) return tab
    changed = true
    return { ...tab, ...update }
  })
  return changed ? next : tabs
}

export function sanitizeWebTerminalTabs(tabs: WebTerminalTabLayout[], availableTerminalIds: ReadonlySet<string>): WebTerminalTabLayout[] {
  let changed = false
  const next = tabs.flatMap((tab) => {
    if (!availableTerminalIds.has(tab.terminalId)) {
      changed = true
      return []
    }
    const root = terminalPaneKeys(tab.root).reduce<TerminalSplitNode>((current, paneKey) => {
      if (paneKey === 'primary') return current
      const terminalId = paneKey.slice('terminal:'.length)
      return availableTerminalIds.has(terminalId)
        ? current
        : removeTerminalPane(current, paneKey) ?? PRIMARY_TERMINAL_PANE
    }, tab.root)
    const activePaneKey = terminalPaneKeys(root).includes(tab.activePaneKey) ? tab.activePaneKey : 'primary'
    if (root === tab.root && activePaneKey === tab.activePaneKey) return [tab]
    changed = true
    return [{ ...tab, root, activePaneKey }]
  })
  return changed ? next : tabs
}
