import { useState } from 'react'
import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { FilePreviewSheet } from './FilePreviewSheet'

describe('FilePreviewSheet', () => {
  afterEach(cleanup)

  it('exposes a named modal and restores the preview trigger after Escape', async () => {
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<Harness onClose={onClose} />)

    const dialog = screen.getByRole('dialog', { name: 'notes.txt' })
    const close = screen.getByRole('button', { name: 'Close preview' })
    const trigger = screen.getByText('Preview notes')

    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.activeElement).toBe(close)
    expect(trigger.closest('[inert]')).toBeTruthy()

    await user.keyboard('{Escape}')

    expect(onClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('dialog', { name: 'notes.txt' })).toBeNull()
    expect(trigger.closest('[inert]')).toBeNull()
    expect(document.activeElement).toBe(trigger)
  })

  it('portals outside the transformed file sidebar and retains full-viewport geometry', () => {
    render(
      <div
        className="relative flex h-full min-h-0 w-full max-w-full flex-col overflow-hidden md:flex-row"
        data-testid="machine-workspace"
        style={{ overflow: 'hidden' }}
      >
        <div
          className="absolute inset-0 z-30 flex flex-col md:left-auto md:right-0 md:w-[450px] translate-y-0 md:translate-x-0 visible"
          data-testid="anytty-machine-files-overlay"
          style={{ transform: 'translateX(0)', width: '450px' }}
        >
          <FilePreviewSheet
            path="/docs/notes.txt"
            preview={null}
            loading
            error={null}
            streamPreview={vi.fn()}
            onClose={vi.fn()}
          />
        </div>
      </div>,
    )

    const workspace = screen.getByTestId('machine-workspace')
    const fileSidebar = screen.getByTestId('anytty-machine-files-overlay')
    const dialog = screen.getByTestId('anytty-file-preview')

    expect(window.getComputedStyle(workspace).overflow).toBe('hidden')
    expect(window.getComputedStyle(fileSidebar).transform).not.toBe('none')
    expect(window.getComputedStyle(fileSidebar).width).toBe('450px')
    expect(dialog.parentElement).toBe(document.body)
    expect(fileSidebar.contains(dialog)).toBe(false)
    expect(dialog.classList.contains('fixed')).toBe(true)
    expect(dialog.classList.contains('inset-0')).toBe(true)
    expect(dialog.classList.contains('h-[100dvh]')).toBe(true)
  })

  it('keeps title and path associations unique across two previews', () => {
    render(
      <>
        <FilePreviewSheet path="/docs/first.txt" preview={null} loading error={null} streamPreview={vi.fn()} onClose={vi.fn()} />
        <FilePreviewSheet path="/docs/second.txt" preview={null} loading error={null} streamPreview={vi.fn()} onClose={vi.fn()} />
      </>,
    )

    const dialogs = Array.from(document.querySelectorAll<HTMLElement>('[data-testid="anytty-file-preview"]'))
    expect(dialogs).toHaveLength(2)
    expectUniqueAssociations(dialogs, ['first.txt', 'second.txt'], ['/docs/first.txt', '/docs/second.txt'])
  })

  it('replaces remote preview loading with an offline state while keeping Close available', async () => {
    const onClose = vi.fn()
    render(
      <FilePreviewSheet
        path="/docs/remote.pdf"
        preview={null}
        loading
        error={null}
        remoteAvailable={false}
        unavailableLabel="Connection interrupted. Reconnecting"
        streamPreview={vi.fn()}
        onClose={onClose}
      />,
    )

    expect(screen.getByRole('status').textContent).toContain('Connection interrupted. Reconnecting')
    expect(screen.getByRole('status').textContent).not.toContain('Your phone is offline')
    expect(screen.queryByText('Loading preview...')).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Close preview' }))
    expect(onClose).toHaveBeenCalledOnce()
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
      <button autoFocus type="button">Preview notes</button>
      {open ? (
        <FilePreviewSheet
          path="/docs/notes.txt"
          preview={null}
          loading
          error={null}
          streamPreview={vi.fn()}
          onClose={() => {
            onClose()
            setOpen(false)
          }}
        />
      ) : null}
    </div>
  )
}
