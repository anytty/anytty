// @vitest-environment jsdom

import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { RegistryStartupScreen, UnsupportedWebPreview } from './RegistryStartupScreen'

describe('mobile registry startup recovery', () => {
  const originalClipboard = Object.getOwnPropertyDescriptor(navigator, 'clipboard')

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    if (originalClipboard) Object.defineProperty(navigator, 'clipboard', originalClipboard)
    else Reflect.deleteProperty(navigator, 'clipboard')
  })

  it('offers native retry, diagnostics, and a confirmed local-only reset', async () => {
    const onRetry = vi.fn(async () => undefined)
    const onResetLocalPairings = vi.fn(async () => undefined)
    render(
      <RegistryStartupScreen
        error="native registry checksum failed"
        onResetLocalPairings={onResetLocalPairings}
        onRetry={onRetry}
      />,
    )

    expect(screen.getByRole('alert').textContent).toMatch(/native endpoint registry/i)
    expect(screen.getByRole('button', { name: 'Copy diagnostics' })).toBeTruthy()
    expect(screen.getByText('registry_integrity_failed')).toBeTruthy()
    expect(screen.queryByText('native registry checksum failed')).toBeNull()
    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))
    await userEvent.click(screen.getByRole('button', { name: 'Reset local pairings' }))
    expect(onResetLocalPairings).not.toHaveBeenCalled()
    expect(screen.getByText(/endpoint registry, saved access credentials, SSH keys, and transfer history/i)).toBeTruthy()
    await userEvent.click(screen.getByRole('button', { name: 'Reset pairings' }))

    expect(onRetry).toHaveBeenCalledTimes(1)
    expect(onResetLocalPairings).toHaveBeenCalledTimes(1)
  })

  it('creates and writes one bounded redacted snapshot only after the user clicks', async () => {
    const writeText = vi.fn(async (_text: string) => undefined)
    const clipboardAccess = vi.fn(() => ({ writeText }))
    Object.defineProperty(navigator, 'clipboard', { configurable: true, get: clipboardAccess })
    const secretError = 'Bearer SECRET_SENTINEL token=TOKEN_SENTINEL credential=CREDENTIAL_SENTINEL password=PASSWORD_SENTINEL https://service.example/private?auth=QUERY_SENTINEL#FRAGMENT_SENTINEL /data/user/0/com.anytty/FILE_SENTINEL checksum failed'
    render(
      <RegistryStartupScreen
        error={secretError}
        onResetLocalPairings={vi.fn(async () => undefined)}
        onRetry={vi.fn(async () => undefined)}
      />,
    )

    expect(clipboardAccess).not.toHaveBeenCalled()
    expect(writeText).not.toHaveBeenCalled()
    expect(screen.queryByText(/SECRET_SENTINEL|TOKEN_SENTINEL|QUERY_SENTINEL|FILE_SENTINEL/)).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: 'Copy diagnostics' }))

    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1))
    expect(clipboardAccess).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('status').textContent).toMatch(/redacted diagnostics copied/i)

    const snapshot = writeText.mock.calls[0]?.[0] ?? ''
    expect(snapshot.length).toBeLessThanOrEqual(512)
    expect(snapshot).not.toMatch(/bearer|token|credential|password|https?:|query_sentinel|fragment_sentinel|\/data\/|file_sentinel/i)
    expect(JSON.parse(snapshot)).toEqual({
      schema: 'anytty.mobile.registry-startup-diagnostic',
      version: 1,
      failure: {
        category: 'native_endpoint_registry',
        code: 'registry_integrity_failed',
      },
      recovery: {
        retry: true,
        reset_local_pairings: true,
      },
    })
  })

  it('announces a clipboard failure without claiming success', async () => {
    const writeText = vi.fn(async () => { throw new Error('clipboard denied') })
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    render(
      <RegistryStartupScreen
        error="native registry unavailable"
        onResetLocalPairings={vi.fn(async () => undefined)}
        onRetry={vi.fn(async () => undefined)}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Copy diagnostics' }))

    const failure = await screen.findByText(/diagnostics could not be copied/i)
    expect(failure.getAttribute('role')).toBe('alert')
    expect(writeText).toHaveBeenCalledTimes(1)
    expect(screen.queryByText(/redacted diagnostics copied/i)).toBeNull()
  })

  it('keeps the recovery page structural scrolling, width, safe-area, and touch-target contracts', () => {
    render(
      <RegistryStartupScreen
        error="native registry checksum failed"
        onResetLocalPairings={vi.fn(async () => undefined)}
        onRetry={vi.fn(async () => undefined)}
      />,
    )

    const panel = screen.getByTestId('registry-startup-error')
    const page = panel.closest('main')
    expect(page?.className).toContain('h-[100dvh]')
    expect(page?.className).toContain('overflow-y-auto')
    expect(page?.className).toContain('w-full')
    expect(page?.className).not.toContain('w-screen')
    expect(page?.className).toContain('pt-[calc(env(safe-area-inset-top)+1rem)]')
    expect(page?.className).toContain('pr-[calc(env(safe-area-inset-right)+1rem)]')
    expect(page?.className).toContain('pb-[calc(env(safe-area-inset-bottom)+1rem)]')
    expect(page?.className).toContain('pl-[calc(env(safe-area-inset-left)+1rem)]')
    expect(panel.className).toContain('min-w-0')
    expect(panel.querySelector('details')?.className).toContain('min-w-0')
    const controls = Array.from(panel.querySelectorAll('button, summary'))
    expect(controls.length).toBeGreaterThan(0)
    for (const control of controls) expect(control.className).toContain('min-h-11')
  })

  it('presents unsupported web preview without native recovery controls', () => {
    render(<UnsupportedWebPreview />)

    expect(screen.getByTestId('unsupported-web-preview')).toBeTruthy()
    expect(screen.getByText(/browser preview cannot access native pairing storage/i)).toBeTruthy()
    expect(screen.queryByRole('button')).toBeNull()
  })
})
