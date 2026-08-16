import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  createFileApi,
  createFilePreviewSource,
  type DirListResponse,
  type FileApi,
  type FileEntry,
  type FilePreviewResponse,
  type FilePreviewStreamOptions,
  type FilePreviewStreamResult,
} from './fileApi'
import type { ConnectionInfo } from '../core/transport'
import type { RemoteRuntimeStorage } from '../core/transport'
import type { ProtoClientSession } from '../core/protoClientSession'
import { anyttyI18n } from '../i18n'
import { createPersistentPathBookmarkApi, type PathBookmark } from './pathBookmarks'
import { fileEntryPath, joinPath, normalizeFilePath, parentPath } from './fileUtils'
import { isKnownUnsupportedPreviewPath } from './preview/filePreviewSupport'

export type FileSortField = 'name' | 'modified' | 'size' | 'type'
export type FileSortDirection = 'asc' | 'desc'

export interface FileSortState {
  field: FileSortField
  direction: FileSortDirection
}

export const DEFAULT_FILE_SORT: FileSortState = { field: 'name', direction: 'asc' }

export interface FileManagerVisibleError {
  message: string
  surface: 'banner' | 'toast' | 'modal'
  recoverable: boolean
}

export interface UseFileManagerOptions {
  machineId: string
  terminalId?: string | undefined
  session: ProtoClientSession
  initialPath?: string | undefined
  storage?: RemoteRuntimeStorage | undefined
}

export interface UseFileManagerResult {
  machineId: string
  terminalId?: string | undefined
  currentPath: string
  entries: FileEntry[]
  visibleEntries: FileEntry[]
  total: number
  loading: boolean
  error: FileManagerVisibleError | null
  sortState: FileSortState
  showHidden: boolean
  newDirName: string
  creatingDirectory: boolean
  actionMessage: string | null
  preview: FilePreviewResponse | null
  previewPath: string | null
  previewLoading: boolean
  previewError: FileManagerVisibleError | null
  fileApi: FileApi
  selectionMode: boolean
  selectedPaths: Set<string>
  clipboard: { mode: 'copy' | 'cut'; paths: string[] } | null
  pathBookmarks: PathBookmark[]
  pathBookmarksLoading: boolean
  pathBookmarkError: string | null
  setSelectionMode(mode: boolean): void
  toggleSelect(path: string): void
  selectAll(): void
  deselectAll(): void
  setClipboard(clipboard: { mode: 'copy' | 'cut'; paths: string[] } | null): void
  copy(paths: string[]): void
  cut(paths: string[]): void
  copyFilePaths(paths: string[]): Promise<void>
  paste(): Promise<void>
  batchDelete(paths: string[]): Promise<void>
  setNewDirName(name: string): void
  setSort(sort: FileSortState): void
  toggleShowHidden(): void
  openEntry(path: string, entry: FileEntry): Promise<void>
  openPreview(path: string): Promise<void>
  streamPreview(path: string, mimeType: string, options?: FilePreviewStreamOptions): Promise<FilePreviewStreamResult>
  closePreview(): void
  createDirectory(): Promise<void>
  deleteEntry(path: string): Promise<void>
  renameEntry(path: string, newName: string): Promise<void>
  navigate(path: string): Promise<void>
  refresh(): Promise<void>
  addCurrentPathBookmark(): Promise<void>
  updatePathBookmark(id: string, input: { label?: string | undefined; path?: string | undefined }): Promise<void>
  removePathBookmark(id: string): Promise<void>
  refreshPathBookmarks(): Promise<void>
}

function fileSessionConnectionInfo(session: UseFileManagerOptions['session']): Promise<ConnectionInfo> {
  return Promise.resolve({
    path: 'hub', connectionId: `${session.stamp.endpointId}:${session.stamp.generation}`,
    machineId: session.stamp.endpointId, relayInUse: false,
  })
}

