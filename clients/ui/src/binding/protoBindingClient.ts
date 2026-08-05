import { create, fromBinary, toBinary } from '@bufbuild/protobuf'
import * as AnyTTYApiApplication from '../generated/apipb/application_pb'
import * as AnyTTYApiCommon from '../generated/apipb/common_pb'
import * as AnyTTYApiFile from '../generated/apipb/file_pb'
import * as AnyTTYApiHistory from '../generated/apipb/history_pb'
import * as AnyTTYClientBinding from '../generated/bindingpb/client_binding_pb'
import * as AnyTTYRemoteAuth from '../generated/remoteauthpb/remote_auth_pb'
import type { ProtoClientSession, ProtoClientSessionCloseError, ProtoClientSubscription, ProtoResourceStream } from '../core/protoClientSession'
import type { ConnectionPolicy, ConnectionPolicyState } from '../core/transport'

export const BindingOperation = {
  OPEN_SESSION: 0x10,
  EXECUTE: 0x11,
  ENGINE_COMMAND: 0x12,
  CANCEL: 0x14,
  CLOSE_SESSION: 0x15,
  RELEASE: 0x16,
  OPEN_RESOURCE_STREAM: 0x17,
  SEND_RESOURCE_STREAM_FRAME: 0x18,
  CLOSE_RESOURCE_STREAM: 0x19,
} as const

export type BindingOperationCode = typeof BindingOperation[keyof typeof BindingOperation]

export interface ProtoBindingBackend {
  start(onEvent: (payload: Uint8Array) => void, onClosed: (error: Error) => void): void
  request(operation: BindingOperationCode, payload: Uint8Array, handle?: bigint, signal?: AbortSignal): Promise<bigint>
  close(): Promise<void>
}

type PendingOperation<T> = {
  resolve(value: T): void
  reject(error: Error): void
}

const MAX_EARLY_STREAM_EVENTS = 64
const MAX_EARLY_OPERATION_EVENTS = 4096
const MAX_ABANDONED_STREAM_HANDLES = 4096
const CANCELLED_CLEANUP_TIMEOUT_MS = 5_000

/** ProtoBindingClient is the shared Android-JNI and Web-WASM session/resource contract owner. */
export class ProtoBindingClient {
  private readonly openOperations = new Map<bigint, PendingOperation<ProtoBindingSession>>()
  private readonly executeOperations = new Map<bigint, PendingOperation<AnyTTYApiApplication.ResultEnvelope>>()
  private readonly importOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.ImportPairingResult>>()
  private readonly deleteOperations = new Map<bigint, PendingOperation<void>>()
  private readonly registryGetOperations = new Map<bigint, PendingOperation<AnyTTYRemoteAuth.EndpointRegistryV1>>()
  private readonly endpointUpsertOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.EndpointUpsertResult>>()
  private readonly endpointDeleteOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.EndpointDeleteResult>>()
  private readonly endpointShareReceiveOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.EndpointShareReceiveResult>>()
  private readonly endpointShareCommitOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.EndpointShareCommitResult>>()
  private readonly sshCredentialProvisionOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.SSHCredentialProvisionResult>>()
  private readonly connectionPolicyGetOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.ConnectionPolicyState>>()
  private readonly connectionPolicyApplyOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.ConnectionPolicyState>>()
  private readonly connectionSnapshotGetOperations = new Map<bigint, PendingOperation<AnyTTYClientBinding.ConnectionSnapshot>>()
  private readonly sessionInvalidateOperations = new Map<bigint, PendingOperation<void>>()
  private readonly sessions = new Map<bigint, ProtoBindingSession>()
  private readonly streams = new Map<bigint, ProtoBindingResourceStream>()
  private readonly earlyOperationEvents = new Map<bigint, AnyTTYClientBinding.EventEnvelope>()
  private readonly cancelledOperations = new Map<bigint, boolean>()
  private readonly earlyStreamEvents = new Map<bigint, AnyTTYClientBinding.EventEnvelope[]>()
  private readonly abandonedStreamHandles = new Set<bigint>()
  private readonly retiredSessionHandles = new Set<bigint>()
  private readonly releasedSessionHandles = new Set<bigint>()
  private readonly releasedSessionOrder: bigint[] = []
  private readonly releasedStreamHandles = new Set<bigint>()
  private readonly releasedStreamOrder: bigint[] = []
  private closed = false
  private intentionalClose = false

  constructor(private readonly backend: ProtoBindingBackend) {
    backend.start((payload) => this.onEvent(fromBinary(AnyTTYClientBinding.EventEnvelopeSchema, payload)), (error) => this.onClosed(error))
  }

  async openSession(request: AnyTTYClientBinding.OpenSessionRequest, signal?: AbortSignal): Promise<ProtoClientSession> {
    const operation = await this.backend.request(BindingOperation.OPEN_SESSION, toBinary(AnyTTYClientBinding.OpenSessionRequestSchema, request), undefined, signal)
    return await this.waitOperation(this.openOperations, operation, signal)
  }

