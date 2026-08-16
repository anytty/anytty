import { assertRemoteModelShape } from '../core/model'
import type { ConnectionPath } from '../core/transport'

export type ConnectionPhase =
  | 'idle'
  | 'probing'
  | 'connecting'
  | 'connected'
  | 'verifying'
  | 'reconnecting'
  | 'waiting_network'
  | 'failed'
  | 'released'

export type UserIntent =
  | { kind: 'connectMachine'; machineId: string }
  | { kind: 'terminal'; machineId: string; terminalId: string }
  | { kind: 'fileManager'; machineId: string }

export interface ChannelLifecycle {
  state: 'closed' | 'opening' | 'open' | 'verifying' | 'failed'
  error?: string | undefined
}

export interface VisibleError {
  message: string
  recoverable: boolean
  surface: 'toast' | 'banner' | 'modal'
}

export interface ConnectionSnapshot {
  phase: ConnectionPhase
  path: ConnectionPath | null
  connectionId?: string | undefined
  machineId?: string | undefined
  activeTerminalId?: string | undefined
  reconnectAttempt: number
  resumeVerificationRequired: boolean
  userIntent?: UserIntent | undefined
  terminalChannels: Record<string, ChannelLifecycle>
  fileManagers: Record<string, ChannelLifecycle>
  visibleError?: VisibleError | undefined
}

export type ConnectionMessage =
  | { type: 'user.connectMachine'; machineId: string }
  | { type: 'user.openTerminal'; machineId: string; terminalId: string }
  | { type: 'user.openFileManager'; machineId: string }
  | { type: 'user.retry' }
  | { type: 'user.release' }
  | { type: 'connection.connecting'; path: ConnectionPath }
  | { type: 'connection.connected'; path: ConnectionPath; connectionId: string }
  | { type: 'connection.disconnected'; reason?: string }
  | { type: 'connection.failed'; reason: string; recoverable: boolean; surface: 'toast' | 'banner' | 'modal' }
  | { type: 'connection.verified'; connectionId: string }
  | { type: 'network.offline' }
  | { type: 'network.online' }
  | { type: 'app.resume'; resumeKind: 'quick' | 'cold' | 'frozen' }
  | { type: 'terminal.channelOpen'; machineId: string; terminalId: string }
  | { type: 'terminal.channelClosed'; machineId: string; terminalId: string; reason?: string }
  | { type: 'file.channelOpen'; machineId: string }
  | { type: 'file.channelClosed'; machineId: string; reason?: string }

export function initialConnectionSnapshot(): ConnectionSnapshot {
  return {
    phase: 'idle',
    path: null,
    reconnectAttempt: 0,
    resumeVerificationRequired: false,
    terminalChannels: {},
    fileManagers: {},
  }
}

