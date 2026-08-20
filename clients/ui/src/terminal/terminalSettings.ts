import type { ITheme } from '@xterm/xterm'
import type { TerminalRenderer } from './Terminal'
import type { RemoteRuntimeStorage } from '../core/transport'

export type TerminalKeyboardMode = 'auto' | 'resize' | 'shift'
export type TerminalScrollInertia = 'off' | 'short' | 'medium' | 'long'
export type TerminalThemeGroup = 'dark' | 'light'

export const TERMINAL_SCROLL_INERTIA_OPTIONS = ['off', 'short', 'medium', 'long'] as const satisfies readonly TerminalScrollInertia[]

export interface TerminalMomentumProfile {
  enabled: boolean
  deceleration: number
  minimumVelocity: number
}

export function resolveTerminalMomentumProfile(inertia: TerminalScrollInertia | undefined): TerminalMomentumProfile {
  switch (inertia) {
    case 'off':
      return { enabled: false, deceleration: 0, minimumVelocity: 0 }
    case 'short':
      return { enabled: true, deceleration: 0.95, minimumVelocity: 60 }
    case 'long':
      return { enabled: true, deceleration: 0.99, minimumVelocity: 10 }
    default:
      return { enabled: true, deceleration: 0.985, minimumVelocity: 20 }
  }
}

export interface TerminalThemeUi {
  page: string
  surface: string
  surfaceRaised: string
  border: string
  borderSubtle: string
  text: string
  muted: string
  faint: string
  accent: string
  accentText: string
  terminalBackground: string
  terminalForeground: string
  terminalCursor: string
  overlay: string
  scrollbar: string
  scrollbarActive: string
}

export interface TerminalFontOption {
  label: string
  value: string
}

export interface TerminalSettings {
  fontSize: number
  fontFamily: string
  themeId: TerminalThemeId
  renderer: TerminalRenderer
  keyboardMode: TerminalKeyboardMode
  scrollInertia: TerminalScrollInertia
  scrollback: number
  scrollbackPrefetchThresholdRows: number
  cursorBlink: boolean
  autoAcquireResizeOwner: boolean
}

export const TERMINAL_SETTINGS_STORAGE_KEY = 'anytty.terminal.settings.v1'

export const ANYTTY_DARK_TERMINAL_THEME: ITheme = {
  background: '#0c0c0c',
  foreground: '#f4f4f5',
  cursor: '#d4d4d8',
  selectionBackground: '#3f3f46',
  black: '#18181b',
  red: '#ef4444',
  green: '#22c55e',
  yellow: '#eab308',
  blue: '#3b82f6',
  magenta: '#d946ef',
  cyan: '#06b6d4',
  white: '#f4f4f5',
  brightBlack: '#71717a',
  brightRed: '#f87171',
  brightGreen: '#4ade80',
  brightYellow: '#fde047',
  brightBlue: '#60a5fa',
  brightMagenta: '#e879f9',
  brightCyan: '#22d3ee',
  brightWhite: '#fafafa',
}

interface TerminalThemeOptionDefinition {
  id: string
  label: string
  group: TerminalThemeGroup
  theme: ITheme
}

interface TerminalThemeUiPreset {
  page: string
  surface: string
  surfaceRaised: string
  overlay: string
  border: string
  borderSubtle: string
  text: string
  muted: string
  faint: string
}

