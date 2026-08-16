// @vitest-environment jsdom

import { NATIVE_BACK_PRIORITY, addNativeBackHandler, anyttyI18n, dispatchNativeBack } from '@anytty/ui'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

type ScanPairingCode = typeof import('./nativeQrScanner')['scanPairingCode']

const scannerMock = {
  constructorOptions: null as unknown,
  start: vi.fn(),
  stop: vi.fn(),
  clear: vi.fn(),
  pause: vi.fn(),
  construct: vi.fn(),
  moduleLoads: vi.fn(),
  moduleGate: null as Promise<void> | null,
  moduleRejection: null as unknown,
  success: null as ((value: string) => void) | null,
}

async function scannerModuleMock() {
  scannerMock.moduleLoads()
  if (scannerMock.moduleGate) await scannerMock.moduleGate
  if (scannerMock.moduleRejection) throw scannerMock.moduleRejection
  return {
    Html5QrcodeSupportedFormats: { QR_CODE: 0 },
    Html5Qrcode: class {
      constructor(_id: string, options: unknown) {
        scannerMock.construct()
        scannerMock.constructorOptions = options
      }
      start(_camera: unknown, _config: unknown, success: (value: string) => void) {
        scannerMock.success = success
        return scannerMock.start()
      }
      stop() { return scannerMock.stop() }
      clear() { return scannerMock.clear() }
      pause() { return scannerMock.pause() }
    },
  }
}

