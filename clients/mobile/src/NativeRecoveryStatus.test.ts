import { describe, expect, it } from 'vitest'
import { reduceNativeRecoveryStatus } from './NativeRecoveryStatus'

describe('reduceNativeRecoveryStatus', () => {
  it('keeps a quick automatic recovery silent', () => {
    const checking = reduceNativeRecoveryStatus('ready', {
      type: 'recovery.started',
      visibleImmediately: false,
    })

    expect(checking).toBe('checking')
    expect(reduceNativeRecoveryStatus(checking, { type: 'recovery.succeeded' })).toBe('ready')
  })

  it('shows recovery only after the notice delay elapses', () => {
    const checking = reduceNativeRecoveryStatus('ready', {
      type: 'recovery.started',
      visibleImmediately: false,
    })

    expect(reduceNativeRecoveryStatus(checking, { type: 'recovery.noticeDelayElapsed' })).toBe('recovering')
  })

  it('keeps manual retries and retries after failure visible', () => {
    expect(reduceNativeRecoveryStatus('ready', {
      type: 'recovery.started',
      visibleImmediately: true,
    })).toBe('recovering')
    expect(reduceNativeRecoveryStatus('failed', {
      type: 'recovery.started',
      visibleImmediately: false,
    })).toBe('recovering')
  })

  it('dismisses presentation state when the page returns to the background', () => {
    expect(reduceNativeRecoveryStatus('checking', { type: 'recovery.dismissed' })).toBe('ready')
    expect(reduceNativeRecoveryStatus('failed', { type: 'recovery.dismissed' })).toBe('ready')
  })
})
