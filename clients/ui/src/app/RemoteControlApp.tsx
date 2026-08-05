import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore, type CSSProperties, type ChangeEvent, type FormEvent, type ReactNode, type RefObject, type TouchEvent as ReactTouchEvent } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { ArrowLeft, Camera, Check, ChevronRight, ClipboardPaste, Cloud, Copy, Download, Info, Keyboard, LaptopMinimal, Link2Off, Monitor, MoreHorizontal, QrCode, RefreshCw, Server, Settings, ShieldCheck, Trash2, Unplug, Wifi, WifiOff, X } from 'lucide-react'
import { MachineWorkspace, type MachineWorkspaceInventoryApi, type MachineWorkspaceConnector } from './MachineWorkspace'
import { createMachineStore, type StoredMachineRecord } from '../state/machineStore'
import type { MachineConnectionSnapshot } from '../connection/machineConnectionSnapshot'
import { FileTransferPanel } from '../files/FileTransferPanel'
import { hapticError, hapticImpact, hapticSelection, hapticSuccess } from '../platform/haptics'
import { NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import { useNativeBackHandler } from '../platform/useNativeBackHandler'
import type { FileTransferContext, TransferInfo } from '../files/fileApi'
import type { MachineConnectionStateEvents, RemoteNetworkRuntime, RemoteRuntimeFetch, RemoteRuntimeStorage, RtcConnectOptions, TerminalInventoryEvents } from '../core/transport'
import { normalizeHubBaseUrlCandidate } from '../api/hubUrl'
import type { RemoteMachine } from '../core/remoteMachine'
import {
  TERMINAL_FONT_OPTIONS,
  TERMINAL_THEME_OPTIONS,
  readTerminalSettings,
  resolveTerminalThemeUi,
  terminalThemeCssVariables,
  writeTerminalSettings,
  type TerminalKeyboardMode,
  type TerminalSettings,
  type TerminalThemeOption,
} from '../terminal/terminalSettings'
import type { TerminalRenderer } from '../terminal/Terminal'
import type { MachineAccessClass } from '../state/appMachine'
import { anyttyIntlLocale, anyttyLanguages, normalizeAnyTTYLanguage } from '../i18n'
import { connectionErrorDisplayMessage } from '../connection/connectionErrorPresentation'
import { RemoteNetworkStateManager, type NativeNetworkStatusPlugin } from '../connection/remoteNetworkState'
import { ModalSurface } from '../ui/ModalSurface'
import { maximumPastedPairingInputLength, parsePastedPairingInput } from './pairingInput'

const appName = 'AnyTTY Remote App'

function noopSubscribe(_listener: () => void): () => void { return () => {} }

function appErrorCode(error: unknown): string {
  if (!error || typeof error !== 'object' || !('code' in error)) return ''
  const code = (error as { code?: unknown }).code
  return typeof code === 'string' ? code.trim().toLowerCase() : ''
}

function localizedAppError(error: unknown, t: TFunction): string {
  switch (appErrorCode(error)) {
    case 'login_required':
    case 'unauthenticated':
      return t('errors.pairAgain')
    case 'capability_invalid':
    case 'capability_expired':
    case 'authorization_revoked':
      return t('errors.pairAgain')
    case 'unavailable':
    case 'route_unavailable':
      return t('errors.connectionFailed')
    case 'temporary':
      return t('errors.temporary')
    case 'entitlement_denied':
      return t('errors.relayEntitlementDenied')
    case 'relay_not_in_plan':
    case 'subscription_inactive':
    case 'relay_region_unavailable':
      return t('errors.relayEntitlementDenied')
    case 'relay_quota_exhausted':
      return t('errors.relayQuotaExhausted')
    case 'relay_concurrency_exhausted':
      return t('errors.relayConcurrencyExhausted')
    case 'resource_exhausted':
      return connectionErrorDisplayMessage(error, t)
    default:
      return t('errors.generic')
  }
}

function cameraScanErrorPresentation(error: unknown, t: TFunction): { message: string; reloadRequired: boolean } {
  switch (appErrorCode(error)) {
    case 'scanner_load_failed':
      return { message: t('pairing.scannerLoadFailed'), reloadRequired: true }
    case 'camera_permission_denied':
      return { message: t('pairing.cameraPermissionDenied'), reloadRequired: false }
    case 'camera_not_found':
      return { message: t('pairing.cameraNotFound'), reloadRequired: false }
    case 'camera_start_failed':
      return { message: t('pairing.cameraStartFailed'), reloadRequired: false }
    default:
      return { message: localizedAppError(error, t), reloadRequired: false }
  }
}

type AppView = 'home' | 'settings' | 'machine'
type PairIntent = 'add-local' | 'authorize-machine'
type ScanFlowState = 'idle' | 'scanning' | 'pairing'
type MachineAuthorizationState = 'ready' | 'expired' | 'unauthorized'
type DisplayMachine = RemoteMachine & {
  reachability?: MachineReachabilityView | undefined
  accessClass: MachineAccessClass
  terminalCount?: number | undefined
}
interface MachineReachabilityView {
  hubOnline: boolean
  localChecked: boolean
  localOnline: boolean
  localOnlineUrls: string[]
}
interface LocalHubReachabilityTarget {
  machineId: string
  urls: string[]
}
interface LocalHubReachabilitySnapshot {
  machineId: string
  urls: string[]
  onlineUrls: string[]
  checkedAt: number
}
const emptyMachineConnectionSnapshot: MachineConnectionSnapshot = {
  machineId: '',
  phase: 'idle',
  statusText: 'Ready',
  connectionInfo: null,
  forceRelay: false,
  relayInUse: false,
  reconnectAttempt: 0,
  error: null,
}
const getEmptyMachineConnectionSnapshot = () => emptyMachineConnectionSnapshot
const localHubReachabilityProbeTimeoutMs = 2_500
export interface ScanPairingCodeOptions {
  signal?: AbortSignal | undefined
}

/** ExternalPairingImportResult 是平台 secure-store 导入成功后可进入共享 UI 的非秘密机器投影。 */
export interface ExternalPairingImportResult {
  machine: { id: string; name: string; hostname?: string | undefined; accessClass?: MachineAccessClass | undefined }
  expiresAt?: string | undefined
  authorizationRequired?: boolean | undefined
  sshCredentials?: { routeId: string; authorizedKey: string; fingerprint: string }[] | undefined
}

/** EndpointSharePreviewView 是 Go Client Engine 验证并计算后的 config-only 导入差异。 */
export interface EndpointSharePreviewView {
  importToken: string
  endpointId: string
  label: string
  deviceId: string
  deviceFingerprint: string
  routes: { id: string; kind: string; action: string }[]
  connectModeChanged: boolean
  selectionPolicyChanged: boolean
  credentialKinds: string[]
}

/**
 * ExternalPairingAdapter 允许 Android/iOS 把 bearer capability 留在 native secure-store。
 * 共享 UI 只查询 grant_ref 是否存在，不接收、持久化或转发原始 grant。
 */
export interface ExternalPairingAdapter {
  /** import 必须在写平台凭据前校验 expectedMachineId；未指定时表示全局新增设备。 */
  import(rawValue: string, expectedMachineId?: string): Promise<ExternalPairingImportResult | null>
  inspectShare?(rawValue: string): Promise<EndpointSharePreviewView>
  commitShare?(importToken: string): Promise<ExternalPairingImportResult>
  isAuthorized(machineId: string): boolean
  authorizationExpiresAt?(machineId: string): string | undefined
  forget(machineId: string): void | Promise<void>
}

export type MachineRuntimeFactory = (input: {
  machine: RemoteMachine
  storage: RemoteRuntimeStorage
}) => MachineRuntime

export interface MachineRuntime {
  api: MachineWorkspaceInventoryApi
  connector: MachineWorkspaceConnector
  inventoryEvents?: TerminalInventoryEvents | undefined
  connectionStateEvents?: MachineConnectionStateEvents | undefined
  listConnectionState?: {
    getSnapshot(): MachineConnectionSnapshot
    subscribe(listener: () => void): () => void
  } | undefined
  fileTransfer?: FileTransferContext | undefined
  disconnect?(): void | Promise<void>
  dispose?(): void | Promise<void>
}

export interface RemoteControlAppProps {
  storage?: RemoteRuntimeStorage | undefined
  networkRuntime?: RemoteNetworkRuntime | undefined
  machineRuntimeFactory?: MachineRuntimeFactory | undefined
  globalFileTransfer?: FileTransferContext | undefined
  scanPairingCode?: ((options?: ScanPairingCodeOptions) => Promise<string | null>) | undefined
  externalPairingAdapter?: ExternalPairingAdapter | undefined
  exportDebugLogs?: (() => Promise<void>) | undefined
  onRefreshMachines?: (() => Promise<void>) | undefined
  nativeNetworkStatusPlugin?: NativeNetworkStatusPlugin | undefined
  connectionReady?: boolean | undefined
  connectionRecoveryFailed?: boolean | undefined
  onRetryConnectionRecovery?: (() => void | Promise<void>) | undefined
  singlePaneWorkspace?: boolean | undefined
}

export function RemoteControlApp({
  storage: storageProp,
  networkRuntime: networkRuntimeProp,
  machineRuntimeFactory = createUnavailableMachineRuntime,
  globalFileTransfer,
  scanPairingCode,
  externalPairingAdapter,
  exportDebugLogs,
  onRefreshMachines,
  nativeNetworkStatusPlugin,
  connectionReady = true,
  connectionRecoveryFailed = false,
  onRetryConnectionRecovery,
  singlePaneWorkspace = false,
}: RemoteControlAppProps) {
  const { t } = useTranslation()
  const networkRuntime = networkRuntimeProp ?? unavailableNetworkRuntime
  const storage = storageProp ?? networkRuntime.storage
  const [view, setView] = useState<AppView>('home')
  const [terminalSettings, setTerminalSettings] = useState<TerminalSettings>(() => readTerminalSettings(storage))
  const [localMachines, setLocalMachines] = useState<StoredMachineRecord[]>(() => {
    return storage ? createMachineStore({ storage }).listMachines() : []
  })
  const [localHubReachability, setLocalHubReachability] = useState<Map<string, LocalHubReachabilitySnapshot>>(() => new Map())
  const [selectedMachineId, setSelectedMachineId] = useState<string | null>(null)
  const [scanOpen, setScanOpen] = useState(false)
  const [pairIntent, setPairIntent] = useState<PairIntent>('add-local')
  const [transferCenterOpen, setTransferCenterOpen] = useState(false)
  const [authorizedMachineIds, setAuthorizedMachineIds] = useState(() => readAuthorizedMachineIds(storage, undefined, externalPairingAdapter))
  const [authorizationExpiries, setAuthorizationExpiries] = useState(() => readAuthorizationExpiries(storage, externalPairingAdapter))
  const [pairVersion, setPairVersion] = useState(0)
  const [error, setError] = useState<string | null>(null)
  const [pairing, setPairing] = useState(false)
  const [sharePreview, setSharePreview] = useState<EndpointSharePreviewView | null>(null)
  const [sshCredentialNotice, setSSHCredentialNotice] = useState<NonNullable<ExternalPairingImportResult['sshCredentials']> | null>(null)
  const [cameraScanning, setCameraScanning] = useState(false)
  const [scannerReloadRequired, setScannerReloadRequired] = useState(false)
  const [scanFlowState, setScanFlowState] = useState<ScanFlowState>('idle')
  const [reachabilityRefreshToken, setReachabilityRefreshToken] = useState(0)
  const remoteNetworkStateManager = useMemo(
    () => new RemoteNetworkStateManager(nativeNetworkStatusPlugin),
    [nativeNetworkStatusPlugin],
  )
  const remoteNetworkState = useSyncExternalStore(
    remoteNetworkStateManager.subscribeSnapshot.bind(remoteNetworkStateManager),
    () => remoteNetworkStateManager.state,
    () => remoteNetworkStateManager.state,
  )
  const effectiveConnectionReady = connectionReady && remoteNetworkState.networkReady
  const appThemeStyle = useMemo(() => terminalThemeCssVariables(terminalSettings.themeId) as CSSProperties, [terminalSettings.themeId])
  const cameraScanInFlightRef = useRef(false)
  const cameraScanAbortRef = useRef<AbortController | null>(null)
  const cameraScanButtonRef = useRef<HTMLButtonElement>(null)
  const restoreCameraFocusAfterScanRef = useRef(false)
  const runtimeCacheRef = useRef<{
    networkRuntime: RemoteNetworkRuntime
    runtimeFactory: MachineRuntimeFactory
    storage: RemoteRuntimeStorage
    runtimes: Map<string, MachineRuntime>
  } | null>(null)

  useEffect(() => {
    remoteNetworkStateManager.init()
    return () => remoteNetworkStateManager.destroy()
  }, [remoteNetworkStateManager])

  if (storage) {
    const cache = runtimeCacheRef.current
    const cacheMatches = cache &&
      cache.networkRuntime === networkRuntime &&
      cache.runtimeFactory === machineRuntimeFactory &&
      cache.storage === storage
    if (!cacheMatches) {
      if (cache) {
        for (const runtime of cache.runtimes.values()) void runtime.dispose?.()
      }
      runtimeCacheRef.current = {
        networkRuntime,
        runtimeFactory: machineRuntimeFactory,
        storage,
        runtimes: new Map(),
      }
    }
  }

  const dropMachineRuntime = useCallback((machineId: string) => {
    const runtime = runtimeCacheRef.current?.runtimes.get(machineId)
    if (!runtime) return
    runtimeCacheRef.current?.runtimes.delete(machineId)
    void runtime.dispose?.()
  }, [])

  const getMachineRuntime = useCallback((machine: DisplayMachine): MachineRuntime | null => {
    if (!storage || !runtimeCacheRef.current) return null
    const cache = runtimeCacheRef.current.runtimes
    const existing = cache.get(machine.id)
    if (existing) return existing
    const created = machineRuntimeFactory({
      machine,
      storage,
    })
    cache.set(machine.id, created)
    return created
  }, [machineRuntimeFactory, storage])

  const getExistingMachineRuntime = useCallback((machine: DisplayMachine): MachineRuntime | null => {
    return runtimeCacheRef.current?.runtimes.get(machine.id) ?? null
  }, [])

  useEffect(() => {
    return () => {
      cameraScanAbortRef.current?.abort()
      const cache = runtimeCacheRef.current
      if (!cache) return
      runtimeCacheRef.current = null
      for (const runtime of cache.runtimes.values()) void runtime.dispose?.()
    }
  }, [])

  useEffect(() => {
    if (storage) {
      setLocalMachines(createMachineStore({ storage }).listMachines())
    }
  }, [storage, pairVersion])

  const localHubReachabilityTargets = useMemo(() => {
    return buildLocalHubReachabilityTargets(localMachines)
  }, [localMachines])

  useEffect(() => {
    setLocalHubReachability((current) => pruneLocalHubReachability(current, localHubReachabilityTargets))
    if (localHubReachabilityTargets.length === 0) return
    const controller = new AbortController()
    for (const target of localHubReachabilityTargets) {
      void probeLocalHubReachability(networkRuntime.fetch, target, controller.signal).then((snapshot) => {
        if (controller.signal.aborted) return
        setLocalHubReachability((current) => {
          const previous = current.get(snapshot.machineId)
          if (sameReachabilitySnapshot(previous, snapshot)) return current
          const next = new Map(current)
          next.set(snapshot.machineId, snapshot)
          return next
        })
      })
    }
    return () => controller.abort()
  }, [localHubReachabilityTargets, networkRuntime.fetch, reachabilityRefreshToken])

  const displayMachines = useMemo(() => {
    const map = new Map<string, DisplayMachine>()
    for (const local of localMachines) {
      const reachability = localHubReachability.get(local.machineId)
      const localOnline = localMachineOnline(local, reachability)
      map.set(local.machineId, {
        id: local.machineId,
        name: userFacingMachineName(local.machineId, local.name, local.hostname, t),
        hostname: local.hostname,
        online: localOnline,
        source: 'local',
        hubUrls: hubUrlsFromStoredMachine(local),
        localHubUrls: localHubUrlsFromStoredMachine(local),
        localFallbackHubUrls: localFallbackHubUrlsFromStoredMachine(local),
        reachability: machineReachabilityView({
          hubOnline: false,
          localOnline,
          snapshot: reachability,
        }),
        accessClass: local.accessClass ?? 'local',
        terminalCount: local.terminalCount,
      })
    }
    return Array.from(map.values())
  }, [localHubReachability, localMachines, t])

  const selectedMachine = displayMachines.find((machine) => machine.id === selectedMachineId) ?? null
  const emptyTransferSnapshot = useMemo(() => ({ transfers: [], hasActiveTransfers: false }), [])
  const globalTransferState = useSyncExternalStore(
    globalFileTransfer?.subscribe ?? noopSubscribe,
    globalFileTransfer?.getSnapshot ?? (() => emptyTransferSnapshot),
  )

  const refreshMachines = useCallback(() => {
    const localMachineList = storage ? createMachineStore({ storage }).listMachines() : []
    setLocalMachines(localMachineList)
    setAuthorizedMachineIds(readAuthorizedMachineIds(storage, undefined, externalPairingAdapter))
    setAuthorizationExpiries(readAuthorizationExpiries(storage, externalPairingAdapter))
    setSelectedMachineId((current) => current && localMachineList.some((machine) => machine.machineId === current) ? current : null)
  }, [externalPairingAdapter, storage])

  const performMachineRefresh = useCallback(async () => {
    if (!remoteNetworkStateManager.state.phoneOnline) throw Object.assign(new Error('phone offline'), { code: 'offline' })
    if (!effectiveConnectionReady) throw Object.assign(new Error('connection generation is not ready'), { code: 'cancelled' })
    await onRefreshMachines?.()
    refreshMachines()
    setReachabilityRefreshToken((current) => current + 1)
  }, [effectiveConnectionReady, onRefreshMachines, refreshMachines, remoteNetworkStateManager])

  const prepareTransferMachineRuntime = useCallback((transferId?: string) => {
    if (!globalFileTransfer || !transferId) return
    const transfer = globalFileTransfer.getSnapshot().transfers.find((item) => item.id === transferId)
    const machineId = transfer?.machineId
    if (!machineId) return
    const machine = displayMachines.find((item) => item.id === machineId)
    if (!machine) return
    getMachineRuntime(machine)
  }, [displayMachines, getMachineRuntime, globalFileTransfer])

  const resumeGlobalTransfer = useCallback(async (transferId: string) => {
    prepareTransferMachineRuntime(transferId)
    await globalFileTransfer?.resumeTransfer?.(transferId)
  }, [globalFileTransfer, prepareTransferMachineRuntime])

  const resumeAllGlobalTransfers = useCallback(async () => {
    if (!globalFileTransfer) return
    const machineIds = new Set(
      globalFileTransfer.getSnapshot().transfers
        .map((transfer) => transfer.machineId)
        .filter((machineId): machineId is string => Boolean(machineId)),
    )
    for (const machineId of machineIds) {
      const machine = displayMachines.find((item) => item.id === machineId)
      if (!machine) continue
      getMachineRuntime(machine)
    }
    await globalFileTransfer.resumeAllTransfers?.()
  }, [displayMachines, getMachineRuntime, globalFileTransfer])

  useEffect(() => {
    refreshMachines()
  }, [refreshMachines])

  useEffect(() => {
    setAuthorizedMachineIds(readAuthorizedMachineIds(storage, undefined, externalPairingAdapter))
    setAuthorizationExpiries(readAuthorizationExpiries(storage, externalPairingAdapter))
  }, [externalPairingAdapter, pairVersion, storage])

  const updateTerminalSettings = useCallback((patch: Partial<TerminalSettings>) => {
    setTerminalSettings((current) => writeTerminalSettings({ ...current, ...patch }, storage))
  }, [storage])

  const openAddLocalSheet = useCallback(() => {
    hapticImpact()
    setSelectedMachineId(null)
    setPairIntent('add-local')
    setSharePreview(null)
    if (!scannerReloadRequired) setError(null)
    setScanOpen(true)
  }, [scannerReloadRequired])

  const openPairSheet = useCallback((machineId: string) => {
    hapticImpact()
    setSelectedMachineId(machineId)
    setPairIntent('authorize-machine')
    setSharePreview(null)
    if (!scannerReloadRequired) setError(null)
    setScanOpen(true)
  }, [scannerReloadRequired])

  const openMachinePairSheet = useCallback((machine: DisplayMachine) => {
    openPairSheet(machine.id)
  }, [openPairSheet])

  const closePairSheet = useCallback(() => {
    setSharePreview(null)
    setSSHCredentialNotice(null)
    setScanOpen(false)
  }, [])

  const requestPairSheetClose = useCallback(() => {
    const activeScan = cameraScanAbortRef.current
    if (activeScan) {
      activeScan.abort()
      return
    }
    closePairSheet()
  }, [closePairSheet])

  const selectMachine = useCallback((machine: DisplayMachine) => {
    hapticImpact()
    setSelectedMachineId(machine.id)
    if (!authorizedMachineIds.has(machine.id)) {
      openMachinePairSheet(machine)
      return
    }
    setView('machine')
    setError(null)
  }, [openMachinePairSheet, authorizedMachineIds])

  const storeImportedMachine = useCallback((external: ExternalPairingImportResult) => {
    if (!storage) throw new Error('Local storage is required before importing a AnyTTY QR')
    if (selectedMachine && selectedMachine.id !== external.machine.id) {
      throw new Error(`This code belongs to ${external.machine.name}, not ${selectedMachine.name}`)
    }
    const store = createMachineStore({ storage })
    const timestamp = new Date().toISOString()
    const existing = store.getMachine(external.machine.id)
    store.saveMachine({
      machineId: external.machine.id,
      name: selectedMachine?.name ?? external.machine.name,
      ...((selectedMachine?.hostname ?? external.machine.hostname) ? { hostname: selectedMachine?.hostname ?? external.machine.hostname } : {}),
      state: external.authorizationRequired ? 'offline' : 'online',
      terminalCount: existing?.terminalCount ?? 0,
      source: existing?.source ?? 'manual',
      accessClass: external.machine.accessClass ?? 'local',
      addresses: existing?.addresses ?? { local: [], lan: [], public: [] },
      endpoints: {
        ...(existing?.endpoints ?? {}),
      },
      addedAt: existing?.addedAt ?? timestamp,
      updatedAt: timestamp,
    })
    dropMachineRuntime(external.machine.id)
    setLocalMachines(store.listMachines())
    setSelectedMachineId(external.machine.id)
    setAuthorizedMachineIds(readAuthorizedMachineIds(storage, undefined, externalPairingAdapter))
    setAuthorizationExpiries(readAuthorizationExpiries(storage, externalPairingAdapter))
    setPairVersion((current) => current + 1)
    setSharePreview(null)
    const sshCredentials = external.sshCredentials?.filter((credential) => credential.authorizedKey.trim() !== '') ?? []
    setSSHCredentialNotice(sshCredentials.length > 0 ? sshCredentials : null)
    setScanOpen(sshCredentials.length > 0)
    setView(external.authorizationRequired ? 'home' : 'machine')
    hapticSuccess()
  }, [dropMachineRuntime, externalPairingAdapter, selectedMachine, storage])

  const pairScannedValue = useCallback(async (rawValue: string) => {
    if (!storage) {
      setError(t('errors.storageRequired'))
      return false
    }
    setPairing(true)
    setScanFlowState('pairing')
    setError(null)
    try {
	  if (rawValue.trim().startsWith('anytty://share?payload=')) {
		if (!externalPairingAdapter?.inspectShare) throw new Error('Endpoint share is unavailable in this client')
		const preview = await externalPairingAdapter.inspectShare(rawValue)
		setSharePreview(preview)
		setScanFlowState('idle')
		return true
	  }
      const external = await externalPairingAdapter?.import(rawValue, selectedMachine?.id)
      if (external) {
		storeImportedMachine(external)
        return true
      }
      throw new Error('Proto binding pairing adapter is required')
    } catch (err) {
      hapticError()
      console.warn('[anytty:pairing] pair claim failed', err instanceof Error ? err.message : String(err))
      setError(localizedAppError(err, t))
      return false
    } finally {
      setPairing(false)
      setScanFlowState('idle')
    }
  }, [externalPairingAdapter, selectedMachine?.id, storage, storeImportedMachine, t])

  const commitEndpointShare = useCallback(async () => {
	if (!sharePreview || !externalPairingAdapter?.commitShare) return
	setPairing(true)
	setError(null)
	try {
	  const imported = await externalPairingAdapter.commitShare(sharePreview.importToken)
	  storeImportedMachine(imported)
	} catch (err) {
	  hapticError()
	  setError(localizedAppError(err, t))
	} finally {
	  setPairing(false)
	}
  }, [externalPairingAdapter, sharePreview, storeImportedMachine, t])

  const scanWithCamera = useCallback(async () => {
    if (!scanPairingCode) return
    if (cameraScanInFlightRef.current) return
    const controller = new AbortController()
    cameraScanInFlightRef.current = true
    cameraScanAbortRef.current = controller
    restoreCameraFocusAfterScanRef.current = false
    hapticImpact()
    setCameraScanning(true)
    setScanFlowState('scanning')
    setError(null)
    try {
      const value = await scanPairingCode({ signal: controller.signal })
      if (cameraScanAbortRef.current === controller) cameraScanAbortRef.current = null
      if (!value) return
      restoreCameraFocusAfterScanRef.current = !(await pairScannedValue(value))
    } catch (err) {
      const presentation = cameraScanErrorPresentation(err, t)
      restoreCameraFocusAfterScanRef.current = true
      setScannerReloadRequired(presentation.reloadRequired)
      setError(presentation.message)
    } finally {
      if (cameraScanAbortRef.current === controller) cameraScanAbortRef.current = null
      cameraScanInFlightRef.current = false
      setCameraScanning(false)
      setScanFlowState((current) => current === 'scanning' ? 'idle' : current)
    }
  }, [pairScannedValue, scanPairingCode, t])

  useEffect(() => {
    if (cameraScanning || !restoreCameraFocusAfterScanRef.current) return undefined
    restoreCameraFocusAfterScanRef.current = false
    const frame = window.requestAnimationFrame(() => {
      const button = cameraScanButtonRef.current
      if (!button?.isConnected || button.disabled || button.closest('[inert], [aria-hidden="true"]')) return
      button.focus()
    })
    return () => window.cancelAnimationFrame(frame)
  }, [cameraScanning])

  const handleMachineNeedsReauthorization = useCallback((machineId: string) => {
    if (!storage) return
    dropMachineRuntime(machineId)
    setAuthorizedMachineIds(readAuthorizedMachineIds(storage, undefined, externalPairingAdapter))
    setAuthorizationExpiries(readAuthorizationExpiries(storage, externalPairingAdapter))
    setPairVersion((current) => current + 1)
    setSelectedMachineId(machineId)
    setPairIntent('authorize-machine')
    setError(t('errors.pairAgain'))
    setScanOpen(true)
  }, [dropMachineRuntime, externalPairingAdapter, storage, t])

  const forgetMachineAuthorization = useCallback((machine: RemoteMachine) => {
    if (!storage) return
    const store = createMachineStore({ storage })
    void externalPairingAdapter?.forget(machine.id)
    dropMachineRuntime(machine.id)
    store.forgetMachine(machine.id)
    setLocalMachines(store.listMachines())
    setAuthorizedMachineIds(readAuthorizedMachineIds(storage, undefined, externalPairingAdapter))
    setAuthorizationExpiries(readAuthorizationExpiries(storage, externalPairingAdapter))
    setPairVersion((current) => current + 1)
    setSelectedMachineId((current) => current === machine.id ? null : current)
    setView((current) => current === 'machine' && selectedMachineId === machine.id ? 'home' : current)
    setError(null)
  }, [dropMachineRuntime, externalPairingAdapter, selectedMachineId, storage])

  const disconnectMachineConnection = useCallback(async (machine: RemoteMachine) => {
    const runtime = runtimeCacheRef.current?.runtimes.get(machine.id)
    if (!runtime?.disconnect) return
    await runtime.disconnect()
  }, [])

  useNativeBackHandler(() => {
    requestPairSheetClose()
  }, NATIVE_BACK_PRIORITY.SCANNER, scanOpen)

  useNativeBackHandler(() => {
    if (view === 'settings') {
      setView('home')
      return
    }
    if (view === 'machine') {
      setView('home')
      setError(null)
    }
  }, NATIVE_BACK_PRIORITY.ROOT, view !== 'home')

  return (
    <main
      className="anytty-app-page flex h-full min-h-0 flex-col"
      data-testid="anytty-web-control-remote"
      style={appThemeStyle}
    >
      {view === 'settings' ? (
        <SettingsView
          error={error}
          terminalSettings={terminalSettings}
          onBack={() => { hapticSelection(); setView('home') }}
          onTerminalSettingsChange={updateTerminalSettings}
          onExportDebugLogs={exportDebugLogs}
        />
      ) : view === 'machine' && selectedMachine ? (
        <MachineTerminalListView
          machine={selectedMachine}
          storage={storage}
          terminalSettings={terminalSettings}
          runtime={getMachineRuntime(selectedMachine)}
          phoneOnline={remoteNetworkState.phoneOnline}
          connectionReady={effectiveConnectionReady}
          connectionRecoveryFailed={connectionRecoveryFailed}
          onRetryConnectionRecovery={onRetryConnectionRecovery}
          singlePaneWorkspace={singlePaneWorkspace}
          onBack={() => {
            hapticSelection()
            setView('home')
            setError(null)
          }}
          onNeedsReauthorization={handleMachineNeedsReauthorization}
          onTerminalSettingsChange={updateTerminalSettings}
        />
      ) : (
        <HomeView
          fileTransfer={globalFileTransfer}
          transferState={globalTransferState as { transfers: TransferInfo[]; hasActiveTransfers: boolean }}
          machines={displayMachines}
          getConnectionStateSource={(machine) => authorizedMachineIds.has(machine.id) ? getExistingMachineRuntime(machine)?.listConnectionState : undefined}
          authorizedMachineIds={authorizedMachineIds}
          authorizationExpiries={authorizationExpiries}
          phoneOnline={remoteNetworkState.phoneOnline}
          connectionReady={effectiveConnectionReady}
          connectionRecoveryFailed={connectionRecoveryFailed}
          onRetryConnectionRecovery={onRetryConnectionRecovery}
          onRefresh={performMachineRefresh}
          onAddLocalDevice={openAddLocalSheet}
          onOpenSettings={() => { hapticSelection(); setView('settings') }}
          onOpenTransferCenter={() => { hapticSelection(); setTransferCenterOpen(true) }}
          onForgetMachineAuthorization={forgetMachineAuthorization}
          onDisconnectMachine={disconnectMachineConnection}
          onSelectMachine={selectMachine}
        />
      )}

      {scanOpen ? (
        <PairSheet
          pairError={error}
          scanFlowState={scanFlowState}
          pairing={pairing}
          sharePreview={sharePreview}
          sshCredentialNotice={sshCredentialNotice}
          cameraScanning={cameraScanning}
          scannerReloadRequired={scannerReloadRequired}
          cameraButtonRef={cameraScanButtonRef}
          pairIntent={pairIntent}
          selectedMachine={selectedMachine}
          canScanWithCamera={Boolean(scanPairingCode)}
          onCommitShare={() => void commitEndpointShare()}
          onClose={() => { hapticSelection(); requestPairSheetClose() }}
          onPairPastedValue={(value) => { hapticImpact(); void pairScannedValue(value) }}
          onScanWithCamera={() => void scanWithCamera()}
        />
      ) : null}
      {transferCenterOpen ? (
        <GlobalTransferCenter
          fileTransfer={globalFileTransfer}
          resolveMachineLabel={(machineId) => displayMachines.find((machine) => machine.id === machineId)?.name ?? machineId}
          onClose={() => { hapticSelection(); setTransferCenterOpen(false) }}
          onResumeTransfer={resumeGlobalTransfer}
          onResumeAllTransfers={resumeAllGlobalTransfers}
        />
      ) : null}
    </main>
  )
}

function GlobalTransferCenter({
  fileTransfer,
  resolveMachineLabel,
  onClose,
  onResumeTransfer,
  onResumeAllTransfers,
}: {
  fileTransfer: FileTransferContext | undefined
  resolveMachineLabel?: ((machineId: string | undefined) => string | null | undefined) | undefined
  onClose: () => void
  onResumeTransfer?: ((id: string) => void | Promise<void>) | undefined
  onResumeAllTransfers?: (() => void | Promise<void>) | undefined
}) {
  const emptySnapshot = useMemo(() => ({ transfers: [], hasActiveTransfers: false }), [])
  const transferState = useSyncExternalStore(
    fileTransfer?.subscribe ?? noopSubscribe,
    fileTransfer?.getSnapshot ?? (() => emptySnapshot),
  )

  if (!fileTransfer) return null

  return (
    <FileTransferPanel
      transfers={transferState.transfers}
      hasActiveTransfers={transferState.hasActiveTransfers}
      resolveMachineLabel={resolveMachineLabel}
      onCancel={(id) => fileTransfer.cancelTransfer(id)}
      onDismiss={(id) => fileTransfer.dismissTransfer(id)}
      onPause={(id) => fileTransfer.pauseTransfer?.(id)}
      onResume={onResumeTransfer ?? ((id) => fileTransfer.resumeTransfer?.(id))}
      onResumeAll={onResumeAllTransfers ?? (() => fileTransfer.resumeAllTransfers?.())}
      open
      onOpenChange={(open) => {
        if (!open) onClose()
      }}
    />
  )
}

function MachineTerminalListView({
  machine,
  storage,
  terminalSettings,
  runtime,
  phoneOnline,
  connectionReady,
  connectionRecoveryFailed,
  onRetryConnectionRecovery,
  singlePaneWorkspace,
  onBack,
  onNeedsReauthorization,
  onTerminalSettingsChange,
}: {
  machine: DisplayMachine
  storage: RemoteRuntimeStorage | undefined
  terminalSettings: TerminalSettings
  runtime: MachineRuntime | null
  phoneOnline: boolean
  connectionReady: boolean
  connectionRecoveryFailed: boolean
  onRetryConnectionRecovery?: (() => void | Promise<void>) | undefined
  singlePaneWorkspace: boolean
  onBack: () => void
  onNeedsReauthorization: (machineId: string) => void
  onTerminalSettingsChange: (patch: Partial<TerminalSettings>) => void
}) {
  if (!storage || !runtime) {
    return (
      <MachineRuntimeErrorShell
        machine={machine}
        onBack={onBack}
      />
    )
  }
  return (
    <section className="anytty-app-page flex min-h-0 flex-1 flex-col animate-in fade-in slide-in-from-right-4 duration-200" data-testid="anytty-machine-terminal-list">
      <MachineWorkspace
        api={runtime.api}
        connector={runtime.connector}
        className="min-h-0 flex-1"
        initialMachine={{
          machineId: machine.id,
          name: machine.name,
          state: machine.online ? 'online' : 'offline',
          ...(machine.lastSeen ? { lastSeenAt: machine.lastSeen } : {}),
        }}
        connectionStateEvents={runtime.connectionStateEvents}
        inventoryEvents={runtime.inventoryEvents}
        fileTransfer={runtime.fileTransfer}
        terminalSettings={terminalSettings}
        phoneOnline={phoneOnline}
        connectionReady={connectionReady}
        connectionRecoveryFailed={connectionRecoveryFailed}
        onRetryConnectionRecovery={onRetryConnectionRecovery}
        singlePane={singlePaneWorkspace}
        onNeedsReauthorization={onNeedsReauthorization}
        onTerminalSettingsChange={onTerminalSettingsChange}
        onBack={onBack}
      />
    </section>
  )
}

function MachineRuntimeHeader({ machine, onBack }: { machine: DisplayMachine; onBack: () => void }) {
  const { t } = useTranslation()
  return (
    <header className="anytty-app-header flex min-h-12 shrink-0 items-center gap-2 border-b px-3 pb-2 pt-[calc(env(safe-area-inset-top)+0.5rem)]">
      <button
        aria-label={t('common.backToMachines')}
        className="anytty-app-icon-button focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)]"
        type="button"
        onClick={() => { hapticSelection(); onBack() }}
      >
        <ArrowLeft className="h-5 w-5" />
      </button>
      <div className="flex min-w-0 flex-1 items-center gap-2">
        <Monitor className="h-5 w-5 shrink-0 text-zinc-500" />
        <div className="min-w-0">
          <h1 className="truncate text-base font-semibold leading-6 text-zinc-950">{machine.name}</h1>
          <p className="truncate text-xs font-medium text-zinc-500">{machine.hostname || machine.id}</p>
        </div>
      </div>
      <span className={`shrink-0 border px-2 py-1 text-[10px] font-semibold leading-4 ${machine.online ? 'border-emerald-200 text-emerald-700' : 'border-zinc-300 text-zinc-600'}`}>
        {t(machine.online ? 'machines.state.online' : 'machines.state.offline')}
      </span>
    </header>
  )
}

function MachineRuntimeErrorShell({
  machine,
  onBack,
}: {
  machine: DisplayMachine
  onBack: () => void
}) {
  const { t } = useTranslation()
  return (
    <section className="anytty-app-page flex min-h-0 flex-1 flex-col animate-in fade-in slide-in-from-right-4 duration-200" data-testid="anytty-machine-terminal-list">
      <MachineRuntimeHeader machine={machine} onBack={onBack} />
      <div className="flex min-h-0 flex-1 items-center justify-center px-6 py-10">
        <div className="w-full max-w-sm text-center">
          <span className="mx-auto grid size-12 place-items-center border border-zinc-300 bg-white text-zinc-600" aria-hidden="true">
            <WifiOff className="h-5 w-5" />
          </span>
          <h2 className="mt-4 text-base font-semibold text-zinc-950">{t('errors.connectionProblemTitle')}</h2>
          <p className="mt-2 text-sm leading-6 text-zinc-600">{t('errors.connectionInterrupted')}</p>
          <button className="anytty-app-secondary-button mt-5 h-11 px-4 text-sm font-semibold" type="button" onClick={onBack}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t('common.backToMachines')}
          </button>
        </div>
      </div>
    </section>
  )
}

