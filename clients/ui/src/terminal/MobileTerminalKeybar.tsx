import { forwardRef, useCallback, useEffect, useRef, useState, type Ref } from 'react'
import { Keyboard, LockKeyhole } from 'lucide-react'
import {
  applyTerminalModifiers,
  nextModifierState,
  type ModifierState,
  type TerminalModifierState,
} from './mobileTerminalInput'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import { useTranslation } from 'react-i18next'
import '../i18n'
import { Button } from '../ui/button'

const keyboardLongPressMs = 400

export interface MobileTerminalKeybarProps {
  onInput: (data: string) => boolean
  onFocusKeyboard?: (() => void) | undefined
  onBlurKeyboard?: (() => void) | undefined
  onToggleKeyboardFocusLock?: (() => void) | undefined
  fnOpen?: boolean | undefined
  onToggleFn?: (() => void) | undefined
  modifierState?: TerminalModifierState | undefined
  onModifierStateChange?: ((state: TerminalModifierState) => void) | undefined
  keyboardVisible?: boolean | undefined
  keyboardFocusLocked?: boolean | undefined
  className?: string | undefined
}

export const MobileTerminalKeybar = forwardRef<HTMLDivElement, MobileTerminalKeybarProps>(function MobileTerminalKeybar({
  onInput,
  onFocusKeyboard,
  onBlurKeyboard,
  onToggleKeyboardFocusLock,
  fnOpen = false,
  onToggleFn,
  modifierState,
  onModifierStateChange,
  keyboardVisible = false,
  keyboardFocusLocked = false,
  className,
}: MobileTerminalKeybarProps, forwardedRef: Ref<HTMLDivElement>) {
  const { t } = useTranslation()
  const [internalModifierState, setInternalModifierState] = useState<TerminalModifierState>({ ctrl: 'off', alt: 'off' })
  const activeModifierState = modifierState ?? internalModifierState
  const setModifierState = onModifierStateChange ?? setInternalModifierState
  const keyboardLongPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const keyboardLongPressTriggeredRef = useRef(false)

  const clearKeyboardLongPress = useCallback(() => {
    if (keyboardLongPressTimerRef.current === null) return
    clearTimeout(keyboardLongPressTimerRef.current)
    keyboardLongPressTimerRef.current = null
  }, [])

  useEffect(() => clearKeyboardLongPress, [clearKeyboardLongPress])

  const send = useCallback((data: string): boolean => {
    const result = applyTerminalModifiers(data, activeModifierState)
    const accepted = onInput(result.data)
    if (!accepted) return false
    hapticSelection()
    if (result.ctrl !== activeModifierState.ctrl || result.alt !== activeModifierState.alt) {
      setModifierState({ ctrl: result.ctrl, alt: result.alt })
    }
    return true
  }, [activeModifierState, onInput, setModifierState])

  const buttonBaseClass = 'relative flex h-9 min-w-0 w-full touch-manipulation select-none items-center justify-center overflow-hidden rounded-md border border-transparent text-center font-mono text-[9px] font-medium'
  const cls = buttonBaseClass
  const keyboardButtonClass = keyboardFocusLocked
    ? 'bg-amber-300 text-zinc-950'
    : keyboardVisible
      ? 'bg-[var(--anytty-accent)] text-[var(--anytty-accent-text)]'
      : 'bg-transparent text-[var(--anytty-muted)] active:bg-[var(--anytty-surface-raised)]'

  const btn = (label: string, data: string, ariaLabel?: string) => {
    const id = label + ':' + data
    return (
      <Button variant="ghost"
        key={id}
        data-key-id={id}
        type="button"
        aria-label={ariaLabel}
        className={`${cls} bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:opacity-70`}
        onPointerDown={(e) => e.preventDefault()}
        onClick={() => send(data)}
      >
        {label}
      </Button>
    )
  }

  const modBtn = (label: string, key: keyof TerminalModifierState, state: ModifierState) => {
    const nextState = nextModifierState(state)
    const stateClass = state === 'locked'
      ? 'bg-amber-300 text-zinc-950'
      : state === 'once'
        ? 'bg-[var(--anytty-accent)] text-[var(--anytty-accent-text)]'
        : 'bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:opacity-70'
    return (
      <Button variant="ghost"
        key={label}
        data-key-id={label}
        data-state={state}
        type="button"
        aria-label={t('terminal.tools.modifierAria', {
          modifier: label,
          state: t(`terminal.tools.modifierState.${state}`),
          nextState: t(`terminal.tools.modifierState.${nextState}`),
        })}
        className={`${cls} gap-0.5 ${stateClass}`}
        onPointerDown={(e) => e.preventDefault()}
        onClick={() => {
          if (nextState === 'locked') hapticImpact()
          else hapticSelection()
          setModifierState({ ...activeModifierState, [key]: nextState })
        }}
      >
        <span className="leading-none">{label}</span>
        {state === 'locked' ? (
          <LockKeyhole aria-hidden="true" className="h-2 w-2" />
        ) : state === 'once' ? (
          <span aria-hidden="true" className="h-1.5 w-1.5 rounded-full bg-current" />
        ) : null}
      </Button>
    )
  }

  return (
    <div
      ref={forwardedRef}
      className={`min-w-0 shrink-0 overflow-x-hidden border-t border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] text-[var(--anytty-text)] md:hidden ${className || ''}`}
      data-testid="anytty-mobile-keybar"
      style={{
        paddingBottom: 'calc(0.125rem + env(safe-area-inset-bottom))',
        paddingLeft: 'env(safe-area-inset-left)',
        paddingRight: 'env(safe-area-inset-right)',
      }}
    >
      <div className="anytty-terminal-key-row grid min-w-0 grid-cols-10 gap-0.5 px-0.5 pt-0.5">
        {btn('Esc', '\x1b')}
        {btn('/', '/')}
        {btn('|', '|')}
        {btn('-', '-')}
        {btn('~', '~')}
        {btn('Home', '\x1b[H')}
        {btn('↑', '\x1b[A')}
        {btn('End', '\x1b[F')}
        {btn('PgU', '\x1b[5~')}
        <Button variant="ghost"
          data-key-id="keyboard-visibility"
          type="button"
          aria-label={t(keyboardFocusLocked
            ? 'terminal.tools.allowKeyboardPopup'
            : keyboardVisible ? 'terminal.tools.hideKeyboard' : 'terminal.tools.showKeyboard')}
          aria-pressed={keyboardFocusLocked || keyboardVisible}
          className={`${cls} ${keyboardButtonClass}`}
          title={t(keyboardFocusLocked
            ? 'terminal.tools.allowKeyboardPopup'
            : keyboardVisible ? 'terminal.tools.hideKeyboard' : 'terminal.tools.showKeyboard')}
          onPointerDown={(e) => {
            e.preventDefault()
            clearKeyboardLongPress()
            keyboardLongPressTriggeredRef.current = false
            keyboardLongPressTimerRef.current = setTimeout(() => {
              keyboardLongPressTimerRef.current = null
              keyboardLongPressTriggeredRef.current = true
              hapticImpact()
              onToggleKeyboardFocusLock?.()
            }, keyboardLongPressMs)
          }}
          onPointerUp={clearKeyboardLongPress}
          onPointerCancel={clearKeyboardLongPress}
          onPointerLeave={clearKeyboardLongPress}
          onClick={() => {
            if (keyboardLongPressTriggeredRef.current) {
              keyboardLongPressTriggeredRef.current = false
              return
            }
            hapticSelection()
            if (keyboardFocusLocked) {
              onToggleKeyboardFocusLock?.()
              onFocusKeyboard?.()
            } else if (keyboardVisible) onBlurKeyboard?.()
            else onFocusKeyboard?.()
          }}
        >
          <Keyboard aria-hidden="true" className="h-3.5 w-3.5" />
        </Button>
      </div>
      <div className="anytty-terminal-key-row grid min-w-0 grid-cols-10 gap-0.5 px-0.5 pb-0.5 pt-px">
        {btn('Tab', '\t', t('terminal.tools.tabKey'))}
        {modBtn('Ctrl', 'ctrl', activeModifierState.ctrl)}
        {modBtn('Alt', 'alt', activeModifierState.alt)}
        {btn('Del', '\x1b[3~')}
        {btn('\\', '\\')}
        {btn('←', '\x1b[D')}
        {btn('↓', '\x1b[B')}
        {btn('→', '\x1b[C')}
        {btn('PgD', '\x1b[6~')}
        <Button variant="ghost"
          data-key-id="Fn"
          type="button"
          aria-label={t('terminal.tools.toggleFn')}
          aria-pressed={fnOpen}
          className={`${cls} ${fnOpen ? 'bg-[var(--anytty-accent)] text-[var(--anytty-accent-text)]' : 'bg-transparent text-[var(--anytty-muted)] active:bg-[var(--anytty-surface-raised)]'}`}
          onPointerDown={(e) => e.preventDefault()}
          onClick={() => { hapticSelection(); onToggleFn?.() }}
        >
          Fn
        </Button>
      </div>
    </div>
  )
})
