import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { App as CapApp } from '@capacitor/app'
import { Capacitor, CapacitorHttp } from '@capacitor/core'
import { Keyboard } from '@capacitor/keyboard'
import { Network } from '@capacitor/network'
import { create } from '@bufbuild/protobuf'
import {
  RemoteControlApp,
  createMachineStore,
  dispatchNativeKeyboardEvent,
  normalizeTerminalInventory,
  openProtoEventSubscription,
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
import { NativeConnection } from './plugins/nativeConnection'
import { NativeFileTransferStore } from './NativeFileTransferStore'
import { GoBindingClient, GoBindingConnector } from './GoBindingClient'
import { settleBindingGeneration } from './BindingGeneration'
import { NativeSessionManager, type NativeSessionConnector } from './NativeSessionManager'
import NativeFilePicker from './plugins/nativeFilePicker'
import { useNativeStatusBarSync } from './nativeStatusBar'
import { NativeForegroundBarrier, runAcrossNativePicker } from './NativeForegroundBarrier'
import { NativeGenerationRecoveryFence } from './NativeGenerationRecoveryFence'
import { RegistryStartupScreen, UnsupportedWebPreview } from './RegistryStartupScreen'
import { useAndroidBackButton } from './androidBack'
import { scanPairingCode } from './nativeQrScanner'

const nativeHttpConnectTimeoutMs = 8_000
const nativeHttpReadTimeoutMs = 15_000
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
  if (!Capacitor.isNativePlatform()) return <UnsupportedWebPreview />
  return <NativeAnyTTYApp />
}

function NativeAnyTTYApp() {
  useAndroidBackButton()
  useNativeKeyboardEvents()
  useNativeStatusBarSync()

  const networkRuntime = useMemo(() => createNativeNetworkRuntime(), [])
  const endpointRegistry = useMemo(() => new NativeEndpointRegistryProjection(), [])
  const nativeAppRuntime = useMemo(() => createNativeAppRuntime(endpointRegistry), [endpointRegistry])
  useNativeNetworkSessionRecovery(nativeAppRuntime.networkChanged)
  const [registryReady, setRegistryReady] = useState(false)
  const [registryError, setRegistryError] = useState<string | null>(null)
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
      <RegistryStartupScreen
        error={registryError}
        onResetLocalPairings={resetLocalPairings}
        onRetry={retryRegistry}
      />
    )
  }

  return (
    <section className="anytty-app-page flex h-[100dvh] w-screen flex-col overflow-hidden antialiased">
      <RemoteControlApp
        singlePaneWorkspace
        externalPairingAdapter={externalPairingAdapter}
        globalFileTransfer={globalFileTransfer}
        machineRuntimeFactory={machineRuntimeFactory}
        networkRuntime={networkRuntime}
        nativeNetworkStatusPlugin={Network}
        connectionReady={nativeConnectionRecovery.connectionReady}
        connectionRecoveryFailed={nativeConnectionRecovery.connectionRecoveryFailed}
        onRetryConnectionRecovery={nativeConnectionRecovery.retryConnectionRecovery}
        onRefreshMachines={() => refreshRegistry()}
        scanPairingCode={scanPairingCode}
      />
    </section>
  )
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
    machine: { id: endpoint.endpointId, name: endpoint.label || endpoint.endpointId, accessClass: 'local' },
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
      machine: { id: endpoint.endpointId, name: endpoint.label || endpoint.endpointId, accessClass: 'local' },
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

  replace(registry: AnyTTYRemoteAuth.EndpointRegistryV1): void {
    this.registry = create(AnyTTYRemoteAuth.EndpointRegistryV1Schema, registry)
    this.versionValue += 1
  }

  snapshot(): AnyTTYRemoteAuth.EndpointRegistryV1 { return create(AnyTTYRemoteAuth.EndpointRegistryV1Schema, this.registry) }
  has(endpointId: string): boolean { return this.registry.endpoints.some((endpoint) => endpoint.endpointId === endpointId) }
  isAuthorized(endpointId: string): boolean {
  const endpoint = this.registry.endpoints.find((candidate) => candidate.endpointId === endpointId)
  return endpoint?.routes.some((route) => route.credentialRef.trim() !== '') ?? false
  }
  version(): number { return this.versionValue }
  authorizationExpiry(endpointId: string): string | undefined { return this.expiries.get(endpointId) }
  setAuthorizationExpiry(endpointId: string, expiresAt: string | undefined): void {
    if (expiresAt) this.expiries.set(endpointId, expiresAt)
    else this.expiries.delete(endpointId)
  }
}

