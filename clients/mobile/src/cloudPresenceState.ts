import type { CloudPresenceReachability, CloudPresenceSnapshot } from '@anytty/ui'

export type CloudPresenceState = CloudPresenceSnapshot

export function mergeCloudPresenceResults(
  current: ReadonlyMap<string, CloudPresenceState>,
  endpointIds: readonly string[],
  results: readonly PromiseSettledResult<CloudPresenceSnapshot>[],
): ReadonlyMap<string, CloudPresenceState> {
  const next = new Map<string, CloudPresenceState>()
  endpointIds.forEach((endpointId, index) => {
    const result = results[index]
    if (result?.status === 'fulfilled') {
      next.set(endpointId, {
        ...result.value,
        endpointId: result.value.endpointId ?? endpointId,
        state: result.value.online === true ? 'online' : result.value.online === false ? 'offline' : result.value.state,
      })
      return
    }
    const previous = current.get(endpointId)
    next.set(endpointId, previous?.state === 'offline' ? previous : cloudPresenceWithState(previous, 'unknown', endpointId))
  })
  return next
}

export function cloudPresenceWithState(
  current: CloudPresenceState | undefined,
  state: CloudPresenceReachability,
  endpointId: string,
): CloudPresenceState {
  return {
    ...current,
    endpointId: current?.endpointId ?? endpointId,
    state,
    online: state === 'online' ? true : state === 'offline' ? false : undefined,
  }
}

export function samePresenceMap(
  left: ReadonlyMap<string, CloudPresenceState>,
  right: ReadonlyMap<string, CloudPresenceState>,
): boolean {
  return left.size === right.size && [...left].every(([key, value]) => sameCloudPresenceSnapshot(value, right.get(key)))
}

function sameCloudPresenceSnapshot(left: CloudPresenceSnapshot | undefined, right: CloudPresenceSnapshot | undefined): boolean {
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
