export function serviceFetchError(service: string, requestUrl: string, error: unknown): Error {
  if (isAbortError(error)) {
    return error instanceof Error ? error : new Error('Request was cancelled')
  }

  const target = requestTarget(requestUrl)
  const message = errorMessage(error)
  if (isBrowserFetchFailure(message)) {
    return new Error(`Cannot reach ${service} at ${target}. Check that the service is online and that CORS allows this Remote UI origin.`)
  }
  return new Error(`Cannot reach ${service} at ${target}: ${message}`)
}

function isAbortError(error: unknown): boolean {
  return error instanceof DOMException && error.name === 'AbortError'
}

function isBrowserFetchFailure(message: string): boolean {
  const normalized = message.toLowerCase()
  return normalized === 'failed to fetch' ||
    normalized.includes("failed to execute 'fetch'") ||
    normalized.includes('networkerror when attempting to fetch resource') ||
    normalized.includes('load failed')
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) return error.message
  return String(error)
}

function requestTarget(requestUrl: string): string {
  try {
    const url = new URL(requestUrl)
    return url.origin
  } catch {
    return requestUrl
  }
}
