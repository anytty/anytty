import { act, cleanup, render, waitFor } from '@testing-library/react'
import { useEffect } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { MockProtoSession } from '../test/mockProtoSession'
import { useTerminalSession, type UseTerminalSessionResult } from './useTerminalSession'

interface DeferredOpen {
  resolve(): void
}

const recoveryHarness = vi.hoisted(() => ({
  failNextConnectionInfo: new Set<string>(),
  failNextSend: new Set<string>(),
  openCalls: new Map<string, number>(),
  recoveryOpens: new Map<string, { resolve(): void }>(),
  sent: [] as Array<{ sessionId: string; data: string; size?: { cols: number; rows: number } }>,
}))

vi.mock('./protoTerminalProtocolSession', () => ({
  createProtoTerminalProtocolSession: (session: { stamp: { endpointId: string } }) => {
    const sessionId = session.stamp.endpointId
    return {
      async getConnectionInfo() {
        if (recoveryHarness.failNextConnectionInfo.delete(sessionId)) {
          throw new Error(`recovery failed for ${sessionId}`)
        }
        return {
          path: 'local' as const,
          connectionId: `recovery-test:${sessionId}`,
          machineId: sessionId,
          relayInUse: false,
        }
      },
      async openTerminal(terminalId: string) {
        const openCall = (recoveryHarness.openCalls.get(sessionId) ?? 0) + 1
        recoveryHarness.openCalls.set(sessionId, openCall)
        const createChannel = () => {
          const channel = {
            label: `proto-terminal:${terminalId}`,
            readyState: 'open' as 'open' | 'closed',
            send() {},
            sendInput(data: string, size?: { cols: number; rows: number }) {
              if (recoveryHarness.failNextSend.delete(sessionId)) {
                throw new Error(`terminal channel send failed for ${sessionId}`)
              }
              recoveryHarness.sent.push({ sessionId, data, ...(size ? { size } : {}) })
            },
            close() { channel.readyState = 'closed' },
          }
          return channel
        }
        if (openCall === 1) return createChannel()
        return await new Promise<ReturnType<typeof createChannel>>((resolve) => {
          recoveryHarness.recoveryOpens.set(sessionId, {
            resolve: () => resolve(createChannel()),
          })
        })
      },
      subscribeTerminal() { return () => {} },
      async loadScrollback() {
        return { beforeOffset: 0, limit: 0, rows: 0, replay: '', hasMore: false, alternate: false }
      },
      closeTerminalChannel() {},
    }
  },
}))

