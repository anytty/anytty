export type MachineState = 'online' | 'offline' | 'unknown'
export type TerminalState = 'running' | 'exited' | 'unknown'

export interface LocalRTCInfo {
  signalingUrl: string
  iceTCPUrl?: string | undefined
}

export interface Machine {
  machineId: string
  name: string
  state: MachineState
  terminalCount?: number | undefined
  lastSeenAt?: string | undefined
  localRTC?: LocalRTCInfo | undefined
}

export interface Terminal {
  terminalId: string
  machineId: string
  title: string
  state: TerminalState
  command?: string | undefined
  cols?: number | undefined
  rows?: number | undefined
  cwd?: string | undefined
  lastActiveAt?: string | undefined
  foregroundProcess?: string | undefined
  lastOutputAt?: string | undefined
  sizeLocked?: boolean | undefined
  sizeLockMode?: 'off' | 'warn' | 'lock' | undefined
  environment?: string | undefined
  tags?: Record<string, string> | undefined
  resizeOwnerSurfaceId?: string | undefined
  resizeOwnerViewId?: string | undefined
  resizeOwnerAttachmentCount?: number | undefined
}

const blockedPublicModelKeys = [
  'workspace',
  'workspace_id',
  'workspaceId',
  'workspaces',
  'tab',
  'tab_id',
  'tabId',
  'tabs',
  'window',
  'window_id',
  'windowId',
  'windows',
  'pane',
  'pane_id',
  'paneId',
  'panes',
] as const

export function assertRemoteModelShape(value: unknown): void {
  const seen = new Set<unknown>()

  function walk(current: unknown): void {
    if (current === null || current === undefined) return
    if (typeof current !== 'object') return
    if (seen.has(current)) return
    seen.add(current)

    if (Array.isArray(current)) {
      for (const item of current) walk(item)
      return
    }

    for (const [key, nested] of Object.entries(current as Record<string, unknown>)) {
      if (key === 'tags') continue
      if ((blockedPublicModelKeys as readonly string[]).includes(key)) {
        throw new Error(`remote public model must not contain ${key}`)
      }
      walk(nested)
    }
  }

  walk(value)
}

export function normalizeMachine(input: Record<string, unknown>): Machine {
  assertRemoteModelShape(input)

  const machineId = getRequiredString(input, 'machine_id', 'machineId')
  return cleanOptional<Machine>({
    machineId,
    name: getOptionalString(input, 'name') ?? machineId,
    state: normalizeMachineState(getOptionalString(input, 'state')),
    terminalCount: getOptionalNumber(input, 'terminal_count', 'terminalCount'),
    lastSeenAt: getOptionalString(input, 'last_seen_at', 'lastSeenAt'),
    localRTC: normalizeLocalRTC(input.local_rtc ?? input.localRTC),
  })
}

export function normalizeTerminal(input: Record<string, unknown>): Terminal {
  assertRemoteModelShape(input)

  const terminalId = getRequiredString(input, 'terminal_id', 'terminalId')
  return cleanOptional<Terminal>({
    terminalId,
    machineId: getRequiredString(input, 'machine_id', 'machineId'),
    title: getOptionalString(input, 'title', 'name') ?? terminalId,
    state: normalizeTerminalState(getOptionalString(input, 'state')),
    command: getOptionalCommand(input, 'command'),
    cols: getOptionalNumber(input, 'cols'),
    rows: getOptionalNumber(input, 'rows'),
    cwd: getOptionalString(input, 'cwd'),
    lastActiveAt: getOptionalString(input, 'last_active_at', 'lastActiveAt'),
    foregroundProcess: getOptionalString(input, 'foreground_process', 'foregroundProcess'),
    lastOutputAt: getOptionalString(input, 'last_output_at', 'lastOutputAt'),
    sizeLocked: getOptionalBoolean(input, 'size_locked', 'sizeLocked'),
    sizeLockMode: normalizeSizeLockMode(getOptionalString(input, 'size_lock_mode', 'sizeLockMode')),
    environment: getOptionalString(input, 'environment', 'env'),
    tags: getOptionalStringRecord(input, 'tags'),
    resizeOwnerSurfaceId: getResizeOwnershipString(input, 'owner_surface_id'),
    resizeOwnerViewId: getResizeOwnershipString(input, 'owner_view_id'),
    resizeOwnerAttachmentCount: getOptionalNumber(input, 'resize_owner_attachment_count', 'resizeOwnerAttachmentCount'),
  })
}

