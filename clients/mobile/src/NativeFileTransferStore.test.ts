import { create, toBinary } from '@bufbuild/protobuf'
import { Capacitor } from '@capacitor/core'
import type { ProtoClientSession, ProtoResourceStream } from '../../ui/src/core/protoClientSession'
import { decodeFileTransferDataPayload, encodeFileTransferAckPayload, encodeFileTransferDataPayload, encodeFileTransferFinishPayload } from '../../ui/src/files/fileStreamProtocol'
import * as AnyTTYApiApplication from '../../ui/src/generated/apipb/application_pb'
import * as AnyTTYApiCommon from '../../ui/src/generated/apipb/common_pb'
import * as AnyTTYApiFile from '../../ui/src/generated/apipb/file_pb'
import * as AnyTTYClientBinding from '../../ui/src/generated/bindingpb/client_binding_pb'
import { ErrorEnvelopeSchema, FileTransferResultSchema, ProtocolErrorSchema } from '../../ui/src/generated/wirepb/terminal_pb'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { FileTransferStoreSnapshot } from './NativeFileTransferStore'
import NativeFilePicker from './plugins/nativeFilePicker'

vi.mock('./plugins/nativeFilePicker', () => ({
  default: {
    getDownloadResumeOffset: vi.fn(),
    appendDownloadPartial: vi.fn(),
    commitDownloadPartial: vi.fn(),
    discardDownloadPartial: vi.fn(),
    openUploadSource: vi.fn(),
    readUploadSource: vi.fn(),
    finishUploadSource: vi.fn(),
    closeUploadSource: vi.fn(),
    pickFiles: vi.fn(),
    saveFile: vi.fn(),
    openFile: vi.fn(),
  },
}))

vi.stubGlobal('self', globalThis)
const { NativeFileTransferStore } = await import('./NativeFileTransferStore')