export function useFileManager(options: UseFileManagerOptions): UseFileManagerResult {
  const [currentPath, setCurrentPath] = useState(options.initialPath ?? '')
  const [entries, setEntries] = useState<FileEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [hasLoadedEntries, setHasLoadedEntries] = useState(false)
  const [error, setError] = useState<FileManagerVisibleError | null>(null)
  const [sortState, setSortState] = useState<FileSortState>(DEFAULT_FILE_SORT)
  const [showHidden, setShowHidden] = useState(false)
  const [newDirName, setNewDirName] = useState('')
  const [creatingDirectory, setCreatingDirectory] = useState(false)
  const [actionMessage, setActionMessage] = useState<string | null>(null)
  const [preview, setPreview] = useState<FilePreviewResponse | null>(null)
  const [previewPath, setPreviewPath] = useState<string | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewError, setPreviewError] = useState<FileManagerVisibleError | null>(null)
  const currentPathRef = useRef(options.initialPath ?? '')
  const requestSeqRef = useRef(0)
  const previewSeqRef = useRef(0)
  const hasLoadedEntriesRef = useRef(hasLoadedEntries)

  const fileApi = useMemo(() => createFileApi(options.session), [options.session])
  const previewSource = useMemo(() => createFilePreviewSource(options.session), [options.session])
  const bookmarkApi = useMemo(
    () => createPersistentPathBookmarkApi(options.machineId, options.session, options.storage),
    [options.machineId, options.session, options.storage],
  )

  const [selectionMode, setSelectionMode] = useState(false)
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set())
  const [clipboard, setClipboard] = useState<{ mode: 'copy' | 'cut'; paths: string[] } | null>(null)
  const [pathBookmarks, setPathBookmarks] = useState<PathBookmark[]>([])
  const [pathBookmarksLoading, setPathBookmarksLoading] = useState(false)
  const [pathBookmarkError, setPathBookmarkError] = useState<string | null>(null)

  useEffect(() => {
    hasLoadedEntriesRef.current = hasLoadedEntries
  }, [hasLoadedEntries])

  const showActionMessage = useCallback((message: string) => {
    setActionMessage(message)
    window.setTimeout(() => setActionMessage(null), 2500)
  }, [])

  const loadPath = useCallback(async (path: string) => {
    const seq = ++requestSeqRef.current
    const isNavigation = path !== currentPathRef.current

    if (!hasLoadedEntriesRef.current) setLoading(true)

    const loadingTimer = window.setTimeout(() => {
      if (seq === requestSeqRef.current) {
        setLoading(true)
        if (isNavigation) {
          setEntries([])
        }
      }
    }, 200)

    setError(null)
    try {
      const info = await fileSessionConnectionInfo(options.session)
      if (seq !== requestSeqRef.current) return
      assertSessionTarget(info, options.machineId, options.terminalId)
      const { response, fallbackFrom } = await listDirWithParentFallback(fileApi, path)
      if (seq !== requestSeqRef.current) return
      setCurrentPath(response.path)
      currentPathRef.current = response.path
      setEntries(response.entries)
      setTotal(response.total)
      setHasLoadedEntries(true)
      if (fallbackFrom && fallbackFrom !== response.path) {
        showActionMessage(anyttyI18n.t('files.feedback.openedFallback', { path: response.path }))
      }
    } catch (err) {
      if (seq !== requestSeqRef.current) return
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'banner',
        recoverable: true,
      })
    } finally {
      clearTimeout(loadingTimer)
      if (seq === requestSeqRef.current) setLoading(false)
    }
  }, [fileApi, options.machineId, options.terminalId, options.session, showActionMessage])

  const toggleSelect = useCallback((path: string) => {
    setSelectedPaths(prev => {
      const next = new Set(prev)
      if (next.has(path)) next.delete(path)
      else next.add(path)
      return next
    })
  }, [])

  const visibleEntries = useMemo(
    () => sortFileEntries(
      showHidden ? entries : entries.filter((entry) => !entry.name.startsWith('.')),
      sortState,
    ),
    [entries, showHidden, sortState],
  )

  const selectAll = useCallback(() => {
    setSelectedPaths(new Set(visibleEntries.map(e => fileEntryPath(currentPathRef.current, e))))
  }, [visibleEntries])

  const deselectAll = useCallback(() => {
    setSelectedPaths(new Set())
  }, [])

  const copy = useCallback((paths: string[]) => {
    setClipboard({ mode: 'copy', paths })
    showActionMessage(anyttyI18n.t('files.feedback.copiedItems', { count: paths.length }))
  }, [showActionMessage])

  const cut = useCallback((paths: string[]) => {
    setClipboard({ mode: 'cut', paths })
    showActionMessage(anyttyI18n.t('files.feedback.cutItems', { count: paths.length }))
  }, [showActionMessage])

  const copyFilePaths = useCallback(async (paths: string[]) => {
    if (paths.length === 0) return
    try {
      await writeTextToClipboard(paths.join('\n'))
      showActionMessage(anyttyI18n.t('files.copiedPaths', { count: paths.length }))
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'toast',
        recoverable: true,
      })
    }
  }, [showActionMessage])

  const paste = useCallback(async () => {
    if (!clipboard || clipboard.paths.length === 0) return
    setError(null)
    if (!hasLoadedEntriesRef.current) setLoading(true)
    try {
      if (clipboard.mode === 'copy') {
        await fileApi.copy(clipboard.paths, currentPathRef.current)
        showActionMessage(anyttyI18n.t('files.feedback.pastedItems', { count: clipboard.paths.length }))
      } else {
        await fileApi.move(clipboard.paths, currentPathRef.current)
        showActionMessage(anyttyI18n.t('files.feedback.movedItems', { count: clipboard.paths.length }))
        setClipboard(null) // clear clipboard after move
      }
      await loadPath(currentPathRef.current)
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'toast',
        recoverable: true,
      })
      setLoading(false)
    }
  }, [clipboard, fileApi, loadPath, showActionMessage])

  const batchDelete = useCallback(async (paths: string[]) => {
    if (paths.length === 0) return
    setError(null)
    if (!hasLoadedEntriesRef.current) setLoading(true)
    try {
      await fileApi.batchDelete(paths)
      showActionMessage(anyttyI18n.t('files.feedback.deletedItems', { count: paths.length }))
      setSelectedPaths(new Set())
      setSelectionMode(false)
      await loadPath(currentPathRef.current)
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'toast',
        recoverable: true,
      })
      setLoading(false)
    }
  }, [fileApi, loadPath, showActionMessage])

  useEffect(() => {
    if (!selectionMode) setSelectedPaths(new Set())
  }, [selectionMode])

  useEffect(() => {
    void loadPath(currentPathRef.current)
    return () => {
      requestSeqRef.current += 1
    }
  }, [loadPath])

  const navigate = useCallback(async (path: string) => {
    await loadPath(path)
  }, [loadPath])

  const refresh = useCallback(async () => {
    await loadPath(currentPathRef.current)
  }, [loadPath])

  const toggleShowHidden = useCallback(() => {
    setShowHidden((current) => !current)
  }, [])

  const refreshPathBookmarks = useCallback(async () => {
    setPathBookmarksLoading(true)
    setPathBookmarkError(null)
    try {
      setPathBookmarks(await bookmarkApi.list())
    } catch (err) {
      setPathBookmarkError(err instanceof Error ? err.message : String(err))
    } finally {
      setPathBookmarksLoading(false)
    }
  }, [bookmarkApi])

  const addCurrentPathBookmark = useCallback(async () => {
    const path = normalizeAbsolutePath(currentPathRef.current || '/')
    setPathBookmarkError(null)
    try {
      await bookmarkApi.add(path)
      showActionMessage(anyttyI18n.t('files.bookmarks.added', { path }))
      await refreshPathBookmarks()
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setPathBookmarkError(message)
      setError({ message, surface: 'toast', recoverable: true })
    }
  }, [bookmarkApi, refreshPathBookmarks, showActionMessage])

  const updatePathBookmark = useCallback(async (id: string, input: { label?: string | undefined; path?: string | undefined }) => {
    setPathBookmarkError(null)
    try {
      await bookmarkApi.update(id, input)
      showActionMessage(anyttyI18n.t('files.bookmarks.updated'))
      await refreshPathBookmarks()
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setPathBookmarkError(message)
      setError({ message, surface: 'toast', recoverable: true })
    }
  }, [bookmarkApi, refreshPathBookmarks, showActionMessage])

  const removePathBookmark = useCallback(async (id: string) => {
    setPathBookmarkError(null)
    try {
      await bookmarkApi.remove(id)
      showActionMessage(anyttyI18n.t('files.bookmarks.removed'))
      await refreshPathBookmarks()
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setPathBookmarkError(message)
      setError({ message, surface: 'toast', recoverable: true })
    }
  }, [bookmarkApi, refreshPathBookmarks, showActionMessage])

  const setSort = useCallback((sort: FileSortState) => {
    setSortState(sort)
  }, [])

  const openPreview = useCallback(async (path: string) => {
    const seq = ++previewSeqRef.current
    setPreviewPath(path)
    setPreview(null)
    setPreviewError(null)
    if (isKnownUnsupportedPreviewPath(path)) {
      setPreviewLoading(false)
      setPreviewError({
        message: anyttyI18n.t('files.preview.unsupported'),
        surface: 'modal',
        recoverable: true,
      })
      return
    }
    setPreviewLoading(true)
    try {
      const info = await fileSessionConnectionInfo(options.session)
      if (seq !== previewSeqRef.current) return
      assertSessionTarget(info, options.machineId, options.terminalId)
      const response = await previewSource.preview(path)
      if (seq !== previewSeqRef.current) return
      setPreview(response)
    } catch (err) {
      if (seq !== previewSeqRef.current) return
      setPreviewError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'modal',
        recoverable: true,
      })
    } finally {
      if (seq === previewSeqRef.current) setPreviewLoading(false)
    }
  }, [options.machineId, options.terminalId, options.session, previewSource])

  const openEntry = useCallback(async (path: string, entry: FileEntry) => {
    setError(null)
    try {
      const resolved = await resolveLinkedEntry(fileApi, path, entry)
      if (resolved.entry.type === 'dir' || resolved.entry.type === 'symlink-dir') {
        await loadPath(resolved.path)
      } else {
        await openPreview(resolved.path)
      }
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'toast',
        recoverable: true,
      })
    }
  }, [fileApi, loadPath, openPreview])

  const streamPreview = useCallback(async (
    path: string,
    mimeType: string,
    streamOptions?: FilePreviewStreamOptions,
  ) => {
    const info = await fileSessionConnectionInfo(options.session)
    assertSessionTarget(info, options.machineId, options.terminalId)
    return await previewSource.stream(path, mimeType, streamOptions)
  }, [options.machineId, options.terminalId, options.session, previewSource])

  const closePreview = useCallback(() => {
    previewSeqRef.current += 1
    setPreview(null)
    setPreviewPath(null)
    setPreviewError(null)
    setPreviewLoading(false)
  }, [])

  const createDirectory = useCallback(async () => {
    const name = newDirName.trim()
    if (!name) return
    setCreatingDirectory(true)
    setError(null)
    try {
      await fileApi.mkdir(joinPath(currentPathRef.current || '/', name))
      setNewDirName('')
      showActionMessage(anyttyI18n.t('files.feedback.createdDirectory', { name }))
      await loadPath(currentPathRef.current)
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'toast',
        recoverable: true,
      })
    } finally {
      setCreatingDirectory(false)
    }
  }, [fileApi, loadPath, newDirName, showActionMessage])

  const deleteEntry = useCallback(async (path: string) => {
    setError(null)
    try {
      await fileApi.delete(path)
      showActionMessage(anyttyI18n.t('files.feedback.deletedEntry', { name: basename(path) }))
      await loadPath(currentPathRef.current)
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'toast',
        recoverable: true,
      })
    }
  }, [fileApi, loadPath, showActionMessage])

  const renameEntry = useCallback(async (path: string, newName: string) => {
    const name = newName.trim()
    if (!name) return
    setError(null)
    try {
      await fileApi.rename(path, joinPath(parentPath(path), name))
      showActionMessage(anyttyI18n.t('files.feedback.renamedEntry', { name }))
      await loadPath(currentPathRef.current)
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : String(err),
        surface: 'toast',
        recoverable: true,
      })
    }
  }, [fileApi, loadPath, showActionMessage])

  return {
    machineId: options.machineId,
    terminalId: options.terminalId,
    currentPath,
    entries,
    visibleEntries,
    total,
    loading,
    error,
    sortState,
    showHidden,
    newDirName,
    creatingDirectory,
    actionMessage,
    preview,
    previewPath,
    previewLoading,
    previewError,
    fileApi,
    selectionMode,
    selectedPaths,
    clipboard,
    pathBookmarks,
    pathBookmarksLoading,
    pathBookmarkError,
    setSelectionMode,
    toggleSelect,
    selectAll,
    deselectAll,
    setClipboard,
    copy,
    cut,
    copyFilePaths,
    paste,
    batchDelete,
    setNewDirName,
    setSort,
    toggleShowHidden,
    openEntry,
    openPreview,
    streamPreview,
    closePreview,
    createDirectory,
    deleteEntry,
    renameEntry,
    navigate,
    refresh,
    addCurrentPathBookmark,
    updatePathBookmark,
    removePathBookmark,
    refreshPathBookmarks,
  }
}

