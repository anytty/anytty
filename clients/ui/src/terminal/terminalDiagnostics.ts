type TerminalDiagnosticLevel = 'debug' | 'info' | 'warn' | 'error'

export interface TerminalDiagnosticEvent {
  level?: TerminalDiagnosticLevel | undefined
  machineId?: string | undefined
  terminalId?: string | undefined
  connectionId?: string | undefined
  channelLabel?: string | undefined
  details?: Record<string, unknown> | undefined
}

export function logTerminalDiagnostic(event: string, input: TerminalDiagnosticEvent = {}): void {
  try {
    const level = input.level ?? 'debug'
    const method = level === 'error' ? 'error' : level === 'warn' ? 'warn' : level === 'info' ? 'info' : 'debug'
    const metadata: Record<string, unknown> = {}
    if (input.machineId) metadata.machineId = input.machineId
    if (input.terminalId) metadata.terminalId = input.terminalId
    if (input.connectionId) metadata.connectionId = input.connectionId
    if (input.channelLabel) metadata.channelLabel = input.channelLabel
    if (input.details) Object.assign(metadata, input.details)
    console[method](`[anytty:terminal] ${event}`, metadata)
    const nativeWriter = (globalThis as {
      __anyttyWriteNativeDebugLog?: (level: TerminalDiagnosticLevel, tag: string, message: string) => void
    }).__anyttyWriteNativeDebugLog
    if (typeof nativeWriter === 'function') {
      nativeWriter(level, 'TerminalJS', `${event} ${JSON.stringify(metadata)}`)
    }
  } catch {
    // Diagnostics must not affect terminal behavior.
  }
}

export function terminalNow(): number {
  return typeof performance !== 'undefined' && typeof performance.now === 'function'
    ? performance.now()
    : Date.now()
}
