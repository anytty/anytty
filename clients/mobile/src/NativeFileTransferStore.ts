import { Capacitor } from '@capacitor/core'
import { create } from '@bufbuild/protobuf'
import {
  AnyTTYApiApplication,
  AnyTTYApiFile,
  AnyTTYClientBinding,
  type ProtoClientSession,
  type ProtoResourceStream,
  decodeFileStreamErrorPayload,
  decodeFileTransferAckPayload,
  decodeFileTransferDataPayload,
  decodeFileTransferFinishPayload,
  decodeFileTransferResultPayload,
  encodeFileTransferAckPayload,
  encodeFileTransferDataPayload,
  encodeFileTransferFinishPayload,
} from '@anytty/ui'
import NativeFilePicker from './plugins/nativeFilePicker'

export type TransferStatus = 'pending' | 'transferring' | 'paused' | 'completed' | 'failed' | 'cancelled' | 'missing'

export interface TransferInfo {
  id: string
  machineId?: string
  name: string
  direction: 'download' | 'upload'
  totalSize: number
  transferredSize: number
  status: TransferStatus
  startedAt: number
  updatedAt?: number
  bytesPerSecond?: number
  error?: string
  filePath?: string
  localUri?: string | undefined
  targetDir?: string | undefined
  savedPath?: string | undefined
  savedUri?: string | undefined
  uploadResumeToken?: Uint8Array | undefined
  remoteModifiedAtUnixNano?: bigint | undefined
  pausedByUser?: boolean | undefined
}

export interface FileTransferStoreSnapshot {
  transfers: TransferInfo[]
  hasActiveTransfers: boolean
}

type NativeTransferSessionResolver = (machineId: string, signal: AbortSignal) => Promise<ProtoClientSession | null | undefined>
type CleanupConfirmation = { confirmed: boolean, error?: Error }

type FreshCleanupOwner = {
  cancel: AbortController
  completion: Promise<CleanupConfirmation>
}

type ActiveTransfer = {
  epoch: number
  storeEpoch: number
  machineId: string
  direction: TransferInfo['direction']
  cancel: AbortController
  session?: ProtoClientSession
  stream?: ProtoResourceStream
  resource?: NonNullable<AnyTTYApiFile.FileTransferHandle['resource']>
  uploadResume?: NonNullable<AnyTTYApiFile.FileTransferHandle['resume']>
  destructiveCancel: boolean
  readyForClose: Promise<void>
  markReadyForClose: () => void
  freshCleanup?: FreshCleanupOwner
  teardown?: Promise<void>
}

/** NativeFileTransferStore consumes apipb and Go-owned resource streams; it owns only UI progress and local Blob/URI access. */
export class NativeFileTransferStore {
  private transfers: TransferInfo[] = []
  private readonly listeners = new Set<() => void>()
  private readonly active = new Map<string, ActiveTransfer>()
  private readonly taskOwners = new Set<ActiveTransfer>()
  private readonly downloadChunks = new Map<string, Uint8Array[]>()
  private taskTeardowns = new WeakMap<ActiveTransfer, Promise<void>>()
  private readonly pendingTeardowns = new Map<string, Promise<void>>()
  private readonly failedCleanupOwners = new Map<string, ActiveTransfer>()
  private readonly detachedCleanupOwners = new Map<string, ActiveTransfer>()
  private readonly destructiveRetries = new Map<string, Promise<void>>()
  private readonly pendingDismissals = new Set<string>()
  private readonly resumeTransitions = new Map<string, Promise<void>>()
  private readonly transitionEpochs = new Map<string, number>()
  private readonly progressSamples = new Map<string, { at: number; bytes: number; speed: number; notifiedAt: number }>()
  private readonly storage: Storage | null
  private resolver: NativeTransferSessionResolver | null = null
  private version = 0
  private cachedVersion = -1
  private cachedSnapshot: FileTransferStoreSnapshot | null = null
  private readonly cachedMachineSnapshots = new Map<string, FileTransferStoreSnapshot>()
  private storeEpoch = 0
  private discarding = false
  private discardPromise: Promise<void> | null = null

  constructor(storage: Storage | null = transferStorage()) {
    this.storage = storage
    this.transfers = loadPersistedTransfers(storage)
  }

  setSessionResolver(resolver: NativeTransferSessionResolver | null): void { this.resolver = resolver }

  startDownload(machineId: string, fileName: string, fileSize: number, filePath: string, offset = 0): void {
    if (this.discarding) return
    const existing = this.transfers.find((item) => item.direction === 'download' && item.machineId === machineId && item.filePath === filePath && item.totalSize === fileSize && item.status !== 'completed' && item.status !== 'cancelled')
    if (existing) {
      if (existing.status === 'pending' || existing.status === 'transferring') return
      this.update(existing.id, { name: fileName, transferredSize: Math.max(existing.transferredSize, offset), pausedByUser: false, error: undefined })
      void this.resumeTransfer(existing.id)
      return
    }
    const id = transferID('download', machineId, filePath)
    const storeEpoch = this.storeEpoch
    this.advanceTransition(id)
    this.upsert({ id, machineId, name: fileName, direction: 'download', totalSize: fileSize, transferredSize: offset, status: 'pending', startedAt: Date.now(), updatedAt: Date.now(), filePath, pausedByUser: false })
    void this.runDownload(id).catch((error) => this.fail(id, error, storeEpoch))
  }

  async getDownloadResumeOffset(machineId?: string, filePath?: string, fileSize?: number): Promise<number> {
    if (Capacitor.isNativePlatform() && machineId && filePath && typeof fileSize === 'number') {
      const result = await NativeFilePicker.getDownloadResumeOffset({ machineId, remotePath: filePath, totalSize: fileSize })
      return Math.max(0, Math.min(fileSize, result.offset))
    }
    const transfer = this.transfers.find((item) => item.direction === 'download' && item.machineId === machineId && item.filePath === filePath && item.totalSize === fileSize)
    return transfer && this.downloadChunks.has(transfer.id) ? transfer.transferredSize : 0
  }

  startUpload(machineId: string, contentUri: string, fileName: string, fileSize: number, targetDir: string): void {
    if (this.discarding) return
    const id = transferID('upload', machineId, `${targetDir}/${fileName}`)
    const storeEpoch = this.storeEpoch
    this.advanceTransition(id)
    this.upsert({ id, machineId, name: fileName, direction: 'upload', totalSize: fileSize, transferredSize: 0, status: 'pending', startedAt: Date.now(), updatedAt: Date.now(), localUri: contentUri, targetDir, pausedByUser: false })
    void this.runUpload(id).catch((error) => this.fail(id, error, storeEpoch))
  }

  cancelTransfer(id: string): void {
    if (!this.transfers.some((item) => item.id === id)) return
    void this.requestCancel(id)?.catch(() => undefined)
  }

