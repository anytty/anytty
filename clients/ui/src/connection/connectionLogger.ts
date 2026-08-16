export type ConnectionLogLevel = 'debug' | 'info' | 'warn' | 'error'

export interface ConnectionLogEvent {
  scope: 'orchestrator' | 'hub' | 'browser_webrtc'
  event: string
  level?: ConnectionLogLevel | undefined
  machineId?: string | undefined
  terminalId?: string | undefined
  path?: string | undefined
  hubUrl?: string | undefined
  sessionId?: string | undefined
  message?: string | undefined
  details?: Record<string, unknown> | undefined
}

export interface ConnectionLogger {
  log(event: ConnectionLogEvent): void
}

export const consoleConnectionLogger: ConnectionLogger = {
  log(event) {
    const level = event.level ?? 'debug'
    const method = level === 'error' ? 'error' : level === 'warn' ? 'warn' : level === 'info' ? 'info' : 'debug'
    const message = event.message ? ` ${event.message}` : ''
    const metadata = compactLogMetadata(event)
    console[method](`[anytty:${event.scope}] ${event.event}${message}`, metadata)
    if (event.event.endsWith('_timeout') && event.details) {
      console[method](`[anytty:${event.scope}] ${event.event}_json ${JSON.stringify(metadata)}`)
    }
  },
}

export function logConnectionEvent(logger: ConnectionLogger | undefined, event: ConnectionLogEvent): void {
  try {
    logger?.log(event)
  } catch {
    // Diagnostics must not affect connection behavior.
  }
}

function compactLogMetadata(event: ConnectionLogEvent): Record<string, unknown> {
  const metadata: Record<string, unknown> = {}
  if (event.machineId) metadata.machineId = event.machineId
  if (event.terminalId) metadata.terminalId = event.terminalId
  if (event.path) metadata.path = event.path
  if (event.hubUrl) metadata.hubUrl = event.hubUrl
  if (event.sessionId) metadata.sessionId = event.sessionId
  if (event.details) Object.assign(metadata, event.details)
  return metadata
}