describe('NativeFileTransferStore', () => {
  it('persists a user-paused download and keeps the supplied resume offset', async () => {
    vi.stubGlobal('self', globalThis)
    const storage = memoryStorage()
    const first = new NativeFileTransferStore(storage)
    first.startDownload('machine-a', 'archive.bin', 4096, '/tmp/archive.bin', 1024)
    await waitFor(() => first.getSnapshot().transfers[0]?.status === 'failed')
    const id = first.getSnapshot().transfers[0]!.id
    first.pauseTransfer(id)

    const restored = new NativeFileTransferStore(storage).getSnapshot().transfers[0]
    expect(restored).toMatchObject({
      id,
      machineId: 'machine-a',
      filePath: '/tmp/archive.bin',
      totalSize: 4096,
      transferredSize: 1024,
      status: 'paused',
      pausedByUser: true,
    })
  })

  it('marks interrupted work as user-paused when all background connections are stopped', async () => {
    const storage = memoryStorage()
    storage.setItem('anytty.file-transfers.v2', JSON.stringify([{
      id: 'pending-upload',
      machineId: 'studio',
      name: 'draft.bin',
      direction: 'upload',
      totalSize: 8,
      transferredSize: 0,
      status: 'pending',
      startedAt: 1,
      updatedAt: 2,
      localUri: 'content://draft',
      targetDir: '/tmp',
    }]))
    const store = new NativeFileTransferStore(storage)

    await store.suspendForUserStop()
    await store.resumeInterruptedTransfers()

    expect(store.getSnapshot().transfers[0]).toMatchObject({
      status: 'paused',
      pausedByUser: true,
      error: undefined,
    })
  })

  it('discards seeded recovery history and ignores late transfer work after reset', async () => {
    const storage = memoryStorage()
    storage.setItem('anytty.file-transfers.v2', JSON.stringify([{
      id: 'seeded-upload',
      machineId: 'studio',
      name: 'seeded.bin',
      direction: 'upload',
      totalSize: 8,
      transferredSize: 4,
      status: 'transferring',
      startedAt: 1,
      updatedAt: 2,
      localUri: 'content://seeded',
      targetDir: '/tmp',
    }]))
    const resolverEntered = deferred<void>()
    const sessionResult = deferred<ProtoClientSession>()
    const lateSessionClosed = deferred<void>()
    const stream: ProtoResourceStream = {
      handle: 1n,
      send: async () => undefined,
      subscribe: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      close: async () => undefined,
    }
    let lateSessionCloses = 0
    const lateSession: ProtoClientSession = {
      ...uploadSession(stream, 8, 4, 4),
      close: async () => {
        lateSessionCloses += 1
        lateSessionClosed.resolve()
      },
    }
    const store = new NativeFileTransferStore(storage)
    store.setSessionResolver(async () => {
      resolverEntered.resolve()
      return await sessionResult.promise
    })
    const snapshots: FileTransferStoreSnapshot[] = []
    store.subscribe(() => snapshots.push(store.getSnapshot()))

    const resume = store.resumeInterruptedTransfers()
    await resolverEntered.promise
    expect(store.getSnapshot().transfers[0]?.status).toBe('pending')
    await store.discardForLocalReset()
    await store.suspendForRuntimeReset()

    expect(store.getSnapshot()).toEqual({ transfers: [], hasActiveTransfers: false })
    expect(snapshots.at(-1)).toEqual({ transfers: [], hasActiveTransfers: false })
    expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
    const notificationsAfterReset = snapshots.length

    sessionResult.resolve(lateSession)
    await lateSessionClosed.promise
    await resume
    store.pauseTransfer('seeded-upload')
    store.cancelTransfer('seeded-upload')
    store.dismissTransfer('seeded-upload')

    expect(lateSessionCloses).toBe(1)
    expect(snapshots).toHaveLength(notificationsAfterReset)
    expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
  })

  it('commits a native download after FILE_FINISH even when the resource stream closes immediately', async () => {
    vi.spyOn(Capacitor, 'isNativePlatform').mockReturnValue(true)
    vi.mocked(NativeFilePicker.getDownloadResumeOffset).mockResolvedValue({ offset: 0 })
    vi.mocked(NativeFilePicker.appendDownloadPartial).mockResolvedValue({ offset: 4 })
    const commit = deferred<{ uri: string, path: string, bytes: number, sha256: string }>()
    vi.mocked(NativeFilePicker.commitDownloadPartial).mockImplementation(async () => await commit.promise)

    let frameListener: ((type: number, payload: Uint8Array) => void) | undefined
    let closeListener: ((error: Error) => void) | undefined
    const stream: ProtoResourceStream = {
      handle: 1n,
      send: async () => undefined,
      subscribe: (listener) => { frameListener = listener; return { close() {} } },
      subscribeClosed: (listener) => { closeListener = listener; return { close() {} } },
      close: async () => undefined,
    }
    const stamp = create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'direct', generation: 1n })
    const resource = create(AnyTTYApiCommon.ResourceHandleSchema, {
      kind: AnyTTYApiCommon.ResourceKind.FILE_TRANSFER,
      opaqueToken: new Uint8Array([1]),
      session: stamp,
    })
    const session: ProtoClientSession = {
      stamp,
      isAlive: () => true,
      close: async () => undefined,
      subscribeEvents: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      openResourceStream: async () => stream,
      execute: async (envelope) => {
        if (envelope.command.case === 'fileDownloadOpen') {
          return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
            result: {
              case: 'fileTransferOpen',
              value: create(AnyTTYApiFile.FileTransferOpenResultSchema, {
                transfer: create(AnyTTYApiFile.FileTransferHandleSchema, {
                  resource,
                  size: 4n,
                  chunkBytes: 4,
                  windowBytes: 4n,
                  modifiedAtUnixNano: 1n,
                }),
              }),
            },
          })
        }
        if (envelope.command.case === 'releaseResource') return create(AnyTTYApiApplication.ResultEnvelopeSchema)
        throw new Error(`unexpected command ${envelope.command.case}`)
      },
    }
    const store = new NativeFileTransferStore(null)
    store.setSessionResolver(async () => session)
    store.startDownload('studio', 'demo.bin', 4, '/tmp/demo.bin')
    await waitFor(() => frameListener !== undefined && closeListener !== undefined)

    frameListener!(AnyTTYClientBinding.ResourceStreamFrameType.FILE_DATA, encodeFileTransferDataPayload({ offset: 0, data: new Uint8Array([1, 2, 3, 4]) }))
    frameListener!(AnyTTYClientBinding.ResourceStreamFrameType.FILE_FINISH, encodeFileTransferFinishPayload({ size: 4, sha256: new Uint8Array(32) }))
    closeListener!(new Error('resource stream closed'))
    await waitFor(() => vi.mocked(NativeFilePicker.commitDownloadPartial).mock.calls.length === 1)
    commit.resolve({ uri: 'content://download', path: 'Downloads/AnyTTY/demo.bin', bytes: 4, sha256: '00' })

    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'completed')
    expect(store.getSnapshot().transfers[0]?.savedPath).toBe('Downloads/AnyTTY/demo.bin')
  })

  it('opens a completed download through the native system chooser', async () => {
    vi.spyOn(Capacitor, 'isNativePlatform').mockReturnValue(true)
    vi.mocked(NativeFilePicker.openFile).mockResolvedValue({ opened: true })
    const storage = memoryStorage()
    storage.setItem('anytty.file-transfers.v2', JSON.stringify([{
      id: 'completed-download',
      machineId: 'studio',
      name: 'report.pdf',
      direction: 'download',
      totalSize: 12,
      transferredSize: 12,
      status: 'completed',
      startedAt: 1,
      updatedAt: 2,
      savedUri: 'content://media/external/downloads/42',
    }]))
    const store = new NativeFileTransferStore(storage)

    await store.openDownloadedFile('completed-download')

    expect(NativeFilePicker.openFile).toHaveBeenCalledWith({
      uri: 'content://media/external/downloads/42',
      name: 'report.pdf',
    })
  })

  it('discards a stale native partial and reopens the download from zero', async () => {
    vi.spyOn(Capacitor, 'isNativePlatform').mockReturnValue(true)
    vi.mocked(NativeFilePicker.getDownloadResumeOffset).mockResolvedValue({ offset: 2 })
    vi.mocked(NativeFilePicker.discardDownloadPartial).mockResolvedValue({ discarded: true })
    const storage = memoryStorage()
    storage.setItem('anytty.file-transfers.v2', JSON.stringify([{
      id: 'restored-download',
      machineId: 'studio',
      name: 'demo.bin',
      direction: 'download',
      totalSize: 4,
      transferredSize: 2,
      status: 'failed',
      startedAt: 1,
      updatedAt: 1,
      filePath: '/tmp/demo.bin',
      remoteModifiedAtUnixNano: '1',
    }]))
    const openOffsets: bigint[] = []
    const stamp = create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'direct', generation: 1n })
    const resource = create(AnyTTYApiCommon.ResourceHandleSchema, {
      kind: AnyTTYApiCommon.ResourceKind.FILE_TRANSFER,
      opaqueToken: new Uint8Array([2]),
      session: stamp,
    })
    const stream: ProtoResourceStream = {
      handle: 1n,
      send: async () => undefined,
      subscribe: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      close: async () => undefined,
    }
    const session: ProtoClientSession = {
      stamp,
      isAlive: () => true,
      close: async () => undefined,
      subscribeEvents: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      openResourceStream: async () => stream,
      execute: async (envelope) => {
        if (envelope.command.case === 'fileDownloadOpen') {
          openOffsets.push(envelope.command.value.offset)
          if (openOffsets.length === 1) throw new Error('stale download source')
          return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
            result: {
              case: 'fileTransferOpen',
              value: create(AnyTTYApiFile.FileTransferOpenResultSchema, {
                transfer: create(AnyTTYApiFile.FileTransferHandleSchema, {
                  resource,
                  size: 4n,
                  chunkBytes: 4,
                  windowBytes: 4n,
                  modifiedAtUnixNano: 2n,
                }),
              }),
            },
          })
        }
        if (envelope.command.case === 'releaseResource') return create(AnyTTYApiApplication.ResultEnvelopeSchema)
        throw new Error(`unexpected command ${envelope.command.case}`)
      },
    }
    const store = new NativeFileTransferStore(storage)
    store.setSessionResolver(async () => session)
    store.startDownload('studio', 'demo.bin', 4, '/tmp/demo.bin', 2)

    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'transferring')
    expect(openOffsets).toEqual([2n, 0n])
    expect(NativeFilePicker.discardDownloadPartial).toHaveBeenCalledOnce()
    store.pauseTransfer('restored-download')
  })

  it('keeps machine snapshots stable until transfer state changes', async () => {
    vi.stubGlobal('self', globalThis)
    const store = new NativeFileTransferStore()
    const initial = store.getSnapshot('machine-a')

    expect(store.getSnapshot('machine-a')).toBe(initial)
    expect(store.getSnapshot('machine-b')).not.toBe(initial)

    store.startDownload('machine-a', 'missing.txt', 1, '/missing.txt')
    const changed = store.getSnapshot('machine-a')
    expect(changed).not.toBe(initial)
    expect(changed.transfers).toHaveLength(1)
    expect(store.getSnapshot('machine-a')).toBe(changed)
    expect(store.getSnapshot('machine-b').transfers).toHaveLength(0)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('waits for pause detach and serializes duplicate resume requests', async () => {
    const detach = deferred<void>()
    const detachStarted = deferred<void>()
    const harness = await createPausedUpload(async () => {
      detachStarted.resolve()
      await detach.promise
    })
    const firstResume = harness.store.resumeTransfer(harness.id)
    const duplicateResume = harness.store.resumeTransfer(harness.id)
    await detachStarted.promise
    expect(harness.openCommands).toHaveLength(1)
    detach.resolve()
    await Promise.all([firstResume, duplicateResume])
    expect(harness.openCommands).toHaveLength(2)
    expect(harness.openCommands[1]?.resume?.opaqueToken).toEqual(harness.resumeToken)
  })

  it('keeps a user pause when a runtime reset overlaps detach cleanup', async () => {
    const detach = deferred<void>()
    const detachStarted = deferred<void>()
    const harness = await createPausedUpload(async () => {
      detachStarted.resolve()
      await detach.promise
    })
    await detachStarted.promise
    const reset = harness.store.suspendForRuntimeReset()
    expect(harness.store.getSnapshot().transfers[0]).toMatchObject({ status: 'paused', pausedByUser: true })
    detach.resolve()
    await reset
    expect(harness.store.getSnapshot().transfers[0]).toMatchObject({ status: 'paused', pausedByUser: true })
  })

  it('keeps a failed detach fence and does not reopen on resume', async () => {
    const harness = await createPausedUpload(async () => { throw new Error('detach failed') })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'failed')
    await harness.store.resumeTransfer(harness.id)
    expect(harness.openCommands).toHaveLength(1)
    expect(harness.store.getSnapshot().transfers[0]?.error).toContain('detach failed')
  })

  it('does not resume after cancel wins while detach is pending', async () => {
    const detach = deferred<void>()
    const detachStarted = deferred<void>()
    const harness = await createPausedUpload(async () => {
      detachStarted.resolve()
      await detach.promise
    })
    const resume = harness.store.resumeTransfer(harness.id)
    await detachStarted.promise
    harness.store.cancelTransfer(harness.id)
    detach.resolve()
    await resume
    await waitFor(() => harness.counters.cancelCommands === 1 && harness.counters.sessionCloses === 1)
    expect(harness.openCommands).toHaveLength(1)
    expect(harness.store.getSnapshot().transfers[0]?.status).toBe('cancelled')
  })

  it('releases a late upload resource when pause wins during FileUploadOpen', async () => {
    const openGate = deferred<void>()
    const openStarted = deferred<void>()
    const harness = await createUploadHarness({
      releaseResource: async () => undefined,
      beforeOpenResult: async () => {
        openStarted.resolve()
        await openGate.promise
      },
    })
    await openStarted.promise
    harness.store.pauseTransfer(harness.id)
    openGate.resolve()
    await waitFor(() => harness.counters.releaseCommands === 1)
    expect(harness.counters.openStreamCalls).toBe(0)
    expect(harness.statuses).not.toContain('transferring')
    expect(harness.store.getSnapshot().transfers[0]?.status).toBe('paused')
  })

  it('lets binding own cleanup when cancel wins before FileUploadOpen delivers a resource', async () => {
    const openGate = deferred<void>()
    const openStarted = deferred<void>()
    const harness = await createUploadHarness({
      releaseResource: async () => undefined,
      cancelOpenBeforeDelivery: true,
      beforeOpenResult: async () => {
        openStarted.resolve()
        await openGate.promise
      },
    })
    await openStarted.promise
    harness.store.cancelTransfer(harness.id)
    openGate.resolve()

    await waitFor(() => harness.counters.sessionCloses === 1)
    expect(harness.store.getSnapshot().transfers[0]?.status).toBe('cancelled')
    expect(harness.counters.cancelCommands).toBe(0)
    expect(harness.counters.releaseCommands).toBe(0)
    expect(harness.counters.bindingCancelledOpenCleanups).toBe(1)
  })

  it('closes a late stream and cancels its resource when cancel wins during stream open', async () => {
    const streamGate = deferred<void>()
    const streamStarted = deferred<void>()
    const harness = await createUploadHarness({
      releaseResource: async () => undefined,
      beforeStreamResult: async () => {
        streamStarted.resolve()
        await streamGate.promise
      },
    })
    await streamStarted.promise
    harness.store.cancelTransfer(harness.id)
    await waitFor(() => harness.counters.cancelCommands === 1 && harness.counters.sessionCloses === 1)
    expect(harness.statuses).not.toContain('transferring')
    expect(harness.store.getSnapshot().transfers[0]?.status).toBe('cancelled')
    streamGate.resolve()
    await waitFor(() => harness.counters.streamCloses === 1)
  })

  it('keeps the failed detach owner until a confirmed destructive cancel', async () => {
    const harness = await createPausedUpload(async () => { throw new Error('detach failed') })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'failed')
    expect(harness.counters.sessionCloses).toBe(0)
    harness.store.cancelTransfer(harness.id)
    await waitFor(() => harness.counters.cancelCommands === 1 && harness.counters.sessionCloses === 1)
  })

  it('upgrades a failing in-flight detach to destructive cancel without owner loss', async () => {
    const detach = deferred<void>()
    const detachStarted = deferred<void>()
    const harness = await createPausedUpload(async () => {
      detachStarted.resolve()
      await detach.promise
      throw new Error('detach failed')
    })
    await detachStarted.promise
    harness.store.cancelTransfer(harness.id)
    detach.resolve()
    await waitFor(() => harness.counters.cancelCommands === 1 && harness.counters.sessionCloses === 1)
    expect(harness.store.getSnapshot().transfers[0]?.status).toBe('cancelled')
  })

  it('uses a fresh session to cancel an already detached paused upload', async () => {
    const harness = await createPausedUpload(async () => undefined)
    await waitFor(() => harness.counters.releaseCommands === 1 && harness.counters.sessionCloses === 1)
    harness.store.cancelTransfer(harness.id)
    await waitFor(() => harness.counters.cancelCommands === 1 && harness.counters.sessionCloses === 2)
    expect(harness.store.getSnapshot().transfers[0]?.status).toBe('cancelled')
	expect(harness.cancelCredentials).toEqual(['upload_resume'])
  })

  it('aborts fresh cleanup on local reset and only closes the late session once', async () => {
    const storage = memoryStorage()
    const harness = await createUploadHarness({ releaseResource: async () => undefined, storage })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
    harness.store.pauseTransfer(harness.id)
    await waitFor(() => harness.counters.releaseCommands === 1 && harness.counters.sessionCloses === 1)
    const discardedOwner = detachedTransferOwner(harness.store, harness.id)

    const resolverEntered = deferred<void>()
    const sessionResult = deferred<ProtoClientSession>()
    let freshSignal: AbortSignal | undefined
    let lateSessionCloses = 0
    const lateSession: ProtoClientSession = {
      stamp: create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 2n }),
      isAlive: () => true,
      close: async () => { lateSessionCloses += 1 },
      subscribeEvents: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      openResourceStream: async () => { throw new Error('unused') },
      execute: async (envelope) => {
        if (envelope.command.case === 'fileTransferCancel') {
          harness.counters.cancelCommands += 1
          return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
            result: {
              case: 'fileTransferCancel',
              value: create(AnyTTYApiFile.FileTransferCancelResultSchema, { cancelled: true }),
            },
          })
        }
        throw new Error(`unexpected command ${envelope.command.case}`)
      },
    }
    harness.store.setSessionResolver(async (_machineId, signal) => {
      freshSignal = signal
      resolverEntered.resolve()
      return await sessionResult.promise
    })
    const snapshots: FileTransferStoreSnapshot[] = []
    harness.store.subscribe(() => snapshots.push(harness.store.getSnapshot()))

    harness.store.cancelTransfer(harness.id)
    await resolverEntered.promise
    expect(freshSignal?.aborted).toBe(false)

    await harness.store.discardForLocalReset()

    expect(freshSignal?.aborted).toBe(true)
    expect(harness.counters.cancelCommands).toBe(0)
    expect(harness.store.getSnapshot()).toEqual({ transfers: [], hasActiveTransfers: false })
    expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
    expect(transferOwnerCounts(harness.store)).toEqual(emptyTransferOwnerCounts())
    expect(discardedOwner.session).toBeUndefined()
    expect(discardedOwner.stream).toBeUndefined()
    expect(discardedOwner.resource).toBeUndefined()
    expect(discardedOwner.uploadResume).toBeUndefined()
    expect(discardedOwner.freshCleanup).toBeUndefined()
    expect(discardedOwner.teardown).toBeUndefined()
    const notificationsAfterReset = snapshots.length

    sessionResult.resolve(lateSession)
    await waitFor(() => lateSessionCloses === 1)
    await Promise.resolve()

    expect(lateSessionCloses).toBe(1)
    expect(harness.counters.cancelCommands).toBe(0)
    expect(harness.store.getSnapshot()).toEqual({ transfers: [], hasActiveTransfers: false })
    expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
    expect(snapshots).toHaveLength(notificationsAfterReset)
    expect(transferOwnerCounts(harness.store)).toEqual(emptyTransferOwnerCounts())
  })

  it('does not wait for a fresh-session resolver that never completes during local reset', async () => {
    const storage = memoryStorage()
    const harness = await createUploadHarness({ releaseResource: async () => undefined, storage })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
    harness.store.pauseTransfer(harness.id)
    await waitFor(() => harness.counters.releaseCommands === 1 && harness.counters.sessionCloses === 1)

    const resolverEntered = deferred<void>()
    let freshSignal: AbortSignal | undefined
    harness.store.setSessionResolver(async (_machineId, signal) => {
      freshSignal = signal
      resolverEntered.resolve()
      return await new Promise<ProtoClientSession>(() => undefined)
    })

    harness.store.cancelTransfer(harness.id)
    await resolverEntered.promise

    await expect(harness.store.discardForLocalReset()).resolves.toBeUndefined()

    expect(freshSignal?.aborted).toBe(true)
    expect(harness.counters.cancelCommands).toBe(0)
    expect(harness.store.getSnapshot()).toEqual({ transfers: [], hasActiveTransfers: false })
    expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
    expect(transferOwnerCounts(harness.store)).toEqual(emptyTransferOwnerCounts())
  })

  it('settles fresh cleanup when a fresh-session cancel command never completes', async () => {
    const storage = memoryStorage()
    const commandEntered = deferred<void>()
    let cancelSignal: AbortSignal | undefined
    const harness = await createUploadHarness({
      releaseResource: async () => undefined,
      storage,
      beforeCancelResult: async (signal) => {
        cancelSignal = signal
        commandEntered.resolve()
        await new Promise<void>(() => undefined)
      },
    })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
    harness.store.pauseTransfer(harness.id)
    await waitFor(() => harness.counters.releaseCommands === 1 && harness.counters.sessionCloses === 1)
    const discardedOwner = detachedTransferOwner(harness.store, harness.id)

    harness.store.cancelTransfer(harness.id)
    await commandEntered.promise
    const freshCleanup = requiredFreshCleanup(discardedOwner)
    expect(freshCleanup.completion).not.toBe(discardedOwner.teardown)
    expect(cancelSignal?.aborted).toBe(false)

    const discard = harness.store.discardForLocalReset()
    const cancellation = await freshCleanup.completion

    expect(cancellation.confirmed).toBe(false)
    expect(cancelSignal?.aborted).toBe(true)
    await discard
    expect(harness.counters.cancelCommands).toBe(1)
    expect(harness.counters.sessionCloses).toBe(2)
    expect(harness.store.getSnapshot()).toEqual({ transfers: [], hasActiveTransfers: false })
    expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
    expect(transferOwnerCounts(harness.store)).toEqual(emptyTransferOwnerCounts())
    expectDiscardedOwnerCleared(discardedOwner)
  })

  it('settles fresh cleanup when a live-session cancel command never completes', async () => {
    const storage = memoryStorage()
    const commandEntered = deferred<void>()
    let cancelSignal: AbortSignal | undefined
    const harness = await createUploadHarness({
      releaseResource: async () => { throw new Error('detach failed') },
      storage,
      beforeCancelResult: async (signal) => {
        cancelSignal = signal
        commandEntered.resolve()
        await new Promise<void>(() => undefined)
      },
    })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
    harness.store.pauseTransfer(harness.id)
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'failed')
    const discardedOwner = failedTransferOwner(harness.store, harness.id)

    harness.store.cancelTransfer(harness.id)
    await commandEntered.promise
    const freshCleanup = requiredFreshCleanup(discardedOwner)
    expect(freshCleanup.completion).not.toBe(discardedOwner.teardown)
    expect(cancelSignal?.aborted).toBe(false)

    const discard = harness.store.discardForLocalReset()
    const completion = freshCleanup.completion.then(
      () => 'fulfilled' as const,
      (error: unknown) => error,
    )

    await expect(completion).resolves.toMatchObject({ name: 'AbortError' })
    expect(cancelSignal?.aborted).toBe(true)
    await discard
    expect(harness.counters.cancelCommands).toBe(1)
    expect(harness.counters.sessionCloses).toBe(1)
    expect(harness.store.getSnapshot()).toEqual({ transfers: [], hasActiveTransfers: false })
    expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
    expect(transferOwnerCounts(harness.store)).toEqual(emptyTransferOwnerCounts())
    expectDiscardedOwnerCleared(discardedOwner)
  })

  it.each(['resolve', 'reject'] as const)('absorbs a late cancel %s after local reset', async (outcome) => {
    const storage = memoryStorage()
    const commandEntered = deferred<void>()
    const cancelResult = deferred<void>()
    let cancelSignal: AbortSignal | undefined
    const unhandled: unknown[] = []
    const onUnhandled = (reason: unknown) => { unhandled.push(reason) }
    nodeTestRuntime().process.on('unhandledRejection', onUnhandled)
    try {
      const harness = await createUploadHarness({
        releaseResource: async () => undefined,
        storage,
        beforeCancelResult: async (signal) => {
          cancelSignal = signal
          commandEntered.resolve()
          await cancelResult.promise
        },
      })
      await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
      harness.store.pauseTransfer(harness.id)
      await waitFor(() => harness.counters.releaseCommands === 1 && harness.counters.sessionCloses === 1)
      const discardedOwner = detachedTransferOwner(harness.store, harness.id)
      const snapshots: FileTransferStoreSnapshot[] = []
      harness.store.subscribe(() => snapshots.push(harness.store.getSnapshot()))

      harness.store.cancelTransfer(harness.id)
      await commandEntered.promise
      await harness.store.discardForLocalReset()
      const notificationsAfterReset = snapshots.length

      if (outcome === 'resolve') cancelResult.resolve()
      else cancelResult.reject(new Error('late cancel rejection'))
      await new Promise<void>((resolve) => { nodeTestRuntime().setImmediate(resolve) })

      expect(unhandled).toEqual([])
      expect(cancelSignal?.aborted).toBe(true)
      expect(harness.counters.cancelCommands).toBe(1)
      expect(harness.counters.sessionCloses).toBe(2)
      expect(harness.store.getSnapshot()).toEqual({ transfers: [], hasActiveTransfers: false })
      expect(storage.getItem('anytty.file-transfers.v2')).toBeNull()
      expect(snapshots).toHaveLength(notificationsAfterReset)
      expect(transferOwnerCounts(harness.store)).toEqual(emptyTransferOwnerCounts())
      expectDiscardedOwnerCleared(discardedOwner)
    } finally {
      nodeTestRuntime().process.off('unhandledRejection', onUnhandled)
    }
  })

  it('keeps the owner reachable when cancel arrives during session close', async () => {
	const closeGate = deferred<void>()
	const closeStarted = deferred<void>()
	const harness = await createUploadHarness({
	  releaseResource: async () => undefined,
	  beforeSessionClose: async () => {
		closeStarted.resolve()
		await closeGate.promise
	  },
	})
	await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
	harness.store.pauseTransfer(harness.id)
	await closeStarted.promise
	harness.store.dismissTransfer(harness.id)
	expect(harness.store.getSnapshot().transfers).toHaveLength(1)
	closeGate.resolve()
	await waitFor(() => harness.store.getSnapshot().transfers.length === 0)
	expect(harness.cancelCredentials).toEqual(['upload_resume'])
  })

  it('retries fresh-session cleanup after the first resolver failure', async () => {
	const closeGate = deferred<void>()
	const closeStarted = deferred<void>()
	const harness = await createUploadHarness({
	  releaseResource: async () => undefined,
	  freshResolverFailures: 1,
	  beforeSessionClose: async () => {
		closeStarted.resolve()
		await closeGate.promise
	  },
	})
	await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
	harness.store.pauseTransfer(harness.id)
	await closeStarted.promise
	harness.store.cancelTransfer(harness.id)
	closeGate.resolve()
	await waitFor(() => harness.resolverCalls() === 2)
	expect(harness.counters.cancelCommands).toBe(0)
	harness.store.cancelTransfer(harness.id)
	await waitFor(() => harness.counters.cancelCommands === 1)
	expect(harness.resolverCalls()).toBe(3)
	expect(harness.cancelCredentials).toEqual(['upload_resume'])
  })

  it('dismisses only after in-flight detach is upgraded and destructive cancel succeeds', async () => {
    const detach = deferred<void>()
    const harness = await createPausedUpload(async () => { await detach.promise })
    harness.store.dismissTransfer(harness.id)
    expect(harness.store.getSnapshot().transfers).toHaveLength(1)
    detach.resolve()
    await waitFor(() => harness.store.getSnapshot().transfers.length === 0)
    expect(harness.counters.cancelCommands).toBe(1)
  })

  it('keeps a dismissed transfer reachable when destructive cleanup fails', async () => {
    const detach = deferred<void>()
    const harness = await createUploadHarness({
      releaseResource: async () => { await detach.promise },
      cancelBehavior: 'false',
    })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
    harness.store.pauseTransfer(harness.id)
    harness.store.dismissTransfer(harness.id)
    detach.resolve()
    await waitFor(() => harness.counters.cancelCommands === 1)
    expect(harness.store.getSnapshot().transfers).toHaveLength(1)
    expect(harness.counters.sessionCloses).toBe(0)
  })

  it.each(['false', 'throw'] as const)('retains destructive owner when cancel result is %s', async (cancelBehavior) => {
    const harness = await createUploadHarness({
      releaseResource: async () => { throw new Error('detach failed') },
      cancelBehavior,
    })
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
    harness.store.pauseTransfer(harness.id)
    await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'failed')
    harness.store.cancelTransfer(harness.id)
    await waitFor(() => harness.counters.cancelCommands === 1)
    expect(harness.counters.sessionCloses).toBe(0)
  })

  it('closes a live failed-cleanup lease before replacing it with a fresh session', async () => {
	const harness = await createUploadHarness({
	  releaseResource: async () => { throw new Error('detach failed') },
	  cancelBehavior: 'false_then_success',
	})
	await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
	harness.store.pauseTransfer(harness.id)
	await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'failed')
	harness.store.cancelTransfer(harness.id)
	await waitFor(() => harness.counters.cancelCommands === 2 && harness.counters.sessionCloses === 2)
	expect(harness.cancelCredentials).toEqual(['transfer', 'upload_resume'])
  })

  it('bounds pause while session resolution never returns and ignores the late session', async () => {
    vi.stubGlobal('self', globalThis)
    const lateSession = deferred<ProtoClientSession>()
    let resolverCalls = 0
    let lateSessionCloses = 0
    const store = new NativeFileTransferStore()
    store.setSessionResolver(async () => {
      resolverCalls += 1
      if (resolverCalls === 1) return await lateSession.promise
      throw new Error('second attempt reached')
    })
    store.startUpload('studio', 'content://upload', 'demo.bin', 8, '/tmp')
    const transfer = store.getSnapshot().transfers[0]
    if (!transfer) throw new Error('transfer was not created')
    store.pauseTransfer(transfer.id)
    await store.resumeTransfer(transfer.id)
    expect(resolverCalls).toBe(2)
    lateSession.resolve({
      stamp: create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n }),
      isAlive: () => true,
      close: async () => { lateSessionCloses += 1 },
      subscribeEvents: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      openResourceStream: async () => { throw new Error('unused') },
      execute: async () => { throw new Error('unused') },
    })
    await waitFor(() => lateSessionCloses === 1)
  })

  it('accepts ordered upload ACKs that lag behind the current send offset', async () => {
    vi.stubGlobal('self', globalThis)
    const subscribers: Array<(type: number, payload: Uint8Array) => void> = []
    const sentChunks: Array<{ offset: number, size: number }> = []
    const stream: ProtoResourceStream = {
      handle: 1n,
      send: async (type, payload) => {
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_DATA) {
          const data = decodeFileTransferDataPayload(payload)
          sentChunks.push({ offset: data.offset, size: data.data.byteLength })
          return
        }
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_FINISH) {
          for (const chunk of sentChunks) {
            const ack = encodeFileTransferAckPayload({ offset: chunk.offset + chunk.size, windowBytes: chunk.size })
            for (const subscriber of subscribers) subscriber(AnyTTYClientBinding.ResourceStreamFrameType.FILE_ACK, ack)
          }
          const result = toBinary(FileTransferResultSchema, create(FileTransferResultSchema, { size: 8n }))
          for (const subscriber of subscribers) subscriber(AnyTTYClientBinding.ResourceStreamFrameType.FILE_RESULT, result)
        }
      },
      subscribe: (listener) => {
        subscribers.push(listener)
        return { close() {} }
      },
      subscribeClosed: () => ({ close() {} }),
      close: async () => undefined,
    }
    const session = uploadSession(stream, 8, 4, 8)
    vi.stubGlobal('fetch', vi.fn(async () => new Response(new Blob([new Uint8Array(8)]))))
    const store = new NativeFileTransferStore()
    store.setSessionResolver(async () => session)

    store.startUpload('studio', 'content://upload', 'demo.bin', 8, '/tmp')

    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'completed')
    expect(sentChunks).toEqual([{ offset: 0, size: 4 }, { offset: 4, size: 4 }])
  })

  it('keeps Android bridge upload messages below the 64 KiB fragmentation boundary', async () => {
    vi.stubGlobal('self', globalThis)
    const subscribers: Array<(type: number, payload: Uint8Array) => void> = []
    const sentChunkSizes: number[] = []
    const totalSize = 64 * 1024
    const stream: ProtoResourceStream = {
      handle: 1n,
      send: async (type, payload) => {
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_DATA) {
          sentChunkSizes.push(decodeFileTransferDataPayload(payload).data.byteLength)
          return
        }
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_FINISH) {
          const result = toBinary(FileTransferResultSchema, create(FileTransferResultSchema, { size: BigInt(totalSize) }))
          for (const subscriber of subscribers) subscriber(AnyTTYClientBinding.ResourceStreamFrameType.FILE_RESULT, result)
        }
      },
      subscribe: (listener) => {
        subscribers.push(listener)
        return { close() {} }
      },
      subscribeClosed: () => ({ close() {} }),
      close: async () => undefined,
    }
    const store = new NativeFileTransferStore()
    store.setSessionResolver(async () => uploadSession(stream, totalSize, totalSize, totalSize))
    vi.stubGlobal('fetch', vi.fn(async () => new Response(new Blob([new Uint8Array(totalSize)]))))

    store.startUpload('studio', 'content://upload', 'boundary.bin', totalSize, '/tmp')

    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'completed')
    expect(sentChunkSizes).toEqual([60 * 1024, 4 * 1024])
  })

  it('fails a window-blocked upload when the daemon sends FILE_ERROR without closing the stream', async () => {
    vi.stubGlobal('self', globalThis)
    const subscribers: Array<(type: number, payload: Uint8Array) => void> = []
    const sentOffsets: number[] = []
    const stream: ProtoResourceStream = {
      handle: 1n,
      send: async (type, payload) => {
        if (type !== AnyTTYClientBinding.ResourceStreamFrameType.FILE_DATA) return
        const data = decodeFileTransferDataPayload(payload)
        sentOffsets.push(data.offset)
        const error = toBinary(ErrorEnvelopeSchema, create(ErrorEnvelopeSchema, {
          error: create(ProtocolErrorSchema, { code: 500, message: 'daemon write failed' }),
        }))
        for (const subscriber of subscribers) subscriber(AnyTTYClientBinding.ResourceStreamFrameType.ERROR, error)
      },
      subscribe: (listener) => {
        subscribers.push(listener)
        return { close() {} }
      },
      subscribeClosed: () => ({ close() {} }),
      close: async () => undefined,
    }
    const store = new NativeFileTransferStore()
    store.setSessionResolver(async () => uploadSession(stream, 8, 4, 4))
    vi.stubGlobal('fetch', vi.fn(async () => new Response(new Blob([new Uint8Array(8)]))))

    store.startUpload('studio', 'content://upload', 'demo.bin', 8, '/tmp')

    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'failed')
    expect(store.getSnapshot().transfers[0]?.error).toBe('daemon write failed')
    expect(sentOffsets).toEqual([0])
  })

  it('closes the session-owned download when the cancel RPC has no usable result', async () => {
    vi.stubGlobal('self', globalThis)
    const counters = { cancelCommands: 0, releaseCommands: 0, sessionCloses: 0, streamCloses: 0, activeResources: 1 }
    const session = downloadSession(counters)
    const store = new NativeFileTransferStore()
    store.setSessionResolver(async () => session)
    store.startDownload('studio', 'large.bin', 1024, '/tmp/large.bin')
    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'transferring')

    store.cancelTransfer(store.getSnapshot().transfers[0]!.id)

    await waitFor(() => counters.cancelCommands === 1 && counters.releaseCommands === 1 && counters.sessionCloses === 1 && counters.streamCloses === 1)
    expect(store.getSnapshot().transfers[0]?.status).toBe('cancelled')
    expect(counters.activeResources).toBe(0)
  })

  it('keeps a download failed when neither cancel nor release reaches the daemon', async () => {
    vi.stubGlobal('self', globalThis)
    const counters = { cancelCommands: 0, releaseCommands: 0, sessionCloses: 0, streamCloses: 0, activeResources: 1 }
    const session = downloadSession(counters, true)
    const store = new NativeFileTransferStore()
    store.setSessionResolver(async () => session)
    store.startDownload('studio', 'large.bin', 1024, '/tmp/large.bin')
    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'transferring')

    store.cancelTransfer(store.getSnapshot().transfers[0]!.id)

    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'failed')
    expect(counters.activeResources).toBe(1)
    expect(counters.sessionCloses).toBe(0)
  })

  it('does not let a late download cleanup error overwrite cancelled state', async () => {
    vi.stubGlobal('self', globalThis)
    const releaseGate = deferred<void>()
    const releaseStarted = deferred<void>()
    let closeListener: ((error: Error) => void) | undefined
    let sessionCloses = 0
    let releaseCommands = 0
    let activeResources = 1
    const stamp = create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'direct', generation: 1n })
    const resource = create(AnyTTYApiCommon.ResourceHandleSchema, {
      kind: AnyTTYApiCommon.ResourceKind.FILE_TRANSFER,
      opaqueToken: new Uint8Array([4]),
      session: stamp,
    })
    const session: ProtoClientSession = {
      stamp,
      isAlive: () => true,
      close: async () => { sessionCloses += 1 },
      subscribeEvents: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      openResourceStream: async () => ({
        handle: 1n,
        send: async () => undefined,
        subscribe: () => ({ close() {} }),
        subscribeClosed: (listener) => { closeListener = listener; return { close() {} } },
        close: async () => undefined,
      }),
      execute: async (envelope) => {
        if (envelope.command.case === 'fileDownloadOpen') {
          return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
            result: {
              case: 'fileTransferOpen',
              value: create(AnyTTYApiFile.FileTransferOpenResultSchema, {
                transfer: create(AnyTTYApiFile.FileTransferHandleSchema, { resource, size: 1024n, chunkBytes: 4, windowBytes: 8n }),
              }),
            },
          })
        }
        if (envelope.command.case === 'releaseResource') {
          releaseCommands += 1
          if (releaseCommands === 1) {
            releaseStarted.resolve()
            await releaseGate.promise
            throw new Error('late release failure')
          }
          activeResources = 0
          return create(AnyTTYApiApplication.ResultEnvelopeSchema)
        }
        if (envelope.command.case === 'fileTransferCancel') {
          throw new Error('cancel response unavailable')
        }
        throw new Error(`unexpected command ${envelope.command.case}`)
      },
    }
    const store = new NativeFileTransferStore()
    store.setSessionResolver(async () => session)
    store.startDownload('studio', 'large.bin', 1024, '/tmp/large.bin')
    await waitFor(() => store.getSnapshot().transfers[0]?.status === 'transferring' && closeListener !== undefined)
    closeListener!(new Error('producer closed'))
    await releaseStarted.promise

    store.cancelTransfer(store.getSnapshot().transfers[0]!.id)
    releaseGate.resolve()

    await waitFor(() => sessionCloses === 1)
    expect(store.getSnapshot().transfers[0]?.status).toBe('cancelled')
    expect(activeResources).toBe(0)
  })
})

