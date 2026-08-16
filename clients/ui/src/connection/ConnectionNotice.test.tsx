import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { ConnectionPresentation } from './connectionPresentation'
import { ConnectionNotice } from './ConnectionNotice'

afterEach(() => cleanup())

describe('ConnectionNotice', () => {
  it('renders recovery as an inline, non-blocking notice with touch-sized actions', async () => {
    const user = userEvent.setup()
    const onRetry = vi.fn()
    render(
      <ConnectionNotice
        presentation={presentation({ state: 'failed', tone: 'critical', action: 'retry' })}
        title="Connection could not be restored"
        description="Try again or choose another connection method."
        primaryAction={{ label: 'Retry', onClick: onRetry }}
      />,
    )

    const notice = screen.getByRole('alert')
    expect(notice.dataset.variant).toBe('notice')
    expect(notice.className).not.toMatch(/\b(?:fixed|absolute|inset-0)\b/)
    const retry = screen.getByRole('button', { name: 'Retry' })
    expect(retry.className).toContain('min-h-11')
    expect(retry.className).toContain('min-w-11')
    await user.click(retry)
    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('downgrades a requested gate when existing content must remain available', () => {
    render(
      <ConnectionNotice
        hasContent
        presentation={presentation({ state: 'connecting', tone: 'info' })}
        title="Restoring connection"
        description="Terminal input is paused."
        variant="gate"
      />,
    )

    const notice = screen.getByRole('status')
    expect(notice.dataset.requestedVariant).toBe('gate')
    expect(notice.dataset.variant).toBe('notice')
    expect(notice.className).toContain('rounded-none')
    expect(notice.getAttribute('aria-modal')).toBeNull()
  })

  it('uses the gate density only for an empty-content state and keeps an escape action', () => {
    render(
      <ConnectionNotice
        presentation={presentation({ state: 'phone_offline', tone: 'warning' })}
        title="Your phone is offline"
        description="Check Wi-Fi or mobile data."
        secondaryAction={{ label: 'Back to devices', onClick: vi.fn() }}
        variant="gate"
      />,
    )

    const gate = screen.getByRole('status')
    expect(gate.dataset.variant).toBe('gate')
    expect(gate.className).not.toMatch(/\b(?:fixed|absolute|inset-0)\b/)
    expect(screen.getByRole('button', { name: 'Back to devices' }).className).toContain('min-h-11')
  })

  it('provides an accessible, tooltip-named details control', async () => {
    const user = userEvent.setup()
    const onDetails = vi.fn()
    render(
      <ConnectionNotice
        presentation={presentation({ state: 'connecting', tone: 'info' })}
        title="Restoring connection"
        detailsAction={{ label: 'Connection details', tooltip: 'Open connection details', onClick: onDetails }}
      />,
    )

    const details = screen.getByRole('button', { name: 'Connection details' })
    expect(details.className).toContain('min-h-11')
    expect(details.className).toContain('min-w-11')
    await user.hover(details)
    expect((await screen.findByRole('tooltip')).textContent).toBe('Open connection details')
    await user.click(details)
    expect(onDetails).toHaveBeenCalledTimes(1)
  })

  it('disables and labels an action while it is pending', () => {
    render(
      <ConnectionNotice
        presentation={presentation({ state: 'failed', tone: 'critical', action: 'retry' })}
        title="Connection failed"
        primaryAction={{ label: 'Retrying', onClick: vi.fn(), pending: true }}
      />,
    )

    const retry = screen.getByRole('button', { name: 'Retrying' }) as HTMLButtonElement
    expect(retry.disabled).toBe(true)
    expect(retry.getAttribute('aria-busy')).toBe('true')
    expect(retry.querySelector('svg')).toBeTruthy()
  })

  it('keeps the connection icon still and spins only a separate progress ring', () => {
    render(
      <ConnectionNotice
        presentation={presentation({ state: 'connecting', tone: 'info' })}
        title="Connecting"
      />,
    )

    const notice = screen.getByRole('status')
    const spinner = notice.querySelector('[data-connection-spinner="true"]')
    expect(spinner?.classList.contains('animate-spin')).toBe(true)
    const icons = notice.querySelectorAll('svg')
    expect(icons[0]?.classList.contains('animate-spin')).toBe(false)
  })
})

function presentation(overrides: Partial<ConnectionPresentation> = {}): ConnectionPresentation {
  return {
    state: 'idle',
    tone: 'neutral',
    reachability: 'unknown',
    policy: 'automatic',
    route: 'none',
    observedPath: 'none',
    action: 'none',
    ...overrides,
  }
}
