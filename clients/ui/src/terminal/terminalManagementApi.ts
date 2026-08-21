import { create } from '@bufbuild/protobuf'
import type { ProtoClientSession } from '../core/protoClientSession'
import type { LocalCreateTerminalInput, LocalUpdateTerminalInput } from '../core/transport'
import { CommandEnvelopeSchema } from '../generated/apipb/application_pb'
import {
  TerminalCreateCommandSchema,
  TerminalCreateSpecSchema,
  TerminalDefaultsCommandSchema,
  TerminalGetCommandSchema,
  TerminalRefSchema,
  TerminalRemoveCommandSchema,
  TerminalRestartCommandSchema,
  TerminalSetMetadataCommandSchema,
  TerminalSetTagsCommandSchema,
  TerminalSizeSchema,
} from '../generated/apipb/terminal_pb'

export interface TerminalManagementApi {
  /** getDefaults 从 owning daemon 查询当前账号的 shell/home，不读取手机进程环境。 */
  getDefaults(): Promise<{ command: string[]; cwd: string }>
  createTerminal(input: LocalCreateTerminalInput): Promise<{ terminalId: string }>
  updateTerminal(input: LocalUpdateTerminalInput): Promise<void>
  updateTerminalSizeLock(input: Pick<LocalUpdateTerminalInput, 'terminalId' | 'cwd' | 'sizeLockMode'>): Promise<void>
  restartTerminal(terminalId: string): Promise<void>
  deleteTerminal(terminalId: string): Promise<void>
  getTerminalDirectory(terminalId: string): Promise<{ path: string; source?: string | undefined }>
}

export function createTerminalManagementApi(
  session: ProtoClientSession,
  machineId: string,
): TerminalManagementApi {
  return createProtoTerminalManagementApi(session, machineId)
}

function createProtoTerminalManagementApi(session: ProtoClientSession, machineId: string): TerminalManagementApi {
  if (session.stamp.endpointId !== machineId) {
    throw new Error(`terminal management session machine mismatch: connected to ${session.stamp.endpointId}, expected ${machineId}`)
  }
  const terminalRef = (terminalId: string) => create(TerminalRefSchema, { endpointId: machineId, terminalId })
  const execute = (caseName: Parameters<typeof protoCommand>[0], value: Parameters<typeof protoCommand>[1]) =>
    session.execute(protoCommand(caseName, value))
  return {
    async getDefaults() {
      const result = await execute('terminalDefaults', create(TerminalDefaultsCommandSchema))
      if (result.result.case !== 'terminalDefaults' || !result.result.value.defaults) {
        throw new Error('terminal defaults returned no defaults')
      }
      return {
        command: [...result.result.value.defaults.defaultCommand],
        cwd: result.result.value.defaults.defaultCwd,
      }
    },
    async createTerminal(input) {
      const terminal = create(TerminalCreateSpecSchema, {
        terminalId: newTerminalId(),
        name: input.name ?? '',
        command: input.command ?? [],
        size: create(TerminalSizeSchema, {
          cols: finiteInt(input.cols) || 80,
          rows: finiteInt(input.rows) || 24,
        }),
        cwd: input.cwd ?? '',
        env: input.environment ?? [],
        scrollbackRows: finiteInt(input.scrollbackSize),
        scrollbackMaxBytes: BigInt(finiteInt(input.scrollbackMaxBytes)),
        scrollbackMaxAgeSeconds: BigInt(finiteInt(input.scrollbackMaxAgeSeconds)),
        tags: terminalTags(input),
      })
      const result = await execute('terminalCreate', create(TerminalCreateCommandSchema, { terminal }))
      if (result.result.case !== 'terminalCreate' || !result.result.value.terminal?.ref?.terminalId) {
        throw new Error('terminal create returned no terminal reference')
      }
      return { terminalId: result.result.value.terminal.ref.terminalId }
    },
    async updateTerminal(input) {
      const result = await execute('terminalSetMetadata', create(TerminalSetMetadataCommandSchema, {
        terminal: terminalRef(input.terminalId),
        name: input.name ?? '',
        tags: terminalTags(input),
      }))
      assertAcknowledge(result, 'terminal metadata update')
    },
    async updateTerminalSizeLock(input) {
      const result = await execute('terminalSetTags', create(TerminalSetTagsCommandSchema, {
        terminal: terminalRef(input.terminalId),
        tags: terminalTags(input),
      }))
      assertAcknowledge(result, 'terminal size lock metadata update')
    },
    async restartTerminal(terminalId) {
      assertAcknowledge(await execute('terminalRestart', create(TerminalRestartCommandSchema, { terminal: terminalRef(terminalId) })), 'terminal restart')
    },
    async deleteTerminal(terminalId) {
      assertAcknowledge(await execute('terminalRemove', create(TerminalRemoveCommandSchema, { terminal: terminalRef(terminalId) })), 'terminal remove')
    },
    async getTerminalDirectory(terminalId) {
      const result = await execute('terminalGet', create(TerminalGetCommandSchema, { terminal: terminalRef(terminalId) }))
      if (result.result.case !== 'terminalGet' || !result.result.value.terminal) throw new Error('terminal get returned no terminal')
      const terminal = result.result.value.terminal
      return {
        path: terminal.liveCwd || terminal.cwd || '',
        source: terminal.liveCwd ? 'live' : terminal.cwd ? 'metadata' : undefined,
      }
    },
  }
}

function protoCommand(caseName: string, value: object) {
  return create(CommandEnvelopeSchema, { command: { case: caseName, value } } as never)
}

function assertAcknowledge(result: Awaited<ReturnType<ProtoClientSession['execute']>>, operation: string): void {
  if (result.result.case !== 'acknowledge') throw new Error(`${operation} was not acknowledged`)
}

function finiteInt(value: number | undefined): number {
  return typeof value === 'number' && Number.isFinite(value) ? Math.max(0, Math.floor(value)) : 0
}

function newTerminalId(): string {
  const randomUUID = globalThis.crypto?.randomUUID?.bind(globalThis.crypto)
  if (randomUUID) return `term-${randomUUID()}`
  const bytes = new Uint8Array(16)
  globalThis.crypto.getRandomValues(bytes)
  return `term-${Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('')}`
}

function terminalTags(input: {
  cwd?: string | undefined
  sizeLockMode?: LocalCreateTerminalInput['sizeLockMode'] | LocalUpdateTerminalInput['sizeLockMode']
}): Record<string, string> {
  const tags: Record<string, string> = {}
  if (input.sizeLockMode) {
    tags['anytty.size_lock'] = input.sizeLockMode
  }
  if (input.cwd) {
    tags.cwd = input.cwd
  }
  return tags
}
