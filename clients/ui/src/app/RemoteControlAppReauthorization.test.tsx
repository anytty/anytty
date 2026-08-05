import { cleanup, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { anyttyI18n } from '../i18n'
import type { RemoteNetworkRuntime, RemoteRuntimeStorage } from '../core/transport'
import { RemoteControlApp, type ExternalPairingAdapter } from './RemoteControlApp'

vi.mock('./MachineWorkspace', () => ({
  MachineWorkspace: ({ onNeedsReauthorization }: { onNeedsReauthorization?: (machineId: string) => void }) => (
    <button type="button" onClick={() => onNeedsReauthorization?.('device-1')}>
      Simulate authorization failure
    </button>
  ),
}))

describe('RemoteControlApp reauthorization', () => {
  beforeEach(async () => {
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

    render(
      <RemoteControlApp
        externalPairingAdapter={externalPairingAdapter}
        networkRuntime={networkRuntime}
        scanPairingCode={async () => 'MXP2-TEST'}
      />,
    )

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const initialSheet = screen.getByTestId('anytty-pair-sheet')
    await userEvent.click(within(initialSheet).getByRole('button', { name: 'Scan QR with camera' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Simulate authorization failure' }))

    expect(await screen.findByTestId('anytty-pair-sheet')).toBeTruthy()
    expect(screen.getByText('This device needs fresh authorization. Pair it again.')).toBeTruthy()
    expect(forget).not.toHaveBeenCalled()
  })
})

class MemoryStorage implements RemoteRuntimeStorage {
  private readonly values = new Map<string, string>()
  getItem(key: string): string | null { return this.values.get(key) ?? null }
  removeItem(key: string): void { this.values.delete(key) }
  setItem(key: string, value: string): void { this.values.set(key, value) }
}
