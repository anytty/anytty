export type ModifierState = 'off' | 'once' | 'locked'

export interface TerminalModifierState {
  ctrl: ModifierState
  alt: ModifierState
}

export interface TerminalModifierResult extends TerminalModifierState {
  data: string
}

const XTERM_MODIFIABLE_SEQUENCES: Readonly<Record<string, readonly [prefix: string, suffix: string]>> = {
  '\x1b[A': ['\x1b[1;', 'A'],
  '\x1b[B': ['\x1b[1;', 'B'],
  '\x1b[C': ['\x1b[1;', 'C'],
  '\x1b[D': ['\x1b[1;', 'D'],
  '\x1b[H': ['\x1b[1;', 'H'],
  '\x1b[F': ['\x1b[1;', 'F'],
  '\x1b[5~': ['\x1b[5;', '~'],
  '\x1b[6~': ['\x1b[6;', '~'],
}

export function applyTerminalModifiers(data: string, state: TerminalModifierState): TerminalModifierResult {
  const navigationResult = applyXtermNavigationModifiers(data, state)
  if (navigationResult.data !== data) return navigationResult

  if (data.length !== 1 || data.charCodeAt(0) > 0x7f) {
    return { data, ctrl: state.ctrl, alt: state.alt }
  }

  let nextData = data
  let nextCtrl = state.ctrl
  let nextAlt = state.alt

  if (state.ctrl !== 'off') {
    const ctrlInput = ctrlData(nextData)
    if (ctrlInput !== nextData) {
      nextData = ctrlInput
      nextCtrl = consumeOnce(state.ctrl)
    }
  }

  if (state.alt !== 'off') {
    nextData = `\x1b${nextData}`
    nextAlt = consumeOnce(state.alt)
  }

  return {
    data: nextData,
    ctrl: nextCtrl,
    alt: nextAlt,
  }
}

export function applyXtermNavigationModifiers(data: string, state: TerminalModifierState): TerminalModifierResult {
  const specialSequence = XTERM_MODIFIABLE_SEQUENCES[data]
  if (specialSequence && (state.ctrl !== 'off' || state.alt !== 'off')) {
    const modifierParameter = 1 + (state.alt !== 'off' ? 2 : 0) + (state.ctrl !== 'off' ? 4 : 0)
    return {
      data: `${specialSequence[0]}${modifierParameter}${specialSequence[1]}`,
      ctrl: consumeOnce(state.ctrl),
      alt: consumeOnce(state.alt),
    }
  }
  return { data, ctrl: state.ctrl, alt: state.alt }
}

export function nextModifierState(current: ModifierState): ModifierState {
  if (current === 'off') return 'once'
  if (current === 'once') return 'locked'
  return 'off'
}

function consumeOnce(state: ModifierState): ModifierState {
  return state === 'once' ? 'off' : state
}

function ctrlData(data: string): string {
  const code = data.charCodeAt(0)
  if (code >= 64 && code <= 95) return String.fromCharCode(code - 64)
  if (code >= 97 && code <= 122) return String.fromCharCode(code - 96)
  return data
}
