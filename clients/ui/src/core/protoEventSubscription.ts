import { create } from '@bufbuild/protobuf'
import { CommandEnvelopeSchema, ReleaseResourceCommandSchema } from '../generated/apipb/application_pb'
import type { EventSubscribeCommand } from '../generated/apipb/events_pb'
import type { ProtoClientSession, ProtoClientSubscription } from './protoClientSession'
import type { EventEnvelope } from '../generated/apipb/application_pb'

const MAX_EARLY_SUBSCRIPTION_EVENTS = 64

/**
 * openProtoEventSubscription 建立 daemon-owned event subscription，并把 binding event sink 与资源释放绑定在一起。
 * 本地 sink 必须先注册，避免 EventSubscribe result 返回后首个事件早于 TypeScript handler；关闭只释放当前 session 签发的 subscription resource。
 */
export async function openProtoEventSubscription(
  session: ProtoClientSession,
  command: EventSubscribeCommand,
  handler: (event: EventEnvelope) => void,
): Promise<ProtoClientSubscription> {
  let resource: EventEnvelope['subscription'] | undefined
  let earlyError: Error | null = null
  const early: EventEnvelope[] = []
  const local = session.subscribeEvents((event) => {
    if (!resource) {
      if (early.length >= MAX_EARLY_SUBSCRIPTION_EVENTS) {
        earlyError = new Error('event subscription correlation buffer overflow')
        return
      }
      early.push(event)
      return
    }
    if (sameResourceHandle(event.subscription, resource)) handler(event)
  })
  try {
    const result = await session.execute(create(CommandEnvelopeSchema, {
      command: { case: 'eventSubscribe', value: command },
    }))
    if (result.result.case !== 'eventSubscription' || !result.result.value.subscription) {
      throw new Error('event subscribe returned no subscription resource')
    }
    resource = result.result.value.subscription
    if (earlyError) {
      void session.execute(create(CommandEnvelopeSchema, {
        command: { case: 'releaseResource', value: create(ReleaseResourceCommandSchema, { resource }) },
      })).catch(() => undefined)
      throw earlyError
    }
    for (const event of early) {
      if (sameResourceHandle(event.subscription, resource)) handler(event)
    }
    early.length = 0
    let closed = false
    return {
      close() {
        if (closed) return
        closed = true
        local.close()
        void session.execute(create(CommandEnvelopeSchema, {
          command: { case: 'releaseResource', value: create(ReleaseResourceCommandSchema, { resource }) },
        })).catch(() => undefined)
      },
    }
  } catch (error) {
    local.close()
    throw error
  }
}

function sameResourceHandle(left: EventEnvelope['subscription'], right: NonNullable<EventEnvelope['subscription']>): boolean {
  if (!left || left.kind !== right.kind || left.generation !== right.generation) return false
  if (!sameBytes(left.opaqueToken, right.opaqueToken)) return false
  const leftSession = left.session
  const rightSession = right.session
  return Boolean(leftSession && rightSession &&
    leftSession.endpointId === rightSession.endpointId &&
    leftSession.routeId === rightSession.routeId &&
    leftSession.generation === rightSession.generation)
}

function sameBytes(left: Uint8Array, right: Uint8Array): boolean {
  if (left.byteLength !== right.byteLength) return false
  for (let index = 0; index < left.byteLength; index += 1) {
    if (left[index] !== right[index]) return false
  }
  return true
}
