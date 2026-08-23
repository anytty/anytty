import { describe, expect, it } from 'vitest'
import mobileAppSource from './AnyTTYApp.tsx?raw'
import remoteControlSource from '../../ui/src/app/RemoteControlApp.tsx?raw'
import nativeConnectionSource from '../android/app/src/main/java/com/anytty/app/NativeConnectionPlugin.kt?raw'
import iosNativeConnectionSource from '../ios/App/CapApp-SPM/Sources/CapApp-SPM/NativeConnectionPlugin.swift?raw'
import iosBridgeViewControllerSource from '../ios/App/CapApp-SPM/Sources/CapApp-SPM/AnyTTYBridgeViewController.swift?raw'
import nativeRuntimeCoordinatorSource from '../android/app/src/main/java/com/anytty/app/NativeConnectionRuntimeCoordinator.kt?raw'
import nativeRuntimeOwnerSource from '../android/app/src/main/java/com/anytty/app/NativeConnectionRuntimeOwner.kt?raw'
import webChromeClientSource from '../android/app/src/main/java/com/anytty/app/AnyTTYWebChromeClient.java?raw'
import goLoopbackSource from '../../../client/binding/loopback/server.go?raw'
import goMobileConfigSource from '../../../client/mobileconfig/cloud.go?raw'
import foregroundServiceSource from '../android/app/src/main/java/com/anytty/app/AnyTTYConnectionService.kt?raw'
import androidBuildScriptSource from '../../../scripts/build-android-client.sh?raw'
import androidBoundarySource from '../../../scripts/verify-android-apk-boundary.sh?raw'
import androidLogSource from '../../../client/binding/enginehost/host.go?raw'
import mainActivitySource from '../android/app/src/main/java/com/anytty/app/MainActivity.java?raw'
import webViewClientSource from '../android/app/src/main/java/com/anytty/app/AnyTTYWebViewClient.java?raw'

