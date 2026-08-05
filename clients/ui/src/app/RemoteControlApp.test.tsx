import { act, cleanup, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { anyttyI18n } from '../i18n'
import type { RemoteNetworkRuntime, RemoteRuntimeStorage } from '../core/transport'
import { createMachineStore } from '../state/machineStore'
import { addNativeBackHandler, dispatchNativeBack, NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import { RemoteControlApp, type ExternalPairingAdapter, type ScanPairingCodeOptions } from './RemoteControlApp'

describe('RemoteControlApp accountless product shell', () => {
  beforeEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  afterEach(() => {
    cleanup()
    document.querySelector('[data-testid="english-back-label-decoy"]')?.remove()
    vi.restoreAllMocks()
  })

  it('starts with device pairing and no retired account action', async () => {
    renderApp()

    expect(await screen.findByTestId('anytty-first-use')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Add device' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Sign in to AnyTTY Cloud' })).toBeNull()
  })

  it('does not invent a static Cloud availability state in settings', async () => {
    renderApp()

    await userEvent.click(await screen.findByRole('button', { name: 'Open settings' }))
    expect(screen.getByText('Device access')).toBeTruthy()
    expect(screen.queryByText('Cloud unavailable')).toBeNull()
    expect(screen.queryByRole('heading', { name: 'Account' })).toBeNull()
  })

  it('keeps the settings switch visual size while exposing a 44 pixel touch target', async () => {
    renderApp()

    await userEvent.click(await screen.findByRole('button', { name: 'Open settings' }))
    const toggle = screen.getByRole('button', { name: 'Cursor blink' })
    const track = toggle.firstElementChild

    expect(toggle.className).toContain('h-11')
    expect(toggle.className).toContain('w-12')
    expect(track?.className).toContain('h-8')
    expect(track?.className).toContain('w-12')
  })

  it('keeps the connection recovery retry target at least 44 pixels tall', async () => {
    const onRetryConnectionRecovery = vi.fn(async () => undefined)
    renderApp({
      connectionReady: false,
      connectionRecoveryFailed: true,
      onRetryConnectionRecovery,
    })

    const retry = await screen.findByRole('button', { name: 'Retry' })
    expect(retry.className).toContain('min-h-11')
    await userEvent.click(retry)
    expect(onRetryConnectionRecovery).toHaveBeenCalledOnce()
  })

  it.each([
    ['camera_permission_denied', 'Camera access was denied. Allow camera access in system settings, then try again.'],
    ['camera_not_found', 'No camera was found on this device.'],
    ['camera_start_failed', 'The camera could not start. Close other apps using it, then try again.'],
  ])('keeps the pasted fallback available and distinguishes %s', async (code, message) => {
    renderApp({
      scanPairingCode: vi.fn(async () => {
        throw Object.assign(new Error('raw camera detail'), { code })
      }),
    })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    await userEvent.click(screen.getByRole('button', { name: 'Scan QR with camera' }))

    const alert = await screen.findByRole('alert')
    expect(within(alert).getByText(message)).toBeTruthy()
    expect(alert.textContent).not.toContain('raw camera detail')
    expect(screen.getByRole('textbox', { name: 'Pairing command or pairing code' })).toBeTruthy()
  })

  it('announces pending camera work and prevents duplicate scans', async () => {
    const scan = deferred<string | null>()
    const scanPairingCode = vi.fn(() => scan.promise)
    renderApp({ scanPairingCode })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const cameraButton = screen.getByRole('button', { name: 'Scan QR with camera' }) as HTMLButtonElement
    await userEvent.click(cameraButton)

    expect(cameraButton.disabled).toBe(true)
    expect(screen.getByRole('status').textContent).toBe('Camera is scanning for an AnyTTY QR code...')
    cameraButton.click()
    expect(scanPairingCode).toHaveBeenCalledOnce()

    scan.resolve(null)
    await waitFor(() => expect(cameraButton.disabled).toBe(false))
  })

  it('shows a sanitized scanner-load alert with reload as the only recovery', async () => {
    const scanPairingCode = vi.fn(async () => {
      throw Object.assign(new Error('secret chunk URL and token'), { code: 'scanner_load_failed' })
    })
    renderApp({ scanPairingCode })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    await userEvent.click(screen.getByRole('button', { name: 'Scan QR with camera' }))

    const alert = await screen.findByRole('alert')
    expect(within(alert).getByText('The QR scanner could not be loaded. Reload the application to recover.')).toBeTruthy()
    expect(within(alert).getByRole('button', { name: 'Reload application' })).toBeTruthy()
    expect(alert.textContent).not.toContain('secret chunk URL')
    expect((screen.getByRole('button', { name: 'Scan QR with camera' }) as HTMLButtonElement).disabled).toBe(true)
    expect(scanPairingCode).toHaveBeenCalledOnce()
  })

  it('imports a scanned service QR and saves the resulting endpoint locally', async () => {
    const pairingImport = vi.fn(async () => ({
      machine: { id: 'device-1', name: 'Office Mac', hostname: 'office.local', accessClass: 'local' as const },
    }))
    const scanPairingCode = vi.fn(async () => {
      if (document.activeElement instanceof HTMLElement) document.activeElement.blur()
      return 'MXP2-SCANNED'
    })
    const { storage } = renderApp({
      pairingImport,
      scanPairingCode,
    })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const cameraButton = screen.getByRole('button', { name: 'Scan QR with camera' })
    await userEvent.click(cameraButton)

    expect(pairingImport).toHaveBeenCalledWith('MXP2-SCANNED', undefined)
    expect(createMachineStore({ storage }).listMachines()).toEqual([
      expect.objectContaining({ machineId: 'device-1', name: 'Office Mac', hostname: 'office.local' }),
    ])
    await waitFor(() => expect(cameraButton.isConnected).toBe(false))
    expect(document.activeElement).not.toBe(cameraButton)
  })

  it('extracts a portable claim from the generated pairing command', async () => {
    const pairingImport = vi.fn(async () => ({
      machine: { id: 'device-1', name: 'Office Mac', accessClass: 'local' as const },
    }))
    renderApp({ pairingImport })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const pastedInput = screen.getByRole('textbox', { name: 'Pairing command or pairing code' })
    await userEvent.type(pastedInput, "anytty pair import --id 'device-1' 'MXP2-PASTED_123'")
    expect(screen.getByText(/one-time access credential/i)).toBeTruthy()
    await userEvent.click(screen.getByRole('button', { name: 'Pair using pasted content' }))

    expect(pairingImport).toHaveBeenCalledWith('MXP2-PASTED_123', undefined)
  })

  it('rejects unrelated pasted commands inline without sending their content', async () => {
    const pairingImport = vi.fn(async () => null)
    renderApp({ pairingImport })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const pastedInput = screen.getByRole('textbox', { name: 'Pairing command or pairing code' })
    await userEvent.type(pastedInput, 'curl https://example.invalid MXP2-NOT_RUN')
    await userEvent.click(screen.getByRole('button', { name: 'Pair using pasted content' }))

    expect((await screen.findByRole('alert')).textContent).toContain('No valid pairing code was found.')
    expect(pastedInput.getAttribute('aria-invalid')).toBe('true')
    expect(pairingImport).not.toHaveBeenCalled()
  })

  it('offers pasted pairing as the primary fallback without a camera', async () => {
    renderApp()

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    expect(screen.getByText('Camera scanning is unavailable on this device. Paste a pairing command or pairing code below.')).toBeTruthy()
    expect(screen.getByRole('textbox', { name: 'Pairing command or pairing code' })).toBeTruthy()
    expect(screen.getByText(/guard against information disclosure/i)).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Pair using pasted content' }).className).toContain('anytty-app-primary-button')
  })

  it('ends a failed scanned pairing and allows camera retry', async () => {
    const failure = Object.assign(new Error('sanitized connection failure'), { code: 'unavailable' })
    const scanPairingCode = vi.fn(async () => 'MXP2-TEST')
    renderApp({ pairingImport: async () => { throw failure }, scanPairingCode })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const sheet = screen.getByTestId('anytty-pair-sheet')
    await userEvent.click(within(sheet).getByRole('button', { name: 'Scan QR with camera' }))

    expect(await screen.findByText('Could not connect to this device. Check both devices\' networks and try again.')).toBeTruthy()
    const cameraButton = within(sheet).getByRole('button', { name: 'Scan QR with camera' }) as HTMLButtonElement
    expect(cameraButton.disabled).toBe(false)

    await userEvent.click(within(sheet).getByRole('button', { name: 'Scan QR with camera' }))
    expect(scanPairingCode).toHaveBeenCalledTimes(2)
  })

  it('keeps PairSheet open when one Back event only closes its raw scanner owner', async () => {
    let unregisterScanner = () => {}
    const scanPairingCode = vi.fn((options?: ScanPairingCodeOptions) => new Promise<null>((resolve) => {
      let settled = false
      const finish = () => {
        if (settled) return
        settled = true
        unregisterScanner()
        resolve(null)
      }
      unregisterScanner = addNativeBackHandler(finish, NATIVE_BACK_PRIORITY.NESTED_OVERLAY)
      options?.signal?.addEventListener('abort', finish, { once: true })
    }))
    renderApp({ scanPairingCode })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    await userEvent.click(screen.getByRole('button', { name: 'Scan QR with camera' }))

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    await waitFor(() => expect((screen.getByRole('button', { name: 'Scan QR with camera' }) as HTMLButtonElement).disabled).toBe(false))
    expect(screen.getByTestId('anytty-pair-sheet')).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-pair-sheet')).toBeNull()
    unregisterScanner()
  })

  it('restores camera focus after scanned pairing import fails', async () => {
    const animationFrames: FrameRequestCallback[] = []
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      animationFrames.push(callback)
      return animationFrames.length
    })
    const failure = Object.assign(new Error('sanitized connection failure'), { code: 'unavailable' })
    renderApp({
      pairingImport: async () => { throw failure },
      scanPairingCode: async () => {
        if (document.activeElement instanceof HTMLElement) document.activeElement.blur()
        return 'MXP2-TEST'
      },
    })

    await userEvent.click(await screen.findByRole('button', { name: 'Add device' }))
    const cameraButton = screen.getByRole('button', { name: 'Scan QR with camera' }) as HTMLButtonElement
    await userEvent.click(cameraButton)

    expect(await screen.findByText('Could not connect to this device. Check both devices\' networks and try again.')).toBeTruthy()
    expect(cameraButton.disabled).toBe(false)
    expect(document.activeElement).not.toBe(cameraButton)
    await waitFor(() => expect(animationFrames).toHaveLength(1))
    act(() => { animationFrames.shift()?.(0) })
    expect(document.activeElement).toBe(cameraButton)
  })

  it('refreshes the native device projection from the home toolbar', async () => {
    const onRefreshMachines = vi.fn(async () => undefined)
    renderApp({ onRefreshMachines })

    await userEvent.click(await screen.findByRole('button', { name: 'Refresh devices' }))

    expect(onRefreshMachines).toHaveBeenCalledTimes(1)
    expect(await screen.findByText('Device status updated')).toBeTruthy()
  })

  it('closes the Chinese pairing surface without querying English aria labels', async () => {
    await anyttyI18n.changeLanguage('zh-CN')
    const englishLabelDecoy = document.createElement('button')
    const decoyClick = vi.fn()
    englishLabelDecoy.dataset.testid = 'english-back-label-decoy'
    englishLabelDecoy.setAttribute('aria-label', 'Close pairing')
    englishLabelDecoy.onclick = decoyClick
    document.body.append(englishLabelDecoy)
    renderApp()

    await userEvent.click(await screen.findByRole('button', { name: anyttyI18n.t('machines.scanService') }))
    expect(screen.getByTestId('anytty-pair-sheet')).toBeTruthy()

    act(() => { expect(dispatchNativeBack()).toBe(true) })
    expect(screen.queryByTestId('anytty-pair-sheet')).toBeNull()
    expect(decoyClick).not.toHaveBeenCalled()
    expect(dispatchNativeBack()).toBe(false)
    expect(decoyClick).not.toHaveBeenCalled()
    englishLabelDecoy.remove()
  })
})

