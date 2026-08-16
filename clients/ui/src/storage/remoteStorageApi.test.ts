import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import {
  StorageDeleteResultSchema, StorageEntrySchema, StorageGetResultSchema, StorageKeySchema,
  StorageListResultSchema, StoragePutResultSchema, StorageScope,
} from '../generated/apipb/storage_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { createRemoteStorageApi, storageText } from './remoteStorageApi'

describe('createRemoteStorageApi generated Proto API', () => {
  it('routes CRUD through typed storage commands', async () => {
    const key = create(StorageKeySchema, { appId: 'anytty.paths', scope: StorageScope.PUBLIC, key: 'bookmarks/project' })
    const entry = (value: Uint8Array) => create(StorageEntrySchema, { key, value, version: 3n })
    const session = new MockProtoSession('machine-local', (command) => {
      switch (command.command.case) {
        case 'storagePut': return protoResult('storagePut', create(StoragePutResultSchema, { entry: entry(command.command.value.value) }))
        case 'storageGet': return protoResult('storageGet', create(StorageGetResultSchema, { entry: entry(new TextEncoder().encode('hello')) }))
        case 'storageList': return protoResult('storageList', create(StorageListResultSchema, { entries: [entry(new TextEncoder().encode('hello'))] }))
        case 'storageDelete': return protoResult('storageDelete', create(StorageDeleteResultSchema, { key, deleted: true, version: 4n }))
        default: throw new Error(`unexpected command ${command.command.case}`)
      }
    })
    const api = createRemoteStorageApi(session)

    expect(storageText(await api.put({ appId: 'anytty.paths', key: 'bookmarks/project', value: 'hello', expectedVersion: 2 }))).toBe('hello')
    expect(storageText(await api.get({ appId: 'anytty.paths', key: 'bookmarks/project' }))).toBe('hello')
    await expect(api.list({ appId: 'anytty.paths', prefix: 'bookmarks/' })).resolves.toHaveLength(1)
    await expect(api.delete({ appId: 'anytty.paths', key: 'bookmarks/project' })).resolves.toEqual({ deleted: true, version: 4 })
    expect(session.commands.map((command) => command.command.case)).toEqual(['storagePut', 'storageGet', 'storageList', 'storageDelete'])
    expect(session.commands[0]?.command.value).toMatchObject({ version: { checkVersion: true, expectedVersion: 2n } })
  })
})