function uploadSession(stream: ProtoResourceStream, size: number, chunkBytes: number, windowBytes: number): ProtoClientSession {
  const stamp = create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n })
  return {
    stamp,
    isAlive: () => true,
    close: async () => undefined,
    subscribeEvents: () => ({ close() {} }),
    subscribeClosed: () => ({ close() {} }),
    openResourceStream: async () => stream,
    execute: async (envelope) => {
      if (envelope.command.case === 'fileUploadOpen') {
        return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
          result: {
            case: 'fileTransferOpen',
            value: create(AnyTTYApiFile.FileTransferOpenResultSchema, {
              transfer: create(AnyTTYApiFile.FileTransferHandleSchema, {
                resource: create(AnyTTYApiCommon.ResourceHandleSchema, {
                  kind: AnyTTYApiCommon.ResourceKind.FILE_TRANSFER,
                  opaqueToken: new Uint8Array([1]),
                  session: stamp,
                }),
                size: BigInt(size),
                chunkBytes,
                windowBytes: BigInt(windowBytes),
                resume: create(AnyTTYApiFile.FileUploadResumeHandleSchema, { opaqueToken: new Uint8Array([2]) }),
              }),
            }),
          },
        })
      }
      if (envelope.command.case === 'releaseResource') return create(AnyTTYApiApplication.ResultEnvelopeSchema)
      throw new Error(`unexpected command ${envelope.command.case}`)
    },
  }
}

