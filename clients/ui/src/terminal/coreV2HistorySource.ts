import { create } from '@bufbuild/protobuf'
import type { ProtoClientSession } from '../core/protoClientSession'
import { CommandEnvelopeSchema } from '../generated/apipb/application_pb'
import {
  HistoryCopyCommandSchema,
  HistoryCursorSchema,
  HistoryReleaseCommandSchema,
  HistoryRangeSchema,
  HistorySearchCommandSchema,
  HistorySearchDirection,
  HistorySearchMode,
  HistoryTextPositionSchema,
  HistoryWindowCommandSchema,
  HistoryWindowMode,
} from '../generated/apipb/history_pb'
import { TerminalRefSchema } from '../generated/apipb/terminal_pb'
import {
  coreV2HistoryWindowFromAPI,
  coreV2HistorySearchFromAPI,
  coreV2HistorySearchScanFromAPI,
  type CoreV2HistoryCopyRequest,
  type CoreV2HistoryReleaseRequest,
  type CoreV2HistorySearchRequest,
  type CoreV2HistorySearchResult,
  type CoreV2HistorySearchScanRequest,
  type CoreV2HistorySearchScanResult,
  type CoreV2HistoryWindow,
  type CoreV2HistoryWindowRequest,
} from './coreV2TerminalProtocol'

const HISTORY_COPY_CHUNK_LINES = 8192
const HISTORY_COPY_CHUNK_BYTES = 512 * 1024

export interface CoreV2HistorySource {
  window(request: CoreV2HistoryWindowRequest, options?: { signal?: AbortSignal }): Promise<CoreV2HistoryWindow>
  copy(request: CoreV2HistoryCopyRequest, options?: { signal?: AbortSignal }): Promise<string>
  search(request: CoreV2HistorySearchRequest, options?: { signal?: AbortSignal }): Promise<CoreV2HistorySearchResult>
  scan(request: CoreV2HistorySearchScanRequest, options?: { signal?: AbortSignal }): Promise<CoreV2HistorySearchScanResult>
  release?(
    request: CoreV2HistoryReleaseRequest & { generation?: string | number | bigint | undefined },
    options?: { signal?: AbortSignal },
  ): Promise<void>
}

export function createCoreV2HistorySource(
  session: ProtoClientSession,
  machineId: string,
): CoreV2HistorySource {
  return createProtoHistorySource(session, machineId)
}

