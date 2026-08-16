import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties } from 'react'
import { App as CapApp } from '@capacitor/app'
import { Capacitor, CapacitorHttp } from '@capacitor/core'
import { Keyboard } from '@capacitor/keyboard'
import { Network } from '@capacitor/network'
import { Browser } from '@capacitor/browser'
import { create } from '@bufbuild/protobuf'
import {
  RemoteControlApp,
  appThemeCssVariables,
  createMachineStore,
  dispatchNativeKeyboardEvent,
  normalizeTerminalInventory,
  openProtoEventSubscription,
  readAppTheme,
  AnyTTYClientBinding,
  AnyTTYApiApplication,
  AnyTTYApiEvents,
  AnyTTYApiTerminal,
  AnyTTYRemoteAuth,
} from '@anytty/ui'
import type {
  FileTransferContext,
  MachineConnectionStateEvents,
  MachineWorkspaceProps,
  LocalStatus,
  Machine,
  RemoteNetworkRuntime,
  RemoteRuntimeFetch,
  RemoteRuntimeStorage,
  RtcConnectOptions,
  RtcEvent,
  RtcSubscription,
  TerminalInventoryEvents,
  RemoteMachine,
  RemoteControlAppProps,
  ExternalPairingAdapter,
  ProtoClientSession,
} from '@anytty/ui'
import { NativeConnection, type NativeNetworkChangedEvent } from './plugins/nativeConnection'
import { NativeFileTransferStore } from './NativeFileTransferStore'
import { GoBindingClient, GoBindingConnector } from './GoBindingClient'
import { settleBindingGeneration } from './BindingGeneration'
import { NativeSessionManager, type NativeSessionConnector } from './NativeSessionManager'
import NativeFilePicker from './plugins/nativeFilePicker'
import { useNativeStatusBarSync } from './nativeStatusBar'
import { NativeForegroundBarrier, runAcrossNativePicker } from './NativeForegroundBarrier'
import { NativeGenerationRecoveryFence } from './NativeGenerationRecoveryFence'
import { NativeRecoveryCoordinator, type NativeRecoveryWork } from './NativeRecoveryCoordinator'
import { RegistryStartupScreen, UnsupportedWebPreview } from './RegistryStartupScreen'
import { useAndroidBackButton } from './androidBack'
import type { NativeQrScannerOptions } from './nativeQrScanner'
import { endpointMachineAccessClass } from './endpointMachineProjection'
import { cloudPresenceWithState, mergeCloudPresenceResults, samePresenceMap, type CloudPresenceState } from './cloudPresenceState'

const nativeHttpConnectTimeoutMs = 8_000
const nativeHttpReadTimeoutMs = 15_000
const localDiscoveryFeedbackDelayMs = 300
const localDiscoverySearchWindowMs = 6_000
const localDiscoveryRefreshIntervalMs = 3_000
const cloudPresenceFeedbackDelayMs = 300
const cloudPresenceProbeTimeoutMs = 15_000
const cloudPresenceRefreshIntervalMs = 20_000
const privacyPolicyUrl = 'https://cloud.anytty.com/privacy'
let goBindingClient = new GoBindingClient()

type MachineRuntimeFactory = NonNullable<RemoteControlAppProps['machineRuntimeFactory']>
type MachineRuntime = ReturnType<MachineRuntimeFactory>
type NativeSessionEntry = {
  endpointIdentity: string
  connector: NativeSessionConnector
  manager: NativeSessionManager
}
type NativeSessionLease = ProtoClientSession

export function AnyTTYApp() {
  const initialAppThemeStyle = useMemo(
    () => appThemeCssVariables(readAppTheme()) as CSSProperties,
    [],
  )
  if (!Capacitor.isNativePlatform()) {
    return <div style={initialAppThemeStyle}><UnsupportedWebPreview /></div>
  }
  return <NativeAnyTTYApp initialAppThemeStyle={initialAppThemeStyle} />
}

function NativeAnyTTYApp({ initialAppThemeStyle }: { initialAppThemeStyle: CSSProperties }) {
  useAndroidBackButton()
  useNativeKeyboardEvents()
  useNativeStatusBarSync()

  const networkRuntime = useMemo(() => createNativeNetworkRuntime(), [])
  const endpointRegistry = useMemo(() => new NativeEndpointRegistryProjection(), [])
  const nativeAppRuntime = useMemo(() => createNativeAppRuntime(endpointRegistry), [endpointRegistry])
  useNativeNetworkSessionRecovery(
    nativeAppRuntime.networkChanged,
    nativeAppRuntime.initializeNetworkState,
  )
  useNativeDisconnectAll(nativeAppRuntime.disconnectAll)
  const [registryReady, setRegistryReady] = useState(false)
  const [registryError, setRegistryError] = useState<string | null>(null)
  const localDiscovery = useNativeLocalDiscovery(endpointRegistry, registryReady)
  const cloudPresenceByMachineId = useNativeCloudPresence(endpointRegistry, registryReady)
  const refreshRegistry = useCallback(async (client: GoBindingClient = goBindingClient) => {
    try {
      const loaded = await settleBindingGeneration(
        client,
        () => goBindingClient,
        () => client.getEndpointRegistry(),
      )
      if (!loaded.current) return
      endpointRegistry.replace(loaded.value)
      if (networkRuntime.storage) syncRegistryMachineProjection(networkRuntime.storage, endpointRegistry.snapshot())
      setRegistryError(null)
      setRegistryReady(true)
    } catch (error) {
      setRegistryError(error instanceof Error ? error.message : String(error))
      throw error
    }
  }, [endpointRegistry, networkRuntime])
  useEffect(() => { void refreshRegistry().catch(() => undefined) }, [refreshRegistry])
  const nativeConnectionRecovery = useAppResumeSync(
    refreshRegistry,
    nativeAppRuntime.resetGeneration,
    nativeAppRuntime.foregroundResume,
    nativeAppRuntime.resumeInterruptedTransfers,
  )
  const externalPairingAdapter = useMemo(
    () => createNativeExternalPairingAdapter(endpointRegistry),
    [endpointRegistry],
  )
  const machineRuntimeFactory = useMemo<MachineRuntimeFactory>(
    () => nativeAppRuntime.createMachineRuntime,
    [nativeAppRuntime],
  )
  const globalFileTransfer = useMemo(
    () => nativeAppRuntime.fileTransfer,
    [nativeAppRuntime],
  )
  const retryRegistry = useCallback(async () => {
    setRegistryError(null)
    try {
      await NativeConnection.handleForegroundResume()
      await replaceNativeGeneration(refreshRegistry, nativeAppRuntime.resetGeneration, true)
    } catch (failure) {
      const message = failure instanceof Error ? failure.message : String(failure)
      setRegistryError(message)
      throw failure
    }
  }, [nativeAppRuntime.resetGeneration, refreshRegistry])
  const resetLocalPairings = useCallback(async () => {
    try {
      await nativeAppRuntime.discardLocalState()
      await NativeConnection.resetLocalPairings()
      networkRuntime.storage?.removeItem('anytty.app.machines.v2')
      endpointRegistry.replace(create(AnyTTYRemoteAuth.EndpointRegistryV1Schema, { schemaVersion: 1 }))
      await replaceNativeGeneration(refreshRegistry, nativeAppRuntime.resetGeneration, true)
    } catch (failure) {
      const message = failure instanceof Error ? failure.message : String(failure)
      setRegistryError(message)
      throw failure
    }
  }, [endpointRegistry, nativeAppRuntime.discardLocalState, nativeAppRuntime.resetGeneration, networkRuntime, refreshRegistry])

  if (!registryReady) {
    return (
      <div style={initialAppThemeStyle}>
        <RegistryStartupScreen
          error={registryError}
          onResetLocalPairings={resetLocalPairings}
          onRetry={retryRegistry}
        />
      </div>
    )
  }

  return (
    <section className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex h-[100dvh] w-screen flex-col overflow-hidden antialiased" style={initialAppThemeStyle}>
      <RemoteControlApp
        singlePaneWorkspace
        externalPairingAdapter={externalPairingAdapter}
        globalFileTransfer={globalFileTransfer}
        machineRuntimeFactory={machineRuntimeFactory}
        networkRuntime={networkRuntime}
        nativeNetworkStatusPlugin={Network}
        locallyDiscoveredMachineIds={localDiscovery.discoveredMachineIds}
        locallyDiscoveringMachineIds={localDiscovery.checkingMachineIds}
        cloudPresenceByMachineId={cloudPresenceByMachineId}
        connectionReady={nativeConnectionRecovery.connectionReady}
        connectionRecoveryFailed={nativeConnectionRecovery.connectionRecoveryFailed}
        onRetryConnectionRecovery={nativeConnectionRecovery.retryConnectionRecovery}
        onRefreshMachines={() => refreshRegistry()}
        pickMachineIconImage={pickNativeMachineIconImage}
        scanPairingCode={scanNativePairingCode}
        privacyPolicyUrl={privacyPolicyUrl}
        onOpenPrivacyPolicy={() => Browser.open({ url: privacyPolicyUrl })}
      />
    </section>
  )
}

