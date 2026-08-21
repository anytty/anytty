import { describe, expect, it } from 'vitest'
import { DEFAULT_TERMINAL_SETTINGS, normalizeTerminalSettings, readTerminalKeyboardMode, resolveTerminalMomentumProfile, resolveTerminalTheme, resolveTerminalThemeUi, TERMINAL_FONT_OPTIONS, terminalThemeCssVariables, writeTerminalKeyboardMode } from './terminalSettings'

describe('terminal theme UI variables', () => {
  it('exposes a complete local UI palette without changing app theme variables', () => {
    expect(terminalThemeCssVariables('tokyo-night')).toMatchObject({
      'color-scheme': 'dark',
      '--background': '#030712',
      '--foreground': '#e5e7eb',
      '--primary': '#7aa2f7',
      '--anytty-bg': '#030712',
      '--anytty-surface': '#1f2937',
      '--anytty-text': '#e5e7eb',
      '--anytty-app-bg': '#030712',
      '--anytty-app-surface': '#1f2937',
      '--anytty-app-text': '#e5e7eb',
      '--anytty-terminal-bg': '#1a1b26',
      '--anytty-terminal-fg': '#a9b1d6',
      '--anytty-terminal-cursor': '#c0caf5',
    })

    expect(terminalThemeCssVariables('github-light')).toMatchObject({
      'color-scheme': 'light',
      '--background': '#f6f8fa',
      '--foreground': '#24292f',
      '--anytty-surface': '#ffffff',
      '--anytty-app-bg': '#f6f8fa',
      '--anytty-app-surface': '#ffffff',
      '--anytty-terminal-bg': '#ffffff',
      '--anytty-terminal-fg': '#24292f',
    })
  })

  it('preserves the matching xterm terminal palette for a selected theme', () => {
    expect(resolveTerminalTheme('tokyo-night')).toMatchObject({
      background: '#1a1b26',
      foreground: '#a9b1d6',
      cursor: '#c0caf5',
    })
  })

  it('keeps primary controls readable for both bright and dark theme accents', () => {
    expect(resolveTerminalThemeUi('nord-light').accentText).toBe('#0a0a0a')
    expect(resolveTerminalThemeUi('github-light').accentText).toBe('#ffffff')
  })

  it('normalizes scroll inertia to a 0-100 value and migrates presets', () => {
    expect(DEFAULT_TERMINAL_SETTINGS.scrollInertia).toBe(60)
    expect(normalizeTerminalSettings({ scrollInertia: 'off' }).scrollInertia).toBe(0)
    expect(normalizeTerminalSettings({ scrollInertia: 'short' }).scrollInertia).toBe(25)
    expect(normalizeTerminalSettings({ scrollInertia: 'medium' }).scrollInertia).toBe(60)
    expect(normalizeTerminalSettings({ scrollInertia: 'long' }).scrollInertia).toBe(100)
    expect(normalizeTerminalSettings({ scrollInertia: 48.6 }).scrollInertia).toBe(49)
    expect(normalizeTerminalSettings({ scrollInertia: -1 }).scrollInertia).toBe(0)
    expect(normalizeTerminalSettings({ scrollInertia: 101 }).scrollInertia).toBe(100)
    expect(normalizeTerminalSettings({ scrollInertia: 'missing' }).scrollInertia).toBe(60)
  })

  it('supports resize, shift, and TUI-aware keyboard modes while migrating old names', () => {
    expect(normalizeTerminalSettings({ keyboardMode: 'resize' }).keyboardMode).toBe('resize')
    expect(normalizeTerminalSettings({ keyboardMode: 'overlay' }).keyboardMode).toBe('shift')
    expect(normalizeTerminalSettings({ keyboardMode: 'cover' }).keyboardMode).toBe('shift')
    expect(normalizeTerminalSettings({ keyboardMode: 'auto' }).keyboardMode).toBe('auto')
    expect(normalizeTerminalSettings({ keyboardMode: 'shift' }).keyboardMode).toBe('shift')
  })

  it('persists keyboard mode independently for each terminal', () => {
    const values = new Map<string, string>()
    const storage = {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => { values.set(key, value) },
    }

    writeTerminalKeyboardMode('studio', 'shell', 'shift', storage)
    writeTerminalKeyboardMode('studio', 'editor', 'resize', storage)
    writeTerminalKeyboardMode('server', 'shell', 'auto', storage)

    expect(readTerminalKeyboardMode('studio', 'shell', storage)).toBe('shift')
    expect(readTerminalKeyboardMode('studio', 'editor', storage)).toBe('resize')
    expect(readTerminalKeyboardMode('server', 'shell', storage)).toBe('auto')
    expect(readTerminalKeyboardMode('server', 'editor', storage)).toBeUndefined()

    values.set('anytty.terminal.keyboard-mode.v1:legacy:shell', 'overlay')
    expect(readTerminalKeyboardMode('legacy', 'shell', storage)).toBe('shift')
  })

  it('interpolates momentum continuously between migrated preset values', () => {
    expect(resolveTerminalMomentumProfile(0)).toEqual({ enabled: false, deceleration: 0, minimumVelocity: 0 })
    expect(resolveTerminalMomentumProfile(25)).toEqual({ enabled: true, deceleration: 0.95, minimumVelocity: 60 })
    expect(resolveTerminalMomentumProfile(60)).toEqual({ enabled: true, deceleration: 0.985, minimumVelocity: 20 })
    expect(resolveTerminalMomentumProfile(100)).toEqual({ enabled: true, deceleration: 0.99, minimumVelocity: 10 })
    expect(resolveTerminalMomentumProfile(50).deceleration).toBeGreaterThan(resolveTerminalMomentumProfile(49).deceleration)
  })

  it('offers bundled and platform terminal fonts while dropping stale settings', () => {
    expect(TERMINAL_FONT_OPTIONS.map((option) => option.label)).toEqual([
      'JetBrains Mono NF',
      'Fira Code NF',
      'Cascadia Code NF',
      'Hack NF',
      'Iosevka NF',
      'System Mono',
    ])
    const firaCode = TERMINAL_FONT_OPTIONS.find((option) => option.label === 'Fira Code NF')!
    expect(normalizeTerminalSettings({ fontFamily: firaCode.value }).fontFamily).toBe(firaCode.value)
    expect(normalizeTerminalSettings({ fontFamily: '"Hack NF", "JetBrainsMono NF", monospace' }).fontFamily)
      .toBe(TERMINAL_FONT_OPTIONS.find((option) => option.label === 'Hack NF')!.value)
    expect(normalizeTerminalSettings({ fontFamily: '"Unbundled NF", monospace' }).fontFamily)
      .toBe(DEFAULT_TERMINAL_SETTINGS.fontFamily)
  })
})