describe('native QR scanner ownership', () => {
  let animationFrames: FrameRequestCallback[]
  let scanPairingCode: ScanPairingCode

  beforeEach(async () => {
    vi.resetModules()
    scannerMock.start.mockReset()
    scannerMock.constructorOptions = null
    scannerMock.stop.mockReset().mockResolvedValue(undefined)
    scannerMock.clear.mockReset()
    scannerMock.pause.mockReset()
    scannerMock.construct.mockReset()
    scannerMock.moduleLoads.mockReset()
    scannerMock.moduleGate = null
    scannerMock.moduleRejection = null
    scannerMock.success = null
    vi.doMock('@anytty/ui', () => ({
      NATIVE_BACK_PRIORITY,
      addNativeBackHandler,
      anyttyI18n,
      buttonVariants: () => 'shadcn-button shadcn-button-secondary',
      cn: (...values: string[]) => values.filter(Boolean).join(' '),
    }))
    vi.doMock('html5-qrcode', scannerModuleMock)
    document.body.replaceChildren()
    animationFrames = []
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      animationFrames.push(callback)
      return animationFrames.length
    })
    await anyttyI18n.changeLanguage('en')
    ;({ scanPairingCode } = await import('./nativeQrScanner'))
  })

  afterEach(() => {
    expect(document.getElementById('anytty-camera-qr-scanner')).toBeNull()
    vi.restoreAllMocks()
  })

  const runAnimationFrames = () => {
    for (const callback of animationFrames.splice(0)) callback(0)
  }

  const waitForScannerConstruction = async () => {
    await vi.waitFor(() => expect(scannerMock.construct).toHaveBeenCalledOnce())
  }

  it('configures the decoder for QR codes only', async () => {
    scannerMock.start.mockReturnValue(new Promise<void>(() => {}))
    const result = scanPairingCode()
    await waitForScannerConstruction()
    expect(scannerMock.constructorOptions).toMatchObject({ formatsToSupport: [0] })
    document.querySelector<HTMLButtonElement>('#anytty-camera-qr-scanner button')?.click()
    await expect(result).resolves.toBeNull()
  })

  it('cancels immediately without waiting for a camera start that never settles', async () => {
    const start = deferred<void>()
    scannerMock.start.mockReturnValue(start.promise)
    const previousFocus = document.createElement('button')
    document.body.append(previousFocus)
    previousFocus.focus()
    const underlyingPageBack = vi.fn()
    const unregisterUnderlyingPage = addNativeBackHandler(underlyingPageBack, NATIVE_BACK_PRIORITY.SCANNER)
    const result = scanPairingCode()
    await waitForScannerConstruction()

    expect(dispatchNativeBack()).toBe(true)
    await expect(result).resolves.toBeNull()
    expect(document.getElementById('anytty-camera-qr-scanner')).toBeNull()
    expect(scannerMock.stop).not.toHaveBeenCalled()
    expect(scannerMock.clear).not.toHaveBeenCalled()
    expect(dispatchNativeBack()).toBe(true)
    expect(underlyingPageBack).toHaveBeenCalledOnce()

    runAnimationFrames()
    expect(document.activeElement).toBe(previousFocus)
    unregisterUnderlyingPage()
    expect(dispatchNativeBack()).toBe(false)
  })

  it('stops and clears exact once when camera start resolves after cancellation', async () => {
    const start = deferred<void>()
    scannerMock.start.mockReturnValue(start.promise)
    const result = scanPairingCode()
    await waitForScannerConstruction()

    document.querySelector<HTMLButtonElement>('#anytty-camera-qr-scanner button')?.click()
    await expect(result).resolves.toBeNull()
    expect(scannerMock.stop).not.toHaveBeenCalled()
    expect(scannerMock.clear).not.toHaveBeenCalled()

    start.resolve()
    await start.promise
    await Promise.resolve()
    await Promise.resolve()
    expect(scannerMock.stop).toHaveBeenCalledOnce()
    expect(scannerMock.clear).toHaveBeenCalledOnce()
  })

  it('removes prompt ownership immediately while an already-started camera is still stopping', async () => {
    const stop = deferred<void>()
    scannerMock.start.mockResolvedValue(undefined)
    scannerMock.stop.mockReturnValue(stop.promise)
    const result = scanPairingCode()
    await waitForScannerConstruction()
    await Promise.resolve()

    document.querySelector<HTMLButtonElement>('#anytty-camera-qr-scanner button')?.click()
    await expect(result).resolves.toBeNull()
    expect(document.getElementById('anytty-camera-qr-scanner')).toBeNull()
    expect(dispatchNativeBack()).toBe(false)
    expect(scannerMock.stop).toHaveBeenCalledOnce()
    expect(scannerMock.clear).not.toHaveBeenCalled()

    stop.resolve()
    await stop.promise
    await Promise.resolve()
    expect(scannerMock.clear).toHaveBeenCalledOnce()
  })

  it('clears without stopping when camera start rejects after cancellation', async () => {
    const start = deferred<void>()
    scannerMock.start.mockReturnValue(start.promise)
    const result = scanPairingCode()
    await waitForScannerConstruction()

    document.querySelector<HTMLButtonElement>('#anytty-camera-qr-scanner button')?.click()
    await expect(result).resolves.toBeNull()
    start.reject(new Error('late camera rejection'))
    await start.promise.catch(() => undefined)
    await Promise.resolve()
    await Promise.resolve()

    expect(scannerMock.stop).not.toHaveBeenCalled()
    expect(scannerMock.clear).toHaveBeenCalledOnce()
  })

  it('shares exact-once cleanup across successful decode and late signals', async () => {
    scannerMock.start.mockResolvedValue(undefined)
    const result = scanPairingCode()
    await waitForScannerConstruction()
    await Promise.resolve()
    scannerMock.success?.('MXP2-SCANNED')
    scannerMock.success?.('late-result')

    await expect(result).resolves.toBe('MXP2-SCANNED')
    await Promise.resolve()
    expect(scannerMock.pause).toHaveBeenCalledOnce()
    expect(scannerMock.stop).toHaveBeenCalledOnce()
    expect(scannerMock.clear).toHaveBeenCalledOnce()
    expect(animationFrames).toHaveLength(0)
  })

  it('keeps exact-once cleanup ownership when decode succeeds before start returns', async () => {
    scannerMock.start.mockImplementation(() => {
      scannerMock.success?.('MXP2-SYNCHRONOUS')
      return Promise.resolve()
    })

    const result = scanPairingCode()
    await expect(result).resolves.toBe('MXP2-SYNCHRONOUS')
    await Promise.resolve()
    await Promise.resolve()

    expect(scannerMock.pause).toHaveBeenCalledOnce()
    expect(scannerMock.stop).toHaveBeenCalledOnce()
    expect(scannerMock.clear).toHaveBeenCalledOnce()
  })

  it('uses the same exact-once cleanup when its React owner aborts on unmount', async () => {
    scannerMock.start.mockResolvedValue(undefined)
    const controller = new AbortController()
    const result = scanPairingCode({ signal: controller.signal })
    await waitForScannerConstruction()
    await Promise.resolve()

    controller.abort()
    controller.abort()
    scannerMock.success?.('late-result')

    await expect(result).resolves.toBeNull()
    await Promise.resolve()
    expect(scannerMock.stop).toHaveBeenCalledOnce()
    expect(scannerMock.clear).toHaveBeenCalledOnce()
  })

  it('mounts an embedded scanner without taking over the page or native Back stack', async () => {
    scannerMock.start.mockReturnValue(new Promise<void>(() => {}))
    const background = document.createElement('main')
    const mountElement = document.createElement('div')
    background.append(mountElement)
    document.body.append(background)
    const controller = new AbortController()

    const result = scanPairingCode({ signal: controller.signal, mountElement })
    await waitForScannerConstruction()

    const scannerRoot = document.getElementById('anytty-camera-qr-scanner')!
    expect(scannerRoot.parentElement).toBe(mountElement)
    expect(scannerRoot.getAttribute('role')).toBe('region')
    expect(scannerRoot.querySelector('button')).toBeNull()
    expect(background.hasAttribute('inert')).toBe(false)
    expect(background.hasAttribute('aria-hidden')).toBe(false)
    expect(dispatchNativeBack()).toBe(false)

    controller.abort()
    await expect(result).resolves.toBeNull()
  })

  it('isolates the background, traps Tab, handles Escape, and restores focus after a frame', async () => {
    const background = document.createElement('main')
    const previousFocus = document.createElement('button')
    background.append(previousFocus)
    const preservedBackground = document.createElement('aside')
    preservedBackground.setAttribute('aria-hidden', 'false')
    preservedBackground.setAttribute('inert', 'preserved')
    document.body.append(background, preservedBackground)
    previousFocus.focus()
    scannerMock.start.mockReturnValue(new Promise<void>(() => {}))

    const result = scanPairingCode()
    await waitForScannerConstruction()
    const scannerRoot = document.getElementById('anytty-camera-qr-scanner')!
    const cancel = scannerRoot.querySelector<HTMLButtonElement>('button')!
    const alternate = document.createElement('button')
    alternate.textContent = 'alternate'
    scannerRoot.append(alternate)

    expect(background.getAttribute('aria-hidden')).toBe('true')
    expect(background.hasAttribute('inert')).toBe(true)
    expect(preservedBackground.getAttribute('aria-hidden')).toBe('true')
    expect(preservedBackground.hasAttribute('inert')).toBe(true)
    expect(document.activeElement).toBe(cancel)

    cancel.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(alternate)
    alternate.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }))
    expect(document.activeElement).toBe(cancel)

    previousFocus.disabled = true
    cancel.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    await expect(result).resolves.toBeNull()
    expect(background.hasAttribute('aria-hidden')).toBe(false)
    expect(background.hasAttribute('inert')).toBe(false)
    expect(preservedBackground.getAttribute('aria-hidden')).toBe('false')
    expect(preservedBackground.getAttribute('inert')).toBe('preserved')
    expect(document.activeElement).not.toBe(previousFocus)

    previousFocus.disabled = false
    runAnimationFrames()
    expect(document.activeElement).toBe(previousFocus)
    expect(dispatchNativeBack()).toBe(false)
  })

  it('clears exact once and restores focus after a startup error', async () => {
    const previousFocus = document.createElement('button')
    document.body.append(previousFocus)
    previousFocus.focus()
    scannerMock.start.mockRejectedValue(new Error('camera unavailable'))
    previousFocus.disabled = true

    await expect(scanPairingCode()).rejects.toMatchObject({ code: 'camera_start_failed' })
    await Promise.resolve()
    expect(scannerMock.stop).not.toHaveBeenCalled()
    expect(scannerMock.clear).toHaveBeenCalledOnce()
    expect(document.activeElement).not.toBe(previousFocus)
    previousFocus.disabled = false
    runAnimationFrames()
    expect(document.activeElement).toBe(previousFocus)
    expect(dispatchNativeBack()).toBe(false)
  })

  it('cancels, aborts, and handles native Back immediately while the module import is pending', async () => {
    const moduleGate = deferred<void>()
    scannerMock.moduleGate = moduleGate.promise

    const first = scanPairingCode()
    const duplicate = scanPairingCode()
    expect(duplicate).toBe(first)
    expect(document.querySelector('[role="status"]')?.textContent).toBe('Loading QR scanner...')
    expect(scannerMock.construct).not.toHaveBeenCalled()

    document.querySelector<HTMLButtonElement>('#anytty-camera-qr-scanner button')?.click()
    await expect(first).resolves.toBeNull()

    const controller = new AbortController()
    const aborted = scanPairingCode({ signal: controller.signal })
    controller.abort()
    await expect(aborted).resolves.toBeNull()

    const backedOut = scanPairingCode()
    expect(dispatchNativeBack()).toBe(true)
    await expect(backedOut).resolves.toBeNull()

    moduleGate.resolve()
    await moduleGate.promise
    await Promise.resolve()
    await Promise.resolve()
    expect(scannerMock.moduleLoads).toHaveBeenCalledOnce()
    expect(scannerMock.construct).not.toHaveBeenCalled()
  })

  it('sanitizes and retains an import rejection without retrying the module import', async () => {
    scannerMock.moduleRejection = new Error('secret CDN URL and bearer token')

    const first = scanPairingCode()
    await expect(first).rejects.toMatchObject({
      name: 'NativeQrScannerError',
      code: 'scanner_load_failed',
      message: 'QR scanner error: scanner_load_failed',
    })
    expect(document.body.textContent).not.toContain('secret CDN URL')

    await expect(scanPairingCode()).rejects.toMatchObject({ code: 'scanner_load_failed' })
    expect(scannerMock.moduleLoads).toHaveBeenCalledOnce()
    expect(scannerMock.construct).not.toHaveBeenCalled()
  })
})

function deferred<T>() {
  let resolve!: (value: T | PromiseLike<T>) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}
