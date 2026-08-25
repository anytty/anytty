import type { TFunction } from 'i18next'

export type ConnectionFailureReason =
  | 'phone_offline'
  | 'timeout'
  | 'device_unavailable'
  | 'service_unavailable'
  | 'authorization'
  | 'identity_mismatch'
  | 'entitlement'
  | 'relay_not_in_plan'
  | 'subscription_inactive'
  | 'relay_region_unavailable'
  | 'relay_quota'
  | 'relay_concurrency'
  | 'daemon_blocked'
  | 'daemon_deleted'
  | 'cancelled'
  | 'internal'

export interface ConnectionFailurePresentation {
  reason: ConnectionFailureReason
  title: string
  message: string
  retryable: boolean
  requiresPairing: boolean
}

export function connectionFailureReason(error: unknown, phoneOnline = true): ConnectionFailureReason {
  if (!phoneOnline) return 'phone_offline'

  const code = connectionErrorCode(error)
  const detail = connectionErrorDetail(error)
  if (
    code === 'cancelled' ||
    /\b(?:cancelled|canceled|aborted|superseded)\b/i.test(detail) ||
    /(?:native|binding|session) generation changed|stale (?:client )?session/i.test(detail)
  ) return 'cancelled'
  if (code === 'daemon_deleted') return 'daemon_deleted'
  if (code === 'daemon_blocked') return 'daemon_blocked'
  if (code === 'relay_concurrency_exhausted') return 'relay_concurrency'
  if (code === 'relay_quota_exhausted') return 'relay_quota'
  if (code === 'relay_not_in_plan') return 'relay_not_in_plan'
  if (code === 'subscription_inactive') return 'subscription_inactive'
  if (code === 'relay_region_unavailable') return 'relay_region_unavailable'
  if (/relay concurrency/i.test(detail)) return 'relay_concurrency'
  if (/relay (?:traffic )?quota|quota (?:is )?exhausted/i.test(detail)) return 'relay_quota'
  if (code === 'resource_exhausted' && /relay/i.test(detail)) return 'relay_quota'
  if (code === 'entitlement_denied' || /\bentitlement\b/i.test(detail)) return 'entitlement'
  if (code === 'device_identity_mismatch' || /\b(?:device identity|identity mismatch|fingerprint mismatch)\b/i.test(detail)) return 'identity_mismatch'
  const explicitAuthorizationCode = code === 'login_required' ||
    code === 'unauthenticated' ||
    code === 'capability_invalid' ||
    code === 'capability_expired' ||
    code === 'authorization_revoked'
  if (explicitAuthorizationCode) return 'authorization'
  if (isInternalRuntimeDetail(detail)) {
    if (code === 'timeout' || code === 'deadline_exceeded' || /(?:timed? out|deadline exceeded)/i.test(detail)) return 'timeout'
    return 'internal'
  }
  if (
    /^(?:auth|unauthenticated|capability_invalid|capability_expired|authorization_revoked|scope_invalid)$/i.test(detail) ||
    /(?:authentication failed|stored session token is invalid|invalid session token|pair this machine again|\bunauthorized\b)/i.test(detail)
  ) return 'authorization'
  if (code === 'timeout' || code === 'deadline_exceeded' || /(?:timed? out|deadline exceeded)/i.test(detail)) return 'timeout'
  if (
    code === 'route_unavailable' ||
    code === 'not_found' ||
    /(?:connection refused|host unreachable|network unreachable|no route|routes? unavailable|target unavailable|device unavailable)/i.test(detail)
  ) return 'device_unavailable'
  if (code === 'temporary' || /(?:cloud|relay|hub|signaling).*(?:unavailable|failed)|service unavailable/i.test(detail)) return 'service_unavailable'
  if (code === 'unavailable') return 'device_unavailable'
  return 'internal'
}