export const TERMINAL_THEME_OPTIONS = [
  {
    id: 'anytty-dark',
    label: 'AnyTTY Dark',
    group: 'dark',
    theme: ANYTTY_DARK_TERMINAL_THEME,
  },
  {
    id: 'tokyo-night',
    label: 'Tokyo Night',
    group: 'dark',
    theme: {
      background: '#1a1b26',
      foreground: '#a9b1d6',
      cursor: '#c0caf5',
      selectionBackground: '#33467c',
      black: '#15161e',
      red: '#f7768e',
      green: '#9ece6a',
      yellow: '#e0af68',
      blue: '#7aa2f7',
      magenta: '#bb9af7',
      cyan: '#7dcfff',
      white: '#a9b1d6',
      brightBlack: '#414868',
      brightRed: '#f7768e',
      brightGreen: '#9ece6a',
      brightYellow: '#e0af68',
      brightBlue: '#7aa2f7',
      brightMagenta: '#bb9af7',
      brightCyan: '#7dcfff',
      brightWhite: '#c0caf5',
    },
  },
  {
    id: 'dracula',
    label: 'Dracula',
    group: 'dark',
    theme: {
      background: '#282a36',
      foreground: '#f8f8f2',
      cursor: '#f8f8f2',
      selectionBackground: '#44475a',
      black: '#21222c',
      red: '#ff5555',
      green: '#50fa7b',
      yellow: '#f1fa8c',
      blue: '#bd93f9',
      magenta: '#ff79c6',
      cyan: '#8be9fd',
      white: '#f8f8f2',
      brightBlack: '#6272a4',
      brightRed: '#ff6e6e',
      brightGreen: '#69ff94',
      brightYellow: '#ffffa5',
      brightBlue: '#d6acff',
      brightMagenta: '#ff92df',
      brightCyan: '#a4ffff',
      brightWhite: '#ffffff',
    },
  },
  {
    id: 'one-dark',
    label: 'One Dark',
    group: 'dark',
    theme: {
      background: '#282c34',
      foreground: '#abb2bf',
      cursor: '#528bff',
      selectionBackground: '#3e4451',
      black: '#1e2127',
      red: '#e06c75',
      green: '#98c379',
      yellow: '#d19a66',
      blue: '#61afef',
      magenta: '#c678dd',
      cyan: '#56b6c2',
      white: '#abb2bf',
      brightBlack: '#5c6370',
      brightRed: '#e06c75',
      brightGreen: '#98c379',
      brightYellow: '#d19a66',
      brightBlue: '#61afef',
      brightMagenta: '#c678dd',
      brightCyan: '#56b6c2',
      brightWhite: '#ffffff',
    },
  },
  {
    id: 'catppuccin-mocha',
    label: 'Catppuccin Mocha',
    group: 'dark',
    theme: {
      background: '#1e1e2e',
      foreground: '#cdd6f4',
      cursor: '#f5e0dc',
      selectionBackground: '#45475a',
      black: '#45475a',
      red: '#f38ba8',
      green: '#a6e3a1',
      yellow: '#f9e2af',
      blue: '#89b4fa',
      magenta: '#f5c2e7',
      cyan: '#94e2d5',
      white: '#bac2de',
      brightBlack: '#585b70',
      brightRed: '#f38ba8',
      brightGreen: '#a6e3a1',
      brightYellow: '#f9e2af',
      brightBlue: '#89b4fa',
      brightMagenta: '#f5c2e7',
      brightCyan: '#94e2d5',
      brightWhite: '#a6adc8',
    },
  },
  {
    id: 'solarized-dark',
    label: 'Solarized Dark',
    group: 'dark',
    theme: {
      background: '#002b36',
      foreground: '#839496',
      cursor: '#839496',
      selectionBackground: '#073642',
      black: '#073642',
      red: '#dc322f',
      green: '#859900',
      yellow: '#b58900',
      blue: '#268bd2',
      magenta: '#d33682',
      cyan: '#2aa198',
      white: '#eee8d5',
      brightBlack: '#586e75',
      brightRed: '#cb4b16',
      brightGreen: '#586e75',
      brightYellow: '#657b83',
      brightBlue: '#839496',
      brightMagenta: '#6c71c4',
      brightCyan: '#93a1a1',
      brightWhite: '#fdf6e3',
    },
  },
  {
    id: 'nord',
    label: 'Nord',
    group: 'dark',
    theme: {
      background: '#2e3440',
      foreground: '#d8dee9',
      cursor: '#d8dee9',
      selectionBackground: '#434c5e',
      black: '#3b4252',
      red: '#bf616a',
      green: '#a3be8c',
      yellow: '#ebcb8b',
      blue: '#81a1c1',
      magenta: '#b48ead',
      cyan: '#88c0d0',
      white: '#e5e9f0',
      brightBlack: '#4c566a',
      brightRed: '#bf616a',
      brightGreen: '#a3be8c',
      brightYellow: '#ebcb8b',
      brightBlue: '#81a1c1',
      brightMagenta: '#b48ead',
      brightCyan: '#8fbcbb',
      brightWhite: '#eceff4',
    },
  },
  {
    id: 'gruvbox-dark',
    label: 'Gruvbox Dark',
    group: 'dark',
    theme: {
      background: '#282828',
      foreground: '#ebdbb2',
      cursor: '#ebdbb2',
      selectionBackground: '#504945',
      black: '#282828',
      red: '#cc241d',
      green: '#98971a',
      yellow: '#d79921',
      blue: '#458588',
      magenta: '#b16286',
      cyan: '#689d6a',
      white: '#a89984',
      brightBlack: '#928374',
      brightRed: '#fb4934',
      brightGreen: '#b8bb26',
      brightYellow: '#fabd2f',
      brightBlue: '#83a598',
      brightMagenta: '#d3869b',
      brightCyan: '#8ec07c',
      brightWhite: '#ebdbb2',
    },
  },
  {
    id: 'github-dark',
    label: 'GitHub Dark',
    group: 'dark',
    theme: {
      background: '#0d1117',
      foreground: '#c9d1d9',
      cursor: '#58a6ff',
      selectionBackground: '#264f78',
      black: '#484f58',
      red: '#ff7b72',
      green: '#3fb950',
      yellow: '#d29922',
      blue: '#58a6ff',
      magenta: '#bc8cff',
      cyan: '#39c5cf',
      white: '#b1bac4',
      brightBlack: '#6e7681',
      brightRed: '#ffa198',
      brightGreen: '#56d364',
      brightYellow: '#e3b341',
      brightBlue: '#79c0ff',
      brightMagenta: '#d2a8ff',
      brightCyan: '#56d4dd',
      brightWhite: '#f0f6fc',
    },
  },
  {
    id: 'github-light',
    label: 'GitHub Light',
    group: 'light',
    theme: {
      background: '#ffffff',
      foreground: '#24292f',
      cursor: '#044289',
      selectionBackground: '#b6d4fe',
      black: '#24292f',
      red: '#cf222e',
      green: '#116329',
      yellow: '#4d2d00',
      blue: '#0969da',
      magenta: '#8250df',
      cyan: '#1b7c83',
      white: '#6e7781',
      brightBlack: '#57606a',
      brightRed: '#a40e26',
      brightGreen: '#1a7f37',
      brightYellow: '#633c01',
      brightBlue: '#218bff',
      brightMagenta: '#a475f9',
      brightCyan: '#3192aa',
      brightWhite: '#8c959f',
    },
  },
  {
    id: 'solarized-light',
    label: 'Solarized Light',
    group: 'light',
    theme: {
      background: '#fdf6e3',
      foreground: '#657b83',
      cursor: '#586e75',
      selectionBackground: '#eee8d5',
      black: '#073642',
      red: '#dc322f',
      green: '#859900',
      yellow: '#b58900',
      blue: '#268bd2',
      magenta: '#d33682',
      cyan: '#2aa198',
      white: '#eee8d5',
      brightBlack: '#002b36',
      brightRed: '#cb4b16',
      brightGreen: '#586e75',
      brightYellow: '#657b83',
      brightBlue: '#839496',
      brightMagenta: '#6c71c4',
      brightCyan: '#93a1a1',
      brightWhite: '#fdf6e3',
    },
  },
  {
    id: 'catppuccin-latte',
    label: 'Catppuccin Latte',
    group: 'light',
    theme: {
      background: '#eff1f5',
      foreground: '#4c4f69',
      cursor: '#dc8a78',
      selectionBackground: '#ccd0da',
      black: '#5c5f77',
      red: '#d20f39',
      green: '#40a02b',
      yellow: '#df8e1d',
      blue: '#1e66f5',
      magenta: '#ea76cb',
      cyan: '#179299',
      white: '#acb0be',
      brightBlack: '#6c6f85',
      brightRed: '#d20f39',
      brightGreen: '#40a02b',
      brightYellow: '#df8e1d',
      brightBlue: '#1e66f5',
      brightMagenta: '#ea76cb',
      brightCyan: '#179299',
      brightWhite: '#bcc0cc',
    },
  },
  {
    id: 'one-light',
    label: 'One Light',
    group: 'light',
    theme: {
      background: '#fafafa',
      foreground: '#383a42',
      cursor: '#526fff',
      selectionBackground: '#bfceff',
      black: '#383a42',
      red: '#e45649',
      green: '#50a14f',
      yellow: '#c18401',
      blue: '#4078f2',
      magenta: '#a626a4',
      cyan: '#0184bc',
      white: '#a0a1a7',
      brightBlack: '#696c77',
      brightRed: '#e45649',
      brightGreen: '#50a14f',
      brightYellow: '#c18401',
      brightBlue: '#4078f2',
      brightMagenta: '#a626a4',
      brightCyan: '#0184bc',
      brightWhite: '#fafafa',
    },
  },
  {
    id: 'nord-light',
    label: 'Nord Light',
    group: 'light',
    theme: {
      background: '#eceff4',
      foreground: '#2e3440',
      cursor: '#2e3440',
      selectionBackground: '#d8dee9',
      black: '#2e3440',
      red: '#bf616a',
      green: '#a3be8c',
      yellow: '#ebcb8b',
      blue: '#81a1c1',
      magenta: '#b48ead',
      cyan: '#88c0d0',
      white: '#d8dee9',
      brightBlack: '#4c566a',
      brightRed: '#bf616a',
      brightGreen: '#a3be8c',
      brightYellow: '#ebcb8b',
      brightBlue: '#81a1c1',
      brightMagenta: '#b48ead',
      brightCyan: '#8fbcbb',
      brightWhite: '#eceff4',
    },
  },
  {
    id: 'gruvbox-light',
    label: 'Gruvbox Light',
    group: 'light',
    theme: {
      background: '#fbf1c7',
      foreground: '#3c3836',
      cursor: '#3c3836',
      selectionBackground: '#d5c4a1',
      black: '#3c3836',
      red: '#cc241d',
      green: '#98971a',
      yellow: '#d79921',
      blue: '#458588',
      magenta: '#b16286',
      cyan: '#689d6a',
      white: '#7c6f64',
      brightBlack: '#928374',
      brightRed: '#9d0006',
      brightGreen: '#79740e',
      brightYellow: '#b57614',
      brightBlue: '#076678',
      brightMagenta: '#8f3f71',
      brightCyan: '#427b58',
      brightWhite: '#3c3836',
    },
  },
  {
    id: 'rose-pine-dawn',
    label: 'Rose Pine Dawn',
    group: 'light',
    theme: {
      background: '#faf4ed',
      foreground: '#575279',
      cursor: '#575279',
      selectionBackground: '#dfdad9',
      black: '#575279',
      red: '#b4637a',
      green: '#286983',
      yellow: '#ea9d34',
      blue: '#56949f',
      magenta: '#907aa9',
      cyan: '#d7827e',
      white: '#f2e9e1',
      brightBlack: '#797593',
      brightRed: '#b4637a',
      brightGreen: '#286983',
      brightYellow: '#ea9d34',
      brightBlue: '#56949f',
      brightMagenta: '#907aa9',
      brightCyan: '#d7827e',
      brightWhite: '#faf4ed',
    },
  },
  {
    id: 'everforest-light',
    label: 'Everforest Light',
    group: 'light',
    theme: {
      background: '#f3ead3',
      foreground: '#5c6a72',
      cursor: '#5c6a72',
      selectionBackground: '#e0dcc7',
      black: '#5c6a72',
      red: '#f85552',
      green: '#8da101',
      yellow: '#dfa000',
      blue: '#3a94c5',
      magenta: '#df69ba',
      cyan: '#35a77c',
      white: '#dfddc8',
      brightBlack: '#829181',
      brightRed: '#f85552',
      brightGreen: '#8da101',
      brightYellow: '#dfa000',
      brightBlue: '#3a94c5',
      brightMagenta: '#df69ba',
      brightCyan: '#35a77c',
      brightWhite: '#f3ead3',
    },
  },
] as const satisfies readonly TerminalThemeOptionDefinition[]

