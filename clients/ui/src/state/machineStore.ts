import { MACHINE_ICON_NAMES, type AppMachineRecord, type AppMachineState, type AppMachineSource, type MachineAccessClass, type MachineIconName } from './appMachine'
import type { ConnectionPath } from '../core/transport'
import { normalizeHubBaseUrlCandidate } from '../api/hubUrl'

export interface StoredMachineRecord extends AppMachineRecord {
  addresses: StoredMachineAddresses
  endpoints: StoredMachineEndpoints
  addedAt: string
  updatedAt: string
}

export interface StoredMachineAddresses {
  local: string[]
  lan: string[]
  public: string[]
}

export interface StoredMachineEndpoints {
  webControl?: string | undefined
  hub?: string | undefined
}

export interface MachineStore {
  listMachines(): StoredMachineRecord[]
  getMachine(machineId: string): StoredMachineRecord | null
  saveMachine(record: StoredMachineRecord): StoredMachineRecord
  forgetMachine(machineId: string): void
}

export interface MachineStoreOptions {
  storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>
  now?: (() => Date) | undefined
}

const storeKey = 'anytty.app.machines.v2'
const maximumMachineIconImageLength = 96 * 1024

export function createMachineStore(options: MachineStoreOptions): MachineStore {
  return {
    listMachines() {
      return readMachines(options.storage)
    },
    getMachine(machineId) {
      return readMachines(options.storage).find((record) => record.machineId === machineId) ?? null
    },
    saveMachine(record) {
      rejectPrivateKeyMaterial(record)
      const clean = normalizeStoredMachine(record)
      const machines = upsertMachine(readMachines(options.storage), clean)
      writeMachines(options.storage, machines)
      return clean
    },
    forgetMachine(machineId) {
      const machines = readMachines(options.storage).filter((record) => record.machineId !== machineId)
      if (machines.length === 0) {
        options.storage.removeItem(storeKey)
        return
      }
      writeMachines(options.storage, machines)
    },
  }
}

function readMachines(storage: Pick<Storage, 'getItem'>): StoredMachineRecord[] {
  const raw = storage.getItem(storeKey)
  if (!raw) return []
  const parsed = JSON.parse(raw)
  rejectPrivateKeyMaterial(parsed)
  if (!Array.isArray(parsed)) {
    throw new Error('stored machines must be an array')
  }
  return parsed.map((record) => normalizeStoredMachine(recordValue(record, 'stored machine')))
}

function writeMachines(storage: Pick<Storage, 'setItem'>, machines: StoredMachineRecord[]): void {
  rejectPrivateKeyMaterial(machines)
  storage.setItem(storeKey, JSON.stringify(machines))
}

function upsertMachine(machines: StoredMachineRecord[], record: StoredMachineRecord): StoredMachineRecord[] {
  const index = machines.findIndex((item) => item.machineId === record.machineId)
  if (index === -1) {
    return [...machines, record]
  }
  const next = [...machines]
  next[index] = record
  return next
}

function normalizeStoredMachine(value: Record<string, unknown> | StoredMachineRecord): StoredMachineRecord {
  const record = value as Record<string, unknown>
  const machineId = stringField(record, 'machineId')
  const name = stringField(record, 'name')
  const state = machineState(record.state)
  const terminalCount = numberField(record, 'terminalCount')
  const source = machineSource(record.source)
  const addresses = addressesField(record.addresses)
  const endpoints = endpointsField(record.endpoints)
  return {
    machineId,
    name,
    ...(optionalString(record.alias) ? { alias: optionalString(record.alias) } : {}),
    ...(machineIconName(record.icon) ? { icon: machineIconName(record.icon) } : {}),
    ...(machineIconImage(record.iconImage) ? { iconImage: machineIconImage(record.iconImage) } : {}),
    ...(optionalString(record.hostname) ? { hostname: optionalString(record.hostname) } : {}),
    ...(optionalString(record.osInfo) ? { osInfo: optionalString(record.osInfo) } : {}),
    ...(optionalString(record.hubId) ? { hubId: optionalString(record.hubId) } : {}),
    state,
    terminalCount,
    ...(optionalString(record.lastSeenAt) ? { lastSeenAt: optionalString(record.lastSeenAt) } : {}),
    ...(connectionPathOrUndefined(record.lastConnectionPath)
      ? { lastConnectionPath: connectionPathOrUndefined(record.lastConnectionPath) }
      : {}),
    ...(connectionPathOrUndefined(record.preferredPath)
      ? { preferredPath: connectionPathOrUndefined(record.preferredPath) }
      : {}),
    ...(typeof record.relayInUse === 'boolean' ? { relayInUse: record.relayInUse } : {}),
    source,
	accessClass: machineAccessClass(record.accessClass),
    addresses,
    endpoints,
    addedAt: stringField(record, 'addedAt'),
    updatedAt: stringField(record, 'updatedAt'),
  }
}

