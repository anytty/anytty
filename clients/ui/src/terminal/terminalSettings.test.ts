import { describe, expect, it } from 'vitest'
import { DEFAULT_TERMINAL_SETTINGS, normalizeTerminalSettings, resolveTerminalTheme, resolveTerminalThemeUi, TERMINAL_FONT_OPTIONS, TERMINAL_SCROLL_INERTIA_OPTIONS, terminalThemeCssVariables } from './terminalSettings'

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

  it('normalizes scroll inertia to a supported preset', () => {
    expect(DEFAULT_TERMINAL_SETTINGS.scrollInertia).toBe('medium')
    expect(normalizeTerminalSettings({ scrollInertia: 'long' }).scrollInertia).toBe('long')
    expect(normalizeTerminalSettings({ scrollInertia: 'missing' }).scrollInertia).toBe('medium')
    expect(TERMINAL_SCROLL_INERTIA_OPTIONS).toEqual(['off', 'short', 'medium', 'long'])
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