function downloadSession(
  counters: { cancelCommands: number, releaseCommands: number, sessionCloses: number, streamCloses: number, activeResources: number },
  releaseFails = false,
): ProtoClientSession {
  const stamp = create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'direct', generation: 1n })
  const resource = create(AnyTTYApiCommon.ResourceHandleSchema, {
    kind: AnyTTYApiCommon.ResourceKind.FILE_TRANSFER,
    opaqueToken: new Uint8Array([3]),
    session: stamp,
  })
  return {
    stamp,
    isAlive: () => true,
    close: async () => { counters.sessionCloses += 1 },
    subscribeEvents: () => ({ close() {} }),
    subscribeClosed: () => ({ close() {} }),
    openResourceStream: async () => ({
      handle: 1n,
      send: async () => undefined,
      subscribe: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      close: async () => { counters.streamCloses += 1 },
    }),
    execute: async (envelope) => {
      if (envelope.command.case === 'fileDownloadOpen') {
        return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
          result: {
            case: 'fileTransferOpen',
            value: create(AnyTTYApiFile.FileTransferOpenResultSchema, {
              transfer: create(AnyTTYApiFile.FileTransferHandleSchema, {
                resource,
                size: 1024n,
                chunkBytes: 4,
                windowBytes: 8n,
              }),
            }),
          },
        })
      }
      if (envelope.command.case === 'fileTransferCancel') {
        counters.cancelCommands += 1
        throw new Error('cancel response unavailable')
      }
      if (envelope.command.case === 'releaseResource') {
        counters.releaseCommands += 1
        if (releaseFails) throw new Error('release response unavailable')
        counters.activeResources = 0
        return create(AnyTTYApiApplication.ResultEnvelopeSchema)
      }
      throw new Error(`unexpected command ${envelope.command.case}`)
    },
  }
}