export function connectionFailurePresentation(
  error: unknown,
  t: TFunction,
  options: { phoneOnline?: boolean } = {},
): ConnectionFailurePresentation {
  const reason = connectionFailureReason(error, options.phoneOnline ?? true)
  const presentation = (() => {
    switch (reason) {
    case 'phone_offline':
      return failure(reason, t('errors.phoneOfflineTitle'), t('errors.phoneOffline'), true)
    case 'timeout':
      return failure(reason, t('errors.connectionTimeoutTitle'), t('errors.connectionTimeout'), true)
    case 'device_unavailable':
      return failure(reason, t('errors.deviceUnavailableTitle'), t('errors.deviceUnavailable'), true)
    case 'service_unavailable':
      return failure(reason, t('errors.serviceUnavailableTitle'), t('errors.serviceUnavailable'), true)
    case 'authorization':
      return failure(reason, t('errors.authorizationTitle'), t('errors.pairAgain'), false, true)
    case 'identity_mismatch':
      return failure(reason, t('errors.identityMismatchTitle'), t('errors.identityMismatch'), false, true)
    case 'entitlement':
      return failure(reason, t('errors.connectionProblemTitle'), t('errors.relayEntitlementDenied'), false)
    case 'relay_not_in_plan':
      return failure(reason, t('errors.relayNotInPlanTitle'), t('errors.relayNotInPlan'), false)
    case 'subscription_inactive':
      return failure(reason, t('errors.subscriptionInactiveTitle'), t('errors.subscriptionInactive'), false)
    case 'relay_region_unavailable':
      return failure(reason, t('errors.relayRegionUnavailableTitle'), t('errors.relayRegionUnavailable'), false)
    case 'relay_quota':
      return failure(reason, t('errors.relayQuotaTitle'), t('errors.relayQuotaExhausted'), false)
    case 'relay_concurrency':
      return failure(reason, t('errors.relayConcurrencyTitle'), t('errors.relayConcurrencyExhausted'), false)
    case 'daemon_blocked':
      return failure(reason, t('errors.daemonBlockedTitle'), t('errors.daemonBlocked'), true)
    case 'daemon_deleted':
      return failure(reason, t('errors.daemonDeletedTitle'), t('errors.daemonDeleted'), true, true)
    case 'cancelled':
      return failure(reason, t('errors.connectionProblemTitle'), t('errors.connectionCancelled'), true)
    default:
      return failure(reason, t('errors.connectionProblemTitle'), t('errors.connectionInterrupted'), true)
    }
  })()
  return connectionErrorRetryable(error) === false
    ? { ...presentation, retryable: false }
    : presentation
}

export function connectionErrorDisplayMessage(error: unknown, t: TFunction, phoneOnline = true): string {
  return connectionFailurePresentation(error, t, { phoneOnline }).message
}

export function isAuthorizationConnectionError(error: unknown): boolean {
  const reason = connectionFailureReason(error)
  return reason === 'authorization' || reason === 'identity_mismatch'
}

export function isCancelledConnectionError(error: unknown): boolean {
  return connectionFailureReason(error) === 'cancelled'
}

function failure(
  reason: ConnectionFailureReason,
  title: string,
  message: string,
  retryable: boolean,
  requiresPairing = false,
): ConnectionFailurePresentation {
  return { reason, title, message, retryable, requiresPairing }
}

function connectionErrorCode(error: unknown): string {
  if (typeof error === 'string') {
    const normalized = error.trim().toLowerCase()
    return /^[a-z][a-z0-9_]*$/.test(normalized) ? normalized : ''
  }
  if (!error || typeof error !== 'object' || !('code' in error)) return ''
  const code = (error as { code?: unknown }).code
  return typeof code === 'string' ? code.trim().toLowerCase() : ''
}

function connectionErrorRetryable(error: unknown): boolean | undefined {
  if (!error || typeof error !== 'object' || !('retryable' in error)) return undefined
  const retryable = (error as { retryable?: unknown }).retryable
  return typeof retryable === 'boolean' ? retryable : undefined
}

function connectionErrorDetail(error: unknown): string {
  if (typeof error === 'string') return error.trim()
  if (error instanceof Error) return `${error.name} ${error.message}`.trim()
  return ''
}

function isInternalRuntimeDetail(detail: string): boolean {
  return /(?:go binding|bridge|client session|application session|native generation|stale session|runtime|jni|handle)/i.test(detail)
}
