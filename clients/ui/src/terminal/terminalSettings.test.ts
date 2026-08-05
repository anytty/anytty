import { describe, expect, it } from 'vitest'
import { DEFAULT_TERMINAL_SETTINGS, normalizeTerminalSettings, resolveTerminalTheme, TERMINAL_FONT_OPTIONS, terminalThemeCssVariables } from './terminalSettings'

describe('terminal theme UI variables', () => {
  it('keeps terminal chrome colors coordinated with the selected theme', () => {
    expect(terminalThemeCssVariables('tokyo-night')).toMatchObject({
      '--anytty-bg': '#030712',
      '--anytty-surface': '#1f2937',
      '--anytty-surface-raised': '#111827',
      '--anytty-border': '#374151',
      '--anytty-border-subtle': 'rgba(255, 255, 255, 0.06)',
      '--anytty-terminal-bg': '#1a1b26',
      '--anytty-terminal-fg': '#a9b1d6',
    })

    expect(terminalThemeCssVariables('github-light')).toMatchObject({
      '--anytty-bg': '#f6f8fa',
      '--anytty-surface': '#ffffff',
      '--anytty-surface-raised': '#f6f8fa',
      '--anytty-border': '#d0d7de',
      '--anytty-border-subtle': 'rgba(0, 0, 0, 0.08)',
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

  it('offers only the bundled JetBrains Mono font and drops stale font settings', () => {
    expect(TERMINAL_FONT_OPTIONS).toEqual([
      { label: 'JetBrains Mono NF', value: '"JetBrainsMono NF", monospace' },
    ])
    expect(normalizeTerminalSettings({ fontFamily: '"FiraCode NF", monospace' }).fontFamily)
      .toBe(DEFAULT_TERMINAL_SETTINGS.fontFamily)
  })
})
