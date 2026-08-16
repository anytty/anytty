import { describe, expect, it } from 'vitest'
import { NativeGenerationRecoveryFence } from './NativeGenerationRecoveryFence'

describe('NativeGenerationRecoveryFence', () => {
  it('prevents an older recovery from committing after a newer native epoch starts', () => {
    const fence = new NativeGenerationRecoveryFence()
    const firstAttempt = fence.beginAttempt()
    const secondAttempt = fence.beginNativeEpoch(10)

    expect(secondAttempt).not.toBeNull()
    expect(fence.isCurrent(firstAttempt)).toBe(false)
    expect(fence.claimNativeReadyAttempt(9)).toBeNull()
    expect(fence.claimNativeReadyAttempt(10)).toBe(secondAttempt)
  })

  it('invalidates an in-flight success when the current epoch fails', () => {
    const fence = new NativeGenerationRecoveryFence()
    fence.beginNativeEpoch(20)
    const readyAttempt = fence.claimNativeReadyAttempt(20)
    const failedAttempt = fence.failNativeEpoch(20)

    expect(readyAttempt).not.toBeNull()
    expect(failedAttempt).not.toBeNull()
    expect(fence.isCurrent(readyAttempt!)).toBe(false)
    expect(fence.claimNativeReadyAttempt(20)).toBeNull()
  })

  it('ignores duplicate and stale native events', () => {
    const fence = new NativeGenerationRecoveryFence()
    expect(fence.beginNativeEpoch(30)).not.toBeNull()
    expect(fence.beginNativeEpoch(30)).toBeNull()
    expect(fence.beginNativeEpoch(29)).toBeNull()
    expect(fence.failNativeEpoch(29)).toBeNull()
  })

  it('rejects a late async success after a newer epoch starts', async () => {
    const fence = new NativeGenerationRecoveryFence()
    let resolveFirst!: () => void
    const firstAttempt = fence.beginAttempt()
    const committed: string[] = []
    const firstRecovery = new Promise<void>((resolve) => { resolveFirst = resolve }).then(() => {
      if (fence.isCurrent(firstAttempt)) committed.push('ready')
    })

    fence.beginNativeEpoch(40)
    resolveFirst()
    await firstRecovery

    expect(committed).toEqual([])
  })

  it('does not let a native epoch borrow a newer app or manual attempt', () => {
    const fence = new NativeGenerationRecoveryFence()
    fence.beginNativeEpoch(50)
    fence.beginAttempt()

    expect(fence.claimNativeReadyAttempt(50)).toBeNull()
    expect(fence.failNativeEpoch(50)).toBeNull()
  })
})
