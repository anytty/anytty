// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { LocalWebLogin } from './LocalWebLogin'

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

describe('LocalWebLogin', () => {
  it('submits the password and continues after authentication', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }))
    vi.stubGlobal('fetch', fetchMock)
    const authenticated = vi.fn().mockResolvedValue(undefined)
    const user = userEvent.setup()
    render(<LocalWebLogin initialAppThemeStyle={{}} onAuthenticated={authenticated} />)

    await user.type(screen.getByLabelText(/access password/i), 'correct horse battery staple')
    await user.click(screen.getByRole('button', { name: /unlock terminal/i }))

    await waitFor(() => expect(authenticated).toHaveBeenCalledOnce())
    expect(fetchMock).toHaveBeenCalledWith('/api/auth/login', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ password: 'correct horse battery staple' }),
    }))
  })

  it('keeps the form visible and reports an incorrect password', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"error":"invalid_password"}', { status: 401 })))
    render(<LocalWebLogin initialAppThemeStyle={{}} onAuthenticated={vi.fn()} />)

    fireEvent.change(screen.getByLabelText(/access password/i), { target: { value: 'wrong password' } })
    fireEvent.submit(screen.getByRole('button', { name: /unlock terminal/i }).closest('form')!)

    expect((await screen.findByRole('alert')).textContent).toContain('incorrect')
  })

  it('provides a named show-password control and stable touch targets', () => {
    vi.stubGlobal('fetch', vi.fn())
    render(<LocalWebLogin initialAppThemeStyle={{}} onAuthenticated={vi.fn()} />)

    expect(screen.getByRole('button', { name: /show password/i }).className).toContain('size-12')
    expect(screen.getByLabelText(/access password/i).className).toContain('h-12')
  })

  it('explains that public access requires HTTPS', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{"error":"https_required"}', { status: 426 })))
    render(<LocalWebLogin initialAppThemeStyle={{}} onAuthenticated={vi.fn()} />)

    fireEvent.change(screen.getByLabelText(/access password/i), { target: { value: 'correct horse battery staple' } })
    fireEvent.submit(screen.getByRole('button', { name: /unlock terminal/i }).closest('form')!)

    expect((await screen.findByRole('alert')).textContent).toContain('HTTPS')
  })
})