function scanNativePairingCode(options?: NativeQrScannerOptions): Promise<string | null> {
  return import('./nativeQrScanner').then(({ scanPairingCode }) => scanPairingCode(options))
}

async function pickNativeMachineIconImage(): Promise<File | null> {
  const result = await runAcrossNativePicker(nativeForegroundBarrier, () => NativeFilePicker.pickFiles({
    multiple: false,
    mimeType: 'image/*',
  }))
  const picked = result.files[0]
  if (!picked) return null
  const source = await NativeFilePicker.readPickedFile({ contentUri: picked.uri, maximumBytes: 8 * 1024 * 1024 })
  const binary = atob(source.dataBase64)
  const bytes = Uint8Array.from(binary, (value) => value.charCodeAt(0))
  const mimeType = source.mimeType.startsWith('image/')
    ? source.mimeType
    : picked.mimeType.startsWith('image/') ? picked.mimeType : 'image/*'
  return new File([bytes], source.name || picked.name, { type: mimeType })
}

function createNativeExternalPairingAdapter(registry: NativeEndpointRegistryProjection): ExternalPairingAdapter {
  return {
    async import(rawValue, expectedMachineId) {
    const imported = await goBindingClient.importPairing(create(AnyTTYClientBinding.ImportPairingRequestSchema, {
    requestId: crypto.randomUUID(),
    portablePayload: rawValue,
    expectedEndpointId: expectedMachineId ?? '',
    }))
    const endpoint = imported.endpoint
    if (!endpoint?.endpointId || !endpoint.identity || endpoint.routes.length === 0) return null
    const expiresAt = new Date(Number(imported.expiresAtUnixNano / 1_000_000n)).toISOString()
    registry.replace(imported.registry ?? await goBindingClient.getEndpointRegistry())
    registry.setAuthorizationExpiry(endpoint.endpointId, expiresAt)
      return {
    machine: { id: endpoint.endpointId, name: endpoint.label || endpoint.endpointId, osInfo: endpoint.platform || undefined, accessClass: endpointMachineAccessClass(endpoint) },
    expiresAt,
      }
    },
  async inspectShare(rawValue) {
    const received = await goBindingClient.receiveEndpointShare(rawValue)
    const preview = received.preview
    if (!preview?.importToken || !preview.identity) throw new Error('Endpoint share preview is incomplete')
    return {
    importToken: preview.importToken,
    endpointId: preview.endpointId,
    label: preview.label || preview.endpointId,
    deviceId: preview.identity.deviceId,
    deviceFingerprint: preview.identity.deviceFingerprint,
    routes: preview.routeDiffs.map((route) => ({ id: route.routeId, kind: route.routeKind, action: route.action })),
    connectModeChanged: preview.connectModeChanged,
    selectionPolicyChanged: preview.selectionPolicyChanged,
    credentialKinds: preview.credentialDescriptors.map((descriptor) => String(descriptor.kind)),
    }
  },
    async commitShare(importToken) {
      const committed = await goBindingClient.commitEndpointShare(importToken)
      let endpoint = committed.endpoint
      if (!endpoint?.endpointId || !endpoint.identity || !committed.registry) throw new Error('Endpoint share commit is incomplete')
      registry.replace(committed.registry)
      const sshCredentials: NonNullable<import('@anytty/ui').ExternalPairingImportResult['sshCredentials']> = []
      for (const route of endpoint.routes) {
      if (route.route.case !== 'sshWebrtcTcp' || route.route.value.credentialDescriptor?.kind !== AnyTTYRemoteAuth.EndpointCredentialKind.SSH_PRIVATE_KEY) continue
      const provisioned = await goBindingClient.provisionSSHCredential(endpoint.endpointId, route.routeId)
      if (!provisioned.endpoint || !provisioned.registry) throw new Error('SSH credential provision result is incomplete')
      registry.replace(provisioned.registry)
      endpoint = provisioned.endpoint
      sshCredentials.push({ routeId: route.routeId, authorizedKey: provisioned.authorizedKey, fingerprint: provisioned.keyFingerprint })
      }
      return {
      machine: { id: endpoint.endpointId, name: endpoint.label || endpoint.endpointId, osInfo: endpoint.platform || undefined, accessClass: endpointMachineAccessClass(endpoint) },
      authorizationRequired: !registry.isAuthorized(endpoint.endpointId),
      ...(sshCredentials.length > 0 ? { sshCredentials } : {}),
      }
  },
    isAuthorized(machineId) {
    return registry.isAuthorized(machineId)
    },
    authorizationExpiresAt(machineId) {
    return registry.authorizationExpiry(machineId)
    },
    async forget(machineId) {
    const deleted = await goBindingClient.deleteEndpoint(machineId)
    if (deleted.registry) registry.replace(deleted.registry)
    registry.setAuthorizationExpiry(machineId, undefined)
    },
  }
}

