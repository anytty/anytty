import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import { EventEnvelopeSchema } from '../generated/apipb/application_pb'
import { ResourceHandleSchema, ResourceKind } from '../generated/apipb/common_pb'
import { EventSubscribeCommandSchema, ApplicationEventType } from '../generated/apipb/events_pb'
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
    subscription.close()
    await Promise.resolve()
    expect(session.commands[1]?.command.case).toBe('releaseResource')
  })

  it('isolates concurrent subscriptions by their complete resource handle', async () => {
    let nextToken = 0
    const session = new MockProtoSession('machine-events', (command) => {
      if (command.command.case === 'eventSubscribe') {
        nextToken += 1
        return protoResult('eventSubscription', {
          subscription: create(ResourceHandleSchema, {
            kind: ResourceKind.SUBSCRIPTION,
            opaqueToken: new Uint8Array([nextToken]),
            session: session.stamp,
            generation: 1n,
          }),
        })
      }
      return protoResult('acknowledge', {})
    })
    const firstEvents: string[] = []
    const secondEvents: string[] = []
    const command = create(EventSubscribeCommandSchema, { types: [ApplicationEventType.TERMINAL_LIFECYCLE] })
    const first = await openProtoEventSubscription(session, command, (event) => firstEvents.push(event.eventId))
    const second = await openProtoEventSubscription(session, command, (event) => secondEvents.push(event.eventId))
    const resource = (token: number) => create(ResourceHandleSchema, {
      kind: ResourceKind.SUBSCRIPTION, opaqueToken: new Uint8Array([token]), session: session.stamp, generation: 1n,
    })
    session.emit(create(EventEnvelopeSchema, { eventId: 'first', subscription: resource(1) }))
    session.emit(create(EventEnvelopeSchema, { eventId: 'second', subscription: resource(2) }))
    expect(firstEvents).toEqual(['first'])
    expect(secondEvents).toEqual(['second'])
    first.close()
    session.emit(create(EventEnvelopeSchema, { eventId: 'first-after-close', subscription: resource(1) }))
    session.emit(create(EventEnvelopeSchema, { eventId: 'second-after-close', subscription: resource(2) }))
    expect(firstEvents).toEqual(['first'])
    expect(secondEvents).toEqual(['second', 'second-after-close'])
    second.close()
  })
})
