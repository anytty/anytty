import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

describe('haptic', () => {
  const originalVibrate = navigator.vibrate

  beforeEach(() => {
    vi.resetModules()
    vi.useFakeTimers()
    vi.setSystemTime(1000)
  })

  afterEach(() => {
    Object.defineProperty(navigator, 'vibrate', {
      configurable: true,
      value: originalVibrate,
    })
    vi.useRealTimers()
  })

  it('uses the installed native impact handler before browser vibration', async () => {
    const vibrate = vi.fn()
    Object.defineProperty(navigator, 'vibrate', {
      configurable: true,
      value: vibrate,
    })
    const { haptic, setHapticImpactHandler } = await import('./haptics')
    const impact = vi.fn()

    setHapticImpactHandler(impact)
    haptic()

    expect(impact).toHaveBeenCalledWith(10)
    expect(vibrate).not.toHaveBeenCalled()
  })

  it('falls back to browser vibration when no native handler is installed', async () => {
    const vibrate = vi.fn()
    Object.defineProperty(navigator, 'vibrate', {
      configurable: true,
      value: vibrate,
    })
    const { haptic } = await import('./haptics')

    haptic([12, 30, 12])

    expect(vibrate).toHaveBeenCalledWith([12, 30, 12])
  })

  it('falls back to browser vibration when the native handler rejects', async () => {
    const vibrate = vi.fn()
    Object.defineProperty(navigator, 'vibrate', {
      configurable: true,
      value: vibrate,
    })
    const { haptic, setHapticImpactHandler } = await import('./haptics')

    setHapticImpactHandler(() => Promise.reject(new Error('native haptic unavailable')))
    haptic(25)
    await Promise.resolve()

    expect(vibrate).toHaveBeenCalledWith(25)
  })

  it('exposes semantic feedback patterns for key interaction types', async () => {
    const vibrate = vi.fn()
    Object.defineProperty(navigator, 'vibrate', {
      configurable: true,
      value: vibrate,
    })
    const { hapticSelection, hapticImpact, hapticSuccess, hapticError } = await import('./haptics')

    hapticSelection()
    vi.advanceTimersByTime(40)
    hapticImpact()
    vi.advanceTimersByTime(40)
    hapticSuccess()
    vi.advanceTimersByTime(40)
    hapticError()

    expect(vibrate).toHaveBeenNthCalledWith(1, 8)
    expect(vibrate).toHaveBeenNthCalledWith(2, 10)
    expect(vibrate).toHaveBeenNthCalledWith(3, 25)
    expect(vibrate).toHaveBeenNthCalledWith(4, [12, 30, 12])
  })
})
