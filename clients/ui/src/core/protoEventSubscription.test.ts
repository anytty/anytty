import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import { EventEnvelopeSchema } from '../generated/apipb/application_pb'
import { ResourceHandleSchema, ResourceKind } from '../generated/apipb/common_pb'
import { EventSubscribeCommandSchema, ApplicationEventType } from '../generated/apipb/events_pb'
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
})
