import { Capacitor } from '@capacitor/core'
import { Keyboard, KeyboardStyle } from '@capacitor/keyboard'
import { StatusBar, Style } from '@capacitor/status-bar'
import { useEffect, useRef } from 'react'

type Rgba = { r: number; g: number; b: number; a: number }

export function useNativeStatusBarSync(): void {
  const lastAppliedRef = useRef<{ color: string; style: Style; keyboardStyle: KeyboardStyle } | null>(null)

  useEffect(() => {
    if (!Capacitor.isNativePlatform()) return undefined

    let frame = 0
    let disposed = false

    const apply = () => {
      frame = 0
      if (disposed) return

      const color = detectTopSurfaceColor()
      const colorHex = rgbaToHex(color)
      const style = isDarkColor(color) ? Style.Dark : Style.Light
      const keyboardStyle = isDarkColor(color) ? KeyboardStyle.Dark : KeyboardStyle.Light
      const last = lastAppliedRef.current
      if (last?.style !== style) void StatusBar.setStyle({ style }).catch(() => {})
      if (last?.color !== colorHex) void StatusBar.setBackgroundColor({ color: colorHex }).catch(() => {})
      if (Capacitor.getPlatform() === 'ios' && last?.keyboardStyle !== keyboardStyle) {
        void Keyboard.setStyle({ style: keyboardStyle }).catch(() => {})
      }
      lastAppliedRef.current = { color: colorHex, style, keyboardStyle }
    }

    const schedule = () => {
      if (frame !== 0 || disposed) return
      frame = window.requestAnimationFrame(apply)
    }

    void StatusBar.setOverlaysWebView({ overlay: false }).catch(() => {})
    schedule()

    const observer = new MutationObserver(schedule)
    observer.observe(document.documentElement, {
      attributes: true,
      childList: true,
      subtree: true,
      attributeFilter: ['class', 'style', 'hidden', 'aria-hidden'],
    })

    window.addEventListener('resize', schedule)
    window.addEventListener('orientationchange', schedule)
    window.addEventListener('scroll', schedule, true)
    document.addEventListener('click', schedule, true)
    document.addEventListener('transitionend', schedule, true)
    document.addEventListener('animationend', schedule, true)

    return () => {
      disposed = true
      if (frame !== 0) window.cancelAnimationFrame(frame)
      observer.disconnect()
      window.removeEventListener('resize', schedule)
      window.removeEventListener('orientationchange', schedule)
      window.removeEventListener('scroll', schedule, true)
      document.removeEventListener('click', schedule, true)
      document.removeEventListener('transitionend', schedule, true)
      document.removeEventListener('animationend', schedule, true)
    }
  }, [])
}

function detectTopSurfaceColor(): Rgba {
  const x = Math.max(0, Math.floor(window.innerWidth / 2))
  const candidates = [
    document.elementFromPoint(x, 1),
    document.querySelector('[role="dialog"]'),
    document.body,
    document.documentElement,
  ].filter((element): element is Element => Boolean(element))

  for (const element of candidates) {
    const color = resolveElementSurfaceColor(element)
    if (color.a > 0.01) return color
  }
  return { r: 255, g: 255, b: 255, a: 1 }
}

function resolveElementSurfaceColor(element: Element): Rgba {
  const colors: Rgba[] = []
  let current: Element | null = element
  while (current) {
    const color = parseCssColor(window.getComputedStyle(current).backgroundColor)
    if (color && color.a > 0.01) colors.push(color)
    current = current.parentElement
  }

  let resolved: Rgba = { r: 255, g: 255, b: 255, a: 1 }
  for (let index = colors.length - 1; index >= 0; index -= 1) {
    resolved = blend(colors[index]!, resolved)
  }
  return resolved
}

function parseCssColor(value: string): Rgba | null {
  const normalized = value.trim()
  if (!normalized || normalized === 'transparent') return null
  const parts = normalized.match(/[\d.]+/g)?.map(Number)
  if (!parts || parts.length < 3) return null
  return {
    r: clampColor(parts[0]!),
    g: clampColor(parts[1]!),
    b: clampColor(parts[2]!),
    a: parts.length >= 4 ? clampAlpha(parts[3]!) : 1,
  }
}

function blend(foreground: Rgba, background: Rgba): Rgba {
  const alpha = foreground.a + background.a * (1 - foreground.a)
  if (alpha <= 0) return { r: 0, g: 0, b: 0, a: 0 }
  return {
    r: Math.round((foreground.r * foreground.a + background.r * background.a * (1 - foreground.a)) / alpha),
    g: Math.round((foreground.g * foreground.a + background.g * background.a * (1 - foreground.a)) / alpha),
    b: Math.round((foreground.b * foreground.a + background.b * background.a * (1 - foreground.a)) / alpha),
    a: clampAlpha(alpha),
  }
}

function isDarkColor(color: Rgba): boolean {
  const srgb = [color.r, color.g, color.b].map((value) => {
    const channel = value / 255
    return channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4
  })
  const luminance = 0.2126 * srgb[0]! + 0.7152 * srgb[1]! + 0.0722 * srgb[2]!
  return luminance < 0.45
}

function rgbaToHex(color: Rgba): string {
  return `#${hex(color.r)}${hex(color.g)}${hex(color.b)}`
}

function hex(value: number): string {
  return clampColor(value).toString(16).padStart(2, '0')
}

function clampColor(value: number): number {
  return Math.max(0, Math.min(255, Math.round(Number.isFinite(value) ? value : 0)))
}

function clampAlpha(value: number): number {
  return Math.max(0, Math.min(1, Number.isFinite(value) ? value : 1))
}
