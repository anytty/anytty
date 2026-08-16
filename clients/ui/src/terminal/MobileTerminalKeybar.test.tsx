import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { anyttyI18n } from '../i18n'
import { MobileTerminalKeybar } from './MobileTerminalKeybar'

const haptics = vi.hoisted(() => ({
  impact: vi.fn(),
  selection: vi.fn(),
}))

vi.mock('../platform/haptics', () => ({
  hapticImpact: haptics.impact,
  hapticSelection: haptics.selection,
}))

const acceptInput = () => true

describe('MobileTerminalKeybar', () => {
  beforeEach(async () => {
    haptics.impact.mockReset()
    haptics.selection.mockReset()
    await anyttyI18n.changeLanguage('en')
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('cycles Ctrl and Alt through visible off, once, and locked states', async () => {
    const user = userEvent.setup()
    render(<MobileTerminalKeybar onInput={acceptInput} />)

    const ctrlOff = screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' })
    const altOff = screen.getByRole('button', { name: 'Alt modifier: off. Next: once' })
    expect(ctrlOff.getAttribute('data-state')).toBe('off')
    expect(altOff.getAttribute('data-state')).toBe('off')

    await user.click(ctrlOff)
    const ctrlOnce = screen.getByRole('button', { name: 'Ctrl modifier: once. Next: locked' })
    expect(ctrlOnce.getAttribute('data-state')).toBe('once')
    expect(ctrlOnce.textContent).toBe('Ctrl')

    await user.click(ctrlOnce)
    const ctrlLocked = screen.getByRole('button', { name: 'Ctrl modifier: locked. Next: off' })
    expect(ctrlLocked.getAttribute('data-state')).toBe('locked')
    expect(ctrlLocked.textContent).toBe('Ctrl')

    await user.click(ctrlLocked)
    expect(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' }).getAttribute('data-state')).toBe('off')
  })

  it('consumes a one-shot modifier only after an input target accepts the send', async () => {
    const user = userEvent.setup()
    const onInput = vi.fn()
      .mockReturnValueOnce(false)
      .mockReturnValueOnce(true)
      .mockReturnValueOnce(true)
    render(<MobileTerminalKeybar onInput={onInput} />)

    await user.click(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' }))
    await user.click(screen.getByRole('button', { name: '\\' }))
    expect(screen.getByRole('button', { name: 'Ctrl modifier: once. Next: locked' })).toBeTruthy()
    expect(haptics.selection).toHaveBeenCalledTimes(1)

    await user.click(screen.getByRole('button', { name: '\\' }))
    expect(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' })).toBeTruthy()

    await user.click(screen.getByRole('button', { name: '\\' }))
    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['\x1c', '\x1c', '\\'])
    expect(haptics.selection).toHaveBeenCalledTimes(3)
  })

  it('keeps locked Ctrl and Alt modifiers across accepted key sequences', async () => {
    const user = userEvent.setup()
    const onInput = vi.fn(acceptInput)
    render(<MobileTerminalKeybar onInput={onInput} />)

    await user.click(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' }))
    await user.click(screen.getByRole('button', { name: 'Ctrl modifier: once. Next: locked' }))
    await user.click(screen.getByRole('button', { name: 'Alt modifier: off. Next: once' }))
    await user.click(screen.getByRole('button', { name: 'Alt modifier: once. Next: locked' }))
    await user.click(screen.getByRole('button', { name: '\\' }))
    await user.click(screen.getByRole('button', { name: '\\' }))

    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['\x1b\x1c', '\x1b\x1c'])
    expect(screen.getByRole('button', { name: 'Ctrl modifier: locked. Next: off' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Alt modifier: locked. Next: off' })).toBeTruthy()
  })

  it('sends keybar navigation with combined xterm modifiers and consumes accepted once states', async () => {
    const user = userEvent.setup()
    const onInput = vi.fn(acceptInput)
    render(<MobileTerminalKeybar onInput={onInput} />)

    await user.click(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' }))
    await user.click(screen.getByRole('button', { name: 'Alt modifier: off. Next: once' }))
    await user.click(screen.getByRole('button', { name: '↑' }))

    expect(onInput).toHaveBeenCalledWith('\x1b[1;7A')
    expect(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Alt modifier: off. Next: once' })).toBeTruthy()
  })

  it('sends the added tilde and Delete shortcuts with terminal-compatible bytes', async () => {
    const user = userEvent.setup()
    const onInput = vi.fn(acceptInput)
    render(<MobileTerminalKeybar onInput={onInput} />)

    await user.click(screen.getByRole('button', { name: '~' }))
    await user.click(screen.getByRole('button', { name: 'Del' }))

    expect(onInput.mock.calls.map(([data]) => data)).toEqual(['~', '\x1b[3~'])
  })

  it('uses a text Tab key and keeps arrow keys in a standard cross layout', () => {
    render(<MobileTerminalKeybar onInput={acceptInput} />)

    const keybar = screen.getByTestId('anytty-mobile-keybar')
    const rows = keybar.querySelectorAll('.anytty-terminal-key-row')
    const firstRow = Array.from(rows[0]?.querySelectorAll('button') ?? [])
    const secondRow = Array.from(rows[1]?.querySelectorAll('button') ?? [])

    expect(screen.getByRole('button', { name: 'Tab key' }).textContent).toBe('Tab')
    expect(firstRow[6]?.textContent).toBe('↑')
    expect(secondRow.map((button) => button.textContent)).toEqual([
      'Tab', 'Ctrl', 'Alt', 'Del', '\\', '←', '↓', '→', 'PgD', 'Fn',
    ])
  })

  it('uses a tap to show or hide the keyboard and a long press to toggle focus lock', () => {
    vi.useFakeTimers()
    const onFocusKeyboard = vi.fn()
    const onBlurKeyboard = vi.fn()
    const onToggleKeyboardFocusLock = vi.fn()
    const view = render(
      <MobileTerminalKeybar
        onInput={acceptInput}
        onFocusKeyboard={onFocusKeyboard}
        onBlurKeyboard={onBlurKeyboard}
        onToggleKeyboardFocusLock={onToggleKeyboardFocusLock}
      />,
    )

    const keyboard = screen.getByRole('button', { name: 'Show system keyboard' })
    fireEvent.pointerDown(keyboard)
    fireEvent.pointerUp(keyboard)
    fireEvent.click(keyboard)
    expect(onFocusKeyboard).toHaveBeenCalledOnce()
    expect(onBlurKeyboard).not.toHaveBeenCalled()
    expect(onToggleKeyboardFocusLock).not.toHaveBeenCalled()

    fireEvent.pointerDown(keyboard)
    act(() => vi.advanceTimersByTime(400))
    fireEvent.pointerUp(keyboard)
    fireEvent.click(keyboard)
    expect(onToggleKeyboardFocusLock).toHaveBeenCalledOnce()
    expect(onFocusKeyboard).toHaveBeenCalledOnce()
    expect(haptics.impact).toHaveBeenCalledOnce()

    view.rerender(
      <MobileTerminalKeybar
        onInput={acceptInput}
        onFocusKeyboard={onFocusKeyboard}
        onBlurKeyboard={onBlurKeyboard}
        onToggleKeyboardFocusLock={onToggleKeyboardFocusLock}
        keyboardFocusLocked
      />,
    )

    const lockedKeyboard = screen.getByRole('button', { name: 'Allow the system keyboard to open' })
    expect(lockedKeyboard.getAttribute('aria-pressed')).toBe('true')
    expect(lockedKeyboard.getAttribute('title')).toBe('Allow the system keyboard to open')
    expect(lockedKeyboard.className).toContain('bg-amber-300')

    fireEvent.pointerDown(lockedKeyboard)
    fireEvent.pointerUp(lockedKeyboard)
    fireEvent.click(lockedKeyboard)
    expect(onToggleKeyboardFocusLock).toHaveBeenCalledTimes(2)
    expect(onFocusKeyboard).toHaveBeenCalledTimes(2)
  })

  it('continues sending terminal shortcuts while keyboard focus is locked', async () => {
    const user = userEvent.setup()
    const onInput = vi.fn(acceptInput)
    render(<MobileTerminalKeybar onInput={onInput} keyboardFocusLocked />)

    await user.click(screen.getByRole('button', { name: '\\' }))

    expect(onInput).toHaveBeenCalledWith('\\')
    expect(haptics.selection).toHaveBeenCalledOnce()
  })

  it('does not attach modifier actions to long press, double click, or swipe', () => {
    vi.useFakeTimers()
    const onInput = vi.fn(acceptInput)
    const onFocusKeyboard = vi.fn()
    const onToggleKeyboardFocusLock = vi.fn()
    render(
      <MobileTerminalKeybar
        onInput={onInput}
        onFocusKeyboard={onFocusKeyboard}
        onToggleKeyboardFocusLock={onToggleKeyboardFocusLock}
      />,
    )

    const ctrl = screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' })
    const keyboard = screen.getByRole('button', { name: 'Show system keyboard' })
    fireEvent.pointerDown(ctrl)
    fireEvent.pointerDown(keyboard)
    fireEvent.touchMove(screen.getByTestId('anytty-mobile-keybar'), {
      touches: [{ clientX: 20, clientY: 20 }],
    })
    expect(vi.getTimerCount()).toBe(1)
    act(() => vi.advanceTimersByTime(1_000))

    expect(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' })).toBeTruthy()
    expect(onInput).not.toHaveBeenCalled()
    expect(onFocusKeyboard).not.toHaveBeenCalled()
    expect(onToggleKeyboardFocusLock).toHaveBeenCalledOnce()
    expect(haptics.selection).not.toHaveBeenCalled()

    fireEvent.click(keyboard)
    fireEvent.click(keyboard)
    expect(onFocusKeyboard).toHaveBeenCalledOnce()
    expect(onToggleKeyboardFocusLock).toHaveBeenCalledOnce()
  })

  it('emits haptic feedback once for an accepted tap and never for pointer cancellation or failure', () => {
    const onInput = vi.fn()
      .mockReturnValueOnce(false)
      .mockReturnValueOnce(true)
    render(<MobileTerminalKeybar onInput={onInput} />)
    const slash = screen.getByRole('button', { name: '/' })

    fireEvent.pointerDown(slash)
    fireEvent.pointerCancel(slash)
    expect(haptics.selection).not.toHaveBeenCalled()

    fireEvent.click(slash)
    expect(haptics.selection).not.toHaveBeenCalled()

    fireEvent.pointerDown(slash)
    fireEvent.click(slash)
    expect(haptics.selection).toHaveBeenCalledOnce()
    expect(onInput).toHaveBeenCalledTimes(2)
  })

  it('localizes modifier states and the combined keyboard and focus-lock button', async () => {
    const view = render(
      <MobileTerminalKeybar
        onInput={acceptInput}
        modifierState={{ ctrl: 'off', alt: 'locked' }}
        keyboardVisible
      />,
    )

    expect(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Alt modifier: locked. Next: off' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Hide system keyboard' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Prevent the system keyboard from opening' })).toBeNull()

    await act(async () => {
      await anyttyI18n.changeLanguage('zh-CN')
    })
    view.rerender(
      <MobileTerminalKeybar
        onInput={acceptInput}
        modifierState={{ ctrl: 'once', alt: 'locked' }}
        keyboardFocusLocked
      />,
    )

    expect(screen.getByRole('button', { name: 'Ctrl 修饰键：单次。下一状态：锁定' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Alt 修饰键：锁定。下一状态：关闭' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '允许系统键盘弹出' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Ctrl 修饰键：单次。下一状态：锁定' }).textContent).toBe('Ctrl')
    expect(screen.getByRole('button', { name: 'Alt 修饰键：锁定。下一状态：关闭' }).textContent).toBe('Alt')
  })

  it('keeps every shortcut in two viewport-width rows', async () => {
    const user = userEvent.setup()
    render(<MobileTerminalKeybar onInput={acceptInput} />)

    const keybar = screen.getByTestId('anytty-mobile-keybar')
    const buttons = Array.from(keybar.querySelectorAll('button'))
    expect(buttons.length).toBeGreaterThan(0)
    for (const button of buttons) {
      expect(button.className).toContain('h-9')
      expect(button.className).toContain('w-full')
      expect(button.className).toContain('min-w-0')
      expect(button.className).toContain('rounded-md')
    }
    const keyboardVisibility = keybar.querySelector('[data-key-id="keyboard-visibility"]')
    const keyboardFocusLock = keybar.querySelector('[data-key-id="keyboard-focus-lock"]')
    expect(keyboardVisibility?.className).toContain('w-full')
    expect(keyboardFocusLock).toBeNull()
    expect(keybar.querySelectorAll('.grid-cols-10')).toHaveLength(2)
    expect(keybar.querySelectorAll('.grid-cols-9')).toHaveLength(0)
    expect(keybar.querySelectorAll('.overflow-x-auto')).toHaveLength(0)
    expect(keybar.querySelectorAll('.anytty-terminal-key-row button')).toHaveLength(20)
    expect(screen.getByRole('button', { name: '~' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Del' })).toBeTruthy()

    await user.click(screen.getByRole('button', { name: 'Ctrl modifier: off. Next: once' }))
    await user.click(screen.getByRole('button', { name: 'Ctrl modifier: once. Next: locked' }))
    const lockedCtrl = screen.getByRole('button', { name: 'Ctrl modifier: locked. Next: off' })
    expect(lockedCtrl.className).toContain('h-9')
    expect(lockedCtrl.className).toContain('w-full')
    expect(lockedCtrl.className).toContain('min-w-0')
  })
})
