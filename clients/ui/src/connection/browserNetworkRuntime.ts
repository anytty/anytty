import type { RemoteNetworkRuntime, RemoteRuntimeFetch, RemoteRuntimeStorage } from '../core/transport'

export interface BrowserRemoteNetworkRuntimeOptions {
  fetch?: RemoteRuntimeFetch | undefined
  storage?: RemoteRuntimeStorage | null | undefined
  location?: Pick<Location, 'search'> | null | undefined
}

export function createBrowserRemoteNetworkRuntime(options: BrowserRemoteNetworkRuntimeOptions = {}): RemoteNetworkRuntime {
  const fetchImpl = options.fetch ?? ((input, init) => globalThis.fetch(input, init))
  const storage = options.storage === null ? undefined : options.storage ?? browserStorage()
  const locationSearch = options.location?.search ?? globalThis.location?.search ?? ''
  return {
    fetch: fetchImpl,
    ...(storage ? { storage } : {}),
    queryParam(name) {
      return new URLSearchParams(locationSearch).get(name)
    },
  }
}

export function createFutureNativeRemoteNetworkRuntime(): RemoteNetworkRuntime {
  throw new Error('native remote network runtime is not implemented')
}

function browserStorage(): RemoteRuntimeStorage | undefined {
  const storage = globalThis.localStorage
  if (
    !storage ||
    typeof storage.getItem !== 'function' ||
    typeof storage.setItem !== 'function' ||
    typeof storage.removeItem !== 'function'
  ) {
    return undefined
  }
  return storage
}
