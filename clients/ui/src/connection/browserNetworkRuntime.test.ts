import { describe, expect, it } from 'vitest'
import browserRuntimeSource from './browserNetworkRuntime.ts?raw'
import remoteMountSource from '../entries/mountRemoteControlApp.tsx?raw'
import webControlAppSource from '../app/RemoteControlApp.tsx?raw'

describe('browser network runtime boundary', () => {
  it('keeps browser globals inside the browser runtime adapter and entry mount points', () => {
    expect(browserRuntimeSource).toMatch(/globalThis\.fetch/)
    expect(browserRuntimeSource).toMatch(/globalThis\.localStorage/)
    expect(browserRuntimeSource).toMatch(/globalThis\.location/)
    expect(webControlAppSource).not.toMatch(/globalThis\.(fetch|localStorage|location)/)
    expect(remoteMountSource).not.toMatch(/globalThis\.(fetch|localStorage|location)/)
  })

  it('does not introduce a native implementation while preserving a future factory boundary', () => {
    expect(browserRuntimeSource).toMatch(/createBrowserRemoteNetworkRuntime/)
    expect(browserRuntimeSource).toMatch(/createFutureNativeRemoteNetworkRuntime/)
    expect(browserRuntimeSource).not.toMatch(/WKWebView|Swift|Kotlin|Android|nativePlugin|NativeWebRTC/)
  })

  it('keeps browser adapter construction out of the public Web Control component', () => {
    expect(webControlAppSource).not.toMatch(/browserNetworkRuntime|createBrowserRemoteNetworkRuntime/)
    expect(webControlAppSource).not.toMatch(/browserRtcSession|createBrowserRtcSession/)
    expect(webControlAppSource).not.toMatch(/createBrowserLocalAppCrypto/)
  })
})
