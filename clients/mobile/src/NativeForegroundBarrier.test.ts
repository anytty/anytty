import { describe, expect, it, vi } from 'vitest'
import { NativeForegroundBarrier, runAcrossNativePicker } from './NativeForegroundBarrier'

describe('NativeForegroundBarrier', () => {
  it('holds a picker result until the replacement generation is ready', async () => {
    const barrier = new NativeForegroundBarrier()
    const settled = vi.fn()

    const result = runAcrossNativePicker(barrier, async () => ({ uri: 'content://selected' }))
    void result.then(settled)
    await Promise.resolve()
    await Promise.resolve()

    expect(settled).not.toHaveBeenCalled()

    barrier.finishForeground()
    await expect(result).resolves.toEqual({ uri: 'content://selected' })
  })

  it('rejects a picker continuation whose original native intent was stopped while foreground was pending', async () => {
    const barrier = new NativeForegroundBarrier()
    const validateContinuation = vi.fn(async () => {
      throw Object.assign(new Error('Stopped by user'), { code: 'user_stopped' })
    })
    const result = runAcrossNativePicker(
      barrier,
      async () => ({ uri: 'content://selected' }),
      validateContinuation,
    )
    await Promise.resolve()
    await Promise.resolve()

    expect(validateContinuation).not.toHaveBeenCalled()
    barrier.finishForeground()

    await expect(result).rejects.toMatchObject({ code: 'user_stopped' })
    expect(validateContinuation).toHaveBeenCalledOnce()
  })

  it('rejects the picker result when foreground generation replacement fails', async () => {
    const barrier = new NativeForegroundBarrier()
    const result = runAcrossNativePicker(barrier, async () => 'selected')

    await Promise.resolve()
    barrier.finishForeground(new Error('Go client engine could not resume'))

    await expect(result).rejects.toThrow('Go client engine could not resume')
  })

  it('releases a picker-owned barrier when launching the picker fails', async () => {
    const barrier = new NativeForegroundBarrier()

    await expect(runAcrossNativePicker(barrier, async () => {
      throw new Error('picker unavailable')
    })).rejects.toThrow('picker unavailable')

    await expect(barrier.wait()).resolves.toBeUndefined()
  })

  it('does not release a barrier reinforced by a later lifecycle event', async () => {
    const barrier = new NativeForegroundBarrier()
    const result = runAcrossNativePicker(barrier, async () => {
      barrier.markBackground()
      throw new Error('picker interrupted')
    })

    await expect(result).rejects.toThrow('picker interrupted')
    const settled = vi.fn()
    void barrier.wait().then(settled)
    await Promise.resolve()
    expect(settled).not.toHaveBeenCalled()

    barrier.finishForeground()
    await barrier.wait()
  })

  it('lets an abandoned business operation cancel its foreground wait', async () => {
    const barrier = new NativeForegroundBarrier()
    const controller = new AbortController()
    barrier.markBackground()

    const waiting = barrier.wait(controller.signal)
    controller.abort(new Error('consumer disposed'))

    await expect(waiting).rejects.toThrow('consumer disposed')
    barrier.finishForeground()
    await expect(barrier.wait()).resolves.toBeUndefined()
  })
})
