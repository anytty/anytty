import { create } from '@bufbuild/protobuf'
import { describe, expect, it, vi } from 'vitest'
import {
  copyHistorySelection,
  rangeFromHistorySelection,
  searchHistorySurface,
  selectionFromSurfaceRows,
} from './coreV2HistoryInteraction'
import { createCoreV2HistorySource } from './coreV2HistorySource'
import { HistoryCopyResultSchema } from '../generated/apipb/history_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import type { CoreV2HistorySource } from './coreV2HistorySource'
import type { CoreV2HistorySurfaceSnapshot } from './coreV2HistorySurface'
import type { CoreV2HistoryRow } from './coreV2TerminalProtocol'

describe('CoreV2 history interaction', () => {
  it('copies final text through core-v2 history.copy logical range', async () => {
    const session = new MockProtoSession('machine-local', () => protoResult(
      'historyCopy',
      create(HistoryCopyResultSchema, { text: 'copied from core', done: true }),
    ))
    const source = createCoreV2HistorySource(session, 'machine-local')
    const snapshot = surfaceSnapshot([
      row('41', 0, 'alpha  '),
      row('42', 0, 'beta'),
      row('43', 0, 'gamma'),
    ])
    const liveSelection = vi.fn(() => {
      throw new Error('live xterm selection must not be used')
    })
    const loadScrollback = vi.fn(() => {
      throw new Error('visual scrollback must not be used')
    })
    const domText = vi.fn(() => {
      throw new Error('renderer text must not be used')
    })

    const selection = selectionFromSurfaceRows(
      snapshot,
      { lineId: '41', rowInLine: 0, col: 2 },
      { lineId: '43', rowInLine: 0, col: 3 },
    )
    const copied = await copyHistorySelection(source, snapshot, selection)

    expect(copied).toBe('copied from core')
    expect(session.commands).toHaveLength(1)
    expect(session.commands[0]?.command).toMatchObject({
      case: 'historyCopy',
      value: {
        terminal: { endpointId: 'machine-local', terminalId: 'terminal-1' },
        window: {
          token: 'token-1', historyGeneration: 7n, boundaryFirstLineId: 41n, boundaryLastLineId: 43n,
          range: { startLineId: 41n, startCol: 2, endLineId: 43n, endCol: 3 },
        },
      },
    })
    expect(liveSelection).not.toHaveBeenCalled()
    expect(loadScrollback).not.toHaveBeenCalled()
    expect(domText).not.toHaveBeenCalled()
  })

  it('normalizes reversed selections and clamps columns to logical line width', () => {
    const snapshot = surfaceSnapshot([
      row('41', 0, 'alpha'),
      row('42', 0, 'beta'),
    ])

    expect(rangeFromHistorySelection(snapshot, {
      anchor: { lineId: '42', col: 99 },
      focus: { lineId: '41', col: -1 },
    })).toEqual({
      startLineId: '41',
      startCol: 0,
      endLineId: '42',
      endCol: 4,
    })
  })

  it('searches the complete frozen snapshot through the source in either direction', async () => {
    const snapshot = surfaceSnapshot([
      row('41', 0, 'alpha ALPHA'),
      row('42', 0, 'beta'),
    ])
    const result = {
      found: true as const,
      match: { startLineId: '4', startCol: 3, endLineId: '4', endCol: 8 },
      window: { ...emptyWindow(), firstLineId: '4', lastLineId: '4' },
      wrapped: true,
    }
    const source: Pick<CoreV2HistorySource, 'search'> = { search: vi.fn(async () => result) }

    await expect(searchHistorySurface(source, snapshot, 'alpha', {
      direction: 'backward', start: { lineId: '41', col: 6 }, limit: 12,
    })).resolves.toEqual(result)
    expect(source.search).toHaveBeenCalledWith({
      terminalId: 'terminal-1', token: 'token-1', generation: '7', query: 'alpha',
      mode: 'text', direction: 'backward', cols: 80, limit: 12,
      start: { lineId: '41', col: 6 },
    })
  })

  it('rejects stale surfaces before building copy/search ranges', async () => {
    const source: Pick<CoreV2HistorySource, 'copy' | 'search'> = {
      copy: vi.fn(async () => 'should not run'),
      search: vi.fn(async () => ({ found: false, wrapped: false } as const)),
    }
    const snapshot = {
      ...surfaceSnapshot([row('41', 0, 'alpha')]),
      stale: true,
      staleReason: 'generation changed',
    }

    await expect(copyHistorySelection(source, snapshot, {
      anchor: { lineId: '41', col: 0 },
      focus: { lineId: '41', col: 5 },
    })).rejects.toThrow(/stale/)
    await expect(searchHistorySurface(source, snapshot, 'alpha')).rejects.toThrow(/stale/)
    expect(source.search).not.toHaveBeenCalled()
  })
})

function emptyWindow() {
  return {
    terminalId: 'terminal-1', token: 'token-1', op: 'replace' as const, cols: 80,
    renderRows: [], lines: [], totalRows: 2, logicalTotal: 2, hasMore: false, generation: '7',
  }
}

function surfaceSnapshot(rows: CoreV2HistoryRow[]): CoreV2HistorySurfaceSnapshot {
  const firstLineId = rows[0]?.logicalLineId ?? '0'
  const lastLineId = rows.at(-1)?.logicalLineId ?? firstLineId
  return {
    terminalId: 'terminal-1',
    cols: 80,
    viewportRows: 24,
    revision: 1,
    token: 'token-1',
    generation: '7',
    stale: false,
    staleReason: null,
    rows,
    renderRows: rows,
    lines: rows.map((item, index) => ({
      startRow: index,
      endRow: index,
      logicalLineId: item.logicalLineId ?? '',
      clippedBefore: false,
      clippedAfter: false,
    })),
    loadedRows: rows.length,
    totalRows: rows.length,
    logicalTotal: rows.length,
    hasOlder: false,
    hasNewer: false,
    renderWindow: {
      startIndex: 0,
      endIndex: rows.length,
      viewportStartIndex: 0,
      viewportEndIndex: rows.length,
      viewportOffsetRows: 0,
      overscanRows: 0,
      edgePrefetchRows: 0,
      shouldPrefetchOlder: false,
      shouldPrefetchNewer: false,
    },
    firstCursor: { lineId: firstLineId, rowInLine: 0 },
    lastCursor: { lineId: lastLineId, rowInLine: 0 },
    firstLineId,
    lastLineId,
  }
}

function row(lineId: string, rowInLine: number, text: string): CoreV2HistoryRow {
  return {
    index: Number(lineId),
    logicalLineId: lineId,
    rowInLine,
    cells: [{ text, width: text.length }],
  }
}