/** NativeEndpointRegistryProjection 是 Go registry 的只读 UI projection，不执行字段合并或持久化。 */
class NativeEndpointRegistryProjection {
  private registry = create(AnyTTYRemoteAuth.EndpointRegistryV1Schema, { schemaVersion: 1 })
  private readonly expiries = new Map<string, string>()
  private versionValue = 0
  private readonly listeners = new Set<() => void>()

  replace(registry: AnyTTYRemoteAuth.EndpointRegistryV1): void {
    this.registry = create(AnyTTYRemoteAuth.EndpointRegistryV1Schema, registry)
    this.versionValue += 1
    for (const listener of this.listeners) listener()
  }

  snapshot(): AnyTTYRemoteAuth.EndpointRegistryV1 { return create(AnyTTYRemoteAuth.EndpointRegistryV1Schema, this.registry) }
  has(endpointId: string): boolean { return this.registry.endpoints.some((endpoint) => endpoint.endpointId === endpointId) }
  isAuthorized(endpointId: string): boolean {
  const endpoint = this.registry.endpoints.find((candidate) => candidate.endpointId === endpointId)
  return endpoint?.routes.some((route) => route.credentialRef.trim() !== '') ?? false
  }
  version(): number { return this.versionValue }
  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => this.listeners.delete(listener)
  }
  localDiscoveryIdentities(): Array<{ machineId: string; deviceId: string; fingerprint: string }> {
    return this.registry.endpoints.flatMap((endpoint) => {
      const deviceId = endpoint.identity?.deviceId.trim() ?? ''
      const fingerprint = endpoint.identity?.deviceFingerprint.trim() ?? ''
      return endpoint.endpointId && deviceId && fingerprint
        ? [{ machineId: endpoint.endpointId, deviceId, fingerprint }]
        : []
    })
  }
  cloudPresenceEndpointIds(): string[] {
    return this.registry.endpoints
      .filter((endpoint) => endpoint.endpointId && endpoint.routes.some((route) => route.enabled && route.route.case === 'managedWebrtc'))
      .map((endpoint) => endpoint.endpointId)
  }
  authorizationExpiry(endpointId: string): string | undefined { return this.expiries.get(endpointId) }
  setAuthorizationExpiry(endpointId: string, expiresAt: string | undefined): void {
    if (expiresAt) this.expiries.set(endpointId, expiresAt)
    else this.expiries.delete(endpointId)
  }
}

function useNativeLocalDiscovery(
  registry: NativeEndpointRegistryProjection,
  enabled: boolean,
): { discoveredMachineIds: ReadonlySet<string>; checkingMachineIds: ReadonlySet<string> } {
  const [view, setView] = useState<{ discoveredMachineIds: ReadonlySet<string>; checkingMachineIds: ReadonlySet<string> }>(() => ({
    discoveredMachineIds: new Set(),
    checkingMachineIds: new Set(),
  }))
  useEffect(() => {
    if (!enabled) {
      setView({ discoveredMachineIds: new Set(), checkingMachineIds: new Set() })
      return
    }
    let cancelled = false
    let revision = 0
    let feedbackTimer: ReturnType<typeof setTimeout> | undefined
    let settleTimer: ReturnType<typeof setTimeout> | undefined
    let interval: ReturnType<typeof setInterval> | undefined
    const startSearch = (restartWindow = true) => {
      const identities = registry.localDiscoveryIdentities()
      if (!restartWindow && settleTimer) {
        void refresh(revision, identities)
        return
      }
      const currentRevision = ++revision
      if (feedbackTimer) clearTimeout(feedbackTimer)
      if (settleTimer) clearTimeout(settleTimer)
      const eligible = new Set(identities.map((identity) => identity.machineId))
      feedbackTimer = setTimeout(() => {
        feedbackTimer = undefined
        if (cancelled || currentRevision !== revision) return
        setView((current) => {
          const checking = new Set([...eligible].filter((machineId) => !current.discoveredMachineIds.has(machineId)))
          return sameStringSet(current.checkingMachineIds, checking) ? current : { ...current, checkingMachineIds: checking }
        })
      }, localDiscoveryFeedbackDelayMs)
      settleTimer = setTimeout(() => {
        settleTimer = undefined
        if (cancelled || currentRevision !== revision) return
        setView((current) => current.checkingMachineIds.size === 0 ? current : { ...current, checkingMachineIds: new Set() })
      }, localDiscoverySearchWindowMs)
      void refresh(currentRevision, identities)
    }
    const refresh = async (currentRevision: number, identities: ReturnType<NativeEndpointRegistryProjection['localDiscoveryIdentities']>) => {
      try {
        const results = await Promise.all(identities.map(async (identity) => ({
          machineId: identity.machineId,
          discovered: (await NativeConnection.isLocalEndpointDiscovered(identity)).discovered,
        })))
        if (cancelled || currentRevision !== revision) return
        const next = new Set(results.filter((result) => result.discovered).map((result) => result.machineId))
        setView((current) => {
          const checking = new Set([...current.checkingMachineIds].filter((machineId) => !next.has(machineId)))
          return sameStringSet(current.discoveredMachineIds, next) && sameStringSet(current.checkingMachineIds, checking)
            ? current
            : { discoveredMachineIds: next, checkingMachineIds: checking }
        })
      } catch {
        // The native bridge can be briefly unavailable while its generation changes.
      }
    }
    const removeRegistryListener = registry.subscribe(() => startSearch(true))
    const nativeListener = NativeConnection.addListener('localDiscoveryChanged', () => startSearch(false))
    void nativeListener.then(() => startSearch(true)).catch(() => undefined)
    interval = setInterval(() => {
      void refresh(revision, registry.localDiscoveryIdentities())
    }, localDiscoveryRefreshIntervalMs)
    return () => {
      cancelled = true
      if (feedbackTimer) clearTimeout(feedbackTimer)
      if (settleTimer) clearTimeout(settleTimer)
      if (interval) clearInterval(interval)
      removeRegistryListener()
      void nativeListener.then((listener) => listener.remove())
    }
  }, [enabled, registry])
  return view
}

