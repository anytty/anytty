import { describe, expect, it } from 'vitest'
import { anyttyI18n } from '../i18n'
import { connectionFailurePresentation, connectionFailureReason, isAuthorizationConnectionError } from './connectionErrorPresentation'

describe('connection error presentation', () => {
  it.each([
    'Go binding bridge authentication timed out',
    'client session timed out',
    'deadline exceeded while opening session',
  ])('projects timeouts without leaking implementation details: %s', (detail) => {
    const result = connectionFailurePresentation(new Error(detail), anyttyI18n.t)
    expect(result.reason).toBe('timeout')
    expect(`${result.title} ${result.message}`).not.toMatch(/go binding|bridge|client session|deadline/i)
  })

  it.each([
    'Go binding bridge connection failed',
    'Go binding bridge authentication failed',
    'application session is unavailable',
  ])('uses a safe fallback for runtime failures: %s', (detail) => {
    const result = connectionFailurePresentation(new Error(detail), anyttyI18n.t)
    expect(result.reason).toBe('internal')
    expect(`${result.title} ${result.message}`).not.toMatch(/go binding|bridge|generation|session|handle/i)
  })

  it('uses phone connectivity as the highest-priority cause', () => {
    const result = connectionFailurePresentation(new Error('anything'), anyttyI18n.t, { phoneOnline: false })
    expect(result.reason).toBe('phone_offline')
    expect(result.retryable).toBe(true)
  })

  it.each(['unavailable', 'daemon_blocked', 'daemon_deleted', 'cancelled'])('honors explicit non-retryable metadata for %s', (code) => {
    const source = Object.assign(new Error('typed failure'), { code, retryable: false })
    expect(connectionFailurePresentation(source, anyttyI18n.t).retryable).toBe(false)
  })

  it('recognizes authorization failures without exposing the source text', () => {
    const source = Object.assign(new Error('stored session token is invalid'), { code: 'unauthenticated' })
    const result = connectionFailurePresentation(source, anyttyI18n.t)
    expect(result.reason).toBe('authorization')
    expect(result.requiresPairing).toBe(true)
    expect(result.message).not.toContain('session token')
  })

  it('trusts a stable authorization code even when the message mentions an internal session', () => {
    const source = Object.assign(new Error('client session authorization failed'), { code: 'unauthenticated' })
    expect(connectionFailureReason(source)).toBe('authorization')
  })

  it.each([
    ['device_identity_mismatch', 'identity_mismatch'],
    ['route_unavailable', 'device_unavailable'],
    ['unavailable', 'device_unavailable'],
    ['timeout', 'timeout'],
  ] as const)('recognizes stable string error codes: %s', (code, reason) => {
    expect(connectionFailureReason(code)).toBe(reason)
  })

  it('keeps cancellations distinct from failures', () => {
    expect(connectionFailureReason(Object.assign(new Error('aborted'), { code: 'cancelled' }))).toBe('cancelled')
    expect(connectionFailureReason(new Error('native session generation changed while connecting'))).toBe('cancelled')
    expect(connectionFailureReason(new Error('stale session handle 42'))).toBe('cancelled')
  })

  it('does not infer pairing from a generic forbidden error', () => {
    expect(connectionFailureReason(Object.assign(new Error('policy denied'), { code: 'forbidden' }))).toBe('internal')
  })

  it.each([
    ['Relay traffic quota is exhausted; Direct, P2P, and SSH remain available', 'relay_quota'],
    ['Relay concurrency is full; keep the existing connection or use Direct, P2P, or SSH', 'relay_concurrency'],
  ] as const)('presents commercial Relay limits without suggesting retry or pairing', (message, reason) => {
    const source = Object.assign(new Error(message), { code: 'resource_exhausted' })
    const result = connectionFailurePresentation(source, anyttyI18n.t)
    expect(result).toMatchObject({ reason, retryable: false, requiresPairing: false })
  })

  it.each([
    ['relay_quota_exhausted', 'relay_quota'],
    ['relay_concurrency_exhausted', 'relay_concurrency'],
    ['relay_not_in_plan', 'relay_not_in_plan'],
    ['subscription_inactive', 'subscription_inactive'],
    ['relay_region_unavailable', 'relay_region_unavailable'],
  ] as const)('uses stable Cloud code %s without parsing its message', (code, reason) => {
    const source = Object.assign(new Error('opaque localized detail'), { code })
    expect(connectionFailurePresentation(source, anyttyI18n.t)).toMatchObject({ reason, requiresPairing: false })
  })

  it('tells App users that the device owner must resolve an inactive subscription', () => {
    const source = Object.assign(new Error('opaque detail'), { code: 'subscription_inactive' })
    const result = connectionFailurePresentation(source, anyttyI18n.t)
    expect(result).toMatchObject({ reason: 'subscription_inactive', retryable: false, requiresPairing: false })
    expect(result.message).toBe(anyttyI18n.t('errors.subscriptionInactive'))
    expect(anyttyI18n.t('errors.subscriptionInactive', { lng: 'zh-CN' })).toContain('设备所有者')
    expect(anyttyI18n.t('errors.subscriptionInactive', { lng: 'zh-CN' })).toContain('Cloud 控制台')
    expect(anyttyI18n.t('errors.subscriptionInactive', { lng: 'en' })).toContain('device owner')
    expect(anyttyI18n.t('errors.subscriptionInactive', { lng: 'en' })).toContain('Cloud console')
  })

  it.each([
    { code: 'daemon_blocked', reason: 'daemon_blocked', retryable: true, requiresPairing: false },
    { code: 'daemon_deleted', reason: 'daemon_deleted', retryable: true, requiresPairing: true },
  ] as const)('presents $code lifecycle recovery', ({ code, reason, retryable, requiresPairing }) => {
    const source = Object.assign(new Error('Cloud lifecycle state'), { code })
    const result = connectionFailurePresentation(source, anyttyI18n.t)
    expect(result).toMatchObject({ reason, retryable, requiresPairing })
    expect(isAuthorizationConnectionError(source)).toBe(false)
  })
})
