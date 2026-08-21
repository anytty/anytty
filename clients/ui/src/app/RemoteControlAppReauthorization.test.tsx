import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { anyttyI18n } from '../i18n'
import type { RemoteNetworkRuntime, RemoteRuntimeStorage } from '../core/transport'
import { RemoteControlApp, type ExternalPairingAdapter } from './RemoteControlApp'

const workspaceLifecycle = vi.hoisted(() => ({ nextInstance: 0 }))

vi.mock('./MachineWorkspace', async () => {
  const { useState } = await import('react')
  return {
    MachineWorkspace: ({ onNeedsReauthorization }: { onNeedsReauthorization?: (machineId: string) => void }) => {
      const [instance] = useState(() => ++workspaceLifecycle.nextInstance)
      return (
        <div>
          <span data-testid="workspace-instance">{instance}</span>
          <button type="button" onClick={() => onNeedsReauthorization?.('device-1')}>
            Simulate authorization failure
          </button>
        </div>
      )
    },
  }
})

describe('RemoteControlApp reauthorization', () => {
  beforeEach(async () => {
    workspaceLifecycle.nextInstance = 0
    await anyttyI18n.changeLanguage('en')
  })

  afterEach(() => cleanup())

  it('preserves native endpoint credentials while requesting a new pairing code', async () => {
    const storage = new MemoryStorage()
    const networkRuntime: RemoteNetworkRuntime = {
      storage,
      queryParam: () => null,
      fetch: async () => new Response('{}', { status: 200 }),
    }
    const forget = vi.fn()
    const externalPairingAdapter: ExternalPairingAdapter = {
      import: async () => ({
        machine: { id: 'device-1', name: 'Test device', accessClass: 'cloud' },
      }),
      isAuthorized: () => true,
      forget,
    }
    const scanPairingCode = vi.fn()
      .mockResolvedValueOnce('MXP2-TEST')
      .mockResolvedValueOnce(null)

    render(
      <RemoteControlApp
        externalPairingAdapter={externalPairingAdapter}
        networkRuntime={networkRuntime}
        scanPairingCode={scanPairingCode}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Simulate authorization failure' }))

    expect(await screen.findByTestId('anytty-pair-sheet')).toBeTruthy()
    expect(screen.getByText('This device needs fresh authorization. Pair it again.')).toBeTruthy()
    expect(scanPairingCode).toHaveBeenCalledTimes(2)
    expect(forget).not.toHaveBeenCalled()
  })

  it('recreates the machine workspace after fresh authorization succeeds', async () => {
    const storage = new MemoryStorage()
    const networkRuntime: RemoteNetworkRuntime = {
      storage,
      queryParam: () => null,
      fetch: async () => new Response('{}', { status: 200 }),
    }
    const externalPairingAdapter: ExternalPairingAdapter = {
      import: async () => ({
        machine: { id: 'device-1', name: 'Test device', accessClass: 'cloud' },
      }),
      isAuthorized: () => true,
      forget: async () => {},
    }
    const scanPairingCode = vi.fn()
      .mockResolvedValueOnce('MXP2-FIRST')
      .mockResolvedValueOnce('MXP2-FRESH')

    render(
      <RemoteControlApp
        externalPairingAdapter={externalPairingAdapter}
        networkRuntime={networkRuntime}
        scanPairingCode={scanPairingCode}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const initialInstance = Number((await screen.findByTestId('workspace-instance')).textContent)
    await userEvent.click(screen.getByRole('button', { name: 'Simulate authorization failure' }))

    await waitFor(() => expect(scanPairingCode).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(screen.queryByTestId('anytty-pair-sheet')).toBeNull())
    expect(Number(screen.getByTestId('workspace-instance').textContent)).toBeGreaterThan(initialInstance)
  })
})

class MemoryStorage implements RemoteRuntimeStorage {
  private readonly values = new Map<string, string>()
  getItem(key: string): string | null { return this.values.get(key) ?? null }
  removeItem(key: string): void { this.values.delete(key) }
  setItem(key: string, value: string): void { this.values.set(key, value) }
}
