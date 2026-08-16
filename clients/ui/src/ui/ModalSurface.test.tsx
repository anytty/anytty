import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useState } from 'react'
import { ModalSurface } from './ModalSurface'

describe('ModalSurface', () => {
  afterEach(() => {
    cleanup()
    document.body.removeAttribute('style')
  })

  it('traps focus, closes on Escape, isolates the background, and restores focus', async () => {
    const onClose = vi.fn()
    const { rerender } = render(<Harness open onClose={onClose} />)
    const dialog = screen.getByRole('dialog', { name: 'Test dialog' })
    const background = screen.getByText('Background action')

    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'First action' }))
    expect(background.hasAttribute('inert')).toBe(true)
    expect(document.body.style.overflow).toBe('hidden')

    await userEvent.tab({ shift: true })
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Last action' }))
    await userEvent.tab()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'First action' }))
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)

    rerender(<Harness open={false} onClose={onClose} />)
    expect(background.hasAttribute('inert')).toBe(false)
    expect(document.body.style.overflow).toBe('')
    expect(document.activeElement).toBe(background)
  })

  it('moves and restores focus without scrolling the underlying view', () => {
    const focus = vi.spyOn(HTMLElement.prototype, 'focus')
    const { rerender } = render(<Harness open onClose={() => undefined} />)

    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
    focus.mockClear()
    rerender(<Harness open={false} onClose={() => undefined} />)
    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
    focus.mockRestore()
  })

  it('keeps focus and Escape inside the topmost nested surface', async () => {
    const user = userEvent.setup()
    const onParentClose = vi.fn()
    const onChildClose = vi.fn()
    render(<NestedHarness onParentClose={onParentClose} onChildClose={onChildClose} />)

    const openChild = screen.getByRole('button', { name: 'Open child' })
    expect(document.activeElement).toBe(openChild)
    await user.click(openChild)

    expect(screen.getByRole('dialog', { name: 'Parent dialog' })).toBeTruthy()
    expect(screen.getByRole('dialog', { name: 'Child dialog' })).toBeTruthy()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Child first' }))

    await user.tab({ shift: true })
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Child last' }))
    await user.tab()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Child first' }))
    await user.keyboard('{Escape}')

    expect(onChildClose).toHaveBeenCalledTimes(1)
    expect(onParentClose).not.toHaveBeenCalled()
    expect(screen.queryByRole('dialog', { name: 'Child dialog' })).toBeNull()
    expect(screen.getByRole('dialog', { name: 'Parent dialog' })).toBeTruthy()
    expect(document.activeElement).toBe(openChild)

    await user.keyboard('{Escape}')
    expect(onParentClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('dialog')).toBeNull()
    expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Nested trigger' }))
  })

  it.each(['parent', 'child'] as const)('restores body and background state when the %s surface unmounts first', (firstToClose) => {
    document.body.style.cssText = 'color: red; overflow: clip;'
    const originalBodyStyle = document.body.style.cssText
    const { rerender } = render(<ParallelHarness parentOpen childOpen={false} />)
    const background = screen.getByText('Lifecycle trigger')

    rerender(<ParallelHarness parentOpen childOpen />)

    expect(document.body.style.overflow).toBe('hidden')
    expect(background.hasAttribute('inert')).toBe(true)

    rerender(
      <ParallelHarness
        parentOpen={firstToClose !== 'parent'}
        childOpen={firstToClose !== 'child'}
      />,
    )
    expect(document.body.style.overflow).toBe('hidden')
    expect(background.hasAttribute('inert')).toBe(true)

    rerender(<ParallelHarness parentOpen={false} childOpen={false} />)
    expect(document.body.style.cssText).toBe(originalBodyStyle)
    expect(background.getAttribute('aria-hidden')).toBe('false')
    expect(background.hasAttribute('inert')).toBe(false)
    expect(document.querySelectorAll('[inert]')).toHaveLength(0)
    expect(document.activeElement).toBe(background)
  })
})

function Harness({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <div>
      <button autoFocus type="button">Background action</button>
      {open ? (
        <ModalSurface aria-label="Test dialog" onRequestClose={onClose}>
          <button type="button">First action</button>
          <button type="button">Last action</button>
        </ModalSurface>
      ) : null}
    </div>
  )
}

function NestedHarness({ onParentClose, onChildClose }: { onParentClose: () => void; onChildClose: () => void }) {
  const [parentOpen, setParentOpen] = useState(true)
  const [childOpen, setChildOpen] = useState(false)
  return (
    <div>
      <button autoFocus type="button">Nested trigger</button>
      {parentOpen ? (
        <ModalSurface
          aria-label="Parent dialog"
          onRequestClose={() => {
            onParentClose()
            setParentOpen(false)
          }}
        >
          <button type="button" onClick={() => setChildOpen(true)}>Open child</button>
          <button type="button">Parent last</button>
          {childOpen ? (
            <ModalSurface
              aria-label="Child dialog"
              onRequestClose={() => {
                onChildClose()
                setChildOpen(false)
              }}
            >
              <button type="button">Child first</button>
              <button type="button">Child last</button>
            </ModalSurface>
          ) : null}
        </ModalSurface>
      ) : null}
    </div>
  )
}

function ParallelHarness({ parentOpen, childOpen }: { parentOpen: boolean; childOpen: boolean }) {
  return (
    <div>
      <button aria-hidden="false" autoFocus type="button">Lifecycle trigger</button>
      {parentOpen ? (
        <ModalSurface aria-label="Lifecycle parent" onRequestClose={() => undefined}>
          <button type="button">Parent control</button>
        </ModalSurface>
      ) : null}
      {childOpen ? (
        <ModalSurface aria-label="Lifecycle child" onRequestClose={() => undefined}>
          <button type="button">Child control</button>
        </ModalSurface>
      ) : null}
    </div>
  )
}
