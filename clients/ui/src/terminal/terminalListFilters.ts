import type { Terminal } from '../core/model'

export type TerminalStatusFilter = 'running' | 'exited' | 'all'

export interface TerminalTagOption {
  id: string
  label: string
  count: number
}

export interface PublicTerminalTag {
  id: string
  label: string
}

const positionalTagKeyPattern = /^tag\d+$/i

export function publicTerminalTags(terminal: Pick<Terminal, 'tags'>): PublicTerminalTag[] {
  const labels = new Set<string>()
  for (const [rawKey, rawValue] of Object.entries(terminal.tags ?? {})) {
    const key = rawKey.trim()
    const value = rawValue.trim()
    if (key === '' || key === 'cwd' || key.startsWith('anytty.')) continue

    // tag1/tag2 are transport placeholders for an ordered list. They are not
    // part of the user-authored tag and must never leak into client UI.
    const label = positionalTagKeyPattern.test(key) && value !== ''
      ? value
      : value === '' ? key : `${key}=${value}`
    if (label !== '') labels.add(label)
  }
  return [...labels]
    .sort((left, right) => left.localeCompare(right))
    .map((label) => ({ id: terminalTagId(label), label }))
}

export function terminalTagOptions(terminals: readonly Terminal[]): TerminalTagOption[] {
  const counts = new Map<string, TerminalTagOption>()
  for (const terminal of terminals) {
    for (const tag of publicTerminalTags(terminal)) {
      const current = counts.get(tag.id)
      counts.set(tag.id, current
        ? { ...current, count: current.count + 1 }
        : { ...tag, count: 1 })
    }
  }
  return [...counts.values()].sort((left, right) => left.label.localeCompare(right.label))
}

export function filterTerminals(
  terminals: readonly Terminal[],
  status: TerminalStatusFilter,
  tagIds: readonly string[] = [],
): Terminal[] {
  return terminals.filter((terminal) => {
    if (status !== 'all' && terminal.state !== status) return false
    if (tagIds.length === 0) return true
    const terminalTags = new Set(publicTerminalTags(terminal).map((tag) => tag.id))
    return tagIds.every((tagId) => terminalTags.has(tagId))
  })
}

export function terminalTagId(label: string): string {
  return label
}
