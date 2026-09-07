import { useEffect, useRef } from 'react'
import type { Rive } from '@rive-app/webgl2'
import mascotUrl from './assets/anytty-mascot.riv?url'
import fallbackUrl from './assets/anytty-mascot.png'
import wasmUrl from '@rive-app/webgl2/rive.wasm?url'
import { cn } from '../ui/utils'

export type MascotScene = 'connecting' | 'processing' | 'history' | 'success' |
  'failure' | 'welcome' | 'running' | 'searching' | 'wake'

/** Decorative artwork only; the surrounding workflow owns status announcements. */
export function BrandMotion({ scene = 'processing', decorative = false, className }: {
  scene?: MascotScene; decorative?: boolean; className?: string
}) {
  const root = useRef<HTMLSpanElement>(null)
  const canvas = useRef<HTMLCanvasElement>(null)
  const fallback = useRef<HTMLImageElement>(null)
  useEffect(() => {
    if (typeof window.matchMedia !== 'function' || typeof ResizeObserver === 'undefined' ||
        typeof IntersectionObserver === 'undefined') return
    const element = root.current!
    const surface = canvas.current!
    const poster = fallback.current!
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)')
    let disposed = false
    let visible = false
    let ready = false
    let played = false
    let player: Rive | undefined
    const update = () => {
      if (!ready || disposed) return
      if (reduce.matches) {
        player?.pause(); surface.hidden = true; poster.hidden = false
      } else {
        surface.hidden = false; poster.hidden = true
        player?.resizeDrawingSurfaceToCanvas()
        if (visible && !document.hidden && !(decorative && played)) player?.play(scene)
        else player?.pause()
      }
    }
    const resize = new ResizeObserver(() => player?.resizeDrawingSurfaceToCanvas())
    const intersection = new IntersectionObserver(([entry]) => {
      visible = entry?.isIntersecting ?? false; update()
    })
    resize.observe(element); intersection.observe(element)
    reduce.addEventListener('change', update)
    document.addEventListener('visibilitychange', update)
    void import('@rive-app/webgl2').then(({ Rive, RuntimeLoader }) => {
      if (disposed) return
      RuntimeLoader.setWasmUrl(wasmUrl)
      RuntimeLoader.setWasmFallbackUrl(null)
      player = new Rive({ canvas: surface, src: mascotUrl, animations: scene,
        autoplay: false, shouldDisableRiveListeners: true,
        onLoad: () => { ready = true; player?.resizeDrawingSurfaceToCanvas(); update() },
        onLoop: () => { if (decorative) { played = true; update() } },
        onStop: () => { if (decorative) played = true },
        onLoadError: () => { surface.hidden = true; poster.hidden = false },
      })
    }).catch(() => { /* Keep the local poster if the runtime cannot load. */ })
    return () => {
      disposed = true; ready = false; resize.disconnect(); intersection.disconnect()
      reduce.removeEventListener('change', update)
      document.removeEventListener('visibilitychange', update)
      player?.cleanup(); surface.hidden = true; poster.hidden = false
    }
  }, [scene, decorative])
  return <span ref={root} aria-hidden="true" data-brand-motion={scene}
    className={cn('relative inline-block h-20 w-20 shrink-0', className)}>
    <img ref={fallback} src={fallbackUrl} alt="" width="513" height="546" className="absolute inset-0 h-full w-full object-contain" />
    <canvas ref={canvas} hidden className="absolute inset-0 h-full w-full" />
  </span>
}
