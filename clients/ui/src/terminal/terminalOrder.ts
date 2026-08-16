const terminalOrderStoragePrefix = 'anytty.terminal-pins.v2:'

export function readTerminalOrder(machineId: string): string[] {
  if (!machineId || typeof window === 'undefined') return []
  try {
    const parsed = JSON.parse(window.localStorage.getItem(`${terminalOrderStoragePrefix}${machineId}`) ?? '[]')
    return Array.isArray(parsed)
      ? parsed.filter((value, index): value is string => typeof value === 'string' && value.length > 0 && parsed.indexOf(value) === index)
      : []
  } catch {
    return []
  }
}

export function writeTerminalOrder(machineId: string, order: string[]): string[] {
  const normalized = order.filter((value, index) => value.length > 0 && order.indexOf(value) === index)
  if (machineId && typeof window !== 'undefined') {
    try {
      window.localStorage?.setItem(`${terminalOrderStoragePrefix}${machineId}`, JSON.stringify(normalized))
    } catch {
      // Local ordering is optional; unavailable client storage must not block terminal access.
    }
  }
  return normalized
}

export function sortTerminalIds<T extends { terminalId: string }>(terminals: T[], order: string[]): T[] {
  const rank = new Map(order.map((terminalId, index) => [terminalId, index]))
  return terminals
    .map((terminal, index) => ({ terminal, index, rank: rank.get(terminal.terminalId) }))
    .sort((left, right) => {
      if (left.rank !== undefined && right.rank !== undefined) return left.rank - right.rank
      if (left.rank !== undefined) return -1
      if (right.rank !== undefined) return 1
      return left.index - right.index
    })
    .map(({ terminal }) => terminal)
}

export function pinTerminal(pinned: string[], terminalId: string): string[] {
  return terminalId && !pinned.includes(terminalId) ? [...pinned, terminalId] : [...pinned]
}

export function unpinTerminal(pinned: string[], terminalId: string): string[] {
  return pinned.filter((id) => id !== terminalId)
}

export function reorderPinnedTerminal(pinned: string[], terminalId: string, targetTerminalId: string, placement: 'before' | 'after' = 'before'): string[] {
  if (terminalId === targetTerminalId || !pinned.includes(terminalId) || !pinned.includes(targetTerminalId)) return [...pinned]
  const next = pinned.filter((id) => id !== terminalId)
  const target = next.indexOf(targetTerminalId)
  next.splice(placement === 'after' ? target + 1 : target, 0, terminalId)
  return next
}