  private requestCancel(id: string): Promise<void> | null {
    const storeEpoch = this.storeEpoch
    this.advanceTransition(id)
    const task = this.active.get(id) ?? this.failedCleanupOwners.get(id) ?? this.detachedCleanupOwners.get(id)
    task?.cancel.abort()
    let cleanup: Promise<void> | null = null
    if (task) {
      task.destructiveCancel = true
      cleanup = this.active.get(id) === task ? this.closeTask(id, task) : this.retryDestructiveCleanup(id, task)
    } else {
      this.pendingTeardowns.delete(id)
    }
    this.downloadChunks.delete(id)
    const transfer = this.transfers.find((item) => item.id === id)
    const discardPartial = Capacitor.isNativePlatform() && transfer?.direction === 'download' && transfer.machineId && transfer.filePath
      ? () => NativeFilePicker.discardDownloadPartial({ machineId: transfer.machineId!, remotePath: transfer.filePath!, totalSize: transfer.totalSize }).then(() => undefined, () => undefined)
      : null
    const cancellation = cleanup
      ? cleanup.then(async () => { await discardPartial?.() })
      : discardPartial?.() ?? null
    if (cancellation) {
      void cancellation.then(
        () => this.update(id, { status: 'cancelled', error: undefined, updatedAt: Date.now(), bytesPerSecond: 0 }, true, storeEpoch),
        (error) => this.fail(id, error, storeEpoch),
      )
    } else {
      this.update(id, { status: 'cancelled', updatedAt: Date.now(), bytesPerSecond: 0 })
    }
    return cancellation
  }

  pauseTransfer(id: string): void {
    if (!this.transfers.some((item) => item.id === id)) return
    const storeEpoch = this.storeEpoch
    this.advanceTransition(id)
    const task = this.active.get(id)
    task?.cancel.abort()
    if (task) void this.closeTask(id, task).catch((error) => this.fail(id, error, storeEpoch))
    this.update(id, { status: 'paused', pausedByUser: true, updatedAt: Date.now(), bytesPerSecond: 0 })
  }

  async resumeTransfer(id: string): Promise<void> {
    if (this.discarding) return
    const transfer = this.transfers.find((item) => item.id === id)
    if (!transfer || !canResume(transfer.status)) return
    const requestEpoch = this.transitionEpochs.get(id) ?? 0
    const storeEpoch = this.storeEpoch
    const previous = this.resumeTransitions.get(id) ?? Promise.resolve()
    let transition!: Promise<void>
    transition = previous.catch(() => undefined).then(() => this.runResumeTransition(id, requestEpoch, storeEpoch))
    this.resumeTransitions.set(id, transition)
    try {
      await transition
    } finally {
      if (this.resumeTransitions.get(id) === transition) this.resumeTransitions.delete(id)
    }
  }

  private async runResumeTransition(id: string, requestEpoch: number, storeEpoch: number): Promise<void> {
    try {
      let transfer = this.transfers.find((item) => item.id === id)
      if (this.storeEpoch !== storeEpoch || !transfer || !canResume(transfer.status) || (this.transitionEpochs.get(id) ?? 0) !== requestEpoch) return
      const teardown = this.pendingTeardowns.get(id)
      if (teardown) await teardown
      transfer = this.transfers.find((item) => item.id === id)
      if (this.storeEpoch !== storeEpoch || !transfer || !canResume(transfer.status) || (this.transitionEpochs.get(id) ?? 0) !== requestEpoch) return
      this.advanceTransition(id)
      this.update(id, { status: 'pending', pausedByUser: false, error: undefined, updatedAt: Date.now() })
      if (transfer.direction === 'download') await this.runDownload(id)
      else await this.runUpload(id)
    } catch (error) {
      if (this.storeEpoch !== storeEpoch) return
      if (this.transfers.find((item) => item.id === id)?.status === 'cancelled') return
      this.fail(id, error, storeEpoch)
    }
  }

  async resumeAllTransfers(machineId?: string): Promise<void> {
    if (this.discarding) return
    const transfers = this.transfers.filter((transfer) => (!machineId || transfer.machineId === machineId) && canResume(transfer.status))
    await Promise.allSettled(transfers.map((transfer) => this.resumeTransfer(transfer.id)))
  }

  async suspendForRuntimeReset(): Promise<void> {
    if (this.discardPromise) await this.discardPromise
    const teardowns = new Set<Promise<unknown>>()
    for (const task of this.taskOwners) {
      const freshCleanup = task.freshCleanup
      if (!freshCleanup) continue
      freshCleanup.cancel.abort()
      teardowns.add(freshCleanup.completion)
      if (task.teardown) teardowns.add(task.teardown)
    }
    for (const [id, task] of this.active) {
      this.advanceTransition(id)
      task.cancel.abort()
      const transfer = this.transfers.find((item) => item.id === id)
      if (transfer?.status === 'pending' || transfer?.status === 'transferring') {
        this.update(id, { status: 'failed', pausedByUser: false, error: 'Connection changed; transfer will resume', updatedAt: Date.now(), bytesPerSecond: 0 })
      }
      teardowns.add(this.closeTask(id, task))
    }
    await Promise.allSettled(teardowns)
  }

  discardForLocalReset(): Promise<void> {
    if (this.discardPromise) return this.discardPromise
    this.discarding = true
    this.storeEpoch += 1
    const discard = this.performLocalResetDiscard()
    let tracked!: Promise<void>
    tracked = discard.finally(() => {
      if (this.discardPromise === tracked) this.discardPromise = null
      this.discarding = false
    })
    this.discardPromise = tracked
    return tracked
  }

  private async performLocalResetDiscard(): Promise<void> {
    const tasks = new Set<ActiveTransfer>([
      ...this.taskOwners,
      ...this.active.values(),
      ...this.failedCleanupOwners.values(),
      ...this.detachedCleanupOwners.values(),
    ])
    const drainableTeardowns = new Set<Promise<unknown>>()
    for (const task of tasks) {
      task.cancel.abort()
      const freshCleanup = task.freshCleanup
      if (!freshCleanup) continue
      freshCleanup.cancel.abort()
      drainableTeardowns.add(freshCleanup.completion)
      if (task.teardown) drainableTeardowns.add(task.teardown)
    }

    this.transfers = []
    this.downloadChunks.clear()
    this.publishEmptyResetSnapshot()

    await Promise.allSettled([
      ...drainableTeardowns,
      ...[...tasks].map((task) => this.closeDiscardedTask(task)),
    ])

    this.active.clear()
    this.taskOwners.clear()
    this.pendingTeardowns.clear()
    this.failedCleanupOwners.clear()
    this.detachedCleanupOwners.clear()
    this.destructiveRetries.clear()
    this.pendingDismissals.clear()
    this.resumeTransitions.clear()
    this.transitionEpochs.clear()
    this.progressSamples.clear()
    this.taskTeardowns = new WeakMap<ActiveTransfer, Promise<void>>()
    removePersistedTransfers(this.storage)
  }

  async resumeInterruptedTransfers(machineId?: string): Promise<void> {
    if (this.discarding) return
    const transfers = this.transfers.filter((transfer) => (!machineId || transfer.machineId === machineId) && !transfer.pausedByUser && canResume(transfer.status))
    await Promise.allSettled(transfers.map((transfer) => this.resumeTransfer(transfer.id)))
  }

