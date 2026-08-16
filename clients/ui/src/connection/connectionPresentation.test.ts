import { describe, expect, it } from 'vitest'
import type { ConnectionInfo, RtcConnectionPhase } from '../core/transport'
import type { MachineConnectionSnapshot } from './machineConnectionSnapshot'
import {
  projectConnectionPresentation,
  type ConnectionPresentationInput,
  type ConnectionReachability,
} from './connectionPresentation'

describe('projectConnectionPresentation', () => {
  it('keeps an idle but reachable endpoint distinct from a ready session', () => {
    expect(project({ phase: 'idle', reachability: 'reachable', statusText: 'Connected via relay' })).toEqual({
      state: 'idle',
      tone: 'positive',
      reachability: 'reachable',
      policy: 'automatic',
      route: 'none',
      observedPath: 'none',
      action: 'connect',
    })
  })

  it.each<RtcConnectionPhase>([
    'probing',
    'resolving',
    'signaling',
    'connecting',
    'authorizing',
    'verifying',
    'reconnecting',
  ])('projects %s as connecting without exposing stale connection details', (phase) => {
    const result = project({
      phase,
      statusText: 'Failed',
      connectionInfo: connectionInfo({ routeKind: 'direct', observedPath: 'direct' }),
    })

    expect(result).toMatchObject({
      state: 'connecting',
      tone: 'info',
      route: 'none',
      observedPath: 'none',
      action: 'none',
    })
  })

  it.each([
    ['direct', 'direct'],
    ['ssh', 'ssh'],
  ] as const)('keeps a ready %s route separate from its observed transport path', (routeKind, route) => {
    expect(project({
      phase: 'connected',
      connectionInfo: connectionInfo({ routeKind }),
    })).toMatchObject({
      state: 'ready',
      tone: 'positive',
      route,
      observedPath: 'none',
      action: 'none',
    })
  })

  it('reports a Cloud P2P observation even when the requested policy requires relay', () => {
    const result = project({
      phase: 'connected',
      forceRelay: true,
      relayInUse: true,
      connectionInfo: connectionInfo({
        path: 'hub',
        routeKind: 'cloud',
        observedPath: 'direct',
        relayInUse: true,
        type: 'relay',
      }),
    })

    expect(result).toMatchObject({
      state: 'ready',
      policy: 'relay_required',
      route: 'cloud',
      observedPath: 'p2p',
    })
  })

  it('reports relay only from the explicit observed path', () => {
    expect(project({
      phase: 'connected',
      connectionInfo: connectionInfo({
        path: 'hub',
        routeKind: 'cloud',
        observedPath: 'single_relay',
        relayInUse: true,
      }),
    })).toMatchObject({
      state: 'ready',
      route: 'cloud',
      observedPath: 'relay',
    })
  })

  it('projects a waiting session independently from device reachability', () => {
    expect(project({ phase: 'waiting_network', reachability: 'reachable' })).toMatchObject({
      state: 'waiting_network',
      tone: 'warning',
      reachability: 'reachable',
      route: 'none',
      action: 'none',
    })
  })

  it('projects a typed failed phase as retryable without reading statusText', () => {
    expect(project({ phase: 'failed', statusText: 'Connected' })).toMatchObject({
      state: 'failed',
      tone: 'critical',
      action: 'retry',
    })
  })

  it('gives device offline state precedence over a stale ready snapshot', () => {
    expect(project({
      phoneOnline: false,
      phase: 'connected',
      reachability: 'reachable',
      connectionInfo: connectionInfo({
        path: 'hub',
        routeKind: 'cloud',
        observedPath: 'single_relay',
      }),
    })).toEqual({
      state: 'phone_offline',
      tone: 'warning',
      reachability: 'reachable',
      policy: 'automatic',
      route: 'none',
      observedPath: 'none',
      action: 'none',
    })
  })

  it('projects unavailable authorization as a reauthorization action', () => {
    expect(project({ authAvailable: false, phase: 'idle' })).toMatchObject({
      state: 'auth_unavailable',
      tone: 'critical',
      route: 'none',
      observedPath: 'none',
      action: 'reauthorize',
    })
  })
})

function project(input: {
  phoneOnline?: boolean
  authAvailable?: boolean
  reachability?: ConnectionReachability
  phase: RtcConnectionPhase
  statusText?: string
  connectionInfo?: ConnectionInfo | null
  forceRelay?: boolean
  relayInUse?: boolean
}) {
  const projectionInput: ConnectionPresentationInput = {
    phoneOnline: input.phoneOnline ?? true,
    authAvailable: input.authAvailable ?? true,
    reachability: input.reachability ?? 'unknown',
    snapshot: snapshot(input),
  }
  return projectConnectionPresentation(projectionInput)
}

function snapshot(input: {
  phase: RtcConnectionPhase
  statusText?: string
  connectionInfo?: ConnectionInfo | null
  forceRelay?: boolean
  relayInUse?: boolean
}): MachineConnectionSnapshot {
  return {
    machineId: 'machine-1',
    phase: input.phase,
    statusText: input.statusText ?? input.phase,
    connectionInfo: input.connectionInfo ?? null,
    forceRelay: input.forceRelay ?? false,
    relayInUse: input.relayInUse ?? false,
    reconnectAttempt: 0,
    error: null,
  }
}

function connectionInfo(overrides: Partial<ConnectionInfo> = {}): ConnectionInfo {
  return {
    path: 'local',
    connectionId: 'connection-1',
    machineId: 'machine-1',
    relayInUse: false,
    ...overrides,
  }
}
