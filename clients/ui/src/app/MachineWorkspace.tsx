import { useCallback, useEffect, useMemo, useRef, useState, useSyncExternalStore, type CSSProperties, type ReactNode, type RefObject } from 'react'
import { createPortal } from 'react-dom'
import { create } from '@bufbuild/protobuf'
import { Bookmark, BookmarkMinus, BookmarkPlus, ChevronDown, ChevronLeft, ClipboardList, Folder, FolderOpen, Info, KeyRound, Monitor, Pin, Plus, RefreshCw, Rows2, Settings2, SlidersHorizontal, SquarePen, Trash2, X } from 'lucide-react'
import { connectionPhaseLabel, connectionSnapshotFromStatus } from '../connection/connectionState'
import { connectionErrorDisplayMessage, connectionFailurePresentation, isAuthorizationConnectionError, isCancelledConnectionError, type ConnectionFailurePresentation } from '../connection/connectionErrorPresentation'
import { ConnectionNotice } from '../connection/ConnectionNotice'
import { projectConnectionPresentation } from '../connection/connectionPresentation'
import { appConnectionIsReady, type AppConnectionState } from '../connection/appConnectionState'
import { ConnectionRecoveryOverlayHost, useConnectionRecoveryOverlay, type ConnectionRecoveryOverlayIntent } from '../connection/ConnectionRecoveryOverlay'
import type { MachineConnectionSnapshot } from '../connection/machineConnectionSnapshot'
import { cloudPresenceEdgeLabel, type CloudPresenceSnapshot } from '../connection/cloudPresence'
import { ConnectionRouteManager, type ConnectionRouteManagementAdapter } from '../connection/ConnectionRouteManager'
import { FileTransferPanel } from '../files/FileTransferPanel'
import { loadFileManager, reloadAfterFileManagerLoadFailure, type FileManagerComponent } from '../files/loadFileManager'
import { createFileApi, type FileEntry } from '../files/fileApi'
import { fileEntryPath, normalizeFilePath, parentPath } from '../files/fileUtils'
import { createPersistentPathBookmarkApi, type PathBookmark } from '../files/pathBookmarks'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import { useMachineNetworkStatus } from '../machine-runtime/useMachineNetworkStatus'
import { createRemoteClipboardApi, type RemoteClipboardEntry } from '../clipboard/clipboardApi'
import { MobileTerminalKeybar } from '../terminal/MobileTerminalKeybar'
import type { TerminalModifierState } from '../terminal/mobileTerminalInput'
import { PasteConfirmDialog } from '../terminal/PasteConfirmDialog'
import { Terminal, type TerminalHandle } from '../terminal/Terminal'
import { TerminalActionToolbar, type TerminalToolbarMode } from '../terminal/TerminalActionToolbar'
import { TerminalEnvironmentEditor } from '../terminal/TerminalEnvironmentEditor'
import { TerminalFnPanel } from '../terminal/TerminalFnPanel'
import { NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import { useNativeBackHandler } from '../platform/useNativeBackHandler'
import { defaultTerminalResizeControl, terminalResizeControlOwnsResize, type TerminalResizeControl } from '../terminal/terminalClient'
import { TerminalList } from '../terminal/TerminalList'
import { pinTerminal, readTerminalOrder, reorderPinnedTerminal, sortTerminalIds, unpinTerminal, writeTerminalOrder } from '../terminal/terminalOrder'
import { createTerminalManagementApi } from '../terminal/terminalManagementApi'
import { formatCommandLine, parseCommandLine, validateEnvironmentVariables } from '../terminal/terminalCreateForm'
import { readTerminalKeyboardMode, readTerminalSettings, terminalThemeCssVariables, writeTerminalKeyboardMode, writeTerminalSettings, type TerminalKeyboardMode, type TerminalSettings } from '../terminal/terminalSettings'
import type { Machine, Terminal as RemoteTerminal } from '../core/model'
import type { ProtoClientSession } from '../core/protoClientSession'
import { openProtoEventSubscription } from '../core/protoEventSubscription'
import { ApplicationEventType, EventSubscribeCommandSchema } from '../generated/apipb/events_pb'
import type { ConnectionInfo, ConnectionPolicy, ConnectionPolicyState, LocalAgentApi, LocalCreateTerminalInput, LocalUpdateTerminalInput, MachineConnectionStateEvents, RemoteRuntimeStorage, RtcConnectOptions, RtcConnectionStateSnapshot, RtcEvent, RtcSubscription, TerminalInventoryEvents } from '../core/transport'
import { ConnectionCandidateType, ConnectionObservedPath, ConnectionRouteKind, ConnectionTransport } from '../generated/bindingpb/client_binding_pb'
import { useTerminalKeyboard } from '../terminal/useTerminalKeyboard'
import { useTranslation } from 'react-i18next'
import { ModalSurface } from '../ui/ModalSurface'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { NativeSelect } from '../ui/native-select'
import { Textarea } from '../ui/textarea'
import { RadioGroup, RadioGroupItem } from '../ui/radio-group'
import { Spinner } from '../ui/spinner'
import { WebTerminalSettingsDialog } from './WebTerminalSettingsDialog'
import { WebTerminalPickerDialog } from './WebTerminalPickerDialog'
import { WebTerminalDropOverlay, type WebPaneDropTarget } from './WebTerminalDropOverlay'
import { WebSplitDivider, WebTerminalPaneHeader, WebTerminalWorkbench } from './WebTerminalWorkbench'
import {
  PRIMARY_TERMINAL_PANE,
  removeTerminalPane,
  splitTerminalPane,
  terminalIdForPane,
  terminalPaneKey,
  terminalPaneKeys,
  updateTerminalSplitRatio,
  type TerminalPaneKey,
  type TerminalSplitNode,
} from './terminalSplitLayout'
import {
  createWebTerminalTab,
  findWebTerminalTab,
  sanitizeWebTerminalTabs,
  updateWebTerminalTab,
  type WebTerminalTabLayout,
} from './webTerminalTabs'
import '../i18n'

export interface MachineWorkspaceInventoryApi extends Pick<LocalAgentApi, 'getStatus'> {
  listTerminals(options?: Pick<RtcConnectOptions, 'forceRelay' | 'onStatus' | 'onConnectionState'>): Promise<RemoteTerminal[]>
}

export interface MachineWorkspaceSessionInput {
  machineId: string
}

export type MachineWorkspaceClientSession = ProtoClientSession

export type MachineWorkspaceConnector = {
  connect(input: MachineWorkspaceSessionInput, options?: RtcConnectOptions): Promise<MachineWorkspaceClientSession>
  reconnect?: ((options?: { forceRelay?: boolean | undefined }) => void | Promise<void>) | undefined
  getConnectionPolicy?: ((signal?: AbortSignal) => Promise<ConnectionPolicyState>) | undefined
  applyConnectionPolicy?: ((policy: ConnectionPolicy, signal?: AbortSignal) => Promise<void>) | undefined
  routeManagement?: ConnectionRouteManagementAdapter | undefined
}

export interface MachineWorkspaceSwitcherMachine {
  machineId: string
  name: string
  state?: Machine['state'] | undefined
  terminalCount?: number | undefined
}

export interface SystemClipboard {
  readText(): Promise<string>
  writeText(text: string): Promise<void>
}

const browserSystemClipboard: SystemClipboard = {
  async readText() {
    if (!navigator.clipboard?.readText) throw new Error('clipboard read is unavailable')
    return await navigator.clipboard.readText()
  },
  async writeText(text: string) {
    if (!navigator.clipboard?.writeText) throw new Error('clipboard write is unavailable')
    await navigator.clipboard.writeText(text)
  },
}

export interface MachineWorkspaceProps {
  api: MachineWorkspaceInventoryApi
  connector: MachineWorkspaceConnector
  retainConnectionDemand?: (() => () => void) | undefined
  className?: string | undefined
  initialMachine?: Machine | undefined
  inventoryEvents?: TerminalInventoryEvents | undefined
  connectionStateEvents?: MachineConnectionStateEvents | undefined
  subscribeRuntimeInventoryEvents?: boolean | undefined
  onBack?: (() => void) | undefined
  fileTransfer?: import('../files/fileApi').FileTransferContext | undefined
  terminalSettings?: TerminalSettings | undefined
  onNeedsReauthorization?: ((machineId: string) => void) | undefined
  onTerminalSettingsChange?: ((patch: Partial<TerminalSettings>) => void) | undefined
  phoneOnline?: boolean | undefined
  connectionState?: AppConnectionState | undefined
  cloudPresence?: CloudPresenceSnapshot | undefined
  singlePane?: boolean | undefined
  webLayout?: boolean | undefined
  storage?: RemoteRuntimeStorage | undefined
  initialTerminalId?: string | undefined
  onInitialTerminalOpened?: ((machineId: string, terminalId: string) => void) | undefined
  terminalSwitcherMachines?: readonly MachineWorkspaceSwitcherMachine[] | undefined
  loadMachineTerminals?: ((machineId: string) => Promise<RemoteTerminal[]>) | undefined
  onSwitchTerminal?: ((intent: { machineId: string; terminalId: string }) => void) | undefined
  systemClipboard?: SystemClipboard | undefined
}

type TerminalEditorSheet = 'create-terminal' | 'edit-terminal'
type MobileSheet = 'terminals' | 'manage-terminal' | TerminalEditorSheet | 'terminal-path-picker' | 'terminal-path-bookmarks' | 'clipboard-history' | null
type AppPage = 'terminal-list' | 'terminal'
type TerminalSwitcherInventory =
  | { status: 'loading'; terminals: RemoteTerminal[] }
  | { status: 'ready'; terminals: RemoteTerminal[] }
  | { status: 'error'; terminals: RemoteTerminal[] }
type FileManagerLoadState =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready'; component: FileManagerComponent }
  | { status: 'error' }
const TERMINAL_CONNECTION_PROGRESS_DELAY_MS = 450
const machineWorkspaceInventoryCache = new WeakMap<MachineWorkspaceConnector, Map<string, {
  machine: Machine
  terminals: RemoteTerminal[]
}>>()

function noopSubscribe(_listener: () => void): () => void { return () => {} }

function inventoryCacheForConnector(connector: MachineWorkspaceConnector): Map<string, {
  machine: Machine
  terminals: RemoteTerminal[]
}> {
  const existing = machineWorkspaceInventoryCache.get(connector)
  if (existing) return existing
  const created = new Map<string, {
    machine: Machine
    terminals: RemoteTerminal[]
  }>()
  machineWorkspaceInventoryCache.set(connector, created)
  return created
}

/**
 * MachineWorkspace 编排单个设备的 terminal/file 用户界面并消费 Go-owned session 投影。
 * 它不拥有 Endpoint、Route、credential 或 generation 真值，连接阶段仅用于本地化展示和交互反馈。
 */
export function MachineWorkspace({ api, connector, retainConnectionDemand, className, initialMachine, inventoryEvents, connectionStateEvents, subscribeRuntimeInventoryEvents = false, onBack, fileTransfer, terminalSettings: terminalSettingsProp, onNeedsReauthorization, onTerminalSettingsChange, phoneOnline = true, connectionState = 'ready', cloudPresence, singlePane = false, webLayout = false, storage, initialTerminalId, onInitialTerminalOpened, terminalSwitcherMachines = [], loadMachineTerminals, onSwitchTerminal, systemClipboard = browserSystemClipboard }: MachineWorkspaceProps) {
  const { t } = useTranslation()
  const connectionReady = appConnectionIsReady(connectionState)
  const initialInventory = initialMachine ? inventoryCacheForConnector(connector).get(initialMachine.machineId) : undefined
  const [machine, setMachine] = useState<Machine | null>(() => initialInventory?.machine ?? initialMachine ?? null)
  const [terminals, setTerminals] = useState<RemoteTerminal[]>(() => initialInventory?.terminals ?? [])
  const [terminalOrder, setTerminalOrder] = useState<string[]>(() => readTerminalOrder(initialMachine?.machineId ?? ''))
  const [hasLoadedTerminals, setHasLoadedTerminals] = useState(() => Boolean(initialInventory))
  const [loadingTerminals, setLoadingTerminals] = useState(() => !initialInventory)
  const [activeTerminalId, setActiveTerminalId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [connectionFailure, setConnectionFailure] = useState<ConnectionFailurePresentation | null>(null)
  const [connectionRetryPending, setConnectionRetryPending] = useState(false)
  const connectionRetryPromiseRef = useRef<Promise<void> | null>(null)
  const [hasConnectedOnce, setHasConnectedOnce] = useState(false)
  const [pairStatus, setPairStatus] = useState<string | null>(null)
  const [verifiedDevice, setVerifiedDevice] = useState<boolean | null>(true)
  const [connectedSession, setConnectedSession] = useState<MachineWorkspaceClientSession | null>(null)
  const [connectedTerminalId, setConnectedTerminalId] = useState<string | null>(null)
  const [connectingTerminalId, setConnectingTerminalId] = useState<string | null>(null)
  const [fileTerminalId, setFileTerminalId] = useState<string | null>(null)
  const [fileInitialPath, setFileInitialPath] = useState('/')
  const [fileContextKey, setFileContextKey] = useState('machine:/')
  const [connectionRetryToken, setConnectionRetryToken] = useState(0)
  const [forceRelayConnection, setForceRelayConnection] = useState<boolean | undefined>(undefined)
  const [connectionInfoOpen, setConnectionInfoOpen] = useState(false)
  const [connectionInfo, setConnectionInfo] = useState<ConnectionInfo | null>(null)
  const [connectionInfoLoading, setConnectionInfoLoading] = useState(false)
  const [connectionInfoError, setConnectionInfoError] = useState<string | null>(null)
  const [terminalInventoryRelay, setTerminalInventoryRelay] = useState(false)
  const [connectionPolicyState, setConnectionPolicyState] = useState<ConnectionPolicyState | null>(null)
  const [connectionPolicyApplying, setConnectionPolicyApplying] = useState(false)
  const connectionPolicyReconnectPendingRef = useRef(false)
  const connectionPolicyFailureRef = useRef<{ stage: 'refresh' | 'apply' | 'reconnect'; policy?: ConnectionPolicy } | null>(null)
  const [manualReconnectNonce, setManualReconnectNonce] = useState(0)
  const [terminalResizeControlBySlot, setTerminalResizeControlBySlot] = useState<Record<TerminalPaneKey, TerminalResizeControl>>({
    primary: defaultTerminalResizeControl,
  })
  const [resizeOwnerPending, setResizeOwnerPending] = useState(false)
  const [sizeLockPending, setSizeLockPending] = useState(false)
  const [terminalKeyboardModeRevision, setTerminalKeyboardModeRevision] = useState(0)
  const [expandedTerminalSwitcherMachineId, setExpandedTerminalSwitcherMachineId] = useState<string | null>(null)
  const [terminalSwitcherInventoryByMachine, setTerminalSwitcherInventoryByMachine] = useState<Record<string, TerminalSwitcherInventory>>({})

  const [page, setPage] = useState<AppPage>('terminal-list')
  const [filesOpen, setFilesOpen] = useState(false)
  const [hasOpenedFiles, setHasOpenedFiles] = useState(false)
  const [fileManagerLoadState, setFileManagerLoadState] = useState<FileManagerLoadState>({ status: 'idle' })
  const [transferCenterOpen, setTransferCenterOpen] = useState(false)
  const [mobileSheet, setMobileSheet] = useState<MobileSheet>(null)
  const [selectedTerminalId, setSelectedTerminalId] = useState<string | null>(null)
  const [terminalForm, setTerminalForm] = useState<{
    name: string
    command: string
    cwd: string
    environment: string[]
    sizeLockMode: 'off' | 'warn' | 'lock'
  }>({
    name: '',
    command: '',
    cwd: '',
    environment: [],
    sizeLockMode: 'off',
  })
  const [terminalSubmitError, setTerminalSubmitError] = useState<string | null>(null)
  const [terminalSubmitting, setTerminalSubmitting] = useState(false)
  const [restartingTerminalId, setRestartingTerminalId] = useState<string | null>(null)
  const [terminalDefaultsLoading, setTerminalDefaultsLoading] = useState(false)
  const terminalDefaultsRequestRef = useRef(0)
  const terminalNameInputRef = useRef<HTMLInputElement>(null)
  const [terminalPathReturnSheet, setTerminalPathReturnSheet] = useState<TerminalEditorSheet>('create-terminal')
  const [terminalPathPickerPath, setTerminalPathPickerPath] = useState('/')
  const [terminalPathPickerEntries, setTerminalPathPickerEntries] = useState<FileEntry[]>([])
  const [terminalPathPickerLoading, setTerminalPathPickerLoading] = useState(false)
  const [terminalPathPickerError, setTerminalPathPickerError] = useState<string | null>(null)
  const [terminalPathBookmarks, setTerminalPathBookmarks] = useState<PathBookmark[]>([])
  const [terminalPathBookmarksLoading, setTerminalPathBookmarksLoading] = useState(false)
  const [terminalPathBookmarksError, setTerminalPathBookmarksError] = useState<string | null>(null)
  const [clipboardEntries, setClipboardEntries] = useState<RemoteClipboardEntry[]>([])
  const [clipboardLoading, setClipboardLoading] = useState(false)
  const [clipboardError, setClipboardError] = useState<string | null>(null)
  const [clipboardDraft, setClipboardDraft] = useState('')
  const [editingClipboardId, setEditingClipboardId] = useState<string | null>(null)
  const [modifierState, setModifierState] = useState<TerminalModifierState>({ ctrl: 'off', alt: 'off' })
  const [terminalToolbarOpen, setTerminalToolbarOpen] = useState(false)
  const [terminalToolbarMode, setTerminalToolbarMode] = useState<TerminalToolbarMode>('default')
  const [terminalFnOpen, setTerminalFnOpen] = useState(false)
  const [hasTerminalSelection, setHasTerminalSelection] = useState(false)
  const [pasteConfirmText, setPasteConfirmText] = useState('')
  const [terminalSplitRoot, setTerminalSplitRoot] = useState<TerminalSplitNode>(PRIMARY_TERMINAL_PANE)
  const [activeTerminalSlot, setActiveTerminalSlot] = useState<TerminalPaneKey>('primary')
  const [webTerminalTabs, setWebTerminalTabs] = useState<WebTerminalTabLayout[]>([])
  const [webDraggedTerminalId, setWebDraggedTerminalId] = useState<string | null>(null)
  const [webTerminalSidebarOpen, setWebTerminalSidebarOpen] = useState(true)
  const [webTerminalPickerOpen, setWebTerminalPickerOpen] = useState(false)
  const [webSettingsOpen, setWebSettingsOpen] = useState(false)
  const [terminalHistorySearchOpenBySlot, setTerminalHistorySearchOpenBySlot] = useState<Record<TerminalPaneKey, boolean>>({ primary: false })

  const [syncSplitInput, setSyncSplitInput] = useState(false)
  const [terminalBufferBySlot, setTerminalBufferBySlot] = useState<Record<TerminalPaneKey, 'normal' | 'alternate'>>({ primary: 'normal' })
  const [terminalSettings, setTerminalSettings] = useState<TerminalSettings>(() => readTerminalSettings())
  const terminalRef = useRef<TerminalHandle | null>(null)
  const splitTerminalRefs = useRef(new Map<string, TerminalHandle>())
  const terminalSplitSequenceRef = useRef(0)
  const fileReturnPageRef = useRef<AppPage>('terminal-list')
  const fileManagerLoadRequestRef = useRef(0)
  const fileManagerLoadStatusRef = useRef<FileManagerLoadState['status']>('idle')
  const fileManagerLoaderMountedRef = useRef(true)
  const fileManagerContextRef = useRef(machine?.machineId ?? null)
  const outerContainerRef = useRef<HTMLDivElement | null>(null)
  const terminalAreaRef = useRef<HTMLDivElement | null>(null)
  const terminalWrapperRef = useRef<HTMLDivElement | null>(null)
  const mobileKeybarRef = useRef<HTMLDivElement | null>(null)
  const activeTerminalSlotRef = useRef<TerminalPaneKey>('primary')
  const terminalToolbarOpenerRef = useRef<HTMLButtonElement | null>(null)
  const [keyboardFocusLocked, setKeyboardFocusLocked] = useState(false)
  const machineSessionRef = useRef<{
    connector: MachineWorkspaceConnector
    machineId: string
    retryToken: number
    session: MachineWorkspaceClientSession
    forceRelay: boolean | undefined
  } | null>(null)
  const machineSessionPromiseRef = useRef<{
    connector: MachineWorkspaceConnector
    machineId: string
    retryToken: number
    forceRelay: boolean | undefined
    promise: Promise<MachineWorkspaceClientSession>
  } | null>(null)
  const machineSessionConnectSeqRef = useRef(0)
  const terminalRefreshSeqRef = useRef(0)
  const runtimeInventorySubscriptionRef = useRef<{
    connector: MachineWorkspaceConnector
    machineId: string
    retryToken: number
    session: MachineWorkspaceClientSession
    subscription: { close(): void }
  } | null>(null)
  const connectionStateSubscriptionRef = useRef<RtcSubscription | null>(null)
  const passiveConnectionPhaseRef = useRef<RtcConnectionStateSnapshot['phase'] | null>(null)
  const sessionConnectionPhaseRef = useRef<RtcConnectionStateSnapshot['phase'] | null>(null)
  const latestActiveTerminalIdRef = useRef<string | null>(null)
  const latestSplitTerminalIdsRef = useRef<string[]>([])
  const handledManualReconnectNonceRef = useRef(0)
  const resizeLockedHintShownRef = useRef(false)
  const previousPhoneOnlineRef = useRef(phoneOnline)
  const previousConnectionStateRef = useRef(connectionState)
  const handledNativeResumeRevisionRef = useRef(0)
  const phoneOnlineRef = useRef(phoneOnline)
  const connectionReadyRef = useRef(connectionReady)
  const hasConnectedOnceRef = useRef(hasConnectedOnce)
  const hasLoadedTerminalsRef = useRef(hasLoadedTerminals)
  phoneOnlineRef.current = phoneOnline
  connectionReadyRef.current = connectionReady
  hasConnectedOnceRef.current = hasConnectedOnce
  const displayedPaneKeys = useMemo(() => terminalPaneKeys(terminalSplitRoot), [terminalSplitRoot])
  const webTabTerminalIds = useMemo(() => webTerminalTabs.map((tab) => tab.terminalId), [webTerminalTabs])
  const webSplitTabTerminalIds = useMemo(() => webTerminalTabs.flatMap((tab) => terminalPaneKeys(tab.root).length > 1 ? [tab.terminalId] : []), [webTerminalTabs])
  const splitTerminalIds = useMemo(() => displayedPaneKeys.flatMap((paneKey) => {
    const terminalId = terminalIdForPane(paneKey, activeTerminalId)
    return paneKey === 'primary' || !terminalId ? [] : [terminalId]
  }), [activeTerminalId, displayedPaneKeys])
  const hasSplitTerminals = splitTerminalIds.length > 0
  const activePaneTerminalId = terminalIdForPane(activeTerminalSlot, activeTerminalId)
  const activeTerminal = terminals.find((terminal) => terminal.terminalId === activeTerminalId)
  const activeToolTerminal = terminals.find((terminal) => terminal.terminalId === activePaneTerminalId) ?? activeTerminal
  const activeTerminalResizeControl = terminalResizeControlBySlot[activeTerminalSlot] ?? defaultTerminalResizeControl
  // connectedSession 是 React 投影，不是 generation 真值。底层 Go session 已失效时，
  // Terminal/FileManager 必须等待新 lease，不能先挂载旧资源再由 effect 事后清理。
  const renderSession = connectedSession && isProtoSessionAlive(connectedSession) ? connectedSession : null
  const LoadedFileManager = fileManagerLoadState.status === 'ready' ? fileManagerLoadState.component : null
  const selectedTerminal = terminals.find((terminal) => terminal.terminalId === selectedTerminalId)
  const orderedTerminals = useMemo(() => sortTerminalIds(terminals, terminalOrder), [terminalOrder, terminals])
  const canSplitTerminal = orderedTerminals.some((terminal) => terminal.terminalId !== activeTerminalId && !splitTerminalIds.includes(terminal.terminalId))
  const selectedTerminalPinned = Boolean(selectedTerminal && terminalOrder.includes(selectedTerminal.terminalId))
  const terminalHeaderTitle = activeToolTerminal?.title || activeToolTerminal?.command || activePaneTerminalId || t('terminal.defaultTitle')
  const terminalHeaderMachine = machine?.name || machine?.machineId || ''
  const terminalHeaderDirectory = activeToolTerminal?.cwd || activeTerminal?.cwd || ''
  const terminalHeaderSummary = [machine?.name, terminalHeaderTitle, terminalHeaderDirectory].filter(Boolean).join(' · ')

  useEffect(() => retainConnectionDemand?.(), [retainConnectionDemand])

  useEffect(() => {
    setTerminalOrder(readTerminalOrder(machine?.machineId ?? ''))
  }, [machine?.machineId])

  const updateTerminalPins = useCallback((update: (current: string[]) => string[]) => {
    if (!machine) return
    setTerminalOrder((current) => writeTerminalOrder(machine.machineId, update(current)))
  }, [machine])
  const toggleTerminalPin = useCallback((terminalId: string) => {
    updateTerminalPins((current) => current.includes(terminalId)
      ? unpinTerminal(current, terminalId)
      : pinTerminal(current, terminalId))
  }, [updateTerminalPins])
  const reorderTerminalPins = useCallback((terminalId: string, targetTerminalId: string, placement: 'before' | 'after') => {
    updateTerminalPins((current) => reorderPinnedTerminal(current, terminalId, targetTerminalId, placement))
  }, [updateTerminalPins])
  const activeTerminalResizeLocked = activeTerminalResizeControl.sizeLocked === true || activeTerminalResizeControl.reason === 'size_locked'
  const resizeLockedHint = t('workspace.resize.lockedHint')
  const requireVerification = verifiedDevice === false
  const emptyTransferSnapshot = useMemo(() => ({ transfers: [], hasActiveTransfers: false }), [])
  const transferState = useSyncExternalStore(
    fileTransfer?.subscribe ?? noopSubscribe,
    fileTransfer?.getSnapshot ?? (() => emptyTransferSnapshot),
  )
  const {
    connectionStatus,
    connectionPhase,
    showMachineNetworkOverlay,
    showDelayedMachineNetworkOverlay,
    setMachineNetworkMachineId,
    updateConnectionStatus,
    clearConnectionStatus,
    clearConnectionStatusSoon,
  } = useMachineNetworkStatus()
  const displayedConnectionStatus = connectionPhase
    ? connectionPhaseLabel(connectionPhase, t)
    : connectionStatus
  const activeSlotTerminalExited = activeToolTerminal?.state === 'exited'
  const connectionSessionUnavailable = !phoneOnline || !connectionReady || showMachineNetworkOverlay || Boolean(connectionFailure)
  const connectionInputBlocked = connectionSessionUnavailable || activeSlotTerminalExited === true
  const canManageTerminals = !connectionSessionUnavailable && !requireVerification
  const initialConnectionFailure = !hasConnectedOnce && phoneOnline && connectionReady ? connectionFailure : null
  const hideTerminalListForUnavailableState = Boolean(initialConnectionFailure)
  const effectiveTerminalSettings = terminalSettingsProp ?? terminalSettings
  const activeTerminalKeyboardMode = useMemo<TerminalKeyboardMode>(() => {
    if (!machine || !activeToolTerminal) return effectiveTerminalSettings.keyboardMode
    return readTerminalKeyboardMode(machine.machineId, activeToolTerminal.terminalId, storage) ?? effectiveTerminalSettings.keyboardMode
  }, [activeToolTerminal, effectiveTerminalSettings.keyboardMode, machine, storage, terminalKeyboardModeRevision])
  const updateActiveTerminalKeyboardMode = useCallback((mode: TerminalKeyboardMode) => {
    if (!machine || !activeToolTerminal) return
    writeTerminalKeyboardMode(machine.machineId, activeToolTerminal.terminalId, mode, storage)
    setTerminalKeyboardModeRevision((current) => current + 1)
  }, [activeToolTerminal, machine, storage])
  const terminalThemeStyle = useMemo(
    () => terminalThemeCssVariables(effectiveTerminalSettings.themeId) as CSSProperties,
    [effectiveTerminalSettings.themeId],
  )
  const startFileManagerLoad = useCallback(() => {
    if (fileManagerLoadStatusRef.current !== 'idle') return
    const request = fileManagerLoadRequestRef.current + 1
    fileManagerLoadRequestRef.current = request
    fileManagerLoadStatusRef.current = 'loading'
    setFileManagerLoadState({ status: 'loading' })
    void loadFileManager().then((component) => {
      if (!fileManagerLoaderMountedRef.current || fileManagerLoadRequestRef.current !== request) return
      fileManagerLoadStatusRef.current = 'ready'
      setFileManagerLoadState({ status: 'ready', component })
    }).catch(() => {
      if (!fileManagerLoaderMountedRef.current || fileManagerLoadRequestRef.current !== request) return
      fileManagerLoadStatusRef.current = 'error'
      setFileManagerLoadState({ status: 'error' })
    })
  }, [])
  const terminalHandleForPane = useCallback((paneKey: TerminalPaneKey): TerminalHandle | null => {
    if (paneKey === 'primary') return terminalRef.current
    return splitTerminalRefs.current.get(terminalIdForPane(paneKey, null) ?? '') ?? null
  }, [])
  const forEachDisplayedTerminalHandle = useCallback((visit: (handle: TerminalHandle) => void) => {
    if (terminalRef.current) visit(terminalRef.current)
    splitTerminalRefs.current.forEach(visit)
  }, [])
  const fitDisplayedTerminals = useCallback(() => {
    forEachDisplayedTerminalHandle((handle) => handle.fit())
  }, [forEachDisplayedTerminalHandle])
  const getKeyboardLayoutMode = useCallback(() => {
    if (hasSplitTerminals) return 'resize' as const
    if (activeTerminalKeyboardMode === 'resize') return 'resize' as const
    if (activeTerminalKeyboardMode === 'shift') return 'shift' as const
    return terminalBufferBySlot[activeTerminalSlot] === 'alternate' ? 'resize' as const : 'shift' as const
  }, [activeTerminalKeyboardMode, activeTerminalSlot, hasSplitTerminals, terminalBufferBySlot])
  const {
    keyboardVisible,
    reapplyKeyboardLayout,
    handleBufferChange,
    handleCursorMove,
    markKeyboardVisible,
    markKeyboardHidden,
    resetKeyboardLayout,
  } = useTerminalKeyboard({
    containerRef: outerContainerRef,
    mainRef: terminalAreaRef,
    termWrapperRef: terminalWrapperRef,
    getTermRef: () => terminalHandleForPane(activeTerminalSlotRef.current),
    getLayoutMode: getKeyboardLayoutMode,
    onKeyboardHide: () => {
      requestAnimationFrame(fitDisplayedTerminals)
    },
  })

  useEffect(() => {
    fileManagerLoaderMountedRef.current = true
    return () => {
      fileManagerLoaderMountedRef.current = false
      fileManagerLoadRequestRef.current += 1
    }
  }, [])

  useEffect(() => {
    activeTerminalSlotRef.current = activeTerminalSlot
  }, [activeTerminalSlot])

  useEffect(() => {
    reapplyKeyboardLayout()
    requestAnimationFrame(fitDisplayedTerminals)
  }, [activeTerminalKeyboardMode, activeTerminalSlot, fitDisplayedTerminals, getKeyboardLayoutMode, reapplyKeyboardLayout, terminalBufferBySlot, terminalSplitRoot])

  useEffect(() => {
    latestActiveTerminalIdRef.current = activeTerminalId
  }, [activeTerminalId])

  useEffect(() => {
    latestSplitTerminalIdsRef.current = splitTerminalIds
  }, [splitTerminalIds])

  useEffect(() => {
    resizeLockedHintShownRef.current = false
  }, [activeTerminalId])

  useEffect(() => {
    setTerminalBufferBySlot((current) => displayedPaneKeys.every((paneKey) => current[paneKey] === 'normal')
      ? current
      : Object.fromEntries(displayedPaneKeys.map((paneKey) => [paneKey, 'normal'])) as Record<TerminalPaneKey, 'normal'>)
  }, [activeTerminalId, displayedPaneKeys])

  useEffect(() => {
    hasLoadedTerminalsRef.current = hasLoadedTerminals
  }, [hasLoadedTerminals])

  useEffect(() => {
    if (!initialMachine) return
    setMachine((current) => {
      if (!current || current.machineId !== initialMachine.machineId) return initialMachine
      return current
    })
  }, [initialMachine])

  useEffect(() => {
    const context = machine?.machineId ?? null
    if (fileManagerContextRef.current === context) return
    fileManagerContextRef.current = context
    fileManagerLoadRequestRef.current += 1
    fileManagerLoadStatusRef.current = 'idle'
    setFileManagerLoadState({ status: 'idle' })
    setFilesOpen(false)
    setHasOpenedFiles(false)
    fileReturnPageRef.current = 'terminal-list'
    setFileTerminalId(null)
    setFileInitialPath('/')
    setFileContextKey('machine:/')
  }, [machine?.machineId])

  useEffect(() => {
    setMachineNetworkMachineId(machine?.machineId ?? null)
  }, [machine?.machineId, setMachineNetworkMachineId])

  useEffect(() => {
    // Endpoint 切换后不提供一次性 override；下一代 session 必须直接消费 Go registry 中的持久策略。
    setForceRelayConnection(undefined)
  }, [machine?.machineId])

  useEffect(() => {
    const keybar = mobileKeybarRef.current
    const terminalArea = terminalAreaRef.current
    if (!keybar || !terminalArea || typeof ResizeObserver === 'undefined') return

    let fitFrame = 0
    const refitTerminals = () => {
      if (keyboardVisible) reapplyKeyboardLayout()
      if (fitFrame) cancelAnimationFrame(fitFrame)
      fitFrame = requestAnimationFrame(() => {
        fitFrame = requestAnimationFrame(() => {
          fitFrame = 0
          fitDisplayedTerminals()
        })
      })
    }

    refitTerminals()
    const observer = new ResizeObserver(refitTerminals)
    observer.observe(keybar)
    observer.observe(terminalArea)
    return () => {
      observer.disconnect()
      if (fitFrame) cancelAnimationFrame(fitFrame)
    }
  }, [fitDisplayedTerminals, keyboardVisible, page, reapplyKeyboardLayout])

  const updateTerminalSettings = useCallback((patch: Partial<TerminalSettings>) => {
    if (onTerminalSettingsChange) {
      onTerminalSettingsChange(patch)
      return
    }
    setTerminalSettings((current) => writeTerminalSettings({ ...current, ...patch }))
  }, [onTerminalSettingsChange])

  const handleConnectionAuthFailure = useCallback((machineId?: string | null) => {
    const targetMachineId = machineId ?? initialMachine?.machineId
    if (!targetMachineId || !onNeedsReauthorization) return
    setVerifiedDevice(false)
    onNeedsReauthorization(targetMachineId)
  }, [initialMachine?.machineId, onNeedsReauthorization])

  const updateFromConnectionState = useCallback((snapshot: RtcConnectionStateSnapshot, session?: MachineWorkspaceClientSession, failureSource?: unknown) => {
    if (snapshot.phase === 'connected') {
      setTerminalInventoryRelay(snapshot.relayInUse)
      setError(null)
      setConnectionFailure(null)
      setHasConnectedOnce(true)
      hasConnectedOnceRef.current = true
      if (session) {
        setConnectedSession(session)
        const terminalId = latestActiveTerminalIdRef.current
        if (terminalId) {
          setConnectedTerminalId(terminalId)
          setConnectingTerminalId(null)
        }
      }
      updateConnectionStatus(snapshot.statusText || connectionPhaseLabel('connected', t), 'connected')
      clearConnectionStatusSoon()
    connectionPolicyFailureRef.current = null
    if (connectionPolicyReconnectPendingRef.current) {
    connectionPolicyReconnectPendingRef.current = false
    setConnectionPolicyApplying(false)
    setConnectionInfoOpen(false)
    }
      return
    }
    if (snapshot.phase === 'idle') {
      clearConnectionStatus()
      return
    }
    if (snapshot.phase === 'reconnecting' || snapshot.phase === 'waiting_network') setError(null)
    if (snapshot.phase === 'failed') {
      const source = (failureSource ?? snapshot.error) || snapshot.statusText || t('machines.connectionFailed')
      if (isCancelledConnectionError(source)) {
        setError(null)
        setConnectionFailure(null)
        if (connectionPolicyReconnectPendingRef.current) {
          connectionPolicyReconnectPendingRef.current = false
          setConnectionPolicyApplying(false)
        }
        clearConnectionStatus()
        return
      }
      const failure = connectionFailurePresentation(source, t, { phoneOnline: phoneOnlineRef.current })
      const message = failure.message
      if (isAuthorizationConnectionError(source)) handleConnectionAuthFailure(snapshot.machineId)
      setError(message)
      setConnectionFailure(failure)
    if (connectionPolicyReconnectPendingRef.current) {
    connectionPolicyReconnectPendingRef.current = false
    setConnectionPolicyApplying(false)
    setConnectionInfoError(message)
    connectionPolicyFailureRef.current = { stage: 'reconnect' }
    setConnectionInfoOpen(true)
    clearConnectionStatus()
    return
    }
      updateConnectionStatus(message, 'failed')
      return
    }
    updateConnectionStatus(snapshot.statusText || connectionPhaseLabel(snapshot.phase, t), snapshot.phase)
  }, [clearConnectionStatus, clearConnectionStatusSoon, handleConnectionAuthFailure, t, updateConnectionStatus])

  const updateFromPassiveConnectionState = useCallback((snapshot: RtcConnectionStateSnapshot, session?: MachineWorkspaceClientSession) => {
    if (isTransientConnectionPhase(snapshot.phase)) return
    updateFromConnectionState(snapshot, session)
  }, [updateFromConnectionState])

  const reattachActiveTerminals = useCallback((session: MachineWorkspaceClientSession) => {
    const terminalId = latestActiveTerminalIdRef.current
    const currentSplitTerminalIds = latestSplitTerminalIdsRef.current
    if (!terminalId && currentSplitTerminalIds.length === 0) return
    setConnectedSession(session)
    if (terminalId) {
      setConnectedTerminalId(terminalId)
      setConnectingTerminalId(null)
      terminalRef.current?.reattach(session, { forceTerminalChannel: true })
    }
    currentSplitTerminalIds.forEach((splitTerminalId) => {
      splitTerminalRefs.current.get(splitTerminalId)?.reattach(session, { forceTerminalChannel: true })
    })
  }, [])

  const disconnectMachineSession = useCallback(() => {
    machineSessionConnectSeqRef.current += 1
    connectionStateSubscriptionRef.current?.close()
    connectionStateSubscriptionRef.current = null
    const current = machineSessionRef.current
    machineSessionPromiseRef.current = null
    machineSessionRef.current = null
    const runtimeInventorySubscription = runtimeInventorySubscriptionRef.current
    runtimeInventorySubscriptionRef.current = null
    runtimeInventorySubscription?.subscription.close()
    if (current) void closeMachineWorkspaceSession(current.session)
  }, [])

  const attachConnectionStateSubscription = useCallback((session: MachineWorkspaceClientSession) => {
    connectionStateSubscriptionRef.current?.close()
    sessionConnectionPhaseRef.current = null
    connectionStateSubscriptionRef.current = session.subscribeClosed((error) => {
      const current = machineSessionRef.current
      if (!current || current.session !== session) return
      connectionStateSubscriptionRef.current?.close()
      connectionStateSubscriptionRef.current = null
      machineSessionRef.current = null
      machineSessionPromiseRef.current = null
      setConnectedSession(null)
      updateFromConnectionState({
        machineId: session.stamp.endpointId,
        phase: 'failed',
        statusText: error.message,
        relayInUse: false,
      }, undefined, error)
    })
  }, [updateFromConnectionState])

  const releaseMachineSession = useCallback(() => {
    disconnectMachineSession()
    setConnectedSession(null)
    setConnectedTerminalId(null)
    setConnectingTerminalId(null)
  }, [disconnectMachineSession])

  const ensureMachineSession = useCallback(async (machineId: string, connectOptions?: RtcConnectOptions): Promise<MachineWorkspaceClientSession> => {
    if (!phoneOnlineRef.current) throw Object.assign(new Error('phone offline'), { code: 'offline' })
    if (!connectionReadyRef.current) throw Object.assign(new Error('connection generation is not ready'), { code: 'cancelled' })
    const forceRelay = connectOptions?.forceRelay ?? forceRelayConnection
    const effectiveConnectOptions: RtcConnectOptions = { ...connectOptions, forceRelay }
    const reusable = machineSessionRef.current
    if (
      reusable &&
      reusable.connector === connector &&
      reusable.machineId === machineId &&
      reusable.retryToken === connectionRetryToken &&
      reusable.forceRelay === forceRelay
    ) {
      if (isProtoSessionAlive(reusable.session)) return reusable.session
      releaseMachineSession()
    }
    const pending = machineSessionPromiseRef.current
    if (
      pending &&
      pending.connector === connector &&
      pending.machineId === machineId &&
      pending.retryToken === connectionRetryToken &&
      pending.forceRelay === forceRelay
    ) {
      return pending.promise
    }
    const entry: {
      connector: MachineWorkspaceConnector
      machineId: string
      retryToken: number
      forceRelay: boolean | undefined
      promise: Promise<MachineWorkspaceClientSession>
    } = {
      connector,
      machineId,
      retryToken: connectionRetryToken,
      forceRelay,
      promise: Promise.resolve(null as unknown as MachineWorkspaceClientSession),
    }
    entry.promise = connector.connect({ machineId }, effectiveConnectOptions).then((session) => {
      if (machineSessionPromiseRef.current !== entry) {
        void closeMachineWorkspaceSession(session)
        return session
      }
      machineSessionPromiseRef.current = null
      machineSessionRef.current = {
        connector,
        machineId,
        retryToken: connectionRetryToken,
        forceRelay,
        session,
      }
      attachConnectionStateSubscription(session)
      setConnectedSession(session)
      return session
    }).catch((err: unknown) => {
      if (machineSessionPromiseRef.current === entry) {
        machineSessionPromiseRef.current = null
      }
      throw err
    })
    machineSessionPromiseRef.current = entry
    return entry.promise
  }, [attachConnectionStateSubscription, connector, connectionRetryToken, forceRelayConnection, releaseMachineSession])

  const withManagementApi = useCallback(async () => {
    if (!machine) throw new Error('machine is required before managing terminals')
    const session = await ensureMachineSession(machine.machineId, { forceRelay: forceRelayConnection })
    return {
      session,
      api: createTerminalManagementApi(session, machine.machineId),
    }
  }, [ensureMachineSession, forceRelayConnection, machine])

  const withMachineSession = useCallback(async () => {
    if (!machine) throw new Error('machine is required before using runtime storage')
    return await ensureMachineSession(machine.machineId, { forceRelay: forceRelayConnection })
  }, [ensureMachineSession, forceRelayConnection, machine])

  const refreshTerminals = useCallback(async () => {
    if (!phoneOnlineRef.current || !connectionReadyRef.current) {
      return
    }
    const seq = terminalRefreshSeqRef.current + 1
    terminalRefreshSeqRef.current = seq
    if (!hasLoadedTerminalsRef.current) setLoadingTerminals(true)
    let refreshMachineId = initialMachine?.machineId ?? null
    try {
      const status = await api.getStatus()
      if (terminalRefreshSeqRef.current !== seq) return
      refreshMachineId = status.machine.machineId
      setMachineNetworkMachineId(status.machine.machineId)
      setMachine(status.machine)
      const terminalList = await api.listTerminals({ forceRelay: forceRelayConnection })
      if (terminalRefreshSeqRef.current !== seq) return
      setTerminals(terminalList)
      setHasLoadedTerminals(true)
      setError(null)
      setConnectionFailure(null)
      setHasConnectedOnce(true)
      hasConnectedOnceRef.current = true
      // generation 恢复可能先收到旧 binding 的失败，再由新 session 成功提交 inventory。
      // 新 inventory 是当前 workspace 的成功真值，必须同时清除旧 generation 留下的网络遮罩。
      clearConnectionStatus()
    } catch (err) {
      if (terminalRefreshSeqRef.current === seq) {
        if (isCancelledConnectionError(err)) {
          setError(null)
          setConnectionFailure(null)
          clearConnectionStatus()
          return
        }
        const failure = connectionFailurePresentation(err, t, { phoneOnline: phoneOnlineRef.current })
        const message = failure.message
        if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(refreshMachineId)
        setError(message)
        setConnectionFailure(failure)
        updateConnectionStatus(message, 'failed')
      }
    } finally {
      if (terminalRefreshSeqRef.current === seq) {
        setLoadingTerminals(false)
      }
    }
  }, [api, clearConnectionStatus, forceRelayConnection, handleConnectionAuthFailure, initialMachine?.machineId, setMachineNetworkMachineId, t, updateConnectionStatus])

  const applyRuntimeTerminalEvent = useCallback((event: RtcEvent | { payload?: unknown }): boolean => {
    const payload = event.payload
    if (typeof payload !== 'object' || payload === null || Array.isArray(payload)) return false
    const record = payload as Record<string, unknown>
    const terminal = normalizeRuntimeTerminalEvent(record)
    if (!terminal) return false

    setTerminals((current) => {
      const index = current.findIndex((item) => item.terminalId === terminal.terminalId)
      if (index < 0) return current
      const next = current.slice()
      next[index] = { ...next[index], ...terminal }
      return next
    })

    return true
  }, [])

  const applyTerminalInfo = useCallback((terminal: RemoteTerminal) => {
    if (machine && terminal.machineId !== machine.machineId) return
    setTerminals((current) => {
      const index = current.findIndex((item) => item.terminalId === terminal.terminalId)
      if (index < 0) return current
      const next = current.slice()
      next[index] = { ...next[index], ...terminal }
      return next
    })
    if (terminal.state === 'exited' && (activeTerminalId === terminal.terminalId || splitTerminalIds.includes(terminal.terminalId))) {
      const slot = terminal.terminalId === activeTerminalId ? 'primary' : terminalPaneKey(terminal.terminalId)
      setTerminalResizeControlBySlot((current) => ({ ...current, [slot]: defaultTerminalResizeControl }))
    }
  }, [activeTerminalId, machine, splitTerminalIds])

  useEffect(() => {
    if (!phoneOnlineRef.current || !connectionReadyRef.current) {
      return
    }
    let cancelled = false
    const seq = terminalRefreshSeqRef.current + 1
    terminalRefreshSeqRef.current = seq
    async function load() {
      if (!hasLoadedTerminalsRef.current) setLoadingTerminals(true)
      let failed = false
      let loadMachineId = initialMachine?.machineId ?? null
      try {
        const status = await api.getStatus()
        if (cancelled || terminalRefreshSeqRef.current !== seq) return
        loadMachineId = status.machine.machineId
        setMachineNetworkMachineId(status.machine.machineId)
        setMachine(status.machine)
        const cachedInventory = inventoryCacheForConnector(connector).get(status.machine.machineId)
        if (cachedInventory && !hasLoadedTerminalsRef.current) {
          setTerminals(cachedInventory.terminals)
          setHasLoadedTerminals(true)
          setLoadingTerminals(false)
        }
        const terminalList = await api.listTerminals({
          onStatus: (status) => {
            if (!cancelled && terminalRefreshSeqRef.current === seq) updateConnectionStatus(status)
          },
        })
        if (cancelled || terminalRefreshSeqRef.current !== seq) return
        setTerminals(terminalList)
        setHasLoadedTerminals(true)
        // 当前 refresh sequence 已由新 generation 成功提交，旧 bridge 的迟到错误不再代表当前连接。
        setError(null)
        setConnectionFailure(null)
        setHasConnectedOnce(true)
        hasConnectedOnceRef.current = true
      } catch (err) {
        if (!cancelled && terminalRefreshSeqRef.current === seq) {
          if (isCancelledConnectionError(err)) {
            setError(null)
            setConnectionFailure(null)
            clearConnectionStatus()
            return
          }
          failed = true
          const failure = connectionFailurePresentation(err, t, { phoneOnline: phoneOnlineRef.current })
          const message = failure.message
          if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(loadMachineId)
          setError(message)
          setConnectionFailure(failure)
          updateConnectionStatus(message, 'failed')
        }
      } finally {
        if (!cancelled && terminalRefreshSeqRef.current === seq) {
          setLoadingTerminals(false)
          if (!failed) clearConnectionStatus()
        }
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [api, clearConnectionStatus, connector, handleConnectionAuthFailure, initialMachine?.machineId, setMachineNetworkMachineId, t, updateConnectionStatus])

  useEffect(() => {
    if (!machine || !hasLoadedTerminals) return
    inventoryCacheForConnector(connector).set(machine.machineId, {
      machine,
      terminals,
    })
  }, [connector, hasLoadedTerminals, machine, terminals])

  useEffect(() => {
    if (page !== 'terminal-list' || !hasLoadedTerminals || !phoneOnline || !connectionReady || requireVerification) return
    let cancelled = false
    let timer: number | undefined
    const interval = terminalInventoryRefreshIntervalMs(forceRelayConnection === true || terminalInventoryRelay)
    const schedule = () => {
      if (!cancelled) timer = window.setTimeout(refresh, interval)
    }
    const refresh = async () => {
      if (cancelled) return
      if (document.visibilityState === 'hidden') {
        schedule()
        return
      }
      try {
        const terminalList = await api.listTerminals({ forceRelay: forceRelayConnection })
        if (!cancelled) setTerminals(terminalList)
      } catch {
        // 后台 inventory 刷新不拥有连接生命周期；session watcher 会处理真实断链。
      } finally {
        schedule()
      }
    }
    schedule()
    return () => {
      cancelled = true
      if (timer !== undefined) window.clearTimeout(timer)
    }
  }, [api, connectionReady, forceRelayConnection, hasLoadedTerminals, page, phoneOnline, requireVerification, terminalInventoryRelay])

  useEffect(() => {
    if (!inventoryEvents || !machine) return
    const subscription = inventoryEvents.subscribe(machine.machineId, (event) => {
      if (!applyRuntimeTerminalEvent(event)) {
        void refreshTerminals()
      }
    })
    return () => {
      subscription.close()
    }
  }, [applyRuntimeTerminalEvent, inventoryEvents, machine, refreshTerminals, connectionRetryToken])

  useEffect(() => {
    if (!connectionStateEvents || !machine) return
    passiveConnectionPhaseRef.current = null
    const subscription = connectionStateEvents.subscribe(machine.machineId, (snapshot) => {
      const previousPhase = passiveConnectionPhaseRef.current
      passiveConnectionPhaseRef.current = snapshot.phase
      const activeSession = machineSessionRef.current?.session
      const recoveredFromInterruption = previousPhase !== null && previousPhase !== 'connected'
      if (
        snapshot.phase === 'connected' &&
        (activeTerminalId || filesOpen) &&
        (!activeSession || !isProtoSessionAlive(activeSession) || recoveredFromInterruption)
      ) {
        void ensureMachineSession(machine.machineId, { forceRelay: forceRelayConnection })
          .then((session) => {
            reattachActiveTerminals(session)
            updateFromPassiveConnectionState(snapshot, session)
          })
          .catch((err: unknown) => {
            const message = connectionErrorDisplayMessage(err, t)
            if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(machine.machineId)
            updateConnectionStatus(message, 'failed')
          })
        return
      }
      updateFromPassiveConnectionState(snapshot, activeSession)
    })
    return () => {
      subscription.close()
    }
  }, [activeTerminalId, connectionStateEvents, ensureMachineSession, filesOpen, forceRelayConnection, handleConnectionAuthFailure, machine, reattachActiveTerminals, updateConnectionStatus, updateFromPassiveConnectionState])

  useEffect(() => {
    const machineId = machine?.machineId
    if (!subscribeRuntimeInventoryEvents || requireVerification || !machineId) return
    let cancelled = false
    void ensureMachineSession(machineId, { forceRelay: forceRelayConnection }).then((session) => {
      if (cancelled) return
      const current = runtimeInventorySubscriptionRef.current
      if (
        current &&
        current.connector === connector &&
        current.machineId === machineId &&
        current.retryToken === connectionRetryToken &&
        current.session === session
      ) {
        return
      }
      current?.subscription.close()
      const subscription = subscribeMachineWorkspaceEvents(session, (event) => {
        if (!isTerminalInventoryRuntimeEvent(event)) return
        if (!applyRuntimeTerminalEvent(event)) {
          void refreshTerminals()
        }
      })
      runtimeInventorySubscriptionRef.current = {
        connector,
        machineId,
        retryToken: connectionRetryToken,
        session,
        subscription,
      }
    }).catch(() => {
      if (cancelled) return
    })
    return () => {
      cancelled = true
      const current = runtimeInventorySubscriptionRef.current
      if (
        current &&
        current.connector === connector &&
        current.machineId === machineId &&
        current.retryToken === connectionRetryToken
      ) {
        runtimeInventorySubscriptionRef.current = null
        current.subscription.close()
      }
    }
  }, [applyRuntimeTerminalEvent, connectionRetryToken, connector, ensureMachineSession, forceRelayConnection, machine?.machineId, refreshTerminals, requireVerification, subscribeRuntimeInventoryEvents])

  useEffect(() => {
    const machineId = machine?.machineId
    const current = machineSessionRef.current
    if (!machineId) {
      if (current) releaseMachineSession()
      return
    }
    if (
      current &&
      (current.connector !== connector ||
        current.machineId !== machineId ||
        current.retryToken !== connectionRetryToken)
    ) {
      releaseMachineSession()
    }
    if (!phoneOnlineRef.current || !connectionReadyRef.current) {
      setConnectingTerminalId(activeTerminalId)
      return
    }
    if (!activeTerminalId) {
      setConnectedTerminalId(null)
      setConnectingTerminalId(null)
      return
    }
    const reusable = machineSessionRef.current
    if (
      reusable &&
      reusable.connector === connector &&
      reusable.machineId === machineId &&
      reusable.retryToken === connectionRetryToken
    ) {
      if (isProtoSessionAlive(reusable.session)) {
        setConnectedSession(reusable.session)
        setConnectedTerminalId(activeTerminalId)
        setConnectingTerminalId(null)
        return
      }
      releaseMachineSession()
    }
    if (page !== 'terminal') {
      setConnectingTerminalId(null)
      return
    }

    let cancelled = false
    const connectSeq = machineSessionConnectSeqRef.current + 1
    machineSessionConnectSeqRef.current = connectSeq
    setConnectedSession(null)
    setConnectedTerminalId(null)
    setConnectingTerminalId(activeTerminalId)
    let showConnectionProgress = false
    let lastConnectionSnapshot: RtcConnectionStateSnapshot | null = null
    let lastConnectionStatus: string | null = null
    const updateFromConnectionStatusText = (status: string) => {
      updateFromConnectionState(connectionSnapshotFromStatus({
        machineId,
        statusText: status,
      }))
    }
    const progressTimer = window.setTimeout(() => {
      if (cancelled || machineSessionConnectSeqRef.current !== connectSeq) return
      showConnectionProgress = true
      if (lastConnectionSnapshot) {
        updateFromConnectionState(lastConnectionSnapshot)
        return
      }
      updateFromConnectionStatusText(lastConnectionStatus ?? (forceRelayConnection
        ? t('workspace.connectingRelay')
        : connectionPhaseLabel('connecting', t)))
    }, TERMINAL_CONNECTION_PROGRESS_DELAY_MS)
    ensureMachineSession(machineId, {
      forceRelay: forceRelayConnection,
      onConnectionState: (snapshot) => {
        if (cancelled || machineSessionConnectSeqRef.current !== connectSeq) return
        lastConnectionSnapshot = snapshot
        if (showConnectionProgress || snapshot.phase === 'failed' || snapshot.phase === 'reconnecting' || snapshot.phase === 'waiting_network') {
          updateFromConnectionState(snapshot)
        }
      },
      onStatus: (status) => {
        if (cancelled || machineSessionConnectSeqRef.current !== connectSeq) return
        lastConnectionStatus = status
        const snapshot = connectionSnapshotFromStatus({ machineId, statusText: status })
        if (showConnectionProgress || snapshot.phase === 'failed' || snapshot.phase === 'reconnecting' || snapshot.phase === 'waiting_network') {
          updateFromConnectionState(snapshot)
        }
      },
    }).then((session) => {
      window.clearTimeout(progressTimer)
      if (cancelled || machineSessionConnectSeqRef.current !== connectSeq) return
      setError(null)
      setConnectionFailure(null)
      setHasConnectedOnce(true)
      hasConnectedOnceRef.current = true
      setConnectedSession(session)
      setConnectedTerminalId(activeTerminalId)
      setConnectingTerminalId(null)
      if (showConnectionProgress) {
        updateConnectionStatus(connectionPhaseLabel('connected', t), 'connected')
        clearConnectionStatusSoon()
      } else {
        clearConnectionStatus()
      }
    }).catch((err: unknown) => {
      window.clearTimeout(progressTimer)
      if (!cancelled && machineSessionConnectSeqRef.current === connectSeq) {
        if (isCancelledConnectionError(err)) {
          setError(null)
          setConnectionFailure(null)
          setConnectingTerminalId(null)
          clearConnectionStatus()
          return
        }
        const failure = connectionFailurePresentation(err, t, { phoneOnline: phoneOnlineRef.current })
        const message = failure.message
        const authFailure = isAuthorizationConnectionError(err)
        if (authFailure) handleConnectionAuthFailure(machineId)
        setConnectedSession(null)
        setConnectedTerminalId(null)
        setConnectingTerminalId(null)
        setError(message)
        setConnectionFailure(failure)
        updateConnectionStatus(message, 'failed')
      }
    })
    return () => {
      cancelled = true
      window.clearTimeout(progressTimer)
    }
  }, [activeTerminalId, clearConnectionStatus, clearConnectionStatusSoon, connector, connectionRetryToken, ensureMachineSession, forceRelayConnection, handleConnectionAuthFailure, machine?.machineId, page, releaseMachineSession, t, updateConnectionStatus, updateFromConnectionState])

  useEffect(() => {
    if (manualReconnectNonce === 0 || handledManualReconnectNonceRef.current === manualReconnectNonce) return
    if (page !== 'terminal-list') {
      handledManualReconnectNonceRef.current = manualReconnectNonce
      return
    }
    const machineId = machine?.machineId
    if (!machineId || requireVerification || !phoneOnlineRef.current || !connectionReadyRef.current) return
    handledManualReconnectNonceRef.current = manualReconnectNonce
    let cancelled = false
    void ensureMachineSession(machineId, {
      forceRelay: forceRelayConnection,
      onConnectionState: (snapshot) => {
        if (!cancelled) updateFromConnectionState(snapshot)
      },
      onStatus: (status) => {
        if (!cancelled) updateConnectionStatus(status, 'connecting')
      },
    }).then((session) => {
      if (cancelled) return
      setError(null)
      setConnectedSession(session)
      // 列表页重连只替换 session generation，不会产生 terminal inventory event；
      // 成功后必须通过现有 API 重新读取 daemon truth，不能继续展示旧缓存的“0 个”。
      void refreshTerminals()
      updateFromConnectionState({
        machineId,
        phase: 'connected',
        statusText: connectionPhaseLabel('connected', t),
        relayInUse: false,
      }, session)
    }).catch((err: unknown) => {
      if (cancelled) return
      if (isCancelledConnectionError(err)) {
        setError(null)
        setConnectionFailure(null)
        clearConnectionStatus()
        return
      }
      const failure = connectionFailurePresentation(err, t, { phoneOnline: phoneOnlineRef.current })
      const message = failure.message
      if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(machineId)
      setConnectedSession(null)
      setConnectedTerminalId(null)
      setConnectingTerminalId(null)
      setConnectionFailure(failure)
      updateFromConnectionState({
        machineId,
        phase: 'failed',
        statusText: message,
        relayInUse: false,
        error: err instanceof Error ? err : new Error(message),
      })
      setConnectionFailure(failure)
    })
    return () => {
      cancelled = true
    }
  }, [clearConnectionStatus, ensureMachineSession, forceRelayConnection, handleConnectionAuthFailure, machine?.machineId, manualReconnectNonce, page, refreshTerminals, requireVerification, t, updateConnectionStatus, updateFromConnectionState])

  useEffect(() => {
    const handleResume = (event: Event) => {
      const revision = event instanceof CustomEvent && Number.isSafeInteger(event.detail?.revision)
        ? Number(event.detail.revision)
        : null
      if (revision !== null) {
        if (revision <= handledNativeResumeRevisionRef.current) return
        handledNativeResumeRevisionRef.current = revision
      }
      resetKeyboardLayout()
      // Native generation recovery owns reconnect while connectionReady is false.
      // The ready transition below will start exactly one replacement attempt.
      if (!connectionReadyRef.current) return
      if (page === 'terminal-list') {
        // terminal list 也承载文件管理器。generation 更换后必须先让既有重连状态机取得新的
        // workspace session lease，再从 daemon 刷新 inventory；只刷新列表会让文件页永久等待旧 lease。
        setManualReconnectNonce((value) => value + 1)
        return
      }
      if (page !== 'terminal') return
      const session = machineSessionRef.current?.session ?? connectedSession
      if (!activeTerminalId && splitTerminalIds.length === 0) return
      if (!session || !isProtoSessionAlive(session)) {
        setManualReconnectNonce((value) => value + 1)
        return
      }

      setConnectedSession(session)
      setError(null)
      setConnectionFailure(null)
      fitDisplayedTerminals()
    }
    const handleSessionInvalidated = () => {
      resetKeyboardLayout()
      if (page === 'terminal-list') {
        setManualReconnectNonce((value) => value + 1)
        return
      }
      if (page !== 'terminal') return
      const session = machineSessionRef.current?.session ?? connectedSession
      if (!activeTerminalId && splitTerminalIds.length === 0) return
      if (!session || !isProtoSessionAlive(session)) {
        setManualReconnectNonce((value) => value + 1)
        return
      }
      reattachActiveTerminals(session)
    }
    document.addEventListener('anytty:resume', handleResume)
    document.addEventListener('anytty:binding-closed', handleSessionInvalidated)
    document.addEventListener('anytty:session-invalidated', handleSessionInvalidated)
    return () => {
      document.removeEventListener('anytty:resume', handleResume)
      document.removeEventListener('anytty:binding-closed', handleSessionInvalidated)
      document.removeEventListener('anytty:session-invalidated', handleSessionInvalidated)
    }
  }, [activeTerminalId, connectedSession, fitDisplayedTerminals, page, reattachActiveTerminals, resetKeyboardLayout, splitTerminalIds.length])

  const activateWebTerminalTab = useCallback((terminalId: string, paneKey?: TerminalPaneKey) => {
    const tab = webTerminalTabs.find((candidate) => candidate.terminalId === terminalId)
    if (!tab) return
    setWebTerminalTabs((current) => activeTerminalId && current.some((candidate) => candidate.terminalId === activeTerminalId)
      ? updateWebTerminalTab(current, activeTerminalId, { root: terminalSplitRoot, activePaneKey: activeTerminalSlot })
      : current)
    const nextPaneKey = paneKey && terminalPaneKeys(tab.root).includes(paneKey) ? paneKey : tab.activePaneKey
    setActiveTerminalId(tab.terminalId)
    setTerminalSplitRoot(tab.root)
    setActiveTerminalSlot(nextPaneKey)
    setPage('terminal')
    setMobileSheet(null)
    window.setTimeout(() => terminalHandleForPane(nextPaneKey)?.focus(), 0)
  }, [activeTerminalId, activeTerminalSlot, terminalHandleForPane, terminalSplitRoot, webTerminalTabs])

  useEffect(() => {
    if (!webLayout || !activeTerminalId) return
    setWebTerminalTabs((current) => current.some((tab) => tab.terminalId === activeTerminalId)
      ? updateWebTerminalTab(current, activeTerminalId, {
          root: terminalSplitRoot,
          activePaneKey: activeTerminalSlot,
        })
      : current)
  }, [activeTerminalId, activeTerminalSlot, terminalSplitRoot, webLayout])

  const openTerminal = useCallback((intent: { machineId: string; terminalId: string }) => {
    if (requireVerification) {
      handleConnectionAuthFailure(intent.machineId)
      return
    }
    if (machine && intent.machineId !== machine.machineId) {
      setError(t('workspace.terminalMachineMismatch'))
      return
    }
    if (connectionSessionUnavailable) return
    if (webLayout) {
      const existing = findWebTerminalTab(webTerminalTabs, intent.terminalId)
      if (existing) {
        activateWebTerminalTab(existing.tab.terminalId, existing.paneKey)
        return
      }
      const tab = createWebTerminalTab(intent.terminalId)
      setWebTerminalTabs((current) => {
        const saved = activeTerminalId && current.some((candidate) => candidate.terminalId === activeTerminalId)
          ? updateWebTerminalTab(current, activeTerminalId, { root: terminalSplitRoot, activePaneKey: activeTerminalSlot })
          : current
        return [...saved, tab]
      })
      setActiveTerminalId(intent.terminalId)
      setTerminalSplitRoot(tab.root)
      setActiveTerminalSlot(tab.activePaneKey)
      setPage('terminal')
      setMobileSheet(null)
      return
    }
    const existingPaneKey = terminalPaneKey(intent.terminalId)
    if (displayedPaneKeys.includes(existingPaneKey)) {
      setActiveTerminalSlot(existingPaneKey)
      setPage('terminal')
      setMobileSheet(null)
      window.setTimeout(() => terminalHandleForPane(existingPaneKey)?.focus(), 0)
      return
    }
    setActiveTerminalId(intent.terminalId)
    setActiveTerminalSlot('primary')
    setPage('terminal')
    setMobileSheet(null)
  }, [activeTerminalId, activeTerminalSlot, activateWebTerminalTab, connectionSessionUnavailable, displayedPaneKeys, handleConnectionAuthFailure, machine, requireVerification, t, terminalHandleForPane, terminalSplitRoot, webLayout, webTerminalTabs])

  const handledInitialTerminalRef = useRef<string | null>(null)
  useEffect(() => {
    if (!initialTerminalId || !machine || !hasLoadedTerminals || connectionSessionUnavailable) return
    const intentKey = `${machine.machineId}:${initialTerminalId}`
    if (handledInitialTerminalRef.current === intentKey) return
    if (!terminals.some((terminal) => terminal.terminalId === initialTerminalId)) {
      handledInitialTerminalRef.current = intentKey
      onInitialTerminalOpened?.(machine.machineId, initialTerminalId)
      return
    }
    handledInitialTerminalRef.current = intentKey
    openTerminal({ machineId: machine.machineId, terminalId: initialTerminalId })
    onInitialTerminalOpened?.(machine.machineId, initialTerminalId)
  }, [connectionSessionUnavailable, hasLoadedTerminals, initialTerminalId, machine, onInitialTerminalOpened, openTerminal, terminals])

  const switcherMachines = useMemo<MachineWorkspaceSwitcherMachine[]>(() => {
    if (!machine) return []
    const current: MachineWorkspaceSwitcherMachine = {
      machineId: machine.machineId,
      name: machine.name,
      state: machine.state,
      terminalCount: terminals.length,
    }
    const seen = new Set([machine.machineId])
    return [current, ...terminalSwitcherMachines.filter((candidate) => {
      if (seen.has(candidate.machineId)) return false
      seen.add(candidate.machineId)
      return true
    })]
  }, [machine, terminalSwitcherMachines, terminals.length])

  const loadTerminalSwitcherMachine = useCallback((machineId: string) => {
    if (!loadMachineTerminals) return
    setTerminalSwitcherInventoryByMachine((current) => ({
      ...current,
      [machineId]: { status: 'loading', terminals: current[machineId]?.terminals ?? [] },
    }))
    void loadMachineTerminals(machineId).then((loaded) => {
      setTerminalSwitcherInventoryByMachine((current) => ({
        ...current,
        [machineId]: { status: 'ready', terminals: loaded.filter((terminal) => terminal.machineId === machineId) },
      }))
    }).catch(() => {
      setTerminalSwitcherInventoryByMachine((current) => ({
        ...current,
        [machineId]: { status: 'error', terminals: current[machineId]?.terminals ?? [] },
      }))
    })
  }, [loadMachineTerminals])

  const toggleTerminalSwitcherMachine = useCallback((machineId: string) => {
    const willExpand = expandedTerminalSwitcherMachineId !== machineId
    setExpandedTerminalSwitcherMachineId(willExpand ? machineId : null)
    if (!willExpand || machineId === machine?.machineId || terminalSwitcherInventoryByMachine[machineId]?.status === 'ready' || terminalSwitcherInventoryByMachine[machineId]?.status === 'loading') return
    loadTerminalSwitcherMachine(machineId)
  }, [expandedTerminalSwitcherMachineId, loadTerminalSwitcherMachine, machine?.machineId, terminalSwitcherInventoryByMachine])

  useEffect(() => {
    setExpandedTerminalSwitcherMachineId(machine?.machineId ?? null)
    setTerminalSwitcherInventoryByMachine({})
  }, [machine?.machineId])

  const selectSplitTerminal = useCallback((intent: { machineId: string; terminalId: string }, target: WebPaneDropTarget = 'bottom', targetPaneKey: TerminalPaneKey = activeTerminalSlot) => {
    if (connectionSessionUnavailable) return
    if (machine && intent.machineId !== machine.machineId) {
      setError(t('workspace.terminalMachineMismatch'))
      return
    }
    if (!activeTerminalId || intent.terminalId === terminalIdForPane(targetPaneKey, activeTerminalId)) {
      setPairStatus(t('workspace.chooseDifferentTerminal'))
      return
    }
    const direction = target === 'left' || target === 'right' ? 'columns' : 'rows'
    const placement = target === 'left' || target === 'top' ? 'before' : 'after'
    terminalSplitSequenceRef.current += 1
    const splitId = `split-${terminalSplitSequenceRef.current}`
    const paneKey = terminalPaneKey(intent.terminalId)
    setTerminalSplitRoot((current) => splitTerminalPane(current, targetPaneKey, intent.terminalId, direction, splitId, placement))
    setTerminalResizeControlBySlot((current) => ({ ...current, [paneKey]: current[paneKey] ?? defaultTerminalResizeControl }))
    setActiveTerminalSlot(paneKey)
    setMobileSheet(null)
    window.setTimeout(() => {
      fitDisplayedTerminals()
      splitTerminalRefs.current.get(intent.terminalId)?.focus()
    }, 0)
  }, [activeTerminalId, activeTerminalSlot, connectionSessionUnavailable, fitDisplayedTerminals, machine, t])

  const splitActiveTerminal = useCallback((target: WebPaneDropTarget = 'bottom') => {
    if (requireVerification) {
      handleConnectionAuthFailure(machine?.machineId)
      return
    }
    if (connectionSessionUnavailable) return
    if (!activeTerminalId || !machine) {
      setError(t('workspace.openBeforeSplit'))
      return
    }
    const displayed = new Set([activeTerminalId, ...splitTerminalIds])
    const candidate = orderedTerminals.find((terminal) => !displayed.has(terminal.terminalId))
    if (!candidate) {
      setPairStatus(t('workspace.noOtherTerminal'))
      return
    }
    setTerminalToolbarOpen(false)
    setTerminalFnOpen(false)
    selectSplitTerminal({ machineId: machine.machineId, terminalId: candidate.terminalId }, target)
  }, [activeTerminalId, connectionSessionUnavailable, handleConnectionAuthFailure, machine, orderedTerminals, requireVerification, selectSplitTerminal, splitTerminalIds, t])

  const removeSplitTerminal = useCallback((terminalId: string) => {
    const paneKey = terminalPaneKey(terminalId)
    setTerminalSplitRoot((current) => removeTerminalPane(current, paneKey) ?? PRIMARY_TERMINAL_PANE)
    if (activeTerminalSlot === paneKey) setActiveTerminalSlot('primary')
    setSyncSplitInput(false)
    window.setTimeout(() => {
      fitDisplayedTerminals()
      terminalRef.current?.focus()
    }, 0)
  }, [activeTerminalSlot, fitDisplayedTerminals])

  const closeSplitTerminal = useCallback(() => {
    const terminalId = activeTerminalSlot === 'primary'
      ? splitTerminalIds.at(-1)
      : terminalIdForPane(activeTerminalSlot, activeTerminalId)
    if (terminalId) removeSplitTerminal(terminalId)
  }, [activeTerminalId, activeTerminalSlot, removeSplitTerminal, splitTerminalIds])

  const activeTerminalHandle = useCallback(() => {
    return terminalHandleForPane(activeTerminalSlot)
  }, [activeTerminalSlot, terminalHandleForPane])

  const sendTerminalInput = useCallback((data: string): boolean => {
    if (connectionInputBlocked) return false
    if (syncSplitInput && hasSplitTerminals) {
      let accepted = false
      forEachDisplayedTerminalHandle((handle) => { accepted = handle.sendInput(data) || accepted })
      return accepted
    }
    return activeTerminalHandle()?.sendInput(data) ?? false
  }, [activeTerminalHandle, connectionInputBlocked, forEachDisplayedTerminalHandle, hasSplitTerminals, syncSplitInput])

  const pasteTerminalText = useCallback((text: string): boolean => {
    if (connectionInputBlocked) return false
    if (syncSplitInput && hasSplitTerminals) {
      let accepted = false
      forEachDisplayedTerminalHandle((handle) => { accepted = handle.pasteText(text) || accepted })
      return accepted
    }
    return activeTerminalHandle()?.pasteText(text) ?? false
  }, [activeTerminalHandle, connectionInputBlocked, forEachDisplayedTerminalHandle, hasSplitTerminals, syncSplitInput])

  const handleTerminalBufferChange = useCallback((slot: TerminalPaneKey, isAlternate: boolean) => {
    setTerminalBufferBySlot((current) => {
      const nextBuffer = isAlternate ? 'alternate' : 'normal'
      if (current[slot] === nextBuffer) return current
      return { ...current, [slot]: nextBuffer }
    })
    handleBufferChange(isAlternate)
  }, [handleBufferChange])

  const focusActiveTerminal = useCallback(() => {
    markKeyboardVisible()
    activeTerminalHandle()?.focus()
  }, [activeTerminalHandle, markKeyboardVisible])

  const blurActiveTerminal = useCallback(() => {
    activeTerminalHandle()?.blur()
    markKeyboardHidden()
  }, [activeTerminalHandle, markKeyboardHidden])

  useEffect(() => {
    const available = new Set(terminals.map((terminal) => terminal.terminalId))
    const invalid = splitTerminalIds.filter((terminalId) => terminalId === activeTerminalId || !available.has(terminalId))
    if (invalid.length === 0) return
    setTerminalSplitRoot((current) => invalid.reduce<TerminalSplitNode>(
      (next, terminalId) => removeTerminalPane(next, terminalPaneKey(terminalId)) ?? PRIMARY_TERMINAL_PANE,
      current,
    ))
    if (activeTerminalSlot !== 'primary' && invalid.includes(terminalIdForPane(activeTerminalSlot, activeTerminalId) ?? '')) {
      setActiveTerminalSlot('primary')
    }
    setSyncSplitInput(false)
  }, [activeTerminalId, activeTerminalSlot, splitTerminalIds, terminals])

  const retryConnection = useCallback(async (options: { closeDialog?: boolean; preservePolicy?: boolean } = {}) => {
    const targetForceRelay = options.preservePolicy ? undefined : forceRelayConnection
    setForceRelayConnection(targetForceRelay)
    if (options.closeDialog !== false) setConnectionInfoOpen(false)
    setConnectionInfo(null)
    setConnectionInfoError(null)
    updateConnectionStatus(targetForceRelay
      ? t('workspace.reconnectingRelay')
      : connectionPhaseLabel('reconnecting', t), 'reconnecting')
    if (connector.reconnect) {
      const current = machineSessionRef.current
      await connector.reconnect({ forceRelay: targetForceRelay })
      machineSessionConnectSeqRef.current += 1
      connectionStateSubscriptionRef.current?.close()
      connectionStateSubscriptionRef.current = null
      machineSessionPromiseRef.current = null
      machineSessionRef.current = null
      const runtimeInventorySubscription = runtimeInventorySubscriptionRef.current
      runtimeInventorySubscriptionRef.current = null
      runtimeInventorySubscription?.subscription.close()
      if (current) void closeMachineWorkspaceSession(current.session)
      setConnectedSession(null)
      setConnectedTerminalId(null)
      setConnectingTerminalId(activeTerminalId)
    } else {
      releaseMachineSession()
      setConnectedSession(null)
      setConnectedTerminalId(null)
      setConnectingTerminalId(activeTerminalId)
    }
    setManualReconnectNonce((value) => value + 1)
    setConnectionRetryToken((value) => value + 1)
  }, [activeTerminalId, connector, forceRelayConnection, releaseMachineSession, t, updateConnectionStatus])

  const retryAfterFailure = useCallback(() => {
    if (connectionRetryPromiseRef.current) return connectionRetryPromiseRef.current
    setError(null)
    setConnectionFailure(null)
    setConnectionRetryPending(true)
    const retry = retryConnection().catch((err: unknown) => {
      if (isCancelledConnectionError(err)) {
        setError(null)
        setConnectionFailure(null)
        clearConnectionStatus()
        return
      }
      const failure = connectionFailurePresentation(err, t, { phoneOnline: phoneOnlineRef.current })
      setError(failure.message)
      setConnectionFailure(failure)
      updateConnectionStatus(failure.message, 'failed')
    }).finally(() => {
      if (connectionRetryPromiseRef.current === retry) connectionRetryPromiseRef.current = null
      setConnectionRetryPending(false)
    })
    connectionRetryPromiseRef.current = retry
    return retry
  }, [clearConnectionStatus, retryConnection, t, updateConnectionStatus])

  const machineConnectionOverlayIntent = useMemo<ConnectionRecoveryOverlayIntent | null>(() => {
    if (!phoneOnline || !connectionReady) return null
    if (hasConnectedOnce && (connectionFailure || connectionPhase === 'failed')) {
      const title = connectionFailure?.title ?? t('machines.connectionFailed')
      const description = connectionFailure?.message ?? t('errors.connectionInterrupted')
      const machineId = machine?.machineId
      const action = connectionFailure?.requiresPairing && machineId && onNeedsReauthorization
        ? {
            label: t('machines.scanPairing'),
            onClick: () => handleConnectionAuthFailure(machineId),
          }
        : !connectionFailure || connectionFailure.retryable || connectionFailure.reason === 'daemon_deleted'
          ? {
              label: t(connectionFailure?.reason === 'daemon_deleted' ? 'workspace.retryOtherRoutes' : 'workspace.connection.retry'),
              onClick: () => { void retryAfterFailure() },
              pending: connectionRetryPending,
            }
          : undefined
      return { kind: 'failed', title, description, ...(action ? { action } : {}) }
    }
    if (!showDelayedMachineNetworkOverlay) return null
    return {
      kind: connectionPhase === 'waiting_network' ? 'offline' : 'recovering',
      title: displayedConnectionStatus || t('connectionStatus.recovering'),
      ...(hasConnectedOnce ? { description: t('connectionStatus.inputPaused') } : {}),
    }
  }, [connectionFailure, connectionPhase, connectionReady, connectionRetryPending, displayedConnectionStatus, handleConnectionAuthFailure, hasConnectedOnce, machine?.machineId, onNeedsReauthorization, phoneOnline, retryAfterFailure, showDelayedMachineNetworkOverlay, t])
  useConnectionRecoveryOverlay(machineConnectionOverlayIntent)

  useEffect(() => {
    const wasPhoneOnline = previousPhoneOnlineRef.current
    const previousConnectionState = previousConnectionStateRef.current
    previousPhoneOnlineRef.current = phoneOnline
    previousConnectionStateRef.current = connectionState
    if (!phoneOnline || connectionState !== 'ready') return
    const appRecoveryCompleted = previousConnectionState !== 'ready' && !connectionStateEvents
    const browserNetworkRecovered = !wasPhoneOnline && phoneOnline && !connectionStateEvents
    if (appRecoveryCompleted || browserNetworkRecovered) {
      retryAfterFailure()
    }
  }, [connectionState, connectionStateEvents, phoneOnline, retryAfterFailure])

  useEffect(() => {
    if (!pairStatus) return
    const timer = setTimeout(() => setPairStatus(null), pairStatus === resizeLockedHint ? 1800 : 3000)
    return () => clearTimeout(timer)
  }, [pairStatus, resizeLockedHint])

  useEffect(() => {
    if (!(activeTerminalResizeControl.sizeLocked || activeTerminalResizeControl.reason === 'size_locked')) return
    if (resizeLockedHintShownRef.current) return
    resizeLockedHintShownRef.current = true
    setPairStatus(resizeLockedHint)
  }, [activeTerminalResizeControl.reason, activeTerminalResizeControl.sizeLocked, resizeLockedHint])

  const showTerminalListPage = useCallback(() => {
    setPage('terminal-list')
    setMobileSheet(null)
    setTerminalSplitRoot(PRIMARY_TERMINAL_PANE)
    setActiveTerminalSlot('primary')
    setSyncSplitInput(false)
    resetKeyboardLayout()
    forEachDisplayedTerminalHandle((handle) => handle.adjustInputPosition(0))
  }, [forEachDisplayedTerminalHandle, resetKeyboardLayout])

  useEffect(() => {
    if (!webLayout || !hasLoadedTerminals) return
    const available = new Set(terminals.map((terminal) => terminal.terminalId))
    const next = sanitizeWebTerminalTabs(webTerminalTabs, available)
    if (next !== webTerminalTabs) setWebTerminalTabs(next)
    if (activeTerminalId && next.some((tab) => tab.terminalId === activeTerminalId)) return
    const fallback = next[0]
    if (fallback) {
      setActiveTerminalId(fallback.terminalId)
      setTerminalSplitRoot(fallback.root)
      setActiveTerminalSlot(fallback.activePaneKey)
      return
    }
    setActiveTerminalId(null)
    setTerminalSplitRoot(PRIMARY_TERMINAL_PANE)
    setActiveTerminalSlot('primary')
    setPage('terminal-list')
  }, [activeTerminalId, hasLoadedTerminals, terminals, webLayout, webTerminalTabs])

  const reorderWebTabs = useCallback((terminalId: string, targetTerminalId: string, placement: 'before' | 'after') => {
    setWebTerminalTabs((current) => {
      if (terminalId === targetTerminalId || !current.some((tab) => tab.terminalId === terminalId) || !current.some((tab) => tab.terminalId === targetTerminalId)) return current
      const moving = current.find((tab) => tab.terminalId === terminalId)!
      const next = current.filter((tab) => tab.terminalId !== terminalId)
      const targetIndex = next.findIndex((tab) => tab.terminalId === targetTerminalId)
      next.splice(targetIndex + (placement === 'after' ? 1 : 0), 0, moving)
      return next
    })
  }, [])

  const closeWebTab = useCallback((terminalId: string) => {
    const closingIndex = webTerminalTabs.findIndex((tab) => tab.terminalId === terminalId)
    const remaining = webTerminalTabs.filter((tab) => tab.terminalId !== terminalId)
    setWebTerminalTabs(remaining)
    if (activeTerminalId !== terminalId) return
    const candidate = remaining[Math.min(Math.max(closingIndex, 0), remaining.length - 1)] ?? remaining.at(-1)
    if (candidate) {
      setActiveTerminalId(candidate.terminalId)
      setTerminalSplitRoot(candidate.root)
      setActiveTerminalSlot(candidate.activePaneKey)
      setPage('terminal')
      return
    }
    setActiveTerminalId(null)
    setTerminalSplitRoot(PRIMARY_TERMINAL_PANE)
    setActiveTerminalSlot('primary')
    setPage('terminal-list')
  }, [activeTerminalId, webTerminalTabs])

  const handleWebPaneDrop = useCallback((terminalId: string, targetPaneKey: TerminalPaneKey, target: WebPaneDropTarget) => {
    setWebDraggedTerminalId(null)
    if (!machine) return
    selectSplitTerminal({ machineId: machine.machineId, terminalId }, target, targetPaneKey)
  }, [machine, selectSplitTerminal])

  const fitWebSplitTerminals = useCallback(() => {
    window.requestAnimationFrame(fitDisplayedTerminals)
  }, [fitDisplayedTerminals])

  useEffect(() => {
    if (!webLayout || !hasSplitTerminals) return
    fitWebSplitTerminals()
  }, [fitWebSplitTerminals, hasSplitTerminals, terminalSplitRoot, webLayout])

  useEffect(() => {
    if (!webLayout) return
    fitWebSplitTerminals()
  }, [fitWebSplitTerminals, webLayout, webTerminalSidebarOpen])

  useEffect(() => {
    if (!webLayout) return
    const handleWebTerminalPickerShortcut = (event: globalThis.KeyboardEvent) => {
      if ((!event.ctrlKey && !event.metaKey) || event.altKey || event.shiftKey || event.key.toLowerCase() !== 'f') return
      const target = event.target instanceof Element ? event.target : null
      if (!webTerminalPickerOpen && target?.closest('[role="dialog"], [aria-modal="true"]')) return
      event.preventDefault()
      event.stopPropagation()
      if (webTerminalPickerOpen) {
        document.querySelector<HTMLInputElement>('[data-testid="anytty-web-terminal-picker-input"]')?.focus({ preventScroll: true })
        return
      }
      setWebTerminalPickerOpen(true)
    }
    window.addEventListener('keydown', handleWebTerminalPickerShortcut, true)
    return () => window.removeEventListener('keydown', handleWebTerminalPickerShortcut, true)
  }, [webLayout, webTerminalPickerOpen])

  useEffect(() => {
    if (!webLayout) return
    const handleWebTabShortcut = (event: globalThis.KeyboardEvent) => {
      if (!event.altKey || event.ctrlKey || event.metaKey || event.shiftKey) return
      const digit = Number(event.key)
      if (!Number.isInteger(digit) || digit < 1 || digit > 9) return
      const terminalId = webTabTerminalIds[digit - 1]
      if (!terminalId || !machine) return
      event.preventDefault()
      activateWebTerminalTab(terminalId)
    }
    window.addEventListener('keydown', handleWebTabShortcut)
    return () => window.removeEventListener('keydown', handleWebTabShortcut)
  }, [activateWebTerminalTab, machine, webLayout, webTabTerminalIds])

  const openFiles = useCallback(() => {
    if (requireVerification) {
      handleConnectionAuthFailure(machine?.machineId)
      return
    }
    if (!machine) {
      setError(t('workspace.fileAccessNotReady'))
      return
    }
    fileReturnPageRef.current = page
    const fileTerminal = page === 'terminal' ? activeToolTerminal : (activeToolTerminal ?? terminals[0] ?? null)
    const fallbackPath = fileTerminal?.cwd || '/'
    const resolveTerminalDirectory = page === 'terminal' && Boolean(fileTerminal)
    const contextScope = page === 'terminal' ? 'terminal' : 'list'
    const nextContextKey = `${contextScope}:${fileTerminal?.terminalId ?? 'machine'}:${fallbackPath}`
    setFileTerminalId(fileTerminal?.terminalId ?? null)
    setFileInitialPath(fallbackPath)
    if (!filesOpen && fileContextKey !== nextContextKey) setFileContextKey(nextContextKey)
    void ensureMachineSession(machine.machineId, {
      forceRelay: forceRelayConnection,
      onConnectionState: updateFromConnectionState,
      onStatus: (status) => updateConnectionStatus(status),
    })
      .then(async (session) => {
        if (!resolveTerminalDirectory || !fileTerminal) return
        try {
          const directory = await createTerminalManagementApi(session, machine.machineId)
            .getTerminalDirectory(fileTerminal.terminalId)
          const livePath = normalizeTerminalDirectory(directory.path)
          if (livePath) {
            setFileInitialPath(livePath)
            const liveContextKey = `${contextScope}:${fileTerminal.terminalId}:${livePath}`
            if (!filesOpen && fileContextKey !== liveContextKey) setFileContextKey(liveContextKey)
          }
        } catch {
          setFileInitialPath(fallbackPath)
          if (!filesOpen && fileContextKey !== nextContextKey) setFileContextKey(nextContextKey)
        }
      })
      .catch((err: unknown) => {
        const message = connectionErrorDisplayMessage(err, t)
        if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(machine.machineId)
        updateConnectionStatus(message, 'failed')
        setFilesOpen(false)
        setFileTerminalId(null)
      })
    setHasOpenedFiles(true)
    startFileManagerLoad()
    setFilesOpen(true)
    setMobileSheet(null)
  }, [activeToolTerminal, ensureMachineSession, fileContextKey, filesOpen, forceRelayConnection, handleConnectionAuthFailure, machine, page, requireVerification, startFileManagerLoad, t, terminals, updateConnectionStatus, updateFromConnectionState])

  const openManageTerminal = useCallback((intent: { machineId: string; terminalId: string }) => {
    if (requireVerification) {
      handleConnectionAuthFailure(intent.machineId)
      return
    }
    if (!canManageTerminals) return
    if (machine && intent.machineId !== machine.machineId) {
      setError(t('workspace.terminalMachineMismatch'))
      return
    }
    setSelectedTerminalId(intent.terminalId)
    setMobileSheet('manage-terminal')
  }, [canManageTerminals, handleConnectionAuthFailure, machine, requireVerification, t])

  const openCreateTerminal = useCallback(() => {
    if (requireVerification) {
      handleConnectionAuthFailure(machine?.machineId)
      return
    }
    if (!canManageTerminals) return
    setSelectedTerminalId(null)
    setTerminalSubmitError(null)
    setTerminalForm({
      name: '',
      command: '',
      cwd: '',
      environment: [],
      sizeLockMode: 'off',
    })
    const request = terminalDefaultsRequestRef.current + 1
    terminalDefaultsRequestRef.current = request
    setTerminalDefaultsLoading(true)
    setMobileSheet('create-terminal')
    void withManagementApi().then(({ api: management }) => management.getDefaults()).then((defaults) => {
      if (terminalDefaultsRequestRef.current !== request) return
      setTerminalForm((current) => ({
        ...current,
        command: current.command || formatCommandLine(defaults.command),
        cwd: current.cwd || defaults.cwd,
      }))
    }).catch(() => {
      if (terminalDefaultsRequestRef.current !== request) return
      setTerminalSubmitError(t('errors.generic'))
    }).finally(() => {
      if (terminalDefaultsRequestRef.current === request) setTerminalDefaultsLoading(false)
    })
  }, [canManageTerminals, handleConnectionAuthFailure, machine?.machineId, requireVerification, t, withManagementApi])

  const openEditTerminal = useCallback(() => {
    if (!selectedTerminal) return
    setTerminalSubmitError(null)
    setTerminalForm({
      name: selectedTerminal.title,
      command: selectedTerminal.command ?? '',
      cwd: selectedTerminal.cwd ?? '',
      environment: [],
      sizeLockMode: selectedTerminal.sizeLockMode ?? 'off',
    })
    setMobileSheet('edit-terminal')
  }, [selectedTerminal])

  const selectTerminalWorkingDirectory = useCallback((path: string) => {
    setTerminalForm((current) => ({ ...current, cwd: normalizeFilePath(path) }))
    setMobileSheet(terminalPathReturnSheet)
  }, [terminalPathReturnSheet])

  const loadTerminalPathPicker = useCallback(async (path: string) => {
    const normalizedPath = normalizeDirectoryPickerPath(path)
    setTerminalPathPickerPath(normalizedPath)
    setTerminalPathPickerLoading(true)
    setTerminalPathPickerError(null)
    try {
      const session = await withMachineSession()
      const response = await createFileApi(session).listDir(normalizedPath)
      setTerminalPathPickerPath(response.path || normalizedPath)
      setTerminalPathPickerEntries(response.entries.filter(isDirectoryEntry))
    } catch (err) {
      setTerminalPathPickerEntries([])
      setTerminalPathPickerError(t('errors.generic'))
    } finally {
      setTerminalPathPickerLoading(false)
    }
  }, [t, withMachineSession])

  const openTerminalPathPicker = useCallback(() => {
    const returnSheet: TerminalEditorSheet = mobileSheet === 'edit-terminal' ? 'edit-terminal' : 'create-terminal'
    const startPath = normalizeTerminalDirectory(terminalForm.cwd) || normalizeTerminalDirectory(activeToolTerminal?.cwd) || '/'
    setTerminalPathReturnSheet(returnSheet)
    setMobileSheet('terminal-path-picker')
    void loadTerminalPathPicker(startPath)
  }, [activeToolTerminal?.cwd, loadTerminalPathPicker, mobileSheet, terminalForm.cwd])

  const loadTerminalPathBookmarks = useCallback(async () => {
    setTerminalPathBookmarksLoading(true)
    setTerminalPathBookmarksError(null)
    try {
      const session = await withMachineSession()
      setTerminalPathBookmarks(await createPersistentPathBookmarkApi(machine?.machineId ?? '', session, storage).list())
    } catch (err) {
      setTerminalPathBookmarksError(t('errors.generic'))
    } finally {
      setTerminalPathBookmarksLoading(false)
    }
  }, [machine?.machineId, storage, t, withMachineSession])

  const openTerminalPathBookmarks = useCallback(() => {
    const returnSheet: TerminalEditorSheet = mobileSheet === 'edit-terminal' ? 'edit-terminal' : 'create-terminal'
    setTerminalPathReturnSheet(returnSheet)
    setMobileSheet('terminal-path-bookmarks')
    void loadTerminalPathBookmarks()
  }, [loadTerminalPathBookmarks, mobileSheet])

  const addTerminalPathBookmark = useCallback(async () => {
    const path = normalizeTerminalDirectory(terminalForm.cwd) || normalizeTerminalDirectory(activeToolTerminal?.cwd) || '/'
    setTerminalPathBookmarksError(null)
    try {
      const session = await withMachineSession()
      await createPersistentPathBookmarkApi(machine?.machineId ?? '', session, storage).add(path)
      setPairStatus(t('workspace.pathBookmarked', { path }))
      await loadTerminalPathBookmarks()
    } catch (err) {
      setTerminalPathBookmarksError(t('errors.generic'))
    }
  }, [activeToolTerminal?.cwd, loadTerminalPathBookmarks, machine?.machineId, storage, t, terminalForm.cwd, withMachineSession])

  const removeTerminalPathBookmark = useCallback(async (id: string) => {
    setTerminalPathBookmarksError(null)
    try {
      const session = await withMachineSession()
      await createPersistentPathBookmarkApi(machine?.machineId ?? '', session, storage).remove(id)
      await loadTerminalPathBookmarks()
    } catch (err) {
      setTerminalPathBookmarksError(t('errors.generic'))
    }
  }, [loadTerminalPathBookmarks, machine?.machineId, storage, t, withMachineSession])

  const submitCreateTerminal = useCallback(async () => {
    if (!canManageTerminals || terminalSubmitting || terminalDefaultsLoading) return
    setTerminalSubmitError(null)
    let command: string[]
    let environment: string[]
    try {
      command = parseCommandLine(terminalForm.command)
      if (command.length === 0) throw new Error(t('workspace.terminalForm.commandRequired'))
      environment = validateEnvironmentVariables(terminalForm.environment)
    } catch (error) {
      setTerminalSubmitError(error instanceof Error ? error.message : String(error))
      return
    }
    setTerminalSubmitting(true)
    const input: LocalCreateTerminalInput = {
      name: terminalForm.name.trim() || undefined,
      command,
      cwd: terminalForm.cwd.trim() || undefined,
      environment,
      sizeLockMode: terminalForm.sizeLockMode,
    }
    try {
      const management = await withManagementApi()
      const created = await management.api.createTerminal(input)
      await refreshTerminals()
      setPairStatus(t('workspace.terminalCreated', { name: created.terminalId || input.name || t('terminal.defaultTitle') }))
      setMobileSheet(null)
    } catch (err) {
      setTerminalSubmitError(t('errors.generic'))
    } finally {
      setTerminalSubmitting(false)
    }
  }, [canManageTerminals, refreshTerminals, t, terminalDefaultsLoading, terminalForm, terminalSubmitting, withManagementApi])

  const submitUpdateTerminal = useCallback(async () => {
    if (!canManageTerminals || !selectedTerminalId || terminalSubmitting) return
    setTerminalSubmitError(null)
    setTerminalSubmitting(true)
    const input: LocalUpdateTerminalInput = {
      terminalId: selectedTerminalId,
      name: terminalForm.name.trim() || undefined,
      cwd: terminalForm.cwd.trim() || undefined,
      sizeLockMode: terminalForm.sizeLockMode,
    }
    try {
      const management = await withManagementApi()
      await management.api.updateTerminal(input)
      await refreshTerminals()
      setPairStatus(t('workspace.terminalUpdated', { name: input.name || selectedTerminal?.title || selectedTerminalId }))
      setMobileSheet(null)
    } catch (err) {
      setTerminalSubmitError(t('errors.generic'))
    } finally {
      setTerminalSubmitting(false)
    }
  }, [canManageTerminals, selectedTerminal, selectedTerminalId, refreshTerminals, t, terminalForm, terminalSubmitting, withManagementApi])

  const toggleActiveTerminalSizeLock = useCallback(async () => {
    if (!canManageTerminals || !activeToolTerminal || sizeLockPending) return
    if (!terminalResizeControlOwnsResize(activeTerminalResizeControl)) {
      setPairStatus(t('workspace.resize.acquireFirst'))
      return
    }
    const terminalId = activeToolTerminal.terminalId
    const slot = activeTerminalSlot
    const nextLocked = !activeTerminalResizeLocked
    const terminalHandle = activeTerminalHandle()
    if (!terminalHandle) return
    setSizeLockPending(true)
    let protocolApplied = false
    let metadataApplied = false
    try {
      const control = await terminalHandle.setResizeLock(nextLocked)
      if (control.sizeLocked !== nextLocked) throw new Error('terminal resize lock was not accepted')
      protocolApplied = true
      const management = await withManagementApi()
      await management.api.updateTerminalSizeLock({
        terminalId,
        cwd: activeToolTerminal.cwd,
        sizeLockMode: nextLocked ? 'lock' : 'off',
      })
      metadataApplied = true
      setTerminals((current) => current.map((terminal) => terminal.terminalId === terminalId
        ? { ...terminal, sizeLocked: nextLocked, sizeLockMode: nextLocked ? 'lock' : 'off' }
        : terminal))
      setTerminalResizeControlBySlot((current) => ({
        ...current,
        [slot]: control,
      }))
      await refreshTerminals()
      window.setTimeout(() => {
        terminalHandle.fit()
      }, 0)
      setPairStatus(t(nextLocked ? 'workspace.resize.locked' : 'workspace.resize.unlocked'))
    } catch (err) {
      if (protocolApplied && !metadataApplied) void terminalHandle.setResizeLock(!nextLocked).catch(() => undefined)
      if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(machine?.machineId)
      else setPairStatus(t('workspace.resize.lockFailed'))
    } finally {
      setSizeLockPending(false)
    }
  }, [activeTerminalHandle, activeTerminalResizeControl, activeTerminalResizeLocked, activeTerminalSlot, activeToolTerminal, canManageTerminals, handleConnectionAuthFailure, machine?.machineId, refreshTerminals, sizeLockPending, t, withManagementApi])

  const acquireActiveResizeOwner = useCallback(async () => {
    if (resizeOwnerPending) return
    const slot = activeTerminalSlot
    setResizeOwnerPending(true)
    try {
      const control = await activeTerminalHandle()?.requestResizeOwner()
      if (control) setTerminalResizeControlBySlot((current) => ({ ...current, [slot]: control }))
      if (control?.canResize) {
        setPairStatus(t('workspace.resize.acquired'))
        window.setTimeout(() => {
          activeTerminalHandle()?.fit()
        }, 0)
      } else if (control?.sizeLocked || control?.reason === 'size_locked') {
        setPairStatus(t('workspace.resize.locked'))
      } else {
        setPairStatus(t('workspace.resize.unavailable'))
      }
    } catch (err) {
      if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(machine?.machineId)
      else setPairStatus(t('workspace.resize.acquireFailed'))
    } finally {
      setResizeOwnerPending(false)
    }
  }, [activeTerminalHandle, activeTerminalSlot, handleConnectionAuthFailure, machine?.machineId, resizeOwnerPending, t])

  const releaseActiveResizeOwner = useCallback(async () => {
    if (resizeOwnerPending) return
    const slot = activeTerminalSlot
    setResizeOwnerPending(true)
    try {
      const control = await activeTerminalHandle()?.releaseResizeOwner()
      if (control) setTerminalResizeControlBySlot((current) => ({ ...current, [slot]: control }))
      setPairStatus(t('workspace.resize.released'))
    } catch (err) {
      if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(machine?.machineId)
      else setPairStatus(t('workspace.resize.releaseFailed'))
    } finally {
      setResizeOwnerPending(false)
    }
  }, [activeTerminalHandle, activeTerminalSlot, handleConnectionAuthFailure, machine?.machineId, resizeOwnerPending, t])

  const autoAcquiredResizeOwnerRef = useRef<string | null>(null)
  useEffect(() => {
    if (!effectiveTerminalSettings.autoAcquireResizeOwner) {
      autoAcquiredResizeOwnerRef.current = null
      return
    }
    if (!activeToolTerminal || !renderSession || activeToolTerminal.state === 'exited') return
    if (activeTerminalResizeControl.canResize || activeTerminalResizeControl.sizeLocked || activeTerminalResizeControl.reason === 'size_locked') return
    if (autoAcquiredResizeOwnerRef.current === activeToolTerminal.terminalId) return
    autoAcquiredResizeOwnerRef.current = activeToolTerminal.terminalId
    void acquireActiveResizeOwner()
  }, [acquireActiveResizeOwner, activeTerminalResizeControl, activeToolTerminal, effectiveTerminalSettings.autoAcquireResizeOwner, renderSession])

  const deleteManagedTerminal = useCallback(async () => {
    if (!canManageTerminals || !selectedTerminalId) return
    const deletedTerminalId = selectedTerminalId
    const deletedTitle = selectedTerminal?.title ?? selectedTerminalId
    const management = await withManagementApi()
    await management.api.deleteTerminal(deletedTerminalId)
    if (activeTerminalId === deletedTerminalId) {
      setActiveTerminalId(null)
      setTerminalSplitRoot(PRIMARY_TERMINAL_PANE)
      setActiveTerminalSlot('primary')
      setSyncSplitInput(false)
      setPage('terminal-list')
    }
    if (fileTerminalId === deletedTerminalId) {
      setFilesOpen(false)
      setFileTerminalId(null)
    }
    if (splitTerminalIds.includes(deletedTerminalId)) removeSplitTerminal(deletedTerminalId)
    await refreshTerminals()
    setPairStatus(t('workspace.terminalDeleted', { name: deletedTitle }))
    setMobileSheet(null)
  }, [activeTerminalId, canManageTerminals, fileTerminalId, removeSplitTerminal, selectedTerminal, selectedTerminalId, refreshTerminals, splitTerminalIds, t, withManagementApi])

  const restartTerminalById = useCallback(async (terminalId: string): Promise<boolean> => {
    if (!canManageTerminals || restartingTerminalId) return false
    const restartedTitle = terminals.find((terminal) => terminal.terminalId === terminalId)?.title ?? terminalId
    setRestartingTerminalId(terminalId)
    try {
      const management = await withManagementApi()
      closeTerminalDataChannel(management.session, terminalId)
      await management.api.restartTerminal(terminalId)
      await refreshTerminals()
      if (activeTerminalId === terminalId) {
        setConnectedSession(management.session)
        setConnectedTerminalId(terminalId)
        setConnectingTerminalId(null)
        terminalRef.current?.reattach(management.session, { forceTerminalChannel: true })
        window.setTimeout(() => {
          terminalRef.current?.fit()
          terminalRef.current?.focus()
        }, 0)
      }
      if (splitTerminalIds.includes(terminalId)) {
        const splitHandle = splitTerminalRefs.current.get(terminalId)
        splitHandle?.reattach(management.session, { forceTerminalChannel: true })
        window.setTimeout(() => {
          splitHandle?.fit()
        }, 0)
      }
      setPairStatus(t('workspace.terminalRestarted', { name: restartedTitle }))
      return true
    } catch (err) {
      const message = connectionErrorDisplayMessage(err, t)
      if (isAuthorizationConnectionError(err)) handleConnectionAuthFailure(machine?.machineId)
      setError(message)
      updateConnectionStatus(message, 'failed')
      return false
    } finally {
      setRestartingTerminalId(null)
    }
  }, [activeTerminalId, canManageTerminals, handleConnectionAuthFailure, machine?.machineId, refreshTerminals, restartingTerminalId, splitTerminalIds, t, terminals, updateConnectionStatus, withManagementApi])

  const restartManagedTerminal = useCallback(async () => {
    if (!selectedTerminalId) return
    if (await restartTerminalById(selectedTerminalId)) setMobileSheet(null)
  }, [restartTerminalById, selectedTerminalId])

  const closeFiles = useCallback(() => {
    setFilesOpen(false)
    setPage(fileReturnPageRef.current)
    window.setTimeout(() => {
      fitDisplayedTerminals()
      focusActiveTerminal()
    }, 0)
  }, [fitDisplayedTerminals, focusActiveTerminal])

  const openConnectionInfo = useCallback(() => {
    const existingSession = connectedSession ?? machineSessionRef.current?.session ?? null
    const connecting = phoneOnline && isConnectingConnectionPhase(connectionPhase)
    if (connecting) return
    if (!existingSession && !machine) {
      setConnectionInfoError(t('workspace.connection.unavailable'))
      setConnectionInfo(null)
      setConnectionInfoOpen(true)
      return
    }
    setConnectionInfoOpen(true)
    setConnectionInfoLoading(true)
    setConnectionInfoError(null)
    connectionPolicyFailureRef.current = null
    const sessionPromise = existingSession
      ? Promise.resolve(existingSession)
      : connectionSessionUnavailable
        ? Promise.reject(Object.assign(new Error('client session is unavailable'), { code: 'unavailable' }))
        : ensureMachineSession(machine!.machineId, { forceRelay: forceRelayConnection })
    void loadConnectionPanelState(
      sessionPromise.then(machineWorkspaceConnectionInfo),
      connector.getConnectionPolicy ? connector.getConnectionPolicy() : Promise.resolve(null),
    ).then((result) => {
      setConnectionInfo(result.info)
      setConnectionPolicyState(result.policy)
      if (result.error && !connecting) {
        connectionPolicyFailureRef.current = { stage: 'refresh' }
        setConnectionInfoError(connectionErrorDisplayMessage(result.error, t, phoneOnline))
      }
    }).finally(() => {
      setConnectionInfoLoading(false)
    })
  }, [connectedSession, connectionPhase, connectionSessionUnavailable, connector, ensureMachineSession, forceRelayConnection, machine, phoneOnline, t])

  const applyConnectionPolicy = useCallback(async (policy: ConnectionPolicy) => {
  if (!connector.applyConnectionPolicy) {
    setConnectionInfoError(t('workspace.connection.policyUnavailable'))
    return
  }
  if (activeTerminalId && !window.confirm(t('workspace.connection.reconnectConfirm'))) return
  setConnectionPolicyApplying(true)
  setConnectionInfoError(null)
  connectionPolicyFailureRef.current = null
  try {
    await connector.applyConnectionPolicy(policy)
  } catch (err) {
    connectionPolicyFailureRef.current = { stage: 'apply', policy }
    setConnectionInfoError(connectionErrorDisplayMessage(err, t, phoneOnline))
    setConnectionPolicyApplying(false)
    return
  }
  setConnectionPolicyState((current) => current ? { ...current, policy } : current)
  connectionPolicyReconnectPendingRef.current = true
  connectionPolicyFailureRef.current = { stage: 'reconnect' }
  try {
    await retryConnection({ preservePolicy: true, closeDialog: false })
  } catch (err) {
    connectionPolicyReconnectPendingRef.current = false
    setConnectionInfoError(connectionErrorDisplayMessage(err, t, phoneOnline))
    setConnectionPolicyApplying(false)
  }
  }, [activeTerminalId, connector, phoneOnline, retryConnection, t])

  const retryConnectionPolicy = useCallback(() => {
  connectionPolicyReconnectPendingRef.current = true
  setConnectionPolicyApplying(true)
  setConnectionInfoError(null)
  void retryConnection({ preservePolicy: true, closeDialog: false })
  }, [retryConnection])

  const retryConnectionPolicyFailure = useCallback(() => {
    const failure = connectionPolicyFailureRef.current
    if (failure?.stage === 'apply' && failure.policy) {
      void applyConnectionPolicy(failure.policy)
      return
    }
    // 诊断读取失败通常表示旧 generation 的 session lease 已失效；重试必须新建 session，不能重复读取旧 lease。
    retryConnectionPolicy()
  }, [applyConnectionPolicy, retryConnectionPolicy])

  const toggleKeyboardFocusLock = useCallback(() => {
    const next = !keyboardFocusLocked
    setKeyboardFocusLocked(next)
    if (next) blurActiveTerminal()
  }, [blurActiveTerminal, keyboardFocusLocked])

  const setTerminalToolbarModeAndReset = useCallback((mode: TerminalToolbarMode) => {
    setTerminalToolbarMode(mode)
    if (mode !== 'selection') {
      setHasTerminalSelection(false)
      activeTerminalHandle()?.clearSelection()
    }
  }, [activeTerminalHandle])

  const setTerminalToolbarVisibility = useCallback((open: boolean) => {
    setTerminalToolbarOpen(open)
    if (open) {
      setTerminalFnOpen(false)
      return
    }
    setTerminalToolbarModeAndReset('default')
  }, [setTerminalToolbarModeAndReset])

  const closeTerminalToolbarFromKeyboard = useCallback(() => {
    const opener = terminalToolbarOpenerRef.current
    terminalToolbarOpenerRef.current = null
    setTerminalToolbarVisibility(false)
    if (opener?.isConnected) opener.focus()
  }, [setTerminalToolbarVisibility])

  useEffect(() => {
    if (page === 'terminal') return
    terminalToolbarOpenerRef.current = null
    if (terminalToolbarOpen) setTerminalToolbarVisibility(false)
  }, [page, setTerminalToolbarVisibility, terminalToolbarOpen])

  useEffect(() => {
    if (!terminalToolbarOpen || terminalToolbarMode !== 'selection') return
    const timer = window.setInterval(() => {
      setHasTerminalSelection(activeTerminalHandle()?.hasSelection() ?? false)
    }, 200)
    return () => window.clearInterval(timer)
  }, [activeTerminalHandle, terminalToolbarMode, terminalToolbarOpen])

  const pasteTerminalTextWithConfirm = useCallback((text: string): boolean => {
    if (!text) {
      setPairStatus(t('workspace.clipboardEmpty'))
      return false
    }
    const needsConfirm = text.length > 200 || text.includes('\n') || text.includes('\r')
    if (needsConfirm) {
      setPasteConfirmText(text)
      setMobileSheet(null)
      return true
    }
    pasteTerminalText(text)
    setTerminalToolbarOpen(false)
    setTerminalToolbarModeAndReset('default')
    setMobileSheet(null)
    return true
  }, [pasteTerminalText, setTerminalToolbarModeAndReset])

  const refreshClipboardEntries = useCallback(async () => {
    setClipboardLoading(true)
    setClipboardError(null)
    try {
      const session = await withMachineSession()
      setClipboardEntries(await createRemoteClipboardApi(session).list())
    } catch (err) {
      setClipboardEntries([])
      setClipboardError(t('workspace.clipboardReadError'))
    } finally {
      setClipboardLoading(false)
    }
  }, [t, withMachineSession])

  const openClipboardHistory = useCallback(() => {
    setTerminalToolbarOpen(false)
    setTerminalToolbarModeAndReset('default')
    setMobileSheet('clipboard-history')
    void refreshClipboardEntries()
  }, [refreshClipboardEntries, setTerminalToolbarModeAndReset])

  const saveClipboardDraft = useCallback(async () => {
    const text = clipboardDraft
    if (!text) {
      setClipboardError(t('workspace.clipboardTextEmpty'))
      return
    }
    setClipboardLoading(true)
    setClipboardError(null)
    try {
      const session = await withMachineSession()
      const api = createRemoteClipboardApi(session)
      if (editingClipboardId) {
        await api.updateText(editingClipboardId, text)
      } else {
        await api.putText(text)
      }
      setClipboardDraft('')
      setEditingClipboardId(null)
      setClipboardEntries(await api.list())
      setPairStatus(t(editingClipboardId ? 'workspace.clipboardUpdated' : 'workspace.clipboardSaved'))
    } catch (err) {
      setClipboardError(t('errors.generic'))
    } finally {
      setClipboardLoading(false)
    }
  }, [clipboardDraft, editingClipboardId, t, withMachineSession])

  const deleteClipboardEntry = useCallback(async (id: string) => {
    setClipboardLoading(true)
    setClipboardError(null)
    try {
      const session = await withMachineSession()
      const api = createRemoteClipboardApi(session)
      await api.delete(id)
      if (editingClipboardId === id) {
        setEditingClipboardId(null)
        setClipboardDraft('')
      }
      setClipboardEntries(await api.list())
    } catch (err) {
      setClipboardError(t('errors.generic'))
    } finally {
      setClipboardLoading(false)
    }
  }, [editingClipboardId, t, withMachineSession])

  const loadBrowserClipboardDraft = useCallback(async () => {
    setClipboardError(null)
    try {
      const text = await systemClipboard.readText()
      setClipboardDraft(text)
      setEditingClipboardId(null)
    } catch {
      setClipboardError(t('workspace.browserClipboardError'))
    }
  }, [systemClipboard, t])

  const handleTerminalPaste = useCallback(async () => {
    setError(null)
    try {
      const text = await systemClipboard.readText()
      pasteTerminalTextWithConfirm(text)
    } catch {
      setError(t('workspace.clipboardReadError'))
    }
  }, [pasteTerminalTextWithConfirm, systemClipboard, t])

  const renderTerminalPathPickerSheet = () => {
    if (mobileSheet !== 'terminal-path-picker') return null
    const normalizedPath = normalizeFilePath(terminalPathPickerPath)
    const directories = [...terminalPathPickerEntries]
      .filter(isDirectoryEntry)
      .sort((left, right) => left.name.localeCompare(right.name, undefined, { numeric: true, sensitivity: 'base' }))
    return (
      <MobileSheetPanel webModal={webLayout} wide title={t('workspace.chooseDirectory')} testId="anytty-terminal-path-picker-sheet" onClose={() => setMobileSheet(terminalPathReturnSheet)}>
        <div className="flex flex-col gap-3">
          <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm p-3">
            <div className="break-all font-mono text-[12px] font-semibold text-zinc-800">{normalizedPath}</div>
            <div className="mt-3 grid grid-cols-2 gap-2">
              <Button variant="secondary"
                type="button"
                className="min-h-11 gap-2 px-3 text-[13px] font-semibold disabled:text-zinc-300"
                disabled={normalizedPath === '/'}
                onClick={() => { hapticImpact(); void loadTerminalPathPicker(parentPath(normalizedPath)) }}
              >
                <ChevronLeft className="h-4 w-4" />
                {t('workspace.parentDirectory')}
              </Button>
              <Button variant="default"
                type="button"
                className="min-h-11 gap-2 px-3 text-[13px] font-semibold"
                onClick={() => { hapticImpact(); selectTerminalWorkingDirectory(normalizedPath) }}
              >
                <FolderOpen className="h-4 w-4" />
                {t('workspace.useThisPath')}
              </Button>
            </div>
          </div>

          {terminalPathPickerError ? (
            <div className="border border-amber-200 bg-amber-50 px-3 py-2 text-[13px] font-medium text-amber-800" role="alert">
              {terminalPathPickerError}
            </div>
          ) : null}

          <div
            className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm flex h-80 max-h-[45vh] min-h-0 flex-col overflow-hidden"
            data-testid="anytty-terminal-path-picker-list"
          >
            {terminalPathPickerLoading ? (
              <div className="flex h-full items-center justify-center gap-2 text-[13px] font-medium text-zinc-500">
                <Spinner aria-hidden="true" />
                {t('common.loading')}
              </div>
            ) : directories.length === 0 ? (
              <div className="flex h-full items-center justify-center text-[13px] font-medium text-zinc-500">
                {t('files.emptyDirectory')}
              </div>
            ) : (
              <div className="min-h-0 flex-1 overflow-y-auto">
                {directories.map((entry) => {
                  const path = fileEntryPath(normalizedPath, entry)
                  return (
                    <Button variant="ghost"
                      key={path}
                      type="button"
                      className="flex min-h-12 w-full items-center gap-3 border-b border-zinc-100 px-3 text-left last:border-b-0 hover:bg-zinc-50 active:bg-zinc-50"
                      onClick={() => { hapticImpact(); void loadTerminalPathPicker(path) }}
                    >
                      <Folder className="h-4 w-4 shrink-0 text-zinc-500" />
                      <span className="min-w-0 flex-1 truncate text-[14px] font-semibold text-zinc-900">{entry.name}</span>
                    </Button>
                  )
                })}
              </div>
            )}
          </div>
        </div>
      </MobileSheetPanel>
    )
  }

  const renderTerminalPathBookmarksSheet = () => {
    if (mobileSheet !== 'terminal-path-bookmarks') return null
    return (
      <MobileSheetPanel webModal={webLayout} wide title={t('files.bookmarks.title')} testId="anytty-terminal-path-bookmarks-sheet" onClose={() => setMobileSheet(terminalPathReturnSheet)}>
        <div className="flex flex-col gap-3">
          <div className="grid grid-cols-2 gap-2">
            <Button variant="secondary"
              type="button"
              className="min-h-11 gap-2 px-3 text-[13px] font-semibold"
              onClick={() => { hapticImpact(); void addTerminalPathBookmark() }}
            >
              <BookmarkPlus className="h-4 w-4" />
              {t('files.bookmarks.saveCurrent')}
            </Button>
            <Button variant="secondary"
              type="button"
              className="min-h-11 gap-2 px-3 text-[13px] font-semibold"
              onClick={() => { hapticImpact(); void loadTerminalPathBookmarks() }}
            >
              <RefreshCw className="h-4 w-4" />
              {t('common.refresh')}
            </Button>
          </div>

          {terminalPathBookmarksError ? (
            <div className="border border-amber-200 bg-amber-50 px-3 py-2 text-[13px] font-medium text-amber-800" role="alert">
              {terminalPathBookmarksError}
            </div>
          ) : null}

          <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm overflow-hidden">
            {terminalPathBookmarksLoading ? (
              <div className="flex min-h-20 items-center justify-center gap-2 text-[13px] font-medium text-zinc-500">
                <Spinner aria-hidden="true" />
                {t('common.loading')}
              </div>
            ) : terminalPathBookmarks.length === 0 ? (
              <div className="flex min-h-20 items-center justify-center px-3 text-center text-[13px] font-medium text-zinc-500">
                {t('files.bookmarks.empty')}
              </div>
            ) : (
              terminalPathBookmarks.map((bookmark) => (
                <div key={bookmark.id} className="flex min-h-14 items-center gap-2 border-b border-zinc-100 px-3 last:border-b-0">
                  <Button variant="ghost"
                    type="button"
                    className="min-w-0 flex-1 text-left active:opacity-70"
                    onClick={() => { hapticImpact(); selectTerminalWorkingDirectory(bookmark.path) }}
                  >
                    <span className="block truncate text-[14px] font-semibold text-zinc-900">{bookmark.label}</span>
                    <span className="mt-0.5 block truncate font-mono text-[11px] font-medium text-zinc-500">{bookmark.path}</span>
                  </Button>
                  <Button variant="ghost"
                    type="button"
                    aria-label={t('files.bookmarks.removeNamed', { name: bookmark.label })}
                    className="flex h-11 w-11 shrink-0 items-center justify-center text-red-500 hover:bg-red-50/80 active:bg-red-50"
                    onClick={() => { hapticImpact(); void removeTerminalPathBookmark(bookmark.id) }}
                  >
                    <BookmarkMinus className="h-4 w-4" />
                  </Button>
                </div>
              ))
            )}
          </div>
        </div>
      </MobileSheetPanel>
    )
  }

  const renderClipboardHistorySheet = () => {
    if (mobileSheet !== 'clipboard-history') return null
    return (
      <MobileSheetPanel webModal={webLayout} wide title={t('workspace.clipboard')} testId="anytty-clipboard-history-sheet" onClose={() => setMobileSheet(null)}>
        <div className="flex flex-col gap-3">
          <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm p-3">
            <Textarea
              aria-label={t('workspace.clipboardText')}
              className="min-h-24 resize-none bg-zinc-50 text-[13px] font-medium text-zinc-900"
              value={clipboardDraft}
              onChange={(event) => setClipboardDraft(event.currentTarget.value)}
            />
            <div className="mt-2 grid grid-cols-3 gap-2">
              <Button variant="secondary"
                type="button"
                className="min-h-11 gap-1.5 px-2 text-[12px] font-semibold"
                onClick={() => { hapticImpact(); void loadBrowserClipboardDraft() }}
              >
                <ClipboardList className="h-4 w-4" />
                {t('workspace.browserClipboard')}
              </Button>
              <Button variant="secondary"
                type="button"
                className="min-h-11 gap-1.5 px-2 text-[12px] font-semibold"
                onClick={() => { hapticImpact(); void refreshClipboardEntries() }}
              >
                <RefreshCw className="h-4 w-4" />
                {t('common.refresh')}
              </Button>
              <Button variant="default"
                type="button"
                className="min-h-11 px-2 text-[12px] font-semibold disabled:bg-zinc-300 disabled:text-zinc-500"
                disabled={!clipboardDraft || clipboardLoading}
                onClick={() => { hapticImpact(); void saveClipboardDraft() }}
              >
                {t(editingClipboardId ? 'workspace.update' : 'files.actions.save')}
              </Button>
            </div>
          </div>

          {clipboardError ? (
            <div className="border border-amber-200 bg-amber-50 px-3 py-2 text-[13px] font-medium text-amber-800" role="alert">
              {clipboardError}
            </div>
          ) : null}

          <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm overflow-hidden">
            {clipboardLoading && clipboardEntries.length === 0 ? (
              <div className="flex min-h-20 items-center justify-center gap-2 text-[13px] font-medium text-zinc-500">
                <Spinner aria-hidden="true" />
                {t('common.loading')}
              </div>
            ) : clipboardEntries.length === 0 ? (
              <div className="flex min-h-20 items-center justify-center px-3 text-center text-[13px] font-medium text-zinc-500">
                {t('workspace.noClipboardHistory')}
              </div>
            ) : (
              clipboardEntries.map((entry) => (
                <div key={entry.id} className="border-b border-zinc-100 p-3 last:border-b-0">
                  <Button variant="ghost"
                    type="button"
                    className="block w-full text-left active:opacity-70"
                    onClick={() => { hapticImpact(); pasteTerminalTextWithConfirm(entry.text) }}
                  >
                    <span className="block max-h-10 overflow-hidden text-[14px] font-semibold text-zinc-900">{entry.preview}</span>
                    <span className="mt-1 block text-[11px] font-medium text-zinc-500">{formatClipboardTimestamp(entry.createdAt)}</span>
                  </Button>
                  <div className="mt-2 flex items-center justify-end gap-2">
                    <Button variant="secondary"
                      type="button"
                      className="min-h-11 gap-1.5 px-2 text-[12px] font-semibold"
                      onClick={() => { hapticImpact(); setEditingClipboardId(entry.id); setClipboardDraft(entry.text) }}
                    >
                      <SquarePen className="h-3.5 w-3.5" />
                      {t('common.edit')}
                    </Button>
                    <Button variant="ghost"
                      type="button"
                      className="flex min-h-11 items-center gap-1.5 border border-red-200 px-2 text-[12px] font-semibold text-red-600 hover:bg-red-50/80 active:bg-red-50"
                      onClick={() => { hapticImpact(); void deleteClipboardEntry(entry.id) }}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                      {t('common.delete')}
                    </Button>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      </MobileSheetPanel>
    )
  }

  const nestedOverlayOpen = Boolean(
    pasteConfirmText
      || connectionInfoOpen
      || mobileSheet
      || (!filesOpen && (terminalFnOpen || terminalToolbarOpen || hasSplitTerminals)),
  )
  useNativeBackHandler(() => {
    if (pasteConfirmText) {
      setPasteConfirmText('')
      return
    }
    if (connectionInfoOpen) {
      setConnectionInfoOpen(false)
      return
    }
    if (mobileSheet) {
      setMobileSheet(
        mobileSheet === 'terminal-path-picker' || mobileSheet === 'terminal-path-bookmarks'
          ? terminalPathReturnSheet
          : null,
      )
      return
    }
    if (!filesOpen && terminalFnOpen) {
      setTerminalFnOpen(false)
      return
    }
    if (!filesOpen && terminalToolbarOpen) {
      setTerminalToolbarVisibility(false)
      return
    }
    if (!filesOpen && hasSplitTerminals) {
      closeSplitTerminal()
    }
  }, NATIVE_BACK_PRIORITY.NESTED_OVERLAY, nestedOverlayOpen)

  useNativeBackHandler(() => {
    closeFiles()
  }, NATIVE_BACK_PRIORITY.FILE_MANAGER, filesOpen)

  const workspaceNavigationOpen = !filesOpen && (page === 'terminal' || Boolean(onBack))
  useNativeBackHandler(() => {
    if (page === 'terminal') {
      showTerminalListPage()
      return
    }
    onBack?.()
  }, NATIVE_BACK_PRIORITY.WORKSPACE, workspaceNavigationOpen)

  const renderTerminalListPage = () => {
    if (!machine) return null
    const showTerminalListLoader = loadingTerminals && !hasLoadedTerminals
    const desktopTerminalListClass = singlePane
      ? ''
      : webLayout
        ? webTerminalSidebarOpen
          ? 'md:flex md:w-[19rem] md:flex-none md:border-r md:border-[var(--anytty-app-line)] lg:w-80 2xl:w-[22rem]'
          : 'md:hidden'
        : 'md:flex md:w-72 md:flex-none md:border-r md:border-[var(--anytty-app-line)]'
    return (
      <aside
        className={`bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] relative min-h-0 flex-1 flex-col ${desktopTerminalListClass} ${page === 'terminal' ? 'hidden' : 'flex'}`}
        data-web-sidebar-open={webLayout ? webTerminalSidebarOpen : undefined}
        data-testid={page === 'terminal' ? undefined : 'anytty-terminal-list-page'}
      >
        <header className={`relative z-50 border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex min-h-12 shrink-0 items-center justify-between border-b px-2 pt-[env(safe-area-inset-top)] ${singlePane ? '' : `md:pt-0 ${webLayout ? 'md:min-h-14 md:px-3' : ''}`}`}>
          <div className="flex min-w-0 items-center gap-2">
            {onBack ? (
              <Button variant="ghost" size="icon"
                type="button"
                aria-label={t('common.backToMachines')}
                className="mr-1 border-transparent bg-transparent"
                onClick={() => { hapticSelection(); onBack() }}
              >
                <ChevronLeft className="h-5 w-5" />
              </Button>
            ) : null}
            <Monitor className="h-4 w-4 shrink-0 text-zinc-500" />
            <div className="min-w-0">
              <h1 className="truncate text-sm font-semibold text-zinc-900">{machine.name || machine.machineId}</h1>
              <p className="truncate text-[11px] text-zinc-500">{machine.machineId}</p>
            </div>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            {!webLayout ? (
              <Button variant="ghost" size="icon"
                type="button"
                aria-hidden={page === 'terminal' ? 'true' : undefined}
                aria-label={t('workspace.connectionInfo')}
                className="border-transparent bg-transparent"
                tabIndex={page === 'terminal' ? -1 : undefined}
                onClick={() => { hapticSelection(); openConnectionInfo() }}
              >
                <Info className="h-5 w-5" />
              </Button>
            ) : null}
            {webLayout ? (
              <Button
                aria-label={t('common.settings')}
                className="hidden border-transparent bg-transparent md:inline-flex"
                onClick={() => setWebSettingsOpen(true)}
                size="icon"
                title={t('common.settings')}
                variant="ghost"
              >
                <Settings2 className="h-5 w-5" />
              </Button>
            ) : null}
            <Button variant="ghost" size="icon"
              type="button"
              aria-hidden={page === 'terminal' ? 'true' : undefined}
              aria-label={t('workspace.openFiles')}
              className="border-transparent bg-transparent disabled:opacity-40"
              disabled={!phoneOnline || !connectionReady}
              tabIndex={page === 'terminal' ? -1 : undefined}
              onClick={() => { hapticSelection(); openFiles() }}
            >
              <Folder className="h-5 w-5" />
            </Button>
            {canManageTerminals ? (
              <Button variant="ghost" size="icon"
                type="button"
                aria-label={t('workspace.createTerminal')}
                className="border-transparent bg-transparent disabled:opacity-40"
                disabled={connectionInputBlocked}
                onClick={() => { hapticImpact(); openCreateTerminal() }}
              >
                <Plus className="h-5 w-5" />
              </Button>
            ) : null}
          </div>
        </header>
        {page === 'terminal-list' && initialConnectionFailure ? (
          <MachineConnectionFailureState
            failure={initialConnectionFailure}
            onBack={onBack}
            onPairAgain={initialConnectionFailure.requiresPairing && onNeedsReauthorization ? () => handleConnectionAuthFailure(machine.machineId) : undefined}
            onRetry={retryAfterFailure}
            retryPending={connectionRetryPending}
          />
        ) : null}
        {error && !connectionFailure ? (
          <div className="m-3 shrink-0 border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-800" role="alert">{error}</div>
        ) : null}
        {requireVerification ? (
          <section className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm mx-3 mt-3 p-5" data-testid="anytty-verification-gate">
            <div className="mb-4 flex items-center gap-3">
              <div className="flex h-11 w-11 items-center justify-center rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-soft)] text-zinc-600">
                <KeyRound className="h-5 w-5" />
              </div>
              <div>
                <h2 className="text-[17px] font-bold tracking-tight text-zinc-900">{t('workspace.verifyDevice')}</h2>
                <p className="text-[13px] font-medium text-zinc-500">{t('workspace.verifyDeviceCopy')}</p>
              </div>
            </div>
            <Button variant="default"
              type="button"
              className="min-h-12 w-full gap-2 px-4 text-[15px] font-semibold"
              onClick={() => { hapticImpact(); handleConnectionAuthFailure(machine.machineId) }}
            >
              <KeyRound className="h-4 w-4" />
              {t('workspace.verifyDevice')}
            </Button>
          </section>
        ) : null}
        {!hideTerminalListForUnavailableState && (hasLoadedTerminals || (!initialConnectionFailure && !requireVerification)) ? (
          <div
            className={`relative min-h-0 flex-1 overflow-y-auto px-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] pt-3 ${webLayout ? 'md:px-4 md:pt-4' : ''}`}
            data-testid="anytty-terminal-list-scroll"
          >
            <h2 className="mb-2 px-1 text-xs font-semibold uppercase tracking-wider text-zinc-500">{t('terminal.list')}</h2>
            <TerminalList
              machineId={machine.machineId}
              terminals={orderedTerminals}
              pinnedTerminalIds={terminalOrder}
              onReorderPinnedTerminal={reorderTerminalPins}
              onOpenTerminal={openTerminal}
              onManageTerminal={openManageTerminal}
              activeTerminalId={activePaneTerminalId ?? undefined}
              compact={webLayout}
              loading={showTerminalListLoader}
              loadingLabel={t('common.loading')}
              interactive={!connectionSessionUnavailable}
              webDraggable={webLayout}
              onWebTerminalDragChange={setWebDraggedTerminalId}
            />
          </div>
        ) : null}

        {mobileSheet === 'manage-terminal' && selectedTerminal ? (
          <MobileSheetPanel webModal={webLayout} title={selectedTerminal.title || t('terminal.defaultTitle')} testId="anytty-terminal-actions-sheet" onClose={() => setMobileSheet(null)}>
            <div className="flex flex-col gap-3">
              {selectedTerminal.state === 'exited' ? (
                <Button variant="secondary"
                  type="button"
                  className="min-h-12 w-full justify-between px-4 text-left text-[15px] font-medium"
                  disabled={!canManageTerminals || restartingTerminalId !== null}
                  onClick={() => { hapticImpact(); void restartManagedTerminal() }}
                >
                  <span>{t('workspace.restartTerminal')}</span>
                  <RefreshCw className="h-4 w-4 text-zinc-500" />
                </Button>
              ) : null}
              <Button variant="secondary" type="button" className="min-h-11 w-full justify-between px-4 text-sm font-medium" onClick={() => { hapticSelection(); toggleTerminalPin(selectedTerminal.terminalId) }}>
                <span>{t(selectedTerminalPinned ? 'terminal.order.unpin' : 'terminal.order.pinTop')}</span>
                <Pin className={`h-4 w-4 ${selectedTerminalPinned ? 'fill-current' : ''}`} />
              </Button>
              <Button variant="secondary"
                type="button"
                className="min-h-12 w-full justify-between px-4 text-left text-[15px] font-medium"
                disabled={!canManageTerminals}
                onClick={() => { hapticImpact(); openEditTerminal() }}
              >
                <span>{t('workspace.editTerminal')}</span>
                <SquarePen className="h-4 w-4 text-zinc-500" />
              </Button>
              <Button variant="ghost"
                type="button"
                className="flex min-h-12 w-full items-center justify-between rounded-md border border-red-200 bg-red-50 px-4 text-left text-[15px] font-medium text-red-700"
                disabled={!canManageTerminals}
                onClick={() => { hapticImpact(); void deleteManagedTerminal() }}
              >
                <span>{t('workspace.deleteTerminal')}</span>
                <Trash2 className="h-4 w-4 text-red-500" />
              </Button>
            </div>
          </MobileSheetPanel>
        ) : null}

        {(mobileSheet === 'create-terminal' || mobileSheet === 'edit-terminal') ? (
          <MobileSheetPanel
            initialFocusRef={webLayout && typeof window !== 'undefined' && window.innerWidth >= 768 ? terminalNameInputRef : undefined}
            title={t(mobileSheet === 'create-terminal' ? 'workspace.newTerminal' : 'workspace.editTerminal')}
            testId="anytty-terminal-editor-sheet"
            webModal={webLayout}
            wide
            onClose={() => setMobileSheet(null)}
          >
            <form
              className="flex flex-col gap-4"
              onSubmit={(event) => {
                event.preventDefault()
                hapticImpact()
                if (mobileSheet === 'create-terminal') {
                  void submitCreateTerminal()
                  return
                }
                void submitUpdateTerminal()
              }}
            >
              {terminalSubmitError ? (
                <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-[13px] font-medium text-red-700" role="alert">
                  {terminalSubmitError}
                </div>
              ) : null}
              <label className="flex flex-col gap-2 text-[14px] font-semibold text-zinc-700">
                {t('workspace.terminalForm.name')}
                <Input
                  ref={terminalNameInputRef}
                  className="min-h-12 bg-zinc-50 px-4 text-[15px] text-zinc-900"
                  value={terminalForm.name}
                  onChange={(event) => {
                    const value = event.currentTarget.value
                    setTerminalForm((current) => ({ ...current, name: value }))
                  }}
                />
              </label>
              {mobileSheet === 'create-terminal' ? (
                <label className="flex flex-col gap-2 text-[14px] font-semibold text-zinc-700">
                  {t('workspace.terminalForm.command')}
                  <Input
                    className="min-h-12 bg-zinc-50 px-4 text-[15px] text-zinc-900"
                    value={terminalForm.command}
                    onChange={(event) => {
                      const value = event.currentTarget.value
                      setTerminalForm((current) => ({ ...current, command: value }))
                    }}
                  />
                </label>
              ) : null}
              <label className="flex flex-col gap-2 text-[14px] font-semibold text-zinc-700">
                {t('workspace.terminalForm.cwd')}
                <Input
                  className="min-h-12 bg-zinc-50 px-4 text-[15px] text-zinc-900"
                  value={terminalForm.cwd}
                  onChange={(event) => {
                    const value = event.currentTarget.value
                    setTerminalForm((current) => ({ ...current, cwd: value }))
                  }}
                />
                <div className="grid grid-cols-2 gap-2">
                  <Button variant="secondary"
                    type="button"
                    className="min-h-11 gap-2 px-3 text-[13px] font-semibold"
                    disabled={!canManageTerminals}
                    onClick={() => {
                      hapticImpact()
                      openTerminalPathPicker()
                    }}
                  >
                    <FolderOpen className="h-4 w-4" />
                    {t('workspace.browse')}
                  </Button>
                  <Button variant="secondary"
                    type="button"
                    className="min-h-11 gap-2 px-3 text-[13px] font-semibold"
                    disabled={!canManageTerminals}
                    onClick={() => {
                      hapticImpact()
                      openTerminalPathBookmarks()
                    }}
                  >
                    <Bookmark className="h-4 w-4" />
                    {t('files.bookmarks.title')}
                  </Button>
                </div>
              </label>
              {mobileSheet === 'create-terminal' ? (
                <TerminalEnvironmentEditor
                  value={terminalForm.environment}
                  onChange={(environment) => setTerminalForm((current) => ({ ...current, environment }))}
                />
              ) : null}
              <label className="flex flex-col gap-2 text-[14px] font-semibold text-zinc-700">
                {t('workspace.terminalForm.sizeLock')}
                <NativeSelect
                  className="min-h-12 bg-zinc-50 px-4 text-[15px] text-zinc-900"
                  value={terminalForm.sizeLockMode}
                  onChange={(event) => {
                    const value = event.currentTarget.value as 'off' | 'warn' | 'lock'
                    setTerminalForm((current) => ({ ...current, sizeLockMode: value }))
                  }}
                >
                  <option value="off">{t('workspace.resizeMode.resizable')}</option>
                  <option value="warn">{t('workspace.resizeMode.warn')}</option>
                  <option value="lock">{t('workspace.resizeMode.locked')}</option>
                </NativeSelect>
              </label>
              <Button variant="default"
                type="submit"
                className="sticky -bottom-4 z-10 mt-2 min-h-12 w-full gap-2 px-4 text-[15px] font-semibold shadow-[0_-12px_24px_var(--anytty-app-bg)]"
                disabled={!canManageTerminals || terminalSubmitting || terminalDefaultsLoading}
              >
                {terminalSubmitting || terminalDefaultsLoading ? t('workspace.saving') : t(mobileSheet === 'create-terminal' ? 'workspace.createTerminal' : 'workspace.saveChanges')}
              </Button>
            </form>
          </MobileSheetPanel>
        ) : null}

        {renderTerminalPathPickerSheet()}
        {renderTerminalPathBookmarksSheet()}
        {page === 'terminal-list' ? <ConnectionRecoveryOverlayHost /> : null}
      </aside>
    )
  }

  useEffect(() => {
    setTerminalResizeControlBySlot(Object.fromEntries(
      displayedPaneKeys.map((paneKey) => [paneKey, defaultTerminalResizeControl]),
    ) as Record<TerminalPaneKey, TerminalResizeControl>)
  }, [activeTerminalId, connectedSession, displayedPaneKeys])

  useEffect(() => () => {
    disconnectMachineSession()
  }, [disconnectMachineSession])

  if (error && !machine) {
    return (
      <div className={`flex h-full min-h-0 items-center justify-center bg-zinc-50 p-4 ${className || ''}`}>
        <div className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm w-full max-w-md border-red-200 p-4 text-sm text-red-700" role="alert">
          <h2 className="mb-2 font-semibold text-red-900">{t('workspace.connectionError')}</h2>
          <p>{error}</p>
        </div>
      </div>
    )
  }

  if (!machine) {
    return (
      <div className={`flex h-full min-h-0 items-center justify-center bg-zinc-50 ${className || ''}`}>
        <div className="flex items-center gap-2 text-sm text-zinc-500">
          <Spinner className="text-zinc-600" aria-hidden="true" />
          {t('workspace.connectingAnyTTY')}
        </div>
      </div>
    )
  }

  const activateTerminalPane = (paneKey: TerminalPaneKey) => {
    if (activeTerminalSlot !== paneKey) hapticSelection()
    setActiveTerminalSlot(paneKey)
    activeTerminalSlotRef.current = paneKey
  }

  const renderTerminalPane = (paneKey: TerminalPaneKey) => {
    const primary = paneKey === 'primary'
    const terminalId = terminalIdForPane(paneKey, activeTerminalId)
    const terminal = terminals.find((candidate) => candidate.terminalId === terminalId)
    const active = activeTerminalSlot === paneKey
    const terminalReady = Boolean(terminalId && renderSession && (!primary || connectedTerminalId === terminalId))
    return (
      <div
        className={`relative h-full min-h-0 min-w-0 flex-1 overflow-hidden bg-[var(--anytty-terminal-bg)] ${active && hasSplitTerminals ? 'ring-1 ring-inset ring-[var(--anytty-accent)]' : ''}`}
        data-active-slot={active ? 'true' : 'false'}
        data-pane-key={paneKey}
        data-pane-terminal-id={terminalId ?? undefined}
        data-testid={primary ? 'anytty-terminal-panel' : 'anytty-split-terminal-panel'}
        key={paneKey}
        onPointerDown={() => activateTerminalPane(paneKey)}
      >
        {hasSplitTerminals ? (
          <WebTerminalPaneHeader
            active={active}
            terminal={terminal}
            onClose={primary || !terminalId ? undefined : () => removeSplitTerminal(terminalId)}
          />
        ) : null}
        {terminalReady && terminalId && renderSession ? (
          <Terminal
            ref={primary ? terminalRef : (handle) => {
              if (handle) splitTerminalRefs.current.set(terminalId, handle)
              else splitTerminalRefs.current.delete(terminalId)
            }}
            machineId={machine.machineId}
            terminalId={terminalId}
            session={renderSession}
            className={hasSplitTerminals
              ? 'absolute inset-x-0 bottom-0 top-7 outline-none'
              : 'absolute inset-0 outline-none'}
            modifierState={modifierState}
            onModifierStateChange={setModifierState}
            onInput={sendTerminalInput}
            onBufferChange={(isAlternate) => handleTerminalBufferChange(paneKey, isAlternate)}
            onCursorMove={handleCursorMove}
            onResizeControl={(control) => setTerminalResizeControlBySlot((current) => ({ ...current, [paneKey]: control }))}
            onTerminalInfoChange={applyTerminalInfo}
            onHistorySearchOpenChange={(open) => setTerminalHistorySearchOpenBySlot((current) => current[paneKey] === open ? current : { ...current, [paneKey]: open })}
            selectionMode={terminalToolbarOpen && terminalToolbarMode === 'selection' && active}
            settings={effectiveTerminalSettings}
            preventFocus={keyboardFocusLocked || connectionInputBlocked}
            suppressConnectingOverlay={false}
            historyOnly={terminal?.state === 'exited'}
          />
        ) : (
          <div className="absolute inset-0 flex items-center justify-center text-sm text-[var(--anytty-muted)]">
            {primary && terminalId && connectingTerminalId === terminalId
              ? connectionSessionUnavailable ? null : t('workspace.connectingTerminal')
              : terminalId && !connectionSessionUnavailable ? t('workspace.connectingTerminal') : t('terminal.noActive')}
          </div>
        )}
      </div>
    )
  }

  const renderTerminalSplitNode = (node: TerminalSplitNode): ReactNode => {
    if (node.type === 'pane') return renderTerminalPane(node.paneKey)
    return (
      <div
        className={`flex h-full min-h-0 min-w-0 flex-1 gap-px overflow-hidden bg-[var(--anytty-border-subtle)] ${node.direction === 'columns' ? 'flex-row' : 'flex-col'}`}
        data-split-direction={node.direction}
        data-split-id={node.id}
        key={node.id}
      >
        <div className="relative min-h-0 min-w-0 flex-none overflow-hidden" style={{ flexBasis: `${node.ratio}%` }}>
          {renderTerminalSplitNode(node.first)}
        </div>
        <WebSplitDivider
          direction={node.direction}
          ratio={node.ratio}
          onRatioChange={(ratio) => setTerminalSplitRoot((current) => updateTerminalSplitRatio(current, node.id, ratio))}
          onResizeEnd={fitDisplayedTerminals}
        />
        <div className="relative min-h-0 min-w-0 flex-1 overflow-hidden">
          {renderTerminalSplitNode(node.second)}
        </div>
      </div>
    )
  }

  return (
    <div
      ref={outerContainerRef}
      className={`relative flex h-full min-h-0 w-full max-w-full flex-col overflow-hidden bg-[var(--anytty-bg)] font-sans text-[var(--anytty-text)] ${singlePane ? '' : 'md:flex-row'} ${className || ''}`}
      data-machine-id={machine.machineId}
      data-workspace-layout={webLayout ? 'web' : 'app'}
    >
      <div className="sr-only" aria-live="polite">
        {t('workspace.splitTerminal')}: {displayedPaneKeys.length}
      </div>
      {renderTerminalListPage()}

      <main
        className={`relative min-h-0 min-w-0 max-w-full flex-1 overflow-hidden bg-[var(--anytty-terminal-bg)] ${page === 'terminal-list' ? (singlePane ? 'hidden' : webLayout ? 'hidden md:grid md:grid-rows-[auto_minmax(0,1fr)]' : 'hidden md:flex md:items-center md:justify-center md:bg-zinc-50/50') : `grid grid-rows-[auto_minmax(0,1fr)_auto] ${singlePane ? '' : webLayout ? 'md:grid-rows-[auto_minmax(0,1fr)]' : 'md:grid-rows-[minmax(0,1fr)]'}`}`}
        data-testid="anytty-terminal-page"
        style={terminalThemeStyle}
      >
        {webLayout ? (
          <WebTerminalWorkbench
            terminals={orderedTerminals}
            tabTerminalIds={webTabTerminalIds}
            activeTabTerminalId={activeTerminalId}
            splitTabTerminalIds={webSplitTabTerminalIds}
            draggedTerminalId={webDraggedTerminalId}
            sidebarOpen={webTerminalSidebarOpen}
            canCreateTerminal={canManageTerminals}
            canSplitTerminal={canSplitTerminal}
            disabled={connectionSessionUnavailable}
            onActivateTab={activateWebTerminalTab}
            onCloseTab={closeWebTab}
            onCreateTerminal={openCreateTerminal}
            onOpenFiles={openFiles}
            onOpenTerminalPicker={() => setWebTerminalPickerOpen(true)}
            onOpenSettings={() => setWebSettingsOpen(true)}
            onOpenSplit={() => splitActiveTerminal('bottom')}
            onReorderTabs={reorderWebTabs}
            onToggleSidebar={() => setWebTerminalSidebarOpen((current) => !current)}
            onTerminalDragChange={setWebDraggedTerminalId}
          />
        ) : null}
        {page === 'terminal-list' ? (
          <div className={`flex flex-col items-center gap-3 text-zinc-400 ${webLayout ? 'row-start-2' : ''}`}>
            <Monitor className="h-12 w-12 opacity-20" />
            <p className="text-sm font-medium">{t('workspace.selectTerminal')}</p>
          </div>
        ) : (
          <>
        <header
          className={`relative z-50 row-start-1 flex min-h-11 min-w-0 max-w-full shrink-0 items-center justify-between overflow-hidden border-b border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] px-0.5 pt-[env(safe-area-inset-top)] ${singlePane ? '' : 'md:hidden'}`}
          data-testid="anytty-terminal-header"
        >
          <div className="flex min-w-0 flex-1 items-center gap-1">
            <Button variant="ghost"
              type="button"
              aria-label={t('workspace.showTerminalList')}
              className="flex h-11 w-11 shrink-0 items-center justify-center text-[var(--anytty-muted)] transition-colors active:bg-[var(--anytty-surface-raised)]"
              onClick={() => { hapticSelection(); showTerminalListPage() }}
            >
              <ChevronLeft className="h-[18px] w-[18px]" />
            </Button>
            <Button variant="ghost"
              type="button"
              aria-label={`${t('workspace.switchTerminal')}: ${terminalHeaderSummary}`}
              className="flex h-11 min-w-0 flex-1 flex-col items-start justify-center gap-0 overflow-hidden px-0.5 text-left transition-colors active:bg-[var(--anytty-surface-raised)] disabled:opacity-40"
              disabled={connectionSessionUnavailable}
              onClick={() => {
                hapticSelection()
                setExpandedTerminalSwitcherMachineId(machine.machineId)
                setMobileSheet('terminals')
              }}
              title={terminalHeaderSummary}
            >
              <span aria-hidden="true" className="flex w-full min-w-0 items-center gap-1 text-[11px] font-semibold leading-4 text-[var(--anytty-text)]" data-testid="anytty-terminal-title">
                <span className="max-w-[38%] shrink-0 truncate text-[var(--anytty-muted)]">{terminalHeaderMachine}</span>
                <span className="shrink-0 text-[var(--anytty-muted)]">/</span>
                <span className="min-w-0 flex-1 truncate">{terminalHeaderTitle}</span>
              </span>
              {terminalHeaderDirectory ? (
                <span aria-hidden="true" className="block w-full min-w-0 truncate font-mono text-[9px] font-medium leading-3 text-[var(--anytty-muted)]" data-testid="anytty-terminal-path">
                  {terminalHeaderDirectory}
                </span>
              ) : null}
            </Button>
          </div>

          <div className="flex shrink-0 items-center">
            <Button variant="ghost"
              type="button"
              aria-label={t('workspace.openFiles')}
              title={t('workspace.openFiles')}
              data-testid="anytty-terminal-files-button"
              className="flex h-11 w-11 shrink-0 items-center justify-center text-[var(--anytty-muted)] transition-colors active:bg-[var(--anytty-surface-raised)] disabled:opacity-40"
              disabled={connectionSessionUnavailable}
              onClick={() => { hapticSelection(); openFiles() }}
            >
              <Folder className="h-[18px] w-[18px]" />
            </Button>
            <Button variant="ghost"
              type="button"
              aria-label={t('workspace.splitBelow')}
              title={t('workspace.splitBelow')}
              data-testid="anytty-terminal-split-button"
              className={`flex h-11 w-11 shrink-0 items-center justify-center transition-colors active:bg-[var(--anytty-surface-raised)] disabled:opacity-40 ${hasSplitTerminals ? 'text-[var(--anytty-text)]' : 'text-[var(--anytty-muted)]'}`}
              disabled={connectionInputBlocked || !canSplitTerminal}
              onClick={() => { hapticSelection(); splitActiveTerminal('bottom') }}
            >
              <Rows2 className="h-[18px] w-[18px]" />
            </Button>
            <Button variant="ghost"
              type="button"
              aria-label={t('workspace.terminalTools')}
              title={t('workspace.terminalTools')}
              data-testid="anytty-terminal-tools-button"
              className="flex h-11 w-11 shrink-0 items-center justify-center text-[var(--anytty-muted)] transition-colors active:bg-[var(--anytty-surface-raised)] disabled:opacity-40"
              onClick={(event) => { hapticImpact(); terminalToolbarOpenerRef.current = event.currentTarget; setTerminalToolbarModeAndReset('default'); setTerminalToolbarVisibility(true) }}
            >
              <SlidersHorizontal className="h-[17px] w-[17px]" />
            </Button>
          </div>
        </header>

        <div
          ref={terminalAreaRef}
          className={`relative row-start-2 h-full min-h-0 min-w-0 flex-1 overflow-hidden bg-[var(--anytty-terminal-bg)] ${singlePane ? '' : webLayout ? 'md:row-start-2' : 'md:row-start-1'}`}
          id={webLayout ? 'anytty-web-terminal-viewport' : undefined}
          data-testid="anytty-terminal-body"
        >
          {webLayout && webDraggedTerminalId ? (
            <WebTerminalDropOverlay
              canSplit={displayedPaneKeys.some((paneKey) => {
                const terminalId = terminalIdForPane(paneKey, activeTerminalId)
                return Boolean(terminalId && terminalId !== webDraggedTerminalId)
              })}
              draggedTerminalId={webDraggedTerminalId}
              onDrop={handleWebPaneDrop}
            />
          ) : null}
          {terminalToolbarOpen ? (
            <TerminalActionToolbar
              mode={terminalToolbarMode}
              hasSelection={hasTerminalSelection}
              wideViewportVisible={singlePane}
              remoteActionsDisabled={connectionInputBlocked}
              escapeEnabled={!mobileSheet && !filesOpen && !pasteConfirmText && !connectionInfoOpen && !transferCenterOpen}
              renderer={effectiveTerminalSettings.renderer}
              fontSize={effectiveTerminalSettings.fontSize}
              keyboardMode={activeTerminalKeyboardMode}
              resizeControl={activeTerminalResizeControl}
              sizeLocked={activeTerminalResizeLocked}
              resizeOwnerPending={resizeOwnerPending}
              sizeLockPending={sizeLockPending}
              onModeChange={setTerminalToolbarModeAndReset}
              onClose={() => setTerminalToolbarVisibility(false)}
              onEscape={closeTerminalToolbarFromKeyboard}
              onSelectAll={() => {
                activeTerminalHandle()?.selectAll()
                setHasTerminalSelection(true)
              }}
              onSelectVisible={() => {
                activeTerminalHandle()?.selectVisible()
                setHasTerminalSelection(true)
              }}
              onCopy={() => {
                const terminal = activeTerminalHandle()
                if (!terminal) return
                void terminal.getSelectionForClipboard().then(async (selected) => {
                  if (!selected) return false
                  await systemClipboard.writeText(selected)
                  return true
                }).then((copied) => {
                  if (!copied) return
                  setPairStatus(t('workspace.copied'))
                  setTerminalToolbarOpen(false)
                  setTerminalToolbarModeAndReset('default')
                }).catch(() => {
                  setError(t('workspace.copyFailed'))
                })
              }}
              onPaste={() => { void handleTerminalPaste() }}
              onOpenHistorySearch={() => {
                activeTerminalHandle()?.openHistorySearch()
                setTerminalToolbarVisibility(false)
              }}
              onOpenClipboardHistory={openClipboardHistory}
              onOpenSnippets={() => {
                setTerminalFnOpen((current) => !current)
                setTerminalToolbarOpen(false)
              }}
              onRendererChange={(renderer) => updateTerminalSettings({ renderer })}
              onFontSizeChange={(fontSize) => updateTerminalSettings({ fontSize })}
              onKeyboardModeChange={updateActiveTerminalKeyboardMode}
              onAcquireResizeOwner={() => { void acquireActiveResizeOwner() }}
              onReleaseResizeOwner={() => { void releaseActiveResizeOwner() }}
              onToggleSizeLock={() => { void toggleActiveTerminalSizeLock() }}
              onOpenConnectionInfo={() => {
                setTerminalToolbarOpen(false)
                openConnectionInfo()
              }}
              canSplitTerminal={canSplitTerminal}
              onSplitTerminal={splitActiveTerminal}
              splitTerminalOpen={hasSplitTerminals}
              syncSplitInput={syncSplitInput}
              onToggleSyncSplitInput={() => {
                hapticImpact()
                setSyncSplitInput((current) => !current)
              }}
              onCloseSplitTerminal={() => {
                hapticImpact()
                closeSplitTerminal()
                setTerminalToolbarOpen(false)
              }}
            />
          ) : null}

          {terminalFnOpen ? (
            <TerminalFnPanel
              command={activeToolTerminal?.command || activeToolTerminal?.title}
              onSend={(data) => {
                sendTerminalInput(data)
                if (data.endsWith('\n')) setTerminalFnOpen(false)
              }}
            />
          ) : null}

          {error && !connectionFailure ? (
            <div className="absolute inset-x-0 top-0 z-40 border-y border-red-500/30 bg-red-950/90 px-3 py-2 text-[12px] font-medium text-red-100 backdrop-blur" role="alert">{error}</div>
          ) : null}

          <div
            ref={terminalWrapperRef}
            className="absolute inset-0 flex min-h-0 min-w-0 bg-[var(--anytty-terminal-bg)]"
          >
            {renderTerminalSplitNode(terminalSplitRoot)}
          </div>
        </div>

        {!activeSlotTerminalExited && !terminalHistorySearchOpenBySlot[activeTerminalSlot] ? <MobileTerminalKeybar
          ref={mobileKeybarRef}
          className={`relative z-20 row-start-3 w-full max-w-full transition-opacity duration-200 ${connectionInputBlocked ? 'pointer-events-none opacity-55' : ''}`}
          onInput={sendTerminalInput}
          onFocusKeyboard={focusActiveTerminal}
          onBlurKeyboard={blurActiveTerminal}
          onToggleKeyboardFocusLock={toggleKeyboardFocusLock}
          fnOpen={terminalFnOpen}
          onToggleFn={() => {
            setTerminalFnOpen((current) => !current)
            setTerminalToolbarOpen(false)
          }}
          modifierState={modifierState}
          onModifierStateChange={setModifierState}
          keyboardVisible={keyboardVisible}
          keyboardFocusLocked={keyboardFocusLocked}
        /> : null}

        {pasteConfirmText ? (
          <PasteConfirmDialog
            text={pasteConfirmText}
            onCancel={() => setPasteConfirmText('')}
            onConfirm={() => {
              pasteTerminalText(pasteConfirmText)
              setPasteConfirmText('')
              setTerminalToolbarOpen(false)
              setTerminalToolbarModeAndReset('default')
            }}
          />
        ) : null}

        {mobileSheet === 'terminals' ? (
          <MobileSheetPanel expandable webModal={webLayout} title={t('terminal.list')} testId="anytty-terminal-switcher-sheet" onClose={() => setMobileSheet(null)}>
            <div className="flex flex-col divide-y divide-[var(--anytty-app-line)]" data-testid="anytty-terminal-machine-groups">
              {switcherMachines.map((switcherMachine) => {
                const currentMachine = switcherMachine.machineId === machine.machineId
                const inventory = currentMachine
                  ? { status: 'ready' as const, terminals: orderedTerminals }
                  : terminalSwitcherInventoryByMachine[switcherMachine.machineId]
                const expanded = expandedTerminalSwitcherMachineId === switcherMachine.machineId
                const count = inventory?.status === 'ready' ? inventory.terminals.length : switcherMachine.terminalCount
                return (
                  <section key={switcherMachine.machineId} data-switcher-machine-id={switcherMachine.machineId}>
                    <Button
                      variant="ghost"
                      type="button"
                      className="flex h-12 w-full items-center gap-3 rounded-none px-1 text-left hover:bg-[var(--anytty-app-surface-soft)]"
                      aria-expanded={expanded}
                      aria-controls={`terminal-switcher-${encodeURIComponent(switcherMachine.machineId)}`}
                      onClick={() => {
                        hapticSelection()
                        toggleTerminalSwitcherMachine(switcherMachine.machineId)
                      }}
                    >
                      <Monitor className="h-4 w-4 shrink-0 text-[var(--anytty-app-muted)]" />
                      <span className="min-w-0 flex-1 truncate text-[13px] font-semibold text-[var(--anytty-app-text)]">{switcherMachine.name}</span>
                      {currentMachine ? (
                        <span className="shrink-0 text-[10px] font-semibold text-[var(--anytty-app-accent)]">{t('terminal.currentMachine')}</span>
                      ) : count !== undefined ? (
                        <span className="shrink-0 text-[10px] font-medium tabular-nums text-[var(--anytty-app-muted)]">{t('terminal.terminalCount', { count })}</span>
                      ) : null}
                      <ChevronDown className={`h-4 w-4 shrink-0 text-[var(--anytty-app-muted)] transition-transform ${expanded ? 'rotate-180' : ''}`} />
                    </Button>
                    {expanded ? (
                      <div id={`terminal-switcher-${encodeURIComponent(switcherMachine.machineId)}`} className="pb-3 pl-7 pt-1">
                        {inventory?.status === 'error' ? (
                          <div className="flex min-h-16 items-center justify-between gap-3 px-2 text-xs text-[var(--anytty-app-muted)]" role="alert">
                            <span>{t('terminal.loadFailed')}</span>
                            <Button
                              variant="secondary"
                              type="button"
                              className="h-11 px-3 text-xs font-semibold"
                              onClick={() => {
                                loadTerminalSwitcherMachine(switcherMachine.machineId)
                              }}
                            >
                              {t('common.retry')}
                            </Button>
                          </div>
                        ) : (
                          <TerminalList
                            compact
                            machineId={switcherMachine.machineId}
                            terminals={inventory?.terminals ?? []}
                            loading={inventory?.status === 'loading'}
                            onOpenTerminal={(intent) => {
                              if (currentMachine) {
                                openTerminal(intent)
                                return
                              }
                              setMobileSheet(null)
                              onSwitchTerminal?.(intent)
                            }}
                            activeTerminalId={currentMachine ? activeTerminalId ?? undefined : undefined}
                            interactive={currentMachine
                              ? !connectionSessionUnavailable
                              : phoneOnline && connectionReady && Boolean(onSwitchTerminal)}
                          />
                        )}
                      </div>
                    ) : null}
                  </section>
                )
              })}
            </div>
          </MobileSheetPanel>
        ) : null}

        {renderClipboardHistorySheet()}
          </>
        )}
        {page === 'terminal' && !filesOpen ? <ConnectionRecoveryOverlayHost /> : null}
      </main>

      {webLayout ? (
        <>
          <WebTerminalPickerDialog
            activeTerminalId={activePaneTerminalId}
            canCreateTerminal={canManageTerminals}
            disabled={connectionSessionUnavailable}
            open={webTerminalPickerOpen}
            terminals={orderedTerminals}
            onCreateTerminal={openCreateTerminal}
            onOpenChange={setWebTerminalPickerOpen}
            onSelectTerminal={(terminalId) => openTerminal({ machineId: machine.machineId, terminalId })}
          />
          <WebTerminalSettingsDialog
            open={webSettingsOpen}
            settings={effectiveTerminalSettings}
            onChange={updateTerminalSettings}
            onOpenChange={setWebSettingsOpen}
          />
        </>
      ) : null}

      {hasOpenedFiles ? (
        <div
          className={`absolute inset-0 z-[60] flex flex-col bg-[var(--background)] transition-transform duration-200 ${singlePane ? '' : 'md:left-auto md:right-0 md:w-[450px] md:border-l md:border-[var(--anytty-app-line)]'} ${filesOpen ? (singlePane ? 'translate-y-0 visible' : 'translate-y-0 md:translate-x-0 visible') : (singlePane ? 'translate-y-full invisible' : 'translate-y-full md:translate-y-0 md:translate-x-full invisible')}`}
          data-testid="anytty-machine-files-overlay"
        >
          <div className={`relative z-50 border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex shrink-0 items-center border-b px-2 pb-2 pt-[calc(env(safe-area-inset-top)+0.5rem)] ${singlePane ? '' : 'md:h-14 md:pb-0 md:pt-0'}`}>
            <Button variant="ghost" size="icon"
              aria-label={t('workspace.closeFiles')}
              className="mr-1 shrink-0 border-transparent bg-transparent"
              onClick={() => { hapticSelection(); closeFiles() }}
            >
              <ChevronLeft className="h-5 w-5" />
            </Button>
            <div className="flex min-w-0 items-center gap-2">
              <Folder className="h-5 w-5 text-zinc-500" />
              <span className="truncate text-[17px] font-bold tracking-tight text-zinc-900">{t('files.list')}</span>
            </div>
          </div>
          {renderSession ? LoadedFileManager ? (
            <LoadedFileManager
              key={fileContextKey}
              machineId={machine.machineId}
              terminalId={fileTerminalId ?? undefined}
              session={renderSession}
              initialPath={fileInitialPath}
              className="flex h-full min-h-0 flex-col relative"
              active={filesOpen && !connectionSessionUnavailable}
              fileTransfer={fileTransfer}
              storage={storage}
              onOpenTransferCenter={() => setTransferCenterOpen(true)}
            />
          ) : fileManagerLoadState.status === 'error' ? (
            <div className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center" role="alert">
              <p className="text-sm font-medium text-zinc-700">{t('files.loadFailed')}</p>
              <Button variant="secondary"
                className="min-h-11 gap-2 px-4 font-semibold"
                onClick={() => { hapticSelection(); reloadAfterFileManagerLoadFailure() }}
              >
                <RefreshCw className="h-4 w-4" />
                {t('files.reloadApplication')}
              </Button>
            </div>
          ) : (
              <div className="flex h-full items-center justify-center gap-2 text-sm text-zinc-500">
                <Spinner aria-hidden="true" />
                <span>{t('common.loading')}</span>
              </div>
          ) : (
            <div className="flex h-full items-center justify-center text-sm text-zinc-500">
              {filesOpen && !connectionSessionUnavailable ? (
                <div className="flex items-center gap-2" role="status" aria-live="polite">
                  <Spinner aria-hidden="true" />
                  <span>{t('workspace.connecting')}</span>
                </div>
              ) : filesOpen ? null : t('workspace.fileAccessNotReady')}
            </div>
          )}
          {filesOpen ? <ConnectionRecoveryOverlayHost /> : null}
        </div>
      ) : null}

      <div className={`pointer-events-none absolute bottom-8 left-1/2 z-50 flex -translate-x-1/2 transform flex-col items-center gap-2 transition-all duration-300 ${pairStatus ? 'translate-y-0 opacity-100' : 'translate-y-4 opacity-0'}`}>
        <div className="flex items-center gap-2 rounded-lg border border-white/10 bg-[#18181b]/95 px-4 py-2.5 text-sm font-medium text-white backdrop-blur-md" role="status" aria-live="polite">
          {pairStatus}
        </div>
      </div>
      {connectionInfoOpen ? (
        <ConnectionInfoDialog
          info={connectionInfo}
          loading={connectionInfoLoading}
          connecting={isConnectingConnectionPhase(connectionPhase)}
          error={connectionInfoError}
      policyState={connectionPolicyState}
      applying={connectionPolicyApplying}
          onClose={() => setConnectionInfoOpen(false)}
          onRefresh={openConnectionInfo}
      onRetry={retryConnectionPolicyFailure}
      onApply={applyConnectionPolicy}
          onRestoreAuto={() => applyConnectionPolicy({ route: 'auto', cloud: 'auto', relayTransport: 'auto' })}
          routeManagement={connector.routeManagement}
          endpointId={machine.machineId}
          cloudPresence={cloudPresence}
        />
      ) : null}
      {fileTransfer && transferCenterOpen ? (
        <FileTransferPanel
          transfers={transferState.transfers}
          hasActiveTransfers={transferState.hasActiveTransfers}
          resolveMachineLabel={() => machine.name}
          onCancel={(id) => fileTransfer.cancelTransfer(id)}
          onDismiss={(id) => fileTransfer.dismissTransfer(id)}
          onPause={(id) => fileTransfer.pauseTransfer?.(id)}
          onResume={(id) => fileTransfer.resumeTransfer?.(id)}
          onResumeAll={() => fileTransfer.resumeAllTransfers?.(machine.machineId)}
          remoteActionsDisabled={connectionSessionUnavailable}
          onOpenFile={(id) => fileTransfer.openDownloadedFile?.(id)}
          open
          onOpenChange={(open) => {
            if (!open) setTransferCenterOpen(false)
          }}
        />
      ) : null}
    </div>
  )
}

function MachineConnectionFailureState({
  failure,
  onBack,
  onPairAgain,
  onRetry,
  retryPending,
}: {
  failure: ConnectionFailurePresentation
  onBack?: (() => void) | undefined
  onPairAgain?: (() => void) | undefined
  onRetry: () => void
  retryPending: boolean
}) {
  const { t } = useTranslation()
  const offline = failure.reason === 'phone_offline'
  const primaryAction = onPairAgain
    ? { label: t('machines.scanPairing'), onClick: onPairAgain }
    : failure.retryable && !offline
      ? {
          label: t(failure.reason === 'daemon_deleted' ? 'workspace.retryOtherRoutes' : 'workspace.connection.retry'),
          onClick: onRetry,
          pending: retryPending,
        }
      : undefined
  return (
    <ConnectionNotice
      data-testid="anytty-connection-failure"
      presentation={workspaceConnectionPresentation({
        phoneOnline: !offline,
        phase: 'failed',
        authAvailable: !failure.requiresPairing,
      })}
      title={failure.title}
      description={failure.message}
      variant="gate"
      primaryAction={primaryAction}
      secondaryAction={onBack ? { label: t('common.backToMachines'), onClick: onBack } : undefined}
    />
  )
}

function workspaceConnectionPresentation(input: {
  phoneOnline: boolean
  phase: RtcConnectionStateSnapshot['phase']
  authAvailable?: boolean | undefined
}) {
  const snapshot: MachineConnectionSnapshot = {
    machineId: 'workspace',
    phase: input.phase,
    statusText: '',
    connectionInfo: null,
    forceRelay: false,
    relayInUse: false,
    reconnectAttempt: 0,
    error: null,
  }
  return projectConnectionPresentation({
    phoneOnline: input.phoneOnline,
    authAvailable: input.authAvailable !== false,
    reachability: 'unknown',
    snapshot,
  })
}

function MobileSheetPanel({
  children,
  expandable = false,
  initialFocusRef,
  onClose,
  testId,
  title,
  webModal = false,
  wide = false,
}: {
  children: ReactNode
  expandable?: boolean | undefined
  initialFocusRef?: RefObject<HTMLElement | null> | undefined
  onClose: () => void
  testId: string
  title: string
  webModal?: boolean | undefined
  wide?: boolean | undefined
}) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const dragStartYRef = useRef<number | null>(null)
  const dragChangedRef = useRef(false)

  const handleDragStart = (event: React.PointerEvent<HTMLButtonElement>) => {
    dragStartYRef.current = event.clientY
    dragChangedRef.current = false
    event.currentTarget.setPointerCapture?.(event.pointerId)
  }
  const handleDragMove = (event: React.PointerEvent<HTMLButtonElement>) => {
    if (dragStartYRef.current === null) return
    const delta = event.clientY - dragStartYRef.current
    if (Math.abs(delta) < 24) return
    setExpanded(delta < 0)
    dragChangedRef.current = true
  }
  const handleDragEnd = () => {
    if (dragStartYRef.current === null) return
    if (!dragChangedRef.current) setExpanded((current) => !current)
    dragStartYRef.current = null
    dragChangedRef.current = false
    hapticSelection()
  }
  const panel = (
    <div
      className={`${webModal ? 'fixed z-[100] bg-black/55 backdrop-blur-[2px] md:p-6' : 'absolute z-40 bg-black/40 backdrop-blur-sm'} inset-0 flex items-end justify-center transition-opacity md:items-center`}
      data-presentation={webModal ? 'dialog' : 'sheet'}
      data-testid={testId}
      onClick={() => { hapticSelection(); onClose() }}
    >
      <ModalSurface
        aria-label={title}
        className={`bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] relative flex w-full flex-col overflow-hidden rounded-t-xl border-t border-[var(--anytty-app-line)] shadow-2xl transition-[height] duration-200 md:h-auto md:max-h-[min(85dvh,44rem)] md:rounded-lg md:border ${wide ? 'md:max-w-2xl' : 'md:max-w-md'} ${expandable ? (expanded ? 'h-[85dvh]' : 'h-[60dvh]') : 'max-h-[85dvh]'}`}
        initialFocusRef={initialFocusRef}
        onRequestClose={onClose}
        onClick={(e) => e.stopPropagation()}
        style={{ paddingBottom: 'env(safe-area-inset-bottom)' }}
      >
        {expandable ? (
          <Button
            aria-expanded={expanded}
            aria-label={t(expanded ? 'common.collapsePanel' : 'common.expandPanel')}
            className="absolute inset-x-0 top-0 z-10 flex h-8 w-full touch-none items-start justify-center rounded-none border-0 bg-transparent pt-2 hover:bg-transparent md:hidden"
            data-testid="anytty-sheet-drag-handle"
            variant="ghost"
            onPointerDown={handleDragStart}
            onPointerMove={handleDragMove}
            onPointerUp={handleDragEnd}
            onPointerCancel={() => {
              dragStartYRef.current = null
              dragChangedRef.current = false
            }}
          >
            <span aria-hidden="true" className="h-1 w-12 rounded-full bg-[var(--anytty-app-line-strong)]" />
          </Button>
        ) : (
          <div className="absolute left-1/2 top-3 h-1 w-12 -translate-x-1/2 rounded-full bg-[var(--anytty-app-line-strong)] md:hidden" />
        )}
        <header className="flex h-16 items-center justify-between border-b border-[var(--anytty-app-line)] px-5 pt-3">
          <h2 className="text-[17px] font-bold tracking-tight text-zinc-900">{title}</h2>
          <Button variant="ghost" size="icon"
            type="button"
            aria-label={t('common.closeNamed', { name: title })}
            className="border-transparent bg-transparent"
            onClick={() => { hapticSelection(); onClose() }}
          >
            <X className="h-5 w-5" />
          </Button>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
          {children}
        </div>
      </ModalSurface>
    </div>
  )
  return webModal && typeof document !== 'undefined' ? createPortal(panel, document.body) : panel
}

/** ConnectionInfoDialog 分离持久连接偏好和当前 ReadySession 诊断，不在 UI 推断网络路径。 */
export function ConnectionInfoDialog({
  info,
  loading,
  connecting = false,
  error,
  policyState,
  applying,
  onClose,
  onRefresh,
  onRetry,
  onApply,
  onRestoreAuto,
  routeManagement,
  endpointId,
  cloudPresence,
}: {
  info: ConnectionInfo | null
  loading: boolean
  connecting?: boolean | undefined
  error: string | null
  policyState: ConnectionPolicyState | null
  applying: boolean
  onClose: () => void
  onRefresh: () => void
  onRetry: () => void
  onApply: (policy: ConnectionPolicy) => void
  onRestoreAuto: () => void
  routeManagement?: ConnectionRouteManagementAdapter | undefined
  endpointId: string
  cloudPresence?: CloudPresenceSnapshot | undefined
}) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<ConnectionPolicy>({ route: 'auto', cloud: 'auto', relayTransport: 'auto' })
  useEffect(() => {
    if (policyState) setDraft(policyState.policy)
  }, [policyState])
  const type = info?.type ?? (info?.relayInUse ? 'relay' : 'unknown')
  const policyChanged = Boolean(policyState) && (
    draft.route !== policyState?.policy.route || draft.cloud !== policyState?.policy.cloud || draft.relayTransport !== policyState?.policy.relayTransport
  )
  const localCandidateHost = candidateHost(info?.localAddr)
  const remoteCandidateHost = candidateHost(info?.remoteAddr)
  const sameNATMapping = info?.observedPath === 'direct' &&
    (info.candidateType === 'srflx' || info.remoteCandidateType === 'srflx') &&
    localCandidateHost !== undefined && localCandidateHost === remoteCandidateHost
  const routeOptions: Array<{ value: ConnectionPolicy['route']; label: string; available: boolean; reason: string | undefined }> = [
    { value: 'auto', label: t('workspace.connection.routeAuto'), available: true, reason: undefined },
    { value: 'direct', label: t('workspace.connection.routeDirect'), available: policyState?.available.direct ?? false, reason: connectionPolicyUnavailableLabel(policyState?.unavailableReasons.direct, t) },
    { value: 'ssh', label: t('workspace.connection.routeSSH'), available: policyState?.available.ssh ?? false, reason: connectionPolicyUnavailableLabel(policyState?.unavailableReasons.ssh, t) },
    { value: 'cloud', label: t('workspace.connection.routeCloud'), available: policyState?.available.cloud ?? false, reason: connectionPolicyUnavailableLabel(policyState?.unavailableReasons.cloud, t) },
  ]
  return (
    <div className="fixed inset-0 z-[70] flex items-end justify-center bg-black/55 md:items-center md:p-4" onClick={() => { hapticSelection(); onClose() }}>
      <ModalSurface className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] flex max-h-[96dvh] w-full max-w-xl flex-col overflow-hidden rounded-t-xl border-t border-[var(--anytty-app-line)] md:max-h-[90vh] md:rounded-xl md:border" aria-labelledby="anytty-connection-title" onRequestClose={onClose} onClick={(event) => event.stopPropagation()}>
        <header className="flex items-center justify-between gap-3 border-b border-zinc-200 px-4 py-3">
          <div className="min-w-0">
            <h2 id="anytty-connection-title" className="text-[17px] font-semibold text-zinc-950">{t('workspace.connection.title')}</h2>
            <p className="mt-0.5 text-[12px] font-medium text-zinc-500">{connecting ? t('workspace.connection.phase.connecting') : connectionTypeLabel(type, t)}</p>
          </div>
          <Button variant="ghost" size="icon" aria-label={t('workspace.connection.closeInfo')} onClick={() => { hapticSelection(); onClose() }}>
            <X className="h-5 w-5" />
          </Button>
        </header>

        <div className="min-h-0 flex-1 overflow-y-auto">
          {connecting && !error ? (
            <div className="border-b border-blue-200 bg-blue-50 px-4 py-3 text-[13px] font-medium text-blue-800" role="status">
              <p>{t('workspace.connection.phase.connecting')}</p>
            </div>
          ) : null}
          {error ? (
            <div className="border-b border-red-200 bg-red-50 px-4 py-3 text-[13px] font-medium text-red-800" role="alert">
              <p>{error}</p>
              <div className="mt-3 flex flex-wrap gap-2">
                <Button variant="secondary" className="min-h-12 px-3 text-[13px] font-semibold" onClick={onRetry}>{t('workspace.connection.retry')}</Button>
                <Button variant="secondary" className="min-h-12 px-3 text-[13px] font-semibold" onClick={onRestoreAuto}>{t('workspace.connection.restoreAuto')}</Button>
              </div>
            </div>
          ) : null}
          <section className="border-b border-[var(--anytty-app-line)] px-4 py-4">
            <h3 className="text-[13px] font-semibold text-zinc-950">{t('workspace.connection.current')}</h3>
            <dl className="mt-2 overflow-hidden rounded-lg border border-[var(--anytty-app-line)]">
              <ConnectionInfoRow label={t('workspace.connection.route')} value={connecting ? t('workspace.connection.phase.connecting') : loading ? t('workspace.connection.reading') : connectionRouteLabel(info?.routeKind, t)} strong />
              <ConnectionInfoRow label={t('workspace.connection.path')} value={connecting ? t('workspace.connection.phase.connecting') : observedPathLabel(info?.observedPath, t)} />
              <ConnectionInfoRow label={t('workspace.connection.localCandidateAddress')} value={candidateDiagnostic(info?.localAddr, info?.candidateType, t)} />
              {info?.localBaseAddr ? <ConnectionInfoRow label={t('workspace.connection.localBaseAddress')} value={info.localBaseAddr} /> : null}
              <ConnectionInfoRow label={t('workspace.connection.remoteCandidateAddress')} value={candidateDiagnostic(info?.remoteAddr, info?.remoteCandidateType, t)} />
              {info?.remoteBaseAddr ? <ConnectionInfoRow label={t('workspace.connection.remoteBaseAddress')} value={info.remoteBaseAddr} /> : null}
              <ConnectionInfoRow label={t('workspace.connection.relayTransport')} value={displayDiagnostic(info?.relayTransport, t)} />
              <ConnectionInfoRow label={t('workspace.connection.rtt')} value={info?.rtt !== undefined ? `${Math.round(info.rtt)} ms` : t('workspace.connection.notProvided')} />
            </dl>
            {info?.observedPath === 'direct' ? (
              <p className="mt-2 text-[11px] leading-4 text-zinc-500">
                {sameNATMapping ? t('workspace.connection.sameNatMappingHint') : t('workspace.connection.candidateAddressHint')}
              </p>
            ) : null}
          </section>

          <section className="border-b border-[var(--anytty-app-line)] px-4 py-4">
            <h3 className="text-[13px] font-semibold text-zinc-950">{t('workspace.connection.cloudEdge')}</h3>
            <dl className="mt-2 overflow-hidden rounded-lg border border-[var(--anytty-app-line)]">
              <ConnectionInfoRow label={t('workspace.connection.endpointId')} value={endpointId} strong />
              <ConnectionInfoRow label={t('workspace.connection.deviceId')} value={displayDiagnostic(cloudPresence?.deviceId, t)} />
              <ConnectionInfoRow label={t('workspace.connection.daemonId')} value={displayDiagnostic(cloudPresence?.daemonId, t)} />
              <ConnectionInfoRow label={t('workspace.connection.currentEdge')} value={displayDiagnostic(cloudPresenceEdgeDescription(cloudPresence), t)} />
              <ConnectionInfoRow label={t('workspace.connection.edgeEndpoint')} value={displayDiagnostic(cloudPresence?.edgePublicEndpoint, t)} />
              <ConnectionInfoRow label={t('workspace.connection.edgeServerName')} value={displayDiagnostic(cloudPresence?.edgeServerName, t)} />
              <ConnectionInfoRow label={t('workspace.connection.locatorSource')} value={cloudPresenceLocatorSourceLabel(cloudPresence, t)} />
            </dl>
          </section>

          <fieldset className="border-b border-[var(--anytty-app-line)] px-4 py-4" disabled={loading || applying || !policyState}>
            <legend className="text-[13px] font-semibold text-zinc-950">{t('workspace.connection.preference')}</legend>
            <RadioGroup
              className="mt-2 divide-y divide-[var(--anytty-app-line)] overflow-hidden rounded-lg border border-[var(--anytty-app-line)] px-3"
              disabled={loading || applying || !policyState}
              value={draft.route}
              onValueChange={(route) => setDraft((current) => ({ ...current, route: route as ConnectionPolicy['route'] }))}
            >
              {routeOptions.map((option) => (
                <label key={option.value} className={`flex min-h-12 items-center gap-3 py-2 text-[14px] ${option.available ? 'text-zinc-900' : 'text-zinc-400'}`}>
                  <RadioGroupItem aria-label={option.label} {...(!option.available ? { 'aria-describedby': `connection-route-${option.value}-reason` } : {})} value={option.value} disabled={!option.available} />
                  <span className="min-w-0 flex-1 font-medium">{option.label}</span>
                  {!option.available ? <span id={`connection-route-${option.value}-reason`} className="max-w-[55%] text-right text-[11px] leading-4">{option.reason ?? t('workspace.connection.unavailableShort')}</span> : null}
                </label>
              ))}
            </RadioGroup>
          </fieldset>

          <details className="border-b border-[var(--anytty-app-line)] px-4 py-3" open={draft.route === 'cloud'}>
            <summary className="flex min-h-12 cursor-pointer items-center text-[13px] font-semibold text-zinc-950">{t('workspace.connection.cloudAdvanced')}</summary>
            <div className="space-y-5 pb-2">
              <ConnectionRadioGroup label={t('workspace.connection.cloudPath')} name="cloud-path" value={draft.cloud} options={[
                ['auto', t('workspace.connection.cloudAuto')], ['p2p', t('workspace.connection.cloudP2P')], ['relay', t('workspace.connection.cloudRelay')],
              ]} disabled={!policyState?.available.cloud || (draft.route !== 'auto' && draft.route !== 'cloud')} onChange={(cloud) => setDraft((current) => ({ ...current, cloud }))} />
              <ConnectionRadioGroup label={t('workspace.connection.relayTransport')} name="relay-transport" value={draft.relayTransport} options={[
                ['auto', t('workspace.connection.transportAuto')], ['udp', t('workspace.connection.transportUDP')], ['tcp', t('workspace.connection.transportTCP')],
              ]} disabled={!policyState?.available.cloud || (draft.route !== 'auto' && draft.route !== 'cloud') || draft.cloud === 'p2p'} onChange={(relayTransport) => setDraft((current) => ({ ...current, relayTransport }))} />
            </div>
          </details>

          {routeManagement ? (
            <details className="border-b border-[var(--anytty-app-line)] px-4 py-3">
              <summary className="flex min-h-12 cursor-pointer items-center text-[13px] font-semibold text-zinc-950">{t('workspace.routeManager.title')}</summary>
              <ConnectionRouteManager adapter={routeManagement} endpointId={endpointId} />
            </details>
          ) : null}

          <details className="px-4 py-3">
            <summary className="flex min-h-12 cursor-pointer items-center text-[13px] font-semibold text-zinc-950">{t('workspace.connection.diagnostics')}</summary>
            <dl className="mb-2 overflow-hidden rounded-lg border border-[var(--anytty-app-line)]">
              <ConnectionInfoRow label={t('workspace.connection.routeId')} value={displayDiagnostic(info?.routeId, t)} />
              <ConnectionInfoRow label={t('workspace.connection.candidatePairId')} value={displayDiagnostic(info?.candidatePairId, t)} />
              <ConnectionInfoRow label={t('workspace.connection.generation')} value={info?.generation?.toString() ?? t('workspace.connection.notProvided')} />
              <ConnectionInfoRow label={t('workspace.connection.reason')} value={displayDiagnostic(info?.routeSelectionReason, t)} />
              <ConnectionInfoRow label={t('workspace.connection.candidates')} value={candidateTypeText(info, t)} />
              <ConnectionInfoRow label={t('workspace.connection.protocols')} value={`${displayDiagnostic(info?.localProtocol, t)} / ${displayDiagnostic(info?.remoteProtocol, t)}`} />
              <ConnectionInfoRow label={t('workspace.connection.networkClass')} value={displayDiagnostic(info?.networkClass, t)} />
              <ConnectionInfoRow label={t('workspace.connection.sampledAt')} value={info?.sampledAt ? new Date(info.sampledAt).toLocaleString() : t('workspace.connection.notProvided')} />
              <ConnectionInfoRow label={t('workspace.connection.traffic')} value={info?.bytesSent !== undefined && info?.bytesReceived !== undefined ? `${info.bytesSent.toString()} / ${info.bytesReceived.toString()} B` : t('workspace.connection.notProvided')} />
            </dl>
          </details>
        </div>

        <footer className="flex flex-wrap items-center justify-end gap-2 border-t border-zinc-200 px-4 py-3">
          <Button variant="secondary" className="min-h-12 px-3 text-[13px] font-semibold" disabled={applying} onClick={() => { hapticImpact(); onRefresh() }}>
            {t('common.refresh')}
          </Button>
          <Button variant="default"
            className="min-h-12 px-4 text-[13px] font-semibold disabled:bg-zinc-300 disabled:text-zinc-500"
            disabled={loading || applying || !policyState || !policyChanged}
            onClick={() => { hapticImpact(); onApply(draft) }}
          >
            {applying ? t('workspace.connection.applying') : t('workspace.connection.applyReconnect')}
          </Button>
        </footer>
      </ModalSurface>
    </div>
  )
}

/** loadConnectionPanelState 独立读取 ReadySession 诊断和 Go-owned 持久策略。
 * Session 失败时仍返回可编辑策略，让用户能够从失败页切换 Direct/SSH Route。
 */
export async function loadConnectionPanelState(
  infoPromise: Promise<ConnectionInfo>,
  policyPromise: Promise<ConnectionPolicyState | null>,
): Promise<{ info: ConnectionInfo | null; policy: ConnectionPolicyState | null; error: unknown | null }> {
  const [info, policy] = await Promise.allSettled([infoPromise, policyPromise])
  return {
    info: info.status === 'fulfilled' ? info.value : null,
    policy: policy.status === 'fulfilled' ? policy.value : null,
    error: info.status === 'rejected' ? info.reason : policy.status === 'rejected' ? policy.reason : null,
  }
}

function ConnectionRadioGroup<T extends string>({ label, name, value, options, disabled, onChange }: {
  label: string
  name: string
  value: T
  options: Array<readonly [T, string]>
  disabled: boolean
  onChange: (value: T) => void
}) {
  return (
    <fieldset>
      <legend className="text-[12px] font-semibold text-zinc-600">{label}</legend>
      <RadioGroup className="mt-1 grid grid-cols-3 gap-1 rounded-lg border border-[var(--anytty-app-line)] p-1" disabled={disabled} name={name} value={value} onValueChange={(next) => onChange(next as T)}>
        {options.map(([option, text]) => (
          <div className="relative" key={option}>
            <RadioGroupItem className="peer sr-only" id={`${name}-${option}`} value={option} />
            <label htmlFor={`${name}-${option}`} className="flex min-h-12 cursor-pointer items-center justify-center rounded-md bg-zinc-50 px-2 text-center text-[12px] font-semibold text-zinc-700 peer-data-[state=checked]:bg-[var(--primary)] peer-data-[state=checked]:text-[var(--primary-foreground)] peer-disabled:cursor-not-allowed peer-disabled:opacity-50">
              {text}
            </label>
          </div>
        ))}
      </RadioGroup>
    </fieldset>
  )
}

function ConnectionInfoRow({ label, value, strong = false }: { label: string; value: string; strong?: boolean | undefined }) {
  return (
    <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] items-start gap-3 border-b border-[var(--anytty-app-line)] bg-zinc-50 px-3 py-2 last:border-b-0">
      <dt className="text-[12px] font-semibold text-zinc-500">{label}</dt>
      <dd className={`min-w-0 break-words text-[12px] ${strong ? 'font-semibold text-zinc-950' : 'font-medium text-zinc-700'}`}>{value}</dd>
    </div>
  )
}

function cloudPresenceEdgeDescription(presence: CloudPresenceSnapshot | undefined): string | undefined {
  if (!presence) return undefined
  const label = cloudPresenceEdgeLabel(presence)
  const endpoint = presence.edgePublicEndpoint?.trim()
  if (label && endpoint) return `${label} · ${endpoint}`
  return label || endpoint || presence.edgeId
}

function cloudPresenceLocatorSourceLabel(presence: CloudPresenceSnapshot | undefined, t: ReturnType<typeof useTranslation>['t']): string {
  if (!presence?.locatorSource) return t('workspace.connection.notProvided')
  if (presence.locatorSource === 'controller') {
    return presence.refreshedFromController ? t('workspace.connection.locatorSourceControllerRefreshed') : t('workspace.connection.locatorSourceController')
  }
  if (presence.locatorSource === 'cached_edge') return t('workspace.connection.locatorSourceCachedEdge')
  return presence.locatorSource
}

function connectionTypeLabel(type: ConnectionInfo['type'], t: ReturnType<typeof useTranslation>['t']): string {
  if (type === 'p2p') return t('machines.path.direct')
  if (type === 'relay') return t('machines.path.relay')
  return t('terminal.state.unknown')
}

function candidateTypeText(info: ConnectionInfo | null, t: ReturnType<typeof useTranslation>['t']): string {
  const local = displayDiagnostic(info?.candidateType, t)
  const remote = displayDiagnostic(info?.remoteCandidateType, t)
  return `${local} / ${remote}`
}

function candidateDiagnostic(address: string | undefined, type: string | undefined, t: ReturnType<typeof useTranslation>['t']): string {
  const value = displayDiagnostic(address, t)
  return type?.trim() ? `${value} (${type.trim()})` : value
}

function candidateHost(address: string | undefined): string | undefined {
  const value = address?.trim()
  if (!value) return undefined
  if (value.startsWith('[')) {
    const closingBracket = value.indexOf(']')
    return closingBracket > 1 ? value.slice(1, closingBracket) : value
  }
  const separator = value.lastIndexOf(':')
  if (separator > 0 && value.indexOf(':') === separator) return value.slice(0, separator)
  return value
}

function displayDiagnostic(value: string | undefined, t: ReturnType<typeof useTranslation>['t']): string {
  return value?.trim() || t('workspace.connection.notProvided')
}

function connectionRouteLabel(kind: ConnectionInfo['routeKind'], t: ReturnType<typeof useTranslation>['t']): string {
  switch (kind) {
    case 'local': return t('workspace.connection.routeLocal')
    case 'direct': return t('workspace.connection.routeDirect')
    case 'ssh': return t('workspace.connection.routeSSH')
    case 'cloud': return t('workspace.connection.routeCloud')
    default: return t('workspace.connection.notProvided')
  }
}

function observedPathLabel(path: ConnectionInfo['observedPath'], t: ReturnType<typeof useTranslation>['t']): string {
  switch (path) {
    case 'direct': return t('workspace.connection.pathDirect')
    case 'single_relay': return t('workspace.connection.pathRelay')
    default: return t('workspace.connection.notProvided')
  }
}

function isProtoSessionAlive(session: MachineWorkspaceClientSession): boolean { return session.isAlive() }

function closeTerminalDataChannel(session: MachineWorkspaceClientSession, terminalId: string): void {
  void session
  void terminalId
}

function closeMachineWorkspaceSession(session: MachineWorkspaceClientSession): Promise<void> {
  return session.close()
}

async function machineWorkspaceConnectionInfo(session: MachineWorkspaceClientSession): Promise<ConnectionInfo> {
  const snapshot = await session.getConnectionSnapshot?.() ?? session.connection
  const routeKind = connectionRouteKindFromProto(snapshot?.routeKind)
  const observedPath = observedPathFromProto(snapshot?.observedPath)
  const relayInUse = observedPath === 'single_relay'
  return {
  path: routeKind === 'cloud' ? 'hub' : 'local',
  routeId: snapshot?.routeId || session.stamp.routeId,
  routeKind,
  observedPath,
  routeSelectionReason: snapshot?.selectionReason as ConnectionInfo['routeSelectionReason'],
    connectionId: `${session.stamp.endpointId}:${session.stamp.generation}`,
    machineId: session.stamp.endpointId,
  relayInUse,
  type: relayInUse ? 'relay' : observedPath === 'direct' ? 'p2p' : 'unknown',
  candidateType: candidateTypeFromProto(snapshot?.localCandidateType),
  remoteCandidateType: candidateTypeFromProto(snapshot?.remoteCandidateType),
  localAddr: candidateAddress(snapshot?.localIp, snapshot?.localPort),
  remoteAddr: candidateAddress(snapshot?.remoteIp, snapshot?.remotePort),
  localBaseAddr: candidateAddress(snapshot?.localRelatedIp, snapshot?.localRelatedPort),
  remoteBaseAddr: candidateAddress(snapshot?.remoteRelatedIp, snapshot?.remoteRelatedPort),
  candidatePairId: snapshot?.candidatePairId || undefined,
  localProtocol: transportFromProto(snapshot?.localProtocol),
  remoteProtocol: transportFromProto(snapshot?.remoteProtocol),
  relayTransport: transportFromProto(snapshot?.relayTransport),
  networkClass: snapshot?.networkClass || undefined,
  rtt: snapshot?.roundTripNanos ? Number(snapshot.roundTripNanos / 1_000_000n) : undefined,
  sampledAt: snapshot?.sampledAtUnixNano ? Number(snapshot.sampledAtUnixNano / 1_000_000n) : undefined,
  bytesSent: snapshot?.bytesSent,
  bytesReceived: snapshot?.bytesReceived,
  packetsSent: snapshot?.packetsSent,
  lossEvents: snapshot?.lossEvents,
  generation: session.stamp.generation,
  }
}

function candidateAddress(ip: string | undefined, port: number | undefined): string | undefined {
  const address = ip?.trim()
  if (!address) return undefined
  if (!port) return address
  return address.includes(':') ? `[${address}]:${port}` : `${address}:${port}`
}

function connectionPolicyUnavailableLabel(reason: ConnectionPolicyState['unavailableReasons']['direct'] | undefined, t: (key: string) => string): string | undefined {
  if (!reason) return undefined
  return t(`workspace.connection.unavailableReason.${reason}`)
}

function connectionRouteKindFromProto(value: ConnectionRouteKind | undefined): ConnectionInfo['routeKind'] {
  switch (value) {
    case ConnectionRouteKind.LOCAL: return 'local'
    case ConnectionRouteKind.DIRECT: return 'direct'
    case ConnectionRouteKind.SSH: return 'ssh'
    case ConnectionRouteKind.CLOUD: return 'cloud'
    default: return undefined
  }
}

function observedPathFromProto(value: ConnectionObservedPath | undefined): ConnectionInfo['observedPath'] {
  switch (value) {
    case ConnectionObservedPath.DIRECT: return 'direct'
    case ConnectionObservedPath.SINGLE_RELAY: return 'single_relay'
    default: return undefined
  }
}

function candidateTypeFromProto(value: ConnectionCandidateType | undefined): string | undefined {
  switch (value) {
    case ConnectionCandidateType.HOST: return 'host'
    case ConnectionCandidateType.SERVER_REFLEXIVE: return 'srflx'
    case ConnectionCandidateType.PEER_REFLEXIVE: return 'prflx'
    case ConnectionCandidateType.RELAY: return 'relay'
    default: return undefined
  }
}

function transportFromProto(value: ConnectionTransport | undefined): string | undefined {
  switch (value) {
    case ConnectionTransport.UDP: return 'UDP'
    case ConnectionTransport.TCP: return 'TCP'
    default: return undefined
  }
}

function subscribeMachineWorkspaceEvents(session: MachineWorkspaceClientSession, handler: (event: RtcEvent) => void): RtcSubscription {
  let closed = false
  let subscription: RtcSubscription | null = null
  void openProtoEventSubscription(session, create(EventSubscribeCommandSchema, {
    types: [ApplicationEventType.TERMINAL_LIFECYCLE],
  }), (event) => {
    if (event.event.case === 'terminalLifecycle') handler({ type: 'terminal_changed' })
  }).then((opened) => {
    if (closed) opened.close()
    else subscription = opened
  }).catch(() => undefined)
  return { close() { closed = true; subscription?.close(); subscription = null } }
}

function isDirectoryEntry(entry: FileEntry): boolean {
  return entry.type === 'dir' || entry.type === 'symlink-dir'
}

function formatClipboardTimestamp(value: string): string {
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return value
  return new Date(timestamp).toLocaleString(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function isTransientConnectionPhase(phase: RtcConnectionStateSnapshot['phase']): boolean {
  return phase === 'probing' || phase === 'connecting'
}

function isConnectingConnectionPhase(phase: RtcConnectionStateSnapshot['phase'] | null | undefined): boolean {
  return phase === 'probing' || phase === 'resolving' || phase === 'signaling' || phase === 'connecting' ||
    phase === 'authorizing' || phase === 'verifying' || phase === 'reconnecting' || phase === 'waiting_network'
}

export function terminalInventoryRefreshIntervalMs(relayInUse: boolean): number {
  return relayInUse ? 5_000 : 2_000
}

function normalizeDirectoryPickerPath(path: string): string {
  return normalizeTerminalDirectory(path) || '/'
}

function normalizeTerminalDirectory(path: string | undefined): string {
  const trimmed = path?.trim()
  if (!trimmed) return ''
  if (trimmed.startsWith('file://')) {
    try {
      const url = new URL(trimmed)
      if (url.pathname) return decodeURIComponent(url.pathname)
    } catch {
      return ''
    }
  }
  if (!trimmed.startsWith('/') && !/^[A-Za-z]:[\\/]/.test(trimmed) && !/^\\\\[^\\]+\\[^\\]+/.test(trimmed)) return ''
  return normalizeFilePath(trimmed)
}

function isTerminalInventoryRuntimeEvent(event: RtcEvent): boolean {
  if (event.type === 'inventory_changed') return true
  if (event.type === 'terminal_changed') return true
  return event.type === 'terminal_created' ||
    event.type === 'terminal_state_changed' ||
    event.type === 'terminal_resized' ||
    event.type === 'terminal_removed' ||
    event.type === 'terminal_metadata_changed'
}

function normalizeRuntimeTerminalEvent(payload: Record<string, unknown>): RemoteTerminal | null {
  const terminal = payload.terminal
  if (typeof terminal !== 'object' || terminal === null || Array.isArray(terminal)) return null
  const record = terminal as Record<string, unknown>
  const terminalId = record.terminal_id ?? record.terminalId ?? record.id ?? record.ID
  const machineId = record.machine_id ?? record.machineId
  if (typeof terminalId !== 'string' || !terminalId.trim()) return null
  if (typeof machineId !== 'string' || !machineId.trim()) return null
  return {
    terminalId: terminalId.trim(),
    machineId: machineId.trim(),
    title: typeof record.title === 'string' && record.title.trim()
      ? record.title.trim()
      : typeof record.name === 'string' && record.name.trim()
        ? record.name.trim()
        : terminalId.trim(),
    state: record.state === 'running' || record.state === 'exited' ? record.state : 'unknown',
    command: typeof record.command === 'string'
      ? record.command
      : Array.isArray(record.command) && record.command.every((item) => typeof item === 'string')
        ? record.command.join(' ')
        : undefined,
    cols: typeof record.cols === 'number' ? record.cols : undefined,
    rows: typeof record.rows === 'number' ? record.rows : undefined,
    cwd: typeof record.cwd === 'string' ? record.cwd : undefined,
    environment: typeof record.environment === 'string' ? record.environment : undefined,
    sizeLocked: record.size_locked === true || record.sizeLocked === true,
    sizeLockMode: record.size_lock_mode === 'off' || record.size_lock_mode === 'warn' || record.size_lock_mode === 'lock'
      ? record.size_lock_mode
      : undefined,
    resizeOwnerSurfaceId: resizeOwnershipString(record, 'owner_surface_id'),
    resizeOwnerViewId: resizeOwnershipString(record, 'owner_view_id'),
    resizeOwnerAttachmentCount: typeof record.resize_owner_attachment_count === 'number'
      ? record.resize_owner_attachment_count
      : undefined,
  }
}

function resizeOwnershipString(record: Record<string, unknown>, key: 'owner_surface_id' | 'owner_view_id'): string | undefined {
  const direct = record[key]
  if (typeof direct === 'string' && direct.trim()) return direct.trim()
  const ownership = record.resize_ownership
  if (typeof ownership !== 'object' || ownership === null || Array.isArray(ownership)) return undefined
  const nested = (ownership as Record<string, unknown>)[key]
  return typeof nested === 'string' && nested.trim() ? nested.trim() : undefined
}
