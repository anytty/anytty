import type * as Html5QrcodeModule from 'html5-qrcode'
import { NATIVE_BACK_PRIORITY, addNativeBackHandler, anyttyI18n } from '@anytty/ui'

const qrScannerRootId = 'anytty-camera-qr-scanner'
const qrScannerReaderId = 'anytty-camera-qr-reader'
const qrScannerTitleId = 'anytty-camera-qr-title'
let scannerModulePromise: Promise<typeof Html5QrcodeModule> | null = null
let activeScanPromise: Promise<string | null> | null = null

export interface NativeQrScannerOptions {
  signal?: AbortSignal | undefined
}

export function scanPairingCode(options?: NativeQrScannerOptions): Promise<string | null> {
  if (options?.signal?.aborted) return Promise.resolve(null)
  if (activeScanPromise) return activeScanPromise

  const result = createScanPrompt(options)
  activeScanPromise = result
  const releaseOwnership = () => {
    if (activeScanPromise === result) activeScanPromise = null
  }
  void result.then(releaseOwnership, releaseOwnership)
  return result
}

function createScanPrompt(options?: NativeQrScannerOptions): Promise<string | null> {
  console.info('[anytty:scan] camera scan requested')

  const scannerSize = scannerSquareSize()
  const qrboxSize = Math.min(220, Math.max(140, scannerSize - 32))
  const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  const root = document.createElement('div')
  root.id = qrScannerRootId
  root.className = 'anytty-app-page fixed inset-0 z-[2147483647] flex flex-col items-stretch overflow-x-hidden overflow-y-auto pb-[calc(env(safe-area-inset-bottom)+12px)] pl-[calc(env(safe-area-inset-left)+16px)] pr-[calc(env(safe-area-inset-right)+16px)] pt-[calc(env(safe-area-inset-top)+12px)]'
  root.setAttribute('aria-labelledby', qrScannerTitleId)
  root.setAttribute('aria-modal', 'true')
  root.setAttribute('role', 'dialog')
  root.tabIndex = -1

  const scannerStyle = document.createElement('style')
  scannerStyle.textContent = `
    #${qrScannerReaderId} {
      width: ${scannerSize}px !important;
      height: ${scannerSize}px !important;
      aspect-ratio: 1 / 1 !important;
      overflow: hidden !important;
      border: none !important;
    }
    #${qrScannerReaderId} > div,
    #${qrScannerReaderId}__scan_region,
    #${qrScannerReaderId}__scan_region > div,
    #${qrScannerReaderId} video,
    #${qrScannerReaderId} canvas {
      width: 100% !important;
      height: 100% !important;
    }
    #${qrScannerReaderId} video,
    #${qrScannerReaderId} canvas {
      object-fit: cover !important;
    }
    #${qrScannerReaderId} img {
      display: none !important;
    }
  `

  const header = document.createElement('div')
  header.className = 'flex items-center justify-between gap-3 min-h-[44px]'
  const title = document.createElement('div')
  title.id = qrScannerTitleId
  title.textContent = anyttyI18n.t('scanner.title')
  title.className = 'text-[17px] font-bold tracking-tight text-zinc-900'
  const cancelButton = document.createElement('button')
  cancelButton.type = 'button'
  cancelButton.textContent = anyttyI18n.t('common.cancel')
  cancelButton.className = 'anytty-app-secondary-button min-h-[44px] min-w-[44px] px-4 text-[14px] font-semibold'

  const status = document.createElement('p')
  status.className = 'mt-3 min-h-[20px] text-center text-[14px] font-medium leading-[20px] text-zinc-600'
  status.setAttribute('aria-live', 'polite')
  status.setAttribute('role', 'status')
  status.textContent = anyttyI18n.t('scanner.loading')

  const reader = document.createElement('div')
  reader.id = qrScannerReaderId
  reader.className = 'mt-4 self-center overflow-hidden border border-[var(--anytty-app-line)] bg-black'
  reader.style.width = `${scannerSize}px`
  reader.style.height = `${scannerSize}px`
  reader.style.minWidth = `${scannerSize}px`
  reader.style.minHeight = `${scannerSize}px`
  reader.style.maxWidth = `${scannerSize}px`
  reader.style.maxHeight = `${scannerSize}px`
  reader.hidden = true

  const hint = document.createElement('div')
  hint.textContent = anyttyI18n.t('scanner.hint')
  hint.className = 'mt-4 px-4 text-center text-[13px] font-medium leading-[20px] text-zinc-500'
  hint.hidden = true

  header.append(title, cancelButton)
  root.append(scannerStyle, header, status, reader, hint)
  document.body.append(root)
  const restoreBackground = isolateBackground(root)
  cancelButton.focus()

  let resolveScan!: (value: string | null) => void
  let rejectScan!: (reason: Error) => void
  const result = new Promise<string | null>((resolve, reject) => {
    resolveScan = resolve
    rejectScan = reject
  })
  let promptSettled = false
  let promptUiCleaned = false
  let cameraCleanupStarted = false
  let removeNativeBackHandler = () => {}
  let scanner: InstanceType<typeof Html5QrcodeModule.Html5Qrcode> | null = null
  let startOutcome: Promise<CameraStartOutcome> | null = null

  const cleanupCamera = () => {
    if (cameraCleanupStarted) return
    cameraCleanupStarted = true
    if (!scanner || !startOutcome) return
    const ownedScanner = scanner
    void startOutcome.then(async (outcome) => {
      if (outcome.started) {
        try {
          await ownedScanner.stop()
        } catch {}
      }
      try {
        ownedScanner.clear()
      } catch {}
    }).catch(() => undefined)
  }

  const cleanupPromptUi = (restorePreviousFocus: boolean) => {
    if (promptUiCleaned) return
    promptUiCleaned = true
    cancelButton.disabled = true
    options?.signal?.removeEventListener('abort', onAbort)
    cancelButton.removeEventListener('click', onCancel)
    root.removeEventListener('keydown', onKeyDown)
    removeNativeBackHandler()
    root.remove()
    restoreBackground()
    if (restorePreviousFocus) scheduleFocusRestore(previousFocus)
  }

  const finishPrompt = (value: string | null, errorCode?: ScanErrorCode, restorePreviousFocus = true) => {
    if (promptSettled) return
    promptSettled = true
    cleanupPromptUi(restorePreviousFocus)
    cleanupCamera()

    if (errorCode !== undefined) {
      console.warn('[anytty:scan] camera scan failed', errorCode)
      rejectScan(createScanError(errorCode))
      return
    }
    console.info(value ? '[anytty:scan] QR decoded' : '[anytty:scan] scan cancelled')
    resolveScan(value)
  }

  function onCancel() {
    finishPrompt(null)
  }
  function onAbort() {
    finishPrompt(null)
  }
  function onKeyDown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      event.preventDefault()
      event.stopPropagation()
      finishPrompt(null)
      return
    }
    if (event.key !== 'Tab') return
    const focusable = scannerFocusableElements(root)
    if (focusable.length === 0) {
      event.preventDefault()
      root.focus()
      return
    }
    const first = focusable[0]!
    const last = focusable[focusable.length - 1]!
    const activeElement = document.activeElement
    if (!root.contains(activeElement)) {
      event.preventDefault()
      first.focus()
      return
    }
    if (event.shiftKey && activeElement === first) {
      event.preventDefault()
      last.focus()
      return
    }
    if (!event.shiftKey && activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }

  cancelButton.addEventListener('click', onCancel)
  root.addEventListener('keydown', onKeyDown)
  removeNativeBackHandler = addNativeBackHandler(() => {
    finishPrompt(null)
  }, NATIVE_BACK_PRIORITY.NESTED_OVERLAY)
  options?.signal?.addEventListener('abort', onAbort, { once: true })

  void loadScannerModule().then((module) => {
    if (promptSettled) return

    reader.hidden = false
    hint.hidden = false
    status.textContent = anyttyI18n.t('scanner.starting')

    // Android WebView 的 BarcodeDetector 可能在缺少 GMS provider 时让进程直接崩溃；扫码只使用库内 decoder。
    try {
      scanner = new module.Html5Qrcode(qrScannerReaderId, {
        verbose: false,
        useBarCodeDetectorIfSupported: false,
        formatsToSupport: [module.Html5QrcodeSupportedFormats.QR_CODE],
      })
    } catch {
      finishPrompt(null, 'camera_start_failed')
      return
    }

    let settleStartOutcome!: (outcome: CameraStartOutcome) => void
    startOutcome = new Promise<CameraStartOutcome>((resolve) => {
      settleStartOutcome = resolve
    })
    try {
      void Promise.resolve(scanner.start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: { width: qrboxSize, height: qrboxSize }, aspectRatio: 1.0 },
        (decodedText) => {
          if (promptSettled || !scanner) return
          try {
            scanner.pause(true)
          } catch {}
          finishPrompt(decodedText, undefined, false)
        },
        () => {},
      )).then(
        () => settleStartOutcome({ started: true }),
        (error: unknown) => settleStartOutcome({ started: false, error }),
      )
    } catch (error) {
      settleStartOutcome({ started: false, error })
    }

    void startOutcome.then((outcome) => {
      if (outcome.started) {
        if (!promptSettled) status.textContent = anyttyI18n.t('scanner.scanning')
        return
      }
      finishPrompt(null, classifyCameraStartError(outcome.error))
    })
  }, () => {
    if (!promptSettled) finishPrompt(null, 'scanner_load_failed')
  })

  return result
}