  dismissTransfer(id: string): void {
    if (!this.transfers.some((item) => item.id === id)) return
    this.pendingDismissals.add(id)
    const cleanup = this.requestCancel(id)
    if (!cleanup) {
      this.completeDismiss(id)
      return
    }
    void cleanup.then(() => this.completeDismiss(id), () => undefined)
  }

  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }

  getSnapshot(machineId?: string): FileTransferStoreSnapshot {
    if (!machineId && this.cachedSnapshot && this.cachedVersion === this.version) return this.cachedSnapshot
    if (machineId) {
      const cached = this.cachedMachineSnapshots.get(machineId)
      if (cached) return cached
    }
    const transfers = machineId ? this.transfers.filter((item) => item.machineId === machineId) : this.transfers
    const snapshot = { transfers, hasActiveTransfers: transfers.some((item) => item.status === 'pending' || item.status === 'transferring') }
    if (machineId) {
      this.cachedMachineSnapshots.set(machineId, snapshot)
    } else {
      this.cachedSnapshot = snapshot
      this.cachedVersion = this.version
    }
    return snapshot
  }

  private async runDownload(id: string): Promise<void> {
    const transfer = this.requiredTransfer(id, 'download')
    if (!transfer.machineId) throw new Error('download machine is missing')
    const task = this.beginAttempt(id)
    try {
      const session = await this.session(transfer.machineId, task.cancel.signal)
      task.session = session
      this.assertCurrentAttempt(id, task)
      let resumeOffset = 0
      if (Capacitor.isNativePlatform()) {
        const persisted = await NativeFilePicker.getDownloadResumeOffset({
          machineId: transfer.machineId,
          remotePath: transfer.filePath ?? '',
          totalSize: transfer.totalSize,
        })
        resumeOffset = persisted.offset
        if (resumeOffset > 0 && !transfer.remoteModifiedAtUnixNano) {
          await NativeFilePicker.discardDownloadPartial({
            machineId: transfer.machineId,
            remotePath: transfer.filePath ?? '',
            totalSize: transfer.totalSize,
          })
          resumeOffset = 0
        }
      } else {
        const chunks = this.downloadChunks.get(id) ?? []
        const bufferedBytes = chunks.reduce((total, chunk) => total + chunk.byteLength, 0)
        resumeOffset = bufferedBytes === transfer.transferredSize ? bufferedBytes : 0
        if (resumeOffset === 0) this.downloadChunks.set(id, [])
      }
      const openDownload = (offset: number, validateSource: boolean) => session.execute(command('fileDownloadOpen', create(AnyTTYApiFile.FileDownloadOpenCommandSchema, {
        path: transfer.filePath ?? '', offset: BigInt(offset),
        expectedSize: validateSource ? BigInt(transfer.totalSize) : 0n,
        expectedModifiedAtUnixNano: validateSource ? transfer.remoteModifiedAtUnixNano ?? 0n : 0n,
      })), { signal: task.cancel.signal })
      let opened
      try {
        opened = await openDownload(resumeOffset, resumeOffset > 0 && transfer.remoteModifiedAtUnixNano !== undefined)
      } catch (error) {
        if (!Capacitor.isNativePlatform() || resumeOffset === 0 || !isStaleDownloadError(error)) throw error
        await NativeFilePicker.discardDownloadPartial({
          machineId: transfer.machineId,
          remotePath: transfer.filePath ?? '',
          totalSize: transfer.totalSize,
        })
        resumeOffset = 0
        this.update(id, { transferredSize: 0, remoteModifiedAtUnixNano: undefined, updatedAt: Date.now() })
        opened = await openDownload(0, false)
      }
      if (opened.result.case !== 'fileTransferOpen' || !opened.result.value.transfer) throw new Error('download returned no transfer resource')
      const remote = opened.result.value.transfer
      if (Number(remote.offset) !== resumeOffset) {
        if (Number(remote.offset) !== 0) throw new Error('download resume offset was not accepted')
        if (Capacitor.isNativePlatform()) {
          await NativeFilePicker.discardDownloadPartial({ machineId: transfer.machineId, remotePath: transfer.filePath ?? '', totalSize: transfer.totalSize })
        } else {
          this.downloadChunks.set(id, [])
        }
      }
      const resource = remote.resource
      if (!resource) throw new Error('download returned no resource handle')
      task.resource = resource
      this.forgetDetachedCleanupOwner(id)
      this.assertCurrentAttempt(id, task)
      const stream = await awaitAbortable(
        session.openResourceStream(resource, { signal: task.cancel.signal }),
        task.cancel.signal,
        (late) => { void late.close().catch(() => undefined) },
      )
      task.stream = stream
      this.assertCurrentAttempt(id, task)
      this.update(id, {
        status: 'transferring', totalSize: Number(remote.size), transferredSize: Number(remote.offset),
        remoteModifiedAtUnixNano: remote.modifiedAtUnixNano, updatedAt: Date.now(),
      })
      if (Capacitor.isNativePlatform()) {
        const saved = await receiveNativeDownload(
          stream,
          remote,
          {
            machineId: transfer.machineId,
            remotePath: transfer.filePath ?? '',
            totalSize: Number(remote.size),
            name: transfer.name,
            mimeType: 'application/octet-stream',
          },
          task.cancel.signal,
          (received) => this.progress(id, received, task),
        )
        this.assertCurrentAttempt(id, task)
        if (saved.bytes !== Number(remote.size)) throw new Error('Android download persistence size mismatch')
        this.update(id, {
          status: 'completed', transferredSize: Number(remote.size), savedUri: saved.uri, savedPath: saved.path,
          updatedAt: Date.now(), bytesPerSecond: 0,
        })
      } else {
        const retained = this.downloadChunks.get(id) ?? []
        const blob = await receiveDownload(stream, remote, retained, task.cancel.signal, (received) => this.progress(id, received, task))
        this.assertCurrentAttempt(id, task)
        const url = URL.createObjectURL(blob)
        const anchor = document.createElement('a')
        anchor.href = url
        anchor.download = transfer.name
        anchor.click()
        setTimeout(() => URL.revokeObjectURL(url), 60_000)
        this.update(id, { status: 'completed', transferredSize: Number(remote.size), savedUri: url, updatedAt: Date.now(), bytesPerSecond: 0 })
      }
      this.downloadChunks.delete(id)
    } finally {
      task.markReadyForClose()
      await this.closeTask(id, task)
    }
  }

  private async runUpload(id: string): Promise<void> {
    const transfer = this.requiredTransfer(id, 'upload')
    if (!transfer.localUri || !transfer.targetDir) throw new Error('upload local URI is missing')
    const task = this.beginAttempt(id)
    try {
      const session = await this.session(transfer.machineId, task.cancel.signal)
      task.session = session
      this.assertCurrentAttempt(id, task)
      const target = `${transfer.targetDir.replace(/\/$/, '')}/${transfer.name}`
      const opened = await session.execute(command('fileUploadOpen', create(AnyTTYApiFile.FileUploadOpenCommandSchema, {
        path: target, size: BigInt(transfer.totalSize), overwrite: true,
        resume: transfer.uploadResumeToken ? create(AnyTTYApiFile.FileUploadResumeHandleSchema, { opaqueToken: transfer.uploadResumeToken }) : undefined,
      })), { signal: task.cancel.signal })
      if (opened.result.case !== 'fileTransferOpen' || !opened.result.value.transfer) throw new Error('upload returned no transfer resource')
      const remote = opened.result.value.transfer
      const resource = remote.resource
      if (!resource) throw new Error('upload returned no resource handle')
      task.resource = resource
      task.uploadResume = remote.resume
      this.forgetDetachedCleanupOwner(id)
      this.assertCurrentAttempt(id, task)
      const stream = await awaitAbortable(
        session.openResourceStream(resource, { initialUploadOffset: remote.offset, signal: task.cancel.signal }),
        task.cancel.signal,
        (late) => { void late.close().catch(() => undefined) },
      )
      task.stream = stream
      this.assertCurrentAttempt(id, task)
      this.update(id, {
        status: 'transferring', transferredSize: Number(remote.offset),
        uploadResumeToken: remote.resume?.opaqueToken.slice(), updatedAt: Date.now(),
      })
      if (Capacitor.isNativePlatform()) {
        await sendNativeUpload(stream, remote, transfer.localUri, task.cancel.signal, (sent) => this.progress(id, sent, task))
      } else {
        const response = await fetch(Capacitor.convertFileSrc(transfer.localUri), { signal: task.cancel.signal })
        this.assertCurrentAttempt(id, task)
        if (!response.ok) throw new Error(`local upload file could not be read (${response.status})`)
        const blob = await response.blob()
        this.assertCurrentAttempt(id, task)
        await sendUpload(stream, remote, blob, task.cancel.signal, (sent) => this.progress(id, sent, task))
      }
      this.assertCurrentAttempt(id, task)
      this.update(id, { status: 'completed', transferredSize: transfer.totalSize, savedPath: target, updatedAt: Date.now(), bytesPerSecond: 0 })
    } finally {
      task.markReadyForClose()
      await this.closeTask(id, task)
    }
  }

  private async cancelRemote(task: ActiveTransfer, signal?: AbortSignal): Promise<CleanupConfirmation> {
    if (!task.session || (!task.resource && !task.uploadResume)) {
      return { confirmed: false, error: new Error('remote file transfer cancellation has no credential') }
    }
    const useSessionResource = task.resource && sameSession(task.resource.session, task.session.stamp)
    let cancelError: Error | undefined
    try {
      const result = await task.session.execute(command('fileTransferCancel', create(AnyTTYApiFile.FileTransferCancelCommandSchema, {
        transfer: useSessionResource ? task.resource : undefined,
        uploadResume: useSessionResource ? undefined : task.uploadResume,
      })), signal ? { signal } : undefined)
      if (result.result.case === 'fileTransferCancel') {
        // Download 的 false 表示 daemon 已处理命令且 resource 已不存在；upload 临时文件仍要求明确 cancelled=true。
        if (result.result.value.cancelled || task.direction === 'download') return { confirmed: true }
        return { confirmed: false, error: new Error('daemon did not confirm upload cancellation') }
      }
      cancelError = new Error('daemon returned no file transfer cancellation result')
    } catch (error) {
      cancelError = error instanceof Error ? error : new Error(String(error))
    }
    const released = await this.releaseCancelledDownload(task, useSessionResource)
    if (released.confirmed) return released
    return {
      confirmed: false,
      error: new Error(`file transfer cancel failed: ${cancelError.message}; release failed: ${released.error?.message ?? 'unavailable'}`),
    }
  }

  private async releaseCancelledDownload(task: ActiveTransfer, useSessionResource: boolean | undefined): Promise<CleanupConfirmation> {
    if (task.direction !== 'download' || !useSessionResource || !task.session || !task.resource) {
      return { confirmed: false, error: new Error('download resource is not bound to the current session') }
    }
    try {
      await this.releaseRemote(task.session, task.resource)
      return { confirmed: true }
    } catch (error) {
      return { confirmed: false, error: error instanceof Error ? error : new Error(String(error)) }
    }
  }

  private async releaseRemote(session: ProtoClientSession, resource: NonNullable<ActiveTransfer['resource']>): Promise<void> {
    await session.execute(command('releaseResource', create(AnyTTYApiApplication.ReleaseResourceCommandSchema, { resource })))
  }

  private closeTask(id: string, task: ActiveTransfer): Promise<void> {
    if (task.storeEpoch !== this.storeEpoch) return this.closeDiscardedTask(task)
    const existing = this.taskTeardowns.get(task)
    if (existing) {
      task.teardown ??= existing
      return existing
    }
    const teardown = this.finishCloseTask(id, task)
    this.taskTeardowns.set(task, teardown)
    task.teardown = teardown
    this.pendingTeardowns.set(id, teardown)
    void teardown.then(
      () => {
        if (task.teardown === teardown) task.teardown = undefined
        this.clearPendingTeardown(id, teardown)
        if (this.failedCleanupOwners.get(id) !== task && this.detachedCleanupOwners.get(id) !== task) this.taskOwners.delete(task)
      },
      () => {
        if (task.teardown === teardown) task.teardown = undefined
      },
    )
    return teardown
  }

  private async finishCloseTask(id: string, task: ActiveTransfer): Promise<void> {
    await task.readyForClose
    if (task.storeEpoch !== this.storeEpoch) {
      await this.closeDiscardedTask(task)
      return
    }
    let cleanupError: unknown
    let destructiveConfirmed = false
    let destructiveAttempted = false
    if (!task.destructiveCancel && task.session && task.resource) {
      try {
        await this.releaseRemote(task.session, task.resource)
      } catch (error) {
        cleanupError = error
      }
    }
    if (task.destructiveCancel && task.session && task.resource) {
      destructiveAttempted = true
      const cancellation = await this.cancelRemote(task)
      destructiveConfirmed = cancellation.confirmed
      cleanupError = cancellation.confirmed ? undefined : cancellation.error
    }
    await task.stream?.close().catch(() => undefined)
    if (task.storeEpoch !== this.storeEpoch) {
      await this.closeDiscardedTask(task)
      return
    }
    if (task.destructiveCancel && !destructiveAttempted && task.session && task.resource) {
      const cancellation = await this.cancelRemote(task)
      destructiveConfirmed = cancellation.confirmed
      cleanupError = cancellation.confirmed ? undefined : cancellation.error
    }
    if (task.destructiveCancel && !task.resource && !task.uploadResume) {
      // FileUploadOpen/FileDownloadOpen 尚未交付 resource 时，binding cancelled-execute owner 负责销毁迟到结果。
      // Store 不能要求一个尚未产生的 resume credential，也不能建立第二套 operation cleanup。
      destructiveConfirmed = true
      cleanupError = undefined
    }
    if (cleanupError) {
      if (this.active.get(id) === task) this.active.delete(id)
      this.failedCleanupOwners.set(id, task)
      throw cleanupError
    }
    this.failedCleanupOwners.delete(id)
    await task.session?.close().catch(() => undefined)
    if (task.storeEpoch !== this.storeEpoch) {
      await this.closeDiscardedTask(task)
      return
    }
    task.session = undefined
    task.stream = undefined
    if (task.destructiveCancel && !destructiveConfirmed) {
      try {
        const cancellation = await this.cancelWithFreshSession(task)
        if (task.storeEpoch !== this.storeEpoch) {
          await this.closeDiscardedTask(task)
          return
        }
        destructiveConfirmed = cancellation.confirmed
        cleanupError = cancellation.error
      } catch (error) {
        if (task.storeEpoch !== this.storeEpoch) {
          await this.closeDiscardedTask(task)
          return
        }
        if (this.active.get(id) === task) this.active.delete(id)
        this.failedCleanupOwners.set(id, task)
        throw error
      }
      if (!destructiveConfirmed) {
        if (this.active.get(id) === task) this.active.delete(id)
        this.failedCleanupOwners.set(id, task)
        throw cleanupError ?? new Error('remote file transfer cancellation was not confirmed')
      }
    }
    const transfer = this.transfers.find((item) => item.id === id)
    if (!task.destructiveCancel && task.direction === 'upload' && transfer?.status !== 'completed' && transfer?.status !== 'cancelled' && task.uploadResume) {
      this.detachedCleanupOwners.set(id, task)
    } else {
      this.detachedCleanupOwners.delete(id)
    }
    if (this.active.get(id) === task) this.active.delete(id)
  }

  private retryDestructiveCleanup(id: string, task: ActiveTransfer): Promise<void> {
    const existing = this.destructiveRetries.get(id)
    if (existing) return existing
    const retry = this.finishDestructiveCleanup(id, task)
    this.destructiveRetries.set(id, retry)
    task.teardown = retry
    this.pendingTeardowns.set(id, retry)
    void retry.then(
      () => {
        if (task.teardown === retry) task.teardown = undefined
        if (this.destructiveRetries.get(id) === retry) this.destructiveRetries.delete(id)
        this.clearPendingTeardown(id, retry)
        this.taskOwners.delete(task)
      },
      () => {
        if (task.teardown === retry) task.teardown = undefined
        if (this.destructiveRetries.get(id) === retry) this.destructiveRetries.delete(id)
      },
    )
    return retry
  }

  private async finishDestructiveCleanup(id: string, task: ActiveTransfer): Promise<void> {
    if (task.storeEpoch !== this.storeEpoch) {
      await this.closeDiscardedTask(task)
      return
    }
    const cancellation = await this.cancelWithFreshSession(task)
    if (task.storeEpoch !== this.storeEpoch) {
      await this.closeDiscardedTask(task)
      return
    }
    if (!cancellation.confirmed) throw cancellation.error ?? new Error('remote file transfer cancellation was not confirmed')
    this.failedCleanupOwners.delete(id)
    this.detachedCleanupOwners.delete(id)
    this.progressSamples.delete(id)
    await task.stream?.close().catch(() => undefined)
	await task.session?.close().catch(() => undefined)
  }

  private cancelWithFreshSession(task: ActiveTransfer): Promise<CleanupConfirmation> {
    if (task.freshCleanup) return task.freshCleanup.completion
    const cancel = new AbortController()
    let completion!: Promise<CleanupConfirmation>
    completion = this.performFreshSessionCancel(task, cancel).finally(() => {
      if (task.freshCleanup?.completion === completion) task.freshCleanup = undefined
    })
    task.freshCleanup = { cancel, completion }
    return completion
  }

  private async performFreshSessionCancel(task: ActiveTransfer, controller: AbortController): Promise<CleanupConfirmation> {
    let priorError: Error | undefined
    if (task.session?.isAlive()) {
      const cancellation = await awaitAbortable(
        this.cancelRemote(task, controller.signal),
        controller.signal,
        () => undefined,
      )
      if (cancellation.confirmed) return cancellation
      priorError = cancellation.error
      await task.session.close().catch(() => undefined)
      task.session = undefined
    }
    if (controller.signal.aborted || task.storeEpoch !== this.storeEpoch) {
      return { confirmed: false, error: abortError(controller.signal) }
    }
    if (!task.uploadResume) {
      return { confirmed: false, error: priorError ?? new Error('fresh-session cancellation has no upload resume credential') }
    }
    try {
      const session = await this.session(task.machineId, controller.signal)
      if (controller.signal.aborted || task.storeEpoch !== this.storeEpoch) {
        await session.close().catch(() => undefined)
        return { confirmed: false, error: abortError(controller.signal) }
      }
      task.session = session
      const cancellation = await awaitAbortable(
        this.cancelRemote(task, controller.signal),
        controller.signal,
        () => undefined,
      )
      if (cancellation.confirmed || !priorError) return cancellation
      return { confirmed: false, error: new Error(`existing-session cleanup failed: ${priorError.message}; fresh-session cleanup failed: ${cancellation.error?.message ?? 'unavailable'}`) }
    } catch (error) {
      const freshError = error instanceof Error ? error : new Error(String(error))
      return { confirmed: false, error: new Error(`existing-session cleanup failed: ${priorError?.message ?? 'unavailable'}; fresh-session resolution failed: ${freshError.message}`) }
    }
  }

  private completeDismiss(id: string): void {
    if (!this.pendingDismissals.delete(id)) return
    this.transfers = this.transfers.filter((item) => item.id !== id)
    this.transitionEpochs.delete(id)
    this.resumeTransitions.delete(id)
    this.pendingTeardowns.delete(id)
    const failedOwner = this.failedCleanupOwners.get(id)
    const detachedOwner = this.detachedCleanupOwners.get(id)
    this.failedCleanupOwners.delete(id)
    this.detachedCleanupOwners.delete(id)
    if (failedOwner) this.taskOwners.delete(failedOwner)
    if (detachedOwner) this.taskOwners.delete(detachedOwner)
    this.progressSamples.delete(id)
    this.notify()
  }

  private clearPendingTeardown(id: string, teardown: Promise<void>): void {
    if (this.pendingTeardowns.get(id) === teardown) this.pendingTeardowns.delete(id)
  }

  private beginAttempt(id: string): ActiveTransfer {
    if (this.active.has(id)) throw new Error('transfer attempt is already active')
    const transfer = this.transfers.find((item) => item.id === id)
    if (!transfer?.machineId) throw new Error('transfer machine is unavailable')
    let markReadyForClose!: () => void
    const readyForClose = new Promise<void>((resolve) => { markReadyForClose = resolve })
    const task: ActiveTransfer = {
      epoch: this.transitionEpochs.get(id) ?? 0,
      storeEpoch: this.storeEpoch,
      machineId: transfer.machineId,
      direction: transfer.direction,
      cancel: new AbortController(),
      destructiveCancel: false,
      readyForClose,
      markReadyForClose,
    }
    this.active.set(id, task)
    this.taskOwners.add(task)
    return task
  }

  private forgetDetachedCleanupOwner(id: string): void {
    const owner = this.detachedCleanupOwners.get(id)
    this.detachedCleanupOwners.delete(id)
    if (owner) this.taskOwners.delete(owner)
  }

  private assertCurrentAttempt(id: string, task: ActiveTransfer): void {
    const transfer = this.transfers.find((item) => item.id === id)
    if (task.storeEpoch !== this.storeEpoch || this.active.get(id) !== task || (this.transitionEpochs.get(id) ?? 0) !== task.epoch || task.cancel.signal.aborted || !transfer || transfer.status === 'paused' || transfer.status === 'cancelled') {
      throw new DOMException('Aborted', 'AbortError')
    }
  }

  private advanceTransition(id: string): number {
    const next = (this.transitionEpochs.get(id) ?? 0) + 1
    this.transitionEpochs.set(id, next)
    return next
  }

  private async session(machineId: string | undefined, signal: AbortSignal): Promise<ProtoClientSession> {
    if (!machineId || !this.resolver) throw new Error('Connect this machine before starting a transfer')
    const session = await awaitAbortable(
      this.resolver(machineId, signal),
      signal,
      (late) => { void late?.close().catch(() => undefined) },
    )
    if (!session?.isAlive()) throw new Error('Go client session is unavailable')
    return session
  }

  private requiredTransfer(id: string, direction: TransferInfo['direction']): TransferInfo {
    const transfer = this.transfers.find((item) => item.id === id)
    if (!transfer || transfer.direction !== direction) throw new Error('transfer is unavailable')
    return transfer
  }

  private progress(id: string, transferredSize: number, task: ActiveTransfer): void {
    if (task.storeEpoch !== this.storeEpoch || this.active.get(id) !== task) return
    const current = this.transfers.find((item) => item.id === id)
    if (!current) return
    const now = Date.now()
    const previous = this.progressSamples.get(id) ?? {
      at: current.updatedAt ?? now,
      bytes: current.transferredSize,
      speed: current.bytesPerSecond ?? 0,
      notifiedAt: current.updatedAt ?? 0,
    }
    const elapsed = now - previous.at
    let speed = previous.speed
    let sampleAt = previous.at
    let sampleBytes = previous.bytes
    if (elapsed >= 200) {
      const instant = Math.max(0, transferredSize - previous.bytes) * 1000 / elapsed
      speed = previous.speed > 0 ? previous.speed * 0.7 + instant * 0.3 : instant
      sampleAt = now
      sampleBytes = transferredSize
    }
    const shouldNotify = now - previous.notifiedAt >= 200 || transferredSize >= current.totalSize
    this.progressSamples.set(id, { at: sampleAt, bytes: sampleBytes, speed, notifiedAt: shouldNotify ? now : previous.notifiedAt })
    this.update(id, { transferredSize, bytesPerSecond: speed, updatedAt: now }, shouldNotify)
  }

  private fail(id: string, error: unknown, storeEpoch: number): void {
    if (storeEpoch !== this.storeEpoch) return
    if (error instanceof DOMException && error.name === 'AbortError') return
    const current = this.transfers.find((item) => item.id === id)
    if (current?.status === 'cancelled' && !this.failedCleanupOwners.has(id)) return
    this.update(id, { status: 'failed', pausedByUser: current?.pausedByUser === true, error: error instanceof Error ? error.message : String(error), updatedAt: Date.now(), bytesPerSecond: 0 })
  }

  private upsert(info: TransferInfo): void {
    const index = this.transfers.findIndex((item) => item.id === info.id)
    this.transfers = index < 0 ? [...this.transfers, info] : this.transfers.map((item) => item.id === info.id ? info : item)
    this.notify()
  }

  private update(id: string, patch: Partial<TransferInfo>, notify = true, storeEpoch = this.storeEpoch): void {
    if (storeEpoch !== this.storeEpoch || !this.transfers.some((item) => item.id === id)) return
    this.transfers = this.transfers.map((item) => item.id === id ? { ...item, ...patch } : item)
    if (notify) this.notify()
  }

  private notify(): void {
    this.version += 1
    this.cachedSnapshot = null
    this.cachedMachineSnapshots.clear()
    persistTransfers(this.storage, this.transfers)
    for (const listener of this.listeners) listener()
  }

  private publishEmptyResetSnapshot(): void {
    this.version += 1
    this.cachedSnapshot = null
    this.cachedMachineSnapshots.clear()
    removePersistedTransfers(this.storage)
    for (const listener of this.listeners) listener()
  }

  private async closeDiscardedTask(task: ActiveTransfer): Promise<void> {
    task.cancel.abort()
    task.freshCleanup?.cancel.abort()
    const stream = task.stream
    const session = task.session
    task.stream = undefined
    task.session = undefined
    task.resource = undefined
    task.uploadResume = undefined
    await Promise.allSettled([
      stream?.close(),
      session?.close(),
    ])
  }
}

