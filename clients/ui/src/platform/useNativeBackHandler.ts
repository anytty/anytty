import { useEffect, useLayoutEffect, useRef } from 'react'
import { addNativeBackHandler, type AnyTTYNativeBackHandler } from './nativeBack'

export function useNativeBackHandler(
  handler: AnyTTYNativeBackHandler,
  priority: number,
  enabled = true,
): void {
  const handlerRef = useRef(handler)

  useLayoutEffect(() => {
    handlerRef.current = handler
  }, [handler])

  useEffect(() => {
    if (!enabled) return undefined
    return addNativeBackHandler(() => handlerRef.current(), priority)
  }, [enabled, priority])
}
