import { create } from '@bufbuild/protobuf'
import { describe, expect, it, vi } from 'vitest'
import { StorageEntrySchema, StoragePutResultSchema } from '../generated/apipb/storage_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { bookmarkLabel, createPathBookmarkApi, createPersistentPathBookmarkApi } from './pathBookmarks'

describe('createPathBookmarkApi generated Proto storage', () => {
  it('stores a normalized bookmark through storage.put', async () => {
    vi.spyOn(globalThis.crypto, 'randomUUID').mockReturnValue('11111111-1111-4111-8111-111111111111')
    const session = new MockProtoSession('machine-local', (command) => {
      if (command.command.case !== 'storagePut') throw new Error('unexpected command')
      return protoResult('storagePut', create(StoragePutResultSchema, {
        entry: create(StorageEntrySchema, { key: command.command.value.key, value: command.command.value.value, version: 7n }),
      }))
    })

    await expect(createPathBookmarkApi(session).add('/srv/app/', 'prod')).resolves.toMatchObject({ path: '/srv/app', label: 'prod', version: 7 })
    expect(session.commands[0]?.command).toMatchObject({ case: 'storagePut', value: { key: { appId: 'anytty.paths' } } })
  })

  it('keeps per-machine bookmarks in client storage across API instances', async () => {
    const values = new Map<string, string>()
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => { values.set(key, value) },
      removeItem: (key: string) => { values.delete(key) },
    }
    await createPersistentPathBookmarkApi('machine-a', undefined, storage).add('/srv/app/')
    expect(await createPersistentPathBookmarkApi('machine-a', undefined, storage).list()).toMatchObject([{ path: '/srv/app', label: 'app' }])
    expect(await createPersistentPathBookmarkApi('machine-b', undefined, storage).list()).toEqual([])
  })

  it('derives compact labels', () => {
    expect(bookmarkLabel('/')).toBe('/')
    expect(bookmarkLabel('/srv/app/')).toBe('app')
  })
})
