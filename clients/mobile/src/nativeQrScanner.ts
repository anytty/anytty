import type * as Html5QrcodeModule from 'html5-qrcode'
import { NATIVE_BACK_PRIORITY, addNativeBackHandler, anyttyI18n, buttonVariants, cn } from '@anytty/ui'

const qrScannerRootId = 'anytty-camera-qr-scanner'
const qrScannerReaderId = 'anytty-camera-qr-reader'
const qrScannerTitleId = 'anytty-camera-qr-title'
let scannerModulePromise: Promise<typeof Html5QrcodeModule> | null = null
let activeScanPromise: Promise<string | null> | null = null

export interface NativeQrScannerOptions {
  signal?: AbortSignal | undefined
  mountElement?: HTMLElement | undefined
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

  const embedded = Boolean(options?.mountElement)
  const scannerSize = scannerSquareSize(options?.mountElement)
  const qrboxSize = Math.min(220, Math.max(140, scannerSize - 32))
  const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  const root = document.createElement('div')
  root.id = qrScannerRootId
  root.className = embedded
    ? 'flex min-h-[320px] w-full flex-col items-center justify-center overflow-hidden bg-[#09090b] px-3 py-3 text-white'
    : 'bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] fixed inset-0 z-[2147483647] flex flex-col items-stretch overflow-x-hidden overflow-y-auto pb-[calc(env(safe-area-inset-bottom)+12px)] pl-[calc(env(safe-area-inset-left)+16px)] pr-[calc(env(safe-area-inset-right)+16px)] pt-[calc(env(safe-area-inset-top)+12px)]'
  if (embedded) {
    root.setAttribute('aria-label', anyttyI18n.t('scanner.title'))
    root.setAttribute('role', 'region')
  } else {
    root.setAttribute('aria-labelledby', qrScannerTitleId)
    root.setAttribute('aria-modal', 'true')
    root.setAttribute('role', 'dialog')
    root.tabIndex = -1
  }

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
      border-radius: 0.5rem !important;
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
  cancelButton.className = cn(buttonVariants({ variant: 'secondary' }), 'min-h-11 min-w-11 px-4 text-[14px] font-semibold')

  const statusRow = document.createElement('div')
  statusRow.className = embedded
    ? 'mt-3 flex min-h-6 items-center justify-center gap-2 text-[#d4d4d8]'
    : 'mt-3 flex min-h-6 items-center justify-center gap-2 text-zinc-600'
  const statusSpinner = document.createElement('span')
  statusSpinner.className = 'h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-current border-r-transparent motion-reduce:animate-none'
  statusSpinner.setAttribute('aria-hidden', 'true')
  const status = document.createElement('p')
  status.className = 'text-center text-[13px] font-medium leading-5'
  status.setAttribute('aria-live', 'polite')
  status.setAttribute('role', 'status')
  status.textContent = anyttyI18n.t('scanner.loading')
  statusRow.append(statusSpinner, status)

  const viewport = document.createElement('div')
  viewport.className = 'relative self-center overflow-hidden rounded-lg bg-black shadow-inner'
  viewport.style.width = `${scannerSize}px`
  viewport.style.height = `${scannerSize}px`
  viewport.hidden = true

  const reader = document.createElement('div')
  reader.id = qrScannerReaderId
  reader.className = 'overflow-hidden rounded-lg bg-black'
  reader.style.width = `${scannerSize}px`
  reader.style.height = `${scannerSize}px`
  reader.style.minWidth = `${scannerSize}px`
  reader.style.minHeight = `${scannerSize}px`
  reader.style.maxWidth = `${scannerSize}px`
  reader.style.maxHeight = `${scannerSize}px`

  const scanFrame = document.createElement('div')
  scanFrame.className = 'pointer-events-none absolute inset-0 flex items-center justify-center'
  const scanFrameBorder = document.createElement('span')
  scanFrameBorder.className = 'block aspect-square h-[72%] rounded-xl border-2 border-white/85 shadow-[0_0_0_999px_rgb(0_0_0/0.18)]'
  scanFrame.append(scanFrameBorder)
  viewport.append(reader, scanFrame)

  const hint = document.createElement('div')
  hint.textContent = anyttyI18n.t('scanner.hint')
  hint.className = 'sr-only'
  hint.hidden = true

  header.append(title, cancelButton)
  if (embedded) root.append(scannerStyle, viewport, statusRow, hint)
  else root.append(scannerStyle, header, viewport, statusRow, hint)
  ;(options?.mountElement ?? document.body).append(root)
  const restoreBackground = embedded ? () => {} : isolateBackground(root)
  if (!embedded) cancelButton.focus()

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
    if (!embedded) cancelButton.disabled = true
    options?.signal?.removeEventListener('abort', onAbort)
    if (!embedded) {
      cancelButton.removeEventListener('click', onCancel)
      root.removeEventListener('keydown', onKeyDown)
    }
    removeNativeBackHandler()
    root.remove()
    restoreBackground()
    if (!embedded && restorePreviousFocus) scheduleFocusRestore(previousFocus)
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

  if (!embedded) {
    cancelButton.addEventListener('click', onCancel)
    root.addEventListener('keydown', onKeyDown)
    removeNativeBackHandler = addNativeBackHandler(() => {
      finishPrompt(null)
    }, NATIVE_BACK_PRIORITY.NESTED_OVERLAY)
  }
  options?.signal?.addEventListener('abort', onAbort, { once: true })

  void loadScannerModule().then((module) => {
    if (promptSettled) return

    viewport.hidden = false
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

function scannerSquareSize(mountElement?: HTMLElement): number {
  const width = Math.max(0, window.innerWidth || document.documentElement.clientWidth || 360)
  const height = Math.max(0, window.innerHeight || document.documentElement.clientHeight || 640)
  const availableHeight = Math.max(180, height - 340)
  const availableWidth = mountElement && mountElement.clientWidth > 0 ? mountElement.clientWidth - 24 : width * 0.78
  return Math.floor(Math.max(180, Math.min(availableWidth, availableHeight, 280)))
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
