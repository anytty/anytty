import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { anyttyI18n } from '../i18n'
import { PasteConfirmDialog } from './PasteConfirmDialog'

describe('PasteConfirmDialog', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  afterEach(cleanup)

  it('portals outside terminal layout constraints and remains keyboard dismissible', async () => {
    const onCancel = vi.fn()
    render(
      <div style={{ transform: 'translateX(0)', width: '320px' }}>
        <button type="button">Paste trigger</button>
        <PasteConfirmDialog text={'first\nsecond'} onCancel={onCancel} onConfirm={vi.fn()} />
      </div>,
    )

    const overlay = screen.getByTestId('anytty-paste-confirm')
    const dialog = screen.getByRole('dialog', { name: 'Confirm paste' })
    expect(overlay.parentElement).toBe(document.body)
    expect(overlay.classList.contains('fixed')).toBe(true)
    expect(dialog.getAttribute('aria-modal')).toBe('true')

    await userEvent.keyboard('{Escape}')
    expect(onCancel).toHaveBeenCalledOnce()
  })
})
