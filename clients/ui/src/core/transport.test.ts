import { describe, expect, expectTypeOf, it } from 'vitest'
import { CONNECTION_PATHS, ROUTE_SELECTION_REASONS } from './transport'
import type {
  ConnectionCapabilities,
  ConnectionInfo,
  ConnectionPath,
  ConnectionPolicy,
  LocalAgentApi,
  ManagedRtcConnector,
  ManagedRtcSession,
  RemoteNetworkRuntime,
  RemoteRuntimeStorage,
  RtcBinaryChannel,
  RtcConnector,
  RtcJsonRpcChannel,
  RtcSession,
  RelayTransportPreference,
  TerminalInventoryEvents,
} from './transport'
import transportSource from './transport.ts?raw'

describe('RtcSession public interfaces', () => {
  it('keeps exactly two client-visible connection paths', () => {
    expect(CONNECTION_PATHS).toEqual(['local', 'hub'])
    expectTypeOf<(typeof CONNECTION_PATHS)[number]>().toEqualTypeOf<ConnectionPath>()
    expect(transportSource).not.toMatch(/anonymous_p2p|managed_p2p|paid_relay/)
  })

  it('keeps SmartRoute diagnostics stable without exposing private scores', () => {
    expect(ROUTE_SELECTION_REASONS).toEqual([
      'initial_best', 'only_viable', 'lower_loss', 'direct_unstable', 'lower_latency', 'lower_score',
      'cost_guard', 'minimum_hold', 'cooldown', 'hysteresis_hold', 'insufficient_improvement',
      'current_unavailable', 'current_best',
    ])
    expect(transportSource).not.toMatch(/routeScore|scoreWeight|costBudget/)
  })

  it('exposes one platform-neutral runtime session surface', () => {
    expectTypeOf<keyof RtcSession>().toEqualTypeOf<
      | 'openTerminal'
      | 'openApi'
      | 'openFileChannel'
      | 'subscribeEvents'
      | 'getConnectionInfo'
      | 'getCapabilities'
      | 'disconnect'
    >()
    expectTypeOf<RtcSession>().toMatchTypeOf<{
      openTerminal(terminalId: string): Promise<RtcBinaryChannel>
      openApi(): Promise<RtcJsonRpcChannel>
      openFileChannel(channel: number, transferId: string): Promise<RtcBinaryChannel>
      subscribeEvents(handler: (event: unknown) => void): { close(): void }
      getConnectionInfo(): Promise<ConnectionInfo>
      getCapabilities(): Promise<ConnectionCapabilities>
      disconnect(): Promise<void>
    }>()
    expectTypeOf<keyof ConnectionInfo>().toEqualTypeOf<
      | 'path'
      | 'routeId'
      | 'routeKind'
      | 'observedPath'
      | 'routeSelectionReason'
      | 'connectionId'
      | 'machineId'
      | 'terminalId'
      | 'relayInUse'
      | 'type'
      | 'localAddr'
      | 'remoteAddr'
      | 'localBaseAddr'
      | 'remoteBaseAddr'
      | 'candidatePairId'
      | 'candidateType'
      | 'remoteCandidateType'
      | 'rtt'
      | 'localProtocol'
      | 'remoteProtocol'
      | 'relayTransport'
      | 'networkClass'
      | 'sampledAt'
      | 'bytesSent'
      | 'bytesReceived'
      | 'packetsSent'
      | 'lossEvents'
      | 'generation'
    >()
    expectTypeOf<keyof ConnectionCapabilities>().toEqualTypeOf<
      | 'terminalAllowed'
      | 'apiAllowed'
      | 'eventsAllowed'
      | 'fileTransferAllowed'
      | 'terminalManagementAllowed'
      | 'relayInUse'
      | 'denialReason'
    >()
    expectTypeOf<RtcConnector<{ machineId: string }>>().toMatchTypeOf<{
      connect(input: { machineId: string }, options?: unknown): Promise<RtcSession>
    }>()
    expectTypeOf<keyof ManagedRtcSession>().toEqualTypeOf<
      | keyof RtcSession
      | 'subscribeConnectionState'
      | 'onDisconnect'
      | 'isAlive'
      | 'handleAppResume'
      | 'waitUntilConnected'
      | 'closeTerminalDataChannel'
    >()
    expectTypeOf<ManagedRtcConnector<{ machineId: string }>>().toMatchTypeOf<{
      connect(input: { machineId: string }, options?: unknown): Promise<ManagedRtcSession>
    }>()
    expectTypeOf<RtcBinaryChannel>().toMatchTypeOf<{
      onMessage(handler: (data: Uint8Array) => void): { close(): void }
      onClose(handler: () => void): { close(): void }
    }>()
  })

  it('does not expose old transport boundaries or browser/native implementation details', () => {
    expect(transportSource).not.toMatch(/\bRemoteTransport\b|\bPeerTransport\b|\bTerminalTransport\b/)
    expect(transportSource).not.toMatch(/RTCPeerConnection|RTCDataChannel|nativePlugin|turnCredential|iceServer/i)
  })

  it('exposes only bounded user policy for relay transport selection', () => {
    expectTypeOf<RelayTransportPreference>().toEqualTypeOf<'auto' | 'udp' | 'tcp'>()
    expectTypeOf<ConnectionPolicy>().toEqualTypeOf<{
      route: 'auto' | 'direct' | 'ssh' | 'cloud'
      cloud: 'auto' | 'p2p' | 'relay'
      relayTransport: RelayTransportPreference
    }>()
  })

  it('keeps local status api and inventory events outside runtime transport taxonomy', () => {
    expectTypeOf<LocalAgentApi>().toHaveProperty('getStatus')
    expectTypeOf<TerminalInventoryEvents>().toMatchTypeOf<{
      subscribe(machineId: string, handler: (event: { type: 'inventory_changed' }) => void): { close(): void }
    }>()
  })

  it('defines browser-neutral network runtime dependencies before browser implementations', () => {
    expectTypeOf<RemoteRuntimeStorage>().toMatchTypeOf<{
      getItem(key: string): string | null
      setItem(key: string, value: string): void
      removeItem(key: string): void
    }>()
    expectTypeOf<RemoteNetworkRuntime>().toMatchTypeOf<{
      fetch(input: string, init?: RequestInit): Promise<Response>
      storage?: RemoteRuntimeStorage | undefined
      queryParam(name: string): string | null
    }>()
    expect(transportSource).toMatch(/interface RemoteNetworkRuntime/)
  })
})
