import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { ConnectionInfoDialog, loadConnectionPanelState, MachineWorkspace } from './MachineWorkspace'
import '../i18n'

describe('MachineWorkspace connection policy ownership', () => {
  it('does not override the Go-owned persistent policy during initial inventory loading', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const listTerminals = vi.fn(async () => [])

    render(<MachineWorkspace
      api={{
        getStatus: vi.fn(async () => ({
          machine,
          localWeb: { httpUrl: '', rtcOfferUrl: '' },
        })),
        listTerminals,
      }}
      connector={{ connect: vi.fn() }}
      initialMachine={machine}
    />)

    await waitFor(() => expect(listTerminals).toHaveBeenCalledOnce())
    expect(listTerminals.mock.calls[0]?.[0]?.forceRelay).toBeUndefined()
  })

  it('hides a stale bridge error and clears the user-facing failure after recovery', async () => {
    const machine = { machineId: 'studio', name: 'Studio', state: 'online' as const }
    const connector = { connect: vi.fn() }
    const getStatus = vi.fn(async () => ({
      machine,
      localWeb: { httpUrl: '', rtcOfferUrl: '' },
    }))
    const failedApi = {
      getStatus,
      listTerminals: vi.fn(async () => { throw new Error('Go binding bridge disconnected') }),
    }
    const recoveredApi = {
      getStatus,
      listTerminals: vi.fn(async () => []),
    }

    const view = render(<MachineWorkspace api={failedApi} connector={connector} initialMachine={machine} />)
    const failure = await screen.findByTestId('anytty-connection-failure')
    expect(failure.textContent).toContain('Connection interrupted')
    expect(failure.textContent).not.toMatch(/Go binding|bridge/i)

    view.rerender(<MachineWorkspace api={recoveredApi} connector={connector} initialMachine={machine} />)
    await waitFor(() => expect(recoveredApi.listTerminals).toHaveBeenCalledOnce())
    await waitFor(() => expect(screen.queryByTestId('anytty-connection-failure')).toBeNull())
  })
})

describe('ConnectionInfoDialog', () => {
  it('keeps the Go-owned policy editable when the current session is unavailable', async () => {
    const policyState = {
      policy: { route: 'auto', cloud: 'auto', relayTransport: 'auto' } as const,
      available: { direct: false, ssh: false, cloud: false },
      unavailableReasons: { direct: 'route_not_configured', ssh: 'credential_unavailable', cloud: 'cloud_unavailable' },
    }

    const result = await loadConnectionPanelState(
      Promise.reject(new Error('client session is unavailable')),
      Promise.resolve(policyState),
    )

    expect(result.info).toBeNull()
    expect(result.policy).toEqual(policyState)
    expect(result.error).toMatchObject({ message: 'client session is unavailable' })
  })

  it('applies a Cloud route policy and keeps unavailable SSH disabled', async () => {
    const user = userEvent.setup()
    const onApply = vi.fn()
    render(<ConnectionInfoDialog
      info={{ path: 'local', routeKind: 'direct', observedPath: 'direct', connectionId: 'studio:7', machineId: 'studio', relayInUse: false, type: 'p2p', localAddr: '182.138.142.220:41000', localBaseAddr: '192.168.123.168:40000', remoteAddr: '[2001:db8::20]:41121', candidatePairId: 'pair-selected', generation: 7n }}
      loading={false}
      error={null}
      policyState={{ policy: { route: 'auto', cloud: 'auto', relayTransport: 'auto' }, available: { direct: true, ssh: false, cloud: true }, unavailableReasons: { ssh: 'credential_unavailable' } }}
      applying={false}
      onClose={vi.fn()}
      onRefresh={vi.fn()}
      onRetry={vi.fn()}
      onApply={onApply}
      onRestoreAuto={vi.fn()}
    />)

    expect((screen.getByRole('radio', { name: 'SSH tunnel' }) as HTMLInputElement).disabled).toBe(true)
    expect(screen.getByText('182.138.142.220:41000')).toBeTruthy()
    expect(screen.getByText('192.168.123.168:40000')).toBeTruthy()
    expect(screen.getByText('[2001:db8::20]:41121')).toBeTruthy()
    expect(screen.getByText('Credential unavailable')).toBeTruthy()
    await user.click(screen.getByRole('radio', { name: 'AnyTTY Cloud' }))
    await user.click(screen.getByRole('radio', { name: 'Relay only' }))
    await user.click(screen.getByRole('radio', { name: 'TCP only' }))
    await user.click(screen.getByRole('button', { name: 'Apply & reconnect' }))

    expect(onApply).toHaveBeenCalledWith({ route: 'cloud', cloud: 'relay', relayTransport: 'tcp' })
  })

  it('offers retry and Restore Auto for a policy or reconnect failure', async () => {
    const user = userEvent.setup()
    const onRestoreAuto = vi.fn()
    render(<ConnectionInfoDialog
      info={null}
      loading={false}
      error="Direct route is unavailable"
      policyState={{ policy: { route: 'direct', cloud: 'auto', relayTransport: 'auto' }, available: { direct: true, ssh: true, cloud: true }, unavailableReasons: {} }}
      applying={false}
      onClose={vi.fn()}
      onRefresh={vi.fn()}
      onRetry={vi.fn()}
      onApply={vi.fn()}
      onRestoreAuto={onRestoreAuto}
    />)

    expect(screen.getByRole('alert').textContent).toContain('Direct route is unavailable')
    await user.click(screen.getByRole('button', { name: 'Restore Auto' }))
    expect(onRestoreAuto).toHaveBeenCalledOnce()
  })

  it('traps focus, hides background content, closes on Escape, and restores focus', async () => {
    const user = userEvent.setup()
    const trigger = document.createElement('button')
    trigger.textContent = 'Open network settings'
    document.body.appendChild(trigger)
    trigger.focus()
    const onClose = vi.fn()
    const rendered = render(<>
      <button type="button">Background action</button>
      <ConnectionInfoDialog
        info={null}
        loading={false}
        error={null}
        policyState={{ policy: { route: 'auto', cloud: 'auto', relayTransport: 'auto' }, available: { direct: true, ssh: true, cloud: true }, unavailableReasons: {} }}
        applying={false}
        onClose={onClose}
        onRefresh={vi.fn()}
        onRetry={vi.fn()}
        onApply={vi.fn()}
        onRestoreAuto={vi.fn()}
      />
    </>)

    const background = rendered.getByText('Background action')
    expect(background.hasAttribute('inert')).toBe(true)
    const close = rendered.container.querySelector<HTMLButtonElement>('button[aria-label="Close connection and network"]')!
    expect(document.activeElement).toBe(close)
    await user.keyboard('{Shift>}{Tab}{/Shift}')
    const refresh = Array.from(rendered.container.querySelectorAll('button')).find((button) => button.textContent === 'Refresh')
    expect(document.activeElement).toBe(refresh)
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledOnce()

    rendered.unmount()
    expect(document.activeElement).toBe(trigger)
    trigger.remove()
  })
})
