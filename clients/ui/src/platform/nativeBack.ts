export type AnyTTYNativeBackHandler = () => void

export const NATIVE_BACK_PRIORITY = {
  ROOT: 10,
  WORKSPACE: 20,
  FILE_MANAGER: 25,
  TRANSFER: 30,
  SCANNER: 40,
  NESTED_OVERLAY: 50,
} as const

interface NativeBackHandlerEntry {
  id: number
  priority: number
  handler: AnyTTYNativeBackHandler
}

let nextNativeBackHandlerId = 1
const nativeBackHandlers: NativeBackHandlerEntry[] = []

export function addNativeBackHandler(handler: AnyTTYNativeBackHandler, priority = 0): () => void {
  const entry = {
    id: nextNativeBackHandlerId++,
    priority,
    handler,
  }
  nativeBackHandlers.push(entry)
  return () => {
    const index = nativeBackHandlers.indexOf(entry)
    if (index >= 0) nativeBackHandlers.splice(index, 1)
  }
}

export function dispatchNativeBack(): boolean {
  const entry = nativeBackHandlers.reduce<NativeBackHandlerEntry | undefined>((selected, candidate) => {
    if (!selected) return candidate
    if (candidate.priority > selected.priority) return candidate
    if (candidate.priority === selected.priority && candidate.id > selected.id) return candidate
    return selected
  }, undefined)
  if (!entry) return false

  try {
    entry.handler()
  } catch (err) {
    globalThis.setTimeout(() => {
      throw err
    }, 0)
  }
  return true
}
