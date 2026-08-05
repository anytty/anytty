import { create } from '@bufbuild/protobuf'
import type { ProtoClientSession, ProtoResourceStream } from '../core/protoClientSession'
import type { ResourceHandle } from '../generated/apipb/common_pb'
import { ResourceStreamFrameType } from '../generated/bindingpb/client_binding_pb'
import { CommandEnvelopeSchema, ReleaseResourceCommandSchema } from '../generated/apipb/application_pb'
import {
  FileCopyCommandSchema,
  FileDeleteCommandSchema,
  FileDownloadOpenCommandSchema,
  FileEntryType as ProtoFileEntryType,
  FileListCommandSchema,
  FileMkdirCommandSchema,
  FileMoveCommandSchema,
  FilePreviewCommandSchema,
  FileRenameCommandSchema,
  FileStatCommandSchema,
  type FileEntry as ProtoFileEntry,
  type FileOperationResult,
} from '../generated/apipb/file_pb'
import { isModelPreviewFile } from './modelFileTypes'
import { normalizeFilePath, parentPath } from './fileUtils'
import {
  decodeFileTransferDataPayload,
  decodeFileTransferFinishPayload,
  decodeFileStreamErrorPayload,
  encodeFileTransferAckPayload,
} from './fileStreamProtocol'

export type FileEntryType = 'file' | 'dir' | 'symlink' | 'symlink-dir'

export interface FileEntry {
  path?: string | undefined
  name: string
  type: FileEntryType | string
  size: number
  mode?: string | undefined
  modTime?: string | undefined
  linkTarget?: string | undefined
  childCount?: number | undefined
  hardLink?: boolean | undefined
  linkCount?: number | undefined
  inode?: number | undefined
}

export interface DirListResponse {
  path: string
  entries: FileEntry[]
  parent: string
  total: number
}

export type FilePreviewCategory = 'text' | 'image' | 'video' | 'model' | 'unsupported'

export interface FilePreviewResponse {
  path: string
  name: string
  size: number
  mimeType: string
  category: FilePreviewCategory
  isText: boolean
  content?: string | undefined
  contentBase64?: string | undefined
  previewLimit?: number | undefined
}

export interface DownloadInitResponse {
  transfer_id: string
  channel: number
  path: string
  name: string
  size: number
  chunk_size: number
  window_bytes: number
  offset: number
  modified_at_unix_nano: number
  resource?: ResourceHandle | undefined
}

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
}

export interface FileTransferContext {
  /** Subscribe to state changes (useSyncExternalStore API). */
  subscribe(listener: () => void): () => void
  /** Get current snapshot (useSyncExternalStore API). */
  getSnapshot(): { transfers: TransferInfo[]; hasActiveTransfers: boolean }
  getDownloadResumeOffset?(machineId: string, filePath: string, fileSize: number): Promise<number> | number
  startDownload(machineId: string, fileName: string, fileSize: number, filePath: string, offset?: number): void
  startUpload(machineId: string, files: Array<{ uri: string; name: string; size: number }>, targetDir: string): void
  /** Native mode only: opens system file picker then starts upload. */
  pickAndUpload?(machineId: string, targetDir: string): void
  pauseTransfer?(id: string): void | Promise<void>
  resumeTransfer?(id: string): void | Promise<void>
  resumeAllTransfers?(machineId?: string): void | Promise<void>
  cancelTransfer(id: string): void
  dismissTransfer(id: string): void
  isNative: boolean
}

export interface FilePreviewStreamProgress {
  receivedSize: number
  totalSize: number
}

export interface FilePreviewStreamChunk extends FilePreviewStreamProgress {
  chunk: ArrayBuffer
}

export interface FilePreviewStreamResult {
  blob: Blob
  receivedSize: number
  offset: number
  totalSize: number
}

export interface FilePreviewStreamOptions {
  signal?: AbortSignal | undefined
  offset?: number | undefined
  length?: number | undefined
  onChunk?: ((chunk: FilePreviewStreamChunk) => void) | undefined
  onProgress?: ((progress: FilePreviewStreamProgress) => void) | undefined
}