describe('mobile product shell', () => {
  it('does not expose staging IP addresses in the official App shell', () => {
    expect(mobileAppSource).not.toContain('114.66.58.243')
    expect(mobileAppSource).not.toContain('VITE_CONTROL_URL')
    expect(remoteControlSource).not.toContain('workspace.connection.unavailableReason.cloud_unavailable')
  })

  it('links the App to the canonical public privacy policy', () => {
    expect(mobileAppSource).toContain("const privacyPolicyUrl = 'https://anytty.com/privacy/'")
    expect(mobileAppSource).not.toContain('https://cloud.anytty.com/privacy')
  })

  it('keeps the process runtime across backgrounding and replaces only a failed binding', () => {
    expect(nativeConnectionSource).not.toContain('override fun onStop')
    expect(nativeConnectionSource).not.toContain('ACTION_SCREEN_OFF')
    expect(nativeConnectionSource).toContain('NativeConnectionRuntimeOwner.ensureStarted')
    expect(nativeRuntimeCoordinatorSource).toMatch(/fun ensureForForeground[\s\S]*if \(!isRuntimeStarted\(\)\) startRuntime\(\)/)
    expect(nativeRuntimeOwnerSource).toContain('private var goEngine')
    expect(nativeRuntimeOwnerSource).toContain('GoClientNative.startBridge')
    expect(iosNativeConnectionSource).toContain('GoClientNative.startBridge')
    expect(nativeRuntimeOwnerSource).toContain('replaceRendererDemand')
    expect(nativeRuntimeOwnerSource).toContain('baseDemandRevision')
    expect(nativeConnectionSource).toContain('nativeNetworkChangedPayload(epoch, connected, reason)')
    expect(nativeConnectionSource).toContain('fun getNetworkSnapshot(call: PluginCall)')
    expect(mobileAppSource).toContain('entry.manager.networkChanged(connected, reason)')
    expect(mobileAppSource).toContain('networkChanged(event.connected, event.reason)')
    expect(mobileAppSource).toContain('NativeConnection.getNetworkSnapshot()')
    expect(mobileAppSource).toContain('heartbeatGap >= rendererStallReconcileMs')
    expect(mobileAppSource).toMatch(
      /async verify\(session, signal\)[\s\S]*case: 'terminalDefaults'[\s\S]*response\.result\.case !== 'terminalDefaults'/,
    )
    expect(mobileAppSource).toContain('initializeNetworkState(status.connected)')
    expect(mobileAppSource).toContain('connectionStateEvents: createNativeConnectionStateEvents(machine.id, sessionManager)')
    expect(mobileAppSource).toMatch(
      /function createNativeInventoryEvents[\s\S]*sessionManager\.connectionState\.subscribe\(synchronize\)/,
    )
    expect(mobileAppSource).toContain("document.addEventListener('anytty:binding-closed'")
    expect(mobileAppSource).toContain("trigger: 'app_resume'")
    expect(mobileAppSource).toContain("trigger: 'renderer_stall'")
    expect(mobileAppSource).toContain('waitForForeground: (signal) => nativeForegroundBarrier.wait(signal)')
    expect(mobileAppSource).toMatch(
      /successfulRecoveryRevision === 0 \|\| connectionState !== 'ready'[\s\S]*new CustomEvent\('anytty:resume'/,
    )
    expect(mobileAppSource).toMatch(
      /intent === 'repair'[\s\S]*goBindingClient\.getEndpointRegistry\(\)[\s\S]*catch/,
    )
    expect(mobileAppSource).toContain('connectionState={nativeConnectionRecovery.connectionState}')
    expect(mobileAppSource).toContain('onRetryConnectionRecovery={nativeConnectionRecovery.retryConnectionRecovery}')
  })

  it('exports the session activity contract on Android and iOS', () => {
    expect(nativeConnectionSource).toContain('fun replaceSessionDemand(call: PluginCall)')
    expect(nativeConnectionSource).toContain('NativeConnectionRuntimeOwner.replaceRendererDemand')
    expect(iosNativeConnectionSource).toContain('CAPPluginMethod(name: "replaceSessionDemand"')
    expect(iosNativeConnectionSource).toMatch(
      /@objc func replaceSessionDemand[\s\S]*getArray\("endpointIds", String\.self\)[\s\S]*call\.resolve\(\["goManagedEndpointIds": \[\]\]\)/,
    )
  })

  it('uses actual iOS keyboard occlusion instead of floating-keyboard height', () => {
    expect(iosBridgeViewControllerSource).toContain('keyboardWillChangeFrameNotification')
    expect(iosBridgeViewControllerSource).toContain('intersection.width >= webView.bounds.width * 0.9')
    expect(iosBridgeViewControllerSource).toContain('occludedHeight')
    expect(mobileAppSource).toContain("Capacitor.getPlatform() === 'ios'")
  })

  it('recovers a terminated or unresponsive Android WebView renderer natively', () => {
    expect(webViewClientSource).toContain('onRenderProcessGone')
    expect(mainActivitySource).toContain('WebViewCompat.setWebViewRenderProcessClient')
    expect(mainActivitySource).toContain('WEB_VIEW_RENDERER_TERMINATE')
    expect(mainActivitySource).toContain('Build.VERSION.SDK_INT >= Build.VERSION_CODES.O')
    expect(mainActivitySource).toContain('webView.destroy()')
    expect(mainActivitySource).toContain('mainHandler.post(this::recreate)')
  })

  it('projects native local discovery without feeding it into session recovery', () => {
    expect(nativeConnectionSource).toContain('fun isLocalEndpointDiscovered(call: PluginCall)')
    expect(nativeConnectionSource).toContain('notifyListeners("localDiscoveryChanged"')
    expect(iosNativeConnectionSource).toContain('CAPPluginMethod(name: "isLocalEndpointDiscovered"')
    expect(iosNativeConnectionSource).toContain('notifyListeners("localDiscoveryChanged"')
    expect(nativeConnectionSource).toContain('GoClientNative.localProbe')
    expect(iosNativeConnectionSource).toContain('GoClientNative.localProbe')
    expect(mobileAppSource).toContain("NativeConnection.addListener('localDiscoveryChanged'")
    expect(mobileAppSource).toContain('locallyDiscoveredMachineIds={localDiscovery.discoveredMachineIds}')
    expect(mobileAppSource).toContain('locallyDiscoveringMachineIds={localDiscovery.checkingMachineIds}')
    expect(mobileAppSource).toContain('cloudPresenceByMachineId={cloudPresenceByMachineId}')
    expect(mobileAppSource).toContain('goBindingClient.getEndpointCloudPresence(endpointId')
    expect(mobileAppSource).toContain("document.addEventListener('anytty:resume', schedule)")
    expect(mobileAppSource).not.toMatch(/localDiscoveryChanged[\s\S]{0,160}networkChanged\(/)
  })

  it('keeps the mobile bridge and Cloud profile policy in shared Go', () => {
    expect(goLoopbackSource).toContain('func (server *Server) pumpEvents()')
    expect(goLoopbackSource).toContain('server.wg.Wait()')
    expect(goLoopbackSource).toContain('subtle.ConstantTimeCompare')
    expect(goMobileConfigSource).toContain('func ResolveCloudProfile')
    expect(iosNativeConnectionSource).not.toContain('AnyTTYCloudControllerAddress')
  })

  it('enforces Play-compatible native packaging and foreground-service lifecycle', () => {
    expect(androidBuildScriptSource).toContain('max-page-size=16384')
    expect(androidBoundarySource).toContain('native library is not 16 KB page aligned')
    expect(foregroundServiceSource).toContain('START_NOT_STICKY')
    expect(foregroundServiceSource).toContain('ACTION_DISCONNECT_ALL')
    expect(foregroundServiceSource).toContain('connection_notification_stop')
    expect(nativeConnectionSource).toContain('notifyDisconnectAllRequested')
    expect(mobileAppSource).toContain("NativeConnection.addListener('disconnectAllRequested'")
  })

  it('does not emit endpoint identifiers or Edge addresses in Android allowlisted logs', () => {
    expect(androidLogSource).not.toContain('stage=request endpoint_id=')
    expect(androidLogSource).not.toContain('stage=edge_probe edge_id=')
    expect(androidLogSource).not.toContain('stage=controller_resolved edge_id=')
  })

  it('projects daemon terminal activity into the native inventory', () => {
    expect(mobileAppSource).toContain('foreground_process: terminal.foregroundProcess || undefined')
    expect(mobileAppSource).toContain('last_output_at: unixNanoISOString(terminal.lastOutputAtUnixNano)')
    expect(mobileAppSource).toContain("new CustomEvent('anytty:machine-metadata-changed'")
    expect(remoteControlSource).toContain("addEventListener('anytty:machine-metadata-changed'")
  })

  it('allows only the scoped native image picker', () => {
    expect(webChromeClientSource).toContain('AnyTTYFileChooserPolicy.allowsSingleImage')
    expect(webChromeClientSource).toContain('super.onShowFileChooser')
    expect(webChromeClientSource).toContain('filePathCallback.onReceiveValue(null)')
  })

  it('discards the loaded transfer store before native pairing and generation reset', () => {
    expect(mobileAppSource).toMatch(
      /const resetLocalPairings[\s\S]*await nativeAppRuntime\.discardLocalState\(\)[\s\S]*await NativeConnection\.resetLocalPairings\(\)[\s\S]*replaceNativeGeneration/,
    )
  })
})