async function createPausedUpload(releaseResource: () => Promise<void>) {
  const harness = await createUploadHarness({ releaseResource })
  await waitFor(() => harness.store.getSnapshot().transfers[0]?.status === 'transferring')
  harness.store.pauseTransfer(harness.id)
  return harness
}

async function createUploadHarness(options: {
  releaseResource: () => Promise<void>
  beforeOpenResult?: () => Promise<void>
  beforeStreamResult?: () => Promise<void>
	beforeSessionClose?: () => Promise<void>
  beforeCancelResult?: (signal: AbortSignal | undefined) => Promise<void>
  freshResolverFailures?: number
  cancelOpenBeforeDelivery?: boolean
  cancelBehavior?: 'success' | 'false' | 'throw' | 'false_then_success'
  storage?: Storage | null
}) {
  vi.stubGlobal('self', globalThis)
  const resumeToken = new Uint8Array([7, 8, 9])
  const resource = create(AnyTTYApiCommon.ResourceHandleSchema, {
    kind: AnyTTYApiCommon.ResourceKind.FILE_TRANSFER,
    opaqueToken: new Uint8Array([1, 2, 3]),
	session: create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n }),
  })
  const counters = { openStreamCalls: 0, releaseCommands: 0, cancelCommands: 0, streamCloses: 0, sessionCloses: 0, bindingCancelledOpenCleanups: 0 }
  const stream: ProtoResourceStream = {
    handle: 1n,
    send: async () => undefined,
    subscribe: () => ({ close() {} }),
    subscribeClosed: () => ({ close() {} }),
    close: async () => { counters.streamCloses += 1 },
  }
  const openCommands: AnyTTYApiFile.FileUploadOpenCommand[] = []
	const cancelCredentials: string[] = []
	let resolverCalls = 0
  const makeSession = (): ProtoClientSession => {
    let alive = true
    return {
      stamp: create(AnyTTYApiCommon.EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: BigInt(counters.sessionCloses + 1) }),
      isAlive: () => alive,
      close: async () => {
        if (!alive) return
		await options.beforeSessionClose?.()
        alive = false
        counters.sessionCloses += 1
      },
      subscribeEvents: () => ({ close() {} }),
      subscribeClosed: () => ({ close() {} }),
      openResourceStream: async () => {
        counters.openStreamCalls += 1
        await options.beforeStreamResult?.()
        return stream
      },
      execute: async (envelope, executeOptions) => {
      if (envelope.command.case === 'fileUploadOpen') {
        openCommands.push(envelope.command.value)
        if (openCommands.length > 1) throw new Error('resume-open-observed')
        await options.beforeOpenResult?.()
        if (options.cancelOpenBeforeDelivery && executeOptions?.signal?.aborted) {
          counters.bindingCancelledOpenCleanups += 1
          throw new DOMException('Aborted', 'AbortError')
        }
        return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
          result: {
            case: 'fileTransferOpen',
            value: create(AnyTTYApiFile.FileTransferOpenResultSchema, {
              transfer: create(AnyTTYApiFile.FileTransferHandleSchema, {
                resource,
                size: 8n,
                offset: 4n,
                chunkBytes: 4,
                windowBytes: 4n,
                resume: create(AnyTTYApiFile.FileUploadResumeHandleSchema, { opaqueToken: resumeToken }),
              }),
            }),
          },
        })
      }
      if (envelope.command.case === 'releaseResource') {
        counters.releaseCommands += 1
        await options.releaseResource()
        return create(AnyTTYApiApplication.ResultEnvelopeSchema)
      }
      if (envelope.command.case === 'fileTransferCancel') {
        counters.cancelCommands += 1
		cancelCredentials.push(envelope.command.value.transfer ? 'transfer' : envelope.command.value.uploadResume ? 'upload_resume' : 'missing')
		await options.beforeCancelResult?.(executeOptions?.signal)
        if (options.cancelBehavior === 'throw') throw new Error('cancel transport failed')
		const cancelled = options.cancelBehavior === 'false_then_success' ? counters.cancelCommands > 1 : options.cancelBehavior !== 'false'
        return create(AnyTTYApiApplication.ResultEnvelopeSchema, {
          result: {
            case: 'fileTransferCancel',
			value: create(AnyTTYApiFile.FileTransferCancelResultSchema, { cancelled }),
          },
        })
      }
        throw new Error(`unexpected command ${envelope.command.case}`)
      },
    }
  }
  vi.stubGlobal('fetch', vi.fn((_input: RequestInfo | URL, init?: RequestInit) => new Promise<Response>((_resolve, reject) => {
    init?.signal?.addEventListener('abort', () => reject(new DOMException('Aborted', 'AbortError')), { once: true })
  })))
  const store = new NativeFileTransferStore(options.storage)
  store.setSessionResolver(async () => {
	resolverCalls += 1
	if (resolverCalls > 1 && resolverCalls <= 1 + (options.freshResolverFailures ?? 0)) throw new Error('fresh session unavailable')
	return makeSession()
  })
  const statuses: string[] = []
  store.subscribe(() => {
    const status = store.getSnapshot().transfers[0]?.status
    if (status) statuses.push(status)
  })
  store.startUpload('studio', 'content://upload', 'demo.bin', 8, '/tmp')
  const transfer = store.getSnapshot().transfers[0]
  if (!transfer) throw new Error('transfer was not created')
  return { store, id: transfer.id, openCommands, resumeToken, counters, statuses, cancelCredentials, resolverCalls: () => resolverCalls }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function memoryStorage(): Storage {
  const values = new Map<string, string>()
  return {
    get length() { return values.size },
    clear() { values.clear() },
    getItem(key) { return values.get(key) ?? null },
    key(index) { return [...values.keys()][index] ?? null },
    removeItem(key) { values.delete(key) },
    setItem(key, value) { values.set(key, value) },
  }
}