function HomeView({
  fileTransfer,
  transferState,
  machines,
  getConnectionStateSource,
  authorizedMachineIds,
  authorizationExpiries,
  phoneOnline,
  connectionReady,
  connectionRecoveryFailed,
  onAddLocalDevice,
  onDisconnectMachine,
  onForgetMachineAuthorization,
  onOpenSettings,
  onOpenTransferCenter,
  onRefresh,
  onRetryConnectionRecovery,
  onSelectMachine,
}: {
  fileTransfer?: FileTransferContext | undefined
  transferState: { transfers: TransferInfo[]; hasActiveTransfers: boolean }
  machines: DisplayMachine[]
  getConnectionStateSource: (machine: DisplayMachine) => MachineRuntime['listConnectionState']
  authorizedMachineIds: Set<string>
  authorizationExpiries: Map<string, string>
  phoneOnline: boolean
  connectionReady: boolean
  connectionRecoveryFailed: boolean
  onAddLocalDevice: () => void
  onDisconnectMachine: (machine: DisplayMachine) => void | Promise<void>
  onForgetMachineAuthorization: (machine: DisplayMachine) => void
  onOpenSettings: () => void
  onOpenTransferCenter: () => void
  onRefresh: () => Promise<void>
  onRetryConnectionRecovery?: (() => void | Promise<void>) | undefined
  onSelectMachine: (machine: DisplayMachine) => void
}) {
  const { t } = useTranslation()
  const [detailMachine, setDetailMachine] = useState<DisplayMachine | null>(null)
  const [refreshing, setRefreshing] = useState(false)
  const [refreshFeedback, setRefreshFeedback] = useState<'success' | 'error' | 'offline' | null>(null)
  const [pullDistance, setPullDistance] = useState(0)
  const refreshFeedbackTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const refreshInFlightRef = useRef(false)
  const pullStartYRef = useRef<number | null>(null)
  const pullArmedRef = useRef(false)
  const runRefresh = useCallback(async () => {
    if (refreshInFlightRef.current) return
    if (refreshFeedbackTimerRef.current) clearTimeout(refreshFeedbackTimerRef.current)
    setPullDistance(0)
    if (!phoneOnline) {
      setRefreshFeedback('offline')
      hapticError()
      refreshFeedbackTimerRef.current = setTimeout(() => setRefreshFeedback(null), 2_400)
      return
    }
    if (!connectionReady) return
    setRefreshing(true)
    refreshInFlightRef.current = true
    setRefreshFeedback(null)
    try {
      await onRefresh()
      setRefreshFeedback('success')
      hapticSuccess()
    } catch {
      setRefreshFeedback('error')
      hapticError()
    } finally {
      refreshInFlightRef.current = false
      setRefreshing(false)
      refreshFeedbackTimerRef.current = setTimeout(() => setRefreshFeedback(null), 2_000)
    }
  }, [connectionReady, onRefresh, phoneOnline])
  useEffect(() => () => {
    if (refreshFeedbackTimerRef.current) clearTimeout(refreshFeedbackTimerRef.current)
  }, [])
  const handlePullStart = useCallback((event: ReactTouchEvent<HTMLDivElement>) => {
    if (event.currentTarget.scrollTop > 0 || refreshing || !connectionReady) return
    pullStartYRef.current = event.touches[0]?.clientY ?? null
    pullArmedRef.current = false
  }, [connectionReady, refreshing])
  const handlePullMove = useCallback((event: ReactTouchEvent<HTMLDivElement>) => {
    const startY = pullStartYRef.current
    const currentY = event.touches[0]?.clientY
    if (startY === null || currentY === undefined || event.currentTarget.scrollTop > 0) return
    const delta = Math.max(0, currentY - startY)
    if (delta <= 0) return
    event.preventDefault()
    const distance = Math.min(88, Math.round(delta * 0.55))
    pullArmedRef.current = distance >= 60
    setPullDistance(distance)
  }, [])
  const handlePullEnd = useCallback(() => {
    const shouldRefresh = pullArmedRef.current
    pullStartYRef.current = null
    pullArmedRef.current = false
    setPullDistance(0)
    if (shouldRefresh) void runRefresh()
  }, [runRefresh])
  const handlePullCancel = useCallback(() => {
    pullStartYRef.current = null
    pullArmedRef.current = false
    setPullDistance(0)
  }, [])
  const refreshStatus = refreshing
    ? t('machines.refreshing')
    : refreshFeedback === 'success'
      ? t('machines.refreshed')
      : refreshFeedback === 'error'
        ? t('machines.refreshFailed')
        : refreshFeedback === 'offline'
          ? t('machines.refreshOffline')
          : pullArmedRef.current
            ? t('machines.releaseToRefresh')
            : t('machines.pullToRefresh')
  const refreshIndicatorVisible = refreshing || refreshFeedback !== null || pullDistance > 0
  return (
    <section className="anytty-app-page flex min-h-0 flex-1 flex-col" data-testid="anytty-app-home">
      <header className="anytty-app-header flex min-h-12 shrink-0 items-center justify-between gap-2 border-b px-3 pb-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] lg:h-16 lg:px-6 lg:py-0">
        <div className="flex min-w-0 items-center gap-3 lg:gap-5">
          <span aria-hidden="true" className="shrink-0 text-sm font-bold text-zinc-950 lg:hidden">AnyTTY</span>
          <span aria-hidden="true" className="hidden text-base font-bold text-zinc-950 lg:inline">AnyTTY</span>
          <div className="hidden h-5 w-px bg-zinc-200 lg:block" />
          <div className="min-w-0 lg:flex lg:items-center lg:gap-3">
            <h1 className="text-lg font-semibold leading-6 lg:text-sm">{t('machines.title')}</h1>
            <p className="truncate text-xs font-medium text-zinc-500 lg:border-l lg:border-zinc-200 lg:pl-3">
            {t('machines.savedCount', { count: machines.length })}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button
            aria-label={t('machines.refresh')}
            className="anytty-app-icon-button focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] disabled:opacity-60"
            disabled={refreshing || !connectionReady}
            title={t('machines.refresh')}
            type="button"
            onClick={() => { hapticSelection(); void runRefresh() }}
          >
            <RefreshCw className={`h-5 w-5 ${refreshing ? 'animate-spin' : ''}`} />
          </button>
          <button
            aria-label={t('machines.scanPairing')}
            className="anytty-app-primary-button min-w-11 gap-2 px-2.5 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] lg:px-3"
            type="button"
            onClick={onAddLocalDevice}
          >
            <QrCode className="h-5 w-5" />
            <span className="hidden text-xs font-semibold lg:inline">{t('machines.scanService')}</span>
          </button>
          {fileTransfer ? (
            <button
              aria-label={t('machines.transfers')}
              className="anytty-app-icon-button relative focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)]"
              type="button"
              onClick={onOpenTransferCenter}
            >
              <Download className="h-5 w-5" />
              {transferState.hasActiveTransfers ? <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-emerald-500" /> : null}
            </button>
          ) : null}
          <button
            aria-label={t('machines.openSettings')}
            className="anytty-app-icon-button focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)]"
            type="button"
            onClick={onOpenSettings}
          >
            <Settings className="h-5 w-5" />
          </button>
        </div>
      </header>

      {!phoneOnline ? (
        <div className="flex min-h-10 shrink-0 items-center gap-2 border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs font-medium text-amber-900" role="status" aria-live="polite">
          <WifiOff className="h-4 w-4 shrink-0" />
          <span>{t('machines.offlineNotice')}</span>
        </div>
      ) : connectionRecoveryFailed ? (
        <div className="flex min-h-10 shrink-0 items-center gap-3 border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs font-medium text-amber-950" role="alert">
          <Link2Off className="h-4 w-4 shrink-0" />
          <span className="min-w-0 flex-1">{t('machines.networkRecoveryFailed')}</span>
          {onRetryConnectionRecovery ? (
            <button className="min-h-11 shrink-0 border border-amber-300 bg-white px-3 text-xs font-semibold" type="button" onClick={() => { hapticSelection(); void onRetryConnectionRecovery() }}>
              {t('workspace.connection.retry')}
            </button>
          ) : null}
        </div>
      ) : !connectionReady ? (
        <div className="flex min-h-10 shrink-0 items-center gap-2 border-b border-zinc-200 bg-white px-4 py-2 text-xs font-medium text-zinc-700" role="status" aria-live="polite">
          <span className="anytty-square-spinner h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{t('machines.restoringNetwork')}</span>
        </div>
      ) : null}
      {phoneOnline && machines.length === 0 && (refreshing || refreshFeedback) ? (
        <div className={`flex min-h-10 shrink-0 items-center gap-2 border-b px-4 py-2 text-xs font-medium ${refreshFeedback === 'error' ? 'border-red-200 bg-red-50 text-red-800' : 'border-zinc-200 bg-white text-zinc-700'}`} role="status" aria-live="polite">
          {refreshFeedback === 'success' ? <Check className="h-4 w-4 text-emerald-700" /> : <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />}
          <span>{refreshStatus}</span>
        </div>
      ) : null}

      {machines.length === 0 ? (
        <FirstUseState
          onAddLocalDevice={onAddLocalDevice}
        />
      ) : (
        <div
          className="relative min-h-0 flex-1 overflow-y-auto overscroll-y-contain py-4 lg:px-8 lg:py-7"
          data-testid="anytty-machine-list-scroller"
          onTouchStart={handlePullStart}
          onTouchMove={handlePullMove}
          onTouchEnd={handlePullEnd}
          onTouchCancel={handlePullCancel}
        >
          {refreshIndicatorVisible ? (
            <div
              className="flex items-center justify-center overflow-hidden text-xs font-semibold text-zinc-600 transition-[height,opacity] duration-200 motion-reduce:transition-none"
              style={{ height: Math.max(40, pullDistance) }}
              role="status"
              aria-live="polite"
            >
              {refreshFeedback === 'success' ? <Check className="mr-2 h-4 w-4 text-emerald-700" /> : <RefreshCw className={`mr-2 h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />}
              {refreshStatus}
            </div>
          ) : null}
          <div className="anytty-app-panel mx-auto w-full max-w-7xl border-x-0 lg:overflow-visible lg:border-x">
            <div className="hidden grid-cols-[40px_minmax(180px,1.3fr)_minmax(160px,.8fr)_minmax(180px,1fr)_32px] items-center gap-4 border-b border-zinc-200 bg-zinc-50 px-4 py-2.5 text-[11px] font-semibold uppercase text-zinc-500 lg:grid">
              <span aria-hidden="true" />
              <span>{t('machines.columns.machine')}</span>
              <span>{t('machines.columns.access')}</span>
              <span>{t('machines.columns.connection')}</span>
              <span aria-hidden="true" />
            </div>
            <ul aria-label={t('machines.title')} className="divide-y divide-[var(--anytty-app-line)]">
          {machines.map((machine) => (
            <li key={machine.id}>
              <MachineRow
                authorizationExpiresAt={authorizationExpiries.get(machine.id)}
                authorizationState={machineAuthorizationState(machine, authorizedMachineIds, authorizationExpiries)}
                machine={machine}
                connectionStateSource={getConnectionStateSource(machine)}
                onDisconnectMachine={onDisconnectMachine}
                onForgetMachineAuthorization={onForgetMachineAuthorization}
                onSelectMachine={onSelectMachine}
                onShowDetails={setDetailMachine}
              />
            </li>
          ))}
            </ul>
          </div>
        </div>
      )}
      {detailMachine ? <DisplayMachineDetailSheet machine={detailMachine} onClose={() => setDetailMachine(null)} /> : null}
    </section>
  )
}

function FirstUseState({
  onAddLocalDevice,
}: {
  onAddLocalDevice: () => void
}) {
  const { t } = useTranslation()
  return (
    <div className="flex min-h-0 flex-1 items-start justify-center overflow-y-auto px-4 py-10 md:items-center">
      <section className="anytty-app-panel w-full max-w-md p-6" data-testid="anytty-first-use">
        <div className="flex h-10 w-10 items-center justify-center rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] text-[var(--anytty-app-accent)]">
          <Server className="h-6 w-6" />
        </div>
        <h2 className="mt-5 text-lg font-semibold text-zinc-950">{t('machines.emptyTitle')}</h2>
        <p className="mt-2 text-sm leading-6 text-zinc-600">{t('machines.emptyServiceCopy')}</p>
        <div className="mt-6 grid gap-3">
          <button className="anytty-app-primary-button h-12 gap-2 px-4 text-sm font-semibold" type="button" onClick={onAddLocalDevice}>
            <QrCode className="h-4 w-4" />
            {t('machines.scanService')}
          </button>
        </div>
      </section>
    </div>
  )
}

function SettingsView({
  error,
  terminalSettings,
  onBack,
  onTerminalSettingsChange,
  onExportDebugLogs,
}: {
  error: string | null
  terminalSettings: TerminalSettings
  onBack: () => void
  onTerminalSettingsChange: (patch: Partial<TerminalSettings>) => void
  onExportDebugLogs?: (() => Promise<void>) | undefined
}) {
  const { t, i18n } = useTranslation()
  const handleNumberSetting = (key: 'fontSize' | 'scrollback' | 'scrollbackPrefetchThresholdRows', min: number, max: number) =>
    (event: ChangeEvent<HTMLInputElement>) => {
      const value = Number(event.currentTarget.value)
      if (!Number.isFinite(value)) return
      onTerminalSettingsChange({ [key]: Math.max(min, Math.min(max, Math.round(value))) })
    }
  const themeGroups = useMemo(() => ({
    dark: TERMINAL_THEME_OPTIONS.filter((option) => option.group === 'dark'),
    light: TERMINAL_THEME_OPTIONS.filter((option) => option.group === 'light'),
  }), [])

  return (
    <section className="anytty-app-page flex min-h-0 flex-1 flex-col animate-in fade-in slide-in-from-bottom-4 duration-200" data-testid="anytty-app-settings">
      <header className="anytty-app-header flex min-h-12 shrink-0 items-center gap-2 border-b px-3 pb-2 pt-[calc(env(safe-area-inset-top)+0.5rem)]">
        <button
          aria-label={t('common.backToMachines')}
          className="anytty-app-icon-button focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)]"
          type="button"
          onClick={onBack}
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div className="min-w-0 flex-1">
          <h1 className="text-lg font-semibold leading-6 text-zinc-900">{t('common.settings')}</h1>
          <p className="truncate text-xs font-medium text-zinc-500">{t('settings.deviceAccess')}</p>
        </div>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 pb-[calc(env(safe-area-inset-bottom)+1.5rem)]">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          {error ? (
            <p className="border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-700">{error}</p>
          ) : null}

          <SettingsSection title={t('common.language')}>
            <SettingsRow label={t('settings.languageHint')}>
              <SettingsSelect
                ariaLabel={t('common.language')}
                value={normalizeAnyTTYLanguage(i18n.resolvedLanguage)}
                onChange={(value) => { hapticSelection(); void i18n.changeLanguage(value) }}
              >
                {anyttyLanguages.map((language) => <option key={language.id} value={language.id}>{language.label}</option>)}
              </SettingsSelect>
            </SettingsRow>
          </SettingsSection>

          {onExportDebugLogs ? (
            <SettingsSection title={t('settings.diagnostics')}>
              <div className="px-4 py-3">
                <button
                  className="anytty-app-primary-button h-11 w-full gap-2 px-3 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
                  type="button"
                  onClick={() => {
                    hapticImpact()
                    void onExportDebugLogs()
                  }}
                >
                  <Download className="h-4 w-4" />
                  {t('settings.exportLogs')}
                </button>
              </div>
            </SettingsSection>
          ) : null}

          <SettingsSection title={t('settings.terminal')}>
            <SettingsRow label={t('settings.fontSize')}>
              <div className="inline-flex h-11 items-center overflow-hidden border border-[var(--anytty-app-line)] bg-white">
                <button
                  aria-label={t('settings.decreaseFont')}
                  className="h-11 w-11 text-lg font-semibold text-zinc-700 hover:bg-zinc-50 active:bg-zinc-100"
                  type="button"
                  onClick={() => { hapticSelection(); onTerminalSettingsChange({ fontSize: Math.max(8, terminalSettings.fontSize - 1) }) }}
                >
                  -
                </button>
                <input
                  aria-label={t('settings.fontSize')}
                  className="h-11 w-12 border-x border-[var(--anytty-app-line)] bg-zinc-50 px-1 text-center text-sm font-semibold text-zinc-900 outline-none focus:ring-2 focus:ring-blue-500/25"
                  inputMode="numeric"
                  max={32}
                  min={8}
                  type="number"
                  value={terminalSettings.fontSize}
                  onChange={handleNumberSetting('fontSize', 8, 32)}
                />
                <button
                  aria-label={t('settings.increaseFont')}
                  className="h-11 w-11 text-lg font-semibold text-zinc-700 hover:bg-zinc-50 active:bg-zinc-100"
                  type="button"
                  onClick={() => { hapticSelection(); onTerminalSettingsChange({ fontSize: Math.min(32, terminalSettings.fontSize + 1) }) }}
                >
                  +
                </button>
              </div>
            </SettingsRow>
            <SettingsRow label={t('settings.font')} stacked>
              <FontPicker
                value={terminalSettings.fontFamily}
                onChange={(value) => onTerminalSettingsChange({ fontFamily: value })}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.theme')} stacked>
              <ThemePicker
                groups={themeGroups}
                value={terminalSettings.themeId}
                onChange={(value) => onTerminalSettingsChange({ themeId: value })}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.renderer')}>
              <SettingsSelect
                ariaLabel={t('settings.renderer')}
                value={terminalSettings.renderer}
                onChange={(value) => { hapticSelection(); onTerminalSettingsChange({ renderer: value as TerminalRenderer }) }}
              >
                <option value="auto">{t('settings.auto')}</option>
                <option value="webgl">WebGL</option>
                <option value="canvas">Canvas</option>
                <option value="dom">DOM</option>
              </SettingsSelect>
            </SettingsRow>
            <SettingsRow label={t('settings.keyboard')}>
              <SettingsSelect
                ariaLabel={t('settings.keyboard')}
                value={terminalSettings.keyboardMode}
                onChange={(value) => { hapticSelection(); onTerminalSettingsChange({ keyboardMode: value as TerminalKeyboardMode }) }}
              >
                <option value="auto">{t('settings.auto')}</option>
                <option value="resize">{t('settings.resize')}</option>
                <option value="shift">{t('settings.shiftUp')}</option>
              </SettingsSelect>
            </SettingsRow>
            <SettingsRow label={t('settings.scrollback')}>
              <input
                aria-label={t('settings.scrollback')}
                className="h-11 w-28 border border-[var(--anytty-app-line)] bg-white px-3 text-right text-sm font-semibold text-zinc-900 outline-none focus:border-[var(--anytty-app-accent)] focus:ring-2 focus:ring-blue-500/25"
                inputMode="numeric"
                max={50000}
                min={500}
                step={500}
                type="number"
                value={terminalSettings.scrollback}
                onChange={handleNumberSetting('scrollback', 500, 50000)}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.prefetch')}>
              <input
                aria-label={t('settings.prefetch')}
                className="h-11 w-28 border border-[var(--anytty-app-line)] bg-white px-3 text-right text-sm font-semibold text-zinc-900 outline-none focus:border-[var(--anytty-app-accent)] focus:ring-2 focus:ring-blue-500/25"
                inputMode="numeric"
                max={1000}
                min={0}
                step={10}
                type="number"
                value={terminalSettings.scrollbackPrefetchThresholdRows}
                onChange={handleNumberSetting('scrollbackPrefetchThresholdRows', 0, 1000)}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.cursorBlink')}>
              <Switch
                ariaLabel={t('settings.cursorBlink')}
                checked={terminalSettings.cursorBlink}
                onChange={(checked) => onTerminalSettingsChange({ cursorBlink: checked })}
              />
            </SettingsRow>
          </SettingsSection>

        </div>
      </div>
    </section>
  )
}

function SettingsSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="mb-2 px-1 text-[10px] font-semibold uppercase text-[var(--anytty-app-muted)]">{title}</h2>
      <div className="anytty-app-panel overflow-hidden">
        {children}
      </div>
    </section>
  )
}

function SettingsRow({
  children,
  label,
  stacked = false,
  value,
}: {
  children?: ReactNode
  label: string
  stacked?: boolean
  value?: string | undefined
}) {
  if (stacked) {
    return (
      <div className="flex min-h-12 flex-col items-stretch gap-3 border-b border-[var(--anytty-app-line)] px-4 py-3 last:border-b-0">
        <div className="min-w-0 text-sm font-medium text-zinc-900">{label}</div>
        {children ? (
          <div className="min-w-0 w-full">{children}</div>
        ) : (
          <div className="min-w-0 truncate text-sm font-medium text-zinc-500">{value}</div>
        )}
      </div>
    )
  }

  return (
    <div className="flex min-h-12 items-center justify-between gap-4 border-b border-[var(--anytty-app-line)] px-4 py-2 last:border-b-0">
      <div className="min-w-0 text-sm font-medium text-zinc-900">{label}</div>
      {children ? (
        <div className="shrink-0">{children}</div>
      ) : (
        <div className="min-w-0 truncate text-right text-sm font-medium text-zinc-500">{value}</div>
      )}
    </div>
  )
}

function SettingsSelect({
  ariaLabel,
  children,
  onChange,
  value,
}: {
  ariaLabel: string
  children: ReactNode
  onChange: (value: string) => void
  value: string
}) {
  return (
    <select
      aria-label={ariaLabel}
      className="h-11 max-w-[54vw] border border-[var(--anytty-app-line)] bg-white px-3 text-right text-sm font-semibold text-zinc-900 outline-none focus:border-[var(--anytty-app-accent)] focus:ring-2 focus:ring-blue-500/25 sm:max-w-xs"
      value={value}
      onChange={(event) => {
        hapticSelection()
        onChange(event.currentTarget.value)
      }}
    >
      {children}
    </select>
  )
}

function FontPicker({
  onChange,
  value,
}: {
  onChange: (value: string) => void
  value: string
}) {
  const { t } = useTranslation()
  return (
    <>
      <select
        aria-label={t('settings.terminalFont')}
        className="sr-only"
        value={value}
        onChange={(event) => {
          hapticSelection()
          onChange(event.currentTarget.value)
        }}
      >
        {TERMINAL_FONT_OPTIONS.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </select>
      <div
        aria-label={t('settings.fontPreviews')}
        className="grid w-full min-w-0 gap-2"
        role="radiogroup"
        style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 13rem), 1fr))' }}
      >
        {TERMINAL_FONT_OPTIONS.map((option) => (
          <FontPreviewButton
            key={option.value}
            option={option}
            selected={option.value === value}
            onSelect={() => onChange(option.value)}
          />
        ))}
      </div>
    </>
  )
}

function FontPreviewButton({
  onSelect,
  option,
  selected,
}: {
  onSelect: () => void
  option: { label: string; value: string }
  selected: boolean
}) {
  return (
    <button
      aria-checked={selected}
      className={`min-w-0 border p-3 text-left transition-colors duration-200 ${
        selected
          ? 'border-[var(--anytty-app-accent)] bg-blue-50'
          : 'border-zinc-200 bg-white hover:bg-zinc-50 active:bg-zinc-50'
      }`}
      role="radio"
      style={{ fontFamily: option.value }}
      type="button"
      onClick={() => {
        hapticSelection()
        onSelect()
      }}
    >
      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate text-sm font-semibold leading-5 text-zinc-950">{option.label}</span>
        <span className={`ml-auto h-2 w-2 shrink-0 ${selected ? 'bg-[var(--anytty-app-accent)]' : 'bg-zinc-200'}`} />
      </div>
      <div className="mt-2 bg-zinc-950 px-2 py-2 text-[12px] leading-5 text-zinc-100">
        <div className="truncate">$ anytty --font</div>
        <div className="truncate text-zinc-300">AaBb 012345 &lt;&gt; ~/</div>
      </div>
    </button>
  )
}

function ThemePicker({
  groups,
  onChange,
  value,
}: {
  groups: Record<'dark' | 'light', TerminalThemeOption[]>
  onChange: (value: TerminalSettings['themeId']) => void
  value: TerminalSettings['themeId']
}) {
  const { t } = useTranslation()
  return (
    <>
      <select
        aria-label={t('settings.terminalTheme')}
        className="sr-only"
        value={value}
        onChange={(event) => {
          hapticSelection()
          onChange(event.currentTarget.value as TerminalSettings['themeId'])
        }}
      >
        <optgroup label={t('settings.dark')}>
          {groups.dark.map((option) => (
            <option key={option.id} value={option.id}>{option.label}</option>
          ))}
        </optgroup>
        <optgroup label={t('settings.light')}>
          {groups.light.map((option) => (
            <option key={option.id} value={option.id}>{option.label}</option>
          ))}
        </optgroup>
      </select>
      <div
        aria-label={t('settings.themePreviews')}
        className="grid w-full min-w-0 gap-2"
        role="radiogroup"
        style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 12rem), 1fr))' }}
      >
        {[...groups.dark, ...groups.light].map((option) => (
          <ThemePreviewButton
            key={option.id}
            option={option}
            selected={option.id === value}
            onSelect={() => onChange(option.id)}
          />
        ))}
      </div>
    </>
  )
}

function ThemePreviewButton({
  onSelect,
  option,
  selected,
}: {
  onSelect: () => void
  option: TerminalThemeOption
  selected: boolean
}) {
  const theme = option.theme
  const ui = resolveTerminalThemeUi(option.id)
  const colors = [theme.red, theme.green, theme.yellow, theme.blue, theme.magenta, theme.cyan]
    .filter((color): color is string => typeof color === 'string')

  return (
    <button
      aria-checked={selected}
      className="min-w-0 border p-2 text-left transition-colors duration-200"
      role="radio"
      style={{
        backgroundColor: ui.surface,
        borderColor: selected ? ui.accent : ui.borderSubtle,
        boxShadow: selected ? `0 0 0 1px ${ui.accent}` : 'none',
      }}
      type="button"
      onClick={() => {
        hapticSelection()
        onSelect()
      }}
    >
      <div className="p-2" style={{ backgroundColor: ui.terminalBackground }}>
        <div className="mb-2 flex gap-1">
          {colors.map((color) => (
            <span key={color} className="h-2.5 flex-1" style={{ backgroundColor: color }} />
          ))}
        </div>
        <div className="space-y-1">
          <div className="h-1.5 w-4/5" style={{ backgroundColor: ui.terminalForeground, opacity: 0.72 }} />
          <div className="flex items-center gap-1">
            <div className="h-1.5 w-1/2" style={{ backgroundColor: ui.terminalForeground, opacity: 0.42 }} />
            <div className="h-2.5 w-1" style={{ backgroundColor: ui.terminalCursor }} />
          </div>
        </div>
      </div>
      <div className="mt-2 flex min-w-0 items-center gap-1.5">
        <span className="truncate text-xs font-semibold" style={{ color: ui.text }}>{option.label}</span>
        <span className="ml-auto h-2 w-2 shrink-0" style={{ backgroundColor: selected ? ui.accent : ui.borderSubtle }} />
      </div>
    </button>
  )
}

function Switch({
  ariaLabel,
  checked,
  onChange,
}: {
  ariaLabel: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <button
      aria-label={ariaLabel}
      aria-pressed={checked}
      className="flex h-11 w-12 items-center"
      type="button"
      onClick={() => {
        hapticSelection()
        onChange(!checked)
      }}
    >
      <span className={`relative block h-8 w-12 rounded-full transition-colors ${checked ? 'bg-[var(--anytty-accent)]' : 'bg-[var(--anytty-border)]'}`}>
        <span className={`absolute left-1 top-1 h-6 w-6 rounded-full bg-white shadow-sm transition-transform ${checked ? 'translate-x-4' : 'translate-x-0'}`} />
      </span>
    </button>
  )
}

function PairSheet({
  cameraScanning,
  scannerReloadRequired,
  cameraButtonRef,
  canScanWithCamera,
  pairError,
  scanFlowState,
  pairIntent,
  pairing,
  sharePreview,
  sshCredentialNotice,
  selectedMachine,
  onClose,
  onCommitShare,
  onPairPastedValue,
  onScanWithCamera,
}: {
  cameraScanning: boolean
  scannerReloadRequired: boolean
  cameraButtonRef: RefObject<HTMLButtonElement | null>
  canScanWithCamera: boolean
  pairError: string | null
  scanFlowState: ScanFlowState
  pairIntent: PairIntent
  pairing: boolean
  sharePreview: EndpointSharePreviewView | null
  sshCredentialNotice: NonNullable<ExternalPairingImportResult['sshCredentials']> | null
  selectedMachine: RemoteMachine | null
  onClose: () => void
  onCommitShare: () => void
  onPairPastedValue: (value: string) => void
  onScanWithCamera: () => void
}) {
  const { t } = useTranslation()
  const [pastedPairingInput, setPastedPairingInput] = useState('')
  const [pastedPairingError, setPastedPairingError] = useState(false)
  const title = sshCredentialNotice ? t('pairing.sshReady') : sharePreview ? t('pairing.importConfig') : pairIntent === 'add-local' ? t('pairing.addLocal') : t('pairing.authorize')
  const statusMessage = scanFlowState === 'pairing'
    ? t('pairing.scanned')
    : scanFlowState === 'scanning'
      ? t('pairing.scanning')
      : null

  const submitPastedPairingInput = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    try {
      const portablePayload = parsePastedPairingInput(pastedPairingInput)
      setPastedPairingError(false)
      onPairPastedValue(portablePayload)
    } catch {
      hapticError()
      setPastedPairingError(true)
    }
  }

  return (
    <div className="anytty-app-page fixed inset-0 z-50">
      <ModalSurface aria-label={title} className="flex h-full min-h-0 flex-col bg-white" data-testid="anytty-pair-sheet" onRequestClose={onClose}>
        <header className="anytty-app-header flex min-h-12 shrink-0 items-center justify-between gap-2 border-b pb-2 pl-[calc(env(safe-area-inset-left)+0.75rem)] pr-[calc(env(safe-area-inset-right)+0.75rem)] pt-[calc(env(safe-area-inset-top)+0.5rem)]">
          <div className="flex min-w-0 items-center gap-2">
            <QrCode className="h-5 w-5 shrink-0 text-[var(--anytty-accent)]" />
            <h2 className="truncate text-base font-semibold">{title}</h2>
          </div>
          <button
            aria-label={t('pairing.close')}
            className="anytty-app-icon-button border-transparent bg-transparent focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)]"
            type="button"
            onClick={onClose}
          >
            <X className="h-5 w-5" />
          </button>
        </header>

        <div className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto pb-[calc(env(safe-area-inset-bottom)+1.5rem)] pl-[calc(env(safe-area-inset-left)+1rem)] pr-[calc(env(safe-area-inset-right)+1rem)] pt-5">
          <div className="mx-auto w-full max-w-md">
            {sshCredentialNotice ? (
              <div className="anytty-app-panel bg-[var(--anytty-app-soft)] px-3 py-3">
                {sshCredentialNotice.map((credential) => (
                  <div className="mb-4 last:mb-0" key={credential.routeId}>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-sm font-semibold text-zinc-950">{credential.routeId}</span>
                      <span className="font-mono text-xs text-zinc-500">{credential.fingerprint}</span>
                    </div>
                    <textarea
                      aria-label={t('pairing.sshAuthorizedKey', { routeId: credential.routeId })}
                      className="mt-3 h-32 w-full resize-none border border-[var(--anytty-app-line)] bg-white p-2 font-mono text-xs leading-5 text-zinc-950 outline-none"
                      value={credential.authorizedKey}
                      readOnly
                    />
                    <button
                      className="anytty-app-secondary-button mt-3 h-11 w-full gap-2 px-3 text-sm font-semibold"
                      type="button"
                      onClick={() => { void navigator.clipboard.writeText(credential.authorizedKey); hapticSuccess() }}
                    >
                      <Copy className="h-4 w-4" />
                      {t('pairing.copyKey')}
                    </button>
                  </div>
                ))}
              </div>
            ) : sharePreview ? (
              <div className="anytty-app-panel bg-[var(--anytty-app-soft)] px-3 py-3">
                <div className="text-sm font-semibold text-zinc-950">{sharePreview.label || sharePreview.endpointId}</div>
                <div className="mt-1 break-all font-mono text-xs text-zinc-500">{sharePreview.deviceFingerprint}</div>
                <div className="mt-3 space-y-1">
                  {sharePreview.routes.map((route) => (
                    <div className="flex items-center justify-between gap-3 text-xs" key={route.id}>
                      <span className="truncate font-medium text-zinc-700">{route.id} · {route.kind}</span>
                      <span className="shrink-0 font-semibold uppercase text-[var(--anytty-app-accent)]">{route.action}</span>
                    </div>
                  ))}
                </div>
                <div className="mt-3 text-xs leading-5 text-zinc-500">
                  {sharePreview.connectModeChanged ? `${t('pairing.connectModeChanged')} ` : ''}
                  {sharePreview.selectionPolicyChanged ? `${t('pairing.selectionPolicyChanged')} ` : ''}
                  {t('pairing.credentialsStayLocal')}
                </div>
                <button
                  className="anytty-app-primary-button mt-4 h-11 w-full gap-2 px-3 text-sm font-semibold disabled:opacity-50"
                  type="button"
                  onClick={onCommitShare}
                  disabled={pairing}
                >
                  {pairing ? <span className="anytty-square-spinner" aria-hidden="true" /> : <Download className="h-4 w-4" />}
                  {t('pairing.importConfig')}
                </button>
              </div>
            ) : null}

            {!sharePreview && !sshCredentialNotice ? (
              <>
                {selectedMachine ? (
                  <div className="anytty-app-panel bg-[var(--anytty-app-soft)] px-3 py-2">
                    <div className="truncate text-sm font-semibold text-zinc-950">{selectedMachine.name}</div>
                    <div className="mt-0.5 truncate text-xs font-medium text-zinc-500">{selectedMachine.hostname || selectedMachine.id}</div>
                  </div>
                ) : null}

                {canScanWithCamera ? (
                  <button
                    ref={cameraButtonRef}
                    className="anytty-app-primary-button mt-4 h-12 w-full gap-2 px-3 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
                    type="button"
                    onClick={onScanWithCamera}
                    disabled={pairing || cameraScanning || scannerReloadRequired}
                  >
                    {cameraScanning || pairing ? <span className="anytty-square-spinner" aria-hidden="true" /> : <Camera className="h-4 w-4" />}
                    {pairing ? t('pairing.pairing') : cameraScanning ? t('pairing.scanProgress') : t('pairing.scanCamera')}
                  </button>
                ) : (
                  <p className="mt-4 border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] px-3 py-2 text-sm text-zinc-600">{t('pairing.cameraUnavailable')}</p>
                )}

                {statusMessage ? (
                  <p aria-live="polite" className="mt-3 border border-blue-500/20 bg-blue-500/10 px-3 py-2 text-sm font-medium text-blue-700" role="status">{statusMessage}</p>
                ) : null}

                {pairError ? (
                  <div className="mt-3 border border-red-500/20 bg-red-500/10 px-3 py-3 text-sm font-medium text-red-600" role="alert">
                    <p>{pairError}</p>
                    {scannerReloadRequired ? (
                      <button
                        className="anytty-app-secondary-button mt-3 min-h-[44px] w-full gap-2 px-3 text-sm font-semibold"
                        type="button"
                        onClick={() => window.location.reload()}
                      >
                        <RefreshCw className="h-4 w-4" />
                        {t('pairing.reloadApplication')}
                      </button>
                    ) : null}
                  </div>
                ) : null}

                {canScanWithCamera ? (
                  <div className="my-5 flex items-center gap-3" aria-hidden="true">
                    <span className="h-px flex-1 bg-[var(--anytty-app-line)]" />
                    <span className="text-xs font-medium text-zinc-500">{t('pairing.or')}</span>
                    <span className="h-px flex-1 bg-[var(--anytty-app-line)]" />
                  </div>
                ) : null}

                <form className={canScanWithCamera ? '' : 'mt-4'} noValidate onSubmit={submitPastedPairingInput}>
                  <label className="block text-sm font-semibold text-zinc-900" htmlFor="anytty-pasted-pairing-input">
                    {t('pairing.pasteLabel')}
                  </label>
                  <textarea
                    id="anytty-pasted-pairing-input"
                    aria-describedby={`anytty-pasted-pairing-security${pastedPairingError ? ' anytty-pasted-pairing-error' : ''}`}
                    aria-invalid={pastedPairingError}
                    autoCapitalize="none"
                    autoComplete="off"
                    autoCorrect="off"
                    className="mt-2 h-24 w-full resize-none border border-[var(--anytty-app-line)] bg-white px-3 py-2 font-mono text-base leading-6 text-zinc-950 outline-none transition-colors focus:border-[var(--anytty-app-accent)] focus:ring-2 focus:ring-blue-500/25 disabled:bg-zinc-100 disabled:text-zinc-500"
                    disabled={pairing || cameraScanning}
                    maxLength={maximumPastedPairingInputLength}
                    placeholder={t('pairing.pastePlaceholder')}
                    spellCheck={false}
                    value={pastedPairingInput}
                    onChange={(event) => {
                      setPastedPairingInput(event.target.value)
                      if (pastedPairingError) setPastedPairingError(false)
                    }}
                  />
                  <p id="anytty-pasted-pairing-security" className="mt-2 text-xs font-medium leading-5 text-zinc-500">
                    {t('pairing.pasteSecurity')}
                  </p>
                  {pastedPairingError ? (
                    <p id="anytty-pasted-pairing-error" className="mt-2 text-sm font-medium leading-5 text-red-600" role="alert">
                      {t('pairing.pasteInvalid')}
                    </p>
                  ) : null}
                  <button
                    className={`${canScanWithCamera ? 'anytty-app-secondary-button' : 'anytty-app-primary-button'} mt-3 min-h-12 w-full gap-2 px-3 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60`}
                    type="submit"
                    disabled={pairing || cameraScanning || pastedPairingInput.trim() === ''}
                  >
                    {pairing ? <span className="anytty-square-spinner" aria-hidden="true" /> : <ClipboardPaste className="h-4 w-4" />}
                    {pairing ? t('pairing.pairing') : t('pairing.pasteSubmit')}
                  </button>
                </form>
              </>
            ) : !sshCredentialNotice && pairError ? (
              <p className="mt-3 border border-red-500/20 bg-red-500/10 px-3 py-2 text-sm font-medium text-red-600" role="alert">{pairError}</p>
            ) : null}

          </div>
        </div>
      </ModalSurface>
    </div>
  )
}

function MachineRow({
  authorizationExpiresAt,
  authorizationState,
  machine,
  connectionStateSource,
  onDisconnectMachine,
  onForgetMachineAuthorization,
  onSelectMachine,
  onShowDetails,
}: {
  authorizationExpiresAt?: string | undefined
  authorizationState: MachineAuthorizationState
  machine: DisplayMachine
  connectionStateSource?: MachineRuntime['listConnectionState']
  onDisconnectMachine: (machine: DisplayMachine) => void | Promise<void>
  onForgetMachineAuthorization: (machine: DisplayMachine) => void
  onSelectMachine: (machine: DisplayMachine) => void
  onShowDetails: (machine: DisplayMachine) => void
}) {
  const { t } = useTranslation()
  const [menuOpen, setMenuOpen] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)
  const [disconnectError, setDisconnectError] = useState(false)
  const connection = useSyncExternalStore(
    connectionStateSource?.subscribe ?? noopSubscribe,
    connectionStateSource?.getSnapshot ?? getEmptyMachineConnectionSnapshot,
    connectionStateSource?.getSnapshot ?? getEmptyMachineConnectionSnapshot,
  )
  const actionLabel = authorizationState === 'ready' ? t('machines.open') : t('machines.pair')
  const subtitle = machine.hostname || t('machines.daemonHost')
  const card = machineCardProjection(machine, authorizationState, authorizationExpiresAt, connection, t)
  const DeviceIcon = machine.accessClass === 'cloud' ? Cloud : LaptopMinimal
  const canForget = Boolean(authorizationExpiresAt || authorizationState === 'ready')
  const canDisconnect = connection.phase === 'connected'
  const disconnect = async () => {
    if (!globalThis.confirm(t('machines.disconnectConfirm', { name: machine.name }))) return
    setDisconnecting(true)
    setDisconnectError(false)
    try {
      await onDisconnectMachine(machine)
      setMenuOpen(false)
    } catch {
      setDisconnectError(true)
    } finally {
      setDisconnecting(false)
    }
  }
  return (
    <div className="relative bg-white">
      <button
        aria-label={`${actionLabel} ${machine.name}`}
        className="relative grid min-h-[92px] min-w-0 w-full grid-cols-[40px_minmax(0,1fr)_20px] grid-rows-[auto_auto_auto] gap-x-3 gap-y-0.5 px-3 py-2 text-left transition-colors duration-150 hover:bg-zinc-50 active:bg-[var(--anytty-app-soft)] focus:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--anytty-app-accent)] lg:min-h-[68px] lg:grid-cols-[40px_minmax(180px,1.3fr)_minmax(160px,.8fr)_minmax(180px,1fr)_32px] lg:grid-rows-1 lg:items-center lg:gap-4 lg:px-4 lg:py-2"
        type="button"
        onClick={() => onSelectMachine(machine)}
      >
        <div className="relative col-start-1 row-start-1 flex h-10 w-10 items-center justify-center rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] text-zinc-700 lg:col-start-1">
          <DeviceIcon className="h-5 w-5" />
          <span className={`absolute bottom-0 right-0 h-2.5 w-2.5 border-2 border-white ${
            card.tone === 'online' ? 'bg-emerald-500' : card.tone === 'active' ? 'bg-blue-500' : card.tone === 'warning' ? 'bg-amber-500' : 'bg-zinc-400'
          }`} />
        </div>
        <div className="col-start-2 row-start-1 min-w-0 self-start pr-24 lg:col-start-2 lg:self-center lg:pr-0">
          <div className="truncate text-[15px] font-semibold leading-5 text-zinc-950">{machine.name}</div>
          <div className="mt-0.5 truncate text-xs font-medium text-zinc-500">{subtitle}</div>
        </div>
          <div className="col-start-2 row-start-2 flex min-w-0 items-center gap-2 text-[11px] font-semibold text-zinc-600 lg:col-start-3 lg:row-start-1">
            <AccessClassLabel accessClass={machine.accessClass} />
            {machine.accessClass === 'local_cloud' ? <ReachabilityLabel reachability={machine.reachability} /> : null}
          </div>
          <div className="contents lg:col-start-4 lg:row-start-1 lg:block lg:min-w-0">
            <span className={`absolute right-12 top-3 shrink-0 text-[11px] font-semibold lg:static lg:block ${card.statusClass}`}>{card.status}</span>
            <div className="col-start-2 row-start-3 truncate text-[12px] font-medium text-zinc-600 lg:mt-1">
            {card.detail}
            </div>
          </div>
        <ChevronRight className="col-start-3 row-span-3 row-start-1 h-4 w-4 shrink-0 self-center text-zinc-400 lg:hidden" />
      </button>
      <div className="absolute right-9 top-2.5 z-10 lg:right-3 lg:top-1/2 lg:-translate-y-1/2">
          <button
            aria-label={t('machines.more', { name: machine.name })}
            className="inline-flex h-11 w-11 items-center justify-center text-zinc-500 hover:bg-zinc-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] lg:h-10 lg:w-10"
            type="button"
            onClick={() => setMenuOpen((open) => !open)}
          >
            <MoreHorizontal className="h-4 w-4" />
          </button>
          {menuOpen ? (
            <div className="absolute right-0 top-11 min-w-44 border border-[var(--anytty-app-line)] bg-white p-1 shadow-lg">
              <button className="flex h-11 w-full items-center gap-2 px-2.5 text-left text-xs font-semibold text-zinc-700 hover:bg-zinc-50" type="button" onClick={() => { setMenuOpen(false); onShowDetails(machine) }}>
                <Info className="h-4 w-4" />
                {t('machines.details')}
              </button>
              {canDisconnect ? (
                <button
                  aria-label={t('machines.disconnectFrom', { name: machine.name })}
                  className="flex h-11 w-full items-center gap-2 border-t border-[var(--anytty-app-line)] px-2.5 text-left text-xs font-semibold text-red-600 hover:bg-red-50 disabled:opacity-50"
                  disabled={disconnecting}
                  type="button"
                  onClick={() => { void disconnect() }}
                >
                  {disconnecting ? <span className="anytty-square-spinner h-4 w-4" aria-hidden="true" /> : <Unplug className="h-4 w-4" />}
                  {t(disconnecting ? 'machines.disconnecting' : 'machines.disconnect')}
                </button>
              ) : null}
              {disconnectError ? <p className="px-2.5 py-2 text-xs font-medium text-red-600" role="alert">{t('machines.disconnectFailed')}</p> : null}
              {canForget ? (
                <button
                  aria-label={t('machines.removeAuthorization')}
                  className="flex h-11 w-full items-center gap-2 px-2.5 text-left text-xs font-semibold text-red-600 hover:bg-red-50"
                  type="button"
                  onClick={() => onForgetMachineAuthorization(machine)}
                >
                  <Trash2 className="h-4 w-4" />
                  {t('machines.removeAuthorization')}
                </button>
              ) : null}
            </div>
          ) : null}
        </div>
    </div>
  )
}

function DisplayMachineDetailSheet({ machine, onClose }: { machine: DisplayMachine; onClose: () => void }) {
  const { t } = useTranslation()
  useNativeBackHandler(onClose, NATIVE_BACK_PRIORITY.NESTED_OVERLAY)
  const source = machine.accessClass === 'cloud'
    ? t('machines.source.hub')
    : machine.accessClass === 'local_cloud'
      ? t('machines.source.localCloud')
      : t('machines.source.local')
  const fields = [
    [t('machines.fields.id'), machine.id],
    [t('machines.fields.hostname'), machine.hostname || '-'],
    [t('machines.fields.platform'), machine.osInfo || '-'],
    [t('machines.fields.source'), source],
    [t('machines.fields.hub'), machine.hubId || '-'],
    [t('machines.fields.lastOnline'), machine.lastSeen ? formatAuthorizationExpiry(machine.lastSeen) : '-'],
  ] as const
  return (
    <div className="fixed inset-0 z-40 flex items-end bg-black/40 md:items-center md:justify-center" onClick={onClose}>
      <ModalSurface className="max-h-[85dvh] w-full overflow-hidden border-t border-[var(--anytty-app-line)] bg-white md:max-w-md md:border" aria-labelledby="anytty-device-details-title" onRequestClose={onClose} onClick={(event) => event.stopPropagation()}>
        <header className="flex min-h-16 items-center justify-between gap-3 border-b border-[var(--anytty-app-line)] px-4">
          <div className="min-w-0">
            <h2 className="truncate text-base font-semibold text-zinc-950" id="anytty-device-details-title">{machine.name}</h2>
            <p className="mt-0.5 text-xs text-zinc-500">{t('machines.details')}</p>
          </div>
          <button aria-label={t('machines.closeDetails')} className="anytty-app-icon-button border-transparent bg-transparent" type="button" onClick={onClose}><X className="h-5 w-5" /></button>
        </header>
        <dl className="max-h-[calc(85dvh-4rem)] overflow-y-auto p-4 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
          {fields.map(([label, value]) => (
            <div className="border-b border-[var(--anytty-app-line)] py-3 last:border-b-0" key={label}>
              <dt className="text-xs font-semibold text-zinc-500">{label}</dt>
              <dd className="mt-1 break-all font-mono text-sm text-zinc-950">{value}</dd>
            </div>
          ))}
        </dl>
      </ModalSurface>
    </div>
  )
}

function authorizationAvailabilityText(
  machine: DisplayMachine,
  authorizationState: MachineAuthorizationState,
  expiresAt: string | undefined,
  t: TFunction,
): string {
  if (authorizationState === 'ready') {
    return expiresAt ? t('machines.authorizedUntil', { time: formatAuthorizationExpiry(expiresAt) }) : machine.online ? t('machines.tapToConnect') : t('machines.authorized')
  }
  if (authorizationState === 'expired') {
    return expiresAt ? t('machines.authorizationExpiredAt', { time: formatAuthorizationExpiry(expiresAt) }) : t('machines.authorizationExpired')
  }
  return t('machines.pairToOpen')
}

interface MachineCardProjection {
  status: string
  statusClass: string
  tone: 'online' | 'active' | 'warning' | 'offline'
  detail: string
}

function machineCardProjection(
  machine: DisplayMachine,
  authorizationState: MachineAuthorizationState,
  expiresAt: string | undefined,
  connection: MachineConnectionSnapshot,
  t: TFunction,
): MachineCardProjection {
  if (authorizationState !== 'ready') {
    return {
      status: authorizationState === 'expired' ? t('machines.authorizationExpired') : t('machines.actionRequired'),
      statusClass: 'text-amber-700',
      tone: 'warning',
      detail: authorizationAvailabilityText(machine, authorizationState, expiresAt, t),
    }
  }
  if (connection.phase === 'connected') {
    const path = connectionPathDetail(connection, t)
    return {
      status: t('machines.connected'),
      statusClass: 'text-emerald-700',
      tone: 'online',
      detail: joinCardDetail(terminalCountLabel(machine.terminalCount, t), path),
    }
  }
  if (connection.phase === 'failed') {
    return {
      status: t('machines.failed'),
      statusClass: 'text-red-600',
      tone: 'warning',
      detail: connectionErrorDisplayMessage(connection.error || connection.statusText || 'connection failed', t),
    }
  }
  if (connection.phase !== 'idle') {
    return {
      status: connectionPhaseShortLabel(connection.phase, t),
      statusClass: 'text-blue-700',
      tone: 'active',
      detail: t('machines.connectionInProgress'),
    }
  }
  if (machine.online) {
    return {
      status: t('machines.available'),
      statusClass: 'text-emerald-700',
      tone: 'online',
      detail: joinCardDetail(terminalCountLabel(machine.terminalCount, t), availablePathLabel(machine, t)),
    }
  }
  return {
    status: t('machines.offline'),
    statusClass: 'text-zinc-500',
    tone: 'offline',
    detail: machine.lastSeen ? t('machines.lastOnline', { time: formatAuthorizationExpiry(machine.lastSeen) }) : t('machines.notReachable'),
  }
}

function AccessClassLabel({ accessClass }: { accessClass: MachineAccessClass }) {
  const { t } = useTranslation()
  if (accessClass === 'local_cloud') {
    return <span className="inline-flex items-center gap-1.5"><Wifi className="h-3.5 w-3.5" />{t('machines.source.localCloud')}</span>
  }
  if (accessClass === 'cloud') {
    return <span className="inline-flex items-center gap-1.5"><Cloud className="h-3.5 w-3.5" />{t('machines.source.hub')}</span>
  }
  return <span className="inline-flex items-center gap-1.5"><Wifi className="h-3.5 w-3.5" />{t('machines.source.local')}</span>
}

function ReachabilityLabel({ reachability }: { reachability?: MachineReachabilityView | undefined }) {
  const { t } = useTranslation()
  const local = reachability?.localChecked ? (reachability.localOnline ? t('machines.reachability.localOnline') : t('machines.reachability.localOffline')) : t('machines.reachability.localChecking')
  const cloud = reachability?.hubOnline ? t('machines.reachability.cloudOnline') : t('machines.reachability.cloudOffline')
  return <span className="truncate font-medium text-zinc-400">{local} · {cloud}</span>
}

function connectionPathDetail(connection: MachineConnectionSnapshot, t: TFunction): string {
  const info = connection.connectionInfo
  const rtt = info?.rtt !== undefined ? `${Math.round(info.rtt)} ms` : ''
  let path = t('machines.connected')
  if (info?.path === 'local') path = t('machines.path.local')
  else if (info?.observedPath === 'single_relay' || info?.relayInUse) path = t('machines.path.relay')
  else if (info?.observedPath === 'direct') path = t('machines.path.direct')
  else if (info?.path === 'hub') path = t('machines.path.cloud')
  return joinCardDetail(path, rtt)
}

function availablePathLabel(machine: DisplayMachine, t: TFunction): string {
  if (machine.accessClass === 'local_cloud') {
    if (machine.reachability?.localOnline) return t('machines.path.localAvailable')
    if (machine.reachability?.hubOnline) return t('machines.path.cloudAvailable')
  }
  if (machine.accessClass === 'cloud') return t('machines.path.cloudAvailable')
  return t('machines.path.localAvailable')
}

function userFacingMachineName(machineId: string, name: string, hostname: string | undefined, t: TFunction): string {
  const candidate = name.trim()
  if (candidate !== '' && candidate !== machineId) return candidate
  const host = hostname?.trim()
  return host || t('machines.daemonHost')
}

function terminalCountLabel(count: number | undefined, t: TFunction): string {
  if (count === undefined) return ''
  return t('machines.terminals', { count })
}

function joinCardDetail(...parts: string[]): string {
  return parts.filter(Boolean).join(' · ')
}

function connectionPhaseShortLabel(phase: MachineConnectionSnapshot['phase'], t: TFunction): string {
  if (phase === 'probing') return t('machines.phase.probing')
  if (phase === 'resolving') return t('machines.phase.resolving')
  if (phase === 'signaling') return t('machines.phase.signaling')
  if (phase === 'authorizing') return t('machines.phase.authorizing')
  if (phase === 'verifying') return t('machines.phase.verifying')
  if (phase === 'reconnecting') return t('machines.phase.reconnecting')
  if (phase === 'waiting_network') return t('machines.phase.waiting_network')
  return t('machines.phase.connecting')
}

function formatAuthorizationExpiry(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  if (date.getFullYear() >= 2099) return value
  return date.toLocaleString(anyttyIntlLocale(), {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function machineAuthorizationState(
  machine: DisplayMachine,
  authorizedMachineIds: Set<string>,
  authorizationExpiries: Map<string, string>,
): MachineAuthorizationState {
  if (authorizedMachineIds.has(machine.id)) return 'ready'
  const expiresAt = authorizationExpiries.get(machine.id)
  if (expiresAt && isExpiredAuthorization(expiresAt)) return 'expired'
  return 'unauthorized'
}

function isExpiredAuthorization(value: string): boolean {
  const date = new Date(value)
  return !Number.isNaN(date.getTime()) && date.getTime() <= Date.now()
}

function readAuthorizedMachineIds(
  storage: RemoteRuntimeStorage | undefined,
  userId: string | undefined,
  externalPairingAdapter?: ExternalPairingAdapter,
): Set<string> {
  void userId
  if (!storage || !externalPairingAdapter) return new Set()
  try {
    return new Set(createMachineStore({ storage }).listMachines()
      .filter((machine) => {
        if (!externalPairingAdapter.isAuthorized(machine.machineId)) return false
        const expiresAt = externalPairingAdapter.authorizationExpiresAt?.(machine.machineId)
        return !expiresAt || !isExpiredAuthorization(expiresAt)
      })
      .map((machine) => machine.machineId))
  } catch {
    return new Set()
  }
}

function readAuthorizationExpiries(
  storage: RemoteRuntimeStorage | undefined,
  externalPairingAdapter?: ExternalPairingAdapter,
): Map<string, string> {
  if (!storage || !externalPairingAdapter) return new Map()
  try {
    const expiries = new Map<string, string>()
    for (const machine of createMachineStore({ storage }).listMachines()) {
      const externalExpiry = externalPairingAdapter.authorizationExpiresAt?.(machine.machineId)
      if (externalExpiry) expiries.set(machine.machineId, externalExpiry)
    }
    return expiries
  } catch {
    return new Map()
  }
}

function buildLocalHubReachabilityTargets(
  localMachines: StoredMachineRecord[],
): LocalHubReachabilityTarget[] {
  const targets = new Map<string, string[]>()
  for (const machine of localMachines) {
    const urls = compactHubUrls([
      ...localHubUrlsFromStoredMachine(machine),
      ...localFallbackHubUrlsFromStoredMachine(machine),
    ])
    if (urls.length > 0) targets.set(machine.machineId, urls)
  }
  return Array.from(targets.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([machineId, urls]) => ({ machineId, urls }))
}

function pruneLocalHubReachability(
  snapshots: Map<string, LocalHubReachabilitySnapshot>,
  targets: LocalHubReachabilityTarget[],
): Map<string, LocalHubReachabilitySnapshot> {
  if (snapshots.size === 0) return snapshots
  const liveTargets = new Map(targets.map((target) => [target.machineId, target.urls.join('\n')]))
  let changed = false
  const next = new Map<string, LocalHubReachabilitySnapshot>()
  for (const [machineId, snapshot] of snapshots) {
    if (liveTargets.get(machineId) !== snapshot.urls.join('\n')) {
      changed = true
      continue
    }
    next.set(machineId, snapshot)
  }
  return changed ? next : snapshots
}

async function probeLocalHubReachability(
  fetchImpl: RemoteRuntimeFetch,
  target: LocalHubReachabilityTarget,
  parentSignal: AbortSignal,
): Promise<LocalHubReachabilitySnapshot> {
  const results = await Promise.all(target.urls.map(async (url) => {
    const online = await probeLocalHubHealth(fetchImpl, url, parentSignal)
    return { url, online }
  }))
  return {
    machineId: target.machineId,
    urls: target.urls,
    onlineUrls: results.filter((result) => result.online).map((result) => result.url),
    checkedAt: Date.now(),
  }
}

async function probeLocalHubHealth(
  fetchImpl: RemoteRuntimeFetch,
  hubUrl: string,
  parentSignal: AbortSignal,
): Promise<boolean> {
  const baseUrl = normalizeHubBaseUrlCandidate(hubUrl)
  if (!baseUrl || parentSignal.aborted) return false
  const controller = new AbortController()
  const abort = () => controller.abort(parentSignal.reason)
  parentSignal.addEventListener('abort', abort, { once: true })
  let timeout: ReturnType<typeof setTimeout> | undefined
  try {
    // 本地模式的 Hub 和 daemon 是同一个运行时；health 可达即认为本地通路在线。
    const request = fetchImpl(`${baseUrl}/api/health`, {
      method: 'GET',
      headers: { accept: 'application/json' },
      signal: controller.signal,
    }).then((response) => response.ok, () => false)
    const deadline = new Promise<boolean>((resolve) => {
      timeout = setTimeout(() => {
        controller.abort(new Error('local Hub health probe timeout'))
        resolve(false)
      }, localHubReachabilityProbeTimeoutMs)
    })
    return await Promise.race([request, deadline])
  } catch {
    return false
  } finally {
    if (timeout) clearTimeout(timeout)
    parentSignal.removeEventListener('abort', abort)
  }
}

function sameReachabilitySnapshot(
  left: LocalHubReachabilitySnapshot | undefined,
  right: LocalHubReachabilitySnapshot,
): boolean {
  return Boolean(left) &&
    left!.machineId === right.machineId &&
    left!.urls.join('\n') === right.urls.join('\n') &&
    left!.onlineUrls.join('\n') === right.onlineUrls.join('\n')
}

function localMachineOnline(
  machine: StoredMachineRecord,
  snapshot: LocalHubReachabilitySnapshot | undefined,
): boolean {
  if (snapshot) return snapshot.onlineUrls.length > 0
  return machine.state === 'online'
}

function machineReachabilityView(input: {
  hubOnline: boolean
  localOnline: boolean
  snapshot?: LocalHubReachabilitySnapshot | undefined
}): MachineReachabilityView {
  return {
    hubOnline: input.hubOnline,
    localChecked: Boolean(input.snapshot),
    localOnline: input.localOnline,
    localOnlineUrls: input.snapshot?.onlineUrls ?? [],
  }
}

function createUnavailableMachineRuntime(): MachineRuntime {
  const unavailable = async () => { throw new Error('a Proto binding machine runtime is required') }
  return { api: { getStatus: unavailable, listTerminals: unavailable }, connector: { connect: unavailable } }
}

const unavailableNetworkRuntime: RemoteNetworkRuntime = {
  fetch() {
    throw new Error('remote network runtime is required')
  },
  queryParam() {
    return null
  },
}

function hubUrlsFromStoredMachine(machine: StoredMachineRecord): string[] {
  return compactHubUrls([machine.endpoints.hub])
}

function localHubUrlsFromStoredMachine(machine: StoredMachineRecord): string[] {
  return compactHubUrls([...machine.addresses.local, ...machine.addresses.lan])
}

function localFallbackHubUrlsFromStoredMachine(
  machine: StoredMachineRecord,
  hubUrls: readonly (string | undefined)[] = [],
): string[] {
  const publicHubUrls = compactHubUrls(machine.addresses.public)
  if (machine.source !== 'hub') return publicHubUrls
  const hub = new Set(compactHubUrls([machine.endpoints.hub, ...hubUrls]))
  return publicHubUrls.filter((hubUrl) => !hub.has(hubUrl))
}

function compactHubUrls(values: readonly (string | undefined)[]): string[] {
  const out: string[] = []
  const seen = new Set<string>()
  for (const raw of values) {
    const value = normalizeHubBaseUrlCandidate(raw)
    if (!value || seen.has(value)) continue
    seen.add(value)
    out.push(value)
  }
  return out
}

function formatLastSeen(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  const diffMs = Date.now() - date.getTime()
  const relative = new Intl.RelativeTimeFormat(anyttyIntlLocale(), { numeric: 'auto' })
  const diffMinutes = Math.round(diffMs / 60_000)
  if (Math.abs(diffMinutes) < 60) return relative.format(-diffMinutes, 'minute')
  const diffHours = Math.round(diffMs / 3_600_000)
  if (Math.abs(diffHours) < 24) return relative.format(-diffHours, 'hour')
  const diffDays = Math.round(diffMs / 86_400_000)
  if (Math.abs(diffDays) < 7) return relative.format(-diffDays, 'day')
  return date.toLocaleDateString(anyttyIntlLocale(), {
    month: 'short',
    day: 'numeric',
  })
}
