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

export interface NativeConnectionPlugin extends Plugin {
  addListener(eventName: 'networkChanged', listener: (event: NativeNetworkChangedEvent) => void): Promise<PluginListenerHandle>
  addListener(eventName: 'localDiscoveryChanged', listener: (event: NativeLocalDiscoveryChangedEvent) => void): Promise<PluginListenerHandle>
  addListener(eventName: 'disconnectAllRequested', listener: (event: NativeDisconnectAllRequestedEvent) => void): Promise<PluginListenerHandle>
  handleForegroundResume(): Promise<void>
  getNetworkSnapshot(): Promise<NativeNetworkChangedEvent>
  resetLocalPairings(): Promise<void>
  getBridgeEndpoint(): Promise<NativeBridgeEndpoint>
  setSessionActive(input: { machineId: string; active: boolean }): Promise<void>
  isLocalEndpointDiscovered(input: { deviceId: string; fingerprint: string }): Promise<{ discovered: boolean }>
}

export const NativeConnection = registerPlugin<NativeConnectionPlugin>('NativeConnection')
