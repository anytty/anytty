import { describe, expect, it } from 'vitest'
import { ConnectionEventQueue } from './eventQueue'
import { initialConnectionSnapshot, reduceConnectionMessage } from './connectionMessageReducer'

describe('ConnectionEventQueue', () => {
  it('flushes connection/lifecycle events in sequence order and drops duplicates', () => {
    const queue = new ConnectionEventQueue({ maxSize: 8 })

    expect(queue.enqueue({ id: 'e2', sequence: 2, message: { type: 'connection.connecting', path: 'local' } })).toBe(true)
    expect(queue.enqueue({ id: 'e1', sequence: 1, message: { type: 'user.connectMachine', machineId: 'machine-local' } })).toBe(true)
    expect(queue.enqueue({ id: 'e2', sequence: 2, message: { type: 'connection.connected', path: 'local', connectionId: 'conn-ignored' } })).toBe(false)
    expect(queue.droppedDuplicateCount).toBe(1)

    const flushed = queue.flush()
    expect(flushed.map((event) => event.id)).toEqual(['e1', 'e2'])

    const snapshot = flushed.reduce(
      (state, event) => reduceConnectionMessage(state, event.message),
      initialConnectionSnapshot(),
    )
    expect(snapshot.phase).toBe('connecting')
    expect(snapshot.machineId).toBe('machine-local')
  })

  it('applies backpressure by keeping lifecycle/user-intent events and dropping stale connection chatter first', () => {
    const queue = new ConnectionEventQueue({ maxSize: 3 })

    queue.enqueue({ id: 'intent', sequence: 1, message: { type: 'user.connectMachine', machineId: 'machine-local' } })
    queue.enqueue({ id: 'connecting', sequence: 2, message: { type: 'connection.connecting', path: 'local' } })
    queue.enqueue({ id: 'noise-1', sequence: 3, message: { type: 'connection.disconnected', reason: 'brief' } })
    queue.enqueue({ id: 'resume', sequence: 4, message: { type: 'app.resume', resumeKind: 'quick' } })

    expect(queue.droppedBackpressureCount).toBe(1)
    expect(queue.flush().map((event) => event.id)).toEqual(['intent', 'connecting', 'resume'])
  })

  it('keeps messages connection-interface shaped without browser or native implementation leakage', () => {
    const queue = new ConnectionEventQueue({ maxSize: 4 })

    expect(() =>
      queue.enqueue({
        id: 'bad',
        sequence: 1,
        message: {
          type: 'connection.connected',
          path: 'local',
          connectionId: 'conn-1',
          peerConnection: {},
        },
      } as never),
    ).toThrow(/peerConnection/)

    expect(() =>
      queue.enqueue({
        id: 'bad-nested',
        sequence: 2,
        message: {
          type: 'connection.connected',
          path: 'local',
          connectionId: 'conn-1',
          detail: { nativePlugin: {} },
        },
      } as never),
    ).toThrow(/nativePlugin/)
  })
})
