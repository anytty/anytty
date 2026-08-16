import { afterEach, describe, expect, it, vi } from 'vitest'
import { consoleConnectionLogger } from './connectionLogger'

describe('consoleConnectionLogger', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('prints a copyable JSON line for timeout diagnostics', () => {
    const error = vi.spyOn(console, 'error').mockImplementation(() => {})

    consoleConnectionLogger.log({
      scope: 'browser_webrtc',
      event: 'data_channel_open_timeout',
      level: 'error',
      machineId: 'machine-1',
      path: 'hub',
      sessionId: 'rtc-1',
      message: 'timed out opening data channel api',
      details: {
        label: 'api',
        channelReadyState: 'connecting',
        selectedCandidatePair: {
          state: 'succeeded',
          local: { candidateType: 'host' },
        },
      },
    })

    expect(error).toHaveBeenCalledTimes(2)
    expect(error.mock.calls[0]).toEqual([
      '[anytty:browser_webrtc] data_channel_open_timeout timed out opening data channel api',
      expect.objectContaining({
        machineId: 'machine-1',
        path: 'hub',
        sessionId: 'rtc-1',
        label: 'api',
      }),
    ])
    expect(error.mock.calls[1]?.[0]).toBe([
      '[anytty:browser_webrtc] data_channel_open_timeout_json',
      '{"machineId":"machine-1","path":"hub","sessionId":"rtc-1","label":"api","channelReadyState":"connecting","selectedCandidatePair":{"state":"succeeded","local":{"candidateType":"host"}}}',
    ].join(' '))
  })
})
