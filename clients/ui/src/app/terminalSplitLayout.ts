export type TerminalPaneKey = 'primary' | `terminal:${string}`
export type TerminalSplitDirection = 'columns' | 'rows'
export type TerminalSplitPlacement = 'before' | 'after'

export type TerminalSplitNode =
  | { type: 'pane'; paneKey: TerminalPaneKey }
  | {
      type: 'split'
      id: string
      direction: TerminalSplitDirection
      ratio: number
      first: TerminalSplitNode
      second: TerminalSplitNode
    }

export const PRIMARY_TERMINAL_PANE: TerminalSplitNode = { type: 'pane', paneKey: 'primary' }

export function terminalPaneKey(terminalId: string): TerminalPaneKey {
  return `terminal:${terminalId}`
}

export function terminalIdForPane(paneKey: TerminalPaneKey, primaryTerminalId: string | null): string | null {
  return paneKey === 'primary' ? primaryTerminalId : paneKey.slice('terminal:'.length)
}

export function terminalPaneKeys(root: TerminalSplitNode): TerminalPaneKey[] {
  if (root.type === 'pane') return [root.paneKey]
  return [...terminalPaneKeys(root.first), ...terminalPaneKeys(root.second)]
}

export function splitTerminalPane(
  root: TerminalSplitNode,
  targetPaneKey: TerminalPaneKey,
  terminalId: string,
  direction: TerminalSplitDirection,
  splitId: string,
  placement: TerminalSplitPlacement = 'after',
): TerminalSplitNode {
  const newPaneKey = terminalPaneKey(terminalId)
  if (newPaneKey === targetPaneKey) return root
  const withoutExisting = removeTerminalPane(root, newPaneKey) ?? PRIMARY_TERMINAL_PANE
  if (!terminalPaneKeys(withoutExisting).includes(targetPaneKey)) return root
  return replaceTerminalPane(withoutExisting, targetPaneKey, {
    type: 'split',
    id: splitId,
    direction,
    ratio: 50,
    first: { type: 'pane', paneKey: placement === 'before' ? newPaneKey : targetPaneKey },
    second: { type: 'pane', paneKey: placement === 'before' ? targetPaneKey : newPaneKey },
  })
}

export function removeTerminalPane(root: TerminalSplitNode, paneKey: TerminalPaneKey): TerminalSplitNode | null {
  if (paneKey === 'primary') return root
  if (root.type === 'pane') return root.paneKey === paneKey ? null : root
  const first = removeTerminalPane(root.first, paneKey)
  const second = removeTerminalPane(root.second, paneKey)
  if (!first) return second
  if (!second) return first
  if (first === root.first && second === root.second) return root
  return { ...root, first, second }
}

export function updateTerminalSplitRatio(root: TerminalSplitNode, splitId: string, ratio: number): TerminalSplitNode {
  if (root.type === 'pane') return root
  if (root.id === splitId) return { ...root, ratio: clampTerminalSplitRatio(ratio) }
  const first = updateTerminalSplitRatio(root.first, splitId, ratio)
  const second = updateTerminalSplitRatio(root.second, splitId, ratio)
  if (first === root.first && second === root.second) return root
  return { ...root, first, second }
}

function replaceTerminalPane(root: TerminalSplitNode, paneKey: TerminalPaneKey, replacement: TerminalSplitNode): TerminalSplitNode {
  if (root.type === 'pane') return root.paneKey === paneKey ? replacement : root
  const first = replaceTerminalPane(root.first, paneKey, replacement)
  const second = replaceTerminalPane(root.second, paneKey, replacement)
  if (first === root.first && second === root.second) return root
  return { ...root, first, second }
}

function clampTerminalSplitRatio(value: number): number {
  return Math.max(20, Math.min(80, value))
}
