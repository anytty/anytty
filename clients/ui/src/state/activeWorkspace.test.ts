import { describe, expect, it } from 'vitest'
import { ACTIVE_WORKSPACE_STORAGE_KEY, readActiveWorkspaceMachineId, writeActiveWorkspaceMachineId } from './activeWorkspace'

class MemoryStorage {
  private readonly values = new Map<string, string>()

  getItem(key: string): string | null { return this.values.get(key) ?? null }
  setItem(key: string, value: string): void { this.values.set(key, value) }
  removeItem(key: string): void { this.values.delete(key) }
}

describe('active workspace journal', () => {
  it('round trips the active machine and clears explicit navigation home', () => {
    const storage = new MemoryStorage()

    writeActiveWorkspaceMachineId(storage, ' machine-a ')
    expect(readActiveWorkspaceMachineId(storage)).toBe('machine-a')

    writeActiveWorkspaceMachineId(storage, null)
    expect(readActiveWorkspaceMachineId(storage)).toBeNull()
  })

  it('ignores malformed or obsolete records without breaking app startup', () => {
    const storage = new MemoryStorage()
    storage.setItem(ACTIVE_WORKSPACE_STORAGE_KEY, '{not-json')
    expect(readActiveWorkspaceMachineId(storage)).toBeNull()

    storage.setItem(ACTIVE_WORKSPACE_STORAGE_KEY, JSON.stringify({ version: 2, machineId: 'machine-a' }))
    expect(readActiveWorkspaceMachineId(storage)).toBeNull()
  })

  it('does not block navigation when storage writes fail', () => {
    const storage = {
      getItem: () => null,
      setItem: () => { throw new Error('quota exceeded') },
      removeItem: () => { throw new Error('storage unavailable') },
    }

    expect(() => writeActiveWorkspaceMachineId(storage, 'machine-a')).not.toThrow()
    expect(() => writeActiveWorkspaceMachineId(storage, null)).not.toThrow()
  })
})