export type TerminalThemeOption = (typeof TERMINAL_THEME_OPTIONS)[number]
export type TerminalThemeId = TerminalThemeOption['id']

const TERMINAL_THEME_UI_PRESETS: Record<TerminalThemeId, TerminalThemeUiPreset> = {
  'anytty-dark': {
    page: '#030712',
    surface: '#0c0c0c',
    surfaceRaised: '#18181b',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#27272a',
    borderSubtle: 'rgba(244, 244, 245, 0.08)',
    text: '#f4f4f5',
    muted: '#a1a1aa',
    faint: '#71717a',
  },
  'tokyo-night': {
    page: '#030712',
    surface: '#1f2937',
    surfaceRaised: '#111827',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#374151',
    borderSubtle: 'rgba(255, 255, 255, 0.06)',
    text: '#e5e7eb',
    muted: '#9ca3af',
    faint: '#6b7280',
  },
  dracula: {
    page: '#1e1f29',
    surface: '#282a36',
    surfaceRaised: '#343746',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#44475a',
    borderSubtle: 'rgba(255, 255, 255, 0.08)',
    text: '#f8f8f2',
    muted: '#bfbfbf',
    faint: '#6272a4',
  },
  'one-dark': {
    page: '#1e2127',
    surface: '#282c34',
    surfaceRaised: '#2c313a',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#3e4451',
    borderSubtle: 'rgba(255, 255, 255, 0.07)',
    text: '#abb2bf',
    muted: '#8b929e',
    faint: '#5c6370',
  },
  'catppuccin-mocha': {
    page: '#11111b',
    surface: '#1e1e2e',
    surfaceRaised: '#313244',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#45475a',
    borderSubtle: 'rgba(205, 214, 244, 0.08)',
    text: '#cdd6f4',
    muted: '#a6adc8',
    faint: '#6c7086',
  },
  'solarized-dark': {
    page: '#001e27',
    surface: '#002b36',
    surfaceRaised: '#073642',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#073642',
    borderSubtle: 'rgba(131, 148, 150, 0.12)',
    text: '#839496',
    muted: '#657b83',
    faint: '#586e75',
  },
  nord: {
    page: '#242933',
    surface: '#2e3440',
    surfaceRaised: '#3b4252',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#3b4252',
    borderSubtle: 'rgba(216, 222, 233, 0.08)',
    text: '#d8dee9',
    muted: '#a5b1c2',
    faint: '#4c566a',
  },
  'gruvbox-dark': {
    page: '#1d2021',
    surface: '#282828',
    surfaceRaised: '#3c3836',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#3c3836',
    borderSubtle: 'rgba(235, 219, 178, 0.08)',
    text: '#ebdbb2',
    muted: '#a89984',
    faint: '#665c54',
  },
  'github-dark': {
    page: '#010409',
    surface: '#0d1117',
    surfaceRaised: '#161b22',
    overlay: 'rgba(0, 0, 0, 0.50)',
    border: '#30363d',
    borderSubtle: 'rgba(240, 246, 252, 0.06)',
    text: '#c9d1d9',
    muted: '#8b949e',
    faint: '#6e7681',
  },
  'github-light': {
    page: '#f6f8fa',
    surface: '#ffffff',
    surfaceRaised: '#f6f8fa',
    overlay: 'rgba(0, 0, 0, 0.30)',
    border: '#d0d7de',
    borderSubtle: 'rgba(0, 0, 0, 0.08)',
    text: '#24292f',
    muted: '#57606a',
    faint: '#8c959f',
  },
  'solarized-light': {
    page: '#eee8d5',
    surface: '#fdf6e3',
    surfaceRaised: '#eee8d5',
    overlay: 'rgba(0, 0, 0, 0.25)',
    border: '#d6cdb7',
    borderSubtle: 'rgba(0, 0, 0, 0.08)',
    text: '#586e75',
    muted: '#657b83',
    faint: '#93a1a1',
  },
  'catppuccin-latte': {
    page: '#e6e9ef',
    surface: '#eff1f5',
    surfaceRaised: '#e6e9ef',
    overlay: 'rgba(0, 0, 0, 0.25)',
    border: '#ccd0da',
    borderSubtle: 'rgba(0, 0, 0, 0.08)',
    text: '#4c4f69',
    muted: '#6c6f85',
    faint: '#9ca0b0',
  },
  'one-light': {
    page: '#e5e5e6',
    surface: '#fafafa',
    surfaceRaised: '#f0f0f1',
    overlay: 'rgba(0, 0, 0, 0.25)',
    border: '#d4d4d5',
    borderSubtle: 'rgba(0, 0, 0, 0.07)',
    text: '#383a42',
    muted: '#696c77',
    faint: '#a0a1a7',
  },
  'nord-light': {
    page: '#d8dee9',
    surface: '#eceff4',
    surfaceRaised: '#e5e9f0',
    overlay: 'rgba(0, 0, 0, 0.20)',
    border: '#c8ced8',
    borderSubtle: 'rgba(0, 0, 0, 0.08)',
    text: '#2e3440',
    muted: '#4c566a',
    faint: '#7b88a1',
  },
  'gruvbox-light': {
    page: '#ebdbb2',
    surface: '#fbf1c7',
    surfaceRaised: '#ebdbb2',
    overlay: 'rgba(0, 0, 0, 0.25)',
    border: '#d5c4a1',
    borderSubtle: 'rgba(0, 0, 0, 0.08)',
    text: '#3c3836',
    muted: '#504945',
    faint: '#928374',
  },
  'rose-pine-dawn': {
    page: '#f2e9e1',
    surface: '#faf4ed',
    surfaceRaised: '#f2e9e1',
    overlay: 'rgba(0, 0, 0, 0.20)',
    border: '#dfdad9',
    borderSubtle: 'rgba(0, 0, 0, 0.06)',
    text: '#575279',
    muted: '#797593',
    faint: '#9893a5',
  },
  'everforest-light': {
    page: '#e5dfc6',
    surface: '#f3ead3',
    surfaceRaised: '#e5dfc6',
    overlay: 'rgba(0, 0, 0, 0.20)',
    border: '#d4ccb0',
    borderSubtle: 'rgba(0, 0, 0, 0.07)',
    text: '#5c6a72',
    muted: '#708089',
    faint: '#939f91',
  },
}

