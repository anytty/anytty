import { describe, expect, it } from 'vitest'
import {
  applyTerminalModifiers,
  applyXtermNavigationModifiers,
  nextModifierState,
  type TerminalModifierState,
} from './mobileTerminalInput'

describe('mobile terminal input helpers', () => {
  it('maps supported single characters and consumes only modifiers that applied', () => {
    expect(applyTerminalModifiers('c', { ctrl: 'once', alt: 'off' })).toEqual({
      data: '\x03',
      ctrl: 'off',
      alt: 'off',
    })
    expect(applyTerminalModifiers('1', { ctrl: 'once', alt: 'once' })).toEqual({
      data: '\x1b1',
      ctrl: 'once',
      alt: 'off',
    })
  })

  it('preserves locked modifiers while applying Ctrl before Alt', () => {
    expect(applyTerminalModifiers('x', { ctrl: 'locked', alt: 'locked' })).toEqual({
      data: '\x1b\x18',
      ctrl: 'locked',
      alt: 'locked',
    })
  })

  it('maps keybar navigation sequences to xterm Ctrl, Alt, and combined modifiers', () => {
    const cases: Array<[string, TerminalModifierState, string]> = [
      ['\x1b[A', { ctrl: 'once', alt: 'off' }, '\x1b[1;5A'],
      ['\x1b[B', { ctrl: 'off', alt: 'once' }, '\x1b[1;3B'],
      ['\x1b[C', { ctrl: 'once', alt: 'once' }, '\x1b[1;7C'],
      ['\x1b[D', { ctrl: 'once', alt: 'once' }, '\x1b[1;7D'],
      ['\x1b[H', { ctrl: 'once', alt: 'once' }, '\x1b[1;7H'],
      ['\x1b[F', { ctrl: 'once', alt: 'once' }, '\x1b[1;7F'],
      ['\x1b[5~', { ctrl: 'once', alt: 'once' }, '\x1b[5;7~'],
      ['\x1b[6~', { ctrl: 'once', alt: 'once' }, '\x1b[6;7~'],
    ]

    for (const [input, state, data] of cases) {
      expect(applyTerminalModifiers(input, state)).toEqual({ data, ctrl: 'off', alt: 'off' })
    }
  })

  it('only applies xterm onData modifiers to the explicit navigation whitelist', () => {
    const state: TerminalModifierState = { ctrl: 'once', alt: 'once' }
    expect(applyXtermNavigationModifiers('\x1b[A', state)).toEqual({
      data: '\x1b[1;7A',
      ctrl: 'off',
      alt: 'off',
    })
    for (const input of ['c', '中', 'paste', '\x1b[200~c\x1b[201~', '\x1b[Z']) {
      expect(applyXtermNavigationModifiers(input, state)).toEqual({ data: input, ...state })
    }
  })

  it('does not apply or consume modifiers for paste, IME, or unsupported text batches', () => {
    const state: TerminalModifierState = { ctrl: 'once', alt: 'once' }
    for (const input of ['中', 'hello', '你好', '\x1b[200~c\x1b[201~', '\x1b[Z']) {
      expect(applyTerminalModifiers(input, state)).toEqual({ data: input, ...state })
    }
  })

  it('cycles modifiers through off, once, locked, and back to off', () => {
    expect(nextModifierState('off')).toBe('once')
    expect(nextModifierState('once')).toBe('locked')
    expect(nextModifierState('locked')).toBe('off')
  })

  it('applies Ctrl before Alt and consumes both one-shot modifiers together', () => {
    expect(applyTerminalModifiers('\\', { ctrl: 'once', alt: 'once' })).toEqual({
      data: '\x1b\x1c',
      ctrl: 'off',
      alt: 'off',
    })
  })
})