function renderApp({
  connectionReady,
  connectionRecoveryFailed,
  pairingImport,
  scanPairingCode,
  onRefreshMachines,
  onRetryConnectionRecovery,
}: {
  connectionReady?: boolean | undefined
  connectionRecoveryFailed?: boolean | undefined
  pairingImport?: ExternalPairingAdapter['import'] | undefined
  scanPairingCode?: ((options?: ScanPairingCodeOptions) => Promise<string | null>) | undefined
  onRefreshMachines?: (() => Promise<void>) | undefined
  onRetryConnectionRecovery?: (() => Promise<void>) | undefined
} = {}) {
  const storage = new MemoryStorage()
  const networkRuntime: RemoteNetworkRuntime = {
    storage,
    queryParam: () => null,
    fetch: async () => new Response('{}', { status: 200 }),
  }
  const externalPairingAdapter: ExternalPairingAdapter = {
    import: pairingImport ?? (async () => null),
    isAuthorized: () => true,
    forget: () => {},
  }
  render(
    <RemoteControlApp
      connectionReady={connectionReady}
      connectionRecoveryFailed={connectionRecoveryFailed}
      externalPairingAdapter={externalPairingAdapter}
      networkRuntime={networkRuntime}
      onRefreshMachines={onRefreshMachines}
      onRetryConnectionRecovery={onRetryConnectionRecovery}
      scanPairingCode={scanPairingCode}
    />,
  )
  return { storage }
}

class MemoryStorage implements RemoteRuntimeStorage {
  private readonly values = new Map<string, string>()
  getItem(key: string): string | null { return this.values.get(key) ?? null }
  removeItem(key: string): void { this.values.delete(key) }
  setItem(key: string, value: string): void { this.values.set(key, value) }
}

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}
