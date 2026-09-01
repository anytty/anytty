import { create, toBinary } from '@bufbuild/protobuf'
import { CommandEnvelopeSchema, ReleaseResourceCommandSchema } from '../generated/apipb/application_pb'
import { ResourceKind } from '../generated/apipb/common_pb'
import { ApplicationEventType, EventSubscribeCommandSchema, type EventSubscribeCommand } from '../generated/apipb/events_pb'
import { StorageScope } from '../generated/apipb/storage_pb'
import type { ProtoClientSession, ProtoClientSubscription } from './protoClientSession'
import type { EventEnvelope } from '../generated/apipb/application_pb'

const MAX_EARLY_SUBSCRIPTION_RESOURCES = 64
const MAX_EARLY_UNPROJECTED_EVENTS_PER_RESOURCE = 64

type SubscriptionResource = NonNullable<EventEnvelope['subscription']>

type SharedEventSubscription = {
  consumers: number
  resource: SubscriptionResource | undefined
  opening: Promise<SubscriptionResource>
  local: ProtoClientSubscription
  listeners: Set<(event: EventEnvelope) => void>
  replay: Map<string, EventEnvelope>
}

type EarlySubscriptionEvents = {
  stateEvents: Map<string, SequencedEvent>
  unprojectedEvents: SequencedEvent[]
  nextSequence: number
  overflowed: boolean
}

type SequencedEvent = { event: EventEnvelope; sequence: number }

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
  const earlyByResource = new Map<string, EarlySubscriptionEvents>()
  const droppedResourceKeys = new Set<string>()
  let droppedResourceKeysOverflowed = false
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
      if (!eventMatchesSubscriptionCommand(event, command)) return
      const resourceKey = subscriptionResourceKey(event.subscription, poolOwner.stamp)
      if (!resourceKey) return
      let early = earlyByResource.get(resourceKey)
      if (!early) {
        if (earlyByResource.size >= MAX_EARLY_SUBSCRIPTION_RESOURCES) {
          if (droppedResourceKeys.size < MAX_EARLY_SUBSCRIPTION_RESOURCES) droppedResourceKeys.add(resourceKey)
          else droppedResourceKeysOverflowed = true
          return
        }
        early = {
          stateEvents: new Map(),
          unprojectedEvents: [],
          nextSequence: 0,
          overflowed: false,
        }
        earlyByResource.set(resourceKey, early)
      }
      bufferEarlySubscriptionEvent(early, event)
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
    const resourceKey = subscriptionResourceKey(shared.resource, poolOwner.stamp)
    if (!resourceKey) throw new Error('event subscribe returned an invalid subscription resource')
    const early = earlyByResource.get(resourceKey)
    if (droppedResourceKeys.has(resourceKey) || (!early && droppedResourceKeysOverflowed) || early?.overflowed) {
      throw new Error('event subscription correlation buffer overflow')
    }
    for (const event of orderedEarlySubscriptionEvents(early)) publishSharedEvent(shared, event)
    earlyByResource.clear()
    return shared.resource
  }).catch((error: unknown) => {
    shared.local.close()
    throw error
  })
  return shared
}

function bufferEarlySubscriptionEvent(early: EarlySubscriptionEvents, event: EventEnvelope): void {
  const sequence = early.nextSequence
  early.nextSequence += 1
  const stateKey = earlyEventStateKey(event)
  if (stateKey) {
    const existing = early.stateEvents.get(stateKey)
    if (!existing || eventStateShouldReplace(existing.event, event)) {
      early.stateEvents.set(stateKey, { event, sequence })
    }
    return
  }
  if (early.unprojectedEvents.length >= MAX_EARLY_UNPROJECTED_EVENTS_PER_RESOURCE) {
    early.overflowed = true
    return
  }
  early.unprojectedEvents.push({ event, sequence })
}

function orderedEarlySubscriptionEvents(early: EarlySubscriptionEvents | undefined): EventEnvelope[] {
  if (!early) return []
  return [...early.stateEvents.values(), ...early.unprojectedEvents]
    .sort((left, right) => left.sequence - right.sequence)
    .map(({ event }) => event)
}

