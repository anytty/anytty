import { describe, expect, it } from 'vitest'
import connectionReducerSource from '../connection/connectionMessageReducer.ts?raw'
import indexSource from '../index.ts?raw'
import useFileManagerSource from '../files/useFileManager.tsx?raw'

const sourceModules = import.meta.glob('../*.ts', { query: '?raw', import: 'default', eager: false })
const runtimeSources = import.meta.glob('../**/*.{ts,tsx}', { query: '?raw', import: 'default', eager: true }) as Record<string, string>

describe('legacy transport cleanup', () => {
  it('does not keep compatibility modules for old local transport names', () => {
    expect(sourceModules).not.toHaveProperty('./localWebRtcTransport.ts')
    expect(sourceModules).not.toHaveProperty('./localTerminalProtocolTransport.ts')
  })

  it('does not keep private local API callers in runtime sources', () => {
    const source = Object.entries(runtimeSources)
      .filter(([path]) => !/\.test\.tsx?$/.test(path) && !path.includes('/test/'))
      .map(([path, content]) => `${path}\n${content}`)
      .join('\n')

    for (const legacyPath of ['/api/local/status', '/api/local/rtc/offer', '/api/local/pair', '/api/local/terminals']) {
      expect(source).not.toContain(legacyPath)
    }
  })

  it('does not export old local WebRTC or terminal transport aliases from the package barrel', () => {
    expect(indexSource).not.toMatch(/LocalWebRtc|LocalTerminalProtocolTransport|createLocalWebRtc|createLocalTerminalProtocolTransport/)
    expect(indexSource).not.toMatch(/RTCDataChannelLike|RTCPeerConnectionLike/)
    expect(indexSource).not.toMatch(/browserRtcSession|BrowserRtc/)
  })

  it('exports the Proto binding client without old Hub or browser session adapters', () => {
    expect(indexSource).not.toMatch(/createHubApi|HubApi/)
    expect(indexSource).not.toMatch(/createBrowserRtcSession|BrowserRtcSession/)
    expect(indexSource).not.toMatch(/BrowserBindingRuntime/)
    expect(indexSource).toMatch(/ProtoBindingClient/)
  })

  it('keeps file manager target validation named as session validation', () => {
    expect(useFileManagerSource).not.toMatch(/assertTransportTarget|file transport/)
    expect(useFileManagerSource).toMatch(/assertSessionTarget/)
  })

  it('does not model UI lifecycle messages as transport taxonomy', () => {
    expect(connectionReducerSource).not.toMatch(/transport\.(connecting|connected|disconnected|failed|verified)/)
    expect(connectionReducerSource).toMatch(/connection\.connected/)
  })
})
