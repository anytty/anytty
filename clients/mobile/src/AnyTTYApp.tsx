import { useCallback, useEffect, useMemo, useReducer, useRef, useState, type CSSProperties } from 'react'
import { App as CapApp } from '@capacitor/app'
import { Capacitor, CapacitorHttp } from '@capacitor/core'
import { Clipboard } from '@capacitor/clipboard'
import { Keyboard } from '@capacitor/keyboard'
import { Network } from '@capacitor/network'
import { Browser } from '@capacitor/browser'
import { create, toBinary } from '@bufbuild/protobuf'
import {
  RemoteControlApp,
  appThemeCssVariables,
  createMachineStore,
  dispatchNativeKeyboardEvent,
  normalizeTerminalInventory,
  openProtoEventSubscription,
  readAppTheme,
  readActiveWorkspaceMachineId,
  writeActiveWorkspaceMachineId,
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
  AppConnectionState,
} from '@anytty/ui'
import {
  NativeConnection,
  type NativeDisconnectAllRequestedEvent,
  type NativeNetworkChangedEvent,
} from './plugins/nativeConnection'
import { NativeFileTransferStore } from './NativeFileTransferStore'
import { GoBindingClient, GoBindingConnector } from './GoBindingClient'
import { settleBindingGeneration } from './BindingGeneration'
import { NativeBindingGenerationReplacement, drainNativeGenerationReset } from './NativeBindingGenerationReplacement'
import { NativeSessionManager, type NativeSessionConnector } from './NativeSessionManager'
import { nativeSessionDemand, type NativeDisconnectAllCleanupRequest } from './NativeSessionDemand'
import NativeFilePicker from './plugins/nativeFilePicker'
import { useNativeStatusBarSync } from './nativeStatusBar'
import { NativeForegroundBarrier, runAcrossNativePicker } from './NativeForegroundBarrier'
import { NativeGenerationRecoveryFence } from './NativeGenerationRecoveryFence'
import {
  NativeRecoveryCoordinator,
  startNativeForegroundWork,
  type NativeRecoveryRequest,
  type NativeRecoveryWork,
} from './NativeRecoveryCoordinator'
import { NATIVE_RECOVERY_NOTICE_DELAY_MS, reduceNativeRecoveryStatus, type NativeRecoveryStatusEvent } from './NativeRecoveryStatus'
import { RegistryStartupScreen, UnsupportedWebPreview } from './RegistryStartupScreen'
import { useAndroidBackButton } from './androidBack'
import type { NativeQrScannerOptions } from './nativeQrScanner'
import { endpointMachineAccessClass } from './endpointMachineProjection'
import { cloudPresenceWithState, mergeCloudPresenceResults, samePresenceMap, type CloudPresenceState } from './cloudPresenceState'
import { LocalWebAnyTTYApp, parseLocalWebBootstrap, type LocalWebBootstrap } from './LocalWebAnyTTYApp'
import { LocalWebLogin } from './LocalWebLogin'

const nativeHttpConnectTimeoutMs = 8_000
const nativeHttpReadTimeoutMs = 15_000
const directReachabilityFeedbackDelayMs = 300
const directReachabilitySearchWindowMs = 6_000
const directReachabilityRefreshIntervalMs = 3_000
const cloudPresenceFeedbackDelayMs = 300
const cloudPresenceProbeTimeoutMs = 15_000
const cloudPresenceRefreshIntervalMs = 20_000
const nativeRecoveryStepTimeoutMs = 20_000
const rendererStallReconcileMs = 10_000
const nativeRecoveryRetryDelaysMs = [250, 500, 1_000, 2_000, 5_000, 15_000] as const
const nativeDisconnectAllRetryDelaysMs = [250, 500, 1_000, 2_000, 5_000, 15_000] as const
const privacyPolicyUrl = 'https://anytty.com/privacy/'
let goBindingClient = new GoBindingClient()
const nativeSystemClipboard = {
  async readText() {
    return (await Clipboard.read()).value
  },
  async writeText(text: string) {
    await Clipboard.write({ string: text })
  },
}

type MachineRuntimeFactory = NonNullable<RemoteControlAppProps['machineRuntimeFactory']>
type MachineRuntime = ReturnType<MachineRuntimeFactory>
type NativeSessionEntry = {
  endpointIdentity: string
  connector: NativeSessionConnector
  manager: NativeSessionManager
}
type NativeSessionLease = ProtoClientSession

/** Prevents delayed user-stop cleanup from overwriting newer connection intent. */
export class NativeDeferredDisconnectFence {
  private readonly endpointGenerations = new Map<string, number>()

  markFreshIntent(endpointId: string): void {
    this.endpointGenerations.set(endpointId, this.generation(endpointId) + 1)
  }

  async run(
    endpointIds: readonly string[],
    suspend: () => Promise<void>,
    disconnect: (endpointId: string) => Promise<void>,
  ): Promise<number> {
    const stopGenerations = new Map(endpointIds.map((endpointId) => [endpointId, this.generation(endpointId)]))
    await suspend()
    let disconnected = 0
    const results = await Promise.allSettled(endpointIds.map(async (endpointId) => {
      if (stopGenerations.get(endpointId) !== this.generation(endpointId)) return
      await disconnect(endpointId)
      disconnected += 1
    }))
    throwNativeCleanupFailures('Native endpoint Stop cleanup failed', results)
    return disconnected
  }

  private generation(endpointId: string): number {
    return this.endpointGenerations.get(endpointId) ?? 0
  }
}

/** Serializes retained Stop notifications and retries until JS cleanup is committed. */
export class NativeDisconnectAllRequestProcessor {
  private cleanup: ((request: NativeDisconnectAllCleanupRequest) => Promise<void>) | null = null
  private pendingEvent: NativeDisconnectAllRequestedEvent | null = null
  private active = false
  private retryAttempt = 0
  private retryTimer: ReturnType<typeof globalThis.setTimeout> | null = null

  constructor(
    private readonly canonicalize: (event: NativeDisconnectAllRequestedEvent) => Promise<NativeDisconnectAllCleanupRequest | null>,
    private readonly commit: (stopEpoch: string) => Promise<void>,
    private readonly retryDelays: readonly number[] = nativeDisconnectAllRetryDelaysMs,
  ) {}

  setCleanup(cleanup: ((request: NativeDisconnectAllCleanupRequest) => Promise<void>) | null): void {
    this.cleanup = cleanup
    if (cleanup === null) {
      this.cancelRetry()
      return
    }
    this.pump()
  }

  enqueue(event: NativeDisconnectAllRequestedEvent): void {
    const incomingEpoch = disconnectAllEventEpoch(event)
    const pendingEpoch = this.pendingEvent ? disconnectAllEventEpoch(this.pendingEvent) : null
    if (this.pendingEvent === null || incomingEpoch === null || pendingEpoch === null || incomingEpoch >= pendingEpoch) {
      this.pendingEvent = event
    }
    this.pump()
  }

  private pump(): void {
    if (this.active || this.retryTimer !== null || this.cleanup === null || this.pendingEvent === null) return
    const event = this.pendingEvent
    const cleanup = this.cleanup
    this.active = true
    void this.canonicalize(event).then(async (request) => {
      if (request !== null) {
        await cleanup(request)
        await this.commit(request.stopEpoch)
      }
      this.dropPendingThrough(request?.stopEpoch ?? event.stopEpoch, event)
      this.retryAttempt = 0
    }).then(() => {
      this.active = false
      this.pump()
    }, () => {
      this.active = false
      if (this.cleanup === null) return
      const delay = this.retryDelays[Math.min(this.retryAttempt, this.retryDelays.length - 1)] ?? 1_000
      this.retryAttempt += 1
      this.retryTimer = globalThis.setTimeout(() => {
        this.retryTimer = null
        this.pump()
      }, delay)
    })
  }

  private dropPendingThrough(stopEpoch: string, processedEvent: NativeDisconnectAllRequestedEvent): void {
    if (this.pendingEvent === null) return
    const completed = /^\d+$/.test(stopEpoch) ? BigInt(stopEpoch) : null
    const pending = disconnectAllEventEpoch(this.pendingEvent)
    if (this.pendingEvent === processedEvent || completed !== null && pending !== null && pending <= completed) {
      this.pendingEvent = null
    }
  }

