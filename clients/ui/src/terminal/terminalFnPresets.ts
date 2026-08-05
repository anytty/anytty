export interface TerminalFnItem {
  label: string
  data: string
  description?: string
}

export interface TerminalFnGroup {
  name: string
  items: TerminalFnItem[]
}

export interface TerminalFnPreset {
  id: string
  name: string
  match: string[]
  groups: TerminalFnGroup[]
}

export const SYSTEM_FN_GROUPS: TerminalFnGroup[] = [
  {
    name: 'Process',
    items: [
      { label: 'Ctrl+C', data: '\x03', description: 'Interrupt' },
      { label: 'Ctrl+D', data: '\x04', description: 'EOF' },
      { label: 'Ctrl+Z', data: '\x1a', description: 'Suspend' },
      { label: 'Ctrl+L', data: '\x0c', description: 'Clear' },
      { label: 'Ctrl+R', data: '\x12', description: 'History' },
      { label: 'Ctrl+\\', data: '\x1c', description: 'Quit' },
    ],
  },
  {
    name: 'Line editing',
    items: [
      { label: 'Ctrl+A', data: '\x01', description: 'Line start' },
      { label: 'Ctrl+E', data: '\x05', description: 'Line end' },
      { label: 'Ctrl+W', data: '\x17', description: 'Delete word' },
      { label: 'Ctrl+U', data: '\x15', description: 'Delete line' },
      { label: 'Ctrl+K', data: '\x0b', description: 'Delete tail' },
      { label: 'Tab', data: '\t', description: 'Complete' },
    ],
  },
]

const PROGRAM_PRESETS: TerminalFnPreset[] = [
  {
    id: 'claude',
    name: 'Claude Code',
    match: ['claude'],
    groups: [
      {
        name: 'Commands',
        items: [
          { label: '/clear', data: '/clear\n', description: 'Reset context' },
          { label: '/compact', data: '/compact\n', description: 'Compact' },
          { label: '/cost', data: '/cost\n', description: 'Usage' },
          { label: '/help', data: '/help\n', description: 'Help' },
          { label: '/review', data: '/review\n', description: 'Review' },
          { label: '/init', data: '/init\n', description: 'Init' },
        ],
      },
      {
        name: 'Replies',
        items: [
          { label: 'yes', data: 'yes\n', description: 'Confirm' },
          { label: 'no', data: 'no\n', description: 'Decline' },
          { label: 'exit', data: 'exit\n', description: 'Exit' },
        ],
      },
    ],
  },
  {
    id: 'opencode',
    name: 'OpenCode',
    match: ['opencode'],
    groups: [
      {
        name: 'Commands',
        items: [
          { label: '/clear', data: '/clear\n', description: 'Reset context' },
          { label: '/compact', data: '/compact\n', description: 'Compact' },
          { label: '/cost', data: '/cost\n', description: 'Usage' },
          { label: '/help', data: '/help\n', description: 'Help' },
        ],
      },
      {
        name: 'Replies',
        items: [
          { label: 'yes', data: 'yes\n', description: 'Confirm' },
          { label: 'no', data: 'no\n', description: 'Decline' },
        ],
      },
    ],
  },
]

export function matchTerminalFnPreset(command: string | undefined): TerminalFnPreset | null {
  if (!command) return null
  const normalized = command.toLowerCase()
  return PROGRAM_PRESETS.find((preset) => preset.match.some((item) => normalized.includes(item))) ?? null
}
