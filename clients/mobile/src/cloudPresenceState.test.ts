import { describe, expect, it } from 'vitest'
import { mergeCloudPresenceResults } from './cloudPresenceState'

describe('mergeCloudPresenceResults', () => {
  it('keeps a confirmed offline result across a transient Edge probe failure', () => {
    expect(mergeCloudPresenceResults(
      new Map([['device-1', { state: 'offline', endpointId: 'device-1', edgeName: 'CN2' }]]),
      ['device-1'],
      [{ status: 'rejected', reason: new Error('Edge unavailable') }],
    ).get('device-1')).toEqual({ state: 'offline', endpointId: 'device-1', edgeName: 'CN2' })
  })

  it('does not keep a stale online result when the Edge cannot be reached', () => {
    expect(mergeCloudPresenceResults(
      new Map([['device-1', { state: 'online', endpointId: 'device-1', edgeName: 'CN2', online: true }]]),
      ['device-1'],
      [{ status: 'rejected', reason: new Error('Edge unavailable') }],
    ).get('device-1')).toEqual({ state: 'unknown', endpointId: 'device-1', edgeName: 'CN2', online: undefined })
  })

  it('uses each authenticated Edge response as the new state', () => {
    const result = mergeCloudPresenceResults(
      new Map([
        ['online-device', { state: 'offline', endpointId: 'online-device' }],
        ['offline-device', { state: 'online', endpointId: 'offline-device' }],
      ]),
      ['online-device', 'offline-device'],
      [
        { status: 'fulfilled', value: { state: 'online', online: true, endpointId: 'online-device', edgeName: 'CN2' } },
        { status: 'fulfilled', value: { state: 'offline', online: false, endpointId: 'offline-device', edgeName: 'US1' } },
      ],
    )
    expect(result).toEqual(new Map([
      ['online-device', { state: 'online', online: true, endpointId: 'online-device', edgeName: 'CN2' }],
      ['offline-device', { state: 'offline', online: false, endpointId: 'offline-device', edgeName: 'US1' }],
    ]))
  })
})
