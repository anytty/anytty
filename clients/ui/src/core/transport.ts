import type { Machine, Terminal } from './model'
import type { ProtoClientSessionCloseError } from './protoClientSession'

export type ConnectionPath = 'local' | 'hub'

export const CONNECTION_PATHS = ['local', 'hub'] as const satisfies readonly ConnectionPath[]

export type ObservedPath = 'direct' | 'single_relay'

export const OBSERVED_PATHS = ['direct', 'single_relay'] as const satisfies readonly ObservedPath[]

export const ROUTE_SELECTION_REASONS = [
  'initial_best',
  'only_viable',
  'lower_loss',
  'direct_unstable',
  'lower_latency',
  'lower_score',
  'cost_guard',
  'minimum_hold',
  'cooldown',
  'hysteresis_hold',
  'insufficient_improvement',
  'current_unavailable',
  'current_best',
] as const

export type RouteSelectionReason = (typeof ROUTE_SELECTION_REASONS)[number]

export type RemoteRuntimeFetch = (input: string, init?: RequestInit) => Promise<Response>

export interface RemoteRuntimeStorage {
  getItem(key: string): string | null
  setItem(key: string, value: string): void
  removeItem(key: string): void
}

export interface RemoteNetworkRuntime {
  fetch: RemoteRuntimeFetch
  storage?: RemoteRuntimeStorage | undefined
  queryParam(name: string): string | null
}

export interface RtcConnectOptions {
  signal?: AbortSignal
  forceRelay?: boolean | undefined
  onStatus?: ((status: string) => void) | undefined
  onConnectionState?: ((snapshot: RtcConnectionStateSnapshot) => void) | undefined
}

export type ConnectionRoutePreference = 'auto' | 'direct' | 'ssh' | 'cloud'

/** CloudConnectionPreference 是 UI 提交给 Go Endpoint registry 的 managed Route 选路意图，不拥有当前连接真值。 */
export type CloudConnectionPreference = 'auto' | 'p2p' | 'relay'

/** RelayTransportPreference 只收缩 managed TURN 的 UDP/TCP 候选；实际 transport 仍来自 Go connection snapshot。 */
export type RelayTransportPreference = 'auto' | 'udp' | 'tcp'

/** ConnectionPolicy 是 Go-owned Endpoint 策略的 UI 投影，应用后由 Go 持久化并建立新的 session generation。 */
export interface ConnectionPolicy {
  route: ConnectionRoutePreference
  cloud: CloudConnectionPreference
  relayTransport: RelayTransportPreference
}

export interface ConnectionPolicyState {
  policy: ConnectionPolicy
  available: Record<Exclude<ConnectionRoutePreference, 'auto'>, boolean>
  unavailableReasons: Partial<Record<Exclude<ConnectionRoutePreference, 'auto'>, ConnectionPolicyUnavailableReason>>
}

export type ConnectionPolicyUnavailableReason = 'route_not_configured' | 'route_disabled' | 'platform_unsupported' | 'credential_unavailable' | 'cloud_unavailable'

export interface RtcSubscription {
  close(): void
}

export type RtcConnectionPhase = 'idle' | 'probing' | 'resolving' | 'signaling' | 'connecting' | 'authorizing' | 'connected' | 'verifying' | 'reconnecting' | 'waiting_network' | 'failed'

export interface RtcConnectionStateSnapshot {
  machineId: string
  phase: RtcConnectionPhase
  path?: ConnectionPath | undefined
  observedPath?: ObservedPath | undefined
  routeSelectionReason?: RouteSelectionReason | undefined
  statusText: string
  relayInUse: boolean
  error?: ProtoClientSessionCloseError | undefined
}

export interface RtcEvent {
  type: string
  payload?: unknown
}

export interface ConnectionInfo {
  path: ConnectionPath
  routeId?: string | undefined
  routeKind?: ConnectionRoutePreference | 'local' | undefined
  observedPath?: ObservedPath | undefined
  routeSelectionReason?: RouteSelectionReason | undefined
  connectionId: string
  machineId: string
  terminalId?: string
  relayInUse: boolean
  type?: 'p2p' | 'relay' | 'unknown' | undefined
  localAddr?: string | undefined
  remoteAddr?: string | undefined
  localBaseAddr?: string | undefined
  remoteBaseAddr?: string | undefined
  candidatePairId?: string | undefined
  candidateType?: string | undefined
  remoteCandidateType?: string | undefined
  rtt?: number | undefined
  localProtocol?: string | undefined
  remoteProtocol?: string | undefined
  relayTransport?: string | undefined
  networkClass?: string | undefined
  sampledAt?: number | undefined
  bytesSent?: bigint | undefined
  bytesReceived?: bigint | undefined
  packetsSent?: bigint | undefined
  lossEvents?: bigint | undefined
  generation?: bigint | undefined
}

export interface MachineConnectionStateEvents {
  subscribe(machineId: string, handler: (snapshot: RtcConnectionStateSnapshot) => void): RtcSubscription
}

export interface LocalStatus {
  machine: Machine
  localWeb: {
    httpUrl: string
    rtcOfferUrl: string
  }
}

export interface TerminalInventorySubscription {
  close(): void
}

export interface TerminalInventoryEvent {
  type: 'inventory_changed'
  payload?: unknown
}

export interface TerminalInventoryEvents {
  subscribe(machineId: string, handler: (event: TerminalInventoryEvent) => void): TerminalInventorySubscription
}

export interface LocalAgentApi {
  getStatus(): Promise<LocalStatus>
}

export interface LocalCreateTerminalInput {
  name?: string | undefined
  command?: string[] | undefined
  cols?: number | undefined
  rows?: number | undefined
  cwd?: string | undefined
  environment?: string[] | undefined
  scrollbackSize?: number | undefined
  scrollbackMaxBytes?: number | undefined
  scrollbackMaxAgeSeconds?: number | undefined
  sizeLockMode?: 'off' | 'warn' | 'lock' | undefined
}

export interface LocalUpdateTerminalInput {
  terminalId: string
  name?: string | undefined
  cwd?: string | undefined
  environment?: string | undefined
  sizeLockMode?: 'off' | 'warn' | 'lock' | undefined
}
