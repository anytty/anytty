let lastHapticAt = 0
let impactHandler: HapticImpactHandler | null = null

export type HapticPattern = number | number[]
export type HapticImpactHandler = (pattern: HapticPattern) => void | Promise<void>

const HAPTIC_SELECTION_PATTERN = 8
const HAPTIC_IMPACT_PATTERN = 10
const HAPTIC_SUCCESS_PATTERN = 25
const HAPTIC_ERROR_PATTERN = [12, 30, 12]

export function setHapticImpactHandler(handler: HapticImpactHandler | null): () => void {
  impactHandler = handler
  return () => {
    if (impactHandler === handler) impactHandler = null
  }
}

export function haptic(pattern: HapticPattern = HAPTIC_IMPACT_PATTERN): void {
  const now = Date.now()
  if (now - lastHapticAt < 35) return
  lastHapticAt = now

  if (impactHandler) {
    try {
      void Promise.resolve(impactHandler(pattern)).catch(() => vibrate(pattern))
      return
    } catch {
      vibrate(pattern)
      return
    }
  }

  vibrate(pattern)
}

export function hapticSelection(): void {
  haptic(HAPTIC_SELECTION_PATTERN)
}

export function hapticImpact(): void {
  haptic(HAPTIC_IMPACT_PATTERN)
}

export function hapticSuccess(): void {
  haptic(HAPTIC_SUCCESS_PATTERN)
}

export function hapticError(): void {
  haptic(HAPTIC_ERROR_PATTERN)
}

function vibrate(pattern: HapticPattern): void {
  if (typeof navigator === 'undefined') return
  try {
    navigator.vibrate?.(pattern)
  } catch {}
}