type NativeSavedDownload = { uri: string; path: string; bytes: number; sha256: string }

const transferStorageKey = 'anytty.file-transfers.v2'

function transferStorage(): Storage | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage
  } catch {
    return null
  }
}

function loadPersistedTransfers(storage: Storage | null): TransferInfo[] {
  if (!storage) return []
  try {
    const value = JSON.parse(storage.getItem(transferStorageKey) ?? '[]') as Array<Record<string, unknown>>
    if (!Array.isArray(value)) return []
    return value.flatMap((item): TransferInfo[] => {
      if (typeof item.id !== 'string' || typeof item.name !== 'string' || (item.direction !== 'download' && item.direction !== 'upload')) return []
      if (typeof item.totalSize !== 'number' || typeof item.transferredSize !== 'number' || typeof item.startedAt !== 'number') return []
      const storedStatus = isTransferStatus(item.status) ? item.status : 'failed'
      const interrupted = storedStatus === 'pending' || storedStatus === 'transferring'
      return [{
        id: item.id,
        machineId: typeof item.machineId === 'string' ? item.machineId : undefined,
        name: item.name,
        direction: item.direction,
        totalSize: item.totalSize,
        transferredSize: item.transferredSize,
        status: interrupted ? 'failed' : storedStatus,
        startedAt: item.startedAt,
        updatedAt: typeof item.updatedAt === 'number' ? item.updatedAt : item.startedAt,
        bytesPerSecond: 0,
        error: interrupted ? 'Transfer was interrupted and can be resumed' : typeof item.error === 'string' ? item.error : undefined,
        filePath: typeof item.filePath === 'string' ? item.filePath : undefined,
        localUri: typeof item.localUri === 'string' ? item.localUri : undefined,
        targetDir: typeof item.targetDir === 'string' ? item.targetDir : undefined,
        savedPath: typeof item.savedPath === 'string' ? item.savedPath : undefined,
        savedUri: typeof item.savedUri === 'string' ? item.savedUri : undefined,
        uploadResumeToken: typeof item.uploadResumeToken === 'string' ? base64Bytes(item.uploadResumeToken) : undefined,
        remoteModifiedAtUnixNano: typeof item.remoteModifiedAtUnixNano === 'string' ? BigInt(item.remoteModifiedAtUnixNano) : undefined,
        pausedByUser: interrupted ? false : item.pausedByUser === true,
      }]
    })
  } catch {
    return []
  }
}

