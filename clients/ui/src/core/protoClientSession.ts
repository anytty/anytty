import type { CommandEnvelope, EventEnvelope, ResultEnvelope } from '../generated/apipb/application_pb'
import type { EndpointSessionStamp, ResourceHandle } from '../generated/apipb/common_pb'
import type { ResourceStreamFrameType } from '../generated/bindingpb/client_binding_pb'
import type { ConnectionSnapshot } from '../generated/bindingpb/client_binding_pb'

export interface ProtoClientSubscription {
  close(): void
}

export interface ProtoClientSessionCloseError extends Error {
  code?: string
  retryable?: boolean
}

export interface ProtoResourceStream {
  readonly handle: bigint
  send(type: ResourceStreamFrameType, payload: Uint8Array): Promise<void>
  subscribe(handler: (type: ResourceStreamFrameType, payload: Uint8Array) => void): ProtoClientSubscription
  subscribeClosed(handler: (error: Error) => void): ProtoClientSubscription
  close(): Promise<void>
}

/**
 * ProtoClientSession 是 UI 到跨端 Go Client Engine 的唯一 application contract。
 * command/result/event 真值来自 generated apipb；实现只拥有当前 binding session handle 与 generation，不解析 transport framing。
 */
export interface ProtoClientSession {
  readonly stamp: EndpointSessionStamp
  readonly connection?: ConnectionSnapshot | undefined
  getConnectionSnapshot?(): Promise<ConnectionSnapshot | undefined>
  execute(command: CommandEnvelope, options?: { signal?: AbortSignal }): Promise<ResultEnvelope>
  subscribeEvents(handler: (event: EventEnvelope) => void): ProtoClientSubscription
  subscribeClosed(handler: (error: ProtoClientSessionCloseError) => void): ProtoClientSubscription
  openResourceStream(resource: ResourceHandle, options?: { initialUploadOffset?: bigint; signal?: AbortSignal }): Promise<ProtoResourceStream>
  isAlive(): boolean
  /** Invalidates the exact Go-owned generation after a confirmed platform network change. */
  invalidate?(): Promise<void>
  close(): Promise<void>
}
