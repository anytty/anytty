import type { RemoteRuntimeStorage } from '../core/transport'

export const ACTIVE_WORKSPACE_STORAGE_KEY = 'anytty.app.active-workspace.v1'

const activeWorkspaceVersion = 1

interface ActiveWorkspaceRecord {
  version: typeof activeWorkspaceVersion
  machineId: string
}

export function readActiveWorkspaceMachineId(storage: Pick<RemoteRuntimeStorage, 'getItem'> | undefined): string | null {
  if (!storage) return null
  try {
    const raw = storage.getItem(ACTIVE_WORKSPACE_STORAGE_KEY)
    if (!raw) return null
    const parsed = JSON.parse(raw) as Partial<ActiveWorkspaceRecord> | null
    if (!parsed || parsed.version !== activeWorkspaceVersion || typeof parsed.machineId !== 'string') return null
    const machineId = parsed.machineId.trim()
    return machineId.length > 0 ? machineId : null
  } catch {
    return null
  }
}

export function writeActiveWorkspaceMachineId(
  storage: Pick<RemoteRuntimeStorage, 'setItem' | 'removeItem'> | undefined,
  machineId: string | null,
): void {
  if (!storage) return
  try {
    const normalized = machineId?.trim() ?? ''
    if (!normalized) {
      storage.removeItem(ACTIVE_WORKSPACE_STORAGE_KEY)
      return
    }
    const record: ActiveWorkspaceRecord = { version: activeWorkspaceVersion, machineId: normalized }
    storage.setItem(ACTIVE_WORKSPACE_STORAGE_KEY, JSON.stringify(record))
  } catch {
    // Navigation remains usable when WebView storage is unavailable or full.
  }
}