export const TERMINAL_FONT_OPTIONS: TerminalFontOption[] = [
  { label: 'JetBrains Mono NF', value: '"JetBrainsMono NF", "Nerd Symbols Mono", monospace' },
  { label: 'Fira Code NF', value: '"FiraCode NF", "Nerd Symbols Mono", "JetBrainsMono NF", monospace' },
  { label: 'Cascadia Code NF', value: '"CascadiaCode NF", "Nerd Symbols Mono", "JetBrainsMono NF", monospace' },
  { label: 'Hack NF', value: '"Hack NF", "Nerd Symbols Mono", "JetBrainsMono NF", monospace' },
  { label: 'Iosevka NF', value: '"Iosevka NF", "Nerd Symbols Mono", "JetBrainsMono NF", monospace' },
  { label: 'System Mono', value: 'ui-monospace, "SFMono-Regular", Menlo, Monaco, Consolas, "Droid Sans Mono", "Nerd Symbols Mono", "JetBrainsMono NF", monospace' },
]

const LEGACY_TERMINAL_FONT_FAMILIES = new Map<string, string>([
  ['"JetBrainsMono NF", monospace', TERMINAL_FONT_OPTIONS[0]!.value],
  ['"FiraCode NF", "JetBrainsMono NF", monospace', TERMINAL_FONT_OPTIONS[1]!.value],
  ['"Hack NF", "JetBrainsMono NF", monospace', TERMINAL_FONT_OPTIONS[3]!.value],
  [
    'ui-monospace, "SFMono-Regular", Menlo, Monaco, Consolas, "Droid Sans Mono", "JetBrainsMono NF", monospace',
    TERMINAL_FONT_OPTIONS[5]!.value,
  ],
])

