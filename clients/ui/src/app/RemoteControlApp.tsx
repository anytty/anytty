import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore, type CSSProperties, type ChangeEvent, type FormEvent, type ReactNode, type RefObject, type TouchEvent as ReactTouchEvent } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { Apple, ArrowLeft, Camera, Check, ChevronDown, ChevronRight, ClipboardPaste, Cloud, Copy, Cpu, Download, ExternalLink, FileText, HardDrive, ImagePlus, Laptop, LoaderCircle, Monitor, Moon, MoreHorizontal, PanelsTopLeft, Pencil, QrCode, RefreshCw, Router, Save, Server, Settings, Smartphone, SquareTerminal, Sun, Tablet, Undo2, Upload, WifiOff, X, type LucideIcon } from 'lucide-react'
import type { MachineWorkspaceInventoryApi, MachineWorkspaceConnector, MachineWorkspaceSwitcherMachine, SystemClipboard } from './MachineWorkspace'
import { createMachineStore, type StoredMachineRecord } from '../state/machineStore'
import type { MachineConnectionSnapshot } from '../connection/machineConnectionSnapshot'
import { FileTransferPanel } from '../files/FileTransferPanel'
import { hapticError, hapticImpact, hapticSelection, hapticSuccess } from '../platform/haptics'
import { NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import { useNativeBackHandler } from '../platform/useNativeBackHandler'
import type { FileTransferContext, TransferInfo } from '../files/fileApi'
import type { ConnectionInfo, ConnectionPolicy, ConnectionPolicyState, MachineConnectionStateEvents, RemoteNetworkRuntime, RemoteRuntimeFetch, RemoteRuntimeStorage, TerminalInventoryEvents } from '../core/transport'
import { normalizeHubBaseUrlCandidate } from '../api/hubUrl'
import type { RemoteMachine } from '../core/remoteMachine'
import {
  TERMINAL_FONT_OPTIONS,
  TERMINAL_SCROLL_INERTIA_MAX,
  TERMINAL_SCROLL_INERTIA_MIN,
  TERMINAL_THEME_OPTIONS,
  readTerminalSettings,
  resolveTerminalThemeOption,
  resolveTerminalThemeUi,
  terminalThemeCssVariables,
  writeTerminalSettings,
  type TerminalKeyboardMode,
  type TerminalScrollInertia,
  type TerminalSettings,
  type TerminalThemeOption,
} from '../terminal/terminalSettings'
import { ScrollInertiaPreview } from '../terminal/ScrollInertiaPreview'
import { appThemeCssVariables, readAppTheme, writeAppTheme, type AppTheme } from './appTheme'
import type { TerminalRenderer } from '../terminal/Terminal'
import type { MachineAccessClass } from '../state/appMachine'
import { MACHINE_ICON_NAMES, type MachineIconName } from '../state/appMachine'
import { anyttyIntlLocale, anyttyLanguages, normalizeAnyTTYLanguage } from '../i18n'
import { connectionErrorDisplayMessage } from '../connection/connectionErrorPresentation'
import { connectionPhaseLabel } from '../connection/connectionState'
import { ConnectionSummary } from '../connection/ConnectionSummary'
import { projectConnectionPresentation, type ConnectionPresentation, type ConnectionReachability } from '../connection/connectionPresentation'
import { appConnectionIsReady, type AppConnectionState } from '../connection/appConnectionState'
import { ConnectionRecoveryOverlayHost, ConnectionRecoveryOverlayProvider, type ConnectionRecoveryOverlayIntent } from '../connection/ConnectionRecoveryOverlay'
import { cloudPresenceEdgeLabel, cloudPresenceReachability, type CloudPresenceInput, type CloudPresenceReachability, type CloudPresenceSnapshot } from '../connection/cloudPresence'
import { RemoteNetworkStateManager, type NativeNetworkStatusPlugin } from '../connection/remoteNetworkState'
import { ModalSurface } from '../ui/ModalSurface'
import { Button } from '../ui/button'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '../ui/dialog'
import { Input } from '../ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { Spinner } from '../ui/spinner'
import { Textarea } from '../ui/textarea'
import { maximumPastedPairingInputLength, parsePastedPairingInput } from './pairingInput'
import { MachineAvailabilitySummary } from './MachineAvailabilitySummary'
import { PrivacyPolicyContent } from '../legal/PrivacyPolicyContent'

function noopSubscribe(_listener: () => void): () => void { return () => {} }

const LazyMachineActionsMenu = lazy(() => import('./MachineActionsMenu').then((module) => ({ default: module.MachineActionsMenu })))
const LazyMachineWorkspace = lazy(() => import('./MachineWorkspace').then((module) => ({ default: module.MachineWorkspace })))
const LazyConnectionInfoDialog = lazy(() => import('./MachineWorkspace').then((module) => ({ default: module.ConnectionInfoDialog })))
const LazySwitch = lazy(() => import('../ui/switch').then((module) => ({ default: module.Switch })))

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
      return t('errors.relayNotInPlan')
    case 'subscription_inactive':
      return t('errors.subscriptionInactive')
    case 'relay_region_unavailable':
      return t('errors.relayRegionUnavailable')
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
  alias?: string | undefined
  icon?: MachineIconName | undefined
  iconImage?: string | undefined
  canonicalName: string
  cloudPresence?: CloudPresenceSnapshot | undefined
  reachability?: MachineReachabilityView | undefined
  accessClass: MachineAccessClass
  terminalCount?: number | undefined
}
interface MachineReachabilityView {
  cloud: ReachabilityState
  local: ReachabilityState
  localOnlineUrls: string[]
}
type ReachabilityState = CloudPresenceReachability
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
  mountElement?: HTMLElement | undefined
}

