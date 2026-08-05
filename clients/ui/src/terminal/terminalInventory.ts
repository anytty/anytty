import {
  initialConnectionSnapshot,
  reduceConnectionMessage,
  type ConnectionSnapshot,
} from '../connection/connectionMessageReducer'
import { assertRemoteModelShape, normalizeTerminal, type Terminal } from '../core/model'

export interface TerminalInventorySnapshot {
  machineId: string
  terminals: Terminal[]
  connection: ConnectionSnapshot
}

export interface TerminalInventoryInput {
  machine_id?: string
  machineId?: string
  terminals?: Record<string, unknown>[]
}

export function normalizeTerminalInventory(input: TerminalInventoryInput): TerminalInventorySnapshot {
  assertRemoteModelShape(input)
  assertNoTerminalListSessionConcepts(input)

  const machineId = input.machine_id ?? input.machineId
  if (!machineId) {
    throw new Error('machine_id is required')
  }
  if (!Array.isArray(input.terminals)) {
    throw new Error('terminals must be an array')
  }

  const rawTerminals = input.terminals
  const snapshot = createTerminalInventorySnapshot(
    machineId,
    rawTerminals.map((terminal) => {
      assertNoTerminalListSessionConcepts(terminal)
      const terminalId = terminal.terminal_id ?? terminal.terminalId ?? terminal.id ?? terminal.ID
      const command = normalizeCommand(terminal.command ?? terminal.Command)
      const normalized = normalizeTerminal({
        ...terminal,
        terminal_id: terminalId,
        machine_id: terminal.machine_id ?? terminal.machineId ?? machineId,
        title: terminal.title ?? terminal.name ?? terminal.Name ?? terminalId,
        state: terminal.state ?? terminal.State,
        command,
        cols: terminal.cols ?? terminal.Cols,
        rows: terminal.rows ?? terminal.Rows,
      })
      if (normalized.machineId !== machineId) {
        throw new Error(`terminal ${normalized.terminalId} belongs to ${normalized.machineId}, not ${machineId}`)
      }
      return normalized
    }),
  )
  logTerminalInventory('normalized', {
    machineId,
    rawCount: rawTerminals.length,
    normalizedCount: snapshot.terminals.length,
    rawTerminals: rawTerminals.slice(0, 25).map(summarizeRawTerminal),
    normalizedTerminals: snapshot.terminals.slice(0, 25).map(summarizeNormalizedTerminal),
  })
  return snapshot
}

function normalizeCommand(value: unknown): string | undefined {
  if (Array.isArray(value) && value.every((item) => typeof item === 'string')) {
    return value.join(' ')
  }
  return value as string | undefined
}

export function createTerminalInventorySnapshot(
  machineId: string,
  terminals: Terminal[],
): TerminalInventorySnapshot {
  return {
    machineId,
    terminals: dedupeTerminals(terminals.map((terminal) => ({ ...terminal, machineId }))),
    connection: reduceConnectionMessage(initialConnectionSnapshot(), {
      type: 'user.connectMachine',
      machineId,
    }),
  }
}

function dedupeTerminals(terminals: Terminal[]): Terminal[] {
  const positions = new Map<string, number>()
  const out: Terminal[] = []
  for (const terminal of terminals) {
    const key = `${terminal.machineId}:${terminal.terminalId}`
    const index = positions.get(key)
    if (index === undefined) {
      positions.set(key, out.length)
      out.push(terminal)
      continue
    }
    out[index] = preferTerminalInventoryRecord(out[index]!, terminal)
  }
  return out
}

function preferTerminalInventoryRecord(current: Terminal, candidate: Terminal): Terminal {
  const currentTitleIsID = current.title === current.terminalId
  const candidateTitleIsID = candidate.title === candidate.terminalId
  if (currentTitleIsID && !candidateTitleIsID) return candidate
  if (candidateTitleIsID && !currentTitleIsID) return current

  const currentScore = terminalInventoryDetailScore(current)
  const candidateScore = terminalInventoryDetailScore(candidate)
  return candidateScore > currentScore ? candidate : current
}

function terminalInventoryDetailScore(terminal: Terminal): number {
  return [
    terminal.title && terminal.title !== terminal.terminalId,
    terminal.command,
    terminal.cwd,
    terminal.environment,
    terminal.cols,
    terminal.rows,
    terminal.sizeLockMode,
    terminal.lastActiveAt,
  ].filter(Boolean).length
}

function logTerminalInventory(event: string, details: Record<string, unknown>): void {
  try {
    console.info(`[anytty:terminal-inventory] ${event} ${JSON.stringify(details)}`)
  } catch {
    // Diagnostics must not affect inventory loading.
  }
}

function summarizeRawTerminal(value: Record<string, unknown>): Record<string, unknown> {
  return cleanSummary({
    terminalId: firstString(value.terminal_id, value.terminalId, value.id, value.ID),
    machineId: firstString(value.machine_id, value.machineId),
    name: firstString(value.name, value.title, value.Name),
    state: firstString(value.state, value.State),
    command: normalizeCommand(value.command ?? value.Command),
  })
}

function summarizeNormalizedTerminal(value: Terminal): Record<string, unknown> {
  return cleanSummary({
    terminalId: value.terminalId,
    machineId: value.machineId,
    title: value.title,
    state: value.state,
    command: value.command,
    cols: value.cols,
    rows: value.rows,
  })
}

function firstString(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value === 'string' && value.trim()) return value
  }
  return undefined
}

function cleanSummary<T extends Record<string, unknown>>(record: T): T {
  for (const key of Object.keys(record)) {
    if (record[key] === undefined) delete record[key]
  }
  return record
}

export function selectTerminal(
  snapshot: TerminalInventorySnapshot,
  terminalId: string,
): TerminalInventorySnapshot {
  return {
    ...snapshot,
    connection: reduceConnectionMessage(snapshot.connection, {
      type: 'user.openTerminal',
      machineId: snapshot.machineId,
      terminalId,
    }),
  }
}

function assertNoTerminalListSessionConcepts(value: unknown): void {
  const seen = new Set<unknown>()
  const blocked = new Set(['session', 'sessions', 'session_id', 'sessionId'])

  function walk(current: unknown): void {
    if (!current || typeof current !== 'object') return
    if (seen.has(current)) return
    seen.add(current)

    if (Array.isArray(current)) {
      for (const item of current) walk(item)
      return
    }

    for (const [key, nested] of Object.entries(current as Record<string, unknown>)) {
      if (blocked.has(key)) {
        throw new Error(`terminal inventory must not contain ${key}`)
      }
      walk(nested)
    }
  }

  walk(value)
}