export const DEFAULT_TERMINAL_SETTINGS: TerminalSettings = {
  fontSize: 14,
  fontFamily: TERMINAL_FONT_OPTIONS[0]!.value,
  themeId: 'anytty-dark',
  renderer: 'auto',
  keyboardMode: 'auto',
  scrollInertia: 'medium',
  scrollback: 10000,
  scrollbackPrefetchThresholdRows: 30,
  cursorBlink: true,
  autoAcquireResizeOwner: false,
}

export function readTerminalSettings(storage: Pick<Storage, 'getItem'> | RemoteRuntimeStorage | undefined = browserStorage()): TerminalSettings {
  const migrated = migrateLegacyTerminalSettings(storage)
  const raw = readStorageValue(storage, TERMINAL_SETTINGS_STORAGE_KEY)
  if (!raw) return migrated
  try {
    return normalizeTerminalSettings({
      ...migrated,
      ...JSON.parse(raw),
    })
  } catch {
    return migrated
  }
}

export function writeTerminalSettings(
  settings: TerminalSettings,
  storage: Pick<Storage, 'setItem'> | RemoteRuntimeStorage | undefined = browserStorage(),
): TerminalSettings {
  const normalized = normalizeTerminalSettings(settings)
  try {
    storage?.setItem(TERMINAL_SETTINGS_STORAGE_KEY, JSON.stringify(normalized))
  } catch {}
  return normalized
}

