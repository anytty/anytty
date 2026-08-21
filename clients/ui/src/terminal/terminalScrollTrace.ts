export interface TerminalScrollTraceEntry {
  sequence: number
  at: number
  event: string
  machineId?: string | undefined
  terminalId?: string | undefined
  details?: Record<string, unknown> | undefined
}

interface TerminalScrollTraceState {
  enabled?: boolean
  events?: TerminalScrollTraceEntry[]
  sequence?: number
}

interface TerminalScrollTraceGlobal {
  __anyttyTerminalScrollTrace?: TerminalScrollTraceState
}

const maximumTraceEntries = 2000

function traceState(): TerminalScrollTraceState | undefined {
  return (globalThis as TerminalScrollTraceGlobal).__anyttyTerminalScrollTrace
}

export function traceTerminalScroll(
  event: string,
  input: {
    machineId?: string | undefined
    terminalId?: string | undefined
    details?: Record<string, unknown> | undefined
  } = {},
): void {
  try {
    const state = traceState()
    if (!state?.enabled) return
    const events = state.events ?? (state.events = [])
    const entry: TerminalScrollTraceEntry = {
      sequence: (state.sequence ?? 0) + 1,
      at: typeof performance !== 'undefined' && typeof performance.now === 'function'
        ? Math.round(performance.now() * 1000) / 1000
        : Date.now(),
      event,
      machineId: input.machineId,
      terminalId: input.terminalId,
      details: input.details,
    }
    state.sequence = entry.sequence
    events.push(entry)
    if (events.length > maximumTraceEntries) events.splice(0, events.length - maximumTraceEntries)
    console.debug(`[anytty:scroll] ${event}`, entry)
  } catch {
    // Scroll diagnostics must never affect input handling.
  }
}

export interface TerminalMouseInputDescription {
  encoding: 'sgr'
  code: number
  col: number
  row: number
  action: 'press' | 'release'
  direction: 'up' | 'down' | 'horizontal-or-other'
}

export function describeTerminalMouseInput(data: string): TerminalMouseInputDescription | undefined {
  if (!data.startsWith('\u001b[<')) return undefined
  const match = /^(\d+);(\d+);(\d+)([mM])$/.exec(data.slice(3))
  if (!match) return undefined
  const code = Number(match[1])
  const wheelCode = code & 0b11
  return {
    encoding: 'sgr',
    code,
    col: Number(match[2]),
    row: Number(match[3]),
    action: match[4] === 'M' ? 'press' : 'release',
    direction: (code & 64) === 0
      ? 'horizontal-or-other'
      : wheelCode === 0
        ? 'up'
        : wheelCode === 1
          ? 'down'
          : 'horizontal-or-other',
  }
}