function normalizeLocalRTC(value: unknown): LocalRTCInfo | undefined {
  if (value === null || value === undefined) return undefined
  if (typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('local_rtc must be an object')
  }
  const record = value as Record<string, unknown>
  const signalingUrl = getRequiredString(record, 'signaling_url', 'signalingUrl')
  return cleanOptional<LocalRTCInfo>({
    signalingUrl,
    iceTCPUrl: getOptionalString(record, 'ice_tcp_url', 'iceTCPUrl'),
  })
}

function normalizeMachineState(value: string | undefined): MachineState {
  if (value === 'online' || value === 'offline' || value === 'unknown') return value
  return 'unknown'
}

function normalizeTerminalState(value: string | undefined): TerminalState {
  if (value === 'running' || value === 'exited' || value === 'unknown') return value
  return 'unknown'
}

function normalizeSizeLockMode(value: string | undefined): Terminal['sizeLockMode'] {
  if (value === 'off' || value === 'warn' || value === 'lock') return value
  return undefined
}

function getResizeOwnershipString(record: Record<string, unknown>, key: 'owner_surface_id' | 'owner_view_id'): string | undefined {
  const direct = getOptionalString(record, key)
  if (direct) return direct
  const ownership = record.resize_ownership
  if (typeof ownership !== 'object' || ownership === null || Array.isArray(ownership)) return undefined
  return getOptionalString(ownership as Record<string, unknown>, key)
}

function getRequiredString(record: Record<string, unknown>, ...keys: string[]): string {
  const value = getOptionalString(record, ...keys)
  if (!value) {
    throw new Error(`missing required string ${keys[0]}`)
  }
  return value
}

function getOptionalString(record: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record[key]
    if (value === undefined || value === null) continue
    if (typeof value !== 'string') {
      throw new Error(`${key} must be a string`)
    }
    return value
  }
  return undefined
}

function getOptionalCommand(record: Record<string, unknown>, ...keys: string[]): string | undefined {
  for (const key of keys) {
    const value = record[key]
    if (value === undefined || value === null) continue
    if (typeof value === 'string') return value
    if (Array.isArray(value) && value.every((part) => typeof part === 'string')) return value.join(' ')
    throw new Error(`${key} must be a string or string array`)
  }
  return undefined
}

function getOptionalNumber(record: Record<string, unknown>, ...keys: string[]): number | undefined {
  for (const key of keys) {
    const value = record[key]
    if (value === undefined || value === null) continue
    if (typeof value !== 'number' || !Number.isFinite(value)) {
      throw new Error(`${key} must be a finite number`)
    }
    return value
  }
  return undefined
}

function getOptionalBoolean(record: Record<string, unknown>, ...keys: string[]): boolean | undefined {
  for (const key of keys) {
    const value = record[key]
    if (value === undefined || value === null) continue
    if (typeof value !== 'boolean') {
      throw new Error(`${key} must be a boolean`)
    }
    return value
  }
  return undefined
}

function getOptionalStringRecord(record: Record<string, unknown>, ...keys: string[]): Record<string, string> | undefined {
  for (const key of keys) {
    const value = record[key]
    if (value === undefined || value === null) continue
    if (typeof value !== 'object' || Array.isArray(value)) {
      throw new Error(`${key} must be an object`)
    }
    const entries = Object.entries(value as Record<string, unknown>)
    if (!entries.every(([, entryValue]) => typeof entryValue === 'string')) {
      throw new Error(`${key} must contain only string values`)
    }
    return Object.fromEntries(entries) as Record<string, string>
  }
  return undefined
}

function cleanOptional<T extends object>(record: T): T {
  for (const key of Object.keys(record) as (keyof T)[]) {
    if (record[key] === undefined) {
      delete record[key]
    }
  }
  return record
}
