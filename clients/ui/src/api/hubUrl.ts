const hubApiSuffixes = [
  '/api/v1/pairing/claims',
  '/api/v1/sessions/ice',
  '/api/v1/sessions',
] as const

export function normalizeHubBaseUrlCandidate(raw: string | undefined): string | undefined {
  const trimmed = raw?.trim()
  if (!trimmed) return undefined
  const withoutTrailingSlash = trimmed.replace(/\/+$/, '')
  const base = hubApiSuffixes.reduce((current, suffix) => {
    return current.toLowerCase().endsWith(suffix) ? current.slice(0, -suffix.length) : current
  }, withoutTrailingSlash)
  const normalized = base.replace(/\/+$/, '')
  return normalized === '' ? undefined : normalized
}
