import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import { EventEnvelopeSchema } from '../generated/apipb/application_pb'
import { ResourceHandleSchema, ResourceKind } from '../generated/apipb/common_pb'
import { EventSubscribeCommandSchema, ApplicationEventType } from '../generated/apipb/events_pb'
import { StorageChangedEventSchema, StorageKeySchema, StorageScope } from '../generated/apipb/storage_pb'
import { TerminalInfoSchema, TerminalLifecycleEventSchema, TerminalRefSchema } from '../generated/apipb/terminal_pb'
import { MockProtoSession, protoResult } from '../test/mockProtoSession'
import { openProtoEventSubscription } from './protoEventSubscription'

describe('openProtoEventSubscription', () => {
  it('creates and releases the daemon subscription resource', async () => {
    const session = new MockProtoSession('machine-events', (command) => {
      if (command.command.case === 'eventSubscribe') {
        return protoResult('eventSubscription', {
          subscription: create(ResourceHandleSchema, {
            kind: ResourceKind.SUBSCRIPTION,
            opaqueToken: new Uint8Array([1, 2, 3]),
            session: session.stamp,
            generation: 1n,
          }),
        })
      }
      return protoResult('acknowledge', {})
    })
    const subscription = await openProtoEventSubscription(session, create(EventSubscribeCommandSchema, {
      types: [ApplicationEventType.TERMINAL_LIFECYCLE],
    }), () => undefined)

    expect(session.commands[0]?.command.case).toBe('eventSubscribe')
    await subscription.close()
    expect(session.commands[1]?.command.case).toBe('releaseResource')
  })

  it('shares one daemon subscription across leases for the same physical session and filter', async () => {
    const commandLog: string[] = []
    const handler = (command: Parameters<MockProtoSession['execute']>[0]) => {
      commandLog.push(command.command.case ?? 'missing')
      if (command.command.case === 'eventSubscribe') {
        return protoResult('eventSubscription', {
          subscription: create(ResourceHandleSchema, {
            kind: ResourceKind.SUBSCRIPTION,
            opaqueToken: new Uint8Array([7]),
            session: poolOwner.stamp,
            generation: 1n,
          }),
        })
      }
      return protoResult('acknowledge', {})
    }
    const poolOwner = new MockProtoSession('machine-events', handler)
    const firstSession = Object.assign(new MockProtoSession('machine-events', handler), { resourcePoolOwner: poolOwner })
    const secondSession = Object.assign(new MockProtoSession('machine-events', handler), { resourcePoolOwner: poolOwner })
    const firstEvents: string[] = []
    const secondEvents: string[] = []
    const command = create(EventSubscribeCommandSchema, { types: [ApplicationEventType.TERMINAL_LIFECYCLE] })
    const first = await openProtoEventSubscription(firstSession, command, (event) => firstEvents.push(event.eventId))

    const resource = create(ResourceHandleSchema, {
      kind: ResourceKind.SUBSCRIPTION, opaqueToken: new Uint8Array([7]), session: poolOwner.stamp, generation: 1n,
    })
    const unrelated = create(ResourceHandleSchema, {
      kind: ResourceKind.SUBSCRIPTION, opaqueToken: new Uint8Array([8]), session: poolOwner.stamp, generation: 1n,
    })
    poolOwner.emit(create(EventEnvelopeSchema, { eventId: 'unrelated', subscription: unrelated }))
    expect(firstEvents).toEqual([])
    poolOwner.emit(create(EventEnvelopeSchema, {
      eventId: 'projection',
      subscription: resource,
      event: {
        case: 'terminalLifecycle',
        value: create(TerminalLifecycleEventSchema, {
          attachmentProjection: true,
          terminal: create(TerminalInfoSchema, {
            ref: create(TerminalRefSchema, { terminalId: 'terminal-1' }),
          }),
        }),
      },
    }))
    expect(firstEvents).toEqual(['projection'])

    const second = await openProtoEventSubscription(secondSession, command, (event) => secondEvents.push(event.eventId))
    expect(commandLog).toEqual(['eventSubscribe'])
    expect(secondEvents).toEqual(['projection'])
    poolOwner.emit(create(EventEnvelopeSchema, { eventId: 'shared', subscription: resource }))
    expect(firstEvents).toEqual(['projection', 'shared'])
    expect(secondEvents).toEqual(['projection', 'shared'])

    await first.close()
    expect(commandLog).toEqual(['eventSubscribe'])
    await second.close()
    expect(commandLog).toEqual(['eventSubscribe', 'releaseResource'])
  })

  it('does not share subscriptions with different filters', async () => {
    let nextToken = 0
    const session = new MockProtoSession('machine-events', (command) => {
      if (command.command.case !== 'eventSubscribe') return protoResult('acknowledge', {})
      nextToken += 1
      return protoResult('eventSubscription', {
        subscription: create(ResourceHandleSchema, {
          kind: ResourceKind.SUBSCRIPTION,
          opaqueToken: new Uint8Array([nextToken]),
          session: session.stamp,
          generation: 1n,
        }),
      })
    })
    const first = await openProtoEventSubscription(session, create(EventSubscribeCommandSchema, {
      types: [ApplicationEventType.TERMINAL_LIFECYCLE],
    }), () => undefined)
    const second = await openProtoEventSubscription(session, create(EventSubscribeCommandSchema, {
      types: [ApplicationEventType.STORAGE_CHANGED],
    }), () => undefined)

    expect(session.commands.filter((command) => command.command.case === 'eventSubscribe')).toHaveLength(2)
    await first.close()
    await second.close()
  })

  it('does not accumulate daemon resources across repeated open and close cycles', async () => {
    let active = 0
    let peak = 0
    let nextToken = 0
    const session = new MockProtoSession('machine-events', (command) => {
      if (command.command.case === 'eventSubscribe') {
        active += 1
        peak = Math.max(peak, active)
        nextToken += 1
        return protoResult('eventSubscription', {
          subscription: create(ResourceHandleSchema, {
            kind: ResourceKind.SUBSCRIPTION,
            opaqueToken: new Uint8Array([nextToken]),
            session: session.stamp,
            generation: BigInt(nextToken),
          }),
        })
      }
      if (command.command.case === 'releaseResource') active -= 1
      return protoResult('acknowledge', {})
    })
    const command = create(EventSubscribeCommandSchema, {
      types: [ApplicationEventType.TERMINAL_LIFECYCLE],
    })

    for (let cycle = 0; cycle < 70; cycle += 1) {
      const subscription = await openProtoEventSubscription(session, command, () => undefined)
      await subscription.close()
    }

    expect(active).toBe(0)
    expect(peak).toBe(1)
    expect(session.commands.filter((entry) => entry.command.case === 'eventSubscribe')).toHaveLength(70)
    expect(session.commands.filter((entry) => entry.command.case === 'releaseResource')).toHaveLength(70)
  })

  it('does not let unrelated early event volume poison subscription correlation', async () => {
    let resolveSubscribe!: (value: ReturnType<typeof protoResult>) => void
    const subscribeResult = new Promise<ReturnType<typeof protoResult>>((resolve) => { resolveSubscribe = resolve })
    const session = new MockProtoSession('machine-events', (command) => (
      command.command.case === 'eventSubscribe' ? subscribeResult : protoResult('acknowledge', {})
    ))
    const targetResource = subscriptionResource(session, 21)
    const unrelatedResource = subscriptionResource(session, 22)
    const received: string[] = []
    const opening = openProtoEventSubscription(session, create(EventSubscribeCommandSchema, {
      terminal: create(TerminalRefSchema, { endpointId: 'machine-events', terminalId: 'target' }),
      types: [ApplicationEventType.TERMINAL_LIFECYCLE],
    }), (event) => received.push(event.eventId))

    for (let index = 0; index < 80; index += 1) {
      session.emit(terminalLifecycleEvent(unrelatedResource, 'target', `unrelated-${index}`, 1n))
    }
    session.emit(terminalLifecycleEvent(targetResource, 'target', 'target-projection', 1n))
    resolveSubscribe(protoResult('eventSubscription', { subscription: targetResource }))

    const subscription = await opening
    expect(received).toEqual(['target-projection'])
    await subscription.close()
  })

  it('coalesces a large target projection snapshot and replays it to shared consumers', async () => {
    let resolveSubscribe!: (value: ReturnType<typeof protoResult>) => void
    const subscribeResult = new Promise<ReturnType<typeof protoResult>>((resolve) => { resolveSubscribe = resolve })
    const session = new MockProtoSession('machine-events', (command) => (
      command.command.case === 'eventSubscribe' ? subscribeResult : protoResult('acknowledge', {})
    ))
    const resource = subscriptionResource(session, 31)
    const command = create(EventSubscribeCommandSchema, {
      types: [ApplicationEventType.TERMINAL_LIFECYCLE],
    })
    const firstEvents: string[] = []
    const firstOpening = openProtoEventSubscription(session, command, (event) => firstEvents.push(event.eventId))

    for (let index = 0; index < 80; index += 1) {
      session.emit(terminalLifecycleEvent(resource, `terminal-${index}`, `initial-${index}`, 1n))
    }
    for (let index = 0; index < 100; index += 1) {
      session.emit(terminalLifecycleEvent(resource, 'target', `target-${index}`, BigInt(index + 1)))
    }
    session.emit(terminalLifecycleEvent(resource, 'target', 'target-stale', 1n))
    session.emit(terminalLifecycleEvent(resource, 'target', 'target-lifecycle', 0n, false))
    resolveSubscribe(protoResult('eventSubscription', { subscription: resource }))

    const first = await firstOpening
    expect(firstEvents).toHaveLength(82)
    expect(firstEvents.slice(-2)).toEqual(['target-99', 'target-lifecycle'])

    const secondEvents: string[] = []
    const second = await openProtoEventSubscription(session, command, (event) => secondEvents.push(event.eventId))
    expect(secondEvents).toHaveLength(81)
    expect(secondEvents.at(-1)).toBe('target-99')
    expect(session.commands.filter((entry) => entry.command.case === 'eventSubscribe')).toHaveLength(1)

    await first.close()
    expect(session.commands.filter((entry) => entry.command.case === 'releaseResource')).toHaveLength(0)
    await second.close()
    expect(session.commands.filter((entry) => entry.command.case === 'releaseResource')).toHaveLength(1)
  })

  it('keeps early storage state by complete key and highest version', async () => {
    let resolveSubscribe!: (value: ReturnType<typeof protoResult>) => void
    const subscribeResult = new Promise<ReturnType<typeof protoResult>>((resolve) => { resolveSubscribe = resolve })
    const session = new MockProtoSession('machine-events', (command) => (
      command.command.case === 'eventSubscribe' ? subscribeResult : protoResult('acknowledge', {})
    ))
    const resource = subscriptionResource(session, 41)
    const received: string[] = []
    const opening = openProtoEventSubscription(session, create(EventSubscribeCommandSchema, {
      types: [ApplicationEventType.STORAGE_CHANGED],
      storageAppId: 'settings',
      storageScope: StorageScope.PRIVATE,
      storageOwnerId: 'owner-a',
      storageKeyPrefix: 'terminal/',
    }), (event) => received.push(event.eventId))

    session.emit(storageChangedEvent(resource, 'theme-v2', 'terminal/theme', 2n))
    session.emit(storageChangedEvent(resource, 'theme-stale', 'terminal/theme', 1n))
    session.emit(storageChangedEvent(resource, 'font-v1', 'terminal/font', 1n))
    session.emit(storageChangedEvent(resource, 'theme-v3', 'terminal/theme', 3n))
    session.emit(storageChangedEvent(resource, 'outside-prefix', 'workspace/layout', 4n))
    resolveSubscribe(protoResult('eventSubscription', { subscription: resource }))

    const subscription = await opening
    expect(received).toEqual(['font-v1', 'theme-v3'])
    await subscription.close()
  })

  it('ignores invalid resource handles while correlating an opening subscription', async () => {
    let resolveSubscribe!: (value: ReturnType<typeof protoResult>) => void
    const subscribeResult = new Promise<ReturnType<typeof protoResult>>((resolve) => { resolveSubscribe = resolve })
    const session = new MockProtoSession('machine-events', (command) => (
      command.command.case === 'eventSubscribe' ? subscribeResult : protoResult('acknowledge', {})
    ))
    const otherSession = new MockProtoSession('other-machine')
    const targetResource = subscriptionResource(session, 51)
    const received: string[] = []
    const opening = openProtoEventSubscription(session, create(EventSubscribeCommandSchema, {
      terminal: create(TerminalRefSchema, { endpointId: 'machine-events', terminalId: 'target' }),
      types: [ApplicationEventType.TERMINAL_LIFECYCLE],
    }), (event) => received.push(event.eventId))

    for (let index = 0; index < 80; index += 1) {
      session.emit(terminalLifecycleEvent(subscriptionResource(otherSession, index), 'target', `wrong-stamp-${index}`, 1n))
    }
    session.emit(create(EventEnvelopeSchema, {
      eventId: 'missing-resource',
      event: {
        case: 'terminalLifecycle',
        value: create(TerminalLifecycleEventSchema, {
          attachmentProjection: true,
          terminal: create(TerminalInfoSchema, {
            ref: create(TerminalRefSchema, { terminalId: 'target' }),
          }),
        }),
      },
    }))
    session.emit(terminalLifecycleEvent(targetResource, 'target', 'target', 1n))
    resolveSubscribe(protoResult('eventSubscription', { subscription: targetResource }))

    const subscription = await opening
    expect(received).toEqual(['target'])
    await subscription.close()
  })

  it('accepts the target as the sixty-fourth early subscription resource', async () => {
    let resolveSubscribe!: (value: ReturnType<typeof protoResult>) => void
    const subscribeResult = new Promise<ReturnType<typeof protoResult>>((resolve) => { resolveSubscribe = resolve })
    const session = new MockProtoSession('machine-events', (command) => (
      command.command.case === 'eventSubscribe' ? subscribeResult : protoResult('acknowledge', {})
    ))
    const targetResource = subscriptionResource(session, 64)
    const received: string[] = []
    const command = create(EventSubscribeCommandSchema, { types: [ApplicationEventType.TERMINAL_LIFECYCLE] })
    const opening = openProtoEventSubscription(session, command, (event) => received.push(event.eventId))

    for (let token = 1; token <= 63; token += 1) {
      session.emit(terminalLifecycleEvent(subscriptionResource(session, token), `terminal-${token}`, `other-${token}`, 1n))
    }
    session.emit(terminalLifecycleEvent(targetResource, 'target', 'target', 1n))
    resolveSubscribe(protoResult('eventSubscription', { subscription: targetResource }))

    const subscription = await opening
    expect(received).toEqual(['target'])
    await subscription.close()
  })

  it('bounds unprojected target events and releases one failed shared resource', async () => {
    let resolveSubscribe!: (value: ReturnType<typeof protoResult>) => void
    const subscribeResult = new Promise<ReturnType<typeof protoResult>>((resolve) => { resolveSubscribe = resolve })
    const session = new MockProtoSession('machine-events', (command) => (
      command.command.case === 'eventSubscribe' ? subscribeResult : protoResult('acknowledge', {})
    ))
    const resource = subscriptionResource(session, 71)
    const command = create(EventSubscribeCommandSchema, { types: [ApplicationEventType.TERMINAL_LIFECYCLE] })
    const first = openProtoEventSubscription(session, command, () => undefined)
    const second = openProtoEventSubscription(session, command, () => undefined)

    for (let index = 0; index < 65; index += 1) {
      session.emit(create(EventEnvelopeSchema, {
        eventId: `unprojected-${index}`,
        subscription: resource,
        event: { case: 'terminalLifecycle', value: create(TerminalLifecycleEventSchema) },
      }))
    }
    resolveSubscribe(protoResult('eventSubscription', { subscription: resource }))

    await expect(first).rejects.toThrow('event subscription correlation buffer overflow')
    await expect(second).rejects.toThrow('event subscription correlation buffer overflow')
    expect(session.commands.filter((entry) => entry.command.case === 'eventSubscribe')).toHaveLength(1)
    expect(session.commands.filter((entry) => entry.command.case === 'releaseResource')).toHaveLength(1)
  })
})

