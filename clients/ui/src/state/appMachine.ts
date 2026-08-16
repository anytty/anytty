import type { ConnectionPath } from '../core/transport'

export type AppMachineState = 'online' | 'offline' | 'stale' | 'unknown' | 'connecting'
export type AppMachineSource = 'local' | 'hub' | 'manual'
export const MACHINE_ICON_NAMES = [
  'apple',
  'windows',
  'monitor',
  'laptop',
  'phone',
  'tablet',
  'server',
  'storage',
  'terminal',
  'router',
  'cloud',
  'chip',
] as const
export type MachineIconName = (typeof MACHINE_ICON_NAMES)[number]

/** MachineAccessClass 表示机器可用的产品接入能力；local 是兼容存储值，UI 将其呈现为包含 Direct/SSH 的“直连”。 */
export type MachineAccessClass = 'local' | 'cloud' | 'local_cloud'

export interface AppMachineRecord {
  machineId: string
  name: string
  alias?: string | undefined
  icon?: MachineIconName | undefined
  iconImage?: string | undefined
  hostname?: string | undefined
  osInfo?: string | undefined
  hubId?: string | undefined
  state: AppMachineState
  terminalCount: number
  lastSeenAt?: string | undefined
  lastConnectionPath?: ConnectionPath | undefined
  preferredPath?: ConnectionPath | undefined
  relayInUse?: boolean | undefined
  source: AppMachineSource
  accessClass?: MachineAccessClass | undefined
}

export type ConnectionFlowStage =
  | 'idle'
  | 'trying_local'
  | 'trying_hub'
  | 'connected'
  | 'failed'

export interface ConnectionFlowSnapshot {
  stage: ConnectionFlowStage
  path?: ConnectionPath | undefined
  relayInUse?: boolean | undefined
}
