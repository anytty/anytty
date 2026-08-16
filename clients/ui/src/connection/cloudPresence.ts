export type CloudPresenceReachability = 'checking' | 'online' | 'offline' | 'unknown'

export interface CloudPresenceSnapshot {
  state: CloudPresenceReachability
  online?: boolean | undefined
  endpointId?: string | undefined
  deviceId?: string | undefined
  deviceFingerprint?: string | undefined
  daemonId?: string | undefined
  edgeId?: string | undefined
  edgeName?: string | undefined
  edgeRegion?: string | undefined
  edgePublicEndpoint?: string | undefined
  edgeServerName?: string | undefined
  locatorSource?: string | undefined
  refreshedFromController?: boolean | undefined
}

export type CloudPresenceInput = CloudPresenceReachability | CloudPresenceSnapshot

export function cloudPresenceReachability(input: CloudPresenceInput | null | undefined): CloudPresenceReachability {
  return typeof input === 'string' ? input : input?.state ?? 'unknown'
}

export function cloudPresenceEdgeLabel(input: CloudPresenceInput | null | undefined): string {
  if (!input || typeof input === 'string') return ''
  return input.edgeName?.trim() || input.edgeRegion?.trim() || input.edgeId?.trim() || ''
}

export function sameCloudPresenceSnapshot(left: CloudPresenceSnapshot | undefined, right: CloudPresenceSnapshot | undefined): boolean {
  if (!left || !right) return left === right
  return left.state === right.state &&
    left.online === right.online &&
    left.endpointId === right.endpointId &&
    left.deviceId === right.deviceId &&
    left.deviceFingerprint === right.deviceFingerprint &&
    left.daemonId === right.daemonId &&
    left.edgeId === right.edgeId &&
    left.edgeName === right.edgeName &&
    left.edgeRegion === right.edgeRegion &&
    left.edgePublicEndpoint === right.edgePublicEndpoint &&
    left.edgeServerName === right.edgeServerName &&
    left.locatorSource === right.locatorSource &&
    left.refreshedFromController === right.refreshedFromController
}
