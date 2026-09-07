import { act, cleanup, render } from '@testing-library/react'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { BrandMotion } from './BrandMotion'

const runtime = vi.hoisted(() => ({
  options: {} as Record<string, () => void>,
  play: vi.fn(), pause: vi.fn(), cleanup: vi.fn(), resizeDrawingSurfaceToCanvas: vi.fn(),
}))
vi.mock('@rive-app/webgl2', () => ({
  RuntimeLoader: { setWasmUrl: vi.fn(), setWasmFallbackUrl: vi.fn() },
  Rive: class {
    constructor(options: Record<string, () => void>) { runtime.options = options }
    play = runtime.play
    pause = runtime.pause
    cleanup = runtime.cleanup
    resizeDrawingSurfaceToCanvas = runtime.resizeDrawingSurfaceToCanvas
  },
}))
let intersect: (entries: { isIntersecting: boolean }[]) => void
let reduce: EventTarget & { matches: boolean }
beforeEach(() => {
  vi.clearAllMocks()
  reduce = Object.assign(new EventTarget(), { matches: false })
  vi.stubGlobal('matchMedia', () => reduce)
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect() {} })
  vi.stubGlobal('IntersectionObserver', class {
    constructor(callback: typeof intersect) { intersect = callback }
    observe() {} disconnect() {}
  })
})
afterEach(() => { cleanup(); vi.unstubAllGlobals() })

it('loads local Rive, preserves a fallback, pauses offscreen and cleans up', async () => {
  const view = render(<BrandMotion scene="connecting" />)
  const canvas = view.container.querySelector('canvas')!
  const poster = view.container.querySelector('img')!
  expect(canvas.hidden).toBe(true)
  await act(async () => {})
  act(() => { intersect([{ isIntersecting: true }]); runtime.options.onLoad!() })
  expect(canvas.hidden).toBe(false)
  expect(poster.hidden).toBe(true)
  expect(runtime.play).toHaveBeenCalledWith('connecting')
  act(() => intersect([{ isIntersecting: false }]))
  expect(runtime.pause).toHaveBeenCalled()
  act(() => { reduce.matches = true; reduce.dispatchEvent(new Event('change')) })
  expect(canvas.hidden).toBe(true)
  expect(poster.hidden).toBe(false)
  view.unmount()
  expect(runtime.cleanup).toHaveBeenCalledOnce()
})

it('decorative artwork stops after one cycle without repeating on reentry', async () => {
  render(<BrandMotion decorative />)
  await act(async () => {})
  act(() => { intersect([{ isIntersecting: true }]); runtime.options.onLoad!(); runtime.options.onLoop!() })
  const count = runtime.play.mock.calls.length
  act(() => { intersect([{ isIntersecting: false }]); intersect([{ isIntersecting: true }]) })
  expect(runtime.play).toHaveBeenCalledTimes(count)
  expect(runtime.pause).toHaveBeenCalled()
})

it('keeps artwork visible when the animation fails to load', async () => {
  const view = render(<BrandMotion />)
  await act(async () => {})
  act(() => runtime.options.onLoadError!())
  expect(view.container.querySelector('img')!.hidden).toBe(false)
  expect(view.container.querySelector('canvas')!.hidden).toBe(true)
})
