import { create } from '@bufbuild/protobuf'
import type { CommandEnvelope, EventEnvelope, ResultEnvelope } from '../generated/apipb/application_pb'
import { ResultEnvelopeSchema } from '../generated/apipb/application_pb'
import { EndpointSessionStampSchema, type ResourceHandle } from '../generated/apipb/common_pb'
import { ResourceStreamFrameType } from '../generated/bindingpb/client_binding_pb'
import type { ProtoClientSession, ProtoClientSessionCloseError, ProtoClientSubscription, ProtoResourceStream } from '../core/protoClientSession'

/** ProtoCommandHandler 以 generated command/result 驱动 UI 单元测试，不复制业务 DTO。 */
export type ProtoCommandHandler = (command: CommandEnvelope) => ResultEnvelope | Promise<ResultEnvelope>

/** MockProtoSession 让 UI 测试经过 JNI/WASM 生产路径共用的 generated Proto session contract。 */
export class MockProtoSession implements ProtoClientSession {
  readonly commands: CommandEnvelope[] = []
  readonly executeSignals: Array<AbortSignal | undefined> = []
  readonly openedResources: ResourceHandle[] = []
  readonly stamp
  private alive = true
  private readonly eventHandlers = new Set<(event: EventEnvelope) => void>()
  private readonly closeHandlers = new Set<(error: ProtoClientSessionCloseError) => void>()

  constructor(
    endpointId = 'machine-local',
    private readonly handler: ProtoCommandHandler = () => protoResult('acknowledge', {}),
    private readonly streamFactory: (resource: ResourceHandle) => ProtoResourceStream = () => new MockProtoResourceStream(),
  ) {
    this.stamp = create(EndpointSessionStampSchema, { endpointId, generation: 1n, sessionId: 'test-session' })
  }

  async execute(command: CommandEnvelope, options?: { signal?: AbortSignal }): Promise<ResultEnvelope> {
    if (!this.alive) throw new Error('mock Proto session is closed')
    this.commands.push(command)
    this.executeSignals.push(options?.signal)
    const result = Promise.resolve(this.handler(command))
    const signal = options?.signal
    if (!signal) return await result
    if (signal.aborted) throw abortReason(signal)
    return await new Promise<ResultEnvelope>((resolve, reject) => {
      const abort = () => reject(abortReason(signal))
      signal.addEventListener('abort', abort, { once: true })
      void result.then(
        (value) => {
          signal.removeEventListener('abort', abort)
          resolve(value)
        },
        (error) => {
          signal.removeEventListener('abort', abort)
          reject(error)
        },
      )
    })
  }

  subscribeEvents(handler: (event: EventEnvelope) => void): ProtoClientSubscription {
    this.eventHandlers.add(handler)
    return { close: () => this.eventHandlers.delete(handler) }
  }

  subscribeClosed(handler: (error: ProtoClientSessionCloseError) => void): ProtoClientSubscription {
    this.closeHandlers.add(handler)
    return { close: () => this.closeHandlers.delete(handler) }
  }

  emit(event: EventEnvelope): void {
    for (const handler of this.eventHandlers) handler(event)
  }

  emitClosed(error: ProtoClientSessionCloseError): void {
    if (!this.alive) return
    this.alive = false
    this.eventHandlers.clear()
    const handlers = [...this.closeHandlers]
    this.closeHandlers.clear()
    for (const handler of handlers) handler(error)
  }

  async openResourceStream(resource: ResourceHandle, _options?: { initialUploadOffset?: bigint; signal?: AbortSignal }): Promise<ProtoResourceStream> {
    this.openedResources.push(resource)
    return this.streamFactory(resource)
  }

  isAlive(): boolean { return this.alive }

  async close(): Promise<void> {
    this.alive = false
    this.eventHandlers.clear()
    this.closeHandlers.clear()
  }
}

function abortReason(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new DOMException('Aborted', 'AbortError')
}

/** MockProtoResourceStream 模拟 binding 的 opaque resource stream，只记录 framing payload。 */
export class MockProtoResourceStream implements ProtoResourceStream {
  readonly handle = 1n
  readonly sent: Array<{ type: ResourceStreamFrameType; payload: Uint8Array }> = []
  private readonly handlers = new Set<(type: ResourceStreamFrameType, payload: Uint8Array) => void>()
  private readonly closedHandlers = new Set<(error: Error) => void>()

  constructor(private readonly frames: Array<{ type: ResourceStreamFrameType; payload: Uint8Array }> = []) {}

  async send(type: ResourceStreamFrameType, payload: Uint8Array): Promise<void> {
    this.sent.push({ type, payload: payload.slice() })
  }

  subscribe(handler: (type: ResourceStreamFrameType, payload: Uint8Array) => void): ProtoClientSubscription {
    this.handlers.add(handler)
    queueMicrotask(() => {
      for (const frame of this.frames) handler(frame.type, frame.payload.slice())
    })
    return { close: () => this.handlers.delete(handler) }
  }

  subscribeClosed(handler: (error: Error) => void): ProtoClientSubscription {
    this.closedHandlers.add(handler)
    return { close: () => this.closedHandlers.delete(handler) }
  }

  async close(): Promise<void> {}
}

/** protoResult 构造 generated ResultEnvelope，避免测试重新定义平行业务结果结构。 */
export function protoResult(caseName: string, value: object): ResultEnvelope {
  return create(ResultEnvelopeSchema, { result: { case: caseName, value } } as never)
}
