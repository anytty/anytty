export interface TerminalFrameHold {
  setTransform(value: string): void
  releaseAfterPaint(): void
  remove(): void
}

/** Keeps the last composited terminal frame visible while xterm rebuilds its buffer. */
export function holdTerminalFrame(container: HTMLElement, screen: HTMLElement): TerminalFrameHold | null {
  if (!screen.isConnected || !container.isConnected) return null

  const xterm = screen.closest('.xterm') as HTMLElement | null
  const host = xterm && container.contains(xterm) ? xterm : container
  const hostRect = host.getBoundingClientRect()
  const screenRect = screen.getBoundingClientRect()
  const width = screenRect.width || screen.clientWidth
  const height = screenRect.height || screen.clientHeight
  if (width <= 0 || height <= 0) return null

  container.querySelectorAll('[data-anytty-terminal-frame-hold]').forEach((element) => element.remove())

  const overlay = document.createElement('div')
  overlay.className = 'anytty-terminal-frame-hold'
  overlay.dataset.anyttyTerminalFrameHold = 'true'
  overlay.setAttribute('aria-hidden', 'true')
  overlay.style.left = `${screenRect.left - hostRect.left}px`
  overlay.style.top = `${screenRect.top - hostRect.top}px`
  overlay.style.width = `${width}px`
  overlay.style.height = `${height}px`

  const clone = screen.cloneNode(true) as HTMLElement
  clone.removeAttribute('id')
  clone.style.position = 'absolute'
  clone.style.inset = '0'
  clone.style.width = `${width}px`
  clone.style.height = `${height}px`
  clone.style.margin = '0'
  copyCanvasFrames(screen, clone)
  overlay.append(clone)
  // Keep the clone under .xterm so xterm.css still positions its canvas layers absolutely.
  host.append(overlay)

  let removed = false
  const remove = () => {
    if (removed) return
    removed = true
    overlay.remove()
  }
  return {
    remove,
    setTransform(value) {
      if (!removed) overlay.style.transform = value
    },
    releaseAfterPaint() {
      if (removed) return
      const view = container.ownerDocument.defaultView
      if (!view?.requestAnimationFrame) {
        remove()
        return
      }
      view.requestAnimationFrame(() => {
        view.requestAnimationFrame(remove)
      })
    },
  }
}

function copyCanvasFrames(source: HTMLElement, clone: HTMLElement): void {
  const sourceCanvases = source.querySelectorAll('canvas')
  const cloneCanvases = clone.querySelectorAll('canvas')
  sourceCanvases.forEach((sourceCanvas, index) => {
    const cloneCanvas = cloneCanvases[index]
    if (!cloneCanvas) return
    cloneCanvas.width = sourceCanvas.width
    cloneCanvas.height = sourceCanvas.height
    try {
      cloneCanvas.getContext('2d')?.drawImage(sourceCanvas, 0, 0)
    } catch {
      // A renderer may own a non-readable GPU buffer; DOM layers still remain available.
    }
  })
}
