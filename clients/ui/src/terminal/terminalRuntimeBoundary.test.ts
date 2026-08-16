import { describe, expect, expectTypeOf, it } from 'vitest'
import terminalSource from './Terminal.tsx?raw'
import hookSource from './useTerminalSession.tsx?raw'
import type { TerminalProps } from './Terminal'
import type { RtcSession } from '../core/transport'
import type { ProtoClientSession } from '../core/protoClientSession'
import { shouldRecoverTerminalChannel } from './useTerminalSession'

describe('terminal runtime boundary', () => {
  it('keeps UI terminal consumers on the shared Proto session boundary', () => {
    expectTypeOf<TerminalProps['session']>().toEqualTypeOf<RtcSession | ProtoClientSession>()
    expect(terminalSource).not.toMatch(/TerminalProtocolSession|RTCPeerConnection|RTCDataChannel/)
    expect(hookSource).toMatch(/createProtoTerminalProtocolSession/)
    expect(hookSource).not.toMatch(/RTCPeerConnection|RTCDataChannel/)
  })

  it('leaves dead generation recovery to the workspace session owner', () => {
    expect(shouldRecoverTerminalChannel({ isAlive: () => false }, 'Proto session lease is closed')).toBe(false)
    expect(shouldRecoverTerminalChannel({ isAlive: () => true }, 'Proto session lease is closed')).toBe(true)
  })

  it('does not reconnect a channel after the terminal process has exited', () => {
    expect(shouldRecoverTerminalChannel({ isAlive: () => true }, 'terminal data channel closed: terminal exited')).toBe(false)
  })
})