function transferOwnerCounts(store: object) {
  const owners = store as {
    active: Map<unknown, unknown>
    taskOwners: Set<unknown>
    pendingTeardowns: Map<unknown, unknown>
    failedCleanupOwners: Map<unknown, unknown>
    detachedCleanupOwners: Map<unknown, unknown>
    destructiveRetries: Map<unknown, unknown>
    pendingDismissals: Set<unknown>
    resumeTransitions: Map<unknown, unknown>
    transitionEpochs: Map<unknown, unknown>
    progressSamples: Map<unknown, unknown>
  }
  return {
    active: owners.active.size,
    tasks: owners.taskOwners.size,
    teardowns: owners.pendingTeardowns.size,
    failed: owners.failedCleanupOwners.size,
    detached: owners.detachedCleanupOwners.size,
    retries: owners.destructiveRetries.size,
    dismissals: owners.pendingDismissals.size,
    resumes: owners.resumeTransitions.size,
    epochs: owners.transitionEpochs.size,
    progress: owners.progressSamples.size,
  }
}

function detachedTransferOwner(store: object, id: string): Record<string, unknown> {
  const owner = (store as { detachedCleanupOwners: Map<string, Record<string, unknown>> }).detachedCleanupOwners.get(id)
  if (!owner) throw new Error('detached transfer owner is unavailable')
  return owner
}