function createProtoHistorySource(session: ProtoClientSession, machineId: string): CoreV2HistorySource {
  if (session.stamp.endpointId !== machineId) {
    throw new Error(`history source session machine mismatch: connected to ${session.stamp.endpointId}, expected ${machineId}`)
  }
  const terminal = (terminalId: string) => create(TerminalRefSchema, { endpointId: machineId, terminalId })
  const windowCommand = (request: CoreV2HistoryWindowRequest) => create(HistoryWindowCommandSchema, {
    terminal: terminal(request.terminalId),
    mode: protoHistoryMode(request.mode),
    limit: request.limit,
    cols: request.cols,
    token: request.token ?? '',
    historyGeneration: BigInt(request.generation ?? 0),
    beforeCursor: request.beforeCursor ? create(HistoryCursorSchema, { lineId: BigInt(request.beforeCursor.lineId), rowInLine: request.beforeCursor.rowInLine }) : undefined,
    afterCursor: request.afterCursor ? create(HistoryCursorSchema, { lineId: BigInt(request.afterCursor.lineId), rowInLine: request.afterCursor.rowInLine }) : undefined,
    boundaryFirstLineId: BigInt(request.boundaryFirstLineId ?? 0),
    boundaryLastLineId: BigInt(request.boundaryLastLineId ?? 0),
    range: request.range ? create(HistoryRangeSchema, {
      startLineId: BigInt(request.range.startLineId), startCol: request.range.startCol,
      endLineId: BigInt(request.range.endLineId), endCol: request.range.endCol,
    }) : undefined,
  })
  return {
    async window(request, options) {
      const command = windowCommand(request)
      const result = await session.execute(
        create(CommandEnvelopeSchema, { command: { case: 'historyWindow', value: command } }),
        options,
      )
      if (result.result.case !== 'historyWindow') throw new Error('history window returned no result')
      return coreV2HistoryWindowFromAPI(result.result.value)
    },
    async copy(request, options) {
      const chunks: string[] = []
      let range = request.range
      for (;;) {
        const command = create(HistoryCopyCommandSchema, {
          terminal: terminal(request.terminalId),
          window: windowCommand({ ...request, mode: 'range', limit: 1, range }),
          maxLines: HISTORY_COPY_CHUNK_LINES,
          maxBytes: HISTORY_COPY_CHUNK_BYTES,
        })
        const result = await session.execute(
          create(CommandEnvelopeSchema, { command: { case: 'historyCopy', value: command } }),
          options,
        )
        if (result.result.case !== 'historyCopy') throw new Error('history copy returned no result')
        chunks.push(result.result.value.text)
        if (result.result.value.done) return chunks.join('\n')
        const next = result.result.value.next
        if (!next || next.lineId === 0n) throw new Error('history copy returned no continuation')
        const nextStart = { lineId: next.lineId.toString(), col: next.col }
        if (nextStart.lineId === range.startLineId && nextStart.col === range.startCol) {
          throw new Error('history copy continuation did not advance')
        }
        range = { ...request.range, startLineId: nextStart.lineId, startCol: nextStart.col }
      }
    },
    async search(request, options) {
      const command = create(HistorySearchCommandSchema, {
        terminal: terminal(request.terminalId),
        token: request.token,
        historyGeneration: BigInt(request.generation ?? 0),
        query: request.query,
        mode: request.mode === 'glob'
          ? HistorySearchMode.GLOB
          : request.mode === 'regex'
            ? HistorySearchMode.REGEX
            : HistorySearchMode.TEXT,
        direction: request.direction === 'backward' ? HistorySearchDirection.BACKWARD : HistorySearchDirection.FORWARD,
        cols: request.cols,
        limit: request.limit,
        start: request.start ? create(HistoryTextPositionSchema, { lineId: BigInt(request.start.lineId), col: request.start.col }) : undefined,
      })
      const result = await session.execute(
        create(CommandEnvelopeSchema, { command: { case: 'historySearch', value: command } }),
        options,
      )
      if (result.result.case !== 'historySearch') throw new Error('history search returned no result')
      return coreV2HistorySearchFromAPI(result.result.value)
    },
    async scan(request, options) {
      const command = create(HistorySearchCommandSchema, {
        terminal: terminal(request.terminalId),
        token: request.token,
        historyGeneration: BigInt(request.generation ?? 0),
        query: request.query,
        mode: request.mode === 'glob'
          ? HistorySearchMode.GLOB
          : request.mode === 'regex'
            ? HistorySearchMode.REGEX
            : HistorySearchMode.TEXT,
        direction: HistorySearchDirection.FORWARD,
        cols: request.cols,
        limit: 1,
        start: request.start ? create(HistoryTextPositionSchema, { lineId: BigInt(request.start.lineId), col: request.start.col }) : undefined,
        scan: true,
        maxMatches: request.maxMatches,
      })
      const result = await session.execute(
        create(CommandEnvelopeSchema, { command: { case: 'historySearch', value: command } }),
        options,
      )
      if (result.result.case !== 'historySearch') throw new Error('history search scan returned no result')
      return coreV2HistorySearchScanFromAPI(result.result.value)
    },
    async release(request, options) {
      await session.execute(
        create(CommandEnvelopeSchema, {
          command: {
            case: 'historyRelease',
            value: create(HistoryReleaseCommandSchema, {
              terminal: terminal(request.terminalId),
              token: request.token,
              historyGeneration: BigInt(request.generation ?? 0),
            }),
          },
        }),
        options,
      )
    },
  }
}

function protoHistoryMode(mode: CoreV2HistoryWindowRequest['mode']): HistoryWindowMode {
  switch (mode) {
    case 'latest': return HistoryWindowMode.LATEST
    case 'older': return HistoryWindowMode.OLDER
    case 'newer': return HistoryWindowMode.NEWER
    case 'oldest': return HistoryWindowMode.OLDEST
    case 'range': return HistoryWindowMode.LATEST
  }
}
