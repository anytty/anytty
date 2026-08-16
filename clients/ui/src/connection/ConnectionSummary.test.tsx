import { cleanup, render, screen } from '@testing-library/react'
import { Cloud } from 'lucide-react'
import { afterEach, describe, expect, it } from 'vitest'
import type { ConnectionPresentation } from './connectionPresentation'
import { ConnectionSummary } from './ConnectionSummary'

afterEach(() => cleanup())

describe('ConnectionSummary', () => {
  it('shows a compact, text-backed status without relying on color alone', () => {
    render(
      <ConnectionSummary
        presentation={presentation({ state: 'ready', tone: 'positive', route: 'direct', observedPath: 'p2p' })}
        label="Connected"
        detail="Direct · 32 ms"
      />,
    )

    const summary = screen.getByText('Connected').parentElement!
    expect(summary.dataset.connectionState).toBe('ready')
    expect(summary.dataset.tone).toBe('positive')
    expect(summary.dataset.route).toBe('direct')
    expect(summary.dataset.density).toBe('compact')
    expect(screen.getByText('Direct · 32 ms')).toBeTruthy()
    expect(summary.querySelector('svg')).toBeTruthy()
  })

  it('announces an explicitly live status and respects reduced-motion for progress', () => {
    render(
      <ConnectionSummary
        announce
        presentation={presentation({ state: 'connecting', tone: 'info', action: 'none' })}
        label="Reconnecting"
      />,
    )

    const summary = screen.getByRole('status')
    expect(summary.getAttribute('aria-live')).toBe('polite')
    expect(summary.querySelector('svg')?.classList.contains('animate-spin')).toBe(true)
    expect(summary.querySelector('svg')?.classList.contains('motion-reduce:animate-none')).toBe(true)
  })

  it('breathes a semantic route icon without adding a secondary spinner', () => {
    render(
      <ConnectionSummary
        icon={Cloud}
        presentation={presentation({ state: 'idle', tone: 'info', reachability: 'checking' })}
        label="Checking cloud route"
      />,
    )

    const summary = screen.getByText('Checking cloud route').parentElement!
    const icons = summary.querySelectorAll('svg')
    expect(icons).toHaveLength(1)
    expect(icons[0]?.classList.contains('animate-spin')).toBe(false)
    expect(icons[0]?.parentElement?.classList.contains('animate-pulse')).toBe(true)
    expect(icons[0]?.parentElement?.classList.contains('motion-reduce:animate-none')).toBe(true)
  })

  it('does not create a noisy live region by default', () => {
    render(
      <ConnectionSummary
        presentation={presentation({ state: 'idle', tone: 'warning', reachability: 'unreachable', action: 'connect' })}
        label="Device unavailable"
      />,
    )

    expect(screen.queryByRole('status')).toBeNull()
    expect(screen.getByText('Device unavailable')).toBeTruthy()
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