/** ExternalPairingImportResult 是平台 secure-store 导入成功后可进入共享 UI 的非秘密机器投影。 */
export interface ExternalPairingImportResult {
  machine: { id: string; name: string; hostname?: string | undefined; osInfo?: string | undefined; hubId?: string | undefined; accessClass?: MachineAccessClass | undefined }
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
  retainConnectionDemand?(): () => void
  probeConnection?(): Promise<ConnectionInfo | null | void>
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
  phoneOnline?: boolean | undefined
  directReachableMachineIds?: ReadonlySet<string> | undefined
  directCheckingMachineIds?: ReadonlySet<string> | undefined
  /** @deprecated Use directReachableMachineIds. */
  locallyDiscoveredMachineIds?: ReadonlySet<string> | undefined
  /** @deprecated Use directCheckingMachineIds. */
  locallyDiscoveringMachineIds?: ReadonlySet<string> | undefined
  cloudPresenceByMachineId?: ReadonlyMap<string, CloudPresenceInput> | undefined
  connectionState?: AppConnectionState | undefined
  onRetryConnectionRecovery?: (() => void | Promise<void>) | undefined
  singlePaneWorkspace?: boolean | undefined
  pickMachineIconImage?: (() => Promise<File | null>) | undefined
  privacyPolicyUrl?: string | undefined
  onOpenPrivacyPolicy?: (() => void | Promise<void>) | undefined
  systemClipboard?: SystemClipboard | undefined
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
  phoneOnline: phoneOnlineProp,
  directReachableMachineIds,
  directCheckingMachineIds,
  locallyDiscoveredMachineIds,
  locallyDiscoveringMachineIds,
  cloudPresenceByMachineId,
  connectionState = 'ready',
  onRetryConnectionRecovery,
  singlePaneWorkspace = false,
  pickMachineIconImage,
  privacyPolicyUrl,
  onOpenPrivacyPolicy,
  systemClipboard,
}: RemoteControlAppProps) {
  const { t } = useTranslation()
  const networkRuntime = networkRuntimeProp ?? unavailableNetworkRuntime
  const storage = storageProp ?? networkRuntime.storage
  const nativeDirectReachableMachineIds = directReachableMachineIds ?? locallyDiscoveredMachineIds
  const nativeDirectCheckingMachineIds = directCheckingMachineIds ?? locallyDiscoveringMachineIds
  const [view, setView] = useState<AppView>('home')
  const [terminalSettings, setTerminalSettings] = useState<TerminalSettings>(() => readTerminalSettings(storage))
  const [appTheme, setAppTheme] = useState<AppTheme>(() => (
    readAppTheme(storage, resolveTerminalThemeOption(terminalSettings.themeId).group)
  ))
  const [localMachines, setLocalMachines] = useState<StoredMachineRecord[]>(() => {
    return storage ? createMachineStore({ storage }).listMachines() : []
  })
  const [localHubReachability, setLocalHubReachability] = useState<Map<string, LocalHubReachabilitySnapshot>>(() => new Map())
  const [selectedMachineId, setSelectedMachineId] = useState<string | null>(null)
  const [pendingTerminalIntent, setPendingTerminalIntent] = useState<{ machineId: string; terminalId: string } | null>(null)
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
  const phoneOnline = phoneOnlineProp ?? remoteNetworkState.phoneOnline
  const effectiveConnectionState = connectionState === 'ready' && phoneOnlineProp === undefined && !remoteNetworkState.networkReady
    ? 'checking'
    : connectionState
  const effectiveConnectionReady = appConnectionIsReady(effectiveConnectionState)
  const appConnectionOverlayIntent = useMemo<ConnectionRecoveryOverlayIntent | null>(() => {
    if (!phoneOnline) {
      return {
        kind: 'offline',
        title: t('errors.phoneOfflineTitle'),
        description: t('errors.phoneOffline'),
      }
    }
    if (effectiveConnectionState === 'failed') {
      return {
        kind: 'failed',
        title: t('machines.appRecoveryFailed'),
        description: t('machines.appRecoveryFailedCopy'),
        ...(onRetryConnectionRecovery ? {
          action: {
            label: t('workspace.connection.retry'),
            onClick: () => {
              hapticSelection()
              void onRetryConnectionRecovery()
            },
          },
        } : {}),
      }
    }
    if (effectiveConnectionState === 'recovering') {
      return {
        kind: 'recovering',
        title: t('machines.restoringApp'),
      }
    }
    return null
  }, [effectiveConnectionState, onRetryConnectionRecovery, phoneOnline, t])
  const appThemeStyle = useMemo(() => appThemeCssVariables(appTheme) as CSSProperties, [appTheme])
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
  useEffect(() => {
    const root = document.documentElement
    const previousValues = new Map<string, string>()
    for (const [name, value] of Object.entries(appThemeStyle)) {
      previousValues.set(name, root.style.getPropertyValue(name))
      root.style.setProperty(name, String(value))
    }
    return () => {
      for (const [name, value] of previousValues) {
        if (value) root.style.setProperty(name, value)
        else root.style.removeProperty(name)
      }
    }
  }, [appThemeStyle])

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

  useEffect(() => {
    if (!storage) return
    const refreshMachineMetadata = () => setLocalMachines(createMachineStore({ storage }).listMachines())
    globalThis.addEventListener('anytty:machine-metadata-changed', refreshMachineMetadata)
    return () => globalThis.removeEventListener('anytty:machine-metadata-changed', refreshMachineMetadata)
  }, [storage])

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
      const directReachable = nativeDirectReachableMachineIds?.has(local.machineId) ?? false
      const directChecking = nativeDirectCheckingMachineIds?.has(local.machineId) ?? false
      const nativeDirectManaged = nativeDirectReachableMachineIds !== undefined || nativeDirectCheckingMachineIds !== undefined
      const localOnline = directReachable || (reachability ? localMachineOnline(local, reachability) : !nativeDirectManaged && localMachineOnline(local, reachability))
      const canonicalName = userFacingMachineName(local.machineId, local.name, local.hostname, t)
      const cloudPresence = normalizeCloudPresence(cloudPresenceByMachineId?.get(local.machineId), local.machineId)
      map.set(local.machineId, {
        id: local.machineId,
        name: local.alias || canonicalName,
        canonicalName,
        ...(local.alias ? { alias: local.alias } : {}),
        ...(local.icon ? { icon: local.icon } : {}),
        ...(local.iconImage ? { iconImage: local.iconImage } : {}),
        hostname: local.hostname,
        osInfo: local.osInfo,
        hubId: local.hubId,
        lastSeen: local.lastSeenAt,
        online: localOnline,
        source: 'local',
        hubUrls: hubUrlsFromStoredMachine(local),
        localHubUrls: localHubUrlsFromStoredMachine(local),
        localFallbackHubUrls: localFallbackHubUrlsFromStoredMachine(local),
        cloudPresence,
        reachability: machineReachabilityView({
          cloud: cloudPresenceReachability(cloudPresence),
          directReachable,
          directChecking,
          nativeDirectManaged,
          hasLocalTargets: localHubReachabilityTargets.some((target) => target.machineId === local.machineId),
          localOnline,
          snapshot: reachability,
        }),
        accessClass: local.accessClass ?? 'local',
        terminalCount: local.terminalCount,
      })
    }
    return Array.from(map.values())
  }, [cloudPresenceByMachineId, localHubReachability, localHubReachabilityTargets, localMachines, nativeDirectCheckingMachineIds, nativeDirectReachableMachineIds, t])

  const selectedMachine = displayMachines.find((machine) => machine.id === selectedMachineId) ?? null
  const terminalSwitcherMachines = useMemo<MachineWorkspaceSwitcherMachine[]>(() => displayMachines
    .filter((machine) => authorizedMachineIds.has(machine.id))
    .map((machine) => ({
      machineId: machine.id,
      name: machine.name,
      state: machine.online ? 'online' : 'offline',
      ...(machine.terminalCount !== undefined ? { terminalCount: machine.terminalCount } : {}),
    })), [authorizedMachineIds, displayMachines])
  const loadMachineTerminals = useCallback(async (machineId: string) => {
    if (!authorizedMachineIds.has(machineId)) throw new Error('machine authorization is required')
    const target = displayMachines.find((machine) => machine.id === machineId)
    if (!target) throw new Error('machine is unavailable')
    const runtime = getMachineRuntime(target)
    if (!runtime) throw new Error('machine runtime is unavailable')
    return await runtime.api.listTerminals()
  }, [authorizedMachineIds, displayMachines, getMachineRuntime])
  const switchWorkspaceTerminal = useCallback((intent: { machineId: string; terminalId: string }) => {
    if (!authorizedMachineIds.has(intent.machineId)) return
    setPendingTerminalIntent(intent)
    setSelectedMachineId(intent.machineId)
    setView('machine')
    setError(null)
  }, [authorizedMachineIds])
  const clearOpenedTerminalIntent = useCallback((machineId: string, terminalId: string) => {
    setPendingTerminalIntent((current) => current?.machineId === machineId && current.terminalId === terminalId ? null : current)
  }, [])
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

  const updateMachineAlias = useCallback((machineId: string, alias: string) => {
    if (!storage) return
    const store = createMachineStore({ storage })
    const machine = store.getMachine(machineId)
    if (!machine) return
    const normalizedAlias = alias.trim()
    store.saveMachine({
      ...machine,
      alias: normalizedAlias || undefined,
      updatedAt: new Date().toISOString(),
    })
    setLocalMachines(store.listMachines())
    hapticSuccess()
  }, [storage])

  const updateMachineAppearance = useCallback((machineId: string, appearance: { icon?: MachineIconName | undefined; iconImage?: string | undefined }) => {
    if (!storage) return
    const store = createMachineStore({ storage })
    const machine = store.getMachine(machineId)
    if (!machine) return
    store.saveMachine({
      ...machine,
      icon: appearance.icon,
      iconImage: appearance.iconImage,
      updatedAt: new Date().toISOString(),
    })
    setLocalMachines(store.listMachines())
    hapticSelection()
  }, [storage])

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

  const updateAppTheme = useCallback((theme: AppTheme) => {
    setAppTheme(writeAppTheme(theme, storage))
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
    cameraScanAbortRef.current?.abort()
    closePairSheet()
  }, [closePairSheet])

  const selectMachine = useCallback((machine: DisplayMachine) => {
    hapticImpact()
    setPendingTerminalIntent(null)
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
      name: external.machine.name || existing?.name || external.machine.id,
      ...(existing?.alias ? { alias: existing.alias } : {}),
      ...(existing?.icon ? { icon: existing.icon } : {}),
      ...(existing?.iconImage ? { iconImage: existing.iconImage } : {}),
      ...((external.machine.hostname ?? existing?.hostname) ? { hostname: external.machine.hostname ?? existing?.hostname } : {}),
      ...((external.machine.osInfo ?? existing?.osInfo) ? { osInfo: external.machine.osInfo ?? existing?.osInfo } : {}),
      ...((external.machine.hubId ?? existing?.hubId) ? { hubId: external.machine.hubId ?? existing?.hubId } : {}),
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

  const scanWithCamera = useCallback(async (mountElement?: HTMLElement) => {
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
      const value = await scanPairingCode({ signal: controller.signal, mountElement })
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

  const pairPastedValue = useCallback((value: string) => {
    cameraScanAbortRef.current?.abort()
    void pairScannedValue(value)
  }, [pairScannedValue])

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

  const forgetMachineAuthorization = useCallback(async (machine: RemoteMachine) => {
    if (!storage) return
    const store = createMachineStore({ storage })
    await externalPairingAdapter?.forget(machine.id)
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
      className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex h-full min-h-0 flex-col"
      data-app-theme={appTheme}
      data-testid="anytty-web-control-remote"
      style={appThemeStyle}
    >
      <ConnectionRecoveryOverlayProvider appIntent={appConnectionOverlayIntent}>
      {view === 'settings' ? (
        <SettingsView
          appTheme={appTheme}
          error={error}
          terminalSettings={terminalSettings}
          onBack={() => { hapticSelection(); setView('home') }}
          onAppThemeChange={updateAppTheme}
          onTerminalSettingsChange={updateTerminalSettings}
          onExportDebugLogs={exportDebugLogs}
          privacyPolicyUrl={privacyPolicyUrl}
          onOpenPrivacyPolicy={onOpenPrivacyPolicy}
        />
      ) : view === 'machine' && selectedMachine ? (
        <MachineTerminalListView
          key={`${selectedMachine.id}:${pairVersion}`}
          machine={selectedMachine}
          storage={storage}
          terminalSettings={terminalSettings}
          runtime={getMachineRuntime(selectedMachine)}
          phoneOnline={phoneOnline}
          connectionState={effectiveConnectionState}
          singlePaneWorkspace={singlePaneWorkspace}
          initialTerminalId={pendingTerminalIntent?.machineId === selectedMachine.id ? pendingTerminalIntent.terminalId : undefined}
          terminalSwitcherMachines={terminalSwitcherMachines}
          loadMachineTerminals={loadMachineTerminals}
          onSwitchTerminal={switchWorkspaceTerminal}
          onInitialTerminalOpened={clearOpenedTerminalIntent}
          onBack={() => {
            hapticSelection()
            setPendingTerminalIntent(null)
            setView('home')
            setError(null)
          }}
          onNeedsReauthorization={handleMachineNeedsReauthorization}
          onTerminalSettingsChange={updateTerminalSettings}
          systemClipboard={systemClipboard}
        />
      ) : (
        <HomeView
          fileTransfer={globalFileTransfer}
          transferState={globalTransferState as { transfers: TransferInfo[]; hasActiveTransfers: boolean }}
          machines={displayMachines}
          getConnectionStateSource={(machine) => authorizedMachineIds.has(machine.id) ? getMachineRuntime(machine)?.listConnectionState : undefined}
          getMachineRuntime={(machine) => authorizedMachineIds.has(machine.id) ? getMachineRuntime(machine) : null}
          authorizedMachineIds={authorizedMachineIds}
          authorizationExpiries={authorizationExpiries}
          phoneOnline={phoneOnline}
          connectionState={effectiveConnectionState}
          onRefresh={performMachineRefresh}
          onAddLocalDevice={openAddLocalSheet}
          onOpenSettings={() => { hapticSelection(); setView('settings') }}
          onOpenTransferCenter={() => { hapticSelection(); setTransferCenterOpen(true) }}
          onForgetMachineAuthorization={forgetMachineAuthorization}
          onDisconnectMachine={disconnectMachineConnection}
          onUpdateMachineAlias={updateMachineAlias}
          onUpdateMachineAppearance={updateMachineAppearance}
          pickMachineIconImage={pickMachineIconImage}
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
          onPairPastedValue={(value) => { hapticImpact(); pairPastedValue(value) }}
          onScanWithCamera={scanWithCamera}
        />
      ) : null}
      {transferCenterOpen ? (
        <GlobalTransferCenter
          fileTransfer={globalFileTransfer}
          resolveMachineLabel={(machineId) => displayMachines.find((machine) => machine.id === machineId)?.name ?? machineId}
          onClose={() => { hapticSelection(); setTransferCenterOpen(false) }}
          onResumeTransfer={resumeGlobalTransfer}
          onResumeAllTransfers={resumeAllGlobalTransfers}
          remoteActionsDisabled={!phoneOnline || !effectiveConnectionReady}
        />
      ) : null}
      </ConnectionRecoveryOverlayProvider>
    </main>
  )
}

function GlobalTransferCenter({
  fileTransfer,
  resolveMachineLabel,
  onClose,
  onResumeTransfer,
  onResumeAllTransfers,
  remoteActionsDisabled,
}: {
  fileTransfer: FileTransferContext | undefined
  resolveMachineLabel?: ((machineId: string | undefined) => string | null | undefined) | undefined
  onClose: () => void
  onResumeTransfer?: ((id: string) => void | Promise<void>) | undefined
  onResumeAllTransfers?: (() => void | Promise<void>) | undefined
  remoteActionsDisabled: boolean
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
      remoteActionsDisabled={remoteActionsDisabled}
      onOpenFile={(id) => fileTransfer.openDownloadedFile?.(id)}
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
  connectionState,
  singlePaneWorkspace,
  initialTerminalId,
  terminalSwitcherMachines,
  loadMachineTerminals,
  onSwitchTerminal,
  onInitialTerminalOpened,
  onBack,
  onNeedsReauthorization,
  onTerminalSettingsChange,
  systemClipboard,
}: {
  machine: DisplayMachine
  storage: RemoteRuntimeStorage | undefined
  terminalSettings: TerminalSettings
  runtime: MachineRuntime | null
  phoneOnline: boolean
  connectionState: AppConnectionState
  singlePaneWorkspace: boolean
  initialTerminalId?: string | undefined
  terminalSwitcherMachines: readonly MachineWorkspaceSwitcherMachine[]
  loadMachineTerminals: (machineId: string) => Promise<import('../core/model').Terminal[]>
  onSwitchTerminal: (intent: { machineId: string; terminalId: string }) => void
  onInitialTerminalOpened: (machineId: string, terminalId: string) => void
  onBack: () => void
  onNeedsReauthorization: (machineId: string) => void
  onTerminalSettingsChange: (patch: Partial<TerminalSettings>) => void
  systemClipboard?: SystemClipboard | undefined
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
    <section className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex min-h-0 flex-1 flex-col animate-in slide-in-from-right-4 duration-200" data-testid="anytty-machine-terminal-list">
      <Suspense fallback={<MachineWorkspaceLoadingShell machine={machine} onBack={onBack} />}>
        <LazyMachineWorkspace
          key={machine.id}
          api={runtime.api}
          connector={runtime.connector}
          retainConnectionDemand={runtime.retainConnectionDemand}
          className="min-h-0 flex-1"
          cloudPresence={machine.cloudPresence}
          initialMachine={{
            machineId: machine.id,
            name: machine.name,
            state: machine.online ? 'online' : 'offline',
            ...(machine.lastSeen ? { lastSeenAt: machine.lastSeen } : {}),
          }}
          connectionStateEvents={runtime.connectionStateEvents}
          inventoryEvents={runtime.inventoryEvents}
          fileTransfer={runtime.fileTransfer}
          storage={storage}
          terminalSettings={terminalSettings}
          phoneOnline={phoneOnline}
          connectionState={connectionState}
          singlePane={singlePaneWorkspace}
          initialTerminalId={initialTerminalId}
          terminalSwitcherMachines={terminalSwitcherMachines}
          loadMachineTerminals={loadMachineTerminals}
          onSwitchTerminal={onSwitchTerminal}
          onInitialTerminalOpened={onInitialTerminalOpened}
          onNeedsReauthorization={onNeedsReauthorization}
          onTerminalSettingsChange={onTerminalSettingsChange}
          systemClipboard={systemClipboard}
          onBack={onBack}
        />
      </Suspense>
    </section>
  )
}

function MachineRuntimeHeader({ machine, onBack }: { machine: DisplayMachine; onBack: () => void }) {
  const { t } = useTranslation()
  return (
    <header className="relative z-50 border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex min-h-12 shrink-0 items-center gap-2 border-b px-3 pb-2 pt-[calc(env(safe-area-inset-top)+0.5rem)]">
      <Button
        aria-label={t('common.backToMachines')}
        size="icon"
        variant="ghost"
        onClick={() => { hapticSelection(); onBack() }}
      >
        <ArrowLeft className="h-5 w-5" />
      </Button>
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

function MachineWorkspaceLoadingShell({ machine, onBack }: { machine: DisplayMachine; onBack: () => void }) {
  const { t } = useTranslation()
  const [showStatus, setShowStatus] = useState(false)

  useEffect(() => {
    const timer = globalThis.setTimeout(() => setShowStatus(true), 300)
    return () => globalThis.clearTimeout(timer)
  }, [])

  return (
    <div className="relative flex min-h-0 flex-1 flex-col" data-testid="anytty-machine-workspace-loading">
      <MachineRuntimeHeader machine={machine} onBack={onBack} />
      <div className={`flex min-h-9 shrink-0 items-center gap-2 px-4 text-xs font-medium text-zinc-500 ${showStatus ? 'border-b border-[var(--anytty-app-line)]' : ''}`} role={showStatus ? 'status' : undefined}>
        {showStatus ? (
          <>
            <LoaderCircle aria-hidden="true" className="h-3.5 w-3.5 animate-spin motion-reduce:animate-none" />
            <span>{t('common.loading')}</span>
          </>
        ) : null}
      </div>
      <div className="min-h-0 flex-1 p-3">
        <h2 className="mb-2 px-1 text-xs font-semibold uppercase tracking-wider text-zinc-500">{t('terminal.list')}</h2>
        <div aria-hidden="true" className="h-28 rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)]" />
      </div>
      <ConnectionRecoveryOverlayHost />
    </div>
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
    <section className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] relative flex min-h-0 flex-1 flex-col animate-in slide-in-from-right-4 duration-200" data-testid="anytty-machine-terminal-list">
      <MachineRuntimeHeader machine={machine} onBack={onBack} />
      <div className="flex min-h-0 flex-1 items-center justify-center px-6 py-10">
        <div className="w-full max-w-sm text-center">
          <span className="mx-auto grid size-12 place-items-center border border-zinc-300 bg-[var(--card)] text-zinc-600" aria-hidden="true">
            <WifiOff className="h-5 w-5" />
          </span>
          <h2 className="mt-4 text-base font-semibold text-zinc-950">{t('errors.connectionProblemTitle')}</h2>
          <p className="mt-2 text-sm leading-6 text-zinc-600">{t('errors.connectionInterrupted')}</p>
          <Button className="mt-5 h-11 px-4 font-semibold" variant="secondary" onClick={onBack}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            {t('common.backToMachines')}
          </Button>
        </div>
      </div>
      <ConnectionRecoveryOverlayHost />
    </section>
  )
}

function HomeView({
  fileTransfer,
  transferState,
  machines,
  getConnectionStateSource,
  getMachineRuntime,
  authorizedMachineIds,
  authorizationExpiries,
  phoneOnline,
  connectionState,
  onAddLocalDevice,
  onDisconnectMachine,
  onForgetMachineAuthorization,
  onOpenSettings,
  onOpenTransferCenter,
  onRefresh,
  onSelectMachine,
  onUpdateMachineAlias,
  onUpdateMachineAppearance,
  pickMachineIconImage,
}: {
  fileTransfer?: FileTransferContext | undefined
  transferState: { transfers: TransferInfo[]; hasActiveTransfers: boolean }
  machines: DisplayMachine[]
  getConnectionStateSource: (machine: DisplayMachine) => MachineRuntime['listConnectionState']
  getMachineRuntime: (machine: DisplayMachine) => MachineRuntime | null
  authorizedMachineIds: Set<string>
  authorizationExpiries: Map<string, string>
  phoneOnline: boolean
  connectionState: AppConnectionState
  onAddLocalDevice: () => void
  onDisconnectMachine: (machine: DisplayMachine) => void | Promise<void>
  onForgetMachineAuthorization: (machine: DisplayMachine) => Promise<void>
  onOpenSettings: () => void
  onOpenTransferCenter: () => void
  onRefresh: () => Promise<void>
  onSelectMachine: (machine: DisplayMachine) => void
  onUpdateMachineAlias: (machineId: string, alias: string) => void
  onUpdateMachineAppearance: (machineId: string, appearance: { icon?: MachineIconName | undefined; iconImage?: string | undefined }) => void
  pickMachineIconImage?: (() => Promise<File | null>) | undefined
}) {
  const { t } = useTranslation()
  const connectionReady = appConnectionIsReady(connectionState)
  const [detailMachineId, setDetailMachineId] = useState<string | null>(null)
  const detailMachine = machines.find((machine) => machine.id === detailMachineId) ?? null
  const [connectionSettingsMachineId, setConnectionSettingsMachineId] = useState<string | null>(null)
  const connectionSettingsMachine = machines.find((machine) => machine.id === connectionSettingsMachineId) ?? null
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
    <section className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex min-h-0 flex-1 flex-col" data-testid="anytty-app-home">
      <header className="border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex min-h-12 shrink-0 items-center justify-between gap-2 border-b px-3 pb-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] lg:h-16 lg:px-6 lg:py-0">
        <div className="flex min-w-0 items-center">
          <div className="min-w-0 lg:flex lg:items-center lg:gap-3">
            <h1 className="text-lg font-semibold leading-6 lg:text-sm">{t('machines.title')}</h1>
            <p className="truncate text-xs font-medium text-zinc-500 lg:border-l lg:border-zinc-200 lg:pl-3">
            {t('machines.savedCount', { count: machines.length })}
            </p>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Button
            aria-label={t('machines.refresh')}
            size="icon"
            variant="ghost"
            disabled={refreshing || !connectionReady}
            title={t('machines.refresh')}
            onClick={() => { hapticSelection(); void runRefresh() }}
          >
            <RefreshCw className={`h-5 w-5 ${refreshing ? 'animate-spin' : ''}`} />
          </Button>
          <Button
            aria-label={t('machines.scanPairing')}
            className="min-w-11 gap-2 px-2.5 lg:px-3"
            onClick={onAddLocalDevice}
          >
            <QrCode className="h-5 w-5" />
            <span className="hidden text-xs font-semibold lg:inline">{t('machines.scanService')}</span>
          </Button>
          {fileTransfer ? (
            <Button
              aria-label={t('machines.transfers')}
              className="relative"
              size="icon"
              variant="ghost"
              onClick={onOpenTransferCenter}
            >
              <Download className="h-5 w-5" />
              {transferState.hasActiveTransfers ? <span className="absolute right-2 top-2 h-2 w-2 rounded-full bg-emerald-500" /> : null}
            </Button>
          ) : null}
          <Button
            aria-label={t('machines.openSettings')}
            size="icon"
            variant="ghost"
            onClick={onOpenSettings}
          >
            <Settings className="h-5 w-5" />
          </Button>
        </div>
      </header>

      <div className="relative flex min-h-0 flex-1 flex-col" data-testid="anytty-machine-list-content">
        {phoneOnline && machines.length === 0 && (refreshing || refreshFeedback) ? (
          <div className={`flex min-h-10 shrink-0 items-center gap-2 border-b px-4 py-2 text-xs font-medium ${refreshFeedback === 'error' ? 'border-red-200 bg-red-50 text-red-800' : 'border-zinc-200 bg-[var(--card)] text-zinc-700'}`} role="status" aria-live="polite">
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
            className="relative min-h-0 flex-1 overflow-y-auto overscroll-y-contain pb-[calc(env(safe-area-inset-bottom)+1rem)] pt-4 lg:px-8 lg:py-7"
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
            <div className="relative mx-3 w-auto max-w-7xl rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm lg:mx-auto lg:w-full lg:overflow-visible" data-testid="anytty-machine-list-panel">
              <div className="hidden grid-cols-[40px_minmax(180px,1.3fr)_minmax(220px,1fr)_32px] items-center gap-4 rounded-t-[var(--radius-lg)] border-b border-zinc-200 bg-zinc-50 px-4 py-2.5 text-[11px] font-semibold uppercase text-zinc-500 lg:grid">
                <span aria-hidden="true" />
                <span>{t('machines.columns.machine')}</span>
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
                      remoteActionsDisabled={!phoneOnline || !connectionReady}
                      onDisconnectMachine={onDisconnectMachine}
                      onForgetMachineAuthorization={onForgetMachineAuthorization}
                      onSelectMachine={onSelectMachine}
                      onShowDetails={(machine) => setDetailMachineId(machine.id)}
                      onConfigureConnection={(machine) => setConnectionSettingsMachineId(machine.id)}
                    />
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </div>
      {detailMachine ? (
        <DisplayMachineDetailSheet
          authorizationState={machineAuthorizationState(detailMachine, authorizedMachineIds, authorizationExpiries)}
          machine={detailMachine}
          connectionStateSource={getConnectionStateSource(detailMachine)}
          onClose={() => setDetailMachineId(null)}
          onUpdateAlias={(alias) => onUpdateMachineAlias(detailMachine.id, alias)}
          onUpdateAppearance={(appearance) => onUpdateMachineAppearance(detailMachine.id, appearance)}
          pickMachineIconImage={pickMachineIconImage}
        />
      ) : null}
      {connectionSettingsMachine ? (
        <MachineConnectionSettingsDialog
          machine={connectionSettingsMachine}
          runtime={getMachineRuntime(connectionSettingsMachine)}
          onClose={() => setConnectionSettingsMachineId(null)}
        />
      ) : null}
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
      <section className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm w-full max-w-md p-6" data-testid="anytty-first-use">
        <div className="flex h-10 w-10 items-center justify-center rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] text-[var(--anytty-app-accent)]">
          <Server className="h-6 w-6" />
        </div>
        <h2 className="mt-5 text-lg font-semibold text-zinc-950">{t('machines.emptyTitle')}</h2>
        <p className="mt-2 text-sm leading-6 text-zinc-600">{t('machines.emptyServiceCopy')}</p>
        <div className="mt-6 grid gap-3">
          <Button className="h-12 gap-2 px-4 font-semibold" onClick={onAddLocalDevice}>
            <QrCode className="h-4 w-4" />
            {t('machines.scanService')}
          </Button>
        </div>
      </section>
    </div>
  )
}

function SettingsView({
  appTheme,
  error,
  terminalSettings,
  onBack,
  onAppThemeChange,
  onTerminalSettingsChange,
  onExportDebugLogs,
  privacyPolicyUrl,
  onOpenPrivacyPolicy,
}: {
  appTheme: AppTheme
  error: string | null
  terminalSettings: TerminalSettings
  onBack: () => void
  onAppThemeChange: (theme: AppTheme) => void
  onTerminalSettingsChange: (patch: Partial<TerminalSettings>) => void
  onExportDebugLogs?: (() => Promise<void>) | undefined
  privacyPolicyUrl?: string | undefined
  onOpenPrivacyPolicy?: (() => void | Promise<void>) | undefined
}) {
  const { t, i18n } = useTranslation()
  const [diagnosticExportState, setDiagnosticExportState] = useState<'idle' | 'sharing' | 'ready' | 'failed'>('idle')
  const handleDiagnosticExport = async () => {
    if (!onExportDebugLogs || diagnosticExportState === 'sharing') return
    hapticImpact()
    setDiagnosticExportState('sharing')
    try {
      await onExportDebugLogs()
      setDiagnosticExportState('ready')
      hapticSuccess()
    } catch {
      setDiagnosticExportState('failed')
      hapticError()
    }
  }
  const handleNumberSetting = (key: 'fontSize', min: number, max: number) =>
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
    <section className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex min-h-0 flex-1 flex-col animate-in fade-in slide-in-from-bottom-4 duration-200" data-testid="anytty-app-settings">
      <header className="border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex min-h-12 shrink-0 items-center gap-2 border-b px-3 pb-2 pt-[calc(env(safe-area-inset-top)+0.5rem)]">
        <Button
          aria-label={t('common.backToMachines')}
          size="icon"
          variant="ghost"
          onClick={onBack}
        >
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <div className="min-w-0 flex-1">
          <h1 className="text-lg font-semibold leading-6 text-zinc-900">{t('common.settings')}</h1>
          <p className="truncate text-xs font-medium text-zinc-500">{t('settings.deviceAccess')}</p>
        </div>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5 pb-[calc(env(safe-area-inset-bottom)+1.5rem)]">
        <div className="mx-auto flex w-full max-w-xl flex-col gap-6">
          {error ? (
            <p className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm font-medium text-red-700">{error}</p>
          ) : null}

          <SettingsSection title={t('common.language')}>
            <SettingsRow label={t('settings.languageHint')}>
              <SettingsSelect
                ariaLabel={t('common.language')}
                options={anyttyLanguages.map((language) => ({ value: language.id, label: language.label }))}
                value={normalizeAnyTTYLanguage(i18n.resolvedLanguage)}
                onChange={(value) => { hapticSelection(); void i18n.changeLanguage(value) }}
              />
            </SettingsRow>
          </SettingsSection>

          <SettingsSection title={t('settings.appearance')}>
            <SettingsRow label={t('settings.interfaceTheme')} stacked>
              <AppThemePicker value={appTheme} onChange={onAppThemeChange} />
            </SettingsRow>
          </SettingsSection>

          {onExportDebugLogs ? (
            <SettingsSection title={t('settings.diagnostics')}>
              <div className="px-4 py-4">
                <p className="mb-3 text-sm leading-5 text-zinc-600" id="anytty-diagnostic-log-description">
                  {t('settings.exportLogsDescription')}
                </p>
                <Button
                  aria-describedby="anytty-diagnostic-log-description"
                  className="h-12 w-full gap-2 px-3 font-semibold"
                  disabled={diagnosticExportState === 'sharing'}
                  onClick={() => { void handleDiagnosticExport() }}
                >
                  {diagnosticExportState === 'sharing'
                    ? <LoaderCircle aria-hidden="true" className="h-4 w-4 animate-spin motion-reduce:animate-none" />
                    : <Download aria-hidden="true" className="h-4 w-4" />}
                  {diagnosticExportState === 'sharing' ? t('settings.exportingLogs') : t('settings.exportLogs')}
                </Button>
                {diagnosticExportState === 'ready' ? (
                  <p aria-live="polite" className="mt-2 text-sm font-medium text-emerald-700" role="status">
                    {t('settings.exportLogsReady')}
                  </p>
                ) : null}
                {diagnosticExportState === 'failed' ? (
                  <p aria-live="assertive" className="mt-2 text-sm font-medium text-red-700" role="alert">
                    {t('settings.exportLogsFailed')}
                  </p>
                ) : null}
              </div>
            </SettingsSection>
          ) : null}

          <SettingsSection title={t('settings.terminal')}>
            <SettingsRow label={t('settings.fontSize')}>
              <div className="inline-flex h-11 items-center overflow-hidden rounded-lg border border-[var(--anytty-app-line)] bg-[var(--card)]">
                <Button
                  aria-label={t('settings.decreaseFont')}
                  className="rounded-none text-lg font-semibold text-zinc-700 hover:bg-zinc-50 active:bg-zinc-100"
                  size="icon"
                  variant="ghost"
                  onClick={() => { hapticSelection(); onTerminalSettingsChange({ fontSize: Math.max(8, terminalSettings.fontSize - 1) }) }}
                >
                  -
                </Button>
                <Input
                  aria-label={t('settings.fontSize')}
                  className="h-11 w-12 rounded-none border-y-0 px-1 text-center font-semibold"
                  inputMode="numeric"
                  max={32}
                  min={8}
                  type="number"
                  value={terminalSettings.fontSize}
                  onChange={handleNumberSetting('fontSize', 8, 32)}
                />
                <Button
                  aria-label={t('settings.increaseFont')}
                  className="rounded-none text-lg font-semibold text-zinc-700 hover:bg-zinc-50 active:bg-zinc-100"
                  size="icon"
                  variant="ghost"
                  onClick={() => { hapticSelection(); onTerminalSettingsChange({ fontSize: Math.min(32, terminalSettings.fontSize + 1) }) }}
                >
                  +
                </Button>
              </div>
            </SettingsRow>
            <SettingsRow label={t('settings.font')} stacked>
              <FontPicker
                themeId={terminalSettings.themeId}
                value={terminalSettings.fontFamily}
                onChange={(value) => onTerminalSettingsChange({ fontFamily: value })}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.terminalTheme')} stacked>
              <TerminalThemePicker
                groups={themeGroups}
                value={terminalSettings.themeId}
                onChange={(value) => onTerminalSettingsChange({ themeId: value })}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.renderer')}>
              <SettingsSelect
                ariaLabel={t('settings.renderer')}
                options={[
                  { value: 'auto', label: t('settings.auto') },
                  { value: 'webgl', label: 'WebGL' },
                  { value: 'canvas', label: 'Canvas' },
                  { value: 'dom', label: 'DOM' },
                ]}
                value={terminalSettings.renderer}
                onChange={(value) => { hapticSelection(); onTerminalSettingsChange({ renderer: value as TerminalRenderer }) }}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.keyboard')}>
              <SettingsSelect
                ariaLabel={t('settings.keyboard')}
                options={[
                  { value: 'auto', label: t('settings.keyboardAuto') },
                  { value: 'resize', label: t('settings.keyboardResize') },
                  { value: 'shift', label: t('settings.keyboardShift') },
                ]}
                value={terminalSettings.keyboardMode}
                onChange={(value) => { hapticSelection(); onTerminalSettingsChange({ keyboardMode: value as TerminalKeyboardMode }) }}
              />
            </SettingsRow>
            <SettingsRow label={t('settings.cursorBlink')}>
              <Suspense fallback={<span aria-hidden="true" className="h-11 w-12 rounded-full bg-zinc-100" />}>
                <LazySwitch
                  aria-label={t('settings.cursorBlink')}
                  checked={terminalSettings.cursorBlink}
                  onCheckedChange={(checked) => { hapticSelection(); onTerminalSettingsChange({ cursorBlink: checked }) }}
                />
              </Suspense>
            </SettingsRow>
            <SettingsRow label={t('settings.autoAcquireResizeOwner')}>
              <Suspense fallback={<span aria-hidden="true" className="h-11 w-12 rounded-full bg-zinc-100" />}>
                <LazySwitch
                  aria-label={t('settings.autoAcquireResizeOwner')}
                  checked={terminalSettings.autoAcquireResizeOwner}
                  onCheckedChange={(checked) => { hapticSelection(); onTerminalSettingsChange({ autoAcquireResizeOwner: checked }) }}
                />
              </Suspense>
            </SettingsRow>
            <ScrollInertiaSettingsSheet
              inertia={terminalSettings.scrollInertia}
              onChange={(value) => onTerminalSettingsChange({ scrollInertia: value })}
              themeId={terminalSettings.themeId}
            />
          </SettingsSection>

          {privacyPolicyUrl ? (
            <SettingsSection title={t('settings.legal')}>
              <Dialog>
                <DialogTrigger asChild>
                  <Button className="h-12 w-full justify-start rounded-none px-4" variant="ghost">
                    <FileText aria-hidden="true" className="h-4 w-4" />
                    <span className="flex-1 text-left font-medium">{t('settings.privacyPolicy')}</span>
                    <ChevronRight aria-hidden="true" className="h-4 w-4 text-[var(--muted-foreground)]" />
                  </Button>
                </DialogTrigger>
                <DialogContent className="bottom-0 left-0 right-0 top-auto flex h-[85dvh] max-h-[85dvh] w-full max-w-none translate-x-0 translate-y-0 flex-col gap-0 overflow-hidden rounded-b-none rounded-t-xl p-0 sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:h-[min(85dvh,44rem)] sm:max-w-2xl sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-lg">
                  <DialogHeader className="shrink-0 border-b border-[var(--anytty-app-line)] px-4 pb-3 pt-4 pr-14">
                    <DialogTitle>{t('settings.privacyPolicy')}</DialogTitle>
                    <DialogDescription>{t('settings.privacyPolicySummary')}</DialogDescription>
                  </DialogHeader>
                  <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5">
                    <PrivacyPolicyContent language={normalizeAnyTTYLanguage(i18n.resolvedLanguage)} />
                  </div>
                  {onOpenPrivacyPolicy ? (
                    <div className="shrink-0 border-t border-[var(--anytty-app-line)] p-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)]">
                      <Button
                        className="h-11 w-full gap-2 font-semibold"
                        onClick={() => {
                          hapticSelection()
                          void onOpenPrivacyPolicy()
                        }}
                      >
                        <ExternalLink aria-hidden="true" className="h-4 w-4" />
                        {t('settings.openPublicPrivacyPolicy')}
                      </Button>
                    </div>
                  ) : null}
                </DialogContent>
              </Dialog>
            </SettingsSection>
          ) : null}

        </div>
      </div>
    </section>
  )
}

function SettingsSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="mb-2 px-1 text-[10px] font-semibold uppercase text-[var(--anytty-app-muted)]">{title}</h2>
      <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm overflow-hidden">
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
  onChange,
  options,
  value,
}: {
  ariaLabel: string
  onChange: (value: string) => void
  options: readonly { value: string; label: string }[]
  value: string
}) {
  return (
    <Select
      value={value}
      onValueChange={(nextValue) => {
        hapticSelection()
        onChange(nextValue)
      }}
    >
      <SelectTrigger aria-label={ariaLabel} className="max-w-[58vw] justify-end sm:max-w-xs">
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="end">
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>{option.label}</SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}

function ScrollInertiaSlider({
  onChange,
  value,
}: {
  onChange: (value: TerminalScrollInertia) => void
  value: TerminalScrollInertia
}) {
  const { t } = useTranslation()
  const valueLabel = value === TERMINAL_SCROLL_INERTIA_MIN ? t('settings.inertiaOff') : String(value)
  return (
    <div className="w-full">
      <input
        aria-label={t('settings.scrollInertia')}
        aria-valuetext={valueLabel}
        className="h-8 w-full accent-[var(--anytty-app-accent)]"
        max={TERMINAL_SCROLL_INERTIA_MAX}
        min={TERMINAL_SCROLL_INERTIA_MIN}
        step={1}
        type="range"
        value={value}
        onChange={(event) => {
          const nextValue = Number(event.currentTarget.value)
          if (nextValue === value) return
          onChange(nextValue)
        }}
        onPointerUp={() => hapticSelection()}
      />
      <div className="mt-1 flex w-full justify-between text-[11px] font-medium text-[var(--anytty-app-muted)]">
        <span>{t('settings.inertiaOff')}</span>
        <span>50</span>
        <span>{TERMINAL_SCROLL_INERTIA_MAX}</span>
      </div>
    </div>
  )
}

function ScrollInertiaSettingsSheet({
  inertia,
  onChange,
  themeId,
}: {
  inertia: TerminalScrollInertia
  onChange: (value: TerminalScrollInertia) => void
  themeId: TerminalSettings['themeId']
}) {
  const { t } = useTranslation()
  const selectedLabel = inertia === TERMINAL_SCROLL_INERTIA_MIN ? t('settings.inertiaOff') : String(inertia)
  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button className="h-12 w-full justify-start rounded-none px-4" variant="ghost">
          <SquareTerminal aria-hidden="true" className="h-4 w-4" />
          <span className="flex-1 text-left font-medium">{t('settings.scrollInertiaSettings')}</span>
          <span className="text-sm font-medium text-[var(--anytty-app-muted)]">{selectedLabel}</span>
          <ChevronRight aria-hidden="true" className="h-4 w-4 text-[var(--anytty-app-muted)]" />
        </Button>
      </DialogTrigger>
      <DialogContent className="bottom-0 left-0 right-0 top-auto flex h-[88dvh] max-h-[88dvh] w-full max-w-none translate-x-0 translate-y-0 flex-col gap-0 overflow-hidden rounded-b-none rounded-t-xl p-0 sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:h-[min(88dvh,48rem)] sm:max-w-lg sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-lg">
        <DialogHeader className="shrink-0 border-b border-[var(--anytty-app-line)] px-4 pb-3 pt-4 pr-14">
          <DialogTitle>{t('settings.scrollInertiaSettings')}</DialogTitle>
          <DialogDescription className="sr-only">{t('settings.scrollInertiaSettings')}</DialogDescription>
        </DialogHeader>
        <div className="flex min-h-0 flex-1 flex-col pb-[env(safe-area-inset-bottom)]">
          <div className="shrink-0 border-b border-[var(--anytty-app-line)] px-5 py-5">
            <div className="mb-3 flex items-center justify-between gap-4">
              <span className="text-sm font-medium text-[var(--anytty-app-text)]">{t('settings.scrollInertia')}</span>
              <span className="text-sm font-semibold text-[var(--anytty-app-accent)]">{selectedLabel}</span>
            </div>
            <ScrollInertiaSlider value={inertia} onChange={onChange} />
          </div>
          <div className="min-h-0 flex-1 p-4">
            <ScrollInertiaPreview inertia={inertia} themeId={themeId} className="h-full min-h-56" />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function AppThemePicker({
  onChange,
  value,
}: {
  onChange: (value: AppTheme) => void
  value: AppTheme
}) {
  const { t } = useTranslation()
  const options = [
    { value: 'light' as const, label: t('settings.light'), Icon: Sun },
    { value: 'dark' as const, label: t('settings.dark'), Icon: Moon },
  ]
  return (
    <div
      aria-label={t('settings.interfaceTheme')}
      className="grid w-full grid-cols-2 gap-1 rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] p-1"
      data-testid="anytty-interface-theme"
      role="radiogroup"
    >
      {options.map(({ value: optionValue, label, Icon }) => {
        const selected = optionValue === value
        return (
          <Button
            aria-checked={selected}
            className="h-11 min-w-0 rounded-md px-3"
            key={optionValue}
            role="radio"
            variant={selected ? 'default' : 'ghost'}
            onClick={() => {
              hapticSelection()
              onChange(optionValue)
            }}
          >
            <Icon aria-hidden="true" className="h-4 w-4" />
            <span className="truncate">{label}</span>
          </Button>
        )
      })}
    </div>
  )
}

function FontPicker({
  onChange,
  themeId,
  value,
}: {
  onChange: (value: string) => void
  themeId: TerminalSettings['themeId']
  value: string
}) {
  const { t } = useTranslation()
  return (
    <div
      aria-label={t('settings.fontPreviews')}
      className="grid w-full min-w-0 grid-cols-1 gap-2 sm:grid-cols-2"
      role="radiogroup"
      style={terminalThemeCssVariables(themeId)}
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
    <Button
      aria-checked={selected}
      className={`h-auto min-w-0 w-full flex-col items-stretch justify-start gap-0 whitespace-normal rounded-lg p-3 text-left transition-colors duration-200 ${
        selected
          ? 'border-[var(--primary)] bg-[var(--secondary)]'
          : 'border-[var(--input)] bg-[var(--card)] hover:bg-[var(--ui-accent)] active:bg-[var(--ui-accent)]'
      }`}
      role="radio"
      style={{ fontFamily: option.value }}
      variant="outline"
      onClick={() => {
        hapticSelection()
        onSelect()
      }}
    >
      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate text-sm font-semibold leading-5 text-zinc-950">{option.label}</span>
        <span className={`ml-auto h-2 w-2 shrink-0 rounded-full ${selected ? 'bg-[var(--anytty-app-accent)]' : 'bg-zinc-200'}`} />
      </div>
      <div className="mt-2 rounded-md bg-[var(--anytty-terminal-bg)] px-2 py-2 text-[12px] leading-5 text-[var(--anytty-terminal-fg)]">
        <div className="truncate">$ anytty --font</div>
        <div className="truncate opacity-75">AaBb 012345 &lt;&gt; ~/   </div>
      </div>
    </Button>
  )
}

function TerminalThemePicker({
  groups,
  onChange,
  value,
}: {
  groups: Record<'dark' | 'light', TerminalThemeOption[]>
  onChange: (value: TerminalSettings['themeId']) => void
  value: TerminalSettings['themeId']
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const options = [...groups.dark, ...groups.light]
  const selectedOption = options.find((option) => option.id === value) ?? options[0]!
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          aria-label={`${t('settings.terminalTheme')}: ${selectedOption.label}`}
          className="h-auto min-h-14 w-full justify-start gap-3 px-3 py-2 text-left"
          variant="outline"
          onClick={() => hapticSelection()}
        >
          <ThemeSwatch option={selectedOption} />
          <span className="min-w-0 flex-1 truncate text-sm font-semibold">{selectedOption.label}</span>
          <ChevronRight aria-hidden="true" className="h-4 w-4 shrink-0 text-zinc-400" />
        </Button>
      </DialogTrigger>
      <DialogContent className="bottom-0 left-0 right-0 top-auto flex h-[85dvh] max-h-[85dvh] w-full max-w-none translate-x-0 translate-y-0 flex-col gap-0 overflow-hidden rounded-b-none rounded-t-xl p-0 sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:h-[min(85dvh,44rem)] sm:max-w-lg sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-lg">
        <DialogHeader className="shrink-0 border-b border-[var(--anytty-app-line)] px-4 pb-3 pt-4 pr-14">
          <DialogTitle>{t('settings.terminalTheme')}</DialogTitle>
          <DialogDescription>{t('settings.terminalThemePreviews')}</DialogDescription>
        </DialogHeader>
        <div
          aria-label={t('settings.terminalThemePreviews')}
          className="min-h-0 flex-1 touch-pan-y overflow-y-auto overscroll-y-contain p-4 pb-[calc(env(safe-area-inset-bottom)+1rem)]"
          data-testid="anytty-theme-picker-scroller"
          role="radiogroup"
        >
          {(['dark', 'light'] as const).map((group) => (
            <section className="mb-5 last:mb-0" key={group}>
              <h3 className="sticky top-0 z-10 mb-2 bg-[var(--popover)] py-1 text-xs font-semibold text-[var(--muted-foreground)]">
                {t(`settings.${group}`)}
              </h3>
              <div className="grid grid-cols-2 gap-2">
                {groups[group].map((option) => (
                  <ThemePreviewButton
                    key={option.id}
                    option={option}
                    selected={option.id === value}
                    onSelect={() => {
                      onChange(option.id)
                      setOpen(false)
                    }}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function ThemeSwatch({ option }: { option: TerminalThemeOption }) {
  const ui = resolveTerminalThemeUi(option.id)
  const colors = [option.theme.red, option.theme.green, option.theme.blue]
    .filter((color): color is string => typeof color === 'string')
  return (
    <span
      aria-hidden="true"
      className="flex h-9 w-14 shrink-0 items-center justify-center gap-1 rounded-md border px-1.5"
      style={{ backgroundColor: ui.terminalBackground, borderColor: ui.borderSubtle }}
    >
      {colors.map((color) => <span className="h-2 w-2 rounded-full" key={color} style={{ backgroundColor: color }} />)}
    </span>
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
    <Button
      aria-checked={selected}
      className="h-auto min-w-0 w-full flex-col items-stretch justify-start gap-0 whitespace-normal rounded-lg p-2 text-left transition-colors duration-200"
      role="radio"
      style={{
        backgroundColor: ui.surface,
        borderColor: selected ? ui.accent : ui.borderSubtle,
        boxShadow: selected ? `0 0 0 1px ${ui.accent}` : 'none',
      }}
      variant="outline"
      onClick={() => {
        hapticSelection()
        onSelect()
      }}
    >
      <div className="rounded-md p-2" style={{ backgroundColor: ui.terminalBackground }}>
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
        <span className="ml-auto h-2 w-2 shrink-0 rounded-full" style={{ backgroundColor: selected ? ui.accent : ui.borderSubtle }} />
      </div>
    </Button>
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
  onScanWithCamera: (mountElement: HTMLElement) => void
}) {
  const { t } = useTranslation()
  const [pastedPairingInput, setPastedPairingInput] = useState('')
  const [pastedPairingError, setPastedPairingError] = useState(false)
  const [pasteOpen, setPasteOpen] = useState(!canScanWithCamera)
  const [automaticScanStarted, setAutomaticScanStarted] = useState(false)
  const cameraMountRef = useRef<HTMLDivElement>(null)
  const title = sshCredentialNotice ? t('pairing.sshReady') : sharePreview ? t('pairing.importConfig') : pairIntent === 'add-local' ? t('pairing.addLocal') : t('pairing.authorize')
  const statusMessage = scanFlowState === 'pairing' ? t('pairing.scanned') : null
  const pasteFormVisible = !canScanWithCamera || pasteOpen || Boolean(pairError)

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

  const startCameraScan = useCallback(() => {
    const mountElement = cameraMountRef.current
    if (!mountElement || pairing || cameraScanning || scannerReloadRequired) return
    setAutomaticScanStarted(true)
    onScanWithCamera(mountElement)
  }, [cameraScanning, onScanWithCamera, pairing, scannerReloadRequired])

  useEffect(() => {
    if (!canScanWithCamera || automaticScanStarted) return
    startCameraScan()
  }, [automaticScanStarted, canScanWithCamera, startCameraScan])

  return (
    <div className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] fixed inset-0 z-50">
      <ModalSurface aria-label={title} className="flex h-full min-h-0 flex-col bg-[var(--card)]" data-testid="anytty-pair-sheet" onRequestClose={onClose}>
        <header className="border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex min-h-12 shrink-0 items-center justify-between gap-2 border-b pb-2 pl-[calc(env(safe-area-inset-left)+0.75rem)] pr-[calc(env(safe-area-inset-right)+0.75rem)] pt-[calc(env(safe-area-inset-top)+0.5rem)]">
          <div className="flex min-w-0 items-center gap-2">
            <QrCode className="h-5 w-5 shrink-0 text-[var(--anytty-accent)]" />
            <h2 className="truncate text-base font-semibold">{title}</h2>
          </div>
          <Button
            aria-label={t('pairing.close')}
            size="icon"
            variant="ghost"
            onClick={onClose}
          >
            <X className="h-5 w-5" />
          </Button>
        </header>

        <div className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto pb-[calc(env(safe-area-inset-bottom)+1.5rem)] pl-[calc(env(safe-area-inset-left)+1rem)] pr-[calc(env(safe-area-inset-right)+1rem)] pt-5">
          <div className="mx-auto w-full max-w-md">
            {sshCredentialNotice ? (
              <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm bg-[var(--anytty-app-soft)] px-3 py-3">
                {sshCredentialNotice.map((credential) => (
                  <div className="mb-4 last:mb-0" key={credential.routeId}>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-sm font-semibold text-zinc-950">{credential.routeId}</span>
                      <span className="font-mono text-xs text-zinc-500">{credential.fingerprint}</span>
                    </div>
                    <Textarea
                      aria-label={t('pairing.sshAuthorizedKey', { routeId: credential.routeId })}
                      className="mt-3 h-32 resize-none p-2 font-mono text-xs leading-5 text-zinc-950"
                      value={credential.authorizedKey}
                      readOnly
                    />
                    <Button
                      className="mt-3 h-11 w-full gap-2 px-3 font-semibold"
                      variant="secondary"
                      onClick={() => { void navigator.clipboard.writeText(credential.authorizedKey); hapticSuccess() }}
                    >
                      <Copy className="h-4 w-4" />
                      {t('pairing.copyKey')}
                    </Button>
                  </div>
                ))}
              </div>
            ) : sharePreview ? (
              <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm bg-[var(--anytty-app-soft)] px-3 py-3">
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
                <Button
                  className="mt-4 h-11 w-full gap-2 px-3 font-semibold"
                  onClick={onCommitShare}
                  disabled={pairing}
                >
                  {pairing ? <Spinner aria-hidden="true" /> : <Download className="h-4 w-4" />}
                  {t('pairing.importConfig')}
                </Button>
              </div>
            ) : null}

            {!sharePreview && !sshCredentialNotice ? (
              <>
                {selectedMachine ? (
                  <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm bg-[var(--anytty-app-soft)] px-3 py-2">
                    <div className="truncate text-sm font-semibold text-zinc-950">{selectedMachine.name}</div>
                    <div className="mt-0.5 truncate text-xs font-medium text-zinc-500">{selectedMachine.hostname || selectedMachine.id}</div>
                  </div>
                ) : null}

                {pairIntent === 'authorize-machine' ? (
                  <p className="mt-3 rounded-md border border-amber-500/25 bg-amber-500/10 px-3 py-2 text-sm font-medium text-amber-800">
                    {t('errors.pairAgain')}
                  </p>
                ) : null}

                {canScanWithCamera ? (
                  <div className="relative mt-4 min-h-[336px] overflow-hidden rounded-xl border border-[#27272a] bg-[#09090b] p-2 shadow-sm" data-testid="anytty-inline-qr-scanner">
                    <div className="min-h-[320px] w-full overflow-hidden rounded-lg" ref={cameraMountRef} />
                    {automaticScanStarted && !cameraScanning && !pairing ? (
                      <Button
                        ref={cameraButtonRef}
                        aria-label={t('pairing.scanAgain')}
                        className="absolute inset-2 h-auto w-auto flex-col gap-3 rounded-lg border border-white/10 bg-[#09090b] px-6 text-center text-white hover:bg-[#18181b]"
                        variant="ghost"
                        onClick={startCameraScan}
                        disabled={scannerReloadRequired}
                      >
                        <span className="flex h-12 w-12 items-center justify-center rounded-full border border-white/15 bg-white/10 text-white">
                          <Camera className="h-5 w-5" aria-hidden="true" />
                        </span>
                        <span className="text-sm font-semibold">{t('pairing.scanAgain')}</span>
                      </Button>
                    ) : null}
                  </div>
                ) : (
                  <p className="mt-4 rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] px-3 py-2 text-sm text-zinc-600">{t('pairing.cameraUnavailable')}</p>
                )}

                {statusMessage ? (
                  <p aria-live="polite" className="mt-3 rounded-md border border-blue-500/20 bg-blue-500/10 px-3 py-2 text-sm font-medium text-blue-700" role="status">{statusMessage}</p>
                ) : null}

                {pairError ? (
                  <div className="mt-3 rounded-md border border-red-500/20 bg-red-500/10 px-3 py-3 text-sm font-medium text-red-600" role="alert">
                    <p>{pairError}</p>
                    {scannerReloadRequired ? (
                      <Button
                        className="mt-3 min-h-11 w-full gap-2 px-3 font-semibold"
                        variant="secondary"
                        onClick={() => window.location.reload()}
                      >
                        <RefreshCw className="h-4 w-4" />
                        {t('pairing.reloadApplication')}
                      </Button>
                    ) : null}
                  </div>
                ) : null}

                {canScanWithCamera ? (
                  <Button
                    aria-expanded={pasteFormVisible}
                    className="mt-4 w-full justify-between px-3 text-zinc-700"
                    variant="outline"
                    onClick={() => {
                      hapticSelection()
                      setPasteOpen((current) => !current)
                    }}
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <ClipboardPaste aria-hidden="true" className="h-4 w-4 shrink-0" />
                      <span className="truncate">{t('pairing.pasteAlternative')}</span>
                    </span>
                    <ChevronDown aria-hidden="true" className={`h-4 w-4 shrink-0 transition-transform ${pasteFormVisible ? 'rotate-180' : ''}`} />
                  </Button>
                ) : null}

                {pasteFormVisible ? <form className="mt-4" noValidate onSubmit={submitPastedPairingInput}>
                  <label className="block text-sm font-semibold text-zinc-900" htmlFor="anytty-pasted-pairing-input">
                    {t('pairing.pasteLabel')}
                  </label>
                  <Textarea
                    id="anytty-pasted-pairing-input"
                    aria-describedby={`anytty-pasted-pairing-security${pastedPairingError ? ' anytty-pasted-pairing-error' : ''}`}
                    aria-invalid={pastedPairingError}
                    autoCapitalize="none"
                    autoComplete="off"
                    autoCorrect="off"
                    className="mt-2 h-24 resize-none font-mono text-base leading-6 text-zinc-950 disabled:bg-zinc-100 disabled:text-zinc-500"
                    disabled={pairing}
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
                  <Button
                    className="mt-3 min-h-12 w-full gap-2 px-3 font-semibold"
                    variant={canScanWithCamera ? 'secondary' : 'default'}
                    type="submit"
                    disabled={pairing || pastedPairingInput.trim() === ''}
                  >
                    {pairing ? <Spinner aria-hidden="true" /> : <ClipboardPaste className="h-4 w-4" />}
                    {pairing ? t('pairing.pairing') : t('pairing.pasteSubmit')}
                  </Button>
                </form> : null}
              </>
            ) : !sshCredentialNotice && pairError ? (
              <p className="mt-3 rounded-md border border-red-500/20 bg-red-500/10 px-3 py-2 text-sm font-medium text-red-600" role="alert">{pairError}</p>
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
  remoteActionsDisabled,
  onDisconnectMachine,
  onForgetMachineAuthorization,
  onSelectMachine,
  onShowDetails,
  onConfigureConnection,
}: {
  authorizationExpiresAt?: string | undefined
  authorizationState: MachineAuthorizationState
  machine: DisplayMachine
  connectionStateSource?: MachineRuntime['listConnectionState']
  remoteActionsDisabled: boolean
  onDisconnectMachine: (machine: DisplayMachine) => void | Promise<void>
  onForgetMachineAuthorization: (machine: DisplayMachine) => Promise<void>
  onSelectMachine: (machine: DisplayMachine) => void
  onShowDetails: (machine: DisplayMachine) => void
  onConfigureConnection: (machine: DisplayMachine) => void
}) {
  const { t } = useTranslation()
  const [menuOpen, setMenuOpen] = useState(false)
  const connection = useSyncExternalStore(
    connectionStateSource?.subscribe ?? noopSubscribe,
    connectionStateSource?.getSnapshot ?? getEmptyMachineConnectionSnapshot,
    connectionStateSource?.getSnapshot ?? getEmptyMachineConnectionSnapshot,
  )
  const actionLabel = authorizationState === 'ready' ? t('machines.open') : t('machines.pair')
  const reachability = effectiveMachineReachability(machine, connection)
  const combinedReachabilityState = combinedMachineReachability(machine, reachability)
  const presentation = projectConnectionPresentation({
    phoneOnline: true,
    authAvailable: authorizationState === 'ready',
    reachability: combinedReachabilityState,
    snapshot: connection,
  })
  const summary = machineConnectionSummary(presentation, machine, reachability, authorizationState, connection, t)
  const canForget = Boolean(authorizationExpiresAt || authorizationState === 'ready')
  const canDisconnect = !remoteActionsDisabled && connection.phase === 'connected'
  const cannotOpen = authorizationState === 'ready' && remoteActionsDisabled
  return (
    <div className="relative">
      <Button
        aria-label={`${actionLabel} ${machine.name}`}
        className="relative grid h-auto min-h-[72px] min-w-0 w-full grid-cols-[40px_minmax(0,1fr)_20px] grid-rows-[auto_auto] gap-x-3 gap-y-0.5 whitespace-normal rounded-lg px-3 py-2 text-left hover:bg-zinc-50 active:bg-[var(--anytty-app-soft)] focus-visible:ring-inset lg:min-h-[60px] lg:grid-cols-[40px_minmax(180px,1.3fr)_minmax(220px,1fr)_32px] lg:grid-rows-1 lg:items-center lg:gap-4 lg:px-4 lg:py-2"
        disabled={cannotOpen}
        variant="ghost"
        onClick={() => onSelectMachine(machine)}
      >
        <MachineIcon className="col-start-1 row-span-2 row-start-1 self-center lg:col-start-1 lg:row-span-1" machine={machine} />
        <div className="col-start-2 row-start-1 min-w-0 self-end pr-12 lg:col-start-2 lg:self-center lg:pr-0">
          <div className="truncate text-[15px] font-semibold leading-5 text-zinc-950">{machine.name}</div>
        </div>
        <div className="col-start-2 row-start-2 flex min-w-0 self-start pr-12 text-[12px] font-semibold text-zinc-600 lg:col-start-3 lg:row-start-1 lg:self-center lg:pr-0">
          {authorizationState === 'ready' ? (
            <MachineAvailabilitySummary
              accessClass={machine.accessClass}
              cloud={reachability.cloud}
              cloudDetail={cloudPresenceEdgeLabel(machine.cloudPresence)}
              connection={connection}
              local={reachability.local}
            />
          ) : (
            <ConnectionSummary
              presentation={presentation}
              label={summary.label}
              detail={summary.detail}
              className="min-w-0 [&>span:first-child]:hidden"
            />
          )}
        </div>
        <ChevronRight className="col-start-3 row-span-2 row-start-1 h-4 w-4 shrink-0 self-center text-zinc-400 lg:hidden" />
      </Button>
      <div className="absolute right-9 top-2.5 z-10 lg:right-3 lg:top-1/2 lg:-translate-y-1/2">
        {menuOpen ? (
          <Suspense fallback={(
            <Button aria-label={t('machines.more', { name: machine.name })} className="text-zinc-500 lg:h-10 lg:w-10" disabled size="icon" variant="ghost">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          )}>
            <LazyMachineActionsMenu
              open
              labels={{
                trigger: t('machines.more', { name: machine.name }),
                details: t('machines.details'),
                connection: t('machines.connectionSettings'),
                disconnect: t('machines.disconnectFrom', { name: machine.name }),
                disconnecting: t('machines.disconnecting'),
                disconnectFailed: t('machines.disconnectFailed'),
                forget: t('machines.removeAuthorization'),
                forgetting: t('machines.removingAuthorization'),
                forgetFailed: t('machines.removeAuthorizationFailed'),
              }}
              canConfigure={authorizationState === 'ready'}
              canDisconnect={canDisconnect}
              canForget={canForget}
              onOpenChange={setMenuOpen}
              onShowDetails={() => onShowDetails(machine)}
              onConfigure={() => onConfigureConnection(machine)}
              onDisconnect={async () => {
                if (!globalThis.confirm(t('machines.disconnectConfirm', { name: machine.name }))) return false
                await onDisconnectMachine(machine)
                return true
              }}
              onForget={() => onForgetMachineAuthorization(machine)}
            />
          </Suspense>
        ) : (
          <Button aria-label={t('machines.more', { name: machine.name })} className="text-zinc-500 lg:h-10 lg:w-10" size="icon" variant="ghost" onClick={() => setMenuOpen(true)}>
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        )}
      </div>
    </div>
  )
}

function MachineConnectionSettingsDialog({
  machine,
  runtime,
  onClose,
}: {
  machine: DisplayMachine
  runtime: MachineRuntime | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const [info, setInfo] = useState<ConnectionInfo | null>(() => runtime?.listConnectionState?.getSnapshot().connectionInfo ?? null)
  const [policyState, setPolicyState] = useState<ConnectionPolicyState | null>(null)
  const [loading, setLoading] = useState(true)
  const [applying, setApplying] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback((probeFirst: boolean) => {
    setLoading(true)
    setError(null)
    const getPolicy = runtime?.connector.getConnectionPolicy
    void (async () => {
      try {
        const probedInfo = probeFirst ? await runtime?.probeConnection?.() : undefined
        setInfo(probedInfo ?? runtime?.listConnectionState?.getSnapshot().connectionInfo ?? null)
        if (!getPolicy) {
          setPolicyState(null)
          setError(t('workspace.connection.policyUnavailable'))
          return
        }
        setPolicyState(await getPolicy())
      } catch (err) {
        setError(connectionErrorDisplayMessage(err, t))
      } finally {
        setLoading(false)
      }
    })()
  }, [runtime, t])

  useEffect(() => {
    refresh(false)
  }, [refresh])

  const apply = useCallback(async (policy: ConnectionPolicy) => {
    const applyPolicy = runtime?.connector.applyConnectionPolicy
    if (!applyPolicy) {
      setError(t('workspace.connection.policyUnavailable'))
      return
    }
    setApplying(true)
    setError(null)
    try {
      await applyPolicy(policy)
      setPolicyState((current) => current ? { ...current, policy } : current)
      const probedInfo = await runtime.probeConnection?.()
      setInfo(probedInfo ?? runtime.listConnectionState?.getSnapshot().connectionInfo ?? null)
    } catch (err) {
      setError(connectionErrorDisplayMessage(err, t))
    } finally {
      setApplying(false)
    }
  }, [runtime, t])

  return (
    <Suspense fallback={null}>
      <LazyConnectionInfoDialog
        info={info}
        loading={loading}
        error={error}
        policyState={policyState}
        applying={applying}
        onClose={onClose}
        onRefresh={() => refresh(true)}
        onRetry={() => refresh(true)}
        onApply={(policy) => { void apply(policy) }}
        onRestoreAuto={() => { void apply({ route: 'auto', cloud: 'auto', relayTransport: 'auto' }) }}
        routeManagement={runtime?.connector.routeManagement}
        endpointId={machine.id}
        cloudPresence={machine.cloudPresence}
      />
    </Suspense>
  )
}

function DisplayMachineDetailSheet({
  authorizationState,
  machine,
  connectionStateSource,
  onClose,
  onUpdateAlias,
  onUpdateAppearance,
  pickMachineIconImage,
}: {
  authorizationState: MachineAuthorizationState
  machine: DisplayMachine
  connectionStateSource?: MachineRuntime['listConnectionState']
  onClose: () => void
  onUpdateAlias: (alias: string) => void
  onUpdateAppearance: (appearance: { icon?: MachineIconName | undefined; iconImage?: string | undefined }) => void
  pickMachineIconImage?: (() => Promise<File | null>) | undefined
}) {
  const { t } = useTranslation()
  const [editingAlias, setEditingAlias] = useState(false)
  const [editingAppearance, setEditingAppearance] = useState(false)
  const [aliasDraft, setAliasDraft] = useState(machine.alias ?? '')
  const [appearanceError, setAppearanceError] = useState<string | null>(null)
  const [processingImage, setProcessingImage] = useState(false)
  const iconImageInputRef = useRef<HTMLInputElement | null>(null)
  const processIconFile = useCallback((file: File) => {
    setAppearanceError(null)
    setProcessingImage(true)
    void import('./machineIconImage').then(async ({ compressMachineIconImage }) => compressMachineIconImage(file)).then((iconImage) => {
      onUpdateAppearance({ iconImage })
    }).catch((uploadError) => {
      const code = uploadError instanceof Error
        && (uploadError.message === 'invalid_image' || uploadError.message === 'image_too_large')
        ? uploadError.message
        : 'image_processing_failed'
      setAppearanceError(t(`machines.iconErrors.${code}`))
      hapticError()
    }).finally(() => setProcessingImage(false))
  }, [onUpdateAppearance, t])
  const connection = useSyncExternalStore(
    connectionStateSource?.subscribe ?? noopSubscribe,
    connectionStateSource?.getSnapshot ?? getEmptyMachineConnectionSnapshot,
    connectionStateSource?.getSnapshot ?? getEmptyMachineConnectionSnapshot,
  )
  useNativeBackHandler(onClose, NATIVE_BACK_PRIORITY.NESTED_OVERLAY)
  useEffect(() => {
    setAliasDraft(machine.alias ?? '')
  }, [machine.alias, machine.id])
  const source = machine.accessClass === 'cloud'
    ? t('machines.source.hub')
    : machine.accessClass === 'local_cloud'
      ? t('machines.source.localCloud')
      : t('machines.source.local')
  const hasDirectAccess = machine.accessClass !== 'cloud'
  const hasCloudAccess = machine.accessClass !== 'local'
  const reachability = effectiveMachineReachability(machine, connection)
  const directStatus = !hasDirectAccess
    ? t('machines.reachability.notConfigured')
    : reachabilityStateLabel('local', reachability.local, t)
  const cloudStatus = !hasCloudAccess
    ? t('machines.reachability.notConfigured')
    : reachabilityStateLabel('cloud', reachability.cloud, t)
  const currentConnection = connection.phase === 'idle'
    ? t('machines.notConnected')
    : connectionPhaseLabel(connection.phase, t)
  const currentPath = connection.phase === 'connected'
    ? connectionPathDetail(connection, t)
    : t('machines.notConnected')
  const presentation = projectConnectionPresentation({
    phoneOnline: true,
    authAvailable: authorizationState === 'ready',
    reachability: combinedMachineReachability(machine, reachability),
    snapshot: connection,
  })
  const summary = machineConnectionSummary(presentation, machine, reachability, authorizationState, connection, t)
  const fields = [
    { label: t('machines.fields.originalName'), value: machine.canonicalName },
    { label: t('machines.fields.endpointId'), value: machine.id, technical: true },
    ...(machine.cloudPresence?.deviceId ? [{ label: t('machines.fields.deviceId'), value: machine.cloudPresence.deviceId, technical: true }] : []),
    ...(machine.cloudPresence?.deviceFingerprint ? [{ label: t('machines.fields.deviceFingerprint'), value: machine.cloudPresence.deviceFingerprint, technical: true }] : []),
    ...(machine.cloudPresence?.daemonId ? [{ label: t('machines.fields.daemonId'), value: machine.cloudPresence.daemonId, technical: true }] : []),
    { label: t('machines.fields.hostname'), value: machine.hostname || t('machines.notReported') },
    { label: t('machines.fields.platform'), value: machinePlatformLabel(machine.osInfo) || t('machines.notReported') },
    { label: t('machines.fields.source'), value: source },
    { label: t('machines.fields.directStatus'), value: directStatus },
    { label: t('machines.fields.cloudStatus'), value: cloudStatus },
    ...(machine.cloudPresence ? cloudPresenceFields(machine.cloudPresence, t) : []),
    { label: t('machines.fields.connection'), value: currentConnection },
    { label: t('machines.fields.path'), value: currentPath },
    ...(machine.hubId ? [{ label: t('machines.fields.hub'), value: machine.hubId, technical: true }] : []),
    { label: t('machines.fields.lastOnline'), value: machine.lastSeen ? formatAuthorizationExpiry(machine.lastSeen) : t('machines.notReported') },
  ]
  const saveAlias = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    onUpdateAlias(aliasDraft)
    setEditingAlias(false)
  }
  return (
    <div className="fixed inset-0 z-[70] flex items-end bg-black/55 md:items-center md:justify-center" onClick={onClose}>
      <ModalSurface className="max-h-[85dvh] w-full overflow-hidden rounded-t-xl border-t border-[var(--anytty-app-line)] bg-[var(--card)] md:max-w-md md:rounded-xl md:border" aria-labelledby="anytty-device-details-title" onRequestClose={onClose} onClick={(event) => event.stopPropagation()}>
        <header className="flex min-h-16 items-center justify-between gap-3 border-b border-[var(--anytty-app-line)] px-4">
          <div className="flex min-w-0 items-center gap-3">
            <MachineIcon machine={machine} />
            <div className="min-w-0">
              <h2 className="truncate text-base font-semibold text-zinc-950" id="anytty-device-details-title">{machine.name}</h2>
              <p className="mt-0.5 text-xs text-zinc-500">{t('machines.details')}</p>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            <Button aria-label={t('machines.editAppearance')} size="icon" title={t('machines.editAppearance')} variant="ghost" onClick={() => setEditingAppearance((current) => !current)}><ImagePlus className="h-4 w-4" /></Button>
            <Button aria-label={t('machines.editName')} size="icon" variant="ghost" onClick={() => setEditingAlias(true)}><Pencil className="h-4 w-4" /></Button>
            <Button aria-label={t('machines.closeDetails')} size="icon" variant="ghost" onClick={onClose}><X className="h-5 w-5" /></Button>
          </div>
        </header>
        <div className="max-h-[calc(85dvh-4rem)] overflow-y-auto p-4 pb-[calc(env(safe-area-inset-bottom)+1rem)]">
          <ConnectionSummary
            presentation={presentation}
            label={summary.label}
            detail={summary.detail}
            className="mb-3 w-full rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] px-3 py-2"
          />
          {editingAppearance ? (
            <section className="border-b border-[var(--anytty-app-line)] pb-4" aria-labelledby="anytty-device-icon-title">
              <h3 className="text-xs font-semibold text-zinc-600" id="anytty-device-icon-title">{t('machines.iconTitle')}</h3>
              <div className="mt-2 grid max-h-[10rem] grid-cols-3 gap-2 overflow-y-auto overscroll-contain pr-1" role="group" aria-label={t('machines.iconOptions')}>
                <Button
                  aria-label={t('machines.icons.automatic')}
                  aria-pressed={!machine.iconImage && !machine.icon}
                  className="relative h-12 min-w-0 px-0"
                  title={t('machines.icons.automatic')}
                  variant={!machine.iconImage && !machine.icon ? 'default' : 'outline'}
                  onClick={() => { setAppearanceError(null); onUpdateAppearance({ icon: undefined }) }}
                >
                  {(() => {
                    const AutomaticIcon = machinePlatformIcon(machine.osInfo)
                    return <AutomaticIcon className="size-5" />
                  })()}
                  {!machine.iconImage && !machine.icon ? <Check className="absolute right-1 top-1 size-3" /> : null}
                </Button>
                {MACHINE_ICON_NAMES.map((icon) => {
                  const selected = !machine.iconImage && machine.icon === icon
                  const Icon = machineIconComponents[icon]
                  return (
                    <Button
                      aria-label={t(`machines.icons.${icon}`)}
                      aria-pressed={selected}
                      className="relative h-12 min-w-0 px-0"
                      key={icon}
                      title={t(`machines.icons.${icon}`)}
                      variant={selected ? 'default' : 'outline'}
                      onClick={() => { setAppearanceError(null); onUpdateAppearance({ icon }) }}
                    >
                      <Icon className="size-5" />
                      {selected ? <Check className="absolute right-1 top-1 size-3" /> : null}
                    </Button>
                  )
                })}
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                <Button className="min-h-11 flex-1 gap-2 px-3 font-semibold" disabled={processingImage} variant="secondary" onClick={() => {
                  if (!pickMachineIconImage) {
                    iconImageInputRef.current?.click()
                    return
                  }
                  setAppearanceError(null)
                  void pickMachineIconImage().then((file) => {
                    if (file) processIconFile(file)
                  }).catch(() => {
                    setAppearanceError(t('machines.iconErrors.image_processing_failed'))
                    hapticError()
                  })
                }}>
                  {processingImage ? <Spinner aria-hidden="true" /> : <Upload className="h-4 w-4" />}
                  {processingImage ? t('machines.processingIcon') : t('machines.uploadIcon')}
                </Button>
                <Input
                  accept="image/*"
                  aria-label={t('machines.uploadIcon')}
                  className="sr-only"
                  disabled={processingImage}
                  id="anytty-device-icon-image"
                  ref={iconImageInputRef}
                  type="file"
                  onChange={(event) => {
                    const file = event.currentTarget.files?.[0]
                    event.currentTarget.value = ''
                    if (!file) return
                    processIconFile(file)
                  }}
                />
                {machine.iconImage ? (
                  <Button className="min-h-11 gap-2 px-3 font-semibold" variant="outline" onClick={() => onUpdateAppearance({ icon: machine.icon })}>
                    <Undo2 className="h-4 w-4" />
                    {t('machines.removeCustomImage')}
                  </Button>
                ) : null}
              </div>
              <p className="mt-2 text-xs leading-5 text-zinc-500">{t('machines.iconHint')}</p>
              {appearanceError ? <p className="mt-2 text-xs font-medium text-red-600" role="alert">{appearanceError}</p> : null}
            </section>
          ) : null}
          {editingAlias ? (
            <form className="border-b border-[var(--anytty-app-line)] pb-4" onSubmit={saveAlias}>
              <label className="text-xs font-semibold text-zinc-600" htmlFor="anytty-device-alias">{t('machines.alias')}</label>
              <Input
                autoFocus
                className="mt-2 min-h-11 font-medium text-zinc-950"
                id="anytty-device-alias"
                maxLength={80}
                placeholder={machine.canonicalName}
                value={aliasDraft}
                onChange={(event) => setAliasDraft(event.currentTarget.value)}
              />
              <p className="mt-2 text-xs leading-5 text-zinc-500">{t('machines.aliasHint')}</p>
              <div className="mt-3 flex gap-2">
                <Button className="min-h-11 flex-1 gap-2 px-3 font-semibold" type="submit">
                  <Save className="h-4 w-4" />
                  {t('common.save')}
                </Button>
                {machine.alias ? (
                  <Button className="min-h-11 gap-2 px-3 font-semibold" variant="secondary" onClick={() => { onUpdateAlias(''); setAliasDraft(''); setEditingAlias(false) }}>
                    <Undo2 className="h-4 w-4" />
                    {t('machines.clearAlias')}
                  </Button>
                ) : null}
              </div>
            </form>
          ) : null}
          <dl>
          {fields.map(({ label, value, technical }) => (
            <div className="border-b border-[var(--anytty-app-line)] py-3 last:border-b-0" key={label}>
              <dt className="text-xs font-semibold text-zinc-500">{label}</dt>
              <dd className={`mt-1 break-all text-sm text-zinc-950 ${technical ? 'font-mono' : 'font-medium'}`}>{value}</dd>
            </div>
          ))}
          </dl>
        </div>
      </ModalSurface>
    </div>
  )
}

const machineIconComponents: Record<MachineIconName, LucideIcon> = {
  apple: Apple,
  windows: PanelsTopLeft,
  monitor: Monitor,
  laptop: Laptop,
  phone: Smartphone,
  tablet: Tablet,
  server: Server,
  storage: HardDrive,
  terminal: SquareTerminal,
  router: Router,
  cloud: Cloud,
  chip: Cpu,
}

function MachineIcon({ machine, className = '' }: { machine: Pick<DisplayMachine, 'icon' | 'iconImage' | 'name' | 'osInfo'>; className?: string | undefined }) {
  const Icon = machine.icon ? machineIconComponents[machine.icon] : machinePlatformIcon(machine.osInfo)
  return (
    <div className={`grid size-10 shrink-0 place-items-center overflow-hidden rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] text-[var(--anytty-app-muted)] ${className}`} aria-hidden="true">
      {machine.iconImage
        ? <img alt="" className="size-full object-cover" src={machine.iconImage} />
        : <Icon className="size-5" />}
    </div>
  )
}

export function machinePlatformIcon(platform?: string): LucideIcon {
  const normalized = platform?.trim().toLowerCase()
  if (normalized === 'darwin' || normalized === 'macos' || normalized === 'ios') return Apple
  if (normalized === 'windows' || normalized === 'win32') return PanelsTopLeft
  if (normalized === 'linux' || normalized === 'freebsd' || normalized === 'openbsd' || normalized === 'netbsd') return Server
  if (normalized === 'android') return Smartphone
  return Monitor
}

export function machinePlatformLabel(platform?: string): string | undefined {
  const normalized = platform?.trim().toLowerCase()
  if (!normalized) return undefined
  if (normalized === 'darwin' || normalized === 'macos') return 'macOS'
  if (normalized === 'ios') return 'iOS'
  if (normalized === 'android') return 'Android'
  if (normalized === 'windows' || normalized === 'win32') return 'Windows'
  if (normalized === 'linux') return 'Linux'
  return platform?.trim()
}

function combinedMachineReachability(
  machine: DisplayMachine,
  reachability: { local: ReachabilityState; cloud: ReachabilityState },
): ConnectionReachability {
  const states = [
    ...(machine.accessClass === 'cloud' ? [] : [reachability.local]),
    ...(machine.accessClass === 'local' ? [] : [reachability.cloud]),
  ]
  if (states.includes('online')) return 'reachable'
  if (states.includes('checking')) return 'checking'
  if (states.length > 0 && states.every((state) => state === 'offline')) return 'unreachable'
  return 'unknown'
}

function machineConnectionSummary(
  presentation: ConnectionPresentation,
  machine: DisplayMachine,
  reachability: { local: ReachabilityState; cloud: ReachabilityState },
  authorizationState: MachineAuthorizationState,
  connection: MachineConnectionSnapshot,
  t: TFunction,
): { label: string; detail?: string | undefined } {
  if (presentation.state === 'phone_offline') return { label: t('workspace.connection.phase.waiting_network') }
  if (presentation.state === 'auth_unavailable') {
    return { label: t(authorizationState === 'expired' ? 'machines.authorizationExpired' : 'machines.actionRequired') }
  }
  if (presentation.state === 'connecting' || presentation.state === 'waiting_network' || presentation.state === 'failed') {
    return { label: connectionPhaseLabel(presentation.state === 'connecting' ? 'connecting' : presentation.state, t) }
  }
  if (presentation.state === 'ready') {
    return { label: t('machines.connected'), detail: connectionPathDetail(connection, t) }
  }

  const available = [
    ...(machine.accessClass !== 'cloud' && reachability.local === 'online' ? [t('machines.path.localAvailable')] : []),
    ...(machine.accessClass !== 'local' && reachability.cloud === 'online' ? [t('machines.path.cloudAvailable')] : []),
  ]
  if (available.length > 0) return { label: available.join(' / ') }
  if (presentation.reachability === 'checking') {
    return { label: machine.accessClass === 'local' ? t('machines.reachability.localChecking') : t('machines.reachability.cloudChecking') }
  }
  if (presentation.reachability === 'unreachable') return { label: t('machines.notConnected') }
  return { label: t('machines.notConnected') }
}

export function connectionPathDetail(connection: MachineConnectionSnapshot, t: TFunction): string {
  const info = connection.connectionInfo
  const rtt = info?.rtt !== undefined ? `${Math.round(info.rtt)} ms` : ''
  let path = t('machines.connected')
  if (info?.routeKind === 'ssh') path = t('machines.path.ssh')
  else if (info?.observedPath === 'single_relay') path = t('machines.path.relay')
  else if (info?.observedPath === 'direct') path = t('machines.path.direct')
  else if (info?.routeKind === 'direct') path = t('machines.path.direct')
  else if (info?.path === 'local') path = t('machines.path.local')
  else if (info?.path === 'hub') path = t('machines.path.cloud')
  return joinCardDetail(path, rtt)
}

function effectiveMachineReachability(
  machine: DisplayMachine,
  connection: MachineConnectionSnapshot,
): { local: ReachabilityState; cloud: ReachabilityState } {
  let local = machine.reachability?.local ?? 'unknown'
  let cloud = machine.reachability?.cloud ?? 'unknown'
  if (connection.phase === 'connected') {
    if (connectionUsesCloud(connection)) cloud = 'online'
    else local = 'online'
  }
  return { local, cloud }
}

function normalizeCloudPresence(input: CloudPresenceInput | undefined, endpointId: string): CloudPresenceSnapshot | undefined {
  if (!input) return undefined
  return typeof input === 'string' ? { state: input, endpointId } : { ...input, endpointId: input.endpointId ?? endpointId }
}

function cloudPresenceFields(presence: CloudPresenceSnapshot, t: TFunction): Array<{ label: string; value: string; technical?: boolean | undefined }> {
  const edgeLabel = cloudPresenceEdgeLabel(presence)
  return [
    ...(edgeLabel || presence.edgeId ? [{ label: t('machines.fields.currentEdge'), value: edgeLabel || presence.edgeId || t('machines.notReported'), technical: !edgeLabel }] : []),
    ...(presence.edgePublicEndpoint ? [{ label: t('machines.fields.edgeEndpoint'), value: presence.edgePublicEndpoint, technical: true }] : []),
    ...(presence.edgeServerName ? [{ label: t('machines.fields.edgeServerName'), value: presence.edgeServerName, technical: true }] : []),
    ...(presence.locatorSource ? [{ label: t('machines.fields.locatorSource'), value: machineCloudLocatorSourceLabel(presence, t) }] : []),
  ]
}

function machineCloudLocatorSourceLabel(presence: CloudPresenceSnapshot, t: TFunction): string {
  if (presence.locatorSource === 'controller') {
    return presence.refreshedFromController ? t('machines.locatorSource.controllerRefreshed') : t('machines.locatorSource.controller')
  }
  if (presence.locatorSource === 'cached_edge') return t('machines.locatorSource.cachedEdge')
  return presence.locatorSource ?? t('machines.notReported')
}

function connectionUsesCloud(connection: MachineConnectionSnapshot): boolean {
  return connection.connectionInfo?.routeKind === 'cloud' || connection.connectionInfo?.path === 'hub'
}

function reachabilityStateLabel(
  kind: 'local' | 'cloud',
  state: ReachabilityState,
  t: TFunction,
): string {
  if (kind === 'local') {
    if (state === 'online') return t('machines.reachability.localOnline')
    if (state === 'offline') return t('machines.reachability.localOffline')
    if (state === 'checking') return t('machines.reachability.localChecking')
    return t('machines.reachability.localUnknown')
  }
  if (state === 'online') return t('machines.reachability.cloudOnline')
  if (state === 'offline') return t('machines.reachability.cloudOffline')
  if (state === 'checking') return t('machines.reachability.cloudChecking')
  return t('machines.reachability.cloudUnknown')
}

function userFacingMachineName(machineId: string, name: string, hostname: string | undefined, t: TFunction): string {
  const candidate = name.trim()
  if (candidate !== '' && candidate !== machineId) return candidate
  const host = hostname?.trim()
  return host || t('machines.daemonHost')
}

function joinCardDetail(...parts: string[]): string {
  return parts.filter(Boolean).join(' · ')
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
  cloud: ReachabilityState
  hasLocalTargets: boolean
  nativeDirectManaged: boolean
  directReachable: boolean
  directChecking: boolean
  localOnline: boolean
  snapshot?: LocalHubReachabilitySnapshot | undefined
}): MachineReachabilityView {
  return {
    cloud: input.cloud,
    local: input.directReachable
      ? 'online'
      : input.directChecking
      ? 'checking'
      : input.nativeDirectManaged
      ? 'unknown'
      : input.snapshot
      ? input.localOnline ? 'online' : 'offline'
      : input.hasLocalTargets ? 'checking' : 'unknown',
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
