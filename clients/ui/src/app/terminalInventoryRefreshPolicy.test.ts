import { describe, expect, it } from 'vitest'
import { terminalInventoryRefreshIntervalMs } from './MachineWorkspace'

describe('terminal inventory refresh policy', () => {
  it('refreshes direct paths more often than relay paths', () => {
    expect(terminalInventoryRefreshIntervalMs(false)).toBe(2_000)
    expect(terminalInventoryRefreshIntervalMs(true)).toBe(5_000)
  })
})
