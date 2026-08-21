import type { AppConnectionState } from '@anytty/ui'

export const NATIVE_RECOVERY_NOTICE_DELAY_MS = 1_200

export type NativeRecoveryStatusEvent =
  | { type: 'recovery.started'; visibleImmediately: boolean }
  | { type: 'recovery.noticeDelayElapsed' }
  | { type: 'recovery.succeeded' }
  | { type: 'recovery.failed' }
  | { type: 'recovery.dismissed' }

/** Projects native recovery into user impact without exposing bridge or generation details. */
export function reduceNativeRecoveryStatus(
  state: AppConnectionState,
  event: NativeRecoveryStatusEvent,
): AppConnectionState {
  switch (event.type) {
    case 'recovery.started':
      if (event.visibleImmediately || state === 'recovering' || state === 'failed') return 'recovering'
      return 'checking'
    case 'recovery.noticeDelayElapsed':
      return state === 'checking' ? 'recovering' : state
    case 'recovery.succeeded':
    case 'recovery.dismissed':
      return 'ready'
    case 'recovery.failed':
      return 'failed'
  }
}
