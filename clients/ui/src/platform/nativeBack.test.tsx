import { cleanup, render } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { NATIVE_BACK_PRIORITY, addNativeBackHandler, dispatchNativeBack } from './nativeBack'
import { useNativeBackHandler } from './useNativeBackHandler'

const unregisterHandlers: Array<() => void> = []

afterEach(() => {
  cleanup()
  for (const unregister of unregisterHandlers.splice(0).reverse()) unregister()
})

describe('native back handler registry', () => {
  it('dispatches by priority and LIFO order within one priority', () => {
    const calls: string[] = []
    register(() => { calls.push('low') }, NATIVE_BACK_PRIORITY.ROOT)
    register(() => { calls.push('high-old') }, NATIVE_BACK_PRIORITY.TRANSFER)
    const unregisterNewest = register(() => { calls.push('high-new') }, NATIVE_BACK_PRIORITY.TRANSFER)

    expect(dispatchNativeBack()).toBe(true)
    expect(calls).toEqual(['high-new'])

    unregisterNewest()
    expect(dispatchNativeBack()).toBe(true)
    expect(calls).toEqual(['high-new', 'high-old'])
  })

  it('never calls an unregistered handler', () => {
    const removed = vi.fn()
    const unregisterRemoved = register(removed, NATIVE_BACK_PRIORITY.ROOT)
    unregisterRemoved()

    expect(dispatchNativeBack()).toBe(false)
    expect(removed).not.toHaveBeenCalled()
  })

  it('consumes at the selected handler even when it is a no-op', () => {
    const lower = vi.fn()
    const top = vi.fn()
    register(lower, NATIVE_BACK_PRIORITY.WORKSPACE)
    register(top, NATIVE_BACK_PRIORITY.NESTED_OVERLAY)

    expect(dispatchNativeBack()).toBe(true)
    expect(top).toHaveBeenCalledOnce()
    expect(lower).not.toHaveBeenCalled()
  })

  it('orders nested overlay, scanner, transfer, file manager, workspace, then root', () => {
    const consumed: string[] = []
    const entries = [
      ['root', NATIVE_BACK_PRIORITY.ROOT],
      ['workspace', NATIVE_BACK_PRIORITY.WORKSPACE],
      ['file-manager', NATIVE_BACK_PRIORITY.FILE_MANAGER],
      ['transfer', NATIVE_BACK_PRIORITY.TRANSFER],
      ['scanner', NATIVE_BACK_PRIORITY.SCANNER],
      ['nested', NATIVE_BACK_PRIORITY.NESTED_OVERLAY],
    ] as const
    const unregisterByName = new Map<string, () => void>()
    for (const [name, priority] of entries) {
      unregisterByName.set(name, register(() => {
        consumed.push(name)
        unregisterByName.get(name)?.()
      }, priority))
    }

    for (let index = 0; index < entries.length; index += 1) {
      expect(dispatchNativeBack()).toBe(true)
    }
    expect(consumed).toEqual(['nested', 'scanner', 'transfer', 'file-manager', 'workspace', 'root'])
    expect(dispatchNativeBack()).toBe(false)
  })

  it('fails closed when the selected handler throws', () => {
    const lower = vi.fn()
    const failure = new Error('handler failed')
    const setTimeoutSpy = vi.spyOn(globalThis, 'setTimeout').mockImplementation(() => 0 as ReturnType<typeof setTimeout>)
    register(lower, NATIVE_BACK_PRIORITY.ROOT)
    register(() => { throw failure }, NATIVE_BACK_PRIORITY.TRANSFER)

    expect(dispatchNativeBack()).toBe(true)
    expect(lower).not.toHaveBeenCalled()
    expect(setTimeoutSpy).toHaveBeenCalledOnce()
    setTimeoutSpy.mockRestore()
  })

  it('keeps one registration across rerenders and invokes the latest closure', () => {
    const calls: string[] = []
    const view = render(<NativeBackHarness value="first" onBack={(value) => calls.push(value)} />)
    view.rerender(<NativeBackHarness value="second" onBack={(value) => calls.push(value)} />)

    expect(dispatchNativeBack()).toBe(true)
    expect(calls).toEqual(['second'])

    view.unmount()
    expect(dispatchNativeBack()).toBe(false)
    expect(calls).toEqual(['second'])
  })
})

function NativeBackHarness({ value, onBack }: { value: string; onBack: (value: string) => void }) {
  useNativeBackHandler(() => {
    onBack(value)
  }, NATIVE_BACK_PRIORITY.WORKSPACE)
  return null
}

function register(handler: () => void, priority: number): () => void {
  const unregister = addNativeBackHandler(handler, priority)
  unregisterHandlers.push(unregister)
  return unregister
}
