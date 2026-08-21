import type { RemoteRuntimeStorage } from '../core/transport'

export type AppTheme = 'light' | 'dark'

export const APP_THEME_STORAGE_KEY = 'anytty.app.theme.v1'
export const DEFAULT_APP_THEME: AppTheme = 'dark'

type AppThemeStorage = Pick<Storage, 'getItem' | 'setItem'> | RemoteRuntimeStorage

const APP_THEME_CSS_VARIABLES: Record<AppTheme, Record<string, string>> = {
  light: {
    'color-scheme': 'light',
    '--background': '#fafafa',
    '--foreground': '#18181b',
    '--card': '#ffffff',
    '--card-foreground': '#18181b',
    '--popover': '#ffffff',
    '--popover-foreground': '#18181b',
    '--primary': '#18181b',
    '--primary-foreground': '#fafafa',
    '--secondary': '#f4f4f5',
    '--secondary-foreground': '#18181b',
    '--muted': '#f4f4f5',
    '--muted-foreground': '#71717a',
    '--ui-accent': '#f4f4f5',
    '--ui-accent-foreground': '#18181b',
    '--destructive': '#b91c1c',
    '--destructive-foreground': '#ffffff',
    '--border': '#e4e4e7',
    '--input': '#d4d4d8',
    '--ring': '#71717a',
    '--anytty-neutral-50': '250 250 250',
    '--anytty-neutral-100': '244 244 245',
    '--anytty-neutral-200': '228 228 231',
    '--anytty-neutral-300': '212 212 216',
    '--anytty-neutral-400': '161 161 170',
    '--anytty-neutral-500': '113 113 122',
    '--anytty-neutral-600': '82 82 91',
    '--anytty-neutral-700': '63 63 70',
    '--anytty-neutral-800': '39 39 42',
    '--anytty-neutral-900': '24 24 27',
    '--anytty-neutral-950': '9 9 11',
    '--anytty-app-bg': '#fafafa',
    '--anytty-app-surface': '#ffffff',
    '--anytty-app-surface-soft': '#f4f4f5',
    '--anytty-app-soft': '#f4f4f5',
    '--anytty-app-line': '#e4e4e7',
    '--anytty-app-line-strong': '#d4d4d8',
    '--anytty-app-text': '#18181b',
    '--anytty-app-muted': '#71717a',
    '--anytty-app-accent': '#18181b',
    '--anytty-app-accent-text': '#fafafa',
    '--anytty-app-success': '#15803d',
    '--anytty-app-danger': '#b91c1c',
    '--anytty-app-inverse': '#09090b',
    '--anytty-bg': '#fafafa',
    '--anytty-surface': '#ffffff',
    '--anytty-surface-raised': '#f4f4f5',
    '--anytty-border': '#e4e4e7',
    '--anytty-border-subtle': 'rgba(0, 0, 0, 0.08)',
    '--anytty-text': '#18181b',
    '--anytty-muted': '#71717a',
    '--anytty-faint': '#a1a1aa',
    '--anytty-accent': '#18181b',
    '--anytty-accent-text': '#fafafa',
    '--anytty-search-match': '#b45309',
    '--anytty-overlay': 'rgba(0, 0, 0, 0.48)',
  },
  dark: {
    'color-scheme': 'dark',
    '--background': '#09090b',
    '--foreground': '#fafafa',
    '--card': '#18181b',
    '--card-foreground': '#fafafa',
    '--popover': '#18181b',
    '--popover-foreground': '#fafafa',
    '--primary': '#fafafa',
    '--primary-foreground': '#18181b',
    '--secondary': '#27272a',
    '--secondary-foreground': '#fafafa',
    '--muted': '#27272a',
    '--muted-foreground': '#a1a1aa',
    '--ui-accent': '#27272a',
    '--ui-accent-foreground': '#fafafa',
    '--destructive': '#dc2626',
    '--destructive-foreground': '#ffffff',
    '--border': '#3f3f46',
    '--input': '#52525b',
    '--ring': '#d4d4d8',
    '--anytty-neutral-50': '39 39 42',
    '--anytty-neutral-100': '39 39 42',
    '--anytty-neutral-200': '63 63 70',
    '--anytty-neutral-300': '82 82 91',
    '--anytty-neutral-400': '113 113 122',
    '--anytty-neutral-500': '161 161 170',
    '--anytty-neutral-600': '212 212 216',
    '--anytty-neutral-700': '228 228 231',
    '--anytty-neutral-800': '244 244 245',
    '--anytty-neutral-900': '250 250 250',
    '--anytty-neutral-950': '250 250 250',
    '--anytty-app-bg': '#09090b',
    '--anytty-app-surface': '#18181b',
    '--anytty-app-surface-soft': '#27272a',
    '--anytty-app-soft': '#27272a',
    '--anytty-app-line': '#3f3f46',
    '--anytty-app-line-strong': '#52525b',
    '--anytty-app-text': '#fafafa',
    '--anytty-app-muted': '#a1a1aa',
    '--anytty-app-accent': '#fafafa',
    '--anytty-app-accent-text': '#18181b',
    '--anytty-app-success': '#4ade80',
    '--anytty-app-danger': '#f87171',
    '--anytty-app-inverse': '#fafafa',
    '--anytty-bg': '#09090b',
    '--anytty-surface': '#18181b',
    '--anytty-surface-raised': '#27272a',
    '--anytty-border': '#3f3f46',
    '--anytty-border-subtle': 'rgba(255, 255, 255, 0.08)',
    '--anytty-text': '#fafafa',
    '--anytty-muted': '#a1a1aa',
    '--anytty-faint': '#71717a',
    '--anytty-accent': '#fafafa',
    '--anytty-accent-text': '#18181b',
    '--anytty-search-match': '#fbbf24',
    '--anytty-overlay': 'rgba(0, 0, 0, 0.56)',
  },
}

export function normalizeAppTheme(value: unknown, fallback: AppTheme = DEFAULT_APP_THEME): AppTheme {
  return value === 'light' || value === 'dark' ? value : fallback
}

export function readAppTheme(
  storage: AppThemeStorage | undefined = browserStorage(),
  migrationFallback: AppTheme = DEFAULT_APP_THEME,
): AppTheme {
  let stored: string | null = null
  try {
    stored = storage?.getItem(APP_THEME_STORAGE_KEY) ?? null
  } catch {}
  if (stored === 'light' || stored === 'dark') return stored
  return writeAppTheme(migrationFallback, storage)
}

export function writeAppTheme(
  theme: AppTheme,
  storage: Pick<Storage, 'setItem'> | RemoteRuntimeStorage | undefined = browserStorage(),
): AppTheme {
  const normalized = normalizeAppTheme(theme)
  try {
    storage?.setItem(APP_THEME_STORAGE_KEY, normalized)
  } catch {}
  return normalized
}

export function appThemeCssVariables(theme: AppTheme | string | undefined): Record<string, string> {
  return APP_THEME_CSS_VARIABLES[normalizeAppTheme(theme)]
}

function browserStorage(): Storage | undefined {
  try {
    return globalThis.localStorage
  } catch {
    return undefined
  }
}
