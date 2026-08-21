import { afterEach, describe, expect, it } from 'vitest'
import { anyttyI18n } from '../i18n'
import type { RtcConnectionPhase } from '../core/transport'
import { connectionPathLabel, connectionPhaseLabel, connectionStatusIsSettled, inferConnectionPhase } from './connectionState'

describe('connectionStatusIsSettled', () => {
  it('stops the busy overlay when a connection attempt fails', () => {
    expect(connectionStatusIsSettled('failed')).toBe(true)
    expect(connectionStatusIsSettled('reconnecting')).toBe(false)
  })
})

describe('connection display projection', () => {
  afterEach(async () => {
    await anyttyI18n.changeLanguage('en')
  })

  it('maps every runtime phase to symmetric English and Chinese user concepts', async () => {
    const phases: RtcConnectionPhase[] = [
      'idle', 'probing', 'resolving', 'signaling', 'connecting', 'authorizing',
      'connected', 'verifying', 'reconnecting', 'waiting_network', 'failed',
    ]

    await anyttyI18n.changeLanguage('en')
    const english = phases.map((phase) => connectionPhaseLabel(phase, anyttyI18n.t))
    await anyttyI18n.changeLanguage('zh-CN')
    const chinese = phases.map((phase) => connectionPhaseLabel(phase, anyttyI18n.t))

    expect(english).toHaveLength(phases.length)
    expect(chinese).toHaveLength(phases.length)
    expect(english.every(Boolean)).toBe(true)
    expect(chinese.every(Boolean)).toBe(true)
    expect([...english, ...chinese].join(' ')).not.toMatch(/\b(?:Direct|SSH|P2P|Relay|ICE|JNI)\b|native runtime|Go runtime|binding|handle|generation/i)
  })

  it('does not expose internal path-owner names as connection labels', () => {
    expect(connectionPathLabel('hub')).toBe('AnyTTY Cloud')
    expect(connectionPathLabel('local')).toBe('Local')
    expect(connectionPathLabel(undefined)).toBe('Connection')
  })

  it('never classifies disconnected status text as connected', () => {
    expect(inferConnectionPhase('Disconnected from device')).toBe('failed')
    expect(inferConnectionPhase('Not connected')).toBe('failed')
    expect(inferConnectionPhase('Connected')).toBe('connected')
  })
})
