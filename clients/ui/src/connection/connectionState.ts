import type {
  ConnectionPath,
  RtcConnectionPhase,
  RtcConnectionStateSnapshot,
  RtcSubscription,
} from '../core/transport'

export interface ConnectionStatePublisher {
  subscribe(handler: (snapshot: RtcConnectionStateSnapshot) => void): RtcSubscription
  publish(snapshot: RtcConnectionStateSnapshot): void
}

export function createConnectionStatePublisher(): ConnectionStatePublisher {
  const handlers = new Set<(snapshot: RtcConnectionStateSnapshot) => void>()
  return {
    subscribe(handler) {
      handlers.add(handler)
      return { close: () => handlers.delete(handler) }
    },
    publish(snapshot) {
      for (const handler of handlers) handler(snapshot)
    },
  }
}

export function connectionStatusIsSettled(phase: RtcConnectionPhase | null | undefined): boolean {
  return phase === 'connected' || phase === 'idle' || phase === 'failed'
}

const connectionPhaseKeys: Record<RtcConnectionPhase, string> = {
  idle: 'workspace.connection.phase.ready',
  probing: 'workspace.connection.phase.probing',
  resolving: 'workspace.connection.phase.resolving',
  signaling: 'workspace.connection.phase.signaling',
  connecting: 'workspace.connection.phase.connecting',
  authorizing: 'workspace.connection.phase.authorizing',
  connected: 'workspace.connection.phase.connected',
  verifying: 'workspace.connection.phase.verifying',
  reconnecting: 'workspace.connection.phase.reconnecting',
  waiting_network: 'workspace.connection.phase.waiting_network',
  failed: 'workspace.connection.phase.failed',
}

const englishConnectionPhaseLabels: Record<RtcConnectionPhase, string> = {
  idle: 'Ready',
  probing: 'Finding the best available connection...',
  resolving: 'Preparing a secure connection...',
  signaling: 'Connecting to the device...',
  connecting: 'Connecting to the device...',
  authorizing: 'Verifying device access...',
  connected: 'Connected',
  verifying: 'Checking the connection...',
  reconnecting: 'Connection interrupted. Reconnecting...',
  waiting_network: 'Your phone is offline.',
  failed: 'Connection failed',
}

/**
 * connectionPhaseLabel 把 Go Client Engine 的稳定阶段投影为用户文案。
 * phase 是连接过程真值；底层自由文本只用于诊断，不能越过这里暴露 JNI、handle 或 runtime 实现细节。
 */
export function connectionPhaseLabel(
  phase: RtcConnectionPhase | null | undefined,
  translate?: ((key: string) => string) | undefined,
): string {
  const normalized = phase ?? 'idle'
  return translate?.(connectionPhaseKeys[normalized]) ?? englishConnectionPhaseLabels[normalized]
}

/**
 * connectionPathLabel 为旧的 local/hub transport projection 提供用户概念标签。
 * 它不改变 path 真值，也不会把内部 Hub 或 runtime owner 名称投影到产品界面。
 */
export function connectionPathLabel(path: ConnectionPath | undefined): string {
  if (path === 'hub') return 'AnyTTY Cloud'
  if (path === 'local') return 'Local'
  return 'Connection'
}

export function connectionSnapshotFromStatus(input: {
  machineId: string
  statusText: string
  phase?: RtcConnectionPhase | undefined
  path?: ConnectionPath | undefined
  relayInUse?: boolean | undefined
}): RtcConnectionStateSnapshot {
  const status = input.statusText.trim()
  return {
    machineId: input.machineId,
    phase: input.phase ?? inferConnectionPhase(status),
    ...(input.path ? { path: input.path } : {}),
    statusText: status || connectionPhaseLabel(input.phase),
    relayInUse: input.relayInUse === true,
  }
}

export function inferConnectionPhase(statusText: string): RtcConnectionPhase {
  const text = statusText.toLowerCase()
  if (text.includes('connected')) return 'connected'
  if (text.includes('verify') || text.includes('verifying')) return 'verifying'
  if (text.includes('reconnect') || text.includes('switching') || text.includes('trying p2p')) return 'reconnecting'
  if (text.includes('waiting') && text.includes('network')) return 'waiting_network'
  if (text.includes('probe') || text.includes('racing') || text.includes('checking')) return 'probing'
  if (text.includes('fail') || text.includes('error')) return 'failed'
  return 'connecting'
}

export function connectionStateFromAttempt(input: {
  machineId: string
  stage: string
  message: string
  path?: ConnectionPath | undefined
  relayInUse?: boolean | undefined
}): RtcConnectionStateSnapshot {
  return {
    machineId: input.machineId,
    phase: phaseFromAttemptStage(input.stage),
    ...(input.path ? { path: input.path } : {}),
    statusText: input.message,
    relayInUse: input.relayInUse === true,
  }
}

function phaseFromAttemptStage(stage: string): RtcConnectionPhase {
  if (stage === 'connected') return 'connected'
  if (stage === 'failed') return 'failed'
  if (stage === 'trying_local' || stage === 'trying_hub') return 'probing'
  return 'connecting'
}
