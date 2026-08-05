import type { ConnectionPath } from '../core/transport'

export type AppMachineState = 'online' | 'offline' | 'stale' | 'unknown' | 'connecting'
export type AppMachineSource = 'local' | 'hub' | 'manual'

/** MachineAccessClass 表示机器可用的产品接入能力；真值由配对来源和账号 Hub 同步合并，不代表某次连接实际选择的 transport。 */
export type MachineAccessClass = 'local' | 'cloud' | 'local_cloud'

export interface AppMachineRecord {
  machineId: string
  name: string
  hostname?: string | undefined
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
