import { registerPlugin } from '@capacitor/core'
import type { Plugin, PluginListenerHandle } from '@capacitor/core'

export interface NativeBridgeEndpoint {
  port: number
  token: string
}

export interface NativeConnectionPlugin extends Plugin {
  addListener(eventName: 'networkChanged', listener: (event: { epoch: number; connected: boolean }) => void): Promise<PluginListenerHandle>
  handleForegroundResume(): Promise<void>
  resetLocalPairings(): Promise<void>
  getBridgeEndpoint(): Promise<NativeBridgeEndpoint>
  setSessionActive(input: { machineId: string; active: boolean }): Promise<void>
}

export const NativeConnection = registerPlugin<NativeConnectionPlugin>('NativeConnection')
