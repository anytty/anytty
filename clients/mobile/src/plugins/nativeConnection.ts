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

export interface NativeDisconnectAllRequestedEvent {}

export interface NativeSessionDemandResult {
  goManagedEndpointIds: string[]
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
  getNetworkSnapshot(): Promise<NativeNetworkChangedEvent>
  resetLocalPairings(): Promise<void>
  getBridgeEndpoint(): Promise<NativeBridgeEndpoint>
  replaceSessionDemand(input: { endpointIds: string[] }): Promise<NativeSessionDemandResult>
  isLocalEndpointDiscovered(input: { deviceId: string; fingerprint: string }): Promise<{ discovered: boolean }>
}

export const NativeConnection = registerPlugin<NativeConnectionPlugin>('NativeConnection')
