import { useEffect, useRef, type CSSProperties, type PointerEvent as ReactPointerEvent } from 'react'
import {
  resolveTerminalMomentumProfile,
  resolveTerminalTheme,
  type TerminalScrollInertia,
  type TerminalSettings,
} from './terminalSettings'

const PREVIEW_LINE_COUNT = 180
const PREVIEW_LEVELS = ['INFO', 'DEBUG', 'NOTICE', 'TRACE'] as const
const PREVIEW_SOURCES = ['session', 'terminal', 'transport', 'shell'] as const
const PREVIEW_MESSAGES = [
  'received output frame',
  'viewport state synchronized',
  'command completed successfully',
  'connection heartbeat acknowledged',
] as const

const PREVIEW_LINES = Array.from({ length: PREVIEW_LINE_COUNT }, (_, index) => {
  const seconds = index % 60
  const minutes = 18 + Math.floor(index / 60)
  return {
    key: index,
    level: PREVIEW_LEVELS[index % PREVIEW_LEVELS.length]!,
    source: PREVIEW_SOURCES[index % PREVIEW_SOURCES.length]!,
    text: PREVIEW_MESSAGES[index % PREVIEW_MESSAGES.length]!,
    time: `14:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`,
  }
})

interface DragState {
  pointerId: number
  lastY: number
  lastTime: number
  velocity: number
}

function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function ScrollInertiaPreview({
  inertia,
  themeId,
  className = 'h-64',
}: {
  inertia: TerminalScrollInertia
  themeId: TerminalSettings['themeId']
  className?: string | undefined
}) {
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const dragRef = useRef<DragState | null>(null)
  const momentumFrameRef = useRef(0)
  const inertiaRef = useRef(inertia)
  inertiaRef.current = inertia

  const theme = resolveTerminalTheme(themeId)
  const previewStyle = {
    backgroundColor: theme.background,
    color: theme.foreground,
  } satisfies CSSProperties
  const viewportStyle = {
    scrollbarColor: `${theme.brightBlack ?? theme.blue ?? '#71717a'} ${theme.background}`,
    touchAction: 'none',
  } satisfies CSSProperties

  const cancelMomentum = () => {
    if (!momentumFrameRef.current) return
    window.cancelAnimationFrame(momentumFrameRef.current)
    momentumFrameRef.current = 0
  }

  const scrollByPixels = (pixels: number): boolean => {
    const viewport = viewportRef.current
    if (!viewport || pixels === 0) return false
    const before = viewport.scrollTop
    const maxScrollTop = Math.max(0, viewport.scrollHeight - viewport.clientHeight)
    viewport.scrollTop = Math.min(maxScrollTop, Math.max(0, before + pixels))
    return viewport.scrollTop === before
  }

  const finishDrag = (event: ReactPointerEvent<HTMLDivElement>, cancelled: boolean) => {
    const drag = dragRef.current
    if (!drag || drag.pointerId !== event.pointerId) return
    dragRef.current = null
    event.currentTarget.classList.remove('cursor-grabbing')
    if (event.currentTarget.hasPointerCapture?.(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    if (cancelled || prefersReducedMotion() || performance.now() - drag.lastTime > 80 || Math.abs(drag.velocity) <= 60) return

    let velocity = drag.velocity
    let lastFrameTime = performance.now()
    const step = (now: number) => {
      const profile = resolveTerminalMomentumProfile(inertiaRef.current)
      if (!profile.enabled) {
        momentumFrameRef.current = 0
        return
      }
      const frameTime = Math.min(0.032, Math.max(0, (now - lastFrameTime) / 1000))
      lastFrameTime = now
      velocity *= Math.pow(profile.deceleration, frameTime * 60)
      if (Math.abs(velocity) < profile.minimumVelocity || scrollByPixels(velocity * frameTime)) {
        momentumFrameRef.current = 0
        return
      }
      momentumFrameRef.current = window.requestAnimationFrame(step)
    }
    const profile = resolveTerminalMomentumProfile(inertiaRef.current)
    if (profile.enabled) momentumFrameRef.current = window.requestAnimationFrame(step)
  }

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      const viewport = viewportRef.current
      if (!viewport) return
      viewport.scrollTop = (viewport.scrollHeight - viewport.clientHeight) * 0.58
    })
    return () => {
      window.cancelAnimationFrame(frame)
      if (momentumFrameRef.current) window.cancelAnimationFrame(momentumFrameRef.current)
    }
  }, [])

  return (
    <div
      className={`${className} flex overflow-hidden rounded-md border border-[var(--anytty-app-line)] font-mono shadow-inner`}
      data-testid="anytty-scroll-inertia-preview"
      style={previewStyle}
    >
      <div
        ref={viewportRef}
        className="min-h-0 min-w-0 flex-1 cursor-grab select-none overflow-y-scroll overscroll-contain"
        data-testid="anytty-scroll-inertia-viewport"
        style={viewportStyle}
        onPointerCancel={(event) => finishDrag(event, true)}
        onPointerDown={(event) => {
          if (!event.isPrimary || event.button !== 0) return
          cancelMomentum()
          dragRef.current = {
            pointerId: event.pointerId,
            lastY: event.clientY,
            lastTime: performance.now(),
            velocity: 0,
          }
          event.currentTarget.classList.add('cursor-grabbing')
          event.currentTarget.setPointerCapture?.(event.pointerId)
        }}
        onPointerMove={(event) => {
          const drag = dragRef.current
          if (!drag || drag.pointerId !== event.pointerId) return
          event.preventDefault()
          const now = performance.now()
          const elapsed = Math.max(1, now - drag.lastTime)
          const requestedPixels = drag.lastY - event.clientY
          const viewport = viewportRef.current
          const before = viewport?.scrollTop ?? 0
          scrollByPixels(requestedPixels)
          const appliedPixels = (viewport?.scrollTop ?? before) - before
          const instantVelocity = (appliedPixels / elapsed) * 1000
          drag.velocity = drag.velocity * 0.3 + instantVelocity * 0.7
          drag.lastY = event.clientY
          drag.lastTime = now
        }}
        onPointerUp={(event) => finishDrag(event, false)}
      >
        <div aria-hidden="true" className="min-w-0 py-2 text-[12px] leading-[22px]">
          {PREVIEW_LINES.map((line) => (
            <div className="flex h-[22px] min-w-0 gap-2 px-3" key={line.key}>
              <span className="shrink-0 opacity-45">{line.time}</span>
              <span
                className="w-[4.5rem] shrink-0 font-semibold"
                style={{ color: line.level === 'INFO' ? theme.green : line.level === 'NOTICE' ? theme.yellow : theme.cyan }}
              >
                {line.level}
              </span>
              <span className="min-w-0 truncate opacity-80">[{line.source}] {line.text} #{String(line.key + 1).padStart(4, '0')}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
