import { useFileManager, type FileSortState } from './useFileManager'
import type { FileTransferContext } from './fileApi'
import { extension, fileEntryMenuSubtitle, fileEntryMeta, fileEntryPath, isMarkdownFile, normalizeFilePath, parentPath, pathBreadcrumbs } from './fileUtils'
import { isModelPreviewFile } from './modelFileTypes'
import { FilePreviewSheet } from './preview/FilePreviewSheet'
import type { ProtoClientSession } from '../core/protoClientSession'
import type { RemoteRuntimeStorage } from '../core/transport'
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import type { PathBookmark } from './pathBookmarks'
import 'highlight.js/styles/github.css'
import { NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import { useNativeBackHandler } from '../platform/useNativeBackHandler'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import { ActionSheet, type ActionSheetItem } from '../ui/ActionSheet'
import { ModalSurface } from '../ui/ModalSurface'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { Spinner } from '../ui/spinner'
import { AlertCircle, ArrowDownAZ, ArrowDownToLine, ArrowUpAZ, ArrowUpFromLine, Bookmark, BookmarkPlus, Box, Check, ChevronRight, ClipboardCopy, ClipboardPaste, Clock, Code2, Eye, EyeOff, File, FileText, FileType, Folder, FolderBookmark, HardDrive, Image, ListChecks, ListFilter, MoreVertical, PlaySquare, RefreshCw, SquarePen, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import '../i18n'

export interface FileManagerProps {
  machineId: string
  terminalId?: string | undefined
  session: ProtoClientSession
  initialPath?: string | undefined
  className?: string | undefined
  active?: boolean | undefined
  unavailableLabel?: string | undefined
  fileTransfer?: FileTransferContext | undefined
  onOpenTransferCenter?: (() => void) | undefined
  storage?: RemoteRuntimeStorage | undefined
}

export function FileManager({
  machineId,
  terminalId,
  session,
  initialPath,
  className,
  active = true,
  unavailableLabel,
  fileTransfer,
  onOpenTransferCenter,
  storage,
}: FileManagerProps) {
  const { t } = useTranslation()
  const manager = useFileManager({ machineId, terminalId, session, initialPath, storage })
  const [newDirOpen, setNewDirOpen] = useState(false)
  const [entryMenuPath, setEntryMenuPath] = useState<string | null>(null)
  const [renamePath, setRenamePath] = useState<string | null>(null)
  const [renameName, setRenameName] = useState('')
  const [deletePath, setDeletePath] = useState<string | null>(null)
  const [sortMenuOpen, setSortMenuOpen] = useState(false)
  const [bookmarksOpen, setBookmarksOpen] = useState(false)
  const [fileToolsOpen, setFileToolsOpen] = useState(false)
  const [editingBookmarkId, setEditingBookmarkId] = useState<string | null>(null)
  const [bookmarkAlias, setBookmarkAlias] = useState('')
  const [transferError, setTransferError] = useState<string | null>(null)
  const deleteConfirmTitleId = useId()
  const deleteConfirmDescriptionId = useId()
  const bookmarkEditorTitleId = useId()
  const bookmarkEditorDescriptionId = useId()
  const pathBarRef = useRef<HTMLDivElement>(null)
  const webUploadRef = useRef<HTMLInputElement>(null)
  const bookmarkAliasInputRef = useRef<HTMLInputElement>(null)
  const rootRef = useRef<HTMLDivElement>(null)

  const breadcrumbs = pathBreadcrumbs(manager.currentPath)
  const entryKeyCounts = new Map<string, number>()
  const sortLabel = fileSortLabel(manager.sortState, t)

  const menuEntry = useMemo(() => {
    if (!entryMenuPath) return null
    return manager.entries.find(e => fileEntryPath(manager.currentPath, e) === entryMenuPath)
  }, [entryMenuPath, manager.entries, manager.currentPath])

  const menuActions: ActionSheetItem[] = useMemo(() => {
    if (!menuEntry || !entryMenuPath) return []
    const isDirectory = menuEntry.type === 'dir' || menuEntry.type === 'symlink-dir'
    const actions: ActionSheetItem[] = []

    if (!isDirectory) {
      actions.push({
        label: t('files.actions.preview'),
        icon: <Eye className="h-5 w-5" />,
        onClick: () => void manager.openPreview(entryMenuPath),
      })
    }

    if (!isDirectory && fileTransfer) {
      actions.push({
        label: t('files.actions.download'),
        icon: <ArrowDownToLine className="h-5 w-5" />,
        onClick: async () => {
          setTransferError(null)
          try {
            const resumeOffset = Math.max(
              0,
              Math.min(
                menuEntry.size,
                await Promise.resolve(fileTransfer.getDownloadResumeOffset?.(machineId, entryMenuPath, menuEntry.size) ?? 0),
              ),
            )
            fileTransfer.startDownload(
              machineId,
              menuEntry.name,
              menuEntry.size,
              entryMenuPath,
              resumeOffset,
            )
            onOpenTransferCenter?.()
          } catch (err) {
            setTransferError(err instanceof Error ? err.message : String(err))
          }
        },
      })
    }

    actions.push({
      label: t('files.actions.copyPath'),
      icon: <ClipboardCopy className="h-5 w-5" />,
      onClick: () => { hapticImpact(); void manager.copyFilePaths([entryMenuPath]) },
    })

    if (manager.currentPath === '/' && /^[A-Za-z]:\/$/.test(entryMenuPath)) return actions

    actions.push({
      label: t('files.actions.copy'),
      icon: <File className="h-5 w-5" />,
      onClick: () => { hapticImpact(); manager.copy([entryMenuPath]) },
    })

    actions.push({
      label: t('files.actions.cut'),
      icon: <Folder className="h-5 w-5" />,
      onClick: () => { hapticImpact(); manager.cut([entryMenuPath]) },
    })

    actions.push({
      label: t('files.actions.rename'),
      icon: <SquarePen className="h-5 w-5" />,
      onClick: () => {
        setRenamePath(entryMenuPath)
        setRenameName(menuEntry.name)
      },
    })

    actions.push({
      label: t('files.actions.delete'),
      icon: <Trash2 className="h-5 w-5" />,
      onClick: () => setDeletePath(entryMenuPath),
      danger: true,
      closeOnClick: false,
    })

    return actions
  }, [menuEntry, entryMenuPath, manager, fileTransfer, machineId, onOpenTransferCenter, t])

  const sortActions: ActionSheetItem[] = useMemo(() => fileSortOptions.map((option) => {
    const active = option.field === manager.sortState.field && option.direction === manager.sortState.direction
    return {
      label: `${active ? `${t('files.sort.selected')}: ` : ''}${t(option.labelKey)}`,
      icon: active ? <Check className="h-5 w-5" /> : option.icon,
      onClick: () => manager.setSort({ field: option.field, direction: option.direction }),
    }
  }), [manager, t])

  const editingBookmark = useMemo(() => (
    editingBookmarkId ? manager.pathBookmarks.find((bookmark) => bookmark.id === editingBookmarkId) ?? null : null
  ), [editingBookmarkId, manager.pathBookmarks])

  const closeBookmarkEditor = () => {
    setEditingBookmarkId(null)
    setBookmarkAlias('')
  }

  const currentPath = normalizeFilePath(manager.currentPath)
  const clipboardOpen = Boolean(manager.clipboard?.paths.length)
  const nestedOverlayOpen = Boolean(
    deletePath
      || manager.previewPath
      || entryMenuPath
      || sortMenuOpen
      || fileToolsOpen
      || bookmarksOpen
      || editingBookmark,
  )

  useNativeBackHandler(() => {
    if (deletePath) {
      setDeletePath(null)
      return
    }
    if (manager.previewPath) {
      manager.closePreview()
      return
    }
    if (entryMenuPath) {
      setEntryMenuPath(null)
      return
    }
    if (sortMenuOpen) {
      setSortMenuOpen(false)
      return
    }
    if (fileToolsOpen) {
      setFileToolsOpen(false)
      return
    }
    if (bookmarksOpen && editingBookmark) {
      closeBookmarkEditor()
      return
    }
    if (bookmarksOpen) {
      setBookmarksOpen(false)
      closeBookmarkEditor()
      return
    }
  }, NATIVE_BACK_PRIORITY.NESTED_OVERLAY, active && nestedOverlayOpen)

  useNativeBackHandler(() => {
    void manager.navigate(parentPath(currentPath))
  }, NATIVE_BACK_PRIORITY.WORKSPACE, active && currentPath !== '/')

  useNativeBackHandler(() => {
    manager.setNewDirName('')
    setNewDirOpen(false)
  }, NATIVE_BACK_PRIORITY.WORKSPACE, active && newDirOpen)

  useNativeBackHandler(() => {
    setRenamePath(null)
    setRenameName('')
  }, NATIVE_BACK_PRIORITY.WORKSPACE, active && Boolean(renamePath))

  useNativeBackHandler(() => {
    manager.setSelectionMode(false)
  }, NATIVE_BACK_PRIORITY.WORKSPACE, active && manager.selectionMode)

  useNativeBackHandler(() => {
    manager.setClipboard(null)
  }, NATIVE_BACK_PRIORITY.WORKSPACE, active && clipboardOpen)

  const openBookmarkEditor = (bookmark: PathBookmark) => {
    setEditingBookmarkId(bookmark.id)
    setBookmarkAlias(bookmark.label)
  }

  const saveBookmarkAlias = () => {
    if (!editingBookmark) return
    const label = bookmarkAlias.trim()
    hapticImpact()
    void manager.updatePathBookmark(editingBookmark.id, { label }).then(closeBookmarkEditor)
  }

  useEffect(() => {
    const pathBar = pathBarRef.current
    if (!pathBar) return undefined
    const frame = window.requestAnimationFrame(() => {
      pathBar.scrollLeft = pathBar.scrollWidth
    })
    return () => window.cancelAnimationFrame(frame)
  }, [manager.currentPath])

  useEffect(() => {
    if (!rootRef.current) return
    rootRef.current.inert = !active
  }, [active])

  return (
    <div
      ref={rootRef}
      aria-disabled={!active}
      className={`relative flex min-h-0 flex-col bg-[var(--background)] ${className || ''}`}
      data-machine-id={machineId}
      data-terminal-id={terminalId}
      data-testid="anytty-file-manager"
    >
      {manager.selectionMode ? (
        <header className="flex h-12 shrink-0 items-center justify-between border-b border-zinc-200/70 bg-[var(--background)] px-4">
          <Button variant="ghost"
            className="text-[15px] font-medium text-zinc-500 hover:text-zinc-700 active:text-zinc-800"
            onClick={() => { hapticSelection(); manager.setSelectionMode(false) }}
          >
            {t('common.cancel')}
          </Button>
          <div className="text-[16px] font-semibold text-zinc-900">
            {t('files.selected', { count: manager.selectedPaths.size })}
          </div>
          <Button variant="ghost"
            className="text-[15px] font-medium text-zinc-950 hover:text-zinc-700 active:text-zinc-500"
            onClick={() => {
              hapticSelection()
              if (manager.selectedPaths.size === manager.visibleEntries.length) manager.deselectAll()
              else manager.selectAll()
            }}
          >
            {t(manager.selectedPaths.size === manager.visibleEntries.length ? 'files.deselectAll' : 'files.selectAll')}
          </Button>
        </header>
      ) : (
        <header className="shrink-0 border-b border-zinc-200/70 bg-[var(--background)] pb-2">
          <div
            className="mx-3 my-2 flex h-10 min-w-0 items-center overflow-hidden rounded-md border border-[var(--anytty-app-line)] bg-zinc-50"
            data-testid="anytty-file-path-toolbar"
          >
            <div
              ref={pathBarRef}
              data-testid="anytty-file-pathbar"
              className="flex h-full min-w-0 flex-1 items-center gap-1 overflow-x-auto bg-transparent px-2 text-[14px] font-medium text-zinc-600 no-scrollbar"
            >
              <HardDrive className="h-4 w-4 shrink-0 text-zinc-400" />
              {breadcrumbs.map((breadcrumb, index) => {
                const isLast = index === breadcrumbs.length - 1
                return (
                  <div key={breadcrumb.path} className="flex shrink-0 items-center">
                    {index > 0 && <ChevronRight className="h-3.5 w-3.5 shrink-0 text-zinc-300" />}
                    {isLast ? (
                      <span className="px-1.5 py-1 font-semibold text-zinc-900">{breadcrumb.label}</span>
                    ) : (
                      <Button variant="ghost"
                        onClick={() => { hapticSelection(); void manager.navigate(breadcrumb.path) }}
                        className="rounded px-1.5 py-1 text-zinc-500 transition-colors hover:bg-zinc-100 active:bg-zinc-200"
                      >
                        {breadcrumb.label}
                      </Button>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        </header>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto bg-[var(--background)]">
        {newDirOpen ? (
          <div className="mx-3 my-2 flex items-center gap-2 rounded-md border border-[var(--anytty-app-line)] bg-zinc-50 p-2">
            <Folder className="h-5 w-5 shrink-0 text-zinc-700" />
            <Input
              aria-label={t('files.directoryName')}
              className="min-h-10 flex-1 border-0 bg-transparent px-2 text-[15px] font-medium text-zinc-900 shadow-none placeholder:text-zinc-400"
              placeholder={t('files.directoryName')}
              value={manager.newDirName}
              onChange={(event) => manager.setNewDirName(event.currentTarget.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') void manager.createDirectory().then(() => setNewDirOpen(false))
                if (event.key === 'Escape') {
                  manager.setNewDirName('')
                  setNewDirOpen(false)
                }
              }}
              autoFocus
            />
            <Button variant="default" size="icon"
              aria-label={t('files.createDirectory')}
              className="h-11 w-12 shrink-0 disabled:bg-zinc-200 disabled:text-zinc-400"
              disabled={!manager.newDirName.trim() || manager.creatingDirectory}
              onClick={() => { hapticImpact(); void manager.createDirectory().then(() => setNewDirOpen(false)) }}
            >
              <Check className="h-4 w-4" />
            </Button>
            <Button variant="secondary" size="icon"
              aria-label={t('files.cancelNewDirectory')}
              className="h-11 w-12 shrink-0 text-zinc-600"
              onClick={() => {
                manager.setNewDirName('')
                setNewDirOpen(false)
              }}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        ) : null}

        {manager.pathBookmarkError ? (
          <div className="m-2 rounded-md border border-amber-200 bg-amber-50 px-4 py-2 text-[13px] font-medium text-amber-800" role="alert">
            {manager.pathBookmarkError}
          </div>
        ) : null}

        {manager.error ? (
          <div className="m-2 flex items-start gap-3 rounded-md border border-red-200 bg-red-50 p-4 text-[14px] text-red-800" role="alert">
            <AlertCircle className="h-6 w-6 shrink-0 text-red-500" />
            <div>
               <h3 className="font-bold text-red-900">{t('files.directoryError')}</h3>
               <p className="mt-1">{manager.error.message}</p>
            </div>
          </div>
        ) : null}

        {manager.actionMessage ? (
          <div className="m-2 rounded-md border border-emerald-200 bg-emerald-50 px-4 py-2 text-[13px] font-medium text-emerald-800" role="status">
            {manager.actionMessage}
          </div>
        ) : null}

        {transferError ? (
          <div className="m-2 rounded-md border border-red-200 bg-red-50 px-4 py-2 text-[13px] font-medium text-red-800" role="alert">
            {transferError}
          </div>
        ) : null}

        {manager.loading && manager.entries.length === 0 && !manager.error ? (
          <div className="flex h-40 flex-col items-center justify-center gap-3 text-[14px] font-medium text-zinc-500">
            <Spinner className="h-6 w-6 text-zinc-500" aria-hidden="true" />
            {t('files.loadingDirectory')}
          </div>
        ) : (
          <ul aria-label={t('files.list')} className="divide-y divide-zinc-100 pb-[120px]">
            {manager.entries.length === 0 && !manager.loading && !manager.error ? (
              <li className="flex h-32 flex-col items-center justify-center gap-3 border-y border-dashed border-zinc-300 bg-zinc-50/50 text-[14px] font-medium text-zinc-500">
                <Folder className="h-8 w-8 text-zinc-300" />
                {t('files.emptyDirectory')}
              </li>
            ) : null}
            {manager.visibleEntries.map((entry) => {
              const entryPath = fileEntryPath(manager.currentPath, entry)
              const isDirectory = entry.type === 'dir' || entry.type === 'symlink-dir'
              const Icon = isDirectory ? Folder : iconForFile(entry.name)
              const itemKey = uniqueFileListKey(entryKeyCounts, entryPath)
              const isRenaming = renamePath === entryPath
              const isSelected = manager.selectedPaths.has(entryPath)

              const handleItemClick = () => {
                if (manager.selectionMode) {
                  hapticSelection()
                  manager.toggleSelect(entryPath)
                } else {
                  hapticSelection()
                  void manager.openEntry(entryPath, entry)
                }
              }

              return (
                <li key={itemKey}>
                  <div
                    className={`group relative flex min-h-[3.25rem] w-full items-center gap-2.5 px-3 py-1.5 text-left transition-colors focus-within:ring-2 focus-within:ring-zinc-950 hover:bg-zinc-50 active:bg-zinc-100 ${isSelected ? 'bg-zinc-100' : ''}`}
                  >
                    {manager.selectionMode ? (
                      <div className="shrink-0 pr-1">
                        <div
                          className={`flex h-6 w-6 items-center justify-center rounded-full border-2 transition-colors ${isSelected ? 'border-[var(--primary)] bg-[var(--primary)]' : 'border-zinc-300 bg-transparent'}`}
                        >
                          {isSelected ? <Check className="h-4 w-4 text-[var(--primary-foreground)]" strokeWidth={3} /> : null}
                        </div>
                      </div>
                    ) : null}
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-[var(--anytty-app-line)] bg-zinc-50 transition-colors group-hover:bg-zinc-100 group-active:bg-zinc-200">
                      <Icon className={`h-5 w-5 ${isDirectory ? 'text-zinc-800' : 'text-zinc-400'}`} />
                    </div>
                    {isRenaming ? (
                      <Input
                          aria-label={t('files.renameEntry')}
                          className="min-h-11 min-w-0 flex-1 border-zinc-400 bg-[var(--background)] px-2 text-[15px] font-medium text-zinc-900 focus-visible:ring-[var(--ring)]"
                          value={renameName}
                          onClick={(event) => event.stopPropagation()}
                          onChange={(event) => setRenameName(event.currentTarget.value)}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter') {
                              event.preventDefault()
                              void manager.renameEntry(entryPath, renameName).then(() => {
                                setRenamePath(null)
                                setRenameName('')
                              })
                            }
                            if (event.key === 'Escape') {
                              setRenamePath(null)
                              setRenameName('')
                            }
                          }}
                          autoFocus
                      />
                    ) : (
                      <Button variant="ghost"
                        aria-label={t(manager.selectionMode ? (isSelected ? 'files.deselectEntry' : 'files.selectEntry') : isDirectory || entry.linkTarget ? 'files.openEntry' : 'files.previewEntry', { name: entry.name })}
                        className="flex h-auto min-w-0 flex-1 flex-col items-start justify-center overflow-hidden whitespace-normal px-0 text-left hover:bg-transparent"
                        onClick={handleItemClick}
                      >
                        <span className={`truncate text-[15px] leading-5 ${isDirectory ? 'font-medium text-zinc-950' : 'font-medium text-zinc-800'}`}>
                          {entry.name}
                        </span>
                        <span className="truncate text-[12px] font-medium text-zinc-500">
                          {fileEntryMeta(entry)}
                        </span>
                      </Button>
                    )}
                    {!manager.selectionMode ? (
                      <>
                        <div
                          data-testid="anytty-file-row-actions"
                          className="ml-auto flex w-16 shrink-0 items-center justify-end gap-1"
                        >
                          <Button variant="ghost"
                            type="button"
                            aria-label={t('files.moreActions', { name: entry.name })}
                            className="flex h-11 w-11 items-center justify-center rounded-md text-zinc-400 hover:bg-zinc-50 active:bg-zinc-100"
                            onClick={(event) => {
                              event.stopPropagation()
                              hapticSelection()
                              setEntryMenuPath((current) => current === entryPath ? null : entryPath)
                            }}
                          >
                            <MoreVertical className="h-5 w-5" />
                          </Button>
                          {isDirectory || entry.linkTarget ? <ChevronRight className="h-5 w-5 text-zinc-300 group-active:text-zinc-400" /> : <div className="h-5 w-5" />}
                        </div>
                      </>
                    ) : null}
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
      {deletePath ? renderFileManagerPortal(
        <div className="fixed inset-0 z-[110] flex items-end justify-center bg-black/40 p-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)] backdrop-blur-sm md:items-center" data-testid="anytty-file-delete-confirm" onClick={() => { hapticSelection(); setDeletePath(null) }}>
          <ModalSurface
            aria-labelledby={deleteConfirmTitleId}
            aria-describedby={deleteConfirmDescriptionId}
            className="rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] shadow-sm w-full p-4 md:max-w-sm"
            onClick={(event) => event.stopPropagation()}
            onRequestClose={() => setDeletePath(null)}
          >
            <h2 id={deleteConfirmTitleId} className="text-[17px] font-bold text-zinc-950">{t('files.deleteConfirm')}</h2>
            <p id={deleteConfirmDescriptionId} className="mt-2 break-all text-sm text-zinc-500">{deletePath}</p>
            <div className="mt-4 grid grid-cols-2 gap-3">
              <Button variant="secondary" className="h-11 font-semibold" onClick={() => { hapticSelection(); setDeletePath(null) }}>
                {t('common.cancel')}
              </Button>
              <Button variant="destructive"
                className="h-11 font-semibold"
                onClick={() => {
                  hapticImpact()
                  const target = deletePath
                  setDeletePath(null)
                  setEntryMenuPath(null)
                  void manager.deleteEntry(target)
                }}
              >
                {t('files.actions.delete')}
              </Button>
            </div>
          </ModalSurface>
        </div>,
      ) : null}
      {manager.previewPath ? (
        <FilePreviewSheet
          remoteAvailable={active}
          unavailableLabel={unavailableLabel}
          preview={manager.preview}
          path={manager.previewPath}
          loading={manager.previewLoading}
          error={manager.previewError?.message ?? null}
          streamPreview={manager.streamPreview}
          onClose={manager.closePreview}
        />
      ) : null}

      {/* Default Bottom Toolbar */}
      {!manager.selectionMode && !clipboardOpen ? (
        <div
          className="absolute bottom-0 left-0 right-0 z-40 bg-[var(--background)] backdrop-blur-xl border-t border-zinc-200 pb-[env(safe-area-inset-bottom)]"
          data-testid="anytty-file-toolbar"
        >
          <div className="flex h-[52px] items-center justify-around px-2">
            <Button variant="ghost"
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
              type="button"
              onClick={() => { hapticImpact(); manager.setSelectionMode(true) }}
              aria-label={t('files.selectFiles')}
            >
              <ListChecks className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.select')}</span>
            </Button>
            <Button variant="ghost"
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
              type="button"
              onClick={() => { hapticSelection(); setNewDirOpen((current) => !current) }}
              aria-label={t('files.newDirectory')}
            >
              <Folder className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.new')}</span>
            </Button>
            <Button variant="ghost"
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
              type="button"
              aria-label={t('files.bookmarks.title')}
              onClick={() => {
                hapticSelection()
                setBookmarksOpen(true)
                void manager.refreshPathBookmarks()
              }}
            >
              <FolderBookmark className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.bookmarks.title')}</span>
            </Button>
            <Button variant="ghost"
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
              type="button"
              aria-label={t('files.copyCurrentPath')}
              onClick={() => { hapticImpact(); void manager.copyFilePaths([manager.currentPath || '/']) }}
            >
              <ClipboardCopy className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.path')}</span>
            </Button>
            {fileTransfer ? (
              <>
                <Button variant="ghost"
                  className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
                  type="button"
                  aria-label={t('files.uploadFiles')}
                  onClick={() => {
                    hapticImpact()
                    if (fileTransfer.isNative) {
                      fileTransfer.pickAndUpload?.(machineId, manager.currentPath || '/')
                      onOpenTransferCenter?.()
                    } else {
                      webUploadRef.current?.click()
                    }
                  }}
                >
                  <ArrowUpFromLine className="h-5 w-5" />
                  <span className="text-[11px] font-medium">{t('files.actions.upload')}</span>
                </Button>
                {!fileTransfer.isNative ? (
                  <Input
                    ref={webUploadRef}
                    type="file"
                    multiple
                    className="hidden"
                    onChange={(e) => {
                      const files = e.target.files
                      if (!files) return
                      const picked = Array.from(files).map((f) => ({
                        uri: URL.createObjectURL(f),
                        name: f.name,
                        size: f.size,
                      }))
                      fileTransfer.startUpload(machineId, picked, manager.currentPath || '/')
                      onOpenTransferCenter?.()
                      e.target.value = ''
                    }}
                  />
                ) : null}
              </>
            ) : null}
            <Button variant="ghost"
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
              type="button"
              onClick={() => { hapticSelection(); setFileToolsOpen(true) }}
              aria-label={t('files.moreActions', { name: manager.currentPath || '/' })}
            >
              <MoreVertical className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.more')}</span>
            </Button>
          </div>
        </div>
      ) : null}

      {!manager.selectionMode && clipboardOpen ? (
        <div className="absolute bottom-0 left-0 right-0 z-40 border-t border-zinc-200 bg-[var(--background)] pb-[env(safe-area-inset-bottom)] backdrop-blur-xl" data-testid="anytty-file-clipboard-toolbar">
          <div className="flex h-[52px] items-center gap-2 px-3">
            <Button variant="secondary" className="h-10 min-w-0 flex-1 gap-2 px-3 font-semibold" onClick={() => { hapticSelection(); manager.setClipboard(null) }}>
              <X className="h-4 w-4" />
              {t('common.cancel')}
            </Button>
            <Button variant="default" className="h-10 min-w-0 flex-1 gap-2 px-3 font-semibold" onClick={() => { hapticImpact(); void manager.paste() }}>
              <ClipboardPaste className="h-4 w-4" />
              {t('files.actions.paste')}
            </Button>
          </div>
        </div>
      ) : null}

      {/* Selection Mode Bottom Bar */}
      {manager.selectionMode && manager.selectedPaths.size > 0 ? (
        <div className="absolute bottom-0 left-0 right-0 z-40 bg-[var(--background)] backdrop-blur-xl border-t border-zinc-200 pb-[env(safe-area-inset-bottom)]">
          <div className="flex h-[52px] items-stretch justify-around px-2">
            <Button variant="ghost"
              onClick={() => {
                hapticImpact()
                manager.copy(Array.from(manager.selectedPaths))
                manager.setSelectionMode(false)
              }}
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
            >
              <File className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.copy')}</span>
            </Button>
            <Button variant="ghost"
              onClick={() => {
                hapticImpact()
                void manager.copyFilePaths(Array.from(manager.selectedPaths)).then(() => manager.setSelectionMode(false))
              }}
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
            >
              <ClipboardCopy className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.path')}</span>
            </Button>
            <Button variant="ghost"
              onClick={() => {
                hapticImpact()
                manager.cut(Array.from(manager.selectedPaths))
                manager.setSelectionMode(false)
              }}
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 active:bg-zinc-200"
            >
              <Folder className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.cut')}</span>
            </Button>
            <Button variant="ghost"
              onClick={() => { hapticImpact(); void manager.batchDelete(Array.from(manager.selectedPaths)) }}
              className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 rounded-md px-2 text-red-500 hover:bg-red-50/80 hover:text-red-700 active:bg-red-50"
            >
              <Trash2 className="h-5 w-5" />
              <span className="text-[11px] font-medium">{t('files.actions.delete')}</span>
            </Button>
          </div>
        </div>
      ) : null}

      <ActionSheet
        isOpen={!!entryMenuPath}
        onClose={() => setEntryMenuPath(null)}
        title={menuEntry?.name}
        subtitle={menuEntry ? fileEntryMenuSubtitle(menuEntry) : ''}
        actions={menuActions}
      />
      <ActionSheet
        isOpen={sortMenuOpen}
        onClose={() => setSortMenuOpen(false)}
        title={t('files.sort.title')}
        subtitle={`${sortLabel} · ${t('files.sort.foldersFirst')}`}
        actions={sortActions}
      />
      <ActionSheet
        isOpen={fileToolsOpen}
        onClose={() => setFileToolsOpen(false)}
        title={t('files.moreActions', { name: manager.currentPath || '/' })}
        actions={[
          {
            label: t(manager.showHidden ? 'files.hideHidden' : 'files.showHidden'),
            icon: manager.showHidden ? <Eye className="h-5 w-5" /> : <EyeOff className="h-5 w-5" />,
            onClick: () => manager.toggleShowHidden(),
          },
          {
            label: t('files.sort.current', { sort: sortLabel }),
            icon: <ListFilter className="h-5 w-5" />,
            onClick: () => setSortMenuOpen(true),
          },
          {
            label: t('common.refresh'),
            icon: <RefreshCw className={`h-5 w-5 ${manager.loading ? 'animate-spin' : ''}`} />,
            onClick: () => void manager.refresh(),
          },
        ]}
      />
      <ActionSheet
        isOpen={bookmarksOpen && !editingBookmark}
        onClose={() => {
          setBookmarksOpen(false)
          closeBookmarkEditor()
        }}
        title={t('files.bookmarks.title')}
        subtitle={manager.pathBookmarksLoading ? t('common.loading') : t('files.bookmarks.saved', { count: manager.pathBookmarks.length })}
        actions={bookmarkActions({
          bookmarks: manager.pathBookmarks,
          t,
          onAddCurrent: () => { hapticImpact(); void manager.addCurrentPathBookmark() },
          onEdit: openBookmarkEditor,
          onOpen: (bookmark) => { hapticSelection(); void manager.navigate(bookmark.path) },
        })}
      />
      {bookmarksOpen && editingBookmark ? renderFileManagerPortal(
        <div className="fixed inset-0 z-[110] flex items-end justify-center bg-black/40 backdrop-blur-[2px] md:items-center" onClick={closeBookmarkEditor}>
          <ModalSurface
            aria-labelledby={bookmarkEditorTitleId}
            aria-describedby={bookmarkEditorDescriptionId}
            className="w-full max-w-xl animate-slide-up rounded-t-xl border-t border-[var(--anytty-app-line)] bg-[var(--background)] pb-[calc(env(safe-area-inset-bottom)+1rem)] md:rounded-xl md:border"
            initialFocusRef={bookmarkAliasInputRef}
            onClick={(event) => event.stopPropagation()}
            onRequestClose={closeBookmarkEditor}
          >
            <div className="mx-auto mt-3 h-1 w-12 rounded-full bg-[var(--anytty-app-line-strong)] md:hidden" />
            <div className="px-5 pb-2 pt-4">
              <h3 id={bookmarkEditorTitleId} className="text-[17px] font-bold text-zinc-900">{t('files.bookmarks.edit')}</h3>
              <p id={bookmarkEditorDescriptionId} className="mt-1 break-all text-[13px] font-medium text-zinc-500">{editingBookmark.path}</p>
            </div>
            <div className="px-5 py-3">
              <label className="flex flex-col gap-2 text-[13px] font-semibold text-zinc-600">
                {t('files.bookmarks.alias')}
                <Input
                  ref={bookmarkAliasInputRef}
                  aria-label={t('files.bookmarks.alias')}
                  className="h-12 bg-zinc-50 text-[16px] font-semibold text-zinc-900 focus-visible:ring-zinc-950"
                  value={bookmarkAlias}
                  onChange={(event) => setBookmarkAlias(event.currentTarget.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter') saveBookmarkAlias()
                  }}
                />
              </label>
              <div className="mt-4 grid grid-cols-2 gap-3">
                <Button variant="secondary"
                  className="h-11 font-semibold"
                  onClick={() => { hapticSelection(); closeBookmarkEditor() }}
                >
                  {t('common.cancel')}
                </Button>
                <Button variant="default"
                  className="h-11 font-semibold"
                  onClick={saveBookmarkAlias}
                >
                  {t('files.actions.save')}
                </Button>
              </div>
              <Button variant="ghost"
                type="button"
                className="mt-3 h-11 w-full rounded-md border border-red-200 bg-red-50 text-sm font-semibold text-red-600 active:bg-red-100"
                onClick={() => {
                  hapticImpact()
                  const id = editingBookmark.id
                  closeBookmarkEditor()
                  void manager.removePathBookmark(id)
                }}
              >
                {t('files.bookmarks.remove')}
              </Button>
            </div>
          </ModalSurface>
        </div>,
      ) : null}
    </div>
  )
}

function renderFileManagerPortal(content: ReactNode): ReactNode {
  return typeof document === 'undefined' ? content : createPortal(content, document.body)
}

function bookmarkActions({
  bookmarks,
  t,
  onAddCurrent,
  onEdit,
  onOpen,
}: {
  bookmarks: PathBookmark[]
  t: TFunction
  onAddCurrent: () => void
  onEdit: (bookmark: PathBookmark) => void
  onOpen: (bookmark: PathBookmark) => void
}): ActionSheetItem[] {
  return [
    {
      label: t('files.bookmarks.saveCurrent'),
      icon: <BookmarkPlus className="h-5 w-5" />,
      onClick: onAddCurrent,
      closeOnClick: false,
    },
    ...bookmarks.map((bookmark) => ({
      label: bookmark.label,
      ariaLabel: t('files.bookmarks.open', { name: bookmark.label }),
      subtitle: bookmark.path,
      icon: <Bookmark className="h-5 w-5" />,
      onClick: () => onOpen(bookmark),
      secondaryAction: {
        label: t('files.bookmarks.editNamed', { name: bookmark.label }),
        icon: <SquarePen className="h-5 w-5" />,
        onClick: () => onEdit(bookmark),
        closeOnClick: false,
      },
    })),
  ]
}

interface FileSortOption extends FileSortState {
  labelKey: string
  icon: ReactNode
}

const fileSortOptions: FileSortOption[] = [
  { field: 'name', direction: 'asc', labelKey: 'files.sort.nameAsc', icon: <ArrowUpAZ className="h-5 w-5" /> },
  { field: 'name', direction: 'desc', labelKey: 'files.sort.nameDesc', icon: <ArrowDownAZ className="h-5 w-5" /> },
  { field: 'modified', direction: 'desc', labelKey: 'files.sort.newest', icon: <Clock className="h-5 w-5" /> },
  { field: 'modified', direction: 'asc', labelKey: 'files.sort.oldest', icon: <Clock className="h-5 w-5" /> },
  { field: 'size', direction: 'desc', labelKey: 'files.sort.largest', icon: <File className="h-5 w-5" /> },
  { field: 'size', direction: 'asc', labelKey: 'files.sort.smallest', icon: <File className="h-5 w-5" /> },
  { field: 'type', direction: 'asc', labelKey: 'files.sort.typeAsc', icon: <FileType className="h-5 w-5" /> },
  { field: 'type', direction: 'desc', labelKey: 'files.sort.typeDesc', icon: <FileType className="h-5 w-5" /> },
]

function fileSortLabel(sort: FileSortState, t: TFunction): string {
  const option = fileSortOptions.find((candidate) => (
    candidate.field === sort.field && candidate.direction === sort.direction
  ))
  return t(option?.labelKey ?? 'files.sort.nameAsc')
}

function iconForFile(name: string) {
  const ext = extension(name)
  if (['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico', 'avif'].includes(ext)) return Image
  if (['mp4', 'webm', 'mov', 'm4v', 'ogv', 'ogg'].includes(ext)) return PlaySquare
  if (isModelPreviewFile(name, '')) return Box
  if (isMarkdownFile(name, '')) return FileText
  if (['js', 'ts', 'jsx', 'tsx', 'go', 'py', 'rs', 'java', 'c', 'cpp', 'h', 'hpp', 'css', 'html', 'json', 'yaml', 'yml', 'toml', 'xml', 'sh', 'sql'].includes(ext)) return Code2
  return File
}

function uniqueFileListKey(counts: Map<string, number>, baseKey: string): string {
  const count = counts.get(baseKey) ?? 0
  counts.set(baseKey, count + 1)
  return count === 0 ? baseKey : `${baseKey}:${count}`
}