export function reduceConnectionMessage(
  snapshot: ConnectionSnapshot,
  message: ConnectionMessage,
): ConnectionSnapshot {
  assertMessageBoundary(message)

  switch (message.type) {
    case 'user.connectMachine':
      return cleanSnapshot({
        ...snapshot,
        phase: 'probing',
        machineId: message.machineId,
        userIntent: { kind: 'connectMachine', machineId: message.machineId },
        visibleError: undefined,
      })
    case 'connection.connecting':
      return {
        ...snapshot,
        phase: 'connecting',
        path: message.path,
      }
    case 'connection.connected':
      return cleanSnapshot({
        ...snapshot,
        phase: 'connected',
        path: message.path,
        connectionId: message.connectionId,
        reconnectAttempt: 0,
        resumeVerificationRequired: false,
        terminalChannels: openVerifyingChannels(snapshot.terminalChannels),
        fileManagers: openVerifyingChannels(snapshot.fileManagers),
        visibleError: undefined,
      })
    case 'user.openTerminal':
      return {
        ...snapshot,
        machineId: message.machineId,
        activeTerminalId: message.terminalId,
        userIntent: {
          kind: 'terminal',
          machineId: message.machineId,
          terminalId: message.terminalId,
        },
        terminalChannels: {
          ...snapshot.terminalChannels,
          [message.terminalId]: { state: 'opening' },
        },
      }
    case 'terminal.channelOpen':
      return {
        ...snapshot,
        machineId: message.machineId,
        activeTerminalId: message.terminalId,
        visibleError: undefined,
        terminalChannels: {
          ...snapshot.terminalChannels,
          [message.terminalId]: { state: 'open' },
        },
      }
    case 'terminal.channelClosed':
      return {
        ...snapshot,
        terminalChannels: setChannelState(
          snapshot.terminalChannels,
          message.terminalId,
          message.reason ? { state: 'failed', error: message.reason } : { state: 'closed' },
        ),
      }
    case 'user.openFileManager':
      return {
        ...snapshot,
        machineId: message.machineId,
        userIntent: {
          kind: 'fileManager',
          machineId: message.machineId,
        },
        fileManagers: {
          ...snapshot.fileManagers,
          [message.machineId]: { state: 'opening' },
        },
      }
    case 'file.channelOpen':
      return {
        ...snapshot,
        machineId: message.machineId,
        visibleError: undefined,
        fileManagers: {
          ...snapshot.fileManagers,
          [message.machineId]: { state: 'open' },
        },
      }
    case 'file.channelClosed':
      return {
        ...snapshot,
        fileManagers: setChannelState(
          snapshot.fileManagers,
          message.machineId,
          message.reason ? { state: 'failed', error: message.reason } : { state: 'closed' },
        ),
      }
    case 'app.resume':
      if (snapshot.phase !== 'connected' && snapshot.phase !== 'reconnecting' && snapshot.phase !== 'waiting_network') {
        return snapshot
      }
      return {
        ...snapshot,
        phase: 'verifying',
        resumeVerificationRequired: true,
        terminalChannels: markOpenChannelsVerifying(snapshot.terminalChannels),
        fileManagers: markOpenChannelsVerifying(snapshot.fileManagers),
      }
    case 'connection.verified':
      return cleanSnapshot({
        ...snapshot,
        phase: 'connected',
        connectionId: message.connectionId,
        resumeVerificationRequired: false,
        terminalChannels: openVerifyingChannels(snapshot.terminalChannels),
        fileManagers: openVerifyingChannels(snapshot.fileManagers),
        visibleError: undefined,
      })
    case 'network.offline':
      return {
        ...snapshot,
        phase: 'waiting_network',
      }
    case 'network.online':
      if (snapshot.phase !== 'waiting_network' && snapshot.phase !== 'failed') {
        return snapshot
      }
      return cleanSnapshot({
        ...snapshot,
        phase: 'reconnecting',
        reconnectAttempt: snapshot.reconnectAttempt + 1,
        visibleError: undefined,
      })
    case 'connection.disconnected':
      return {
        ...snapshot,
        phase: 'reconnecting',
        reconnectAttempt: snapshot.reconnectAttempt + 1,
      }
    case 'connection.failed':
      return {
        ...snapshot,
        phase: 'failed',
        visibleError: {
          message: message.reason,
          recoverable: message.recoverable,
          surface: message.surface,
        },
      }
    case 'user.retry':
      return cleanSnapshot({
        ...snapshot,
        phase: 'reconnecting',
        reconnectAttempt: 0,
        resumeVerificationRequired: false,
        visibleError: undefined,
      })
    case 'user.release':
      return cleanSnapshot({
        ...snapshot,
        phase: 'released',
        visibleError: undefined,
      })
  }
}

export function assertMessageBoundary(message: ConnectionMessage): void {
  assertRemoteModelShape(message)
  assertNoTransportImplementationLeak(message)
}

function assertNoTransportImplementationLeak(value: unknown): void {
  const seen = new Set<unknown>()
  const blockedKeys = new Set(['peerConnection', 'rtcDataChannel', 'nativePlugin'])

  function walk(current: unknown): void {
    if (current === null || current === undefined) return
    if (typeof current !== 'object') return
    if (seen.has(current)) return
    seen.add(current)

    if (Array.isArray(current)) {
      for (const item of current) walk(item)
      return
    }

    for (const [key, nested] of Object.entries(current as Record<string, unknown>)) {
      if (blockedKeys.has(key)) {
        throw new Error(`connection message must not contain ${key}`)
      }
      walk(nested)
    }
  }

  walk(value)
}

function setChannelState(
  channels: Record<string, ChannelLifecycle>,
  channelId: string,
  state: ChannelLifecycle,
): Record<string, ChannelLifecycle> {
  return {
    ...channels,
    [channelId]: state,
  }
}

function markOpenChannelsVerifying(
  channels: Record<string, ChannelLifecycle>,
): Record<string, ChannelLifecycle> {
  return mapChannelStates(channels, (channel) =>
    channel.state === 'open' ? { state: 'verifying' } : channel,
  )
}

function openVerifyingChannels(
  channels: Record<string, ChannelLifecycle>,
): Record<string, ChannelLifecycle> {
  return mapChannelStates(channels, (channel) =>
    channel.state === 'verifying' ? { state: 'open' } : channel,
  )
}

function mapChannelStates(
  channels: Record<string, ChannelLifecycle>,
  mapper: (channel: ChannelLifecycle) => ChannelLifecycle,
): Record<string, ChannelLifecycle> {
  const next: Record<string, ChannelLifecycle> = {}
  for (const [channelId, channel] of Object.entries(channels)) {
    next[channelId] = mapper(channel)
  }
  return next
}

function cleanSnapshot(snapshot: ConnectionSnapshot): ConnectionSnapshot {
  return clearOptional(snapshot)
}

function clearOptional<T extends object>(record: T): T {
  for (const key of Object.keys(record) as (keyof T)[]) {
    if (record[key] === undefined) {
      delete record[key]
    }
  }
  return record
}
