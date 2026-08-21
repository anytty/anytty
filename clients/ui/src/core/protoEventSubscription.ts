import { create, toBinary } from '@bufbuild/protobuf'
import { CommandEnvelopeSchema, ReleaseResourceCommandSchema } from '../generated/apipb/application_pb'
import { EventSubscribeCommandSchema, type EventSubscribeCommand } from '../generated/apipb/events_pb'
import type { ProtoClientSession, ProtoClientSubscription } from './protoClientSession'
import type { EventEnvelope } from '../generated/apipb/application_pb'

const MAX_EARLY_SUBSCRIPTION_EVENTS = 64

type SubscriptionResource = NonNullable<EventEnvelope['subscription']>

type SharedEventSubscription = {
  consumers: number
  resource: SubscriptionResource | undefined
  opening: Promise<SubscriptionResource>
  local: ProtoClientSubscription
  listeners: Set<(event: EventEnvelope) => void>
  replay: Map<string, EventEnvelope>
}

const sessionSubscriptionPools = new WeakMap<ProtoClientSession, Map<string, SharedEventSubscription>>()

/**
 * openProtoEventSubscription 按 physical session 与完整 filter 复用 daemon-owned subscription。
 * session 级 hub 负责本地分发并重放当前 attachment projection；最后一个 consumer 关闭时释放远端资源。
 */
export async function openProtoEventSubscription(
  session: ProtoClientSession,
  command: EventSubscribeCommand,
  handler: (event: EventEnvelope) => void,
): Promise<ProtoClientSubscription> {
  const poolOwner = session.resourcePoolOwner ?? session
  const pool = subscriptionPool(poolOwner)
  const poolKey = eventSubscriptionKey(command)
  let shared = pool.get(poolKey)
  if (!shared) {
    shared = createSharedSubscription(poolOwner, session, command)
    pool.set(poolKey, shared)
  }
  shared.consumers += 1
  try {
    if (shared.resource) {
      for (const event of shared.replay.values()) handler(event)
    }
    shared.listeners.add(handler)
    await shared.opening
    let closing: Promise<void> | null = null
    return {
      close() {
        if (closing) return closing
        shared.listeners.delete(handler)
        closing = releaseSharedSubscription(pool, poolKey, shared, session)
        return closing
      },
    }
  } catch (error) {
    shared.listeners.delete(handler)
    if (shared.resource) await releaseSharedSubscription(pool, poolKey, shared, session)
    else forgetSharedConsumer(pool, poolKey, shared)
    throw error
  }
}

function subscriptionPool(owner: ProtoClientSession): Map<string, SharedEventSubscription> {
  let pool = sessionSubscriptionPools.get(owner)
  if (!pool) {
    pool = new Map()
    sessionSubscriptionPools.set(owner, pool)
  }
  return pool
}

function createSharedSubscription(
  poolOwner: ProtoClientSession,
  session: ProtoClientSession,
  command: EventSubscribeCommand,
): SharedEventSubscription {
  const early: EventEnvelope[] = []
  let earlyError: Error | null = null
  const shared: SharedEventSubscription = {
    consumers: 0,
    resource: undefined,
    opening: Promise.resolve(null as unknown as SubscriptionResource),
    local: { close() {} },
    listeners: new Set(),
    replay: new Map(),
  }
  shared.local = poolOwner.subscribeEvents((event) => {
    if (!shared.resource) {
      if (early.length >= MAX_EARLY_SUBSCRIPTION_EVENTS) {
        earlyError = new Error('event subscription correlation buffer overflow')
        return
      }
      early.push(event)
      return
    }
    if (sameResourceHandle(event.subscription, shared.resource)) publishSharedEvent(shared, event)
  })
  shared.opening = session.execute(create(CommandEnvelopeSchema, {
    command: { case: 'eventSubscribe', value: command },
  })).then((result) => {
    if (result.result.case !== 'eventSubscription' || !result.result.value.subscription) {
      throw new Error('event subscribe returned no subscription resource')
    }
    shared.resource = result.result.value.subscription
    if (earlyError) throw earlyError
    for (const event of early) {
      if (sameResourceHandle(event.subscription, shared.resource)) publishSharedEvent(shared, event)
    }
    early.length = 0
    return shared.resource
  }).catch((error: unknown) => {
    shared.local.close()
    throw error
  })
  return shared
}

function publishSharedEvent(shared: SharedEventSubscription, event: EventEnvelope): void {
  const replayKey = attachmentProjectionReplayKey(event)
  if (replayKey) shared.replay.set(replayKey, event)
  for (const listener of shared.listeners) listener(event)
}

function attachmentProjectionReplayKey(event: EventEnvelope): string | undefined {
  if (event.event.case !== 'terminalLifecycle' || !event.event.value.attachmentProjection) return undefined
  return event.event.value.terminal?.ref?.terminalId || undefined
}

function releaseSharedSubscription(
  pool: Map<string, SharedEventSubscription>,
  poolKey: string,
  shared: SharedEventSubscription,
  session: ProtoClientSession,
): Promise<void> {
  if (shared.consumers > 0) shared.consumers -= 1
  if (shared.consumers > 0) return Promise.resolve()
  if (pool.get(poolKey) === shared) pool.delete(poolKey)
  shared.local.close()
  shared.listeners.clear()
  shared.replay.clear()
  const resource = shared.resource
  if (!resource) return Promise.resolve()
  return session.execute(create(CommandEnvelopeSchema, {
    command: { case: 'releaseResource', value: create(ReleaseResourceCommandSchema, { resource }) },
  })).then(() => undefined).catch(() => undefined)
}

function forgetSharedConsumer(
  pool: Map<string, SharedEventSubscription>,
  poolKey: string,
  shared: SharedEventSubscription,
): void {
  if (shared.consumers > 0) shared.consumers -= 1
  if (shared.consumers !== 0) return
  if (pool.get(poolKey) === shared) pool.delete(poolKey)
  shared.listeners.clear()
  shared.replay.clear()
}

function eventSubscriptionKey(command: EventSubscribeCommand): string {
  return bytesKey(toBinary(EventSubscribeCommandSchema, command))
}

function bytesKey(bytes: Uint8Array): string {
  let key = ''
  for (const byte of bytes) key += byte.toString(16).padStart(2, '0')
  return key
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
