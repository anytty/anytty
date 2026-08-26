import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { TerminalListFilterBar } from './TerminalListFilterBar'

describe('TerminalListFilterBar', () => {
  afterEach(cleanup)

  it('keeps status and tag controls in one stable row and toggles desktop tags', async () => {
    const onTagToggle = vi.fn()
    render(
      <TerminalListFilterBar
        filteredCount={2}
        mobileTagSheet={false}
        onClearTags={vi.fn()}
        onOpenTagSheet={vi.fn()}
        onStatusChange={vi.fn()}
        onTagToggle={onTagToggle}
        selectedTagIds={[]}
        status="running"
        statusCounts={{ running: 2, exited: 1, all: 3 }}
        tagOptions={[{ id: 'tag=123', label: 'tag=123', count: 2 }]}
        tagSheetOpen={false}
      />,
    )

    const toolbar = screen.getByTestId('anytty-terminal-filter-bar')
    expect(toolbar.className).toContain('h-11')
    expect(toolbar.className).toContain('grid-cols-4')
    expect(screen.getByTestId('anytty-terminal-status-running').getAttribute('aria-pressed')).toBe('true')

    await userEvent.click(screen.getByTestId('anytty-terminal-tag-filter'))
    await userEvent.click(screen.getByRole('menuitemcheckbox', { name: /tag=123/ }))
    expect(onTagToggle).toHaveBeenCalledWith('tag=123')
  })

  it('exposes the mobile tag sheet state from the compact trigger', async () => {
    const onOpenTagSheet = vi.fn()
    render(
      <TerminalListFilterBar
        filteredCount={1}
        mobileTagSheet
        onClearTags={vi.fn()}
        onOpenTagSheet={onOpenTagSheet}
        onStatusChange={vi.fn()}
        onTagToggle={vi.fn()}
        selectedTagIds={['ABC']}
        status="all"
        statusCounts={{ running: 1, exited: 0, all: 1 }}
        tagOptions={[{ id: 'ABC', label: 'ABC', count: 1 }]}
        tagSheetOpen
      />,
    )

    const trigger = screen.getByTestId('anytty-terminal-tag-filter')
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(trigger.textContent).toContain('1')
    await userEvent.click(trigger)
    expect(onOpenTagSheet).toHaveBeenCalledOnce()
  })
})
