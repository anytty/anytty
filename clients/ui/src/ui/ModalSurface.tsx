import { useEffect, useRef, type ComponentPropsWithoutRef, type KeyboardEvent, type RefObject } from 'react'

const focusableSelector = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  'summary',
  '[tabindex]:not([tabindex="-1"])',
].join(',')

interface IsolatedElementState {
  count: number
  ariaHidden: string | null
  inert: string | null
}

const activeSurfaces = new Set<HTMLElement>()
const isolatedElements = new Map<HTMLElement, IsolatedElementState>()
const pendingFocusTargets: HTMLElement[] = []
let bodyScrollLockCount = 0
let lockedBody: HTMLElement | null = null
let previousBodyOverflow = ''

export interface ModalSurfaceProps extends Omit<ComponentPropsWithoutRef<'section'>, 'role'> {
  onRequestClose: () => void
  initialFocusRef?: RefObject<HTMLElement | null> | undefined
}

export function ModalSurface({ children, initialFocusRef, onKeyDown, onRequestClose, tabIndex = -1, ...props }: ModalSurfaceProps) {
  const surfaceRef = useRef<HTMLElement>(null)

  useEffect(() => {
    const surface = surfaceRef.current
    if (!surface) return
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    activeSurfaces.add(surface)
    const releaseIsolation = isolateOutsideElements(surface)
    const releaseBodyScrollLock = lockBodyScroll()
    if (topmostSurface() === surface) {
      const focusTarget = initialFocusRef?.current ?? focusableElements(surface)[0] ?? surface
      focusWithoutScrolling(focusTarget)
    }
    return () => {
      releaseIsolation()
      releaseBodyScrollLock()
      activeSurfaces.delete(surface)
      restoreFocus(previousFocus)
    }
  }, [initialFocusRef])

  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    onKeyDown?.(event)
    if (event.defaultPrevented) return
    const surface = surfaceRef.current
    if (!surface || topmostSurface() !== surface) return
    if (event.key === 'Escape') {
      event.preventDefault()
      event.stopPropagation()
      onRequestClose()
      return
    }
    if (event.key !== 'Tab') return
    const focusable = focusableElements(surface)
    if (focusable.length === 0) {
      event.preventDefault()
      focusWithoutScrolling(surface)
      return
    }
    const first = focusable[0]!
    const last = focusable[focusable.length - 1]!
    if (event.shiftKey && (document.activeElement === first || !surface.contains(document.activeElement))) {
      event.preventDefault()
      focusWithoutScrolling(last)
    } else if (!event.shiftKey && (document.activeElement === last || !surface.contains(document.activeElement))) {
      event.preventDefault()
      focusWithoutScrolling(first)
    }
  }

  return (
    <section {...props} aria-modal="true" ref={surfaceRef} role="dialog" tabIndex={tabIndex} onKeyDown={handleKeyDown}>
      {children}
    </section>
  )
}

function focusableElements(surface: HTMLElement): HTMLElement[] {
  return Array.from(surface.querySelectorAll<HTMLElement>(focusableSelector))
    .filter((element) => (
      !element.hasAttribute('hidden')
      && element.getAttribute('aria-hidden') !== 'true'
      && element.closest<HTMLElement>('[role="dialog"][aria-modal="true"]') === surface
    ))
}

function collectOutsideElements(surface: HTMLElement): HTMLElement[] {
  const outside: HTMLElement[] = []
  let current: HTMLElement | null = surface
  while (current?.parentElement) {
    for (const sibling of current.parentElement.children) {
      if (sibling !== current && sibling instanceof HTMLElement && sibling.tagName !== 'SCRIPT') outside.push(sibling)
    }
    current = current.parentElement
    if (current === document.body) break
  }
  return outside
}

function isolateOutsideElements(surface: HTMLElement): () => void {
  const outside = collectOutsideElements(surface)
  for (const element of outside) {
    const current = isolatedElements.get(element)
    if (current) {
      current.count += 1
      continue
    }
    isolatedElements.set(element, {
      count: 1,
      ariaHidden: element.getAttribute('aria-hidden'),
      inert: element.getAttribute('inert'),
    })
    element.setAttribute('aria-hidden', 'true')
    element.setAttribute('inert', '')
  }

  let released = false
  return () => {
    if (released) return
    released = true
    for (const element of outside) {
      const current = isolatedElements.get(element)
      if (!current) continue
      current.count -= 1
      if (current.count > 0) continue
      isolatedElements.delete(element)
      restoreAttribute(element, 'aria-hidden', current.ariaHidden)
      restoreAttribute(element, 'inert', current.inert)
    }
  }
}

function lockBodyScroll(): () => void {
  const body = document.body
  if (bodyScrollLockCount === 0) {
    lockedBody = body
    previousBodyOverflow = body.style.overflow
    body.style.overflow = 'hidden'
  }
  bodyScrollLockCount += 1

  let released = false
  return () => {
    if (released) return
    released = true
    bodyScrollLockCount -= 1
    if (bodyScrollLockCount > 0) return
    if (lockedBody) lockedBody.style.overflow = previousBodyOverflow
    lockedBody = null
    previousBodyOverflow = ''
    bodyScrollLockCount = 0
  }
}

function topmostSurface(): HTMLElement | null {
  let topmost: HTMLElement | null = null
  for (const surface of activeSurfaces) {
    if (!surface.isConnected) continue
    if (!topmost || topmost.contains(surface)) {
      topmost = surface
      continue
    }
    if (surface.contains(topmost)) continue
    topmost = surface
  }
  return topmost
}

function restoreFocus(target: HTMLElement | null): void {
  if (target?.isConnected && !pendingFocusTargets.includes(target)) pendingFocusTargets.push(target)
  for (let index = pendingFocusTargets.length - 1; index >= 0; index -= 1) {
    const candidate = pendingFocusTargets[index]!
    if (!candidate.isConnected) {
      pendingFocusTargets.splice(index, 1)
      continue
    }
    if (candidate.closest('[inert]')) continue
    pendingFocusTargets.splice(index, 1)
    focusWithoutScrolling(candidate)
    if (activeSurfaces.size === 0) pendingFocusTargets.length = 0
    return
  }
  if (activeSurfaces.size === 0) pendingFocusTargets.length = 0
}

function focusWithoutScrolling(target: HTMLElement): void {
  target.focus({ preventScroll: true })
}

function restoreAttribute(element: HTMLElement, name: string, value: string | null): void {
  if (value === null) element.removeAttribute(name)
  else element.setAttribute(name, value)
}