function publishSharedEvent(shared: SharedEventSubscription, event: EventEnvelope): void {
  const replayKey = attachmentProjectionReplayKey(event)
  const replayed = replayKey ? shared.replay.get(replayKey) : undefined
  if (replayKey && (!replayed || eventStateShouldReplace(replayed, event))) shared.replay.set(replayKey, event)
  for (const listener of shared.listeners) listener(event)
}

function attachmentProjectionReplayKey(event: EventEnvelope): string | undefined {
  if (event.event.case !== 'terminalLifecycle' || !event.event.value.attachmentProjection) return undefined
  return event.event.value.terminal?.ref?.terminalId || undefined
}

function earlyEventStateKey(event: EventEnvelope): string | undefined {
  if (event.event.case === 'terminalLifecycle') {
    const terminalId = event.event.value.terminal?.ref?.terminalId
    if (!terminalId) return undefined
    return `${event.event.value.attachmentProjection ? 'terminal.attachment' : 'terminal.lifecycle'}\0${terminalId}`
  }
  if (event.event.case === 'storageChanged') {
    const key = event.event.value.key
    if (!key) return undefined
    return `storage\0${JSON.stringify([key.appId, key.scope, key.ownerId, key.key])}`
  }
  return undefined
}

function eventStateShouldReplace(existing: EventEnvelope, incoming: EventEnvelope): boolean {
  if (existing.event.case === 'terminalLifecycle' && incoming.event.case === 'terminalLifecycle') {
    if (existing.event.value.attachmentProjection && incoming.event.value.attachmentProjection) {
      return incoming.event.value.resizeEpoch >= existing.event.value.resizeEpoch
    }
    return true
  }
  if (existing.event.case === 'storageChanged' && incoming.event.case === 'storageChanged') {
    return incoming.event.value.version >= existing.event.value.version
  }
  return true
}

function eventMatchesSubscriptionCommand(event: EventEnvelope, command: EventSubscribeCommand): boolean {
  if (event.event.case === 'terminalLifecycle') {
    if (!commandAcceptsType(command, ApplicationEventType.TERMINAL_LIFECYCLE)) return false
    const targetTerminalId = command.terminal?.terminalId
    return !targetTerminalId || event.event.value.terminal?.ref?.terminalId === targetTerminalId
  }
  if (event.event.case === 'storageChanged') {
    if (!commandAcceptsType(command, ApplicationEventType.STORAGE_CHANGED)) return false
    const key = event.event.value.key
    if (!key) return true
    return (!command.storageAppId || key.appId === command.storageAppId) &&
      (command.storageScope === StorageScope.UNSPECIFIED || key.scope === command.storageScope) &&
      (!command.storageOwnerId || key.ownerId === command.storageOwnerId) &&
      (!command.storageKeyPrefix || key.key.startsWith(command.storageKeyPrefix))
  }
  return false
}

function commandAcceptsType(command: EventSubscribeCommand, eventType: ApplicationEventType): boolean {
  return command.types.length === 0 || command.types.includes(eventType)
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

function subscriptionResourceKey(
  resource: EventEnvelope['subscription'],
  owner: ProtoClientSession['stamp'],
): string | undefined {
  const session = resource?.session
  if (!resource || resource.kind !== ResourceKind.SUBSCRIPTION || !session || !sameSessionStamp(session, owner)) return undefined
  return JSON.stringify([
    resource.kind,
    resource.generation.toString(),
    bytesKey(resource.opaqueToken),
    session.endpointId,
    session.routeId,
    session.generation.toString(),
  ])
}

function sameSessionStamp(left: ProtoClientSession['stamp'], right: ProtoClientSession['stamp']): boolean {
  return left.endpointId === right.endpointId && left.routeId === right.routeId && left.generation === right.generation
}

function sameBytes(left: Uint8Array, right: Uint8Array): boolean {
  if (left.byteLength !== right.byteLength) return false
  for (let index = 0; index < left.byteLength; index += 1) {
    if (left[index] !== right[index]) return false
  }
  return true
}