function useNativeCloudPresence(
  registry: NativeEndpointRegistryProjection,
  enabled: boolean,
): ReadonlyMap<string, CloudPresenceState> {
  const [presence, setPresence] = useState<ReadonlyMap<string, CloudPresenceState>>(() => new Map())
  useEffect(() => {
    if (!enabled) {
      setPresence(new Map())
      return
    }
    let cancelled = false
    let revision = 0
    let feedbackTimer: ReturnType<typeof setTimeout> | undefined
    let interval: ReturnType<typeof setInterval> | undefined
    let activeController: AbortController | undefined
    const refresh = async () => {
      const currentRevision = ++revision
      activeController?.abort(new Error('Cloud presence probe superseded'))
      const endpointIds = registry.cloudPresenceEndpointIds()
      const eligible = new Set(endpointIds)
      setPresence((current) => prunePresence(current, eligible))
      if (feedbackTimer) clearTimeout(feedbackTimer)
      feedbackTimer = setTimeout(() => {
        if (cancelled || currentRevision !== revision) return
        setPresence((current) => {
          const next = new Map(current)
          for (const endpointId of endpointIds) {
            const previous = next.get(endpointId)
            if (!previous || previous.state === 'unknown') next.set(endpointId, cloudPresenceWithState(previous, 'checking', endpointId))
          }
          return samePresenceMap(current, next) ? current : next
        })
      }, cloudPresenceFeedbackDelayMs)
      const controller = new AbortController()
      activeController = controller
      const timeout = setTimeout(() => controller.abort(new Error('Cloud presence probe timed out')), cloudPresenceProbeTimeoutMs)
      const results = await Promise.allSettled(endpointIds.map((endpointId) => goBindingClient.getEndpointCloudPresence(endpointId, controller.signal)))
      clearTimeout(timeout)
      if (activeController === controller) activeController = undefined
      if (cancelled || currentRevision !== revision) return
      setPresence((current) => {
        const next = mergeCloudPresenceResults(current, endpointIds, results)
        return samePresenceMap(current, next) ? current : next
      })
    }
    const schedule = () => { void refresh() }
    const removeRegistryListener = registry.subscribe(schedule)
    document.addEventListener('anytty:resume', schedule)
    void refresh()
    interval = setInterval(schedule, cloudPresenceRefreshIntervalMs)
    return () => {
      cancelled = true
      revision += 1
      if (feedbackTimer) clearTimeout(feedbackTimer)
      if (interval) clearInterval(interval)
      activeController?.abort(new Error('Cloud presence probe stopped'))
      removeRegistryListener()
      document.removeEventListener('anytty:resume', schedule)
    }
  }, [enabled, registry])
  return presence
}

function prunePresence(current: ReadonlyMap<string, CloudPresenceState>, eligible: ReadonlySet<string>): ReadonlyMap<string, CloudPresenceState> {
  const next = new Map([...current].filter(([machineId]) => eligible.has(machineId)))
  return samePresenceMap(current, next) ? current : next
}

function sameStringSet(left: ReadonlySet<string>, right: ReadonlySet<string>): boolean {
  return left.size === right.size && [...left].every((value) => right.has(value))
}

function syncRegistryMachineProjection(storage: RemoteRuntimeStorage, registry: AnyTTYRemoteAuth.EndpointRegistryV1): void {
  const store = createMachineStore({ storage })
  const now = new Date().toISOString()
  for (const endpoint of registry.endpoints) {
    const existing = store.getMachine(endpoint.endpointId)
    store.saveMachine({
      machineId: endpoint.endpointId,
      name: endpoint.label || existing?.name || endpoint.endpointId,
      ...(existing?.alias ? { alias: existing.alias } : {}),
      ...(existing?.icon ? { icon: existing.icon } : {}),
      ...(existing?.iconImage ? { iconImage: existing.iconImage } : {}),
      ...(existing?.hostname ? { hostname: existing.hostname } : {}),
      ...((endpoint.platform || existing?.osInfo) ? { osInfo: endpoint.platform || existing?.osInfo } : {}),
      ...(existing?.hubId ? { hubId: existing.hubId } : {}),
      state: existing?.state ?? 'offline',
      terminalCount: existing?.terminalCount ?? 0,
      ...(existing?.lastSeenAt ? { lastSeenAt: existing.lastSeenAt } : {}),
      ...(existing?.lastConnectionPath ? { lastConnectionPath: existing.lastConnectionPath } : {}),
      ...(existing?.preferredPath ? { preferredPath: existing.preferredPath } : {}),
      ...(typeof existing?.relayInUse === 'boolean' ? { relayInUse: existing.relayInUse } : {}),
      source: existing?.source ?? 'manual',
      accessClass: endpointMachineAccessClass(endpoint),
      addresses: existing?.addresses ?? { local: [], lan: [], public: [] },
      endpoints: existing?.endpoints ?? {},
      addedAt: existing?.addedAt ?? now,
      updatedAt: now,
    })
  }
}

function useNativeKeyboardEvents(): void {
  useEffect(() => {
    const subscriptions = [
      Keyboard.addListener('keyboardWillShow', (info) => {
        dispatchNativeKeyboardEvent({ visible: true, keyboardHeight: info.keyboardHeight })
      }),
      Keyboard.addListener('keyboardDidShow', (info) => {
        dispatchNativeKeyboardEvent({ visible: true, keyboardHeight: info.keyboardHeight })
      }),
      Keyboard.addListener('keyboardWillHide', () => {
        dispatchNativeKeyboardEvent({ visible: false })
      }),
      Keyboard.addListener('keyboardDidHide', () => {
        dispatchNativeKeyboardEvent({ visible: false })
      }),
    ]

    return () => {
      for (const subscription of subscriptions) {
        void subscription.then((handle) => handle.remove())
      }
    }
  }, [])
}

const nativeForegroundBarrier = new NativeForegroundBarrier()

function markNativeBackground(): void {
  nativeForegroundBarrier.markBackground()
}

function finishNativeForeground(failure?: unknown): void {
  nativeForegroundBarrier.finishForeground(failure)
}

function reportNativeGenerationFailure(failure: unknown): void {
  void failure
}

let nativeGenerationReplacement: Promise<void> = Promise.resolve()

