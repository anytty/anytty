import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import { StorageEntrySchema, StorageKeySchema, StorageListResultSchema, StorageScope } from '../generated/apipb/storage_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { createRemoteClipboardApi } from './clipboardApi'

describe('createRemoteClipboardApi generated Proto storage', () => {
  it('projects clipboard history from typed storage entries', async () => {
    const value = new TextEncoder().encode(JSON.stringify({
      schema_version: 1, id: 'clip-1', text: 'hello', preview: 'hello', source_app: 'tui', created_at: '2026-05-16T08:00:00Z',
    }))
    const session = new MockProtoSession('machine-local', () => protoResult('storageList', create(StorageListResultSchema, {
      entries: [create(StorageEntrySchema, {
        key: create(StorageKeySchema, { appId: 'anytty.clipboard', scope: StorageScope.PUBLIC, key: 'history/clip-1' }),
        value, version: 5n,
      })],
    })))

    await expect(createRemoteClipboardApi(session).list()).resolves.toMatchObject([{ id: 'clip-1', text: 'hello', version: 5 }])
    expect(session.commands[0]?.command).toMatchObject({ case: 'storageList', value: { appId: 'anytty.clipboard', prefix: 'history/' } })
  })
})