function syncRegistryMachineProjection(storage: RemoteRuntimeStorage, registry: AnyTTYRemoteAuth.EndpointRegistryV1): void {
  const store = createMachineStore({ storage })
  const now = new Date().toISOString()
  for (const endpoint of registry.endpoints) {
    const existing = store.getMachine(endpoint.endpointId)
    store.saveMachine({
      machineId: endpoint.endpointId,
      name: endpoint.label || existing?.name || endpoint.endpointId,
      state: existing?.state ?? 'offline',
      terminalCount: existing?.terminalCount ?? 0,
      source: existing?.source ?? 'manual',
      accessClass: existing?.accessClass ?? 'local',
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
  const runRecovery = useCallback(async (replaceBinding: boolean, reloadRegistry: boolean, claimedAttempt?: number) => {
    const attempt = claimedAttempt ?? recoveryFence.beginAttempt()
    if (!recoveryFence.isCurrent(attempt)) return
    if (replaceBinding) setStatus('restoring')
    markNativeBackground()
    try {
      await NativeConnection.handleForegroundResume()
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
  networkChanged: () => Promise<void>
  resumeInterruptedTransfers: () => void
} {
  const transferStore = new NativeFileTransferStore()
  const sessionManagers = new Map<string, NativeSessionEntry>()
  const autoResumeMachines = new Set<string>()
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
      await Promise.all([...sessionManagers.values()].map((entry) => entry.manager.foregroundResume()))
    },
    async networkChanged() {
      await Promise.all([...sessionManagers.values()].map((entry) => entry.manager.networkChanged()))
    },
    createMachineRuntime(input) {
      return createNativeMachineRuntime(input.machine, input.storage, endpointRegistry, {
        sessionManagers,
        transferStore,
        autoResumeMachines,
      })
    },
  }
}

function useNativeNetworkSessionRecovery(networkChanged: () => Promise<void>): void {
  useEffect(() => {
    let latestEpoch = 0
    const listener = NativeConnection.addListener('networkChanged', (event) => {
      if (!Number.isSafeInteger(event.epoch) || event.epoch <= latestEpoch) return
      latestEpoch = event.epoch
      void networkChanged().catch(() => undefined)
    })
    return () => { void listener.then((subscription) => subscription.remove()) }
  }, [networkChanged])
}

function createNativeMachineRuntime(
  machine: RemoteMachine,
  storage: RemoteRuntimeStorage,
  endpointRegistry: NativeEndpointRegistryProjection,
  shared: {
    sessionManagers: Map<string, NativeSessionEntry>
    transferStore: NativeFileTransferStore
    autoResumeMachines: Set<string>
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
    const connector = createNativeConnector(machine, endpointRegistry)
    entry = {
      endpointIdentity,
      connector,
      manager: new NativeSessionManager(machine.id, connector),
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
      const statusMachine: Machine = {
        machineId: machine.id,
        name: machine.name,
        state: machine.online ? 'online' : 'offline',
        terminalCount: storedMachine?.terminalCount,
        ...(machine.lastSeen || storedMachine?.lastSeenAt ? { lastSeenAt: machine.lastSeen ?? storedMachine?.lastSeenAt } : {}),
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
    fileTransfer: createFileTransferContext(machine.id, transferStore),
    async disconnect() {
      await sessionManager.reset()
      await connector.release?.(machine.id)
    },
    dispose: () => {
      if (shared.sessionManagers.get(machine.id)?.manager === sessionManager) {
        shared.sessionManagers.delete(machine.id)
      }
      return sessionManager.reset().finally(() => connector.release?.(machine.id))
    },
  }
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
    cancelTransfer(id) { store.cancelTransfer(id) },
    dismissTransfer(id) { store.dismissTransfer(id) },
  }
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

  const connector = new GoBindingConnector(() => goBindingClient, {
  endpointId: machine.id,
  })
  return {
    connect: (target, options) => connector.connect(target, options),
    async verify(session, signal) {
      const response = await session.execute(create(AnyTTYApiApplication.CommandEnvelopeSchema, {
        command: { case: 'terminalList', value: create(AnyTTYApiTerminal.TerminalListCommandSchema) },
      }), { signal })
      if (response.result.case !== 'terminalList') throw new Error('foreground session verification returned no terminal list')
    },
    getConnectionPolicy: (signal) => connector.getConnectionPolicy(signal),
    applyConnectionPolicy: (policy, signal) => connector.applyConnectionPolicy(policy, signal),
    setActive: (machineId, active) => NativeConnection.setSessionActive({ machineId, active }),
  }
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
