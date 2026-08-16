export const maximumPastedPairingInputLength = 12_000

const pairingClaimPattern = /^MXP2-[A-Za-z0-9_-]{3,}$/
const endpointSharePattern = /^anytty:\/\/share\?payload=[A-Za-z0-9_-]+$/

/**
 * Extracts a portable pairing payload without ever evaluating the pasted shell command.
 */
export function parsePastedPairingInput(input: string): string {
  const trimmed = input.trim()
  if (!trimmed || trimmed.length > maximumPastedPairingInputLength) throw new PastedPairingInputError()
  if (isPortablePairingPayload(trimmed)) return trimmed

  let arguments_: string[]
  try {
    arguments_ = parseCommandLine(trimmed)
  } catch {
    throw new PastedPairingInputError()
  }

  if (arguments_[0] === '$') arguments_ = arguments_.slice(1)
  if (arguments_.length === 1 && isPortablePairingPayload(arguments_[0]!)) return arguments_[0]!
  if (!isAnyTTYPairImportCommand(arguments_)) throw new PastedPairingInputError()

  const claims = arguments_.slice(3).filter((argument) => pairingClaimPattern.test(argument))
  if (claims.length !== 1) throw new PastedPairingInputError()
  return claims[0]!
}

function isPortablePairingPayload(value: string): boolean {
  return pairingClaimPattern.test(value) || endpointSharePattern.test(value)
}

function isAnyTTYPairImportCommand(arguments_: readonly string[]): boolean {
  if (arguments_.length < 4) return false
  const executable = arguments_[0]!.replaceAll('\\', '/').split('/').at(-1)?.toLowerCase()
  return (executable === 'anytty' || executable === 'anytty.exe')
    && arguments_[1] === 'pair'
    && arguments_[2] === 'import'
}

export class PastedPairingInputError extends Error {
  constructor() {
    super('pasted pairing input is invalid')
    this.name = 'PastedPairingInputError'
  }
}
import { parseCommandLine } from '../terminal/terminalCreateForm'