function isTransferStatus(value: unknown): value is TransferStatus {
  return value === 'pending' || value === 'transferring' || value === 'paused' || value === 'completed' || value === 'failed' || value === 'cancelled' || value === 'missing'
}

function isStaleDownloadError(error: unknown): boolean {
  const message = (error instanceof Error ? error.message : String(error)).toLowerCase()
  return message.includes('stale download source') || message.includes('invalid download offset')
}

function persistTransfers(storage: Storage | null, transfers: TransferInfo[]): void {
  if (!storage) return
  try {
    storage.setItem(transferStorageKey, JSON.stringify(transfers.map((transfer) => ({
      ...transfer,
      uploadResumeToken: transfer.uploadResumeToken ? bytesBase64(transfer.uploadResumeToken) : undefined,
      remoteModifiedAtUnixNano: transfer.remoteModifiedAtUnixNano?.toString(),
    }))))
  } catch {
    // A full/disabled WebView storage must not break an active transfer.
  }
}

function removePersistedTransfers(storage: Storage | null): void {
  if (!storage) return
  try {
    storage.removeItem(transferStorageKey)
  } catch {
    // A disabled WebView storage must not block local recovery.
  }
}

async function receiveNativeDownload(
  stream: ProtoResourceStream,
  transfer: AnyTTYApiFile.FileTransferHandle,
  target: { machineId: string; remotePath: string; totalSize: number; name: string; mimeType: string },
  signal: AbortSignal,
  progress: (bytes: number) => void,
): Promise<NativeSavedDownload> {
  let offset = Number(transfer.offset)
  let persistedOffset = offset
  let credit = 0
  let pending: Uint8Array[] = []
  let settled = false
  let acceptingFrames = true
  let chain = Promise.resolve()

  return await new Promise<NativeSavedDownload>((resolve, reject) => {
    let subscription: { close(): void } | null = null
    let closeSubscription: { close(): void } | null = null
    const detachStream = () => {
      subscription?.close()
      closeSubscription?.close()
      subscription = null
      closeSubscription = null
    }
    const cleanup = () => {
      acceptingFrames = false
      detachStream()
      signal.removeEventListener('abort', abort)
    }
    const fail = (error: unknown) => {
      if (settled) return
      settled = true
      cleanup()
      reject(error)
    }
    const succeed = (saved: NativeSavedDownload) => {
      if (settled) return
      settled = true
      cleanup()
      resolve(saved)
    }
    const abort = () => {
      if (settled || !acceptingFrames) return
      acceptingFrames = false
      detachStream()
      signal.removeEventListener('abort', abort)
      const error = new DOMException('Aborted', 'AbortError')
      void chain.then(() => fail(error), () => fail(error))
    }
    const flush = async (acknowledge: boolean) => {
      if (pending.length === 0) return
      if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
      const bytes = concatBytes(pending)
      pending = []
      const windowBytes = credit
      credit = 0
      const result = await NativeFilePicker.appendDownloadPartial({
        machineId: target.machineId,
        remotePath: target.remotePath,
        totalSize: target.totalSize,
        offset: persistedOffset,
        dataBase64: bytesBase64(bytes),
      })
      const expectedOffset = persistedOffset + bytes.byteLength
      if (result.offset !== expectedOffset) throw new Error('Android download partial offset mismatch')
      persistedOffset = result.offset
      if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
      progress(persistedOffset)
      if (acknowledge && windowBytes > 0) {
        await stream.send(AnyTTYClientBinding.ResourceStreamFrameType.FILE_ACK, encodeFileTransferAckPayload({ offset: persistedOffset, windowBytes }))
      }
    }
    const enqueue = (operation: () => Promise<void>) => {
      chain = chain.then(operation)
      void chain.catch(fail)
    }

    signal.addEventListener('abort', abort, { once: true })
    subscription = stream.subscribe((type, payload) => {
      if (settled || !acceptingFrames) return
      const retainedPayload = payload.slice()
      if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_FINISH) {
        acceptingFrames = false
        detachStream()
        signal.removeEventListener('abort', abort)
      }
      enqueue(async () => {
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_DATA) {
          const data = decodeFileTransferDataPayload(retainedPayload)
          if (data.offset !== offset || data.data.byteLength === 0 || data.data.byteLength > transfer.chunkBytes) throw new Error('download chunk is invalid')
          pending.push(data.data)
          offset += data.data.byteLength
          credit += data.data.byteLength
          if (credit >= Number(transfer.windowBytes)) await flush(true)
          return
        }
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_FINISH) {
          const finish = decodeFileTransferFinishPayload(retainedPayload)
          if (finish.size !== Number(transfer.size) || offset !== finish.size) throw new Error('download completed with the wrong size')
          await flush(false)
          const saved = await NativeFilePicker.commitDownloadPartial({
            machineId: target.machineId,
            remotePath: target.remotePath,
            totalSize: target.totalSize,
            name: target.name,
            mimeType: target.mimeType,
            sha256Base64: bytesBase64(finish.sha256),
          })
          succeed(saved)
          return
        }
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.ERROR) {
          throw new Error(decodeFileStreamErrorPayload(retainedPayload))
        }
      })
    })
    if (!acceptingFrames) subscription.close()
    closeSubscription = stream.subscribeClosed((error) => {
      if (acceptingFrames) fail(error)
    })
    if (!acceptingFrames) closeSubscription.close()
  })
}