  private cancelRetry(): void {
    if (this.retryTimer === null) return
    globalThis.clearTimeout(this.retryTimer)
    this.retryTimer = null
  }
}

function throwNativeCleanupFailures(label: string, results: readonly PromiseSettledResult<unknown>[]): void {
  const failures = results.flatMap((result) => result.status === 'rejected' ? [result.reason] : [])
  if (failures.length === 0) return
  if (failures.length === 1) throw failures[0]
  throw new AggregateError(failures, label)
}

function disconnectAllEventEpoch(event: NativeDisconnectAllRequestedEvent): bigint | null {
  const value = event?.stopEpoch?.trim()
  return value && /^\d+$/.test(value) ? BigInt(value) : null
}

type NativeEndpointStopCleanupEntry = {
  machineId: string
  resumeIntent: object | null
  isCurrent(): boolean
  adoptUserStop(): Promise<void>
}

/** Persists per-endpoint Stop state after another endpoint reopens the process-wide native gate. */
export class NativeEndpointStopRegistry {
  private readonly stoppedEpochs = new Map<string, string>()

  constructor(
    private readonly resumeIntentCoversStopEpoch: (intent: object | null, stopEpoch: string) => boolean,
  ) {}

  latchStop(
    stopEpoch: string,
    entries: readonly Pick<NativeEndpointStopCleanupEntry, 'machineId' | 'resumeIntent'>[],
    canonicalProtectedMachineIds: ReadonlySet<string> = new Set<string>(),
  ): ReadonlySet<string> {
    const protectedMachineIds = new Set(canonicalProtectedMachineIds)
    for (const entry of entries) {
      if (canonicalProtectedMachineIds.has(entry.machineId)) {
        this.stoppedEpochs.delete(entry.machineId)
        continue
      }
      const latchedEpoch = this.stoppedEpochs.get(entry.machineId)
      const effectiveStopEpoch = latchedEpoch && BigInt(latchedEpoch) > BigInt(stopEpoch)
        ? latchedEpoch
        : stopEpoch
      if (
        this.resumeIntentCoversStopEpoch(entry.resumeIntent, stopEpoch) &&
        this.resumeIntentCoversStopEpoch(entry.resumeIntent, effectiveStopEpoch)
      ) {
        protectedMachineIds.add(entry.machineId)
        this.stoppedEpochs.delete(entry.machineId)
        continue
      }
      this.stoppedEpochs.set(entry.machineId, effectiveStopEpoch)
    }
    return protectedMachineIds
  }

  adoptIfStopped(machineId: string, adoptUserStop: () => Promise<void>): Promise<void> | null {
    if (!this.stoppedEpochs.has(machineId)) return null
    let operation: Promise<void>
    try {
      operation = adoptUserStop()
    } catch (failure) {
      operation = Promise.reject(failure)
    }
    void operation.catch(() => undefined)
    return operation
  }

  acceptUserResume(machineId: string, intent: object): void {
    const stopEpoch = this.stoppedEpochs.get(machineId)
    if (stopEpoch && this.resumeIntentCoversStopEpoch(intent, stopEpoch)) {
      this.stoppedEpochs.delete(machineId)
    }
  }

  isStopped(machineId: string): boolean { return this.stoppedEpochs.has(machineId) }
}

type NativeWorkspaceResumeIntentManager = Pick<NativeSessionManager, 'beginUserConnectionIntent'>

/** Bridges the eager device-list action to a manager that may be created behind React Suspense. */
export class NativeWorkspaceResumeIntentRegistry {
  private readonly pendingIntents = new Map<string, object>()
  private readonly registeredIntents = new WeakMap<object, string>()

  constructor(private readonly markFreshConnectionIntent: (machineId: string) => void) {}

  register(machineId: string, intent: object, manager?: NativeWorkspaceResumeIntentManager): void {
    const registeredMachineId = this.registeredIntents.get(intent)
    if (registeredMachineId !== undefined && registeredMachineId !== machineId) {
      throw new Error('Native resume intent cannot move between endpoints')
    }
    if (registeredMachineId === undefined) {
      this.registeredIntents.set(intent, machineId)
      this.markFreshConnectionIntent(machineId)
    }
    this.pendingIntents.set(machineId, intent)
    manager?.beginUserConnectionIntent(intent)
  }

  attachManager(machineId: string, manager: NativeWorkspaceResumeIntentManager): void {
    const intent = this.pendingIntents.get(machineId)
    if (intent) manager.beginUserConnectionIntent(intent)
  }

  consume(machineId: string, intent: object): void {
    if (this.pendingIntents.get(machineId) === intent) this.pendingIntents.delete(machineId)
  }

  clear(machineId?: string): void {
    if (machineId) this.pendingIntents.delete(machineId)
    else this.pendingIntents.clear()
  }

  entries(): Array<{ machineId: string; resumeIntent: object }> {
    return [...this.pendingIntents].map(([machineId, resumeIntent]) => ({ machineId, resumeIntent }))
  }

  currentIntent(machineId: string): object | null {
    return this.pendingIntents.get(machineId) ?? null
  }

  isRegistered(machineId: string, intent: object): boolean {
    return this.registeredIntents.get(intent) === machineId
  }

  isPending(machineId: string, intent: object): boolean {
    return this.pendingIntents.get(machineId) === intent
  }
}

/** Owns one exact native resume intent across an asynchronous transfer UI flow. */
export class NativeTransferIntentCoordinator {
  private readonly owners = new WeakMap<object, string>()

  constructor(
    private readonly beginIntent: (machineId: string) => object,
    private readonly confirmIntent: (machineId: string, intent: object) => Promise<void>,
    private readonly finishIntent: (machineId: string, intent: object) => void,
  ) {}

  begin(machineId: string): object {
    const intent = this.beginIntent(machineId)
    this.owners.set(intent, machineId)
    return intent
  }

  async confirm(machineId: string, intent: object): Promise<void> {
    if (this.owners.get(intent) !== machineId) throw new Error('Native transfer intent is stale')
    await this.confirmIntent(machineId, intent)
  }

  finish(machineId: string, intent: object): void {
    if (this.owners.get(intent) !== machineId) return
    this.owners.delete(intent)
    this.finishIntent(machineId, intent)
  }

  async run<T>(machineId: string, intent: object, commit: () => T | Promise<T>): Promise<T> {
    try {
      await this.confirm(machineId, intent)
      return await commit()
    } finally {
      this.finish(machineId, intent)
    }
  }
}

export async function runNativeEndpointStopCleanup(
  stopEpoch: string,
  entries: readonly NativeEndpointStopCleanupEntry[],
  endpointStops: NativeEndpointStopRegistry,
  suspendTransfers: (protectedMachineIds: ReadonlySet<string>) => Promise<void>,
  disconnectFence: NativeDeferredDisconnectFence,
  canonicalProtectedMachineIds: ReadonlySet<string> = new Set<string>(),
): Promise<void> {
  const protectedMachineIds = endpointStops.latchStop(stopEpoch, entries, canonicalProtectedMachineIds)
  const stoppedEntries = new Map(entries
    .filter((entry) => !protectedMachineIds.has(entry.machineId))
    .map((entry) => [entry.machineId, entry]))
  const adoptions = new Map([...stoppedEntries].map(([machineId, entry]) => [
    machineId,
    endpointStops.adoptIfStopped(machineId, () => entry.adoptUserStop()),
  ]))
  await disconnectFence.run(
    [...stoppedEntries.keys()],
    () => suspendTransfers(protectedMachineIds),
    async (machineId) => {
      const entry = stoppedEntries.get(machineId)
      if (!entry?.isCurrent()) return
      await adoptions.get(machineId)
    },
  )
}