export function resolveTerminalTheme(themeId: TerminalThemeId | string | undefined): ITheme {
  return TERMINAL_THEME_OPTIONS.find((option) => option.id === themeId)?.theme ?? ANYTTY_DARK_TERMINAL_THEME
}

export function resolveTerminalThemeOption(themeId: TerminalThemeId | string | undefined): TerminalThemeOption {
  return TERMINAL_THEME_OPTIONS.find((option) => option.id === themeId) ?? TERMINAL_THEME_OPTIONS[0]!
}

export function resolveTerminalThemeUi(themeId: TerminalThemeId | string | undefined): TerminalThemeUi {
  const option = resolveTerminalThemeOption(themeId)
  const theme = option.theme
  const preset = TERMINAL_THEME_UI_PRESETS[option.id]
  const background = colorValue(theme.background, ANYTTY_DARK_TERMINAL_THEME.background!)
  const foreground = colorValue(theme.foreground, ANYTTY_DARK_TERMINAL_THEME.foreground!)
  const cursor = colorValue(theme.cursor, foreground)
  const accent = colorValue(theme.blue ?? theme.cyan ?? theme.brightBlue, cursor)

  return {
    page: preset.page,
    surface: preset.surface,
    surfaceRaised: preset.surfaceRaised,
    border: preset.border,
    borderSubtle: preset.borderSubtle,
    text: preset.text,
    muted: preset.muted,
    faint: preset.faint,
    accent,
    accentText: contrastingTextColor(accent),
    terminalBackground: background,
    terminalForeground: foreground,
    terminalCursor: cursor,
    overlay: preset.overlay,
    scrollbar: withAlpha(preset.text, option.group === 'dark' ? 0.35 : 0.42),
    scrollbarActive: withAlpha(preset.text, option.group === 'dark' ? 0.58 : 0.68),
  }
}