async function resolveLinkedEntry(
  fileApi: FileApi,
  initialPath: string,
  initialEntry: FileEntry,
): Promise<{ path: string; entry: FileEntry }> {
  let path = normalizeFilePath(initialPath)
  let entry = initialEntry
  const visited = new Set<string>()

  while (entry.linkTarget) {
    if (visited.has(path)) throw new Error(anyttyI18n.t('files.linkLoop'))
    visited.add(path)
    const target = normalizeFilePath(entry.linkTarget)
    const targetPath = isAbsoluteFilePath(target) ? target : joinPath(parentPath(path), target)
    entry = await fileApi.stat(targetPath)
    path = normalizeFilePath(entry.path || targetPath)
  }

  return { path, entry }
}

function isAbsoluteFilePath(path: string): boolean {
  return path.startsWith('/') || /^[A-Za-z]:\//.test(path)
}

function sortFileEntries(entries: FileEntry[], sortState: FileSortState): FileEntry[] {
  return entries
    .map((entry, index) => ({ entry, index }))
    .sort((left, right) => {
      const directoryOrder = directoryRank(left.entry) - directoryRank(right.entry)
      if (directoryOrder !== 0) return directoryOrder

      const valueOrder = compareFileEntries(left.entry, right.entry, sortState)
      if (valueOrder !== 0) return valueOrder

      const nameOrder = compareNames(left.entry.name, right.entry.name)
      if (nameOrder !== 0) return nameOrder

      return left.index - right.index
    })
    .map(({ entry }) => entry)
}