function replaceNativeGeneration(
  refreshRegistry: (client?: GoBindingClient) => Promise<void>,
  resetRuntime: () => Promise<void>,
  reloadRegistry: boolean,
): Promise<void> {
  const replacement = nativeGenerationReplacement.catch(() => undefined).then(async () => {
    const staleClient = goBindingClient
    const currentClient = new GoBindingClient()
    goBindingClient = currentClient
    // Endpoint runtime 仍被 React workspace 缓存；必须先清除其旧 binding session，下一次操作才会
    // 通过动态 connector 进入 currentClient，而不是继续返回已失效的 generation。
    await resetRuntime()
    await staleClient.close()
    // 网络切换不会修改 Endpoint registry；此时重读 registry 会让恢复依赖刚启动 engine 的
    // 额外 operation。只有 WebView 前后台恢复才重新读取 Go-owned 持久投影。
    if (reloadRegistry) await refreshRegistry(currentClient)
  })
  nativeGenerationReplacement = replacement
  return replacement
}

/** Keep the native generation across backgrounding; replace only a bridge that actually failed. */
function useAppResumeSync(
  refreshRegistry: (client?: GoBindingClient) => Promise<void>,
  resetRuntime: () => Promise<void>,
  foregroundResume: () => Promise<void>,
  resumeInterruptedTransfers: () => void,
): {
  connectionReady: boolean
  connectionRecoveryFailed: boolean
  retryConnectionRecovery: () => Promise<void>
} {
  const [status, setStatus] = useState<'ready' | 'restoring' | 'failed'>('ready')
  const [recoveryFence] = useState(() => new NativeGenerationRecoveryFence())
  const [recoveryCoordinator] = useState(() => new NativeRecoveryCoordinator())
  const executeRecovery = useCallback(async ({ attempt, replaceBinding, reloadRegistry }: NativeRecoveryWork) => {
    if (!recoveryFence.isCurrent(attempt)) return
    markNativeBackground()
    try {
      await NativeConnection.handleForegroundResume()
      if (!recoveryFence.isCurrent(attempt)) return
      if (replaceBinding) {
        await replaceNativeGeneration(refreshRegistry, resetRuntime, reloadRegistry)
      } else if (reloadRegistry) {
        try {
          // Resume only probes the existing bridge. Replacing an unchanged registry increments
          // its projection version and makes healthy machine runtimes look stale, which would
          // unnecessarily tear down their live sessions on every foreground transition.
          await goBindingClient.getEndpointRegistry()
        } catch {
          if (!recoveryFence.isCurrent(attempt)) return
          setStatus('restoring')
          await replaceNativeGeneration(refreshRegistry, resetRuntime, true)
        }
      }
      if (!recoveryFence.isCurrent(attempt)) return
      await foregroundResume()
      if (!recoveryFence.isCurrent(attempt)) return
      resumeInterruptedTransfers()
      setStatus('ready')
      finishNativeForeground()
      document.dispatchEvent(new Event('anytty:resume'))
    } catch (failure) {
      if (!recoveryFence.isCurrent(attempt)) return
      setStatus('failed')
      reportNativeGenerationFailure(failure)
      finishNativeForeground(failure)
      throw failure
    }
  }, [foregroundResume, recoveryFence, refreshRegistry, resetRuntime, resumeInterruptedTransfers])
  const runRecovery = useCallback((replaceBinding: boolean, reloadRegistry: boolean) => {
    if (replaceBinding) setStatus('restoring')
    return recoveryCoordinator.request({ replaceBinding, reloadRegistry }, {
      beginAttempt: () => recoveryFence.beginAttempt(),
      isCurrent: (attempt) => recoveryFence.isCurrent(attempt),
      execute: executeRecovery,
    })
  }, [executeRecovery, recoveryCoordinator, recoveryFence])

  const retryConnectionRecovery = useCallback(async () => {
    await runRecovery(true, false).catch(() => undefined)
  }, [runRecovery])

  useEffect(() => {
    const promise = CapApp.addListener('appStateChange', (state) => {
      if (!state.isActive) {
        recoveryFence.invalidate()
        markNativeBackground()
        return
      }
      void runRecovery(false, true).catch(() => undefined)
    })
    const handleBindingClosed = () => { void runRecovery(true, false).catch(() => undefined) }
    document.addEventListener('anytty:binding-closed', handleBindingClosed)
    return () => {
      void promise.then((sub) => sub.remove())
      document.removeEventListener('anytty:binding-closed', handleBindingClosed)
    }
  }, [recoveryFence, runRecovery])

  return {
    connectionReady: status === 'ready',
    connectionRecoveryFailed: status === 'failed',
    retryConnectionRecovery,
  }
}

function createNativeNetworkRuntime(): RemoteNetworkRuntime {
  return {
    fetch: nativeFetch,
    storage: browserStorage(),
    queryParam(name) {
      return new URLSearchParams(globalThis.location?.search ?? '').get(name)
    },
  }
}

const nativeFetch: RemoteRuntimeFetch = async (input, init = {}) => {
  if (!Capacitor.isNativePlatform()) {
    return globalThis.fetch(input, init)
  }

  const url = String(input)
  const method = init.method ?? 'GET'
  const headers = headersRecord(init.headers)
  const data = requestData(init.body)
  const response = await CapacitorHttp.request({
    url,
    method,
    headers,
    ...(data !== undefined ? { data } : {}),
    responseType: 'text',
    connectTimeout: nativeHttpConnectTimeoutMs,
    readTimeout: nativeHttpReadTimeoutMs,
  })

  return new Response(responseText(response.data), {
    status: response.status,
    headers: response.headers,
  })
}

function browserStorage(): RemoteRuntimeStorage | undefined {
  const storage = globalThis.localStorage
  if (
    !storage ||
    typeof storage.getItem !== 'function' ||
    typeof storage.setItem !== 'function' ||
    typeof storage.removeItem !== 'function'
  ) {
    return undefined
  }
  return storage
}