export function terminalThemeCssVariables(themeId: TerminalThemeId | string | undefined): Record<string, string> {
  const option = resolveTerminalThemeOption(themeId)
  const ui = resolveTerminalThemeUi(themeId)
  const destructive = colorValue(option.theme.red ?? option.theme.brightRed, '#ef4444')
  return {
    'color-scheme': option.group,
    '--background': ui.page,
    '--foreground': ui.text,
    '--card': ui.surface,
    '--card-foreground': ui.text,
    '--popover': ui.surface,
    '--popover-foreground': ui.text,
    '--primary': ui.accent,
    '--primary-foreground': ui.accentText,
    '--secondary': ui.surfaceRaised,
    '--secondary-foreground': ui.text,
    '--muted': ui.surfaceRaised,
    '--muted-foreground': ui.muted,
    '--ui-accent': ui.surfaceRaised,
    '--ui-accent-foreground': ui.text,
    '--border': ui.border,
    '--input': ui.border,
    '--ring': ui.accent,
    '--destructive': destructive,
    '--destructive-foreground': contrastingTextColor(destructive),
    '--anytty-app-success': colorValue(option.theme.green ?? option.theme.brightGreen, '#22c55e'),
    '--anytty-bg': ui.page,
    '--anytty-surface': ui.surface,
    '--anytty-surface-raised': ui.surfaceRaised,
    '--anytty-border': ui.border,
    '--anytty-border-subtle': ui.borderSubtle,
    '--anytty-text': ui.text,
    '--anytty-muted': ui.muted,
    '--anytty-faint': ui.faint,
    '--anytty-accent': ui.accent,
    '--anytty-accent-text': ui.accentText,
    '--anytty-overlay': ui.overlay,
    '--anytty-app-bg': ui.page,
    '--anytty-app-surface': ui.surface,
    '--anytty-app-surface-soft': ui.surfaceRaised,
    '--anytty-app-soft': ui.surfaceRaised,
    '--anytty-app-line': ui.border,
    '--anytty-app-line-strong': ui.border,
    '--anytty-app-text': ui.text,
    '--anytty-app-muted': ui.muted,
    '--anytty-app-accent': ui.accent,
    '--anytty-app-accent-text': ui.accentText,
    '--anytty-terminal-bg': ui.terminalBackground,
    '--anytty-terminal-fg': ui.terminalForeground,
    '--anytty-terminal-cursor': ui.terminalCursor,
    '--anytty-scrollbar': ui.scrollbar,
    '--anytty-scrollbar-active': ui.scrollbarActive,
  }
}

export function normalizeTerminalSettings(input: Partial<TerminalSettings> | Record<string, unknown>): TerminalSettings {
  const fontFamily = typeof input.fontFamily === 'string'
    ? LEGACY_TERMINAL_FONT_FAMILIES.get(input.fontFamily) ?? input.fontFamily
    : undefined
  return {
    fontSize: clampNumber(input.fontSize, DEFAULT_TERMINAL_SETTINGS.fontSize, 8, 32),
    fontFamily: TERMINAL_FONT_OPTIONS.some((option) => option.value === fontFamily)
      ? fontFamily as string
      : DEFAULT_TERMINAL_SETTINGS.fontFamily,
    themeId: isTerminalThemeId(input.themeId) ? input.themeId : DEFAULT_TERMINAL_SETTINGS.themeId,
    renderer: isTerminalRenderer(input.renderer) ? input.renderer : DEFAULT_TERMINAL_SETTINGS.renderer,
    keyboardMode: isTerminalKeyboardMode(input.keyboardMode) ? input.keyboardMode : DEFAULT_TERMINAL_SETTINGS.keyboardMode,
    scrollInertia: isTerminalScrollInertia(input.scrollInertia) ? input.scrollInertia : DEFAULT_TERMINAL_SETTINGS.scrollInertia,
    scrollback: clampNumber(input.scrollback, DEFAULT_TERMINAL_SETTINGS.scrollback, 500, 50000),
    scrollbackPrefetchThresholdRows: clampNumber(
      input.scrollbackPrefetchThresholdRows,
      DEFAULT_TERMINAL_SETTINGS.scrollbackPrefetchThresholdRows,
      0,
      1000,
    ),
    cursorBlink: typeof input.cursorBlink === 'boolean' ? input.cursorBlink : DEFAULT_TERMINAL_SETTINGS.cursorBlink,
    autoAcquireResizeOwner: typeof input.autoAcquireResizeOwner === 'boolean'
      ? input.autoAcquireResizeOwner
      : DEFAULT_TERMINAL_SETTINGS.autoAcquireResizeOwner,
  }
}