async function receiveDownload(
  stream: ProtoResourceStream,
  transfer: AnyTTYApiFile.FileTransferHandle,
  chunks: Uint8Array[],
  signal: AbortSignal,
  progress: (bytes: number) => void,
): Promise<Blob> {
  let offset = Number(transfer.offset)
  let credit = 0
  return await new Promise<Blob>((resolve, reject) => {
    let closeSubscription: { close(): void } | null = null
    const cleanup = () => {
      subscription.close()
      closeSubscription?.close()
      signal.removeEventListener('abort', abort)
    }
    const abort = () => {
      cleanup()
      reject(new DOMException('Aborted', 'AbortError'))
    }
    signal.addEventListener('abort', abort, { once: true })
    const subscription = stream.subscribe((type, payload) => {
      try {
        if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_DATA) {
          const data = decodeFileTransferDataPayload(payload)
          if (data.offset !== offset || data.data.byteLength === 0 || data.data.byteLength > transfer.chunkBytes) throw new Error('download chunk is invalid')
          chunks.push(data.data)
          offset += data.data.byteLength
          credit += data.data.byteLength
          progress(offset)
          if (credit >= Number(transfer.windowBytes)) {
            const windowBytes = credit
            credit = 0
            void stream.send(AnyTTYClientBinding.ResourceStreamFrameType.FILE_ACK, encodeFileTransferAckPayload({ offset, windowBytes })).catch(reject)
          }
        } else if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_FINISH) {
          const finish = decodeFileTransferFinishPayload(payload)
          if (finish.size !== Number(transfer.size) || offset !== finish.size) throw new Error('download completed with the wrong size')
          cleanup()
          const blob = new Blob(chunks.map((chunk) => chunk.buffer.slice(chunk.byteOffset, chunk.byteOffset + chunk.byteLength) as ArrayBuffer))
          void verifyBlobDigest(blob, finish.sha256).then(() => resolve(blob), reject)
        } else if (type === AnyTTYClientBinding.ResourceStreamFrameType.ERROR) {
          throw new Error(decodeFileStreamErrorPayload(payload))
        }
      } catch (error) {
        cleanup()
        reject(error)
      }
    })
    closeSubscription = stream.subscribeClosed((error) => {
      cleanup()
      reject(error)
    })
  })
}

