import { registerPlugin } from '@capacitor/core'
import type { Plugin, PluginListenerHandle } from '@capacitor/core'

export interface NativeBridgeEndpoint {
  port: number
  token: string
}

export interface NativeNetworkChangedEvent {
  epoch: number
  connected: boolean
  reason: 'available' | 'offline' | 'network_replaced' | 'path_changed'
  scope: 'session'
}

export interface NativeLocalDiscoveryChangedEvent {}

export interface NativeDisconnectAllRequestedEvent {
  stopEpoch: string
  stopped: boolean
}

export interface NativeSessionDemandLease {
  attachmentId: string
  demandRevision: string
  stopEpoch: string
  endpointIds: string[]
  stopped: boolean
}

export interface NativeSessionDemandInput {
  attachmentId: string
  baseDemandRevision: string
  endpointIds: string[]
}

export interface NativeSessionDemandResumeInput {
  intentId: string
  baseStopEpoch: string
}

export interface NativeSessionDemandResumeResult extends NativeSessionDemandLease {
  outcome: 'resumed' | 'stopped'
}

export interface NativeDiagnosticBundleResult {
  name: string
  path: string
  bytes: number
  sha256: string
}

export interface NativeConnectionPlugin extends Plugin {
  addListener(eventName: 'networkChanged', listener: (event: NativeNetworkChangedEvent) => void): Promise<PluginListenerHandle>
  addListener(eventName: 'localDiscoveryChanged', listener: (event: NativeLocalDiscoveryChangedEvent) => void): Promise<PluginListenerHandle>
  addListener(eventName: 'disconnectAllRequested', listener: (event: NativeDisconnectAllRequestedEvent) => void): Promise<PluginListenerHandle>
  writeDebugDiagnostic(input: { value: string }): Promise<void>
  shareDiagnosticBundle(): Promise<NativeDiagnosticBundleResult>
  handleForegroundResume(): Promise<void>
  requestEndpointRecovery(input: { endpointId: string }): Promise<void>
  getNetworkSnapshot(): Promise<NativeNetworkChangedEvent>
  resetLocalPairings(): Promise<void>
  getBridgeEndpoint(): Promise<NativeBridgeEndpoint>
  getSessionDemandLease(): Promise<NativeSessionDemandLease>
  acknowledgeDisconnectAll(input: { stopEpoch: string }): Promise<void>
  resumeSessionDemand(input: NativeSessionDemandResumeInput): Promise<NativeSessionDemandResumeResult>
  replaceSessionDemand(input: NativeSessionDemandInput): Promise<NativeSessionDemandLease>
  isLocalEndpointDiscovered(input: { deviceId: string; fingerprint: string }): Promise<{ discovered: boolean }>
  isDirectRouteReachable(input: { routeProtoBase64: string }): Promise<{ reachable: boolean }>
}

export const NativeConnection = registerPlugin<NativeConnectionPlugin>('NativeConnection')
