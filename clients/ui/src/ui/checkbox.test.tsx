import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { Checkbox } from './checkbox'

describe('Checkbox', () => {
  afterEach(() => cleanup())

  it('announces checked state and reports the next value', async () => {
    const user = userEvent.setup()
    const onCheckedChange = vi.fn()
    render(<Checkbox aria-label="Enable route" checked={false} onCheckedChange={onCheckedChange} />)

    const checkbox = screen.getByRole('checkbox', { name: 'Enable route' })
    expect(checkbox.getAttribute('aria-checked')).toBe('false')
    expect(checkbox.getAttribute('data-state')).toBe('unchecked')

    await user.click(checkbox)
    expect(onCheckedChange).toHaveBeenCalledWith(true)
  })

  it('supports uncontrolled toggling', async () => {
    const user = userEvent.setup()
    render(<Checkbox aria-label="Use yearly price" defaultChecked />)

    const checkbox = screen.getByRole('checkbox', { name: 'Use yearly price' })
    expect(checkbox.getAttribute('aria-checked')).toBe('true')
    await user.click(checkbox)
    expect(checkbox.getAttribute('aria-checked')).toBe('false')
  })
})
