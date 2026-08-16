import { describe, expect, it } from 'vitest'
import {
  APP_THEME_STORAGE_KEY,
  appThemeCssVariables,
  readAppTheme,
  writeAppTheme,
} from './appTheme'

describe('app theme', () => {
  it('provides independent light and dark shadcn semantics', () => {
    expect(appThemeCssVariables('light')).toMatchObject({
      'color-scheme': 'light',
      '--background': '#fafafa',
      '--card': '#ffffff',
      '--foreground': '#18181b',
      '--anytty-app-bg': '#fafafa',
      '--anytty-bg': '#fafafa',
    })
    expect(appThemeCssVariables('dark')).toMatchObject({
      'color-scheme': 'dark',
      '--background': '#09090b',
      '--card': '#18181b',
      '--foreground': '#fafafa',
      '--anytty-app-bg': '#09090b',
      '--anytty-bg': '#09090b',
    })
    expect(appThemeCssVariables('dark')).not.toHaveProperty('--anytty-terminal-bg')
  })

  it.each(['light', 'dark'] as const)('keeps %s interface text readable', (theme) => {
    const variables = appThemeCssVariables(theme)
    expect(contrastRatio(variables['--foreground']!, variables['--background']!)).toBeGreaterThanOrEqual(4.5)
    expect(contrastRatio(variables['--muted-foreground']!, variables['--card']!)).toBeGreaterThanOrEqual(4.5)
    expect(contrastRatio(variables['--primary-foreground']!, variables['--primary']!)).toBeGreaterThanOrEqual(4.5)
  })

  it('migrates once from the previous terminal light or dark appearance', () => {
    const storage = new MemoryStorage()
    expect(readAppTheme(storage, 'light')).toBe('light')
    expect(storage.getItem(APP_THEME_STORAGE_KEY)).toBe('light')
    expect(readAppTheme(storage, 'dark')).toBe('light')
  })

  it('persists an explicit interface theme', () => {
    const storage = new MemoryStorage()
    expect(writeAppTheme('dark', storage)).toBe('dark')
    expect(readAppTheme(storage, 'light')).toBe('dark')
  })
})

class MemoryStorage {
  private readonly values = new Map<string, string>()
  getItem(key: string): string | null { return this.values.get(key) ?? null }
  setItem(key: string, value: string): void { this.values.set(key, value) }
}

function contrastRatio(foreground: string, background: string): number {
  const luminance = (value: string) => {
    const channels = value.slice(1).match(/.{2}/g)!.map((channel) => Number.parseInt(channel, 16) / 255)
    const linear = channels.map((channel) => channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4)
    return 0.2126 * linear[0]! + 0.7152 * linear[1]! + 0.0722 * linear[2]!
  }
  const values = [luminance(foreground), luminance(background)].sort((left, right) => right - left)
  return (values[0]! + 0.05) / (values[1]! + 0.05)
}