function loadScannerModule(): Promise<typeof Html5QrcodeModule> {
  scannerModulePromise ??= import('html5-qrcode')
  return scannerModulePromise
}

function scannerSquareSize(): number {
  const width = Math.max(0, window.innerWidth || document.documentElement.clientWidth || 360)
  const height = Math.max(0, window.innerHeight || document.documentElement.clientHeight || 640)
  const availableHeight = Math.max(180, height - 340)
  return Math.floor(Math.max(180, Math.min(width * 0.78, availableHeight, 280)))
}

function classifyCameraStartError(error: unknown): ScanErrorCode {
  const detail = error instanceof Error ? `${error.name} ${error.message}` : String(error ?? '')
  if (/NotAllowedError|PermissionDeniedError|Permission denied|SecurityError/i.test(detail)) {
    return 'camera_permission_denied'
  }
  if (/NotFoundError|DevicesNotFoundError|no camera|camera not found/i.test(detail)) {
    return 'camera_not_found'
  }
  return 'camera_start_failed'
}

function createScanError(code: ScanErrorCode): Error & { code: ScanErrorCode } {
  return Object.assign(new Error(`QR scanner error: ${code}`), {
    name: 'NativeQrScannerError',
    code,
  })
}

type CameraStartOutcome = { started: true } | { started: false; error: unknown }
type ScanErrorCode = 'scanner_load_failed' | 'camera_permission_denied' | 'camera_not_found' | 'camera_start_failed'

