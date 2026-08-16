import { beforeEach, describe, expect, it } from 'vitest'
import { pinTerminal, readTerminalOrder, reorderPinnedTerminal, sortTerminalIds, unpinTerminal, writeTerminalOrder } from './terminalOrder'

describe('terminalOrder', () => {
  beforeEach(() => {
    const values = new Map<string, string>()
    Object.defineProperty(window, 'localStorage', {
      configurable: true,
      value: {
        clear: () => values.clear(),
        getItem: (key: string) => values.get(key) ?? null,
        setItem: (key: string, value: string) => values.set(key, value),
      },
    })
  })

  it('supports multiple pins and reorders only the pinned group', () => {
    expect(pinTerminal(pinTerminal([], 'c'), 'a')).toEqual(['c', 'a'])
    expect(reorderPinnedTerminal(['c', 'a', 'b'], 'b', 'c')).toEqual(['b', 'c', 'a'])
    expect(reorderPinnedTerminal(['c', 'a', 'b'], 'c', 'b', 'after')).toEqual(['a', 'b', 'c'])
    expect(unpinTerminal(['c', 'a'], 'c')).toEqual(['a'])
    expect(sortTerminalIds([{ terminalId: 'a' }, { terminalId: 'new' }, { terminalId: 'c' }], ['c', 'a']))
      .toEqual([{ terminalId: 'c' }, { terminalId: 'a' }, { terminalId: 'new' }])
  })

  it('stores each machine order locally', () => {
    writeTerminalOrder('machine-a', ['b', 'a', 'b'])
    expect(readTerminalOrder('machine-a')).toEqual(['b', 'a'])
    expect(readTerminalOrder('machine-b')).toEqual([])
  })
})
