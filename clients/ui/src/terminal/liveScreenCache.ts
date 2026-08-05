import { create } from '@bufbuild/protobuf'
import {
  ScreenRowSchema,
  TerminalCursorSchema,
  TerminalModesSchema,
  type NativeScreenResult,
  type ScreenRow,
  type TerminalCursor,
  type TerminalModes,
} from '../generated/apipb/history_pb'
import { TerminalRefSchema, type TerminalRef } from '../generated/apipb/terminal_pb'

export interface CanonicalLiveScreen {
  terminal: TerminalRef
  generation: bigint
  revision: bigint
  cols: number
  rows: number
  screenRows: ScreenRow[]
  alternateScreen: boolean
  cursor?: TerminalCursor | undefined
  modes?: TerminalModes | undefined
  timestampUnixNano: bigint
}

export interface LiveScreenDamage {
  fullReplace: boolean
  changedRows: number[]
}

export interface LiveScreenMergeResult {
  screen: CanonicalLiveScreen
  damage: LiveScreenDamage
}

/** Merge the daemon's latest-only update. Row copies always read from the same base screen. */
export function mergeLiveScreenResult(
  current: CanonicalLiveScreen | undefined,
  incoming: NativeScreenResult,
  generation: bigint,
  expectedTerminal: TerminalRef,
): LiveScreenMergeResult | undefined {
  const terminal = incoming.terminal ?? expectedTerminal
  if (
    terminal.endpointId !== expectedTerminal.endpointId ||
    terminal.terminalId !== expectedTerminal.terminalId
  ) return undefined

  const cols = incoming.size?.cols ?? 0
  const rows = incoming.size?.rows ?? 0
  if (cols < 0 || rows < 0) return undefined

  if (!incoming.fullReplace) {
    if (
      !current ||
      current.generation !== generation ||
      incoming.baseRevision !== current.revision ||
      current.cols !== cols ||
      current.rows !== rows
    ) return undefined
  }

  const baseRows = incoming.fullReplace
    ? Array.from({ length: rows }, () => create(ScreenRowSchema))
    : current!.screenRows
  const nextRows = baseRows.slice()
  const changedRows = new Set<number>()

  for (const copy of incoming.rowCopies) {
    if (
      copy.count < 0 ||
      copy.sourceRow < 0 ||
      copy.destinationRow < 0 ||
      copy.sourceRow + copy.count > rows ||
      copy.destinationRow + copy.count > rows
    ) return undefined
    const copied = baseRows.slice(copy.sourceRow, copy.sourceRow + copy.count)
    for (let offset = 0; offset < copied.length; offset += 1) {
      const rowIndex = copy.destinationRow + offset
      nextRows[rowIndex] = copied[offset]!
      changedRows.add(rowIndex)
    }
  }

  for (const replacement of incoming.rowReplacements) {
    if (replacement.rowIndex < 0 || replacement.rowIndex >= rows || !replacement.row) return undefined
    nextRows[replacement.rowIndex] = create(ScreenRowSchema, replacement.row)
    changedRows.add(replacement.rowIndex)
  }

  return {
    screen: {
      terminal: create(TerminalRefSchema, terminal),
      generation,
      revision: incoming.liveRevision,
      cols,
      rows,
      screenRows: nextRows,
      alternateScreen: incoming.alternateScreen,
      cursor: incoming.cursor ? create(TerminalCursorSchema, incoming.cursor) : undefined,
      modes: incoming.modes ? create(TerminalModesSchema, incoming.modes) : undefined,
      timestampUnixNano: incoming.timestampUnixNano,
    },
    damage: {
      fullReplace: incoming.fullReplace,
      changedRows: incoming.fullReplace
        ? Array.from({ length: rows }, (_, rowIndex) => rowIndex)
        : [...changedRows].sort((left, right) => left - right),
    },
  }
}
