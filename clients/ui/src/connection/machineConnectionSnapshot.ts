import type { ConnectionInfo, RtcConnectionPhase } from '../core/transport'
import type { ProtoClientSessionCloseError } from '../core/protoClientSession'

/** MachineConnectionSnapshot is a UI-only connection projection; it never owns a session or reconnect state. */
export interface MachineConnectionSnapshot {
  machineId: string
  phase: RtcConnectionPhase
  statusText: string
  connectionInfo: ConnectionInfo | null
  forceRelay: boolean
  relayInUse: boolean
  reconnectAttempt: number
  error: ProtoClientSessionCloseError | null
}
