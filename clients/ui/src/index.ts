export * from './connection/connectionMessageReducer'
export * from './connection/connectionPresentation'
export { anyttyI18n, anyttyIntlLocale, anyttyLanguages, normalizeAnyTTYLanguage } from './i18n'
export type { AnyTTYLanguage } from './i18n'
export { PrivacyPolicyContent } from './legal/PrivacyPolicyContent'
export type { PrivacyPolicyLanguage } from './legal/PrivacyPolicyContent'
export * from './connection/eventQueue'
export {
  NATIVE_BACK_PRIORITY,
  addNativeBackHandler,
  dispatchNativeBack,
} from './platform/nativeBack'
export type { AnyTTYNativeBackHandler } from './platform/nativeBack'
export {
  ANYTTY_NATIVE_KEYBOARD_EVENT,
  addNativeKeyboardListener,
  dispatchNativeKeyboardEvent,
} from './platform/nativeKeyboard'
export type { AnyTTYNativeKeyboardEventDetail, AnyTTYNativeKeyboardHandler } from './platform/nativeKeyboard'
export {
  assertRemoteModelShape,
  normalizeMachine,
  normalizeTerminal,
} from './core/model'
export type {
  LocalRTCInfo,
  Machine,
  MachineState,
  Terminal as RemoteTerminal,
  TerminalState,
} from './core/model'
export { openProtoEventSubscription } from './core/protoEventSubscription'
export { Terminal } from './terminal/Terminal'
export type { TerminalHandle, TerminalProps } from './terminal/Terminal'
export { TerminalList } from './terminal/TerminalList'
export type { OpenTerminalIntent, TerminalListProps } from './terminal/TerminalList'
export { MobileTerminalKeybar } from './terminal/MobileTerminalKeybar'
export type { MobileTerminalKeybarProps } from './terminal/MobileTerminalKeybar'
export {
  applyTerminalModifiers,
  nextModifierState,
} from './terminal/mobileTerminalInput'
export type {
  ModifierState,
  TerminalModifierResult,
  TerminalModifierState,
} from './terminal/mobileTerminalInput'
export { createFileApi, createFilePreviewSource } from './files/fileApi'
export {
  decodeFileStreamErrorPayload,
  decodeFileTransferAckPayload,
  decodeFileTransferDataPayload,
  decodeFileTransferFinishPayload,
  decodeFileTransferResultPayload,
  encodeFileTransferAckPayload,
  encodeFileTransferDataPayload,
  encodeFileTransferFinishPayload,
} from './files/fileStreamProtocol'
export type {
  DirListResponse,
  DownloadInitResponse,
  FileApi,
  FileEntry,
  FileEntryType,
  FilePreviewSource,
  FilePreviewStreamChunk,
  FilePreviewStreamOptions,
  FilePreviewStreamProgress,
  FilePreviewStreamResult,
  FileTransferContext,
  TransferInfo,
  TransferStatus,
} from './files/fileApi'
export { FileTransferPanel } from './files/FileTransferPanel'
export { ConnectionStatusBanner } from './connection/ConnectionStatusBanner'
export type { ConnectionStatusBannerProps } from './connection/ConnectionStatusBanner'
export {
  connectionPathLabel,
  connectionPhaseLabel,
  connectionSnapshotFromStatus,
  connectionStateFromAttempt,
  connectionStatusIsSettled,
  createConnectionStatePublisher,
  inferConnectionPhase,
} from './connection/connectionState'
export type { ConnectionStatePublisher } from './connection/connectionState'
export type { FileManagerProps } from './files/FileManager'
export type { MachineWorkspaceProps, MachineWorkspaceConnector, MachineWorkspaceClientSession, MachineWorkspaceSessionInput, MachineWorkspaceInventoryApi } from './app/MachineWorkspace'
export { ModalSurface } from './ui/ModalSurface'
export type { ModalSurfaceProps } from './ui/ModalSurface'
export {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
  Input,
  NativeSelect,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectScrollDownButton,
  SelectScrollUpButton,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
  Spinner,
  Textarea,
  badgeVariants,
  buttonVariants,
  cn,
} from './ui/primitives'
export type {
  BadgeProps,
  ButtonProps,
  ButtonSize,
  ButtonVariant,
  InputProps,
  NativeSelectProps,
  SpinnerProps,
  TextareaProps,
} from './ui/primitives'
export { RemoteControlApp } from './app/RemoteControlApp'
export type { MachineRuntime, MachineRuntimeFactory, RemoteControlAppProps } from './app/RemoteControlApp'
export type { ExternalPairingAdapter, ExternalPairingImportResult } from './app/RemoteControlApp'
export { mountRemoteControlApp } from './entries/mountRemoteControlApp'
export type { RemoteControlEntryOptions } from './entries/mountRemoteControlApp'
export {
  APP_THEME_STORAGE_KEY,
  DEFAULT_APP_THEME,
  appThemeCssVariables,
  normalizeAppTheme,
  readAppTheme,
  writeAppTheme,
} from './app/appTheme'
export type { AppTheme } from './app/appTheme'
export type {
  AppMachineRecord,
  AppMachineSource,
  AppMachineState,
  ConnectionFlowSnapshot,
  ConnectionFlowStage,
  MachineAccessClass,
  MachineIconName,
} from './state/appMachine'
export { MACHINE_ICON_NAMES } from './state/appMachine'
export { createMachineStore } from './state/machineStore'
export type {
  MachineStore,
  MachineStoreOptions,
  StoredMachineAddresses,
  StoredMachineEndpoints,
  StoredMachineRecord,
} from './state/machineStore'
export {
  createBrowserRemoteNetworkRuntime,
  createFutureNativeRemoteNetworkRuntime,
} from './connection/browserNetworkRuntime'
export type { BrowserRemoteNetworkRuntimeOptions } from './connection/browserNetworkRuntime'
export type { MachineConnectionSnapshot } from './connection/machineConnectionSnapshot'
export {
  cloudPresenceEdgeLabel,
  cloudPresenceReachability,
  sameCloudPresenceSnapshot,
} from './connection/cloudPresence'
export type {
  CloudPresenceInput,
  CloudPresenceReachability,
  CloudPresenceSnapshot,
} from './connection/cloudPresence'
export { RemoteNetworkStateManager } from './connection/remoteNetworkState'
export type { NativeNetworkStatusPlugin, RemoteNetworkState, RemoteResumeType } from './connection/remoteNetworkState'
export { TerminalClient } from './terminal/terminalClient'
export type {
  TerminalClientCallbacks,
  TerminalInfoPayload,
  TerminalProtocolEvent,
  TerminalProtocolSession,
  TerminalSnapshotPayload,
} from './terminal/terminalClient'
export {
  copyHistorySelection,
  rangeFromHistorySelection,
  searchHistorySurface,
  selectionFromSurfaceRows,
} from './terminal/coreV2HistoryInteraction'
export {
  CoreV2HistorySurfaceStaleError,
  createCoreV2HistorySurface,
} from './terminal/coreV2HistorySurface'
export { createCoreV2HistorySource } from './terminal/coreV2HistorySource'
export type {
  CoreV2HistoryCellPoint,
  CoreV2HistorySearchMatch,
  CoreV2HistorySelection,
} from './terminal/coreV2HistoryInteraction'
export type {
  CoreV2HistoryRenderWindow,
  CoreV2HistorySurface,
  CoreV2HistorySurfaceLoadOptions,
  CoreV2HistorySurfaceOptions,
  CoreV2HistorySurfaceSnapshot,
} from './terminal/coreV2HistorySurface'
export type {
  CoreV2HistoryCell,
  CoreV2HistoryCellStyle,
  CoreV2HistoryCopyRequest,
  CoreV2HistoryCursor,
  CoreV2HistoryLineSpan,
  CoreV2HistoryRange,
  CoreV2HistoryReleaseRequest,
  CoreV2HistoryRow,
  CoreV2HistoryWindow,
  CoreV2HistoryWindowMode,
  CoreV2HistoryWindowOp,
  CoreV2HistoryWindowRequest,
} from './terminal/coreV2TerminalProtocol'
export type { CoreV2HistorySource } from './terminal/coreV2HistorySource'
export {
  DEFAULT_TERMINAL_SETTINGS,
  TERMINAL_FONT_OPTIONS,
  TERMINAL_SCROLL_INERTIA_OPTIONS,
  TERMINAL_SETTINGS_STORAGE_KEY,
  TERMINAL_THEME_OPTIONS,
  ANYTTY_DARK_TERMINAL_THEME,
  normalizeTerminalSettings,
  readTerminalSettings,
  resolveTerminalMomentumProfile,
  resolveTerminalThemeOption,
  resolveTerminalTheme,
  resolveTerminalThemeUi,
  terminalThemeCssVariables,
  writeTerminalSettings,
} from './terminal/terminalSettings'
export type {
  TerminalFontOption,
  TerminalKeyboardMode,
  TerminalMomentumProfile,
  TerminalScrollInertia,
  TerminalSettings,
  TerminalThemeId,
  TerminalThemeOption,
  TerminalThemeUi,
} from './terminal/terminalSettings'
export {
  haptic,
  hapticError,
  hapticImpact,
  hapticSelection,
  hapticSuccess,
  setHapticImpactHandler,
} from './platform/haptics'
export type { HapticImpactHandler, HapticPattern } from './platform/haptics'
export {
  createTerminalInventorySnapshot,
  normalizeTerminalInventory,
  selectTerminal,
} from './terminal/terminalInventory'
export type { TerminalInventoryInput, TerminalInventorySnapshot } from './terminal/terminalInventory'
export * from './core/transport'
export type { RemoteMachine } from './core/remoteMachine'
export { useFileManager } from './files/useFileManager'
export type { FileManagerVisibleError, UseFileManagerOptions, UseFileManagerResult } from './files/useFileManager'
export * from './terminal/useTerminalSession'
export * as AnyTTYApiAccessRemote from './generated/apipb/access_remote_pb'
export * as AnyTTYApiApplication from './generated/apipb/application_pb'
export * as AnyTTYApiCommon from './generated/apipb/common_pb'
export * as AnyTTYApiEvents from './generated/apipb/events_pb'
export * as AnyTTYApiFile from './generated/apipb/file_pb'
export * as AnyTTYApiHistory from './generated/apipb/history_pb'
export * as AnyTTYApiRuntime from './generated/apipb/runtime_pb'
export * as AnyTTYApiStorage from './generated/apipb/storage_pb'
export * as AnyTTYApiTerminal from './generated/apipb/terminal_pb'
export * as AnyTTYApiWorkbench from './generated/apipb/workbench_pb'
export * as AnyTTYClientBinding from './generated/bindingpb/client_binding_pb'
export type {
  ProtoClientSession,
  ProtoClientSubscription,
  ProtoResourceStream,
} from './core/protoClientSession'
export { BindingOperation, ProtoBindingClient, ProtoBindingConnector } from './binding/protoBindingClient'
export type { BindingOperationCode, EndpointInput, ProtoBindingBackend } from './binding/protoBindingClient'
export * as AnyTTYRemoteAuth from './generated/remoteauthpb/remote_auth_pb'