describe('useTerminalSession input recovery owner', () => {
  beforeEach(() => {
    recoveryHarness.failNextConnectionInfo.clear()
    recoveryHarness.failNextSend.clear()
    recoveryHarness.openCalls.clear()
    recoveryHarness.recoveryOpens.clear()
    recoveryHarness.sent.length = 0
  })

  afterEach(() => cleanup())

  it('owns rapid follow-up input and replays every message once in order after recovery', async () => {
    const session = new MockProtoSession('session-a')
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitForInitialOpen('session-a')

    recoveryHarness.failNextSend.add('session-a')
    act(() => {
      expect(current!.sendInput('first', { cols: 80, rows: 24 })).toBe(true)
      expect(current!.sendInput('second', { cols: 81, rows: 25 })).toBe(true)
      expect(current!.sendInput('third')).toBe(true)
    })
    await waitForRecoveryOpen('session-a')
    expect(recoveryHarness.sent).toEqual([])

    await resolveRecoveryOpen('session-a')
    await waitFor(() => expect(recoveryHarness.sent).toEqual([
      { sessionId: 'session-a', data: 'first', size: { cols: 80, rows: 24 } },
      { sessionId: 'session-a', data: 'second', size: { cols: 81, rows: 25 } },
      { sessionId: 'session-a', data: 'third' },
    ]))
    view.unmount()
  })

  it('keeps B owner and its failure intact when A completes after session replacement', async () => {
    const sessionA = new MockProtoSession('session-a')
    const sessionB = new MockProtoSession('session-b')
    let current: UseTerminalSessionResult | undefined
    const onChange = (value: UseTerminalSessionResult) => { current = value }
    const view = render(<Harness session={sessionA} onChange={onChange} />)
    await waitForInitialOpen('session-a')

    recoveryHarness.failNextSend.add('session-a')
    act(() => { expect(current!.sendInput('A')).toBe(true) })
    await waitForRecoveryOpen('session-a')

    view.rerender(<Harness session={sessionB} onChange={onChange} />)
    await waitForInitialOpen('session-b')
    recoveryHarness.failNextSend.add('session-b')
    act(() => {
      expect(current!.sendInput('B')).toBe(true)
      expect(current!.sendInput('x'.repeat(64 * 1024))).toBe(false)
    })
    await waitForRecoveryOpen('session-b')
    await waitFor(() => {
      expect(current?.inputRecoveryFailure).toBe('Terminal input is blocked because the recovery buffer is full')
    })

    await resolveRecoveryOpen('session-a')
    expect(current?.inputRecoveryFailure).toBe('Terminal input is blocked because the recovery buffer is full')
    act(() => { expect(current!.sendInput('C')).toBe(true) })
    await waitFor(() => expect(current?.inputRecoveryFailure).toBeNull())
    expect(recoveryHarness.openCalls.get('session-b')).toBe(2)

    await resolveRecoveryOpen('session-b')
    await waitFor(() => expect(recoveryHarness.sent).toEqual([
      { sessionId: 'session-b', data: 'B' },
      { sessionId: 'session-b', data: 'C' },
    ]))
    view.unmount()
  })

  it('keeps overflow visible while recovery is blocked and clears it after a successful drain', async () => {
    const session = new MockProtoSession('session-bounded')
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitForInitialOpen('session-bounded')

    recoveryHarness.failNextSend.add('session-bounded')
    act(() => {
      expect(current!.sendInput('first')).toBe(true)
      for (let index = 1; index < 64; index += 1) {
        expect(current!.sendInput(`queued-${index}`)).toBe(true)
      }
      expect(current!.sendInput('entry-overflow')).toBe(false)
    })

    await waitFor(() => {
      expect(current?.inputRecoveryFailure).toBe('Terminal input is blocked because the recovery buffer is full')
    })
    await waitForRecoveryOpen('session-bounded')
    expect(current?.inputRecoveryFailure).toBe('Terminal input is blocked because the recovery buffer is full')
    await resolveRecoveryOpen('session-bounded')
    await waitFor(() => expect(recoveryHarness.sent).toHaveLength(64))
    expect(recoveryHarness.sent.some(({ data }) => data === 'entry-overflow')).toBe(false)
    await waitFor(() => expect(current?.inputRecoveryFailure).toBeNull())
    view.unmount()
  })

  it('enforces the byte bound independently of the entry limit', async () => {
    const session = new MockProtoSession('session-byte-bound')
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitForInitialOpen('session-byte-bound')

    recoveryHarness.failNextSend.add('session-byte-bound')
    act(() => {
      expect(current!.sendInput('first')).toBe(true)
      expect(current!.sendInput('x'.repeat(64 * 1024))).toBe(false)
    })

    await waitFor(() => {
      expect(current?.inputRecoveryFailure).toBe('Terminal input is blocked because the recovery buffer is full')
    })
    view.unmount()
  })

  it('surfaces recovery failure and clears it on the next locally accepted input', async () => {
    const session = new MockProtoSession('session-failure')
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitForInitialOpen('session-failure')

    recoveryHarness.failNextSend.add('session-failure')
    recoveryHarness.failNextConnectionInfo.add('session-failure')
    act(() => { expect(current!.sendInput('not-acked')).toBe(true) })

    await waitFor(() => {
      expect(current?.inputRecoveryFailure).toBe('Terminal input recovery failed')
    })
    expect(recoveryHarness.sent).toEqual([])

    act(() => { expect(current!.sendInput('accepted-after-recovery-failure')).toBe(true) })
    await waitFor(() => expect(current?.inputRecoveryFailure).toBeNull())
    expect(recoveryHarness.sent).toEqual([{
      sessionId: 'session-failure',
      data: 'accepted-after-recovery-failure',
    }])
    view.unmount()
  })

  it('surfaces replay failure and clears it on the next locally accepted input', async () => {
    const session = new MockProtoSession('session-replay-failure')
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitForInitialOpen('session-replay-failure')

    recoveryHarness.failNextSend.add('session-replay-failure')
    act(() => { expect(current!.sendInput('failed-replay')).toBe(true) })
    await waitForRecoveryOpen('session-replay-failure')
    recoveryHarness.failNextSend.add('session-replay-failure')
    await resolveRecoveryOpen('session-replay-failure')

    await waitFor(() => {
      expect(current?.inputRecoveryFailure).toBe('Terminal input recovery failed while replaying queued input')
    })
    expect(recoveryHarness.sent).toEqual([])

    act(() => { expect(current!.sendInput('accepted-after-replay-failure')).toBe(true) })
    await waitFor(() => expect(current?.inputRecoveryFailure).toBeNull())
    expect(recoveryHarness.sent).toEqual([{
      sessionId: 'session-replay-failure',
      data: 'accepted-after-replay-failure',
    }])
    view.unmount()
  })

  it('clears an input failure when the session is replaced', async () => {
    const sessionA = new MockProtoSession('session-replace-a')
    const sessionB = new MockProtoSession('session-replace-b')
    let current: UseTerminalSessionResult | undefined
    const onChange = (value: UseTerminalSessionResult) => { current = value }
    const view = render(<Harness session={sessionA} onChange={onChange} />)
    await waitForInitialOpen('session-replace-a')

    recoveryHarness.failNextSend.add('session-replace-a')
    act(() => {
      expect(current!.sendInput('first')).toBe(true)
      expect(current!.sendInput('x'.repeat(64 * 1024))).toBe(false)
    })
    await waitFor(() => expect(current?.inputRecoveryFailure).not.toBeNull())

    view.rerender(<Harness session={sessionB} onChange={onChange} />)
    await waitForInitialOpen('session-replace-b')
    await waitFor(() => expect(current?.inputRecoveryFailure).toBeNull())

    await resolveRecoveryOpen('session-replace-a')
    expect(current?.inputRecoveryFailure).toBeNull()
    view.unmount()
  })

  it('atomically clears failure ownership on unmount so queued input cannot replay', async () => {
    const session = new MockProtoSession('session-unmount')
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitForInitialOpen('session-unmount')

    recoveryHarness.failNextSend.add('session-unmount')
    act(() => {
      expect(current!.sendInput('first')).toBe(true)
      expect(current!.sendInput('x'.repeat(64 * 1024))).toBe(false)
    })
    await waitForRecoveryOpen('session-unmount')
    await waitFor(() => expect(current?.inputRecoveryFailure).not.toBeNull())
    view.unmount()

    await resolveRecoveryOpen('session-unmount')
    expect(recoveryHarness.sent).toEqual([])

    const replacement = new MockProtoSession('session-after-unmount')
    let replacementCurrent: UseTerminalSessionResult | undefined
    const replacementView = render(<Harness
      session={replacement}
      onChange={(value) => { replacementCurrent = value }}
    />)
    await waitForInitialOpen('session-after-unmount')
    expect(replacementCurrent?.inputRecoveryFailure).toBeNull()
    replacementView.unmount()
  })

  it('returns false without creating an owner after the session dies', async () => {
    const session = new MockProtoSession('session-dead')
    let current: UseTerminalSessionResult | undefined
    const view = render(<Harness session={session} onChange={(value) => { current = value }} />)
    await waitForInitialOpen('session-dead')
    await session.close()

    expect(current!.sendInput('blocked')).toBe(false)
    expect(recoveryHarness.sent).toEqual([])
    expect(recoveryHarness.openCalls.get('session-dead')).toBe(1)
    view.unmount()
  })
})

function Harness({
  session,
  onChange,
}: {
  session: MockProtoSession
  onChange: (value: UseTerminalSessionResult) => void
}) {
  const value = useTerminalSession({ machineId: session.stamp.endpointId, terminalId: 'terminal-1', session })
  useEffect(() => onChange(value), [onChange, value])
  return null
}

async function waitForInitialOpen(sessionId: string): Promise<void> {
  await waitFor(() => expect(recoveryHarness.openCalls.get(sessionId)).toBe(1))
}

async function waitForRecoveryOpen(sessionId: string): Promise<DeferredOpen> {
  await waitFor(() => expect(recoveryHarness.recoveryOpens.has(sessionId)).toBe(true))
  return recoveryHarness.recoveryOpens.get(sessionId)!
}

async function resolveRecoveryOpen(sessionId: string): Promise<void> {
  const deferred = await waitForRecoveryOpen(sessionId)
  await act(async () => {
    deferred.resolve()
    await Promise.resolve()
  })
}