function compareFileEntries(left: FileEntry, right: FileEntry, sortState: FileSortState): number {
  const direction = sortState.direction === 'asc' ? 1 : -1
  if (sortState.field === 'name') {
    return compareNames(left.name, right.name) * direction
  }
  if (sortState.field === 'modified') {
    return compareNumbers(modifiedTimeValue(left), modifiedTimeValue(right)) * direction
  }
  if (sortState.field === 'size') {
    return compareNumbers(left.size, right.size) * direction
  }
  return compareNames(fileExtension(left.name), fileExtension(right.name)) * direction
}

function directoryRank(entry: FileEntry): number {
  return entry.type === 'dir' || entry.type === 'symlink-dir' ? 0 : 1
}

function compareNames(left: string, right: string): number {
  return left.localeCompare(right, undefined, {
    numeric: true,
    sensitivity: 'base',
  })
}

function compareNumbers(left: number, right: number): number {
  if (left === right) return 0
  return left < right ? -1 : 1
}

function modifiedTimeValue(entry: FileEntry): number {
  if (!entry.modTime) return 0
  const timestamp = Date.parse(entry.modTime)
  return Number.isFinite(timestamp) ? timestamp : 0
}

function fileExtension(name: string): string {
  const base = basename(name).toLowerCase()
  const dot = base.lastIndexOf('.')
  return dot >= 0 ? base.slice(dot + 1) : ''
}

