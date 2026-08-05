import { describe, expect, it } from 'vitest'
import { createMachineStore, type StoredMachineRecord } from './machineStore'

const machine = (): StoredMachineRecord => ({
  machineId: 'machine-1', name: 'Dev MacBook', state: 'online', terminalCount: 2, source: 'manual', accessClass: 'cloud',
  addresses: { local: [], lan: [], public: [] }, endpoints: {},
  addedAt: '2026-05-03T16:00:00.000Z', updatedAt: '2026-05-03T16:00:00.000Z',
})

describe('machine store', () => {
  it('stores only non-secret machine projection data', () => {
    const storage = new MemoryStorage()
    const store = createMachineStore({ storage })
    expect(store.saveMachine(machine())).toEqual(machine())
    expect(store.getMachine('machine-1')).toEqual(machine())
    expect(storage.getItem('anytty.app.machines.v2')).not.toMatch(/pairing|sessionToken|secret/)
  })

  it('drops obsolete pairing fields while reading the current store', () => {
    const storage = new MemoryStorage()
    storage.setItem('anytty.app.machines.v2', JSON.stringify([{ ...machine(), pairing: { sessionId: 'old', secret: 'old-secret' } }]))
    expect(createMachineStore({ storage }).listMachines()[0]).not.toHaveProperty('pairing')
  })

  it('does not read the previous development store version', () => {
    const storage = new MemoryStorage()
    storage.setItem('anytty.app.machines.v1', JSON.stringify([machine()]))
    expect(createMachineStore({ storage }).listMachines()).toEqual([])
  })

  it('rejects records that omit the explicit access class', () => {
    const storage = new MemoryStorage()
    const { accessClass: _accessClass, ...legacy } = machine()
    storage.setItem('anytty.app.machines.v2', JSON.stringify([legacy]))
    expect(() => createMachineStore({ storage }).listMachines()).toThrow('invalid machine access class')
  })

  it('rejects removed connection path names', () => {
    const storage = new MemoryStorage()
    storage.setItem('anytty.app.machines.v2', JSON.stringify([{ ...machine(), lastConnectionPath: 'managed' }]))
    expect(() => createMachineStore({ storage }).listMachines()).toThrow(/invalid connection path managed/i)
  })

  it('rejects private key material before persistence and on read', () => {
    const storage = new MemoryStorage()
    const store = createMachineStore({ storage })
    expect(() => store.saveMachine({ ...machine(), appPrivateKey: 'not-allowed' } as never)).toThrow(/private key/i)
    storage.setItem('anytty.app.machines.v2', JSON.stringify([{ ...machine(), private_key: 'not-allowed' }]))
    expect(() => store.listMachines()).toThrow(/private key/i)
  })
})

class MemoryStorage implements Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> {
  private readonly values = new Map<string, string>()
  getItem(key: string): string | null { return this.values.get(key) ?? null }
  removeItem(key: string): void { this.values.delete(key) }
  setItem(key: string, value: string): void { this.values.set(key, value) }
}
