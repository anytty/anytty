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

  it('rejects the picker result when foreground generation replacement fails', async () => {
    const barrier = new NativeForegroundBarrier()
    const result = runAcrossNativePicker(barrier, async () => 'selected')

    await Promise.resolve()
    barrier.finishForeground(new Error('Go client engine could not resume'))

    await expect(result).rejects.toThrow('Go client engine could not resume')
  })
})
