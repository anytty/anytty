import { useEffect, useRef } from 'react'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal as XTerm } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import {
  resolveTerminalMomentumProfile,
  resolveTerminalTheme,
  type TerminalScrollInertia,
  type TerminalSettings,
} from './terminalSettings'

export function ScrollInertiaPreview({
  inertia,
  themeId,
  className = 'h-64',
}: {
  inertia: TerminalScrollInertia
  themeId: TerminalSettings['themeId']
  className?: string | undefined
}) {
  const containerRef = useRef<HTMLDivElement | null>(null)
  const inertiaRef = useRef(inertia)
  inertiaRef.current = inertia

  useEffect(() => {
    const container = containerRef.current
    if (!container) return

    const term = new XTerm({
      convertEol: true,
      cursorBlink: false,
      fontSize: 13,
      scrollback: 2000,
      theme: resolveTerminalTheme(themeId),
    })
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(container)
    fitAddon.fit()

    const colorCycles = [
      '\x1b[32m',
      '\x1b[34m',
      '\x1b[33m',
      '\x1b[35m',
      '\x1b[36m',
      '\x1b[31m',
    ]
    const tags = ['info', 'debug', 'warn', 'pkg', 'net', 'error']
    const lines = Array.from({ length: 160 }, (_, index) => {
      const row = String(index).padStart(3, '0')
      const color = colorCycles[index % colorCycles.length]!
      const tag = tags[index % tags.length]!
      return `${color}$ anytty preview ${row}  [${tag}] inertia sample ${index}\x1b[0m`
    })
    term.write(lines.join('\r\n'), () => {
      term.scrollToBottom()
    })

    let disposed = false
    let momentumFrame = 0
    let velocity = 0
    let totalOffset = 0
    let baseViewportY = term.buffer.active.viewportY
    let lastY = 0
    let lastTime = 0
    let touchMoved = false

    const lineHeight = () => Math.max(1, Math.ceil((term.element?.clientHeight ?? 0) / term.rows) || 20)

    const cancelMomentum = () => {
      if (!momentumFrame) return
      cancelAnimationFrame(momentumFrame)
      momentumFrame = 0
    }

    const applyPixels = (px: number) => {
      totalOffset += px
      const desired = baseViewportY + Math.trunc(totalOffset / lineHeight())
      const delta = desired - term.buffer.active.viewportY
      if (delta !== 0) term.scrollLines(delta)
    }

    const handleTouchStart = (event: TouchEvent) => {
      event.preventDefault()
      cancelMomentum()
      totalOffset = 0
      baseViewportY = term.buffer.active.viewportY
      lastY = event.touches[0]?.clientY ?? 0
      lastTime = performance.now()
      velocity = 0
      touchMoved = false
    }

    const handleTouchMove = (event: TouchEvent) => {
      event.preventDefault()
      const touch = event.touches[0]
      if (!touch) return
      const now = performance.now()
      const dt = Math.max(1, now - lastTime)
      const dy = lastY - touch.clientY
      touchMoved ||= Math.abs(dy) > 2
      if (touchMoved) velocity = velocity * 0.3 + ((dy / dt) * 1000) * 0.7
      lastY = touch.clientY
      lastTime = now
      applyPixels(dy)
    }

    const handleTouchEnd = () => {
      if (!touchMoved) return
      const profile = resolveTerminalMomentumProfile(inertiaRef.current)
      if (!profile.enabled || Math.abs(velocity) < 60) return
      let frameTime = performance.now()
      let runningVelocity = velocity
      const step = (now: number) => {
        if (disposed) {
          momentumFrame = 0
          return
        }
        const frameDt = (now - frameTime) / 1000
        frameTime = now
        runningVelocity *= Math.pow(profile.deceleration, frameDt * 60)
        if (Math.abs(runningVelocity) < profile.minimumVelocity) {
          momentumFrame = 0
          return
        }
        applyPixels(runningVelocity * frameDt)
        momentumFrame = requestAnimationFrame(step)
      }
      momentumFrame = requestAnimationFrame(step)
    }

    container.addEventListener('touchstart', handleTouchStart, { passive: false })
    container.addEventListener('touchmove', handleTouchMove, { passive: false })
    container.addEventListener('touchend', handleTouchEnd)
    container.addEventListener('touchcancel', handleTouchEnd)

    const resizeObserver = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(() => {
      try {
        fitAddon.fit()
      } catch {
        // Ignore transient layout races while the drawer is animating.
      }
    })
    resizeObserver?.observe(container)

    return () => {
      disposed = true
      cancelMomentum()
      resizeObserver?.disconnect()
      container.removeEventListener('touchstart', handleTouchStart)
      container.removeEventListener('touchmove', handleTouchMove)
      container.removeEventListener('touchend', handleTouchEnd)
      container.removeEventListener('touchcancel', handleTouchEnd)
      term.dispose()
    }
  }, [themeId])

  return (
    <div
      className={`${className} overflow-hidden rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-terminal-bg)] text-[var(--anytty-terminal-fg)]`}
      data-testid="anytty-scroll-inertia-preview"
      style={{ touchAction: 'none' }}
    >
      <div ref={containerRef} className="h-full w-full" />
    </div>
  )
}
