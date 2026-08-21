export type AppConnectionState = 'ready' | 'checking' | 'recovering' | 'failed'

export function appConnectionIsReady(state: AppConnectionState): boolean {
  return state === 'ready'
}

export function appConnectionShowsRecovery(state: AppConnectionState): boolean {
  return state === 'recovering' || state === 'failed'
}
