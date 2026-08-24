import { act, cleanup, render, screen } from '@testing-library/react'
import { useRef } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { dispatchNativeKeyboardEvent } from '../platform/nativeKeyboard'
import type { TerminalHandle } from './Terminal'
import { useTerminalKeyboard, type TerminalKeyboardLayoutMode } from './useTerminalKeyboard'

const originalInnerHeight = window.innerHeight
const originalVisualViewport = Object.getOwnPropertyDescriptor(window, 'visualViewport')

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
    if (originalVisualViewport) Object.defineProperty(window, 'visualViewport', originalVisualViewport)
    else Reflect.deleteProperty(window, 'visualViewport')
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

  it('uses the visual viewport bottom when iOS pans the viewport above the keyboard', () => {
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 800 })
    setVisualViewport({ height: 450, offsetTop: 50 })
    render(<KeyboardHarness mode="resize" adjustInputPosition={vi.fn()} />)

    act(() => dispatchNativeKeyboardEvent({ visible: true }))

    expect(screen.getByTestId('container').style.height).toBe('500px')
  })

  it('does not shrink the workspace for an iPad floating keyboard', () => {
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 800 })
    setVisualViewport({ height: 800, offsetTop: 0 })
    render(<KeyboardHarness mode="resize" adjustInputPosition={vi.fn()} />)

    act(() => dispatchNativeKeyboardEvent({
      visible: true,
      keyboardHeight: 300,
      occludedHeight: 0,
    }))

    expect(screen.getByTestId('container').style.height).toBe('')
  })

  it('cancels delayed keyboard updates when the workspace unmounts', () => {
    vi.useFakeTimers()
    try {
      const view = render(<KeyboardHarness mode="resize" adjustInputPosition={vi.fn()} />)
      act(() => {
        document.dispatchEvent(new FocusEvent('focusin'))
        document.dispatchEvent(new FocusEvent('focusout'))
        document.dispatchEvent(new Event('anytty:resume'))
      })

      view.unmount()

      expect(vi.getTimerCount()).toBe(0)
    } finally {
      vi.useRealTimers()
    }
  })
})

function setVisualViewport(input: { height: number; offsetTop: number }): void {
  Object.defineProperty(window, 'visualViewport', {
    configurable: true,
    value: {
      height: input.height,
      offsetTop: input.offsetTop,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    },
  })
}
