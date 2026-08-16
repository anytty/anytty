export const ANYTTY_NATIVE_KEYBOARD_EVENT = 'anytty:native-keyboard'

export interface AnyTTYNativeKeyboardEventDetail {
  visible: boolean
  keyboardHeight?: number | undefined
}

export type AnyTTYNativeKeyboardHandler = (detail: AnyTTYNativeKeyboardEventDetail) => void

export function dispatchNativeKeyboardEvent(detail: AnyTTYNativeKeyboardEventDetail): void {
  if (typeof window === 'undefined' || typeof window.dispatchEvent !== 'function') return
  window.dispatchEvent(new CustomEvent<AnyTTYNativeKeyboardEventDetail>(ANYTTY_NATIVE_KEYBOARD_EVENT, {
    detail: normalizeNativeKeyboardDetail(detail),
  }))
}

export function addNativeKeyboardListener(handler: AnyTTYNativeKeyboardHandler): () => void {
  if (typeof window === 'undefined' || typeof window.addEventListener !== 'function') return () => {}
  const listener = (event: Event) => {
    const detail = nativeKeyboardDetailFromEvent(event)
    if (!detail) return
    handler(detail)
  }
  window.addEventListener(ANYTTY_NATIVE_KEYBOARD_EVENT, listener)
  return () => window.removeEventListener(ANYTTY_NATIVE_KEYBOARD_EVENT, listener)
}

function nativeKeyboardDetailFromEvent(event: Event): AnyTTYNativeKeyboardEventDetail | null {
  const detail = (event as Event & { detail?: unknown }).detail
  if (!detail || typeof detail !== 'object') return null
  const record = detail as Record<string, unknown>
  if (typeof record.visible !== 'boolean') return null
  return normalizeNativeKeyboardDetail({
    visible: record.visible,
    keyboardHeight: typeof record.keyboardHeight === 'number' ? record.keyboardHeight : undefined,
  })
}

function normalizeNativeKeyboardDetail(detail: AnyTTYNativeKeyboardEventDetail): AnyTTYNativeKeyboardEventDetail {
  const keyboardHeight = typeof detail.keyboardHeight === 'number' && Number.isFinite(detail.keyboardHeight)
    ? Math.max(0, detail.keyboardHeight)
    : undefined
  return {
    visible: detail.visible,
    ...(keyboardHeight !== undefined ? { keyboardHeight } : {}),
  }
}