export function AnyTTYApp() {
  const initialAppThemeStyle = useMemo(
    () => appThemeCssVariables(readAppTheme()) as CSSProperties,
    [],
  )
  if (!Capacitor.isNativePlatform()) {
    return <BrowserAnyTTYApp initialAppThemeStyle={initialAppThemeStyle} />
  }
  return <NativeAnyTTYApp initialAppThemeStyle={initialAppThemeStyle} />
}

function BrowserAnyTTYApp({ initialAppThemeStyle }: { initialAppThemeStyle: CSSProperties }) {
  const [bootstrap, setBootstrap] = useState<LocalWebBootstrap | null | undefined>(undefined)
  const [authenticationRequired, setAuthenticationRequired] = useState(false)
  const loadBootstrap = useCallback(async (signal?: AbortSignal) => {
    const response = await fetch('/api/bootstrap', { cache: 'no-store', signal })
    if (response.status === 401) {
      setBootstrap(null)
      setAuthenticationRequired(true)
      return false
    }
    if (!response.ok || !response.headers.get('content-type')?.includes('application/json')) {
      setBootstrap(null)
      setAuthenticationRequired(false)
      return false
    }
    const value = parseLocalWebBootstrap(await response.json())
    setBootstrap(value)
    setAuthenticationRequired(false)
    return value !== null
  }, [])
  useEffect(() => {
    const controller = new AbortController()
    void loadBootstrap(controller.signal)
      .catch(() => {
        if (!controller.signal.aborted) setBootstrap(null)
      })
    return () => controller.abort()
  }, [loadBootstrap])
  if (bootstrap === undefined) return <div className="h-full" style={initialAppThemeStyle} />
  if (authenticationRequired) {
    return <LocalWebLogin initialAppThemeStyle={initialAppThemeStyle} onAuthenticated={async () => {
      if (!await loadBootstrap()) throw new Error('local Web bootstrap is unavailable')
    }} />
  }
  if (bootstrap === null) return <div className="h-full" style={initialAppThemeStyle}><UnsupportedWebPreview /></div>
  return <LocalWebAnyTTYApp bootstrap={bootstrap} initialAppThemeStyle={initialAppThemeStyle} />
}

function NativeAnyTTYApp({ initialAppThemeStyle }: { initialAppThemeStyle: CSSProperties }) {
  useAndroidBackButton()
  useNativeKeyboardEvents()
  useNativeStatusBarSync()

  const networkRuntime = useMemo(() => createNativeNetworkRuntime(), [])
  const endpointRegistry = useMemo(() => new NativeEndpointRegistryProjection(), [])
  const nativeAppRuntime = useMemo(() => createNativeAppRuntime(endpointRegistry), [endpointRegistry])
  const [registryReady, setRegistryReady] = useState(false)
  const [registryError, setRegistryError] = useState<string | null>(null)
  const rendererDemandRestoredRef = useRef(false)
  const directReachability = useNativeDirectReachability(endpointRegistry, registryReady)
  const cloudPresenceByMachineId = useNativeCloudPresence(endpointRegistry, registryReady)
  const refreshRegistry = useCallback(async (
    client: GoBindingClient = goBindingClient,
    signal?: AbortSignal,
  ) => {
    try {
      throwIfAborted(signal)
      const loaded = await settleBindingGeneration(
        client,
        () => goBindingClient,
        () => client.getEndpointRegistry(signal),
      )
      throwIfAborted(signal)
      if (!loaded.current) return
      endpointRegistry.replace(loaded.value)
      if (networkRuntime.storage) syncRegistryMachineProjection(networkRuntime.storage, endpointRegistry.snapshot())
      if (!rendererDemandRestoredRef.current) {
        rendererDemandRestoredRef.current = true
        const restoredMachineId = restorableNativeWorkspaceMachineId(networkRuntime.storage, endpointRegistry)
        if (!restoredMachineId) writeActiveWorkspaceMachineId(networkRuntime.storage, null)
        await nativeSessionDemand.restoreRenderer(restoredMachineId ? [restoredMachineId] : []).catch((failure) => {
          nativeDiagnostic('renderer_demand_restore_deferred', { failure: diagnosticFailureCode(failure) })
        })
      }
      throwIfAborted(signal)
      setRegistryError(null)
      setRegistryReady(true)
    } catch (error) {
      if (signal?.aborted) throw abortError(signal)
      setRegistryError(error instanceof Error ? error.message : String(error))
      throw error
    }
  }, [endpointRegistry, networkRuntime])
  useEffect(() => { void refreshRegistry().catch(() => undefined) }, [refreshRegistry])
  useEffect(() => {
    if (!registryReady) return
    let disposed = false
    queueMicrotask(() => {
      if (!disposed) void nativeSessionDemand.reconcileRenderer().catch(() => undefined)
    })
    return () => { disposed = true }
  }, [registryReady])
  const nativeConnectionRecovery = useNativeNetworkRecovery(
    refreshRegistry,
    nativeAppRuntime.resetGeneration,
    nativeAppRuntime.foregroundResume,
    nativeAppRuntime.resumeInterruptedTransfers,
    nativeAppRuntime.networkChanged,
    nativeAppRuntime.initializeNetworkState,
  )
  useNativeDisconnectAll(nativeAppRuntime.disconnectAll, nativeConnectionRecovery.cancelForStop)
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
      await withNativeRecoveryTimeout(
        async () => await NativeConnection.handleForegroundResume(),
        'Native runtime recovery',
      )
      await withNativeRecoveryTimeout(
        (signal) => replaceNativeGeneration(refreshRegistry, nativeAppRuntime.resetGeneration, signal),
        'Native binding replacement',
      )
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
      writeActiveWorkspaceMachineId(networkRuntime.storage, null)
      await nativeSessionDemand.setWorkspaceEndpoint(null)
      endpointRegistry.replace(create(AnyTTYRemoteAuth.EndpointRegistryV1Schema, { schemaVersion: 1 }))
      await withNativeRecoveryTimeout(
        (signal) => replaceNativeGeneration(refreshRegistry, nativeAppRuntime.resetGeneration, signal),
        'Native binding replacement',
      )
    } catch (failure) {
      const message = failure instanceof Error ? failure.message : String(failure)
      setRegistryError(message)
      throw failure
    }
  }, [endpointRegistry, nativeAppRuntime.discardLocalState, nativeAppRuntime.resetGeneration, networkRuntime, refreshRegistry])

  if (!registryReady) {
    return (
      <div className="h-full" style={initialAppThemeStyle}>
        <RegistryStartupScreen
          error={registryError}
          onResetLocalPairings={resetLocalPairings}
          onRetry={retryRegistry}
        />
      </div>
    )
  }

  return (
    <section className="anytty-native-app-frame bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex h-full w-full flex-col overflow-hidden antialiased" style={initialAppThemeStyle}>
      <RemoteControlApp
        singlePaneWorkspace
        externalPairingAdapter={externalPairingAdapter}
        globalFileTransfer={globalFileTransfer}
        machineRuntimeFactory={machineRuntimeFactory}
        networkRuntime={networkRuntime}
        phoneOnline={nativeConnectionRecovery.phoneOnline}
        directReachableMachineIds={directReachability.reachableMachineIds}
        directCheckingMachineIds={directReachability.checkingMachineIds}
        cloudPresenceByMachineId={cloudPresenceByMachineId}
        connectionState={nativeConnectionRecovery.connectionState}
        onRetryConnectionRecovery={nativeConnectionRecovery.retryConnectionRecovery}
        createWorkspaceResumeIntent={() => nativeSessionDemand.createResumeIntent()}
        onWorkspaceResumeIntent={(machineId, intent) => nativeAppRuntime.registerWorkspaceResumeIntent(machineId, intent)}
        onActiveWorkspaceChange={(machineId) => {
          if (machineId === null) nativeAppRuntime.clearPendingWorkspaceResumeIntents()
          return nativeSessionDemand.setWorkspaceEndpoint(machineId)
        }}
        onRefreshMachines={() => refreshRegistry()}
        pickMachineIconImage={pickNativeMachineIconImage}
        scanPairingCode={scanNativePairingCode}
        exportDebugLogs={Capacitor.getPlatform() === 'android'
          ? async () => { await NativeConnection.shareDiagnosticBundle() }
          : undefined}
        privacyPolicyUrl={privacyPolicyUrl}
        onOpenPrivacyPolicy={() => Browser.open({ url: privacyPolicyUrl })}
        systemClipboard={nativeSystemClipboard}
      />
    </section>
  )
}

