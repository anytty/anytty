import { describe, expect, it } from 'vitest'
import { maximumPastedPairingInputLength, parsePastedPairingInput, PastedPairingInputError } from './pairingInput'

describe('pasted pairing input', () => {
  it.each([
    ['MXP2-Ab_c-123', 'MXP2-Ab_c-123'],
    ["'MXP2-Ab_c-123'", 'MXP2-Ab_c-123'],
    ['anytty://share?payload=Ab_c-123', 'anytty://share?payload=Ab_c-123'],
    ["anytty pair import --id 'device-1' 'MXP2-Ab_c-123'", 'MXP2-Ab_c-123'],
    ["$ /usr/local/bin/anytty pair import --id 'device-1' 'MXP2-Ab_c-123'", 'MXP2-Ab_c-123'],
    ["'C:\\Tools\\anytty.exe' pair import --id 'device-1' 'MXP2-Ab_c-123'", 'MXP2-Ab_c-123'],
  ])('extracts a portable payload from %s', (input, expected) => {
    expect(parsePastedPairingInput(input)).toBe(expected)
  })

  it.each([
    '',
    'MXP2-A',
    'MXP2-not+base64',
    "anytty pair import --id 'device-1' 'MXP2-unfinished",
    "curl https://example.invalid/MXP2-Ab_c-123",
    "other pair import --id 'device-1' 'MXP2-Ab_c-123'",
    "anytty pair import 'MXP2-first' 'MXP2-second'",
    'x'.repeat(maximumPastedPairingInputLength + 1),
  ])('rejects invalid or ambiguous input without evaluating it', (input) => {
    expect(() => parsePastedPairingInput(input)).toThrow(PastedPairingInputError)
  })
})