function failedTransferOwner(store: object, id: string): Record<string, unknown> {
  const owner = (store as { failedCleanupOwners: Map<string, Record<string, unknown>> }).failedCleanupOwners.get(id)
  if (!owner) throw new Error('failed transfer owner is unavailable')
  return owner
}

function requiredFreshCleanup(owner: Record<string, unknown>): { cancel: AbortController, completion: Promise<{ confirmed: boolean }> } {
  const freshCleanup = owner.freshCleanup as { cancel: AbortController, completion: Promise<{ confirmed: boolean }> } | undefined
  if (!freshCleanup) throw new Error('fresh cleanup owner is unavailable')
  return freshCleanup
}

function expectDiscardedOwnerCleared(owner: Record<string, unknown>): void {
  expect(owner.session).toBeUndefined()
  expect(owner.stream).toBeUndefined()
  expect(owner.resource).toBeUndefined()
  expect(owner.uploadResume).toBeUndefined()
  expect(owner.freshCleanup).toBeUndefined()
  expect(owner.teardown).toBeUndefined()
}

function emptyTransferOwnerCounts() {
  return { active: 0, tasks: 0, teardowns: 0, failed: 0, detached: 0, retries: 0, dismissals: 0, resumes: 0, epochs: 0, progress: 0 }
}

type NodeTestRuntime = {
  process: { on(event: 'unhandledRejection', listener: (reason: unknown) => void): void, off(event: 'unhandledRejection', listener: (reason: unknown) => void): void }
  setImmediate(callback: () => void): unknown
}

function nodeTestRuntime(): NodeTestRuntime {
  return globalThis as unknown as NodeTestRuntime
}

async function waitFor(predicate: () => boolean): Promise<void> {
  const deadline = Date.now() + 1_000
  while (!predicate()) {
    if (Date.now() > deadline) throw new Error('condition was not reached')
    await new Promise((resolve) => setTimeout(resolve, 1))
  }
}
