import { useState } from 'react'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ActionSheet } from './ActionSheet'
import { anyttyI18n } from '../i18n'

describe('ActionSheet', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  afterEach(async () => {
    cleanup()
    await anyttyI18n.changeLanguage('en')
  })

  it('exposes a named modal, traps focus, closes on Escape, and returns focus', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<Harness onClose={onClose} />)

    const dialog = screen.getByRole('dialog', { name: 'File actions' })
    const trigger = screen.getByText('Open actions')
    const firstAction = screen.getByRole('button', { name: 'Rename' })
    const close = screen.getByRole('button', { name: 'Close' })
    const lastAction = screen.getByRole('button', { name: 'Delete' })

    expect(screen.getByTestId('action-sheet-backdrop').parentElement).toBe(document.body)
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.activeElement).toBe(firstAction)
    expect(trigger.closest('[inert]')).toBeTruthy()

    close.focus()
    await user.tab({ shift: true })
    expect(document.activeElement).toBe(lastAction)
    await user.tab()
    expect(document.activeElement).toBe(close)
    await user.keyboard('{Escape}')

    expect(onClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('dialog', { name: 'File actions' })).toBeNull()
    expect(trigger.closest('[inert]')).toBeNull()
    expect(document.activeElement).toBe(trigger)
  })

  it('localizes the fallback dialog and close labels in English and Chinese', async () => {
    const actions = [{ label: 'Run', icon: <span aria-hidden="true">R</span>, onClick: vi.fn() }]
    const onClose = vi.fn()

    render(<ActionSheet isOpen onClose={onClose} actions={actions} />)
    expect(screen.getByRole('dialog', { name: 'Actions' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Close' })).toBeTruthy()

    cleanup()
    await anyttyI18n.changeLanguage('zh-CN')
    render(<ActionSheet isOpen onClose={onClose} actions={actions} />)
    expect(screen.getByRole('dialog', { name: '操作' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '关闭' })).toBeTruthy()
  })

  it('keeps title and subtitle associations unique across two sheets', () => {
    const actions = [{ label: 'Run', icon: <span aria-hidden="true">R</span>, onClick: vi.fn() }]
    render(
      <>
        <ActionSheet isOpen onClose={vi.fn()} title="First actions" subtitle="First target" actions={actions} />
        <ActionSheet isOpen onClose={vi.fn()} title="Second actions" subtitle="Second target" actions={actions} />
      </>,
    )

    const dialogs = Array.from(document.querySelectorAll<HTMLElement>('[role="dialog"]'))
    expect(dialogs).toHaveLength(2)
    expectUniqueAssociations(dialogs, ['First actions', 'Second actions'], ['First target', 'Second target'])
  })
})

function expectUniqueAssociations(dialogs: HTMLElement[], titles: string[], descriptions: string[]) {
  const ids: string[] = []
  dialogs.forEach((dialog, index) => {
    const titleId = dialog.getAttribute('aria-labelledby')
    const descriptionId = dialog.getAttribute('aria-describedby')
    expect(titleId).toBeTruthy()
    expect(descriptionId).toBeTruthy()
    ids.push(titleId!, descriptionId!)

    const title = document.getElementById(titleId!)
    const description = document.getElementById(descriptionId!)
    expect(dialog.contains(title)).toBe(true)
    expect(dialog.contains(description)).toBe(true)
    expect(title?.textContent).toBe(titles[index])
    expect(description?.textContent).toBe(descriptions[index])
  })
  expect(new Set(ids).size).toBe(ids.length)
}

function Harness({ onClose }: { onClose: () => void }) {
  const [open, setOpen] = useState(true)
  return (
    <div>
      <button autoFocus type="button">Open actions</button>
      <ActionSheet
        isOpen={open}
        onClose={() => {
          onClose()
          setOpen(false)
        }}
        title="File actions"
        actions={[
          { label: 'Rename', icon: <span aria-hidden="true">R</span>, onClick: () => undefined },
          { label: 'Delete', icon: <span aria-hidden="true">D</span>, onClick: () => undefined, danger: true },
        ]}
      />
    </div>
  )
}