  async importPairing(request: AnyTTYClientBinding.ImportPairingRequest, signal?: AbortSignal): Promise<AnyTTYClientBinding.ImportPairingResult> {
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'importPairing', value: request } }), signal)
    return await this.waitOperation(this.importOperations, operation, signal)
  }

  async deleteCredential(request: AnyTTYClientBinding.DeleteCredentialRequest, signal?: AbortSignal): Promise<void> {
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'deleteCredential', value: request } }), signal)
    await this.waitOperation(this.deleteOperations, operation, signal)
  }

  async getEndpointRegistry(signal?: AbortSignal): Promise<AnyTTYRemoteAuth.EndpointRegistryV1> {
    const request = create(AnyTTYClientBinding.EndpointRegistryGetRequestSchema, { requestId: crypto.randomUUID() })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'endpointRegistryGet', value: request } }), signal)
    return await this.waitOperation(this.registryGetOperations, operation, signal)
  }

  async upsertEndpoint(endpoint: AnyTTYRemoteAuth.EndpointConfigV1, makeDefault = false, signal?: AbortSignal): Promise<AnyTTYClientBinding.EndpointUpsertResult> {
    const request = create(AnyTTYClientBinding.EndpointUpsertRequestSchema, { requestId: crypto.randomUUID(), endpoint, makeDefault })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'endpointUpsert', value: request } }), signal)
    return await this.waitOperation(this.endpointUpsertOperations, operation, signal)
  }

  async deleteEndpoint(endpointId: string, signal?: AbortSignal): Promise<AnyTTYClientBinding.EndpointDeleteResult> {
    const request = create(AnyTTYClientBinding.EndpointDeleteRequestSchema, { requestId: crypto.randomUUID(), endpointId })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'endpointDelete', value: request } }), signal)
    return await this.waitOperation(this.endpointDeleteOperations, operation, signal)
  }

  async receiveEndpointShare(portableOffer: string, signal?: AbortSignal): Promise<AnyTTYClientBinding.EndpointShareReceiveResult> {
    const request = create(AnyTTYClientBinding.EndpointShareReceiveRequestSchema, { requestId: crypto.randomUUID(), portableOffer })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'endpointShareReceive', value: request } }), signal)
    return await this.waitOperation(this.endpointShareReceiveOperations, operation, signal)
  }

  async commitEndpointShare(importToken: string, signal?: AbortSignal): Promise<AnyTTYClientBinding.EndpointShareCommitResult> {
    const request = create(AnyTTYClientBinding.EndpointShareCommitRequestSchema, { requestId: crypto.randomUUID(), importToken })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'endpointShareCommit', value: request } }), signal)
    return await this.waitOperation(this.endpointShareCommitOperations, operation, signal)
  }

  async provisionSSHCredential(endpointId: string, routeId: string, signal?: AbortSignal): Promise<AnyTTYClientBinding.SSHCredentialProvisionResult> {
    const request = create(AnyTTYClientBinding.SSHCredentialProvisionRequestSchema, { requestId: crypto.randomUUID(), endpointId, routeId })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'sshCredentialProvision', value: request } }), signal)
    return await this.waitOperation(this.sshCredentialProvisionOperations, operation, signal)
  }

  async getConnectionPolicy(endpointId: string, signal?: AbortSignal): Promise<AnyTTYClientBinding.ConnectionPolicyState> {
    const request = create(AnyTTYClientBinding.ConnectionPolicyGetRequestSchema, { requestId: crypto.randomUUID(), endpointId })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'connectionPolicyGet', value: request } }), signal)
    return await this.waitOperation(this.connectionPolicyGetOperations, operation, signal)
  }

  async applyConnectionPolicy(endpointId: string, policy: AnyTTYClientBinding.ConnectionPolicy, signal?: AbortSignal): Promise<AnyTTYClientBinding.ConnectionPolicyState> {
    const request = create(AnyTTYClientBinding.ConnectionPolicyApplyRequestSchema, { requestId: crypto.randomUUID(), endpointId, policy })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'connectionPolicyApply', value: request } }), signal)
    return await this.waitOperation(this.connectionPolicyApplyOperations, operation, signal)
  }

  async getConnectionSnapshot(sessionHandle: bigint, signal?: AbortSignal): Promise<AnyTTYClientBinding.ConnectionSnapshot> {
    const request = create(AnyTTYClientBinding.ConnectionSnapshotGetRequestSchema, { requestId: crypto.randomUUID(), sessionHandle })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'connectionSnapshotGet', value: request } }), signal)
    return await this.waitOperation(this.connectionSnapshotGetOperations, operation, signal)
  }

  async invalidateSession(sessionHandle: bigint, signal?: AbortSignal): Promise<void> {
    const request = create(AnyTTYClientBinding.SessionInvalidateRequestSchema, { requestId: crypto.randomUUID(), sessionHandle })
    const operation = await this.engineCommand(create(AnyTTYClientBinding.EngineCommandSchema, { command: { case: 'sessionInvalidate', value: request } }), signal)
    await this.waitOperation(this.sessionInvalidateOperations, operation, signal)
  }

  private async engineCommand(command: AnyTTYClientBinding.EngineCommand, signal?: AbortSignal): Promise<bigint> {
    return await this.backend.request(BindingOperation.ENGINE_COMMAND, toBinary(AnyTTYClientBinding.EngineCommandSchema, command), undefined, signal)
  }

  async close(): Promise<void> {
    if (this.closed) return
    this.closed = true
    this.intentionalClose = true
    await this.backend.close()
    const error = new Error('Proto binding client is closed')
    this.rejectAll(error)
    this.sessions.forEach((session) => session.markClosed())
    this.streams.forEach((stream) => stream.markClosed(error))
    this.sessions.clear()
    this.streams.clear()
  }

  async execute(session: bigint, command: AnyTTYApiApplication.CommandEnvelope, signal?: AbortSignal): Promise<AnyTTYApiApplication.ResultEnvelope> {
    const operation = await this.backend.request(BindingOperation.EXECUTE, toBinary(AnyTTYApiApplication.CommandEnvelopeSchema, command), session, signal)
    const historyMode = command.command.case === 'historyWindow' ? command.command.value.mode : undefined
    const ownsHistorySnapshot = historyMode === AnyTTYApiHistory.HistoryWindowMode.UNSPECIFIED || historyMode === AnyTTYApiHistory.HistoryWindowMode.LATEST
    return await this.waitOperation(this.executeOperations, operation, signal, ownsHistorySnapshot)
  }

  async openResourceStream(session: bigint, resource: NonNullable<AnyTTYClientBinding.OpenResourceStreamRequest['resource']>, options?: { initialUploadOffset?: bigint; signal?: AbortSignal }): Promise<ProtoBindingResourceStream> {
    const request = create(AnyTTYClientBinding.OpenResourceStreamRequestSchema, { resource, initialUploadOffset: options?.initialUploadOffset ?? 0n })
    const handle = await awaitAbortableHandle(
      this.backend.request(BindingOperation.OPEN_RESOURCE_STREAM, toBinary(AnyTTYClientBinding.OpenResourceStreamRequestSchema, request), session, options?.signal),
      options?.signal,
      (late) => this.abandonResourceStream(late),
    )
    const stream = new ProtoBindingResourceStream(this, handle)
    this.streams.set(handle, stream)
    const early = this.earlyStreamEvents.get(handle)
    if (early) {
      this.earlyStreamEvents.delete(handle)
      for (const event of early) queueMicrotask(() => this.onEvent(event))
    }
    return stream
  }

  async sendResourceStreamFrame(handle: bigint, type: AnyTTYClientBinding.ResourceStreamFrameType, payload: Uint8Array): Promise<void> {
    const frame = create(AnyTTYClientBinding.ResourceStreamFrameSchema, { streamHandle: handle, type, payload })
    await this.backend.request(BindingOperation.SEND_RESOURCE_STREAM_FRAME, toBinary(AnyTTYClientBinding.ResourceStreamFrameSchema, frame), handle)
  }

  async closeResourceStream(handle: bigint): Promise<void> {
    await this.backend.request(BindingOperation.CLOSE_RESOURCE_STREAM, new Uint8Array(), handle)
  }

  async closeSession(handle: bigint): Promise<void> {
    await this.backend.request(BindingOperation.CLOSE_SESSION, new Uint8Array(), handle)
  }

  async release(handle: bigint): Promise<void> {
    await this.backend.request(BindingOperation.RELEASE, new Uint8Array(), handle)
  }

  private releaseCompletedOperation(handle: bigint): void {
    // operation handle 仍由 binding client 拥有；generation 关闭时的异步释放失败必须在这里
    // 收口到 client lifecycle，不能形成 WebView 未处理 rejection。
    void this.release(handle).catch((error) => this.onClosed(new Error(`completed operation release failed: ${errorMessage(error)}`)))
  }

  private async waitOperation<T>(registry: Map<bigint, PendingOperation<T>>, operation: bigint, signal?: AbortSignal, ownsHistorySnapshot = false): Promise<T> {
    if (signal?.aborted) {
      this.abandonOperation(registry, operation, ownsHistorySnapshot)
      throw abortError(signal)
    }
    return await new Promise<T>((resolve, reject) => {
      registry.set(operation, { resolve, reject })
      const abort = () => {
        this.abandonOperation(registry, operation, ownsHistorySnapshot)
        reject(signal ? abortError(signal) : new DOMException('Aborted', 'AbortError'))
      }
      signal?.addEventListener('abort', abort, { once: true })
      const entry = registry.get(operation)
      if (entry) {
        entry.resolve = (value) => {
          signal?.removeEventListener('abort', abort)
          resolve(value)
        }
        entry.reject = (error) => {
          signal?.removeEventListener('abort', abort)
          reject(error)
        }
      }
      const early = this.earlyOperationEvents.get(operation)
      if (early) {
        this.earlyOperationEvents.delete(operation)
        queueMicrotask(() => this.onEvent(early))
      }
    })
  }

  private async cancel(operation: bigint): Promise<void> {
    await this.backend.request(BindingOperation.CANCEL, new Uint8Array(), operation).catch(() => undefined)
  }

  private abandonOperation<T>(registry: Map<bigint, PendingOperation<T>>, operation: bigint, ownsHistorySnapshot = false): void {
    registry.delete(operation)
    const completed = this.earlyOperationEvents.get(operation)
    if (completed) {
      this.earlyOperationEvents.delete(operation)
      this.retireCancelledOperation(operation, completed, ownsHistorySnapshot)
      return
    }
    this.cancelledOperations.set(operation, ownsHistorySnapshot)
    void this.cancel(operation)
  }

  private onEvent(envelope: AnyTTYClientBinding.EventEnvelope): void {
    const event = envelope.event
    const operationHandle = bindingOperationHandle(envelope)
    const ownsHistorySnapshot = operationHandle === undefined ? undefined : this.cancelledOperations.get(operationHandle)
    if (operationHandle !== undefined && ownsHistorySnapshot !== undefined) {
      this.cancelledOperations.delete(operationHandle)
      this.retireCancelledOperation(operationHandle, envelope, ownsHistorySnapshot)
      return
    }
    if (operationHandle !== undefined && !this.hasPendingOperation(event.case, operationHandle)) {
      if (!this.earlyOperationEvents.has(operationHandle) && this.earlyOperationEvents.size >= MAX_EARLY_OPERATION_EVENTS) {
        this.onClosed(new Error('Proto binding early operation event queue overflow'))
        return
      }
      this.earlyOperationEvents.set(operationHandle, envelope)
      return
    }
    switch (event.case) {
      case 'openSession': {
        const pending = this.openOperations.get(event.value.operationHandle)
        if (!pending) return
        this.openOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.session || event.value.sessionHandle === 0n) {
          pending.reject(apiError(event.value.error, 'open session failed'))
          this.releaseCompletedOperation(event.value.operationHandle)
          return
        }
        const session = new ProtoBindingSession(this, event.value.sessionHandle, event.value.session, event.value.connection)
        this.sessions.set(event.value.sessionHandle, session)
        pending.resolve(session)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'execute': {
        const pending = this.executeOperations.get(event.value.operationHandle)
        if (!pending) return
        this.executeOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.result) pending.reject(apiError(event.value.error, 'application command failed'))
        else pending.resolve(event.value.result)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'importPairing': {
        const pending = this.importOperations.get(event.value.operationHandle)
        if (!pending) return
        this.importOperations.delete(event.value.operationHandle)
        if (event.value.error) pending.reject(apiError(event.value.error, 'pairing import failed'))
        else pending.resolve(event.value)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'deleteCredential': {
        const pending = this.deleteOperations.get(event.value.operationHandle)
        if (!pending) return
        this.deleteOperations.delete(event.value.operationHandle)
        if (event.value.error) pending.reject(apiError(event.value.error, 'credential delete failed'))
        else pending.resolve()
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'endpointRegistryGet': {
        const pending = this.registryGetOperations.get(event.value.operationHandle)
        if (!pending) return
        this.registryGetOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.registry) pending.reject(apiError(event.value.error, 'endpoint registry get failed'))
        else pending.resolve(event.value.registry)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'endpointUpsert': {
        const pending = this.endpointUpsertOperations.get(event.value.operationHandle)
        if (!pending) return
        this.endpointUpsertOperations.delete(event.value.operationHandle)
        if (event.value.error) pending.reject(apiError(event.value.error, 'endpoint upsert failed'))
        else pending.resolve(event.value)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'endpointDelete': {
        const pending = this.endpointDeleteOperations.get(event.value.operationHandle)
        if (!pending) return
        this.endpointDeleteOperations.delete(event.value.operationHandle)
        if (event.value.error) pending.reject(apiError(event.value.error, 'endpoint delete failed'))
        else pending.resolve(event.value)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'endpointShareReceive': {
        const pending = this.endpointShareReceiveOperations.get(event.value.operationHandle)
        if (!pending) return
        this.endpointShareReceiveOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.preview) pending.reject(apiError(event.value.error, 'endpoint share receive failed'))
        else pending.resolve(event.value)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'endpointShareCommit': {
        const pending = this.endpointShareCommitOperations.get(event.value.operationHandle)
        if (!pending) return
        this.endpointShareCommitOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.endpoint || !event.value.registry) pending.reject(apiError(event.value.error, 'endpoint share commit failed'))
        else pending.resolve(event.value)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'sshCredentialProvision': {
        const pending = this.sshCredentialProvisionOperations.get(event.value.operationHandle)
        if (!pending) return
        this.sshCredentialProvisionOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.endpoint || !event.value.registry || !event.value.authorizedKey) {
          pending.reject(apiError(event.value.error, 'SSH credential provision failed'))
        } else {
          pending.resolve(event.value)
        }
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'connectionPolicyGet': {
        const pending = this.connectionPolicyGetOperations.get(event.value.operationHandle)
        if (!pending) return
        this.connectionPolicyGetOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.state) pending.reject(apiError(event.value.error, 'connection policy get failed'))
        else pending.resolve(event.value.state)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'connectionPolicyApply': {
        const pending = this.connectionPolicyApplyOperations.get(event.value.operationHandle)
        if (!pending) return
        this.connectionPolicyApplyOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.state) pending.reject(apiError(event.value.error, 'connection policy apply failed'))
        else pending.resolve(event.value.state)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'connectionSnapshotGet': {
        const pending = this.connectionSnapshotGetOperations.get(event.value.operationHandle)
        if (!pending) return
        this.connectionSnapshotGetOperations.delete(event.value.operationHandle)
        if (event.value.error || !event.value.connection) pending.reject(apiError(event.value.error, 'connection snapshot get failed'))
        else pending.resolve(event.value.connection)
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'sessionInvalidate': {
        const pending = this.sessionInvalidateOperations.get(event.value.operationHandle)
        if (!pending) return
        this.sessionInvalidateOperations.delete(event.value.operationHandle)
        if (event.value.error) pending.reject(apiError(event.value.error, 'session invalidation failed'))
        else pending.resolve()
        this.releaseCompletedOperation(event.value.operationHandle)
        return
      }
      case 'application':
        this.sessions.get(event.value.sessionHandle)?.publish(event.value.event)
        return
      case 'sessionClosed':
    if (this.releasedSessionHandles.has(event.value.sessionHandle)) return
    rememberHandle(this.releasedSessionHandles, this.releasedSessionOrder, event.value.sessionHandle)
    if (!this.retiredSessionHandles.delete(event.value.sessionHandle)) {
      this.sessions.get(event.value.sessionHandle)?.markClosed(apiError(event.value.error, 'session closed'))
      this.sessions.delete(event.value.sessionHandle)
    }
    void this.release(event.value.sessionHandle).catch((error) => this.onClosed(new Error(`closed session release failed: ${errorMessage(error)}`)))
        return
      case 'resourceStreamFrame': {
        if (this.abandonedStreamHandles.has(event.value.streamHandle)) return
        const stream = this.streams.get(event.value.streamHandle)
        if (stream) stream.publish(event.value.type, event.value.payload)
        else this.queueEarlyStreamEvent(event.value.streamHandle, envelope)
        return
      }
      case 'resourceStreamClosed': {
        if (this.abandonedStreamHandles.delete(event.value.streamHandle)) {
          this.earlyStreamEvents.delete(event.value.streamHandle)
      if (this.releasedStreamHandles.has(event.value.streamHandle)) return
      rememberHandle(this.releasedStreamHandles, this.releasedStreamOrder, event.value.streamHandle)
          return
        }
        const stream = this.streams.get(event.value.streamHandle)
        if (!stream) {
          this.queueEarlyStreamEvent(event.value.streamHandle, envelope)
          return
        }
    if (this.releasedStreamHandles.has(event.value.streamHandle)) return
    rememberHandle(this.releasedStreamHandles, this.releasedStreamOrder, event.value.streamHandle)
        stream.markClosed(apiError(event.value.error, 'resource stream closed'))
        this.streams.delete(event.value.streamHandle)
        void this.release(event.value.streamHandle).catch((error) => this.onClosed(new Error(`closed resource stream release failed: ${errorMessage(error)}`)))
        return
      }
    }
  }

  private hasPendingOperation(eventCase: AnyTTYClientBinding.EventEnvelope['event']['case'], handle: bigint): boolean {
    switch (eventCase) {
      case 'openSession': return this.openOperations.has(handle)
      case 'execute': return this.executeOperations.has(handle)
      case 'importPairing': return this.importOperations.has(handle)
      case 'deleteCredential': return this.deleteOperations.has(handle)
      case 'endpointRegistryGet': return this.registryGetOperations.has(handle)
      case 'endpointUpsert': return this.endpointUpsertOperations.has(handle)
      case 'endpointDelete': return this.endpointDeleteOperations.has(handle)
      case 'endpointShareReceive': return this.endpointShareReceiveOperations.has(handle)
      case 'endpointShareCommit': return this.endpointShareCommitOperations.has(handle)
      case 'sshCredentialProvision': return this.sshCredentialProvisionOperations.has(handle)
      case 'connectionPolicyGet': return this.connectionPolicyGetOperations.has(handle)
      case 'connectionPolicyApply': return this.connectionPolicyApplyOperations.has(handle)
      case 'connectionSnapshotGet': return this.connectionSnapshotGetOperations.has(handle)
      case 'sessionInvalidate': return this.sessionInvalidateOperations.has(handle)
      default: return true
    }
  }

  private onClosed(error: Error): void {
    if (this.closed) return
    this.closed = true
    void this.backend.close().catch(() => undefined)
    this.rejectAll(error)
    this.sessions.forEach((session) => session.markClosed(error))
    this.streams.forEach((stream) => stream.markClosed(error))
    this.sessions.clear()
    this.streams.clear()
    if (!this.intentionalClose) document.dispatchEvent(new CustomEvent('anytty:binding-closed', { detail: error.message }))
  }

  private rejectAll(error: Error): void {
    for (const registry of [this.openOperations, this.executeOperations, this.importOperations, this.deleteOperations, this.registryGetOperations, this.endpointUpsertOperations, this.endpointDeleteOperations, this.endpointShareReceiveOperations, this.endpointShareCommitOperations, this.sshCredentialProvisionOperations, this.connectionPolicyGetOperations, this.connectionPolicyApplyOperations, this.connectionSnapshotGetOperations, this.sessionInvalidateOperations]) {
      for (const pending of registry.values()) pending.reject(error)
      registry.clear()
    }
    this.earlyOperationEvents.clear()
    this.cancelledOperations.clear()
    this.earlyStreamEvents.clear()
    this.abandonedStreamHandles.clear()
    this.retiredSessionHandles.clear()
  this.releasedSessionHandles.clear()
  this.releasedSessionOrder.length = 0
  this.releasedStreamHandles.clear()
  this.releasedStreamOrder.length = 0
  }

  private queueEarlyStreamEvent(handle: bigint, envelope: AnyTTYClientBinding.EventEnvelope): void {
    if (this.abandonedStreamHandles.has(handle)) return
    const pending = this.earlyStreamEvents.get(handle) ?? []
    if (pending.length >= MAX_EARLY_STREAM_EVENTS) {
      this.onClosed(new Error('Proto binding early stream event queue overflow'))
      return
    }
    pending.push(envelope)
    this.earlyStreamEvents.set(handle, pending)
  }

  private abandonResourceStream(handle: bigint): void {
    const early = this.earlyStreamEvents.get(handle)
    const terminalClosed = early?.some((event) => event.event.case === 'resourceStreamClosed') === true
    this.earlyStreamEvents.delete(handle)
  if (terminalClosed) rememberHandle(this.releasedStreamHandles, this.releasedStreamOrder, handle)
    if (!terminalClosed) {
      if (this.abandonedStreamHandles.size >= MAX_ABANDONED_STREAM_HANDLES) {
        this.onClosed(new Error('Proto binding abandoned stream handle capacity is exhausted'))
        return
      }
      this.abandonedStreamHandles.add(handle)
    }
    void this.closeResourceStream(handle)
      .then(() => this.release(handle))
      .catch((error) => this.onClosed(new Error(`abandoned resource stream cleanup failed: ${errorMessage(error)}`)))
  }

  private async cleanupCancelledExecute(result: AnyTTYClientBinding.ExecuteResult, ownsHistorySnapshot: boolean): Promise<void> {
    if (result.error && result.error.code !== AnyTTYApiCommon.ApiErrorCode.CANCELLED) {
      throw apiError(result.error, 'cancelled operation cleanup failed')
    }
    const session = this.sessions.get(result.sessionHandle)
    if (!session) return
    const value = result.result?.result
    const transfer = value?.case === 'fileTransferOpen' ? value.value.transfer : undefined
    const history = value?.case === 'historyWindow' ? value.value : undefined
    const subscription = value?.case === 'eventSubscription' ? value.value.subscription : undefined
    let cleanup: AnyTTYApiApplication.CommandEnvelope | undefined
    if (transfer?.resource) {
      cleanup = transfer.resume
        ? create(AnyTTYApiApplication.CommandEnvelopeSchema, { command: { case: 'fileTransferCancel', value: create(AnyTTYApiFile.FileTransferCancelCommandSchema, { uploadResume: transfer.resume }) } })
        : create(AnyTTYApiApplication.CommandEnvelopeSchema, { command: { case: 'releaseResource', value: create(AnyTTYApiApplication.ReleaseResourceCommandSchema, { resource: transfer.resource }) } })
    } else if (ownsHistorySnapshot && history?.terminal && history.token) {
      cleanup = create(AnyTTYApiApplication.CommandEnvelopeSchema, { command: { case: 'historyRelease', value: create(AnyTTYApiHistory.HistoryReleaseCommandSchema, {
        terminal: history.terminal, token: history.token, historyGeneration: history.historyGeneration,
      }) } })
    } else if (subscription) {
      cleanup = create(AnyTTYApiApplication.CommandEnvelopeSchema, { command: { case: 'releaseResource', value: create(AnyTTYApiApplication.ReleaseResourceCommandSchema, { resource: subscription }) } })
    }
    if (!cleanup) return
    const controller = new AbortController()
    const timeout = setTimeout(() => controller.abort(new DOMException('cancelled operation cleanup timed out', 'TimeoutError')), CANCELLED_CLEANUP_TIMEOUT_MS)
    try {
      const cleanupResult = await session.execute(cleanup, { signal: controller.signal })
      if (transfer?.resume && (cleanupResult.result.case !== 'fileTransferCancel' || !cleanupResult.result.value.cancelled)) {
        throw new Error('cancelled upload cleanup was not confirmed')
      }
    } finally {
      clearTimeout(timeout)
    }
  }

  private retireCancelledOperation(operationHandle: bigint, envelope: AnyTTYClientBinding.EventEnvelope, ownsHistorySnapshot = false): void {
    const event = envelope.event
    if (event.case === 'execute') {
      void this.cleanupCancelledExecute(event.value, ownsHistorySnapshot).catch((error) => this.onClosed(new Error(`cancelled operation cleanup failed: ${errorMessage(error)}`)))
    } else if (event.case === 'openSession' && event.value.sessionHandle !== 0n) {
    if (this.retiredSessionHandles.size >= MAX_ABANDONED_STREAM_HANDLES) {
    this.onClosed(new Error('Proto binding retired session handle capacity is exhausted'))
    return
    }
    this.retiredSessionHandles.add(event.value.sessionHandle)
      void this.closeSession(event.value.sessionHandle)
        .catch((error) => this.onClosed(new Error(`cancelled session cleanup failed: ${errorMessage(error)}`)))
    }
    void this.release(operationHandle).catch((error) => this.onClosed(new Error(`cancelled operation release failed: ${errorMessage(error)}`)))
  }
}

export interface EndpointInput {
  endpointId: string
  routeId?: string
}

/** ProtoBindingConnector builds only OpenSessionRequest; route/auth/reconnect ownership remains in Go. */
export class ProtoBindingConnector {
  constructor(private readonly client: () => ProtoBindingClient, private readonly input: EndpointInput) {}

  async connect(input: { machineId: string }, options?: { signal?: AbortSignal; forceRelay?: boolean; onStatus?: (status: string) => void; onConnectionState?: (snapshot: { machineId: string; phase: 'connecting' | 'connected' | 'failed'; statusText: string; relayInUse: boolean }) => void }): Promise<ProtoClientSession> {
  const endpointId = this.input.endpointId.trim()
    if (!endpointId || input.machineId !== endpointId) throw new Error('endpoint identity mismatch')
    options?.onStatus?.('Connecting...')
    options?.onConnectionState?.({ machineId: input.machineId, phase: 'connecting', statusText: 'Connecting...', relayInUse: options.forceRelay === true })
    try {
      const session = await this.client().openSession(create(AnyTTYClientBinding.OpenSessionRequestSchema, {
    requestId: crypto.randomUUID(), endpointId, routeOverride: this.input.routeId ?? '',
    intent: AnyTTYClientBinding.ConnectIntent.INTERACTIVE,
      }), options?.signal)
      options?.onStatus?.('Connected')
      options?.onConnectionState?.({
    machineId: input.machineId,
    phase: 'connected',
    statusText: 'Connected',
    relayInUse: session.connection?.localCandidateType === AnyTTYClientBinding.ConnectionCandidateType.RELAY,
    })
      return session
    } catch (error) {
      options?.onConnectionState?.({ machineId: input.machineId, phase: 'failed', statusText: error instanceof Error ? error.message : 'Connection failed', relayInUse: options?.forceRelay === true })
      throw error
    }
  }

  async getConnectionPolicy(signal?: AbortSignal): Promise<ConnectionPolicyState> {
    return connectionPolicyStateFromProto(await this.client().getConnectionPolicy(this.input.endpointId, signal))
  }

  async applyConnectionPolicy(policy: ConnectionPolicy, signal?: AbortSignal): Promise<void> {
    await this.client().applyConnectionPolicy(this.input.endpointId, create(AnyTTYClientBinding.ConnectionPolicySchema, {
      routePreference: routePreferenceToProto(policy.route),
      cloudRelayMode: cloudPreferenceToProto(policy.cloud),
      relayTransport: relayTransportToProto(policy.relayTransport),
    }), signal)
  }
}

class ProtoBindingSession implements ProtoClientSession {
  private alive = true
  private readonly eventHandlers = new Set<(event: AnyTTYApiApplication.EventEnvelope) => void>()
  private readonly closeHandlers = new Set<(error: ProtoClientSessionCloseError) => void>()
  private closeError: ProtoClientSessionCloseError | null = null

  constructor(
  private readonly client: ProtoBindingClient,
  readonly handle: bigint,
  readonly stamp: NonNullable<AnyTTYClientBinding.OpenSessionResult['session']>,
  public connection?: AnyTTYClientBinding.ConnectionSnapshot,
  ) {}

  execute(command: AnyTTYApiApplication.CommandEnvelope, options?: { signal?: AbortSignal }): Promise<AnyTTYApiApplication.ResultEnvelope> {
    if (!this.alive) return Promise.reject(new Error('Proto session is closed'))
    return this.client.execute(this.handle, command, options?.signal).catch((error: unknown) => {
      if (isSessionInvalidationError(error)) {
        this.markClosed(error instanceof Error ? error : new Error(String(error)))
        document.dispatchEvent(new CustomEvent('anytty:session-invalidated', {
          detail: error instanceof Error ? error.message : String(error),
        }))
      }
      throw error
    })
  }

  subscribeEvents(handler: (event: AnyTTYApiApplication.EventEnvelope) => void): ProtoClientSubscription {
    this.eventHandlers.add(handler)
    return { close: () => this.eventHandlers.delete(handler) }
  }

  subscribeClosed(handler: (error: ProtoClientSessionCloseError) => void): ProtoClientSubscription {
    const error = this.closeError
    if (error) {
      let subscribed = true
      queueMicrotask(() => {
        if (subscribed) handler(error)
      })
      return { close: () => { subscribed = false } }
    }
    if (!this.alive) return { close() {} }
    this.closeHandlers.add(handler)
    return { close: () => this.closeHandlers.delete(handler) }
  }

  openResourceStream(resource: NonNullable<AnyTTYClientBinding.OpenResourceStreamRequest['resource']>, options?: { initialUploadOffset?: bigint; signal?: AbortSignal }): Promise<ProtoResourceStream> {
    if (!this.alive) return Promise.reject(new Error('Proto session is closed'))
    return this.client.openResourceStream(this.handle, resource, options)
  }

  isAlive(): boolean { return this.alive }

  async getConnectionSnapshot(): Promise<AnyTTYClientBinding.ConnectionSnapshot | undefined> {
    if (!this.alive) throw new Error('Proto session is closed')
    this.connection = await this.client.getConnectionSnapshot(this.handle)
    return this.connection
  }

  async invalidate(): Promise<void> {
    if (!this.alive) return
    await this.client.invalidateSession(this.handle)
  }

  async close(): Promise<void> {
    if (!this.alive) return
    this.alive = false
    this.eventHandlers.clear()
    this.closeHandlers.clear()
    await this.client.closeSession(this.handle)
  }

  publish(event: AnyTTYApiApplication.EventEnvelope | undefined): void {
    if (!event || !this.alive) return
    this.eventHandlers.forEach((handler) => handler(event))
  }

  markClosed(error?: ProtoClientSessionCloseError): void {
    if (!this.alive) return
    this.alive = false
    this.eventHandlers.clear()
    const handlers = [...this.closeHandlers]
    this.closeHandlers.clear()
    if (!error) return
    this.closeError = error
    handlers.forEach((handler) => handler(error))
  }
}

class ProtoBindingResourceStream implements ProtoResourceStream {
  private closed = false
  private readonly handlers = new Set<(type: AnyTTYClientBinding.ResourceStreamFrameType, payload: Uint8Array) => void>()
  private readonly closeHandlers = new Set<(error: Error) => void>()

  constructor(private readonly client: ProtoBindingClient, readonly handle: bigint) {}

  send(type: AnyTTYClientBinding.ResourceStreamFrameType, payload: Uint8Array): Promise<void> {
    if (this.closed) return Promise.reject(new Error('Proto resource stream is closed'))
    return this.client.sendResourceStreamFrame(this.handle, type, payload)
  }

  subscribe(handler: (type: AnyTTYClientBinding.ResourceStreamFrameType, payload: Uint8Array) => void): ProtoClientSubscription {
    this.handlers.add(handler)
    return { close: () => this.handlers.delete(handler) }
  }

  subscribeClosed(handler: (error: Error) => void): ProtoClientSubscription {
    this.closeHandlers.add(handler)
    return { close: () => this.closeHandlers.delete(handler) }
  }

  async close(): Promise<void> {
    if (this.closed) return
    this.closed = true
    await this.client.closeResourceStream(this.handle)
  }

  publish(type: AnyTTYClientBinding.ResourceStreamFrameType, payload: Uint8Array): void {
    if (this.closed) return
    this.handlers.forEach((handler) => handler(type, payload.slice()))
  }

  markClosed(error: Error): void {
    if (this.closed) return
    this.closed = true
    this.closeHandlers.forEach((handler) => handler(error))
    this.handlers.clear()
    this.closeHandlers.clear()
  }
}

function apiError(error: { code?: AnyTTYApiCommon.ApiErrorCode, message?: string, retryable?: boolean } | undefined, fallback: string): Error {
  const result = new Error(error?.message || fallback) as Error & { code?: string, retryable?: boolean }
  const code = error ? apiErrorCode(error.code) : ''
  if (code) result.code = code
  if (typeof error?.retryable === 'boolean') result.retryable = error.retryable
  return result
}

function isSessionInvalidationError(error: unknown): boolean {
  if (!(error instanceof Error)) return false
  const code = (error as Error & { code?: string }).code
  if (code === 'stale_session') return true
  return code === 'unavailable' && /(?:client|application) session is unavailable/i.test(error.message)
}

function routePreferenceFromProto(value: AnyTTYRemoteAuth.EndpointRoutePreference | undefined): ConnectionPolicy['route'] {
  switch (value) {
    case AnyTTYRemoteAuth.EndpointRoutePreference.DIRECT: return 'direct'
    case AnyTTYRemoteAuth.EndpointRoutePreference.SSH: return 'ssh'
    case AnyTTYRemoteAuth.EndpointRoutePreference.MANAGED_CLOUD: return 'cloud'
    default: return 'auto'
  }
}

function routePreferenceToProto(value: ConnectionPolicy['route']): AnyTTYRemoteAuth.EndpointRoutePreference {
  switch (value) {
    case 'direct': return AnyTTYRemoteAuth.EndpointRoutePreference.DIRECT
    case 'ssh': return AnyTTYRemoteAuth.EndpointRoutePreference.SSH
    case 'cloud': return AnyTTYRemoteAuth.EndpointRoutePreference.MANAGED_CLOUD
    default: return AnyTTYRemoteAuth.EndpointRoutePreference.AUTO
  }
}

function cloudPreferenceFromProto(value: AnyTTYRemoteAuth.ManagedWebRTCRelayMode): ConnectionPolicy['cloud'] {
  switch (value) {
    case AnyTTYRemoteAuth.ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_DIRECT: return 'p2p'
    case AnyTTYRemoteAuth.ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY: return 'relay'
    default: return 'auto'
  }
}

function cloudPreferenceToProto(value: ConnectionPolicy['cloud']): AnyTTYRemoteAuth.ManagedWebRTCRelayMode {
  switch (value) {
    case 'p2p': return AnyTTYRemoteAuth.ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_DIRECT
    case 'relay': return AnyTTYRemoteAuth.ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY
    default: return AnyTTYRemoteAuth.ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_AUTO
  }
}

function relayTransportFromProto(value: AnyTTYRemoteAuth.ManagedWebRTCRelayTransport): ConnectionPolicy['relayTransport'] {
  switch (value) {
    case AnyTTYRemoteAuth.ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_UDP: return 'udp'
    case AnyTTYRemoteAuth.ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_TCP: return 'tcp'
    default: return 'auto'
  }
}

function relayTransportToProto(value: ConnectionPolicy['relayTransport']): AnyTTYRemoteAuth.ManagedWebRTCRelayTransport {
  switch (value) {
    case 'udp': return AnyTTYRemoteAuth.ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_UDP
    case 'tcp': return AnyTTYRemoteAuth.ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_TCP
    default: return AnyTTYRemoteAuth.ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO
  }
}

function connectionPolicyStateFromProto(state: AnyTTYClientBinding.ConnectionPolicyState): ConnectionPolicyState {
  const policy = state.policy
  const result: ConnectionPolicyState = {
    policy: {
      route: routePreferenceFromProto(policy?.routePreference),
      cloud: cloudPreferenceFromProto(policy?.cloudRelayMode ?? AnyTTYRemoteAuth.ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_AUTO),
      relayTransport: relayTransportFromProto(policy?.relayTransport ?? AnyTTYRemoteAuth.ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO),
    },
    available: { direct: false, ssh: false, cloud: false },
    unavailableReasons: {},
  }
  for (const route of state.routes) {
    const key = route.routeKind === AnyTTYClientBinding.ConnectionRouteKind.DIRECT
      ? 'direct'
      : route.routeKind === AnyTTYClientBinding.ConnectionRouteKind.SSH
        ? 'ssh'
        : route.routeKind === AnyTTYClientBinding.ConnectionRouteKind.CLOUD
          ? 'cloud'
          : undefined
    if (!key) continue
    result.available[key] = route.available
    if (!route.available) result.unavailableReasons[key] = connectionPolicyReasonFromProto(route.reason)
  }
  return result
}

function connectionPolicyReasonFromProto(reason: AnyTTYClientBinding.ConnectionPolicyAvailabilityReason): NonNullable<ConnectionPolicyState['unavailableReasons']['direct']> {
  switch (reason) {
    case AnyTTYClientBinding.ConnectionPolicyAvailabilityReason.ROUTE_DISABLED: return 'route_disabled'
    case AnyTTYClientBinding.ConnectionPolicyAvailabilityReason.PLATFORM_UNSUPPORTED: return 'platform_unsupported'
    case AnyTTYClientBinding.ConnectionPolicyAvailabilityReason.CREDENTIAL_UNAVAILABLE: return 'credential_unavailable'
    case AnyTTYClientBinding.ConnectionPolicyAvailabilityReason.CLOUD_UNAVAILABLE: return 'cloud_unavailable'
    default: return 'route_not_configured'
  }
}

function apiErrorCode(code: AnyTTYApiCommon.ApiErrorCode | undefined): string {
  switch (code) {
    case AnyTTYApiCommon.ApiErrorCode.INVALID_REQUEST: return 'invalid_request'
    case AnyTTYApiCommon.ApiErrorCode.UNSUPPORTED_VERSION: return 'unsupported_version'
    case AnyTTYApiCommon.ApiErrorCode.UNSUPPORTED_CAPABILITY: return 'unsupported_capability'
    case AnyTTYApiCommon.ApiErrorCode.UNAUTHORIZED: return 'unauthenticated'
    case AnyTTYApiCommon.ApiErrorCode.FORBIDDEN: return 'forbidden'
    case AnyTTYApiCommon.ApiErrorCode.NOT_FOUND: return 'not_found'
    case AnyTTYApiCommon.ApiErrorCode.CONFLICT: return 'conflict'
    case AnyTTYApiCommon.ApiErrorCode.STALE_SESSION: return 'stale_session'
    case AnyTTYApiCommon.ApiErrorCode.CANCELLED: return 'cancelled'
    case AnyTTYApiCommon.ApiErrorCode.UNAVAILABLE: return 'unavailable'
    case AnyTTYApiCommon.ApiErrorCode.INTERNAL: return 'internal'
    case AnyTTYApiCommon.ApiErrorCode.ENTITLEMENT_DENIED: return 'entitlement_denied'
    case AnyTTYApiCommon.ApiErrorCode.RESOURCE_EXHAUSTED: return 'resource_exhausted'
    case AnyTTYApiCommon.ApiErrorCode.STALE_RESOURCE: return 'stale_resource'
    case AnyTTYApiCommon.ApiErrorCode.DAEMON_BLOCKED: return 'daemon_blocked'
    case AnyTTYApiCommon.ApiErrorCode.DAEMON_DELETED: return 'daemon_deleted'
    case AnyTTYApiCommon.ApiErrorCode.RELAY_NOT_IN_PLAN: return 'relay_not_in_plan'
    case AnyTTYApiCommon.ApiErrorCode.RELAY_QUOTA_EXHAUSTED: return 'relay_quota_exhausted'
    case AnyTTYApiCommon.ApiErrorCode.RELAY_CONCURRENCY_EXHAUSTED: return 'relay_concurrency_exhausted'
    case AnyTTYApiCommon.ApiErrorCode.SUBSCRIPTION_INACTIVE: return 'subscription_inactive'
    case AnyTTYApiCommon.ApiErrorCode.RELAY_REGION_UNAVAILABLE: return 'relay_region_unavailable'
    default: return ''
  }
}

function bindingOperationHandle(envelope: AnyTTYClientBinding.EventEnvelope): bigint | undefined {
  switch (envelope.event.case) {
    case 'openSession':
    case 'execute':
    case 'importPairing':
    case 'deleteCredential':
    case 'endpointRegistryGet':
    case 'endpointUpsert':
    case 'endpointDelete':
    case 'endpointShareReceive':
    case 'endpointShareCommit':
    case 'sshCredentialProvision':
    case 'connectionPolicyGet':
    case 'connectionPolicyApply':
    case 'connectionSnapshotGet':
    case 'sessionInvalidate':
      return envelope.event.value.operationHandle
    default:
      return undefined
  }
}

function abortError(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new DOMException('Aborted', 'AbortError')
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}

function rememberHandle(handles: Set<bigint>, order: bigint[], handle: bigint): void {
  if (handles.has(handle)) return
  if (order.length >= MAX_ABANDONED_STREAM_HANDLES) {
    const oldest = order.shift()
    if (oldest !== undefined) handles.delete(oldest)
  }
  handles.add(handle)
  order.push(handle)
}

async function awaitAbortableHandle(promise: Promise<bigint>, signal: AbortSignal | undefined, onLate: (handle: bigint) => void): Promise<bigint> {
  if (!signal) return await promise
  if (signal.aborted) {
    void promise.then(onLate, () => undefined)
    throw abortError(signal)
  }
  return await new Promise<bigint>((resolve, reject) => {
    let settled = false
    const abort = () => {
      if (settled) return
      settled = true
      signal.removeEventListener('abort', abort)
      reject(abortError(signal))
    }
    signal.addEventListener('abort', abort, { once: true })
    void promise.then(
      (handle) => {
        if (settled) {
          onLate(handle)
          return
        }
        settled = true
        signal.removeEventListener('abort', abort)
        resolve(handle)
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
