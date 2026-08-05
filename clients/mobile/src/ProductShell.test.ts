import { describe, expect, it } from 'vitest'
import mobileAppSource from './AnyTTYApp.tsx?raw'
import remoteControlSource from '../../ui/src/app/RemoteControlApp.tsx?raw'
import nativeConnectionSource from '../android/app/src/main/java/com/anytty/app/NativeConnectionPlugin.kt?raw'
import nativeRuntimeCoordinatorSource from '../android/app/src/main/java/com/anytty/app/NativeConnectionRuntimeCoordinator.kt?raw'
import nativeRuntimeOwnerSource from '../android/app/src/main/java/com/anytty/app/NativeConnectionRuntimeOwner.kt?raw'

describe('mobile product shell', () => {
  it('does not expose staging IP addresses in the official App shell', () => {
    expect(mobileAppSource).not.toContain('114.66.58.243')
    expect(mobileAppSource).not.toContain('VITE_CONTROL_URL')
    expect(remoteControlSource).not.toContain('workspace.connection.unavailableReason.cloud_unavailable')
  })

  it('keeps the process runtime across backgrounding and replaces only a failed binding', () => {
    expect(nativeConnectionSource).not.toContain('override fun onStop')
    expect(nativeConnectionSource).not.toContain('ACTION_SCREEN_OFF')
    expect(nativeConnectionSource).toContain('NativeConnectionRuntimeOwner.ensureStarted')
    expect(nativeRuntimeCoordinatorSource).toMatch(/fun ensureForForeground[\s\S]*if \(!isRuntimeStarted\(\)\) startRuntime\(\)/)
    expect(nativeRuntimeOwnerSource).toContain('private var goBridgeServer')
    expect(nativeRuntimeOwnerSource).toContain('setEndpointActive')
    expect(mobileAppSource).toContain('connectionStateEvents: createNativeConnectionStateEvents(machine.id, sessionManager)')
    expect(mobileAppSource).toMatch(
      /function createNativeInventoryEvents[\s\S]*sessionManager\.connectionState\.subscribe\(synchronize\)/,
    )
    expect(mobileAppSource).toContain("document.addEventListener('anytty:binding-closed'")
    expect(mobileAppSource).toContain('void runRecovery(false, true)')
    expect(mobileAppSource).toMatch(
      /else if \(reloadRegistry\)[\s\S]*await goBindingClient\.getEndpointRegistry\(\)[\s\S]*catch/,
    )
    expect(mobileAppSource).toContain('connectionReady={nativeConnectionRecovery.connectionReady}')
    expect(mobileAppSource).toContain('onRetryConnectionRecovery={nativeConnectionRecovery.retryConnectionRecovery}')
  })

  it('discards the loaded transfer store before native pairing and generation reset', () => {
    expect(mobileAppSource).toMatch(
      /const resetLocalPairings[\s\S]*await nativeAppRuntime\.discardLocalState\(\)[\s\S]*await NativeConnection\.resetLocalPairings\(\)[\s\S]*replaceNativeGeneration/,
    )
  })
})