function migrateLegacyTerminalSettings(storage: Pick<Storage, 'getItem'> | RemoteRuntimeStorage | undefined): TerminalSettings {
  const base = { ...DEFAULT_TERMINAL_SETTINGS }
  const legacyRenderer = readStorageValue(storage, 'anytty.renderer')
  if (isTerminalRenderer(legacyRenderer)) base.renderer = legacyRenderer
  const legacyFontSizeValue = readStorageValue(storage, 'anytty.fontSize')
  const legacyFontSize = legacyFontSizeValue === null ? Number.NaN : Number(legacyFontSizeValue)
  if (Number.isFinite(legacyFontSize)) base.fontSize = legacyFontSize
  return normalizeTerminalSettings(base)
}

function readStorageValue(storage: Pick<Storage, 'getItem'> | RemoteRuntimeStorage | undefined, key: string): string | null {
  try {
    return storage?.getItem(key) ?? null
  } catch {
    return null
  }
}

function browserStorage(): Storage | undefined {
  try {
    return globalThis.localStorage
  } catch {
    return undefined
  }
}

function clampNumber(value: unknown, fallback: number, min: number, max: number): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) return fallback
  return Math.max(min, Math.min(max, Math.round(value)))
}

function isTerminalRenderer(value: unknown): value is TerminalRenderer {
  return value === 'auto' || value === 'webgl' || value === 'canvas' || value === 'dom'
}

function isTerminalKeyboardMode(value: unknown): value is TerminalKeyboardMode {
  return value === 'auto' || value === 'resize' || value === 'shift'
}

function isTerminalScrollInertia(value: unknown): value is TerminalScrollInertia {
  return typeof value === 'string' && TERMINAL_SCROLL_INERTIA_OPTIONS.some((option) => option === value)
}

function isTerminalThemeId(value: unknown): value is TerminalThemeId {
  return typeof value === 'string' && TERMINAL_THEME_OPTIONS.some((option) => option.id === value)
}

function colorValue(value: unknown, fallback: string): string {
  return typeof value === 'string' && value.trim() ? value : fallback
}

function withAlpha(color: string, alpha: number): string {
  const hex = color.trim()
  const normalized = hex.startsWith('#') ? hex.slice(1) : hex
  if (/^[0-9a-fA-F]{3}$/.test(normalized)) {
    const [r, g, b] = normalized.split('').map((part) => parseInt(part + part, 16))
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  if (/^[0-9a-fA-F]{6}$/.test(normalized)) {
    const r = parseInt(normalized.slice(0, 2), 16)
    const g = parseInt(normalized.slice(2, 4), 16)
    const b = parseInt(normalized.slice(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }
  return color
}

function rgbChannels(color: string): string {
  const normalized = color.trim().replace(/^#/, '')
  if (/^[0-9a-fA-F]{3}$/.test(normalized)) {
    return normalized.split('').map((part) => parseInt(part + part, 16)).join(' ')
  }
  if (/^[0-9a-fA-F]{6}$/.test(normalized)) {
    return [normalized.slice(0, 2), normalized.slice(2, 4), normalized.slice(4, 6)]
      .map((part) => parseInt(part, 16))
      .join(' ')
  }
  return '113 113 122'
}

function contrastingTextColor(background: string): '#0a0a0a' | '#ffffff' {
  const channels = rgbChannels(background).split(' ').map(Number)
  const luminance = channels.reduce((total, channel, index) => {
    const normalized = channel / 255
    const linear = normalized <= 0.03928 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4
    return total + linear * [0.2126, 0.7152, 0.0722][index]!
  }, 0)
  return 1.05 / (luminance + 0.05) >= 4.5 ? '#ffffff' : '#0a0a0a'
}