function createNativeAppRuntime(endpointRegistry: NativeEndpointRegistryProjection): {
  createMachineRuntime: MachineRuntimeFactory
  fileTransfer: FileTransferContext
  discardLocalState: () => Promise<void>
  resetGeneration: () => Promise<void>
  foregroundResume: () => Promise<void>
  initializeNetworkState: (connected: boolean) => Promise<void>
  networkChanged: (connected: boolean, reason: NativeNetworkChangedEvent['reason']) => Promise<void>
  disconnectAll: () => Promise<void>
  resumeInterruptedTransfers: () => void
} {
  const transferStore = new NativeFileTransferStore()
  const sessionManagers = new Map<string, NativeSessionEntry>()
  const autoResumeMachines = new Set<string>()
  let networkRevision = 0
  let networkConnected = true
  transferStore.setSessionResolver(async (machineId, signal) => {
    return await sessionManagers.get(machineId)?.manager.get({ signal }) ?? null
  })

  return {
    fileTransfer: createFileTransferContext(undefined, transferStore),
    discardLocalState() {
      return transferStore.discardForLocalReset()
    },
    resumeInterruptedTransfers() {
      void transferStore.resumeInterruptedTransfers()
    },
    async resetGeneration() {
      await transferStore.suspendForRuntimeReset()
      await Promise.all([...sessionManagers.values()].map(async (entry) => {
        await entry.manager.reset()
        await entry.connector.release?.(entry.manager.machineID())
      }))
    },
    async foregroundResume() {
      await Promise.allSettled([...sessionManagers.values()].map((entry) => entry.manager.foregroundResume()))
    },
    async initializeNetworkState(connected) {
      networkConnected = connected
      await Promise.all([...sessionManagers.values()].map((entry) => entry.manager.initializeNetworkState(connected)))
    },
    async networkChanged(connected, reason) {
      networkRevision += 1
      networkConnected = connected
      await Promise.all([...sessionManagers.values()].map((entry) => entry.manager.networkChanged(connected, reason)))
    },
    async disconnectAll() {
      await transferStore.suspendForUserStop()
      await Promise.allSettled([...sessionManagers.values()].map((entry) => entry.manager.disconnect()))
    },
    createMachineRuntime(input) {
      return createNativeMachineRuntime(input.machine, input.storage, endpointRegistry, {
        sessionManagers,
        transferStore,
        autoResumeMachines,
        networkConnected: () => networkConnected,
      })
    },
  }
}

function useNativeDisconnectAll(disconnectAll: () => Promise<void>): void {
  useEffect(() => {
    const listener = NativeConnection.addListener('disconnectAllRequested', () => {
      void disconnectAll().catch(() => undefined)
    })
    return () => { void listener.then((subscription) => subscription.remove()) }
  }, [disconnectAll])
}

function useNativeNetworkSessionRecovery(
  networkChanged: (connected: boolean, reason: NativeNetworkChangedEvent['reason']) => Promise<void>,
  initializeNetworkState: (connected: boolean) => Promise<void>,
): void {
  useEffect(() => {
    let latestEpoch = 0
    let cancelled = false
    let receivedNativeEvent = false
    let lastTimerTick = Date.now()
    const applyNetworkEvent = (event: NativeNetworkChangedEvent) => {
      if (!Number.isSafeInteger(event.epoch) || event.epoch <= latestEpoch) return
      latestEpoch = event.epoch
      receivedNativeEvent = true
      void networkChanged(event.connected, event.reason).catch(() => undefined)
    }
    const synchronizeNetworkSnapshot = async (initialize: boolean) => {
      try {
        const snapshot = await NativeConnection.getNetworkSnapshot()
        if (cancelled) return
        if (snapshot.epoch > latestEpoch) {
          applyNetworkEvent(snapshot)
        } else if (initialize && !receivedNativeEvent) {
          const status = await Network.getStatus()
          if (!cancelled && !receivedNativeEvent) await initializeNetworkState(status.connected)
        }
      } catch {
        if (!initialize || cancelled || receivedNativeEvent) return
        const status = await Network.getStatus()
        if (!cancelled && !receivedNativeEvent) await initializeNetworkState(status.connected)
      }
    }
    const listener = NativeConnection.addListener('networkChanged', applyNetworkEvent)
    void listener.then(() => synchronizeNetworkSnapshot(true)).catch(() => undefined)
    const synchronizeVisiblePage = () => {
      if (document.visibilityState === 'visible') void synchronizeNetworkSnapshot(false)
    }
    document.addEventListener('visibilitychange', synchronizeVisiblePage)
    const stallDetector = globalThis.setInterval(() => {
      const now = Date.now()
      const elapsed = now - lastTimerTick
      lastTimerTick = now
      if (elapsed >= 2_500 && document.visibilityState === 'visible') {
        void synchronizeNetworkSnapshot(false)
      }
    }, 1_000)
    return () => {
      cancelled = true
      globalThis.clearInterval(stallDetector)
      document.removeEventListener('visibilitychange', synchronizeVisiblePage)
      void listener.then((subscription) => subscription.remove())
    }
  }, [initializeNetworkState, networkChanged])
}