export interface FileApi {
  listDir(path?: string, cursor?: string, limit?: number): Promise<DirListResponse>
  stat(path: string): Promise<FileEntry>
  preview(path: string, maxSize?: number): Promise<FilePreviewResponse>
  mkdir(path: string): Promise<{ path: string }>
  delete(path: string): Promise<{ path: string }>
  rename(path: string, newPath: string): Promise<{ path: string }>
  copy(paths: string[], targetDir: string): Promise<void>
  move(paths: string[], targetDir: string): Promise<void>
  batchDelete(paths: string[]): Promise<void>
  downloadOpen(path: string, offset?: number, expectedSize?: number, expectedModifiedAtUnixNano?: number): Promise<DownloadInitResponse>
}

export interface FilePreviewSource {
  preview(path: string, maxSize?: number): Promise<FilePreviewResponse>
  stream(path: string, mimeType: string, options?: FilePreviewStreamOptions): Promise<FilePreviewStreamResult>
}

export function createFileApi(session: ProtoClientSession): FileApi {
  return createProtoFileApi(session)
}

function createProtoFileApi(session: ProtoClientSession): FileApi {
  const execute = (caseName: string, value: object) => session.execute(
    create(CommandEnvelopeSchema, { command: { case: caseName, value } } as never),
  ).catch((error) => { throw normalizeFileError(error) })
  const operation = async (caseName: string, value: object) => {
    const result = await execute(caseName, value)
    if (result.result.case !== 'fileOperation') throw new Error(`${caseName} returned no file operation result`)
    return protoOperationPath(result.result.value)
  }
  const batch = async (caseName: string, value: object) => {
    const result = await execute(caseName, value)
    if (result.result.case !== 'fileBatch') throw new Error(`${caseName} returned no file batch result`)
    for (const item of result.result.value.results) protoOperationPath(item)
  }
  return {
    async listDir(path = '/', cursor = '', limit = 500) {
      const result = await execute('fileList', create(FileListCommandSchema, { path: normalizeFilePath(path), cursor, limit }))
      if (result.result.case !== 'fileList') throw new Error('file list returned no result')
      return {
        path: normalizeFilePath(result.result.value.path),
        entries: result.result.value.entries.map(protoFileEntry),
        parent: parentPath(result.result.value.path),
        total: result.result.value.entries.length,
      }
    },
    async stat(path) {
      const result = await execute('fileStat', create(FileStatCommandSchema, { path }))
      if (result.result.case !== 'fileStat' || !result.result.value.entry) throw new Error('file stat returned no entry')
      return protoFileEntry(result.result.value.entry)
    },
    async preview(path, maxSize) {
      const result = await execute('filePreview', create(FilePreviewCommandSchema, { path, maxBytes: BigInt(maxSize ?? 0) }))
      if (result.result.case !== 'filePreview' || !result.result.value.entry) throw new Error('file preview returned no entry')
      const entry = protoFileEntry(result.result.value.entry)
      const mimeType = result.result.value.mimeType || 'application/octet-stream'
      const category = normalizePreviewCategory(undefined, mimeType, basename(path))
      return {
        path,
        name: entry.name,
        size: entry.size,
        mimeType,
        category,
        isText: category === 'text',
        ...(category === 'text' ? { content: new TextDecoder().decode(result.result.value.content) } : {}),
        previewLimit: result.result.value.truncated ? result.result.value.content.byteLength : undefined,
      }
    },
    mkdir: (path) => operation('fileMkdir', create(FileMkdirCommandSchema, { path, recursive: true })),
    delete: (path) => operation('fileDelete', create(FileDeleteCommandSchema, { path, recursive: true })),
    rename: (path, newPath) => operation('fileRename', create(FileRenameCommandSchema, { path, newPath })),
    copy: (paths, targetDir) => batch('fileCopy', create(FileCopyCommandSchema, { paths, targetDirectory: targetDir })),
    move: (paths, targetDir) => batch('fileMove', create(FileMoveCommandSchema, { paths, targetDirectory: targetDir })),
    async batchDelete(paths) {
      for (const path of paths) await operation('fileDelete', create(FileDeleteCommandSchema, { path, recursive: true }))
    },
    async downloadOpen(path, offset = 0, expectedSize = 0, expectedModifiedAtUnixNano = 0) {
      const result = await execute('fileDownloadOpen', create(FileDownloadOpenCommandSchema, {
        path,
        offset: BigInt(offset),
        expectedSize: BigInt(expectedSize),
        expectedModifiedAtUnixNano: BigInt(expectedModifiedAtUnixNano),
      }))
      if (result.result.case !== 'fileTransferOpen' || !result.result.value.transfer?.resource) {
        throw new Error('file download returned no transfer resource')
      }
      const transfer = result.result.value.transfer
      return {
        transfer_id: '', channel: 0, path: transfer.path, name: basename(transfer.path),
        size: Number(transfer.size), chunk_size: transfer.chunkBytes, window_bytes: Number(transfer.windowBytes),
        offset: Number(transfer.offset), modified_at_unix_nano: Number(transfer.modifiedAtUnixNano),
        resource: transfer.resource,
      }
    },
  }
}