async function sendUpload(
  stream: ProtoResourceStream,
  transfer: AnyTTYApiFile.FileTransferHandle,
  blob: Blob,
  signal: AbortSignal,
  progress: (bytes: number) => void,
): Promise<void> {
  const source: UploadSource = {
    size: blob.size,
    async read(offset, length) { return new Uint8Array(await blob.slice(offset, offset + length).arrayBuffer()) },
    async digest() { return new Uint8Array(await crypto.subtle.digest('SHA-256', await blob.arrayBuffer())) },
    async close() {},
  }
  await sendUploadSource(stream, transfer, source, signal, progress)
}

async function sendNativeUpload(
  stream: ProtoResourceStream,
  transfer: AnyTTYApiFile.FileTransferHandle,
  contentUri: string,
  signal: AbortSignal,
  progress: (bytes: number) => void,
): Promise<void> {
  const opened = await NativeFilePicker.openUploadSource({
    contentUri,
    offset: Number(transfer.offset),
    totalSize: Number(transfer.size),
  })
  if (opened.offset !== Number(transfer.offset)) {
    await NativeFilePicker.closeUploadSource({ handle: opened.handle }).catch(() => undefined)
    throw new Error('Android upload source offset mismatch')
  }
  let finished = false
  const source: UploadSource = {
    size: Number(transfer.size),
    async read(offset, length) {
      const result = await NativeFilePicker.readUploadSource({ handle: opened.handle, length })
      const data = base64Bytes(result.dataBase64)
      if (result.offset !== offset + data.byteLength || data.byteLength === 0) throw new Error('Android upload source returned an invalid chunk')
      return data
    },
    async digest() {
      const result = await NativeFilePicker.finishUploadSource({ handle: opened.handle })
      finished = true
      return base64Bytes(result.sha256Base64)
    },
    async close() {
      if (!finished) await NativeFilePicker.closeUploadSource({ handle: opened.handle }).catch(() => undefined)
    },
  }
  await sendUploadSource(stream, transfer, source, signal, progress)
}

