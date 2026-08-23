// @vitest-environment jsdom

import { cleanup, render, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useNativeStatusBarSync } from './nativeStatusBar'

const nativeMocks = vi.hoisted(() => ({
  platform: 'ios',
  setBackgroundColor: vi.fn(async () => undefined),
  setKeyboardStyle: vi.fn(async () => undefined),
  setOverlaysWebView: vi.fn(async () => undefined),
  setStatusStyle: vi.fn(async () => undefined),
}))

vi.mock('@capacitor/core', () => ({
  Capacitor: {
    getPlatform: () => nativeMocks.platform,
    isNativePlatform: () => true,
  },
}))

vi.mock('@capacitor/keyboard', () => ({
  Keyboard: { setStyle: nativeMocks.setKeyboardStyle },
  KeyboardStyle: { Dark: 'DARK', Light: 'LIGHT' },
}))

vi.mock('@capacitor/status-bar', () => ({
  StatusBar: {
    setBackgroundColor: nativeMocks.setBackgroundColor,
    setOverlaysWebView: nativeMocks.setOverlaysWebView,
    setStyle: nativeMocks.setStatusStyle,
  },
  Style: { Dark: 'DARK', Light: 'LIGHT' },
}))

function NativeAppearanceHarness() {
  useNativeStatusBarSync()
  return null
}

describe('native iOS appearance sync', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
    nativeMocks.platform = 'ios'
    document.body.removeAttribute('style')
    Reflect.deleteProperty(document, 'elementFromPoint')
  })

  it('matches the iOS status bar and software keyboard to a dark app surface', async () => {
    document.body.style.backgroundColor = 'rgb(9, 9, 11)'
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: () => document.body,
    })

    render(<NativeAppearanceHarness />)

    await waitFor(() => {
      expect(nativeMocks.setStatusStyle).toHaveBeenCalledWith({ style: 'DARK' })
      expect(nativeMocks.setBackgroundColor).toHaveBeenCalledWith({ color: '#09090b' })
      expect(nativeMocks.setKeyboardStyle).toHaveBeenCalledWith({ style: 'DARK' })
      expect(nativeMocks.setOverlaysWebView).toHaveBeenCalledWith({ overlay: false })
    })
  })

  it('does not call the iOS-only keyboard appearance API on Android', async () => {
    nativeMocks.platform = 'android'
    document.body.style.backgroundColor = 'rgb(250, 250, 250)'
    Object.defineProperty(document, 'elementFromPoint', {
      configurable: true,
      value: () => document.body,
    })

    render(<NativeAppearanceHarness />)

    await waitFor(() => expect(nativeMocks.setStatusStyle).toHaveBeenCalledWith({ style: 'LIGHT' }))
    expect(nativeMocks.setKeyboardStyle).not.toHaveBeenCalled()
  })
})