async function readProtoFileTransferStream(
  stream: ProtoResourceStream,
  init: DownloadInitResponse,
  mimeType: string,
  options: FilePreviewStreamOptions,
): Promise<FilePreviewStreamResult> {
  if (init.chunk_size <= 0 || init.window_bytes <= 0) throw new Error('file stream returned invalid flow-control limits')
  const requestedLength = options.length === undefined
    ? undefined
    : Number.isFinite(options.length) ? Math.max(0, Math.floor(options.length)) : 0
  if (requestedLength === 0) throw new Error('file preview range length must be positive')
  const targetEnd = requestedLength === undefined ? init.size : Math.min(init.size, init.offset + requestedLength)
  return await new Promise<FilePreviewStreamResult>((resolve, reject) => {
    const chunks: Uint8Array[] = []
    let wireOffset = init.offset
    let receivedSize = 0
    let bytesSinceAck = 0
    let settled = false
    let subscription: { close(): void } | null = null
    let closeSubscription: { close(): void } | null = null
    const cleanup = () => {
      subscription?.close()
      closeSubscription?.close()
      options.signal?.removeEventListener('abort', abortListener)
    }
    const finish = (result: FilePreviewStreamResult) => {
      if (settled) return
      settled = true
      cleanup()
      resolve(result)
    }
    const fail = (error: unknown) => {
      if (settled) return
      settled = true
      cleanup()
      reject(error)
    }
    const finishRange = () => {
      const blobParts = chunks.map((chunk) => chunk.buffer.slice(chunk.byteOffset, chunk.byteOffset + chunk.byteLength) as ArrayBuffer)
      finish({ blob: new Blob(blobParts, { type: mimeType.trim() || 'application/octet-stream' }), receivedSize, offset: init.offset, totalSize: init.size })
    }
    const abortListener = () => {
      void stream.close().catch(() => undefined)
      fail(abortError())
    }
    subscription = stream.subscribe((type, payload) => {
      if (settled) return
      if (type === ResourceStreamFrameType.FILE_DATA) {
        const data = decodeFileTransferDataPayload(payload)
        if (data.offset !== wireOffset || data.data.byteLength === 0 || data.data.byteLength > init.chunk_size) {
          fail(new Error('file stream offset or chunk is invalid'))
          return
        }
        wireOffset += data.data.byteLength
        bytesSinceAck += data.data.byteLength
        const retainedLength = Math.max(0, Math.min(data.data.byteLength, targetEnd - data.offset))
        if (retainedLength > 0) chunks.push(data.data.subarray(0, retainedLength))
        receivedSize += retainedLength
        const chunk = data.data.buffer.slice(data.data.byteOffset, data.data.byteOffset + retainedLength) as ArrayBuffer
        options.onChunk?.({ chunk, receivedSize, totalSize: init.size })
        options.onProgress?.({ receivedSize, totalSize: init.size })
        if (requestedLength !== undefined && wireOffset >= targetEnd) {
          finishRange()
          return
        }
        if (bytesSinceAck >= init.window_bytes) {
          const credit = bytesSinceAck
          bytesSinceAck = 0
          void stream.send(ResourceStreamFrameType.FILE_ACK, encodeFileTransferAckPayload({ offset: wireOffset, windowBytes: credit })).catch(fail)
        }
        return
      }
      if (type === ResourceStreamFrameType.FILE_FINISH) {
        const declared = decodeFileTransferFinishPayload(payload)
        if (declared.size !== init.size || wireOffset !== init.size || init.offset + receivedSize !== targetEnd) {
          fail(new Error('file stream completed with the wrong size'))
          return
        }
        if (init.offset === 0 && receivedSize === init.size) {
          cleanup()
          void verifyFileDigest(chunks, declared.sha256).then(finishRange).catch(fail)
        } else {
          finishRange()
        }
        return
      }
      if (type === ResourceStreamFrameType.ERROR) fail(new Error(decodeFileStreamErrorPayload(payload)))
    })
    closeSubscription = stream.subscribeClosed(fail)
    options.signal?.addEventListener('abort', abortListener, { once: true })
    if (options.signal?.aborted) abortListener()
  })
}