function scanNativePairingCode(options?: NativeQrScannerOptions): Promise<string | null> {
  return import('./nativeQrScanner').then(({ scanPairingCode }) => scanPairingCode(options))
}

function restorableNativeWorkspaceMachineId(
  storage: RemoteRuntimeStorage | undefined,
  endpointRegistry: NativeEndpointRegistryProjection,
): string | null {
  const machineId = readActiveWorkspaceMachineId(storage)
  if (!machineId || !storage || !endpointRegistry.has(machineId) || !endpointRegistry.isAuthorized(machineId)) return null
  try {
    return createMachineStore({ storage }).getMachine(machineId) ? machineId : null
  } catch {
    return null
  }
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
    const expiresAt = imported.expiresAtUnixNano > 0n
      ? new Date(Number(imported.expiresAtUnixNano / 1_000_000n)).toISOString()
      : undefined
    registry.replace(imported.registry ?? await goBindingClient.getEndpointRegistry())
    registry.setAuthorizationExpiry(endpoint.endpointId, expiresAt)
    return {
      machine: { id: endpoint.endpointId, name: endpoint.label || endpoint.endpointId, osInfo: endpoint.platform || undefined, accessClass: endpointMachineAccessClass(endpoint) },
      ...(expiresAt ? { expiresAt } : {}),
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
  directReachabilityTargets(): Array<{
    machineId: string
    identity?: { deviceId: string; fingerprint: string }
    routeProtoBase64: string[]
  }> {
    return this.registry.endpoints.flatMap((endpoint) => {
      const deviceId = endpoint.identity?.deviceId.trim() ?? ''
      const fingerprint = endpoint.identity?.deviceFingerprint.trim() ?? ''
      const routeProtoBase64 = endpoint.routes.flatMap((route) => {
        if (!route.enabled || route.route.case !== 'directWebrtcTcp') return []
        return [directRouteProtoBase64(route.route.value)]
      })
      if (!endpoint.endpointId || ((!deviceId || !fingerprint) && routeProtoBase64.length === 0)) return []
      return [{
        machineId: endpoint.endpointId,
        ...(deviceId && fingerprint ? { identity: { deviceId, fingerprint } } : {}),
        routeProtoBase64,
      }]
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

function useNativeDirectReachability(
  registry: NativeEndpointRegistryProjection,
  enabled: boolean,
): { reachableMachineIds: ReadonlySet<string>; checkingMachineIds: ReadonlySet<string> } {
  const [view, setView] = useState<{ reachableMachineIds: ReadonlySet<string>; checkingMachineIds: ReadonlySet<string> }>(() => ({
    reachableMachineIds: new Set(),
    checkingMachineIds: new Set(),
  }))
  useEffect(() => {
    if (!enabled) {
      setView({ reachableMachineIds: new Set(), checkingMachineIds: new Set() })
      return
    }
    let cancelled = false
    let revision = 0
    let probeRevision = 0
    let feedbackTimer: ReturnType<typeof setTimeout> | undefined
    let settleTimer: ReturnType<typeof setTimeout> | undefined
    let interval: ReturnType<typeof setInterval> | undefined
    const startSearch = (restartWindow = true) => {
      const targets = registry.directReachabilityTargets()
      if (!restartWindow && settleTimer) {
        void refresh(revision, targets)
        return
      }
      const currentRevision = ++revision
      if (feedbackTimer) clearTimeout(feedbackTimer)
      if (settleTimer) clearTimeout(settleTimer)
      const eligible = new Set(targets.map((target) => target.machineId))
      feedbackTimer = setTimeout(() => {
        feedbackTimer = undefined
        if (cancelled || currentRevision !== revision) return
        setView((current) => {
          const checking = new Set([...eligible].filter((machineId) => !current.reachableMachineIds.has(machineId)))
          return sameStringSet(current.checkingMachineIds, checking) ? current : { ...current, checkingMachineIds: checking }
        })
      }, directReachabilityFeedbackDelayMs)
      settleTimer = setTimeout(() => {
        settleTimer = undefined
        if (cancelled || currentRevision !== revision) return
        setView((current) => current.checkingMachineIds.size === 0 ? current : { ...current, checkingMachineIds: new Set() })
      }, directReachabilitySearchWindowMs)
      void refresh(currentRevision, targets)
    }
    const refresh = async (currentRevision: number, targets: ReturnType<NativeEndpointRegistryProjection['directReachabilityTargets']>) => {
      const currentProbeRevision = ++probeRevision
      try {
        const results = await Promise.all(targets.map(async (target) => {
          const probes: Array<Promise<boolean>> = target.routeProtoBase64.map(async (routeProtoBase64) => (
            await NativeConnection.isDirectRouteReachable({ routeProtoBase64 })
          ).reachable)
          if (target.identity) {
            probes.push(NativeConnection.isLocalEndpointDiscovered(target.identity).then((result) => result.discovered))
          }
          const outcomes = await Promise.allSettled(probes)
          return {
            machineId: target.machineId,
            reachable: outcomes.some((outcome) => outcome.status === 'fulfilled' && outcome.value),
          }
        }))
        if (cancelled || currentRevision !== revision || currentProbeRevision !== probeRevision) return
        const next = new Set(results.filter((result) => result.reachable).map((result) => result.machineId))
        setView((current) => {
          const checking = new Set([...current.checkingMachineIds].filter((machineId) => !next.has(machineId)))
          return sameStringSet(current.reachableMachineIds, next) && sameStringSet(current.checkingMachineIds, checking)
            ? current
            : { reachableMachineIds: next, checkingMachineIds: checking }
        })
      } catch {
        // The native bridge can be briefly unavailable while its generation changes.
      }
    }
    const removeRegistryListener = registry.subscribe(() => startSearch(true))
    const nativeListener = NativeConnection.addListener('localDiscoveryChanged', () => startSearch(false))
    const networkListener = NativeConnection.addListener('networkChanged', () => startSearch(true))
    void nativeListener.then(() => startSearch(true)).catch(() => undefined)
    interval = setInterval(() => {
      void refresh(revision, registry.directReachabilityTargets())
    }, directReachabilityRefreshIntervalMs)
    return () => {
      cancelled = true
      if (feedbackTimer) clearTimeout(feedbackTimer)
      if (settleTimer) clearTimeout(settleTimer)
      if (interval) clearInterval(interval)
      removeRegistryListener()
      void nativeListener.then((listener) => listener.remove())
      void networkListener.then((listener) => listener.remove())
    }
  }, [enabled, registry])
  return view
}

function directRouteProtoBase64(value: AnyTTYRemoteAuth.DirectWebRTCTCPRouteConfig): string {
  const bytes = toBinary(AnyTTYRemoteAuth.DirectWebRTCTCPRouteConfigSchema, value)
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
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
    const reportsNativeOcclusion = Capacitor.getPlatform() === 'ios'
    const subscriptions = [
      Keyboard.addListener('keyboardWillShow', (info) => {
        dispatchNativeKeyboardEvent({
          visible: true,
          ...(reportsNativeOcclusion ? {} : { keyboardHeight: info.keyboardHeight }),
        })
      }),
      Keyboard.addListener('keyboardDidShow', (info) => {
        dispatchNativeKeyboardEvent({
          visible: true,
          ...(reportsNativeOcclusion ? {} : { keyboardHeight: info.keyboardHeight }),
        })
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
  nativeDiagnostic('foreground_recovery_failure', {
    failure: diagnosticFailureCode(failure),
  })
}

function nativeDiagnostic(event: string, fields: Record<string, string | number | boolean> = {}): void {
  const details = Object.entries(fields).map(([key, value]) => `${diagnosticToken(key)}=${diagnosticToken(String(value))}`)
  const value = `event=${diagnosticToken(event)}${details.length > 0 ? ` ${details.join(' ')}` : ''}`
  writeNativeDiagnostic(value)
}

function writeNativeDiagnostic(value: string): void {
  console.info(`[anytty:diagnostic] ${value}`)
  void NativeConnection.writeDebugDiagnostic({ value }).catch(() => undefined)
}

function diagnosticFailureCode(failure: unknown): string {
  if (failure && typeof failure === 'object') {
    const code = 'code' in failure && typeof failure.code === 'string' ? failure.code : ''
    if (code.trim()) return code
    if ('name' in failure && typeof failure.name === 'string' && failure.name.trim()) return failure.name
    const constructorName = failure.constructor?.name
    if (constructorName) return constructorName
  }
  return typeof failure
}

function diagnosticToken(value: string): string {
  return value.trim().replace(/[^A-Za-z0-9_.-]+/g, '_').slice(0, 80) || 'none'
}

const nativeGenerationReplacement = new NativeBindingGenerationReplacement(
  () => goBindingClient,
  (client) => { goBindingClient = client },
  () => new GoBindingClient(),
)

function replaceNativeGeneration(
  refreshRegistry: (client?: GoBindingClient, signal?: AbortSignal) => Promise<void>,
  resetRuntime: (signal?: AbortSignal) => Promise<void>,
  signal?: AbortSignal,
): Promise<void> {
  return nativeGenerationReplacement.replace(
    resetRuntime,
    (client, replacementSignal) => refreshRegistry(client, replacementSignal),
    signal,
  )
}

export async function withNativeRecoveryTimeout<T>(
  operation: (signal: AbortSignal) => Promise<T>,
  label: string,
  parentSignal?: AbortSignal,
  timeoutMs = nativeRecoveryStepTimeoutMs,
): Promise<T> {
  const controller = new AbortController()
  let timeout: ReturnType<typeof globalThis.setTimeout> | undefined
  let removeParentAbort: (() => void) | undefined
  try {
    throwIfAborted(parentSignal)
    return await Promise.race([
      Promise.resolve().then(() => operation(controller.signal)),
      new Promise<never>((_, reject) => {
        const abortFromParent = () => {
          const failure = parentSignal ? abortError(parentSignal) : new DOMException('Aborted', 'AbortError')
          controller.abort(failure)
          reject(failure)
        }
        if (parentSignal) {
          parentSignal.addEventListener('abort', abortFromParent, { once: true })
          removeParentAbort = () => parentSignal.removeEventListener('abort', abortFromParent)
        }
        timeout = globalThis.setTimeout(() => {
          const failure = Object.assign(new Error(`${label} timed out`), {
            code: 'unavailable',
            retryable: true,
          })
          controller.abort(failure)
          reject(failure)
        }, timeoutMs)
      }),
    ])
  } finally {
    if (timeout !== undefined) globalThis.clearTimeout(timeout)
    removeParentAbort?.()
  }
}

function nativeRecoveryRetryDelay(failure: unknown, retryAttempt: number): number | null {
  const value = failure as { code?: unknown; retryable?: unknown } | null
  const code = typeof value?.code === 'string' ? value.code.trim().toLowerCase() : ''
  if (value?.retryable === false || code === 'user_stopped') return null
  return nativeRecoveryRetryDelaysMs[
    Math.min(Math.max(retryAttempt, 0), nativeRecoveryRetryDelaysMs.length - 1)
  ] ?? 15_000
}

function throwIfAborted(signal?: AbortSignal): void {
  if (signal?.aborted) throw abortError(signal)
}

/** Keep the native generation across backgrounding; replace only a bridge that actually failed. */
function useNativeNetworkRecovery(
  refreshRegistry: (client?: GoBindingClient, signal?: AbortSignal) => Promise<void>,
  resetRuntime: (signal?: AbortSignal) => Promise<void>,
  foregroundResume: (signal?: AbortSignal) => Promise<void>,
  resumeInterruptedTransfers: () => void,
  networkChanged: (connected: boolean, reason: NativeNetworkChangedEvent['reason']) => Promise<void>,
  initializeNetworkState: (connected: boolean) => Promise<void>,
): {
  phoneOnline: boolean
  connectionState: AppConnectionState
  retryConnectionRecovery: () => Promise<void>
  cancelForStop: () => void
} {
  const [phoneOnline, setPhoneOnline] = useState(true)
  const [connectionState, dispatchRecoveryStatus] = useReducer(reduceNativeRecoveryStatus, 'ready')
  const [successfulRecoveryRevision, setSuccessfulRecoveryRevision] = useState(0)
  const [recoveryFence] = useState(() => new NativeGenerationRecoveryFence())
  const [recoveryCoordinator] = useState(() => new NativeRecoveryCoordinator())
  const lastHeartbeatRef = useRef(globalThis.performance.now())
  const connectionStateRef = useRef<AppConnectionState>('ready')
  const recoveryNoticeTimerRef = useRef<ReturnType<typeof globalThis.setTimeout> | null>(null)
  const updateRecoveryStatus = useCallback((event: NativeRecoveryStatusEvent) => {
    connectionStateRef.current = reduceNativeRecoveryStatus(connectionStateRef.current, event)
    dispatchRecoveryStatus(event)
  }, [])
  const clearRecoveryNoticeTimer = useCallback(() => {
    if (recoveryNoticeTimerRef.current === null) return
    globalThis.clearTimeout(recoveryNoticeTimerRef.current)
    recoveryNoticeTimerRef.current = null
  }, [])
  const beginRecoveryStatus = useCallback((request: NativeRecoveryRequest) => {
    const previous = connectionStateRef.current
    const visibleImmediately = request.trigger === 'manual_retry'
    updateRecoveryStatus({ type: 'recovery.started', visibleImmediately })
    if (visibleImmediately || previous === 'recovering' || previous === 'failed') {
      clearRecoveryNoticeTimer()
      return
    }
    if (previous !== 'ready' || recoveryNoticeTimerRef.current !== null) return
    recoveryNoticeTimerRef.current = globalThis.setTimeout(() => {
      recoveryNoticeTimerRef.current = null
      updateRecoveryStatus({ type: 'recovery.noticeDelayElapsed' })
    }, NATIVE_RECOVERY_NOTICE_DELAY_MS)
  }, [clearRecoveryNoticeTimer, updateRecoveryStatus])
  const finishRecoveryStatus = useCallback((event: Extract<NativeRecoveryStatusEvent, { type: 'recovery.succeeded' | 'recovery.failed' }>) => {
    clearRecoveryNoticeTimer()
    updateRecoveryStatus(event)
  }, [clearRecoveryNoticeTimer, updateRecoveryStatus])
  const dismissRecoveryStatus = useCallback(() => {
    clearRecoveryNoticeTimer()
    updateRecoveryStatus({ type: 'recovery.dismissed' })
  }, [clearRecoveryNoticeTimer, updateRecoveryStatus])
  useEffect(() => clearRecoveryNoticeTimer, [clearRecoveryNoticeTimer])
  useEffect(() => {
    if (successfulRecoveryRevision === 0 || connectionState !== 'ready') return
    document.dispatchEvent(new CustomEvent('anytty:resume', {
      detail: { revision: successfulRecoveryRevision },
    }))
  }, [connectionState, successfulRecoveryRevision])

  const executeRecovery = useCallback(async ({ attempt, intent, trigger, signal }: NativeRecoveryWork) => {
    if (!recoveryFence.isCurrent(attempt) || signal.aborted) {
      nativeDiagnostic('foreground_recovery_skipped', { attempt, intent, trigger, reason: 'superseded' })
      return
    }
    const startedAt = globalThis.performance.now()
    let stage = 'native_runtime'
    nativeDiagnostic('foreground_recovery_start', { attempt, intent, trigger })
    markNativeBackground()
    try {
      await withNativeRecoveryTimeout(
        async () => await NativeConnection.handleForegroundResume(),
        'Native runtime recovery',
        signal,
      )
      nativeDiagnostic('foreground_recovery_stage', {
        attempt,
        stage,
        status: 'done',
        elapsed_ms: Math.round(globalThis.performance.now() - startedAt),
      })
      if (!recoveryFence.isCurrent(attempt) || signal.aborted) return
      stage = intent === 'repair' ? 'binding_replacement' : 'binding_health'
      if (intent === 'repair') {
        await withNativeRecoveryTimeout(
          (stepSignal) => replaceNativeGeneration(refreshRegistry, resetRuntime, stepSignal),
          'Native binding replacement',
          signal,
        )
      } else {
        try {
          // Resume only probes the existing bridge. Replacing an unchanged registry increments
          // its projection version and makes healthy machine runtimes look stale, which would
          // unnecessarily tear down their live sessions on every foreground transition.
          await withNativeRecoveryTimeout(
            (stepSignal) => goBindingClient.getEndpointRegistry(stepSignal),
            'Native binding health check',
            signal,
          )
        } catch {
          if (!recoveryFence.isCurrent(attempt) || signal.aborted) return
          await withNativeRecoveryTimeout(
            (stepSignal) => replaceNativeGeneration(refreshRegistry, resetRuntime, stepSignal),
            'Native binding replacement',
            signal,
          )
        }
      }
      nativeDiagnostic('foreground_recovery_stage', {
        attempt,
        stage,
        status: 'done',
        elapsed_ms: Math.round(globalThis.performance.now() - startedAt),
      })
      if (!recoveryFence.isCurrent(attempt) || signal.aborted) return
      lastHeartbeatRef.current = globalThis.performance.now()
      stage = 'endpoint_resume'
      await withNativeRecoveryTimeout(
        (stepSignal) => foregroundResume(stepSignal),
        'Native endpoint recovery',
        signal,
      )
      nativeDiagnostic('foreground_recovery_stage', {
        attempt,
        stage,
        status: 'done',
        elapsed_ms: Math.round(globalThis.performance.now() - startedAt),
      })
      if (!recoveryFence.isCurrent(attempt) || signal.aborted) return
      resumeInterruptedTransfers()
      finishRecoveryStatus({ type: 'recovery.succeeded' })
      finishNativeForeground()
      setSuccessfulRecoveryRevision((revision) => revision + 1)
      nativeDiagnostic('foreground_recovery_done', {
        attempt,
        duration_ms: Math.round(globalThis.performance.now() - startedAt),
      })
    } catch (failure) {
      if (!recoveryFence.isCurrent(attempt) || signal.aborted) return
      nativeDiagnostic('foreground_recovery_attempt_failed', {
        attempt,
        stage,
        duration_ms: Math.round(globalThis.performance.now() - startedAt),
        failure: diagnosticFailureCode(failure),
      })
      reportNativeGenerationFailure(failure)
      throw failure
    }
  }, [finishRecoveryStatus, foregroundResume, recoveryFence, refreshRegistry, resetRuntime, resumeInterruptedTransfers])
  const runRecovery = useCallback((request: NativeRecoveryRequest) => {
    beginRecoveryStatus(request)
    return recoveryCoordinator.request(request, {
      beginAttempt: () => recoveryFence.beginAttempt(),
      isCurrent: (attempt) => recoveryFence.isCurrent(attempt),
      execute: executeRecovery,
      retryDelay: (failure, retryAttempt) => nativeRecoveryRetryDelay(failure, retryAttempt),
      onRetryScheduled: (failure, delay, retryAttempt, work) => {
        nativeDiagnostic('foreground_recovery_retry_scheduled', {
          attempt: work.attempt,
          retry_attempt: retryAttempt,
          delay_ms: delay,
          failure: diagnosticFailureCode(failure),
        })
      },
      onTerminalFailure: (failure, work) => {
        if (!recoveryFence.isCurrent(work.attempt) || work.signal.aborted) return
        finishRecoveryStatus({ type: 'recovery.failed' })
        finishNativeForeground(failure)
      },
    })
  }, [beginRecoveryStatus, executeRecovery, finishRecoveryStatus, recoveryCoordinator, recoveryFence])

  const cancelForStop = useCallback(() => {
    recoveryFence.invalidate()
    recoveryCoordinator.cancel(Object.assign(new Error('Native recovery was stopped by the user'), {
      code: 'user_stopped',
      retryable: false,
    }))
    dismissRecoveryStatus()
    markNativeBackground()
    finishNativeForeground()
  }, [dismissRecoveryStatus, recoveryCoordinator, recoveryFence])

  const retryConnectionRecovery = useCallback(async () => {
    await runRecovery({
      intent: 'repair',
      trigger: 'manual_retry',
    }).catch(() => undefined)
  }, [runRecovery])

  useEffect(() => {
    const promise = CapApp.addListener('appStateChange', (state) => {
      if (!state.isActive) {
        nativeDiagnostic('app_state', { state: 'background' })
        recoveryFence.invalidate()
        recoveryCoordinator.cancel()
        dismissRecoveryStatus()
        markNativeBackground()
        return
      }
      nativeDiagnostic('app_state', { state: 'foreground' })
      void runRecovery({
        intent: 'ensure_ready',
        trigger: 'app_resume',
      }).catch(() => undefined)
    })
    const handleBindingClosed = () => {
      void runRecovery({
        intent: 'repair',
        trigger: 'binding_closed',
      }).catch(() => undefined)
    }
    document.addEventListener('anytty:binding-closed', handleBindingClosed)
    return () => {
      void promise.then((sub) => sub.remove())
      document.removeEventListener('anytty:binding-closed', handleBindingClosed)
    }
  }, [dismissRecoveryStatus, recoveryCoordinator, recoveryFence, runRecovery])

  useEffect(() => {
    let latestEpoch = -1
    let cancelled = false
    let receivedNativeEvent = false
    const applyNetworkEvent = (event: NativeNetworkChangedEvent) => {
      if (!Number.isSafeInteger(event.epoch) || event.epoch <= latestEpoch) return
      latestEpoch = event.epoch
      receivedNativeEvent = true
      setPhoneOnline(event.connected)
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
          if (!cancelled && !receivedNativeEvent) {
            setPhoneOnline(status.connected)
            await initializeNetworkState(status.connected)
          }
        }
      } catch {
        if (!initialize || cancelled || receivedNativeEvent) return
        const status = await Network.getStatus()
        if (!cancelled && !receivedNativeEvent) {
          setPhoneOnline(status.connected)
          await initializeNetworkState(status.connected)
        }
      }
    }
    const listener = NativeConnection.addListener('networkChanged', applyNetworkEvent)
    void listener.then(() => synchronizeNetworkSnapshot(true)).catch(() => undefined)
    const recoverAfterFreeze = () => {
      if (document.visibilityState !== 'visible') return
      const now = globalThis.performance.now()
      lastHeartbeatRef.current = now
      void runRecovery({
        intent: 'ensure_ready',
        trigger: 'page_visible',
      }).catch(() => undefined)
      void synchronizeNetworkSnapshot(false)
    }
    const synchronizeVisiblePage = () => {
      if (document.visibilityState === 'hidden') {
        recoveryFence.invalidate()
        recoveryCoordinator.cancel()
        dismissRecoveryStatus()
        markNativeBackground()
        return
      }
      recoverAfterFreeze()
    }
    document.addEventListener('visibilitychange', synchronizeVisiblePage)
    const stallDetector = globalThis.setInterval(() => {
      const now = globalThis.performance.now()
      const heartbeatGap = now - lastHeartbeatRef.current
      lastHeartbeatRef.current = now
      if (heartbeatGap >= rendererStallReconcileMs && document.visibilityState === 'visible') {
        void runRecovery({
          intent: 'ensure_ready',
          trigger: 'renderer_stall',
        }).catch(() => undefined)
        void synchronizeNetworkSnapshot(false)
      }
    }, 1_000)
    return () => {
      cancelled = true
      globalThis.clearInterval(stallDetector)
      document.removeEventListener('visibilitychange', synchronizeVisiblePage)
      void listener.then((subscription) => subscription.remove())
    }
  }, [dismissRecoveryStatus, initializeNetworkState, networkChanged, recoveryCoordinator, recoveryFence, runRecovery])

  useEffect(() => () => recoveryCoordinator.cancel(), [recoveryCoordinator])

  return {
    phoneOnline,
    connectionState,
    retryConnectionRecovery,
    cancelForStop,
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
  const signal = init.signal
  if (signal?.aborted) throw abortError(signal)
  const responsePromise = CapacitorHttp.request({
    url,
    method,
    headers,
    ...(data !== undefined ? { data } : {}),
    responseType: 'text',
    connectTimeout: nativeHttpConnectTimeoutMs,
    readTimeout: nativeHttpReadTimeoutMs,
  })
  if (!signal) {
    const response = await responsePromise
    return new Response(responseText(response.data), {
      status: response.status,
      headers: response.headers,
    })
  }
  return await new Promise<Response>((resolve, reject) => {
    const abort = () => reject(abortError(signal))
    signal.addEventListener('abort', abort, { once: true })
    void responsePromise.then(
      (response) => {
        signal.removeEventListener('abort', abort)
        resolve(new Response(responseText(response.data), {
          status: response.status,
          headers: response.headers,
        }))
      },
      (error) => {
        signal.removeEventListener('abort', abort)
        reject(error)
      },
    )
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
  resetGeneration: (signal?: AbortSignal) => Promise<void>
  foregroundResume: (signal?: AbortSignal) => Promise<void>
  initializeNetworkState: (connected: boolean) => Promise<void>
  networkChanged: (connected: boolean, reason: NativeNetworkChangedEvent['reason']) => Promise<void>
  disconnectAll: (request: NativeDisconnectAllCleanupRequest) => Promise<void>
  registerWorkspaceResumeIntent: (machineId: string, intent: object) => void
  clearPendingWorkspaceResumeIntents: () => void
  resumeInterruptedTransfers: () => void
} {
  const transferStore = new NativeFileTransferStore()
  const sessionManagers = new Map<string, NativeSessionEntry>()
  const disconnectFence = new NativeDeferredDisconnectFence()
  const endpointStops = new NativeEndpointStopRegistry(
    (intent, stopEpoch) => nativeSessionDemand.resumeIntentCoversStopEpoch(intent, stopEpoch),
  )
  let networkConnected = true
  const markFreshConnectionIntent = (machineId: string) => {
    disconnectFence.markFreshIntent(machineId)
    transferStore.markFreshConnectionIntent(machineId)
  }
  const workspaceResumeIntents = new NativeWorkspaceResumeIntentRegistry(markFreshConnectionIntent)
  const transferIntents = new NativeTransferIntentCoordinator(
    (machineId) => {
      const intent = nativeSessionDemand.createResumeIntent()
      workspaceResumeIntents.register(machineId, intent, sessionManagers.get(machineId)?.manager)
      return intent
    },
    async (machineId, intent) => {
      await nativeSessionDemand.confirmResumeIntent(intent)
      sessionManagers.get(machineId)?.manager.beginUserConnectionIntent(intent)
    },
    (machineId, intent) => workspaceResumeIntents.consume(machineId, intent),
  )
  transferStore.setSessionResolver(async (machineId, signal) => {
    return await sessionManagers.get(machineId)?.manager.get({ signal }) ?? null
  })

  return {
    registerWorkspaceResumeIntent(machineId, intent) {
      workspaceResumeIntents.register(machineId, intent, sessionManagers.get(machineId)?.manager)
    },
    clearPendingWorkspaceResumeIntents() {
      workspaceResumeIntents.clear()
    },
    fileTransfer: createFileTransferContext(undefined, transferStore, transferIntents),
    discardLocalState() {
      return transferStore.discardForLocalReset()
    },
    resumeInterruptedTransfers() {
      // The device list creates managers for status projection. Only a visible
      // workspace or a live task may turn foreground recovery into a connection.
      for (const [machineId, entry] of sessionManagers) {
        if (entry.manager.hasConnectionDemand()) void transferStore.resumeInterruptedTransfers(machineId)
      }
    },
    async resetGeneration(signal) {
      throwIfAborted(signal)
      await transferStore.suspendForRuntimeReset()
      throwIfAborted(signal)
      await drainNativeGenerationReset([...sessionManagers.values()].map((entry) => ({
        reset: () => entry.manager.resetBindingGeneration(),
        release: entry.connector.release
          ? () => entry.connector.release!(entry.manager.machineID())
          : undefined,
      })), signal)
      throwIfAborted(signal)
    },
    async foregroundResume(signal) {
      const targets = [...sessionManagers]
        .filter(([, entry]) => entry.manager.hasConnectionDemand())
        .map(([endpointId, entry]) => ({
          endpointId,
          resume: () => entry.manager.foregroundResume(signal),
        }))
      const work = startNativeForegroundWork(
        () => nativeSessionDemand.reconcileRenderer(),
        targets,
      )
      void work.demand.catch((failure) => {
        nativeDiagnostic('demand_reconcile_failed', { failure: diagnosticFailureCode(failure) })
      })
      const batch = work.endpoints
      nativeDiagnostic('endpoint_resume_started', { total: batch.total })
      const result = await batch.settled
      if (!signal?.aborted) {
        nativeDiagnostic('endpoint_resume_settled', {
          total: result.total,
          resumed: result.resumed,
          failed: result.failures.length,
        })
        for (const item of result.failures) {
          nativeDiagnostic('endpoint_resume_failed', {
            endpoint: diagnosticToken(item.endpointId),
            failure: diagnosticFailureCode(item.failure),
          })
        }
      }
    },
    async initializeNetworkState(connected) {
      networkConnected = connected
      await Promise.all([...sessionManagers.values()].map((entry) => entry.manager.initializeNetworkState(connected)))
    },
    async networkChanged(connected, reason) {
      networkConnected = connected
      await Promise.all([...sessionManagers.values()].map((entry) => entry.manager.networkChanged(connected, reason)))
    },
    async disconnectAll(request: NativeDisconnectAllCleanupRequest) {
      const stoppedEntries = [...sessionManagers].map(([machineId, entry]) => ({
        machineId,
        resumeIntent: entry.manager.latestUserResumeIntent() ?? workspaceResumeIntents.currentIntent(machineId),
        isCurrent: () => sessionManagers.get(machineId) === entry,
        adoptUserStop: () => entry.manager.adoptUserStop(),
      }))
      for (const pending of workspaceResumeIntents.entries()) {
        if (sessionManagers.has(pending.machineId)) continue
        stoppedEntries.push({
          ...pending,
          isCurrent: () => (
            !sessionManagers.has(pending.machineId) &&
            workspaceResumeIntents.isPending(pending.machineId, pending.resumeIntent)
          ),
          adoptUserStop: async () => undefined,
        })
      }
      await runNativeEndpointStopCleanup(
        request.stopEpoch,
        stoppedEntries,
        endpointStops,
        (protectedMachineIds) => transferStore.suspendForUserStop(protectedMachineIds),
        disconnectFence,
        new Set(request.protectedEndpointIds),
      )
    },
    createMachineRuntime(input) {
      return createNativeMachineRuntime(input.machine, input.storage, endpointRegistry, {
        sessionManagers,
        transferStore,
        networkConnected: () => networkConnected,
        markFreshConnectionIntent,
        endpointStops,
        workspaceResumeIntents,
        transferIntents,
      })
    },
  }
}

function useNativeDisconnectAll(
  disconnectAll: (request: NativeDisconnectAllCleanupRequest) => Promise<void>,
  cancelRecovery: () => void,
): void {
  const processorRef = useRef<NativeDisconnectAllRequestProcessor | null>(null)
  if (processorRef.current === null) {
    processorRef.current = new NativeDisconnectAllRequestProcessor(
      (event) => nativeSessionDemand.handleDisconnectAllRequested(event),
      async (stopEpoch) => {
        await NativeConnection.acknowledgeDisconnectAll({ stopEpoch })
        nativeSessionDemand.commitDisconnectAllCleanup(stopEpoch)
      },
    )
  }
  useEffect(() => {
    const processor = processorRef.current!
    processor.setCleanup(disconnectAll)
    const listener = NativeConnection.addListener('disconnectAllRequested', (event) => {
      cancelRecovery()
      processor.enqueue(event)
    })
    return () => {
      processor.setCleanup(null)
      void listener.then((subscription) => subscription.remove())
    }
  }, [cancelRecovery, disconnectAll])
}

function createNativeMachineRuntime(
  machine: RemoteMachine,
  storage: RemoteRuntimeStorage,
  endpointRegistry: NativeEndpointRegistryProjection,
  shared: {
    sessionManagers: Map<string, NativeSessionEntry>
    transferStore: NativeFileTransferStore
    networkConnected: () => boolean
    markFreshConnectionIntent: (machineId: string) => void
    endpointStops: NativeEndpointStopRegistry
    workspaceResumeIntents: NativeWorkspaceResumeIntentRegistry
    transferIntents: NativeTransferIntentCoordinator
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
    const normalizedPlatform = normalizeDaemonPlatform(machine.osInfo ?? '')
    const currentMachine = machineStore.getMachine(machine.id)
    if (normalizedPlatform && currentMachine && currentMachine.osInfo !== normalizedPlatform) {
      machineStore.saveMachine({ ...currentMachine, osInfo: normalizedPlatform, updatedAt: new Date().toISOString() })
      globalThis.dispatchEvent(new CustomEvent('anytty:machine-metadata-changed', { detail: { machineId: machine.id } }))
    }
    const connector = createNativeConnector(machine, endpointRegistry)
    const manager = new NativeSessionManager(machine.id, connector, {
      initiallyConnected: shared.networkConnected(),
      waitForForeground: (signal) => nativeForegroundBarrier.wait(signal),
      writeDiagnostic: writeNativeDiagnostic,
      onUserResumeAccepted: (intent) => shared.endpointStops.acceptUserResume(machine.id, intent),
    })
    entry = {
      endpointIdentity,
      connector,
      manager,
    }
    shared.sessionManagers.set(machine.id, entry)
    shared.endpointStops.adoptIfStopped(machine.id, () => manager.adoptUserStop())
    shared.workspaceResumeIntents.attachManager(machine.id, manager)
  }
  const sessionManager = entry.manager
  const connector = entry.connector
  const transferStore = shared.transferStore

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
            size_locked: terminal.tags['anytty.size_lock'] === 'lock',
            size_lock_mode: terminal.tags['anytty.size_lock'],
            tags: { ...terminal.tags },
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
        shared.markFreshConnectionIntent(machine.id)
        return sessionManager.resetClientOnly(options).then(() => {
          if (sessionManager.hasConnectionDemand()) void transferStore.resumeInterruptedTransfers(machine.id)
        })
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
    retainConnectionDemand: (resumeIntent = null) => {
      if (resumeIntent) {
        if (!shared.workspaceResumeIntents.isRegistered(machine.id, resumeIntent)) {
          shared.workspaceResumeIntents.register(machine.id, resumeIntent, sessionManager)
        }
        shared.workspaceResumeIntents.consume(machine.id, resumeIntent)
      }
      const releaseDemand = sessionManager.retainConnectionDemand(resumeIntent)
      // Entering this workspace is the user-intent boundary for persisted transfers.
      queueMicrotask(() => {
        if (sessionManager.hasConnectionDemand()) void transferStore.resumeInterruptedTransfers(machine.id)
      })
      return releaseDemand
    },
    probeConnection: () => sessionManager.probe(),
    fileTransfer: createFileTransferContext(machine.id, transferStore, shared.transferIntents),
    async disconnect() {
      shared.workspaceResumeIntents.clear(machine.id)
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

export function createFileTransferContext(
  machineId: string | undefined,
  store: NativeFileTransferStore,
  transferIntents: NativeTransferIntentCoordinator,
): FileTransferContext {
  const beginTransferIntent = (requestedMachineId: string | undefined, transferId?: string) => {
    const transfer = transferId
      ? store.getSnapshot().transfers.find((candidate) => candidate.id === transferId)
      : undefined
    if (transferId && (!transfer || !isResumableTransferStatus(transfer.status))) return null
    const endpointId = requestedMachineId ?? machineId ?? transfer?.machineId
    return endpointId ? { endpointId, intent: transferIntents.begin(endpointId) } : null
  }
  return {
    subscribe: (listener) => store.subscribe(listener),
    getSnapshot: () => store.getSnapshot(machineId),
    isNative: true,
    getDownloadResumeOffset(mid, filePath, fileSize) {
      return store.getDownloadResumeOffset(mid, filePath, fileSize)
    },
    beginTransferIntent(mid) {
      return transferIntents.begin(mid)
    },
    discardTransferIntent(mid, intent) {
      transferIntents.finish(mid, intent)
    },
    async startDownload(mid, fileName, fileSize, filePath, offset, intent) {
      const currentIntent = intent ?? transferIntents.begin(mid)
      await transferIntents.run(mid, currentIntent, () => {
        store.startDownload(mid, fileName, fileSize, filePath, offset)
      })
    },
    async startUpload(mid, files, targetDir, intent) {
      const currentIntent = intent ?? transferIntents.begin(mid)
      await transferIntents.run(mid, currentIntent, () => {
        for (const f of files) store.startUpload(mid, f.uri, f.name, f.size, targetDir)
      })
    },
    async pickAndUpload(mid, targetDir) {
      const intent = transferIntents.begin(mid)
      try {
        const result = await runAcrossNativePicker(
          nativeForegroundBarrier,
          () => NativeFilePicker.pickFiles({ multiple: true }),
          () => transferIntents.confirm(mid, intent),
        )
        for (const f of result.files) {
          store.startUpload(mid, f.uri, f.name, f.size, targetDir)
        }
      } catch {
        // Picker cancellation and a Stop-fenced continuation both leave no task.
      } finally {
        transferIntents.finish(mid, intent)
      }
    },
    pauseTransfer(id) { store.pauseTransfer(id) },
    resumeTransfer(id) {
      const action = beginTransferIntent(undefined, id)
      if (action === null) return
      return store.resumeTransfer(id).finally(() => {
        transferIntents.finish(action.endpointId, action.intent)
      })
    },
    resumeAllTransfers(requestedMachineId) {
      const endpointIds = new Set(store.getSnapshot(requestedMachineId ?? machineId).transfers.flatMap((transfer) => (
        transfer.machineId && isResumableTransferStatus(transfer.status) ? [transfer.machineId] : []
      )))
      const actions = [...endpointIds].map((endpointId) => ({ endpointId, intent: transferIntents.begin(endpointId) }))
      return store.resumeAllTransfers(requestedMachineId ?? machineId).finally(() => {
        for (const action of actions) transferIntents.finish(action.endpointId, action.intent)
      })
    },
    openDownloadedFile(id) { return store.openDownloadedFile(id) },
    cancelTransfer(id) { store.cancelTransfer(id) },
    dismissTransfer(id) { store.dismissTransfer(id) },
  }
}

function isResumableTransferStatus(status: string): boolean {
  return status === 'paused' || status === 'failed' || status === 'missing'
}

function createNativeConnector(
  machine: RemoteMachine,
  endpointRegistry: NativeEndpointRegistryProjection,
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

  const demandOwner = Symbol(machine.id)
  const connector = new GoBindingConnector(() => goBindingClient, {
  endpointId: machine.id,
  })
  return {
    connect: (target, options) => connector.connect(target, options),
    getConnectionPolicy: (signal) => connector.getConnectionPolicy(signal),
    applyConnectionPolicy: (policy, signal) => connector.applyConnectionPolicy(policy, signal),
    disconnect: (machineId) => {
      if (machineId !== machine.id) return Promise.reject(new Error('endpoint identity mismatch'))
      return goBindingClient.disconnectEndpoint(machine.id)
    },
    setActive: (machineId, active) => nativeSessionDemand.setActive(machineId, active, demandOwner),
    createResumeIntent: () => nativeSessionDemand.createResumeIntent(),
    resumeDemand: (intent) => nativeSessionDemand.resumeForUserIntent(intent),
    requestRecovery: () => NativeConnection.requestEndpointRecovery({ endpointId: machine.id }),
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