function createNativeMachineRuntime(
  machine: RemoteMachine,
  storage: RemoteRuntimeStorage,
  endpointRegistry: NativeEndpointRegistryProjection,
  shared: {
    sessionManagers: Map<string, NativeSessionEntry>
    transferStore: NativeFileTransferStore
    autoResumeMachines: Set<string>
    networkConnected: () => boolean
  },
): MachineRuntime {
  const machineStore = createMachineStore({ storage })
  const storedMachine = machineStore.getMachine(machine.id)
  const endpointIdentity = [
    machine.id,
  endpointRegistry.version(),
  ].join('|')
  let entry = shared.sessionManagers.get(machine.id)
  if (!entry || entry.endpointIdentity !== endpointIdentity) {
    void entry?.manager.reset().catch(() => {})
    void entry?.connector.release?.(machine.id).catch(() => {})
    const connector = createNativeConnector(machine, endpointRegistry, (platform) => {
      const normalized = normalizeDaemonPlatform(platform)
      if (!normalized) return
      const current = machineStore.getMachine(machine.id)
      if (!current || current.osInfo === normalized) return
      machineStore.saveMachine({ ...current, osInfo: normalized, updatedAt: new Date().toISOString() })
      globalThis.dispatchEvent(new CustomEvent('anytty:machine-metadata-changed', { detail: { machineId: machine.id } }))
    })
    entry = {
      endpointIdentity,
      connector,
      manager: new NativeSessionManager(machine.id, connector, {
        // Android keeps the Go engine alive across WebView recreation. The first
        // JS lease may therefore be a retained physical session even with no new
        // native network event; verify it with one lightweight RPC before reuse.
        verifyOnFirstAcquire: true,
        initiallyConnected: shared.networkConnected(),
      }),
    }
    shared.sessionManagers.set(machine.id, entry)
  }
  const sessionManager = entry.manager
  const connector = entry.connector
  const transferStore = shared.transferStore
  if (!shared.autoResumeMachines.has(machine.id)) {
    shared.autoResumeMachines.add(machine.id)
    queueMicrotask(() => { void transferStore.resumeInterruptedTransfers(machine.id) })
  }

  const api: MachineWorkspaceProps['api'] = {
    async getStatus(): Promise<LocalStatus> {
      const currentStoredMachine = machineStore.getMachine(machine.id)
      const statusMachine: Machine = {
        machineId: machine.id,
        name: currentStoredMachine?.alias || machine.name,
        state: machine.online ? 'online' : 'offline',
        terminalCount: currentStoredMachine?.terminalCount,
        ...(machine.lastSeen || currentStoredMachine?.lastSeenAt ? { lastSeenAt: machine.lastSeen ?? currentStoredMachine?.lastSeenAt } : {}),
      }
      return {
        machine: statusMachine,
        localWeb: {
          httpUrl: storedMachine?.endpoints.webControl ?? '',
          rtcOfferUrl: firstNonEmpty(machine.hubUrls) ?? storedMachine?.endpoints.hub ?? '',
        },
      }
    },
    async listTerminals(options) {
      const session = await sessionManager.get({
        forceRelay: options?.forceRelay,
        onStatus: options?.onStatus,
        onConnectionState: options?.onConnectionState,
      })
      try {
        const response = await session.execute(create(AnyTTYApiApplication.CommandEnvelopeSchema, {
          command: { case: 'terminalList', value: create(AnyTTYApiTerminal.TerminalListCommandSchema) },
        }))
        if (response.result.case !== 'terminalList') throw new Error('terminal list returned no result')
        return normalizeTerminalInventory({
          machine_id: machine.id,
          terminals: response.result.value.terminals.map((terminal) => ({
            terminal_id: terminal.ref?.terminalId ?? '',
            name: terminal.name,
            state: terminal.state === AnyTTYApiTerminal.TerminalState.RUNNING ? 'running' : terminal.state === AnyTTYApiTerminal.TerminalState.EXITED ? 'exited' : 'unknown',
            command: terminal.command,
            cwd: terminal.cwd,
            live_cwd: terminal.liveCwd,
            cols: terminal.size?.cols ?? 0,
            rows: terminal.size?.rows ?? 0,
            foreground_process: terminal.foregroundProcess || undefined,
            last_output_at: unixNanoISOString(terminal.lastOutputAtUnixNano),
          })),
        }).terminals
      } finally {
        await session.close()
      }
    },
  }

  return {
    api,
    connector: {
      connect(target, options) {
        if (target.machineId !== machine.id) {
          throw new Error(`machine runtime mismatch: ${target.machineId} != ${machine.id}`)
        }
        const sessionPromise = sessionManager.lease(options)
        return sessionPromise
      },
      reconnect(options) {
    return sessionManager.resetClientOnly(options)
      },
      getConnectionPolicy: (signal) => connector.getConnectionPolicy?.(signal) ?? Promise.reject(new Error('Connection policy is unavailable')),
      applyConnectionPolicy: (policy, signal) => connector.applyConnectionPolicy?.(policy, signal) ?? Promise.reject(new Error('Connection policy is unavailable')),
      routeManagement: {
        async load(signal) {
          const registry = await goBindingClient.getEndpointRegistry(signal)
          const endpoint = registry.endpoints.find((candidate) => candidate.endpointId === machine.id)
          if (!endpoint) throw new Error('Endpoint is not configured')
          return create(AnyTTYRemoteAuth.EndpointConfigV1Schema, endpoint)
        },
        async save(endpoint, signal) {
          await sessionManager.reset()
          const result = await goBindingClient.upsertEndpoint(endpoint, false, signal)
          if (result.registry) endpointRegistry.replace(result.registry)
          if (!result.endpoint) throw new Error('Endpoint update returned no endpoint')
          return result.endpoint
        },
        async test(routeId, signal) {
          await sessionManager.reset()
          const routeConnector = new GoBindingConnector(() => goBindingClient, { endpointId: machine.id, routeId })
          const session = await routeConnector.connect({ machineId: machine.id }, { signal })
          await session.close()
        },
        async provisionSSH(routeId, signal) {
          await sessionManager.reset()
          const result = await goBindingClient.provisionSSHCredential(machine.id, routeId, signal)
          if (result.registry) endpointRegistry.replace(result.registry)
          return result
        },
      },
    },
    inventoryEvents: createNativeInventoryEvents(machine.id, sessionManager),
    connectionStateEvents: createNativeConnectionStateEvents(machine.id, sessionManager),
    listConnectionState: sessionManager.connectionState,
    probeConnection: () => sessionManager.probe(),
    fileTransfer: createFileTransferContext(machine.id, transferStore),
    async disconnect() {
      await sessionManager.disconnect()
    },
    dispose: () => {
      if (shared.sessionManagers.get(machine.id)?.manager === sessionManager) {
        shared.sessionManagers.delete(machine.id)
      }
      return sessionManager.reset().finally(() => connector.release?.(machine.id))
    },
  }
}

function unixNanoISOString(value: bigint): string | undefined {
  if (value <= 0n) return undefined
  return new Date(Number(value / 1_000_000n)).toISOString()
}

function createNativeConnectionStateEvents(
  machineId: string,
  sessionManager: NativeSessionManager,
): MachineConnectionStateEvents {
  return {
    subscribe(targetMachineId, handler) {
      if (targetMachineId !== machineId) return { close() {} }
      let closed = false
      const publish = () => {
        if (closed) return
        const snapshot = sessionManager.connectionState.getSnapshot()
        handler({
          machineId,
          phase: snapshot.phase,
          statusText: snapshot.statusText,
          relayInUse: snapshot.relayInUse,
          ...(snapshot.connectionInfo?.path ? { path: snapshot.connectionInfo.path } : {}),
          ...(snapshot.connectionInfo?.observedPath ? { observedPath: snapshot.connectionInfo.observedPath } : {}),
          ...(snapshot.connectionInfo?.routeSelectionReason ? { routeSelectionReason: snapshot.connectionInfo.routeSelectionReason } : {}),
          ...(snapshot.error ? { error: snapshot.error } : {}),
        })
      }
      const unsubscribe = sessionManager.connectionState.subscribe(publish)
      queueMicrotask(publish)
      return {
        close() {
          closed = true
          unsubscribe()
        },
      }
    },
  }
}