type UploadSource = {
  size: number
  read(offset: number, length: number): Promise<Uint8Array>
  digest(): Promise<Uint8Array>
  close(): Promise<void>
}

async function sendUploadSource(
  stream: ProtoResourceStream,
  transfer: AnyTTYApiFile.FileTransferHandle,
  source: UploadSource,
  signal: AbortSignal,
  progress: (bytes: number) => void,
): Promise<void> {
  let offset = Number(transfer.offset)
  let acknowledgedOffset = offset
  let credit = Number(transfer.windowBytes)
  let wake: (() => void) | null = null
  let terminalError: unknown
  let resultResolve: (() => void) | null = null
  let resultReject: ((error: unknown) => void) | null = null
  const result = new Promise<void>((resolve, reject) => { resultResolve = resolve; resultReject = reject })
  void result.catch(() => undefined)
  const fail = (error: unknown) => {
    if (terminalError) return
    terminalError = error
    wake?.()
    wake = null
    resultReject?.(error)
  }
  const subscription = stream.subscribe((type, payload) => {
    if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_ACK) {
      const ack = decodeFileTransferAckPayload(payload)
      const acknowledgedBytes = ack.offset - acknowledgedOffset
      if (ack.offset <= acknowledgedOffset || ack.offset > offset || ack.windowBytes !== acknowledgedBytes) {
        fail(new Error('upload ACK offset mismatch'))
        return
      }
      acknowledgedOffset = ack.offset
      credit += ack.windowBytes
      wake?.()
      wake = null
    } else if (type === AnyTTYClientBinding.ResourceStreamFrameType.FILE_RESULT) {
      const completed = decodeFileTransferResultPayload(payload)
      if (completed.size !== source.size) {
        fail(new Error('upload completed with the wrong size'))
        return
      }
      resultResolve?.()
    } else if (type === AnyTTYClientBinding.ResourceStreamFrameType.ERROR) {
      fail(new Error(decodeFileStreamErrorPayload(payload)))
    }
  })
  const closeSubscription = stream.subscribeClosed(fail)
  const abort = () => fail(new DOMException('Aborted', 'AbortError'))
  signal.addEventListener('abort', abort, { once: true })
  try {
    while (offset < source.size) {
      if (signal.aborted) throw new DOMException('Aborted', 'AbortError')
      if (terminalError) throw terminalError
      if (credit <= 0) await new Promise<void>((resolve) => { wake = resolve })
      if (terminalError) throw terminalError
      const length = Math.min(transfer.chunkBytes, credit, source.size - offset)
      const data = await source.read(offset, length)
      if (data.byteLength !== length) throw new Error('upload source returned the wrong chunk length')
      const chunkOffset = offset
      offset += data.byteLength
      credit -= data.byteLength
      await stream.send(AnyTTYClientBinding.ResourceStreamFrameType.FILE_DATA, encodeFileTransferDataPayload({ offset: chunkOffset, data }))
      progress(offset)
    }
    const digest = await source.digest()
    await stream.send(AnyTTYClientBinding.ResourceStreamFrameType.FILE_FINISH, encodeFileTransferFinishPayload({ size: source.size, sha256: digest }))
    await result
  } finally {
    subscription.close()
    closeSubscription.close()
    signal.removeEventListener('abort', abort)
    await source.close()
  }
}

function concatBytes(chunks: Uint8Array[]): Uint8Array {
  const size = chunks.reduce((total, chunk) => total + chunk.byteLength, 0)
  const result = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) {
    result.set(chunk, offset)
    offset += chunk.byteLength
  }
  return result
}

function bytesBase64(bytes: Uint8Array): string {
  let binary = ''
  for (let offset = 0; offset < bytes.byteLength; offset += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(offset, Math.min(bytes.byteLength, offset + 0x8000)))
  }
  return btoa(binary)
}

function base64Bytes(value: string): Uint8Array {
  const binary = atob(value)
  const result = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) result[index] = binary.charCodeAt(index)
  return result
}

async function verifyBlobDigest(blob: Blob, expected: Uint8Array): Promise<void> {
  if (expected.byteLength !== 32) throw new Error('download digest is invalid')
  const actual = new Uint8Array(await crypto.subtle.digest('SHA-256', await blob.arrayBuffer()))
  if (!bytesEqual(actual, expected)) throw new Error('download digest mismatch')
}

function bytesEqual(left: Uint8Array, right: Uint8Array): boolean {
  if (left.byteLength !== right.byteLength) return false
  let mismatch = 0
  for (let index = 0; index < left.byteLength; index += 1) mismatch |= left[index]! ^ right[index]!
  return mismatch === 0
}

function command(caseName: string, value: object) {
  return create(AnyTTYApiApplication.CommandEnvelopeSchema, { command: { case: caseName, value } } as never)
}

function canResume(status: TransferStatus): boolean { return status === 'paused' || status === 'failed' || status === 'missing' }

async function awaitAbortable<T>(promise: Promise<T>, signal: AbortSignal, onLate: (value: T) => void): Promise<T> {
  if (signal.aborted) {
    void promise.then(onLate, () => undefined)
    throw abortError(signal)
  }
  return await new Promise<T>((resolve, reject) => {
    let settled = false
    const abort = () => {
      if (settled) return
      settled = true
      signal.removeEventListener('abort', abort)
      reject(abortError(signal))
    }
    signal.addEventListener('abort', abort, { once: true })
    void promise.then(
      (value) => {
        if (settled) {
          onLate(value)
          return
        }
        settled = true
        signal.removeEventListener('abort', abort)
        resolve(value)
      },
      (error) => {
        if (settled) return
        settled = true
        signal.removeEventListener('abort', abort)
        reject(error)
      },
    )
  })
}

function abortError(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new DOMException('Aborted', 'AbortError')
}

function transferID(direction: string, machineId: string, path: string): string {
  return `${direction}-${machineId}-${path}-${Date.now()}`
}

function sameSession(left: { endpointId: string; routeId: string; generation: bigint } | undefined, right: ProtoClientSession['stamp']): boolean {
  return left?.endpointId === right.endpointId && left.routeId === right.routeId && left.generation === right.generation
}
