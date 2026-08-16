import { create } from '@bufbuild/protobuf'
import type { ProtoClientSession } from '../core/protoClientSession'
import { CommandEnvelopeSchema } from '../generated/apipb/application_pb'
import {
  StorageDeleteCommandSchema,
  StorageGetCommandSchema,
  StorageKeySchema,
  StorageListCommandSchema,
  StoragePutCommandSchema,
  StorageScope,
  StorageVersionFenceSchema,
  type StorageEntry,
} from '../generated/apipb/storage_pb'

export type RemoteStorageScope = 'public' | 'private'

export interface RemoteStorageEntry {
  appId: string
  scope: RemoteStorageScope | string
  ownerId: string
  key: string
  value: Uint8Array
  version: number
  updatedAt?: string | undefined
}

export interface RemoteStorageApi {
  get(input: RemoteStorageKey): Promise<RemoteStorageEntry>
  put(input: RemoteStoragePutInput): Promise<RemoteStorageEntry>
  delete(input: RemoteStorageDeleteInput): Promise<{ deleted: boolean; version: number }>
  list(input: RemoteStorageListInput): Promise<RemoteStorageEntry[]>
}

export interface RemoteStorageKey {
  appId: string
  scope?: RemoteStorageScope | string | undefined
  ownerId?: string | undefined
  key: string
}

export interface RemoteStoragePutInput extends RemoteStorageKey {
  value: Uint8Array | string
  expectedVersion?: number | undefined
}

export interface RemoteStorageDeleteInput extends RemoteStorageKey {
  expectedVersion?: number | undefined
}

export interface RemoteStorageListInput {
  appId: string
  scope?: RemoteStorageScope | string | undefined
  ownerId?: string | undefined
  prefix?: string | undefined
}

export function createRemoteStorageApi(session: ProtoClientSession): RemoteStorageApi {
  return createProtoRemoteStorageApi(session)
}

function createProtoRemoteStorageApi(session: ProtoClientSession): RemoteStorageApi {
  const execute = (caseName: string, value: object) => session.execute(
    create(CommandEnvelopeSchema, { command: { case: caseName, value } } as never),
  )
  return {
    async get(input) {
      const result = await execute('storageGet', create(StorageGetCommandSchema, { key: protoStorageKey(input) }))
      if (result.result.case !== 'storageGet' || !result.result.value.entry) throw new Error('storage get returned no entry')
      return protoStorageEntry(result.result.value.entry)
    },
    async put(input) {
      const result = await execute('storagePut', create(StoragePutCommandSchema, {
        key: protoStorageKey(input),
        value: storageValue(input.value),
        version: protoVersionFence(input.expectedVersion),
      }))
      if (result.result.case !== 'storagePut' || !result.result.value.entry) throw new Error('storage put returned no entry')
      return protoStorageEntry(result.result.value.entry)
    },
    async delete(input) {
      const result = await execute('storageDelete', create(StorageDeleteCommandSchema, {
        key: protoStorageKey(input),
        version: protoVersionFence(input.expectedVersion),
      }))
      if (result.result.case !== 'storageDelete') throw new Error('storage delete returned no result')
      return { deleted: result.result.value.deleted, version: Number(result.result.value.version) }
    },
    async list(input) {
      const result = await execute('storageList', create(StorageListCommandSchema, {
        appId: input.appId,
        scope: protoStorageScope(input.scope),
        ownerId: input.ownerId ?? '',
        prefix: input.prefix ?? '',
      }))
      if (result.result.case !== 'storageList') throw new Error('storage list returned no result')
      return result.result.value.entries.map(protoStorageEntry)
    },
  }
}

function protoStorageKey(input: RemoteStorageKey) {
  return create(StorageKeySchema, {
    appId: input.appId,
    scope: protoStorageScope(input.scope),
    ownerId: input.ownerId ?? '',
    key: input.key,
  })
}

function protoStorageScope(scope: string | undefined): StorageScope {
  return scope === 'private' ? StorageScope.PRIVATE : StorageScope.PUBLIC
}

function protoVersionFence(expectedVersion: number | undefined) {
  return create(StorageVersionFenceSchema, {
    checkVersion: typeof expectedVersion === 'number',
    expectedVersion: BigInt(expectedVersion ?? 0),
  })
}

function protoStorageEntry(entry: StorageEntry): RemoteStorageEntry {
  const key = entry.key
  if (!key) throw new Error('storage entry key is missing')
  return {
    appId: key.appId,
    scope: key.scope === StorageScope.PRIVATE ? 'private' : 'public',
    ownerId: key.ownerId,
    key: key.key,
    value: entry.value.slice(),
    version: Number(entry.version),
    updatedAt: entry.updatedAtUnixNano > 0n ? new Date(Number(entry.updatedAtUnixNano / 1_000_000n)).toISOString() : undefined,
  }
}

export function storageText(entry: Pick<RemoteStorageEntry, 'value'>): string {
  return new TextDecoder().decode(entry.value)
}

function storageValue(value: Uint8Array | string): Uint8Array {
  return typeof value === 'string' ? new TextEncoder().encode(value) : value
}
