import { describe, expect, it } from 'vitest'
import {
  initialConnectionSnapshot,
  reduceConnectionMessage,
  type ConnectionSnapshot,
} from './connectionMessageReducer'

function reduceAll(events: Parameters<typeof reduceConnectionMessage>[1][]): ConnectionSnapshot {
  return events.reduce(
    (snapshot, event) => reduceConnectionMessage(snapshot, event),
    initialConnectionSnapshot(),
  )
}

describe('connectionMessageReducer', () => {
  it('turns user intent and connection events into a stable machine/terminal snapshot', () => {
    const snapshot = reduceAll([
      { type: 'user.connectMachine', machineId: 'machine-local' },
      { type: 'connection.connecting', path: 'local' },
      { type: 'connection.connected', path: 'local', connectionId: 'conn-1' },
      { type: 'user.openTerminal', machineId: 'machine-local', terminalId: 'term-1' },
      { type: 'terminal.channelOpen', machineId: 'machine-local', terminalId: 'term-1' },
      { type: 'user.openFileManager', machineId: 'machine-local' },
      { type: 'file.channelOpen', machineId: 'machine-local' },
    ])

    expect(snapshot.phase).toBe('connected')
    expect(snapshot.path).toBe('local')
    expect(snapshot.machineId).toBe('machine-local')
    expect(snapshot.activeTerminalId).toBe('term-1')
    expect(snapshot.terminalChannels['term-1']?.state).toBe('open')
    expect(snapshot.fileManagers['machine-local']?.state).toBe('open')
    expect(JSON.stringify(snapshot)).not.toMatch(/workspace|tab|window|pane/i)
  })

  it('models native-like app resume as verification before returning to connected', () => {
    const connected = reduceAll([
      { type: 'user.connectMachine', machineId: 'machine-local' },
      { type: 'connection.connected', path: 'local', connectionId: 'conn-1' },
      { type: 'user.openTerminal', machineId: 'machine-local', terminalId: 'term-1' },
      { type: 'terminal.channelOpen', machineId: 'machine-local', terminalId: 'term-1' },
    ])

    const verifying = reduceConnectionMessage(connected, {
      type: 'app.resume',
      resumeKind: 'cold',
    })
    expect(verifying.phase).toBe('verifying')
    expect(verifying.resumeVerificationRequired).toBe(true)
    expect(verifying.userIntent).toEqual({
      kind: 'terminal',
      machineId: 'machine-local',
      terminalId: 'term-1',
    })
    expect(verifying.terminalChannels['term-1']?.state).toBe('verifying')

    const recovered = reduceConnectionMessage(verifying, {
      type: 'connection.verified',
      connectionId: 'conn-1',
    })
    expect(recovered.phase).toBe('connected')
    expect(recovered.resumeVerificationRequired).toBe(false)
    expect(recovered.terminalChannels['term-1']?.state).toBe('open')
  })

  it('preserves user intent through offline/online reconnect and routes errors to visible surfaces', () => {
    const connected = reduceAll([
      { type: 'user.connectMachine', machineId: 'machine-local' },
      { type: 'connection.connected', path: 'local', connectionId: 'conn-1' },
      { type: 'user.openFileManager', machineId: 'machine-local' },
    ])

    const offline = reduceConnectionMessage(connected, { type: 'network.offline' })
    expect(offline.phase).toBe('waiting_network')
    expect(offline.userIntent).toEqual({
      kind: 'fileManager',
      machineId: 'machine-local',
    })

    const reconnecting = reduceConnectionMessage(offline, { type: 'network.online' })
    expect(reconnecting.phase).toBe('reconnecting')
    expect(reconnecting.reconnectAttempt).toBe(1)

    const failed = reduceConnectionMessage(reconnecting, {
      type: 'connection.failed',
      reason: 'ice-timeout',
      recoverable: true,
      surface: 'banner',
    })
    expect(failed.phase).toBe('failed')
    expect(failed.visibleError).toEqual({
      message: 'ice-timeout',
      recoverable: true,
      surface: 'banner',
    })

    const verified = reduceConnectionMessage(failed, {
      type: 'connection.verified',
      connectionId: 'conn-2',
    })
    expect(verified.phase).toBe('connected')
    expect(verified.connectionId).toBe('conn-2')
    expect(verified.visibleError).toBeUndefined()
  })
})