function subscriptionResource(session: MockProtoSession, token: number) {
  return create(ResourceHandleSchema, {
    kind: ResourceKind.SUBSCRIPTION,
    opaqueToken: new Uint8Array([token]),
    session: session.stamp,
    generation: 1n,
  })
}

function terminalLifecycleEvent(
  subscription: ReturnType<typeof subscriptionResource>,
  terminalId: string,
  eventId: string,
  resizeEpoch: bigint,
  attachmentProjection = true,
) {
  return create(EventEnvelopeSchema, {
    eventId,
    subscription,
    event: {
      case: 'terminalLifecycle',
      value: create(TerminalLifecycleEventSchema, {
        attachmentProjection,
        resizeEpoch,
        terminal: create(TerminalInfoSchema, {
          ref: create(TerminalRefSchema, { terminalId }),
        }),
      }),
    },
  })
}

function storageChangedEvent(
  subscription: ReturnType<typeof subscriptionResource>,
  eventId: string,
  key: string,
  version: bigint,
) {
  return create(EventEnvelopeSchema, {
    eventId,
    subscription,
    event: {
      case: 'storageChanged',
      value: create(StorageChangedEventSchema, {
        key: create(StorageKeySchema, {
          appId: 'settings',
          scope: StorageScope.PRIVATE,
          ownerId: 'owner-a',
          key,
        }),
        version,
        operation: 'put',
      }),
    },
  })
}