function protoFileEntry(entry: ProtoFileEntry): FileEntry {
  return {
    path: entry.path ? normalizeFilePath(entry.path) : undefined,
    name: entry.name,
    type: entry.type === ProtoFileEntryType.DIRECTORY ? 'dir' : entry.type === ProtoFileEntryType.SYMLINK ? 'symlink' : entry.type === ProtoFileEntryType.FILE ? 'file' : 'other',
    size: Number(entry.size),
    mode: entry.mode ? entry.mode.toString(8) : undefined,
    modTime: entry.modifiedAtUnixNano > 0n ? new Date(Number(entry.modifiedAtUnixNano / 1_000_000n)).toISOString() : undefined,
    linkTarget: entry.linkTarget || undefined,
  }
}

function protoOperationPath(result: FileOperationResult): { path: string } {
  if (!result.success) throw new Error(result.errorMessage || result.errorCode || 'file operation failed')
  return { path: result.targetPath || result.path }
}

export function createFilePreviewSource(session: ProtoClientSession): FilePreviewSource {
  const api = createFileApi(session)
  return {
    async preview(path: string, maxSize?: number) {
      if (maxSize !== undefined) return await api.preview(path, maxSize)
      const probe = await api.preview(path, 64 << 10)
      if (probe.category === 'text' && probe.previewLimit) return await api.preview(path)
      return probe
    },
    async stream(path: string, mimeType: string, options: FilePreviewStreamOptions = {}) {
      const requestedOffset = options.offset ?? 0
      const offset = Number.isFinite(requestedOffset) ? Math.max(0, Math.floor(requestedOffset)) : 0
      const init = await api.downloadOpen(path, offset)
      const resource = init.resource
      if (!resource) throw new Error('file download returned no resource handle')
      let stream: ProtoResourceStream | null = null
      try {
        throwIfAborted(options.signal)
        if (init.offset !== offset) throw new Error('file preview range offset was not accepted')
        stream = await session.openResourceStream(resource, options.signal ? { signal: options.signal } : undefined)
        return await readProtoFileTransferStream(stream, init, mimeType, options)
      } finally {
        await stream?.close().catch(() => undefined)
        await session.execute(create(CommandEnvelopeSchema, {
          command: { case: 'releaseResource', value: create(ReleaseResourceCommandSchema, { resource }) },
        })).catch(() => undefined)
      }
    },
  }
}


async function verifyFileDigest(chunks: Uint8Array[], expected: Uint8Array): Promise<void> {
  if (expected.byteLength !== 32) throw new Error('file stream returned an invalid SHA-256')
  const size = chunks.reduce((total, chunk) => total + chunk.byteLength, 0)
  const content = new Uint8Array(size)
  let offset = 0
  for (const chunk of chunks) { content.set(chunk, offset); offset += chunk.byteLength }
  const actual = new Uint8Array(await crypto.subtle.digest('SHA-256', content))
  if (!actual.every((value, index) => value === expected[index])) throw new Error('file stream SHA-256 mismatch')
}

function normalizePreviewCategory(raw: string | undefined, mimeType: string, name: string): FilePreviewCategory {
  const category = raw?.trim().toLowerCase()
  const normalizedMime = mimeType.trim().toLowerCase()
  if (category === 'text' || category === 'image' || category === 'video' || category === 'model') return category
  if (isModelPreviewFile(name, normalizedMime)) return 'model'
  if (category === 'unsupported') return category
  if (normalizedMime.startsWith('text/')) return 'text'
  if (normalizedMime.startsWith('image/')) return 'image'
  if (normalizedMime.startsWith('video/')) return 'video'
  if (normalizedMime.startsWith('model/')) return 'model'
  return 'unsupported'
}

function basename(path: string): string {
  const normalized = path.replace(/\/+$/, '')
  return normalized.slice(normalized.lastIndexOf('/') + 1) || normalized
}

function normalizeFileError(err: unknown): Error {
  if (err instanceof Error) return err
  return new Error(String(err))
}

function throwIfAborted(signal: AbortSignal | undefined): void {
  if (signal?.aborted) throw abortError()
}

function normalizeByteRangeValue(value: number | undefined): number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? Math.floor(value) : 0
}

function abortError(): Error {
  const err = new Error('File preview stream was cancelled.')
  err.name = 'AbortError'
  return err
}