function isolateBackground(scannerRoot: HTMLElement): () => void {
  const states = Array.from(document.body.children)
    .filter((element): element is HTMLElement => element instanceof HTMLElement && element !== scannerRoot)
    .map((element) => ({
      element,
      ariaHidden: element.getAttribute('aria-hidden'),
      inert: element.getAttribute('inert'),
    }))
  for (const { element } of states) {
    element.setAttribute('aria-hidden', 'true')
    element.setAttribute('inert', '')
  }
  let restored = false
  return () => {
    if (restored) return
    restored = true
    for (const { element, ariaHidden, inert } of states) {
      if (ariaHidden === null) element.removeAttribute('aria-hidden')
      else element.setAttribute('aria-hidden', ariaHidden)
      if (inert === null) element.removeAttribute('inert')
      else element.setAttribute('inert', inert)
    }
  }
}

function scannerFocusableElements(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(
    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])',
  )).filter((element) => (
    !element.matches(':disabled')
      && element.getAttribute('aria-disabled') !== 'true'
      && !element.closest('[inert], [aria-hidden="true"]')
  ))
}

function scheduleFocusRestore(target: HTMLElement | null): void {
  if (!target) return
  window.requestAnimationFrame(() => {
    if (!target.isConnected) return
    if (target.matches(':disabled') || target.getAttribute('aria-disabled') === 'true') return
    if (target.closest('[inert], [aria-hidden="true"]')) return
    target.focus()
  })
}
