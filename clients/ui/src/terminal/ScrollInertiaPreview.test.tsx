import { act, cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ScrollInertiaPreview } from './ScrollInertiaPreview'

describe('ScrollInertiaPreview', () => {
  let animationFrames: Map<number, FrameRequestCallback>
  let nextFrameId: number
  let now: number

  beforeEach(() => {
    animationFrames = new Map()
    nextFrameId = 1
    now = 0
    vi.spyOn(performance, 'now').mockImplementation(() => now)
    vi.stubGlobal('requestAnimationFrame', vi.fn((callback: FrameRequestCallback) => {
      const id = nextFrameId++
      animationFrames.set(id, callback)
      return id
    }))
    vi.stubGlobal('cancelAnimationFrame', vi.fn((id: number) => {
      animationFrames.delete(id)
    }))
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  function renderPreview(inertia: number) {
    render(<ScrollInertiaPreview inertia={inertia} themeId="anytty-dark" />)
    const viewport = screen.getByTestId('anytty-scroll-inertia-viewport')
    Object.defineProperties(viewport, {
      clientHeight: { configurable: true, value: 300 },
      scrollHeight: { configurable: true, value: 3_960 },
    })
    viewport.scrollTop = 500
    animationFrames.clear()
    return viewport
  }

  it('tracks drags in both directions at pixel precision without line snapping', () => {
    const viewport = renderPreview(0)

    fireEvent.pointerDown(viewport, { button: 0, clientY: 180, isPrimary: true, pointerId: 1 })
    now = 16
    fireEvent.pointerMove(viewport, { clientY: 117, isPrimary: true, pointerId: 1 })
    fireEvent.pointerUp(viewport, { clientY: 117, isPrimary: true, pointerId: 1 })
    expect(viewport.scrollTop).toBe(563)

    fireEvent.pointerDown(viewport, { button: 0, clientY: 117, isPrimary: true, pointerId: 2 })
    now = 32
    fireEvent.pointerMove(viewport, { clientY: 162, isPrimary: true, pointerId: 2 })
    fireEvent.pointerUp(viewport, { clientY: 162, isPrimary: true, pointerId: 2 })
    expect(viewport.scrollTop).toBe(518)
    expect(animationFrames).toHaveLength(0)
  })

  it('continues moving after a fast release when inertia is enabled', () => {
    const viewport = renderPreview(100)

    fireEvent.pointerDown(viewport, { button: 0, clientY: 180, isPrimary: true, pointerId: 2 })
    now = 16
    fireEvent.pointerMove(viewport, { clientY: 100, isPrimary: true, pointerId: 2 })
    now = 20
    fireEvent.pointerUp(viewport, { clientY: 100, isPrimary: true, pointerId: 2 })
    const scrollTopAtRelease = viewport.scrollTop
    const frame = [...animationFrames.values()][0]
    expect(frame).toBeDefined()

    act(() => frame?.(36))

    expect(viewport.scrollTop).toBeGreaterThan(scrollTopAtRelease)
  })

  it('does not create momentum when a drag is already clamped at an edge', () => {
    const viewport = renderPreview(100)
    viewport.scrollTop = 0

    fireEvent.pointerDown(viewport, { button: 0, clientY: 100, isPrimary: true, pointerId: 3 })
    now = 16
    fireEvent.pointerMove(viewport, { clientY: 180, isPrimary: true, pointerId: 3 })
    fireEvent.pointerUp(viewport, { clientY: 180, isPrimary: true, pointerId: 3 })

    expect(viewport.scrollTop).toBe(0)
    expect(animationFrames).toHaveLength(0)
  })
})