function createFileTransferContext(machineId: string | undefined, store: NativeFileTransferStore): FileTransferContext {
  return {
    subscribe: (listener) => store.subscribe(listener),
    getSnapshot: () => store.getSnapshot(machineId),
    isNative: true,
    getDownloadResumeOffset(mid, filePath, fileSize) {
      return store.getDownloadResumeOffset(mid, filePath, fileSize)
    },
    startDownload(mid, fileName, fileSize, filePath, offset) {
      store.startDownload(mid, fileName, fileSize, filePath, offset)
    },
    startUpload(mid, files, targetDir) {
      for (const f of files) {
        store.startUpload(mid, f.uri, f.name, f.size, targetDir)
      }
    },
    pickAndUpload(mid, targetDir) {
      runAcrossNativePicker(nativeForegroundBarrier, () => NativeFilePicker.pickFiles({ multiple: true })).then((result) => {
        // SAF 返回后只保存平台 URI；upload session 必须在新 foreground generation 上重新取得。
        for (const f of result.files) {
          store.startUpload(mid, f.uri, f.name, f.size, targetDir)
        }
      }).catch(() => {})
    },
    pauseTransfer(id) { store.pauseTransfer(id) },
    resumeTransfer(id) { store.resumeTransfer(id) },
    resumeAllTransfers(machineId) { store.resumeAllTransfers(machineId) },
    openDownloadedFile(id) { return store.openDownloadedFile(id) },
    cancelTransfer(id) { store.cancelTransfer(id) },
    dismissTransfer(id) { store.dismissTransfer(id) },
  }
}

function createNativeConnector(
  machine: RemoteMachine,
  endpointRegistry: NativeEndpointRegistryProjection,
  onPlatform: (platform: string) => void,
): NativeSessionConnector {
  if (!endpointRegistry.has(machine.id)) {
    return {
      async connect() {
        throw Object.assign(new Error('Endpoint requires a valid Proto configuration from the pairing flow'), {
          code: 'endpoint_not_configured',
          retryable: false,
        })
      },
    }
  }

  const connector = new GoBindingConnector(() => goBindingClient, {
  endpointId: machine.id,
  })
  return {
    connect: (target, options) => connector.connect(target, options),
    async verify(session, signal) {
      const response = await session.execute(create(AnyTTYApiApplication.CommandEnvelopeSchema, {
        command: { case: 'terminalDefaults', value: create(AnyTTYApiTerminal.TerminalDefaultsCommandSchema) },
      }), { signal })
      if (response.result.case !== 'terminalDefaults') throw new Error('session verification returned no terminal defaults')
      onPlatform(response.result.value.defaults?.platform ?? '')
    },
    getConnectionPolicy: (signal) => connector.getConnectionPolicy(signal),
    applyConnectionPolicy: (policy, signal) => connector.applyConnectionPolicy(policy, signal),
    disconnect: (machineId) => {
      if (machineId !== machine.id) return Promise.reject(new Error('endpoint identity mismatch'))
      return goBindingClient.disconnectEndpoint(machine.id)
    },
    setActive: (machineId, active) => NativeConnection.setSessionActive({ machineId, active }),
  }
}

function normalizeDaemonPlatform(value: string): string | undefined {
  const normalized = value.trim().toLowerCase()
  return normalized.length > 0 && normalized.length <= 32 && /^[a-z0-9_-]+$/.test(normalized)
    ? normalized
    : undefined
}

function createNativeInventoryEvents(
  machineId: string,
  sessionManager: NativeSessionManager,
): TerminalInventoryEvents {
  return {
    subscribe(targetMachineId, handler) {
      if (targetMachineId !== machineId) return { close() {} }
      let closed = false
      let subscription: RtcSubscription | null = null
      let session: NativeSessionLease | null = null
      let sequence = 0
      let targetConnectionId: string | null = null

      const detach = () => {
        sequence += 1
        targetConnectionId = null
        subscription?.close()
        subscription = null
        void session?.close().catch(() => undefined)
        session = null
      }
      const synchronize = () => {
        if (closed) return
        const snapshot = sessionManager.connectionState.getSnapshot()
        const connectionId = snapshot.phase === 'connected'
          ? snapshot.connectionInfo?.connectionId ?? null
          : null
        if (!connectionId) {
          if (targetConnectionId !== null || subscription !== null || session !== null) detach()
          return
        }
        if (targetConnectionId === connectionId) return

        detach()
        targetConnectionId = connectionId
        const attachSequence = sequence
        void sessionManager.get().then(async (connectedSession) => {
          if (closed || attachSequence !== sequence || targetConnectionId !== connectionId) {
            await connectedSession.close().catch(() => undefined)
            return
          }
          let connectedSubscription: RtcSubscription
          try {
            connectedSubscription = await openProtoEventSubscription(connectedSession, create(AnyTTYApiEvents.EventSubscribeCommandSchema, {
              types: [AnyTTYApiEvents.ApplicationEventType.TERMINAL_LIFECYCLE],
            }), (event) => {
              if (event.event.case === 'terminalLifecycle') handler({ type: 'inventory_changed', payload: event.event.value })
            })
          } catch {
            await connectedSession.close().catch(() => undefined)
            if (attachSequence === sequence && targetConnectionId === connectionId) targetConnectionId = null
            return
          }
          if (closed || attachSequence !== sequence || targetConnectionId !== connectionId) {
            connectedSubscription.close()
            await connectedSession.close().catch(() => undefined)
            return
          }
          session = connectedSession
          subscription = connectedSubscription
        }).catch(() => {
          if (attachSequence === sequence && targetConnectionId === connectionId) targetConnectionId = null
        })
      }
      const unsubscribe = sessionManager.connectionState.subscribe(synchronize)
      queueMicrotask(synchronize)
      return {
        close() {
          if (closed) return
          closed = true
          unsubscribe()
          detach()
        },
      }
    },
  }
}

function firstNonEmpty(values: readonly (string | undefined)[]): string | undefined {
  return compactStrings(values)[0]
}

function compactStrings(values: readonly (string | undefined)[]): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const raw of values) {
    const value = raw?.trim()
    if (!value || seen.has(value)) continue
    seen.add(value)
    out.push(value)
  }
  return out
}

function isTerminalInventoryEvent(event: RtcEvent): boolean {
  return event.type === 'inventory_changed' ||
    event.type === 'terminal_changed' ||
    event.type === 'terminal_created' ||
    event.type === 'terminal_state_changed' ||
    event.type === 'terminal_resized' ||
    event.type === 'terminal_removed' ||
    event.type === 'terminal_metadata_changed'
}

function abortError(signal: AbortSignal): Error {
  return signal.reason instanceof Error ? signal.reason : new Error('connection aborted')
}

function headersRecord(headers: HeadersInit | undefined): Record<string, string> {
  if (!headers) return {}
  if (headers instanceof Headers) return Object.fromEntries(headers.entries())
  if (Array.isArray(headers)) return Object.fromEntries(headers)
  return { ...headers }
}

function requestData(body: BodyInit | null | undefined): string | undefined {
  if (body === undefined || body === null) return undefined
  if (typeof body === 'string') return body
  throw new Error('native fetch only supports string request bodies')
}

function responseText(data: unknown): string {
  if (data === undefined || data === null) return ''
  if (typeof data === 'string') return data
  return JSON.stringify(data)
}
