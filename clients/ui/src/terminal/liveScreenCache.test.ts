import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import {
  NativeScreenResultSchema,
  ScreenCellSchema,
  ScreenRowCopySchema,
  ScreenRowReplaceSchema,
  ScreenRowSchema,
} from '../generated/apipb/history_pb'
import { TerminalRefSchema, TerminalSizeSchema } from '../generated/apipb/terminal_pb'
import { mergeLiveScreenResult, type CanonicalLiveScreen } from './liveScreenCache'

const terminal = create(TerminalRefSchema, { endpointId: 'machine-1', terminalId: 'terminal-1' })

describe('mergeLiveScreenResult', () => {
  it('reads every row copy from the same base and applies replacements afterward', () => {
    const base = canonical(1n, ['A', 'B', 'C'])
    const update = create(NativeScreenResultSchema, {
      terminal,
      liveRevision: 2n,
      baseRevision: 1n,
      size: create(TerminalSizeSchema, { cols: 1, rows: 3 }),
      rowCopies: [create(ScreenRowCopySchema, {
        sourceRow: 0,
        destinationRow: 1,
        count: 2,
      })],
      rowReplacements: [create(ScreenRowReplaceSchema, {
        rowIndex: 1,
        row: row('X'),
      })],
    })

    const merged = mergeLiveScreenResult(base, update, 1n, terminal)

    expect(merged?.screen.screenRows.map(rowText)).toEqual(['A', 'X', 'B'])
    expect(merged?.damage).toEqual({ fullReplace: false, changedRows: [1, 2] })
  })

  it('rejects a delta whose base revision does not match the canonical screen', () => {
    const update = create(NativeScreenResultSchema, {
      terminal,
      liveRevision: 3n,
      baseRevision: 2n,
      size: create(TerminalSizeSchema, { cols: 1, rows: 1 }),
      rowReplacements: [create(ScreenRowReplaceSchema, { rowIndex: 0, row: row('B') })],
    })

    expect(mergeLiveScreenResult(canonical(1n, ['A']), update, 1n, terminal)).toBeUndefined()
  })
})

function canonical(revision: bigint, values: string[]): CanonicalLiveScreen {
  return {
    terminal,
    generation: 1n,
    revision,
    cols: 1,
    rows: values.length,
    screenRows: values.map(row),
    alternateScreen: false,
    timestampUnixNano: 0n,
  }
}

function row(text: string) {
  return create(ScreenRowSchema, {
    cells: [create(ScreenCellSchema, { content: text, width: 1 })],
  })
}

function rowText(value: ReturnType<typeof row>): string {
  return value.cells.map((cell) => cell.content).join('')
}
