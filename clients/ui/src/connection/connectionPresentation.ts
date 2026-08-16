import type { ConnectionInfo, RtcConnectionPhase } from '../core/transport'
import type { MachineConnectionSnapshot } from './machineConnectionSnapshot'

export type ConnectionReachability = 'checking' | 'reachable' | 'unreachable' | 'unknown'

export type ConnectionPresentationState =
  | 'phone_offline'
  | 'auth_unavailable'
  | 'idle'
  | 'connecting'
  | 'ready'
  | 'waiting_network'
  | 'failed'

export type ConnectionPresentationTone = 'neutral' | 'info' | 'positive' | 'warning' | 'critical'
export type ConnectionPresentationRoute = 'none' | 'local' | 'direct' | 'ssh' | 'cloud'
export type ConnectionPresentationObservedPath = 'none' | 'p2p' | 'relay'
export type ConnectionPresentationPolicy = 'automatic' | 'relay_required'
export type ConnectionPresentationAction = 'none' | 'connect' | 'retry' | 'reauthorize'

export interface ConnectionPresentationInput {
  phoneOnline: boolean
  authAvailable: boolean
  reachability: ConnectionReachability
  snapshot: MachineConnectionSnapshot
}

/** Stable semantic keys for UI components. Components own translation, icons, and concrete colors. */
export interface ConnectionPresentation {
  state: ConnectionPresentationState
  tone: ConnectionPresentationTone
  reachability: ConnectionReachability
  policy: ConnectionPresentationPolicy
  route: ConnectionPresentationRoute
  observedPath: ConnectionPresentationObservedPath
  action: ConnectionPresentationAction
}

export function projectConnectionPresentation(input: ConnectionPresentationInput): ConnectionPresentation {
  const state = presentationState(input)
  const connectionInfo = state === 'ready' ? input.snapshot.connectionInfo : null

  return {
    state,
    tone: presentationTone(state, input.reachability),
    reachability: input.reachability,
    // forceRelay is desired policy only. It must never become an observed relay path.
    policy: input.snapshot.forceRelay ? 'relay_required' : 'automatic',
    route: presentationRoute(connectionInfo),
    observedPath: presentationObservedPath(connectionInfo),
    action: presentationAction(state),
  }
}

function presentationState(input: ConnectionPresentationInput): ConnectionPresentationState {
  if (!input.phoneOnline) return 'phone_offline'
  if (!input.authAvailable) return 'auth_unavailable'
  return lifecycleState(input.snapshot.phase)
}

function lifecycleState(phase: RtcConnectionPhase): Exclude<ConnectionPresentationState, 'phone_offline' | 'auth_unavailable'> {
  switch (phase) {
    case 'idle':
      return 'idle'
    case 'probing':
    case 'resolving':
    case 'signaling':
    case 'connecting':
    case 'authorizing':
    case 'verifying':
    case 'reconnecting':
      return 'connecting'
    case 'connected':
      return 'ready'
    case 'waiting_network':
      return 'waiting_network'
    case 'failed':
      return 'failed'
    default:
      return unreachablePhase(phase)
  }
}

function presentationTone(
  state: ConnectionPresentationState,
  reachability: ConnectionReachability,
): ConnectionPresentationTone {
  switch (state) {
    case 'idle':
      if (reachability === 'reachable') return 'positive'
      if (reachability === 'unreachable') return 'warning'
      if (reachability === 'checking') return 'info'
      return 'neutral'
    case 'connecting':
      return 'info'
    case 'ready':
      return 'positive'
    case 'phone_offline':
    case 'waiting_network':
      return 'warning'
    case 'auth_unavailable':
    case 'failed':
      return 'critical'
    default:
      return unreachableState(state)
  }
}

function presentationRoute(connectionInfo: ConnectionInfo | null): ConnectionPresentationRoute {
  if (!connectionInfo) return 'none'
  if (connectionInfo.routeKind === 'local') return 'local'
  if (connectionInfo.routeKind === 'direct') return 'direct'
  if (connectionInfo.routeKind === 'ssh') return 'ssh'
  if (connectionInfo.routeKind === 'cloud') return 'cloud'
  return connectionInfo.path === 'hub' ? 'cloud' : 'local'
}

function presentationObservedPath(connectionInfo: ConnectionInfo | null): ConnectionPresentationObservedPath {
  if (connectionInfo?.observedPath === 'direct') return 'p2p'
  if (connectionInfo?.observedPath === 'single_relay') return 'relay'
  return 'none'
}

function presentationAction(state: ConnectionPresentationState): ConnectionPresentationAction {
  if (state === 'auth_unavailable') return 'reauthorize'
  if (state === 'idle') return 'connect'
  if (state === 'failed') return 'retry'
  return 'none'
}

function unreachablePhase(phase: never): never {
  throw new Error(`Unsupported connection phase: ${String(phase)}`)
}

function unreachableState(state: never): never {
  throw new Error(`Unsupported connection presentation state: ${String(state)}`)
}