function addressesField(value: unknown): StoredMachineAddresses {
  const record = recordValue(value, 'stored machine addresses')
  return {
    local: stringArray(record.local),
    lan: stringArray(record.lan),
    public: stringArray(record.public),
  }
}

function endpointsField(value: unknown): StoredMachineEndpoints {
  const record = recordValue(value, 'stored machine endpoints')
  const hub = normalizeHubBaseUrlCandidate(optionalString(record.hub))
  return {
    ...(optionalString(record.webControl) ? { webControl: optionalString(record.webControl) } : {}),
    ...(hub ? { hub } : {}),
  }
}

function rejectPrivateKeyMaterial(value: unknown): void {
  const stack: unknown[] = [value]
  while (stack.length > 0) {
    const current = stack.pop()
    if (Array.isArray(current)) {
      stack.push(...current)
      continue
    }
    if (!current || typeof current !== 'object') continue
    if (looksLikePrivateJwk(current)) {
      throw new Error('private key material must not be stored in MachineStore')
    }
    for (const [key, child] of Object.entries(current)) {
      const normalized = key.replace(/[-_\s]/g, '').toLowerCase()
      if (normalized.includes('machineprivatekey')) {
        throw new Error('machine private key material must not be stored in MachineStore')
      }
      if (normalized.includes('appprivatekey')) {
        throw new Error('app private key material must not be stored in MachineStore')
      }
      if (normalized === 'privatekey' || normalized.endsWith('privatekey')) {
        throw new Error('private key material must not be stored in MachineStore')
      }
      if (typeof child === 'string' && /BEGIN [A-Z ]*PRIVATE KEY/i.test(child)) {
        throw new Error('private key material must not be stored in MachineStore')
      }
      stack.push(child)
    }
  }
}

function looksLikePrivateJwk(value: object): boolean {
  const record = value as Record<string, unknown>
  if (typeof record.d !== 'string') return false
  return typeof record.kty === 'string' ||
    typeof record.crv === 'string' ||
    typeof record.x === 'string' ||
    typeof record.n === 'string'
}

function machineState(value: unknown): AppMachineState {
  if (value === 'online' || value === 'offline' || value === 'stale' || value === 'unknown' || value === 'connecting') {
    return value
  }
  throw new Error(`invalid machine state ${String(value)}`)
}

function machineSource(value: unknown): AppMachineSource {
  if (value === 'local' || value === 'hub' || value === 'manual') return value
  throw new Error(`invalid machine source ${String(value)}`)
}

function machineAccessClass(value: unknown): MachineAccessClass {
  if (value === 'local' || value === 'cloud' || value === 'local_cloud') return value
  throw new Error(`invalid machine access class ${String(value)}`)
}

function machineIconName(value: unknown): MachineIconName | undefined {
  return typeof value === 'string' && (MACHINE_ICON_NAMES as readonly string[]).includes(value)
    ? value as MachineIconName
    : undefined
}

function machineIconImage(value: unknown): string | undefined {
  if (typeof value !== 'string' || value.length > maximumMachineIconImageLength) return undefined
  return /^data:image\/webp;base64,[a-z0-9+/]+=*$/i.test(value) ? value : undefined
}

function connectionPathOrUndefined(value: unknown): ConnectionPath | undefined {
  if (value === undefined || value === null) return undefined
  if (value === 'local' || value === 'hub') return value
  throw new Error(`invalid connection path ${String(value)}`)
}

function recordValue(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} must be an object`)
  }
  return value as Record<string, unknown>
}

function stringField(record: Record<string, unknown>, key: string): string {
  const value = optionalString(record[key])
  if (!value) {
    throw new Error(`stored machine ${key} is required`)
  }
  return value
}

function numberField(record: Record<string, unknown>, key: string): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(`stored machine ${key} must be a number`)
  }
  return value
}

function optionalString(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed === '' ? undefined : trimmed
}

function stringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value
    .map((item) => typeof item === 'string' ? normalizeHubBaseUrlCandidate(item) : undefined)
    .filter((item): item is string => typeof item === 'string' && item !== '')
}
