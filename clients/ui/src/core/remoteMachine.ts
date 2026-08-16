/**
 * RemoteMachine 是共享客户端可展示的设备投影。
 * 它来自本地 Endpoint registry，不拥有账号目录、连接生命周期或 terminal 真值。
 */
export interface RemoteMachine {
  id: string
  name: string
  hostname?: string | undefined
  osInfo?: string | undefined
  online: boolean
  source: 'local'
  hubId?: string | undefined
  hubUrls: string[]
  localHubUrls?: string[] | undefined
  localFallbackHubUrls?: string[] | undefined
  lastSeen?: string | undefined
}
