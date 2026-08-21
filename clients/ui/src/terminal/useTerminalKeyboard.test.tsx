import { act, cleanup, render, screen } from '@testing-library/react'
import { useRef } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { dispatchNativeKeyboardEvent } from '../platform/nativeKeyboard'
import type { TerminalHandle } from './Terminal'
import { useTerminalKeyboard, type TerminalKeyboardLayoutMode } from './useTerminalKeyboard'

const originalInnerHeight = window.innerHeight

function KeyboardHarness({ mode, adjustInputPosition, cursorInfo = null }: {
  mode: TerminalKeyboardLayoutMode
  adjustInputPosition: (offset: number) => void
  cursorInfo?: { cursorY: number; rows: number; lineHeight: number } | null
}) {
  const containerRef = useRef<HTMLDivElement>(null)
  const mainRef = useRef<HTMLDivElement>(null)
  const wrapperRef = useRef<HTMLDivElement>(null)
  useTerminalKeyboard({
    containerRef,
    mainRef,
    termWrapperRef: wrapperRef,
    getLayoutMode: () => mode,
    getTermRef: () => ({ adjustInputPosition, getCursorInfo: () => cursorInfo }) as TerminalHandle,
  })
  return (
    <div ref={containerRef} data-testid="container">
      <div ref={mainRef} data-testid="main">
        <div ref={wrapperRef} data-testid="wrapper" />
      </div>
    </div>
  )
}

describe('useTerminalKeyboard', () => {
  afterEach(() => {
    act(() => dispatchNativeKeyboardEvent({ visible: false, keyboardHeight: 0 }))
    cleanup()
    vi.restoreAllMocks()
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: originalInnerHeight })
  })

  it('shrinks the app container and terminal in resize mode', () => {
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 800 })
    render(<KeyboardHarness mode="resize" adjustInputPosition={vi.fn()} />)

    act(() => dispatchNativeKeyboardEvent({ visible: true, keyboardHeight: 300 }))

    expect(screen.getByTestId('container').style.height).toBe('500px')
    expect(screen.getByTestId('wrapper').style.height).toBe('')
    expect(screen.getByTestId('wrapper').style.transform).toBe('')
  })

  it('keeps terminal history height and shifts the cursor above the keyboard in shift mode', () => {
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 800 })
    vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockReturnValue(800)
    const adjustInputPosition = vi.fn()
    render(<KeyboardHarness
      mode="shift"
      adjustInputPosition={adjustInputPosition}
      cursorInfo={{ cursorY: 35, rows: 40, lineHeight: 20 }}
    />)

    act(() => dispatchNativeKeyboardEvent({ visible: true, keyboardHeight: 300 }))

    expect(screen.getByTestId('container').style.height).toBe('500px')
    expect(screen.getByTestId('wrapper').style.height).toBe('800px')
    expect(screen.getByTestId('wrapper').style.transform).toBe('translateY(-240px)')
    expect(adjustInputPosition).toHaveBeenLastCalledWith(60)
  })
})
