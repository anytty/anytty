import { describe, expect, it, vi } from 'vitest'
import { holdTerminalFrame } from './terminalFrameHold'

describe('holdTerminalFrame', () => {
  it('keeps a cloned frame for two paints and then removes it', () => {
    const frames: FrameRequestCallback[] = []
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback) => {
      frames.push(callback)
      return frames.length
    })
    const container = document.createElement('div')
    const xterm = document.createElement('div')
    xterm.className = 'xterm'
    const screen = document.createElement('div')
    screen.className = 'xterm-screen'
    screen.textContent = 'stable output'
    xterm.append(screen)
    container.append(xterm)
    document.body.append(container)
    vi.spyOn(container, 'getBoundingClientRect').mockReturnValue(rect(0, 0, 320, 200))
    vi.spyOn(xterm, 'getBoundingClientRect').mockReturnValue(rect(4, 5, 312, 180))
    vi.spyOn(screen, 'getBoundingClientRect').mockReturnValue(rect(12, 17, 300, 160))

    const hold = holdTerminalFrame(container, screen)

    expect(hold).not.toBeNull()
    const overlay = container.querySelector('[data-anytty-terminal-frame-hold]') as HTMLElement | null
    expect(overlay?.parentElement).toBe(xterm)
    expect(overlay?.style.left).toBe('8px')
    expect(overlay?.style.top).toBe('12px')
    expect(overlay?.textContent).toBe('stable output')
    hold?.setTransform('translateY(12px)')
    expect((container.querySelector('[data-anytty-terminal-frame-hold]') as HTMLElement | null)?.style.transform).toBe('translateY(12px)')
    hold?.releaseAfterPaint()
    frames.shift()?.(0)
    expect(container.querySelector('[data-anytty-terminal-frame-hold]')).not.toBeNull()
    frames.shift()?.(16)
    expect(container.querySelector('[data-anytty-terminal-frame-hold]')).toBeNull()
  })
})

function rect(x: number, y: number, width: number, height: number): DOMRect {
  return {
    x,
    y,
    width,
    height,
    top: y,
    left: x,
    right: x + width,
    bottom: y + height,
    toJSON: () => ({}),
  }
}
