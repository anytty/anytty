export const terminalTextSoftLimitChars = 1_500_000

export function appendTerminalText(current: string, next: string): string {
  return trimTerminalTextToRecentWindow(current + next)
}

export function trimTerminalTextToRecentWindow(text: string, limit = terminalTextSoftLimitChars): string {
  if (limit <= 0) return ''
  if (text.length <= limit) return text
  const start = text.length - limit
  const newline = text.indexOf('\n', start)
  return text.slice(newline >= 0 ? newline + 1 : start)
}
