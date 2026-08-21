import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ModalSurface } from '../ui/ModalSurface'
import {
  ConnectionRecoveryOverlayHost,
  ConnectionRecoveryOverlayProvider,
  useConnectionRecoveryOverlay,
  type ConnectionRecoveryOverlayIntent,
} from './ConnectionRecoveryOverlay'

describe('ConnectionRecoveryOverlay', () => {
  afterEach(cleanup)

  it('keeps one scoped frosted surface without nesting a card and lets the app state override child sources', async () => {
    render(
      <ConnectionRecoveryOverlayProvider
        appIntent={{ kind: 'offline', title: 'Phone offline', description: 'Waiting for the network' }}
      >
        <header data-testid="page-header"><button type="button">Back</button></header>
        <div className="relative h-96" data-testid="page-content">
          <IntentSource intent={{ kind: 'failed', title: 'Terminal failed' }} />
          <ConnectionRecoveryOverlayHost />
        </div>
      </ConnectionRecoveryOverlayProvider>,
    )

    const overlay = await screen.findByTestId('anytty-connection-recovery-overlay')
    const root = overlay.closest<HTMLElement>('[data-anytty-connection-overlay-root]')
    const host = overlay.closest<HTMLElement>('[data-anytty-connection-overlay-host]')
    expect(screen.getAllByTestId('anytty-connection-recovery-overlay')).toHaveLength(1)
    expect(overlay.getAttribute('data-connection-overlay-kind')).toBe('offline')
    expect(overlay.textContent).toContain('Phone offline')
    expect(overlay.textContent).not.toContain('Terminal failed')
    expect(overlay.dataset.connectionOverlaySurface).toBe('unframed')
    expect(overlay.className).not.toContain('rounded')
    expect(overlay.className).not.toContain('border')
    expect(overlay.className).not.toContain('shadow')
    expect(overlay.className).not.toContain('backdrop-blur')
    expect(root?.className).toContain('absolute')
    expect(root?.className).toContain('inset-0')
    expect(root?.className).toContain('backdrop-blur-[6px]')
    expect(root?.className).not.toContain('fixed')
    expect(host?.parentElement).toBe(screen.getByTestId('page-content'))
    expect(screen.getByTestId('page-header').contains(root)).toBe(false)
  })

  it('uses one connection signal animation instead of a spinner while recovering', async () => {
    render(
      <ConnectionRecoveryOverlayProvider appIntent={{ kind: 'recovering', title: 'Restoring connection' }}>
        <div className="relative h-80"><ConnectionRecoveryOverlayHost /></div>
      </ConnectionRecoveryOverlayProvider>,
    )

    const overlay = await screen.findByRole('status')
    const loader = overlay.querySelector('[data-connection-recovery-loader="signal"]')
    expect(loader).not.toBeNull()
    expect(loader?.querySelectorAll('[data-connection-signal-bar]')).toHaveLength(5)
    expect(loader?.querySelector('.animate-spin')).toBeNull()
    expect(overlay.querySelector('svg')).toBeNull()
  })

  it('selects the most important child outcome and exposes one retry action', async () => {
    const retry = vi.fn()
    render(
      <ConnectionRecoveryOverlayProvider appIntent={null}>
        <IntentSource intent={{ kind: 'recovering', title: 'Reconnecting' }} />
        <IntentSource intent={{ kind: 'failed', title: 'Connection failed', action: { label: 'Retry', onClick: retry } }} />
        <div className="relative h-80"><ConnectionRecoveryOverlayHost /></div>
      </ConnectionRecoveryOverlayProvider>,
    )

    const overlay = await screen.findByRole('alert')
    expect(screen.getAllByTestId('anytty-connection-recovery-overlay')).toHaveLength(1)
    expect(overlay.textContent).toContain('Connection failed')
    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
    expect(retry).toHaveBeenCalledOnce()
  })

  it('moves the single overlay into the latest modal content host without covering its close action', async () => {
    render(
      <ConnectionRecoveryOverlayProvider appIntent={{ kind: 'failed', title: 'Connection failed' }}>
        <div className="relative h-80" data-testid="background-content">
          <ConnectionRecoveryOverlayHost />
        </div>
        <ModalSurface aria-label="Preview" className="relative h-80" onRequestClose={() => undefined}>
          <header data-testid="preview-header"><button type="button">Close</button></header>
          <div className="relative h-64" data-testid="preview-content">
            <ConnectionRecoveryOverlayHost />
          </div>
        </ModalSurface>
      </ConnectionRecoveryOverlayProvider>,
    )

    const overlay = await screen.findByTestId('anytty-connection-recovery-overlay')
    const root = overlay.closest<HTMLElement>('[data-anytty-connection-overlay-root]')
    expect(screen.getAllByTestId('anytty-connection-recovery-overlay')).toHaveLength(1)
    expect(screen.getByTestId('preview-content').contains(root)).toBe(true)
    expect(screen.getByTestId('background-content').contains(root)).toBe(false)
    expect(screen.getByTestId('preview-header').contains(root)).toBe(false)
    expect(overlay.closest('[inert]')).toBeNull()
    expect(overlay.closest('[aria-hidden="true"]')).toBeNull()
  })
})

function IntentSource({ intent }: { intent: ConnectionRecoveryOverlayIntent | null }) {
  useConnectionRecoveryOverlay(intent)
  return null
}