async function listDirWithParentFallback(
  fileApi: FileApi,
  path: string,
): Promise<{ response: DirListResponse; fallbackFrom?: string | undefined }> {
  const requested = normalizeAbsolutePath(path)
  let current = requested
  let fallbackFrom: string | undefined
  let lastErr: unknown
  for (;;) {
    try {
      return { response: await fileApi.listDir(current), fallbackFrom }
    } catch (err) {
      lastErr = err
      const parent = parentPath(current)
      if (parent === current) break
      fallbackFrom = requested
      current = parent
    }
  }
  throw lastErr instanceof Error ? lastErr : new Error(String(lastErr))
}

function normalizeAbsolutePath(path: string): string {
  return normalizeFilePath(path)
}

function basename(path: string): string {
  const normalized = path.replace(/\/+$/, '')
  return normalized.slice(normalized.lastIndexOf('/') + 1) || normalized
}

async function writeTextToClipboard(text: string): Promise<void> {
  let clipboardErr: unknown
  if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch (err) {
      clipboardErr = err
    }
  }
  if (typeof document === 'undefined') {
    throw clipboardErr instanceof Error ? clipboardErr : new Error(anyttyI18n.t('files.clipboardUnavailable'))
  }
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.left = '-9999px'
  textarea.style.top = '0'
  document.body.appendChild(textarea)
  textarea.select()
  try {
    if (!document.execCommand('copy')) {
      throw new Error(anyttyI18n.t('files.clipboardCopyFailed'))
    }
  } finally {
    document.body.removeChild(textarea)
  }
}

function assertSessionTarget(info: ConnectionInfo, machineId: string, terminalId?: string): void {
  if (info.machineId !== machineId) {
    throw new Error(`file session machine mismatch: connected to ${info.machineId}, expected ${machineId}`)
  }
  if (terminalId !== undefined && info.terminalId !== undefined && info.terminalId !== terminalId) {
    throw new Error(`file session terminal mismatch: connected to ${info.terminalId}, expected ${terminalId}`)
  }
}
