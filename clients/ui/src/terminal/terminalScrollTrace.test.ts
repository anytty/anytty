import { afterEach, describe, expect, it, vi } from 'vitest'
import { describeTerminalMouseInput, traceTerminalScroll } from './terminalScrollTrace'

describe('terminal scroll tracing', () => {
  afterEach(() => {
    delete (globalThis as typeof globalThis & { __anyttyTerminalScrollTrace?: unknown }).__anyttyTerminalScrollTrace
    vi.restoreAllMocks()
  })

  it('recognizes SGR wheel input without treating keyboard input as mouse input', () => {
    expect(describeTerminalMouseInput('\u001b[<64;12;7M')).toEqual({
      encoding: 'sgr',
      code: 64,
      col: 12,
      row: 7,
      action: 'press',
      direction: 'up',
    })
    expect(describeTerminalMouseInput('\u001b[<65;12;7M')?.direction).toBe('down')
    expect(describeTerminalMouseInput('\u001b[A')).toBeUndefined()
    expect(describeTerminalMouseInput('secret typed text')).toBeUndefined()
  })

  it('records only while explicitly enabled', () => {
    const debug = vi.spyOn(console, 'debug').mockImplementation(() => undefined)
    const state: { enabled: boolean; events: unknown[] } = { enabled: false, events: [] }
    ;(globalThis as typeof globalThis & { __anyttyTerminalScrollTrace?: unknown }).__anyttyTerminalScrollTrace = state

    traceTerminalScroll('ignored')
    state.enabled = true
    traceTerminalScroll('touch.move', { terminalId: 'term-1', details: { dy: 12 } })

    expect(state.events).toHaveLength(1)
    expect(state.events[0]).toMatchObject({
      sequence: 1,
      event: 'touch.move',
      terminalId: 'term-1',
      details: { dy: 12 },
    })
    expect(debug).toHaveBeenCalledOnce()
  })
})
