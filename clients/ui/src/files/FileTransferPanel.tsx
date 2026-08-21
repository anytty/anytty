import { useEffect, useId, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { ArrowDownToLine, ArrowUpFromLine, CheckSquare, Download, ExternalLink, Pause, Play, RotateCw, Square, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import '../i18n'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import { NATIVE_BACK_PRIORITY } from '../platform/nativeBack'
import { useNativeBackHandler } from '../platform/useNativeBackHandler'
import { ModalSurface } from '../ui/ModalSurface'
import { Button } from '../ui/button'
import { Spinner } from '../ui/spinner'
import type { TransferInfo } from './fileApi'

interface Props {
  transfers: TransferInfo[]
  hasActiveTransfers: boolean
  resolveMachineLabel?: ((machineId: string | undefined) => string | null | undefined) | undefined
  onCancel: (id: string) => void
  onDismiss: (id: string) => void
  onPause?: ((id: string) => void | Promise<void>) | undefined
  onResume?: ((id: string) => void | Promise<void>) | undefined
  onResumeAll?: (() => void | Promise<void>) | undefined
  onOpenFile?: ((id: string) => void | Promise<void>) | undefined
  variant?: 'inline' | 'floating' | 'icon' | undefined
  open?: boolean | undefined
  onOpenChange?: ((open: boolean) => void) | undefined
  remoteActionsDisabled?: boolean | undefined
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function formatSpeed(bytesPerSecond: number): string {
  const speed = Math.max(0, bytesPerSecond)
  if (speed < 1024) return `${Math.round(speed)} B/s`
  if (speed < 1024 * 1024) return `${(speed / 1024).toFixed(1)} KB/s`
  return `${(speed / (1024 * 1024)).toFixed(1)} MB/s`
}

function transferSpeed(transfer: TransferInfo, now: number): string {
  if (transfer.status === 'pending' || transfer.status === 'paused' || transfer.status === 'missing') return '0 B/s'
  const updatedAt = transfer.updatedAt ?? transfer.startedAt
  if (now - updatedAt > 4000) return '0 B/s'
  const measured = transfer.bytesPerSecond
  if (typeof measured === 'number' && Number.isFinite(measured) && measured >= 0) {
    return formatSpeed(measured)
  }
  const elapsed = (now - transfer.startedAt) / 1000
  if (!Number.isFinite(elapsed) || elapsed <= 0) return '0 B/s'
  return formatSpeed(Math.max(0, transfer.transferredSize / elapsed))
}

export function FileTransferPanel({
  transfers,
  hasActiveTransfers,
  resolveMachineLabel,
  onCancel,
  onDismiss,
  onPause,
  onResume,
  onResumeAll,
  onOpenFile,
  variant = 'floating',
  open: controlledOpen,
  onOpenChange,
  remoteActionsDisabled = false,
}: Props) {
  const { t: translate } = useTranslation()
  const active = useMemo(() => sortTransfersByNewestFirst(transfers), [transfers])
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false)
  const [now, setNow] = useState(() => Date.now())
  const open = controlledOpen ?? uncontrolledOpen
  const setOpen = (next: boolean) => {
    if (controlledOpen === undefined) setUncontrolledOpen(next)
    onOpenChange?.(next)
  }

  useEffect(() => {
    if (active.length === 0) return undefined
    const timer = window.setInterval(() => setNow(Date.now()), 1000)
    return () => window.clearInterval(timer)
  }, [active.length])

  if (active.length === 0 && !open) return null

  const inProgressCount = active.filter((t) => t.status === 'pending' || t.status === 'transferring').length
  const pausedCount = active.filter((t) => t.status === 'paused' || t.status === 'missing').length
  const failedCount = active.filter((t) => t.status === 'failed' || t.status === 'missing').length
  const completedCount = active.filter((t) => t.status === 'completed').length
  const summaryParts: string[] = []
  if (inProgressCount > 0) summaryParts.push(translate('files.transfer.summary.active', { count: inProgressCount }))
  if (pausedCount > 0) summaryParts.push(translate('files.transfer.summary.paused', { count: pausedCount }))
  if (failedCount > 0) summaryParts.push(translate('files.transfer.summary.failed', { count: failedCount }))
  if (completedCount > 0) summaryParts.push(translate('files.transfer.summary.done', { count: completedCount }))
  const summary = summaryParts.join(' · ') || (active.length > 0 ? translate('files.transfer.summary.total', { count: active.length }) : translate('files.transfer.summary.none'))
  const measurable = active.filter((t) => t.totalSize > 0 && (t.status === 'pending' || t.status === 'transferring' || t.status === 'paused' || t.status === 'missing'))
  const totalSize = measurable.reduce((sum, t) => sum + t.totalSize, 0)
  const totalTransferred = measurable.reduce((sum, t) => sum + Math.min(t.transferredSize, t.totalSize), 0)
  const totalProgress = totalSize > 0 ? (totalTransferred / totalSize) * 100 : 0
  const rootClass = variant === 'inline'
    ? 'border-t border-zinc-200 bg-[var(--background)]'
    : variant === 'icon'
      ? 'relative'
      : 'pointer-events-auto fixed bottom-[calc(env(safe-area-inset-bottom)+1rem)] right-3 z-40'

  return (
    <div className={rootClass}>
      <Button variant="ghost"
        onClick={() => { hapticSelection(); setOpen(true) }}
        aria-label={translate('files.transfer.openSummary', { summary })}
        className={variant === 'inline'
          ? 'flex w-full items-center justify-between px-4 py-2 text-left'
          : variant === 'icon'
            ? 'flex h-8 w-8 items-center justify-center rounded-md text-zinc-400 transition-colors hover:bg-white/10 active:bg-white/10'
            : 'flex min-h-12 items-center gap-3 rounded-lg border border-[var(--anytty-app-line-strong)] bg-[var(--anytty-app-surface)] px-3 py-2 text-left shadow-[0_8px_24px_rgba(15,23,42,0.12)] active:bg-[var(--anytty-app-surface-soft)]'}
      >
        <span className={variant === 'icon' ? 'relative flex h-8 w-8 items-center justify-center' : 'relative flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-blue-50 text-blue-600'}>
          <Download className={variant === 'icon' ? 'h-4 w-4' : 'h-4 w-4'} />
          {hasActiveTransfers ? <span className="absolute right-0 top-0 h-2.5 w-2.5 rounded-full border-2 border-white bg-emerald-500" /> : null}
        </span>
        {variant === 'icon' ? null : (
          <span className={variant === 'inline' ? 'text-xs font-medium text-zinc-500' : 'min-w-0 pr-1'}>
            <span className="block text-[12px] font-semibold leading-4 text-zinc-900">{translate('files.transfer.title')}</span>
            <span className="block text-[11px] font-medium leading-4 text-zinc-500">{summary}</span>
          </span>
        )}
      </Button>

      {variant === 'floating' && hasActiveTransfers && totalProgress > 0 ? (
        <div className="absolute inset-x-4 bottom-1 h-0.5 overflow-hidden rounded-full bg-zinc-100">
          <div className="h-full rounded-full bg-blue-500 transition-all duration-500" style={{ width: `${Math.min(100, totalProgress)}%` }} />
        </div>
      ) : null}

      {open ? (
        <TransferCenterDialog
          transfers={active}
          now={now}
          summary={summary}
          resolveMachineLabel={resolveMachineLabel}
          onCancel={onCancel}
          onDismiss={onDismiss}
          onPause={onPause}
          onResume={onResume}
          onResumeAll={onResumeAll}
          remoteActionsDisabled={remoteActionsDisabled}
          onOpenFile={onOpenFile}
          onClose={() => setOpen(false)}
        />
      ) : null}
    </div>
  )
}

function TransferCenterDialog({
  transfers,
  now,
  summary,
  resolveMachineLabel,
  onCancel,
  onDismiss,
  onPause,
  onResume,
  onResumeAll,
  onOpenFile,
  remoteActionsDisabled,
  onClose,
}: {
  transfers: TransferInfo[]
  now: number
  summary: string
  resolveMachineLabel?: ((machineId: string | undefined) => string | null | undefined) | undefined
  onCancel: (id: string) => void
  onDismiss: (id: string) => void
  onPause?: ((id: string) => void | Promise<void>) | undefined
  onResume?: ((id: string) => void | Promise<void>) | undefined
  onResumeAll?: (() => void | Promise<void>) | undefined
  onOpenFile?: ((id: string) => void | Promise<void>) | undefined
  remoteActionsDisabled: boolean
  onClose: () => void
}) {
  const { t: translate } = useTranslation()
  const titleId = useId()
  const summaryId = useId()
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set())
  const [selectionMode, setSelectionMode] = useState(false)
  const [openingId, setOpeningId] = useState<string | null>(null)
  const [openErrorId, setOpenErrorId] = useState<string | null>(null)

  useNativeBackHandler(() => {
    if (selectionMode) {
      setSelectionMode(false)
      setSelectedIds(new Set())
      return
    }
    onClose()
  }, NATIVE_BACK_PRIORITY.TRANSFER)

  useEffect(() => {
    setSelectedIds((current) => {
      if (current.size === 0) return current
      const availableIds = new Set(transfers.map((transfer) => transfer.id))
      let changed = false
      const next = new Set<string>()
      for (const id of current) {
        if (availableIds.has(id)) {
          next.add(id)
        } else {
          changed = true
        }
      }
      return changed ? next : current
    })
  }, [transfers])

  useEffect(() => {
    if (transfers.length > 0 || !selectionMode) return
    setSelectionMode(false)
  }, [selectionMode, transfers.length])

  const resumableCount = transfers.filter((t) => isResumable(t.status)).length
  const selectedTransfers = transfers.filter((transfer) => selectedIds.has(transfer.id))
  const selectedPausableCount = selectedTransfers.filter((transfer) => isPausable(transfer.status)).length
  const selectedStartableCount = selectedTransfers.filter((transfer) => isResumable(transfer.status)).length
  const completedCount = transfers.filter((transfer) => transfer.status === 'completed').length
  const failedCount = transfers.filter((transfer) => isFailedClearable(transfer.status)).length
  const allSelected = transfers.length > 0 && selectedIds.size === transfers.length
  const enterSelectionMode = () => {
    hapticImpact()
    setSelectionMode(true)
  }
  const exitSelectionMode = () => {
    hapticSelection()
    setSelectionMode(false)
    setSelectedIds(new Set())
  }
  const toggleSelectAll = () => {
    setSelectedIds((current) => {
      if (transfers.length > 0 && current.size === transfers.length) return new Set()
      return new Set(transfers.map((transfer) => transfer.id))
    })
  }
  const toggleSelected = (transferId: string) => {
    if (!selectionMode) return
    setSelectedIds((current) => {
      const next = new Set(current)
      if (next.has(transferId)) next.delete(transferId)
      else next.add(transferId)
      return next
    })
  }
  const pauseSelected = () => {
    if (!onPause) return
    const pausable = selectedTransfers.filter((transfer) => isPausable(transfer.status))
    if (pausable.length === 0) return
    hapticImpact()
    for (const transfer of pausable) onPause(transfer.id)
  }
  const startSelected = () => {
    if (!onResume || remoteActionsDisabled) return
    const startable = selectedTransfers.filter((transfer) => isResumable(transfer.status))
    if (startable.length === 0) return
    hapticImpact()
    for (const transfer of startable) onResume(transfer.id)
  }
  const clearCompleted = () => {
    const completed = transfers.filter((transfer) => transfer.status === 'completed')
    if (completed.length === 0) return
    hapticImpact()
    for (const transfer of completed) onDismiss(transfer.id)
  }
  const openCompletedDownload = async (transfer: TransferInfo) => {
    if (!onOpenFile || openingId) return
    setOpeningId(transfer.id)
    setOpenErrorId(null)
    hapticSelection()
    try {
      await onOpenFile(transfer.id)
    } catch {
      setOpenErrorId(transfer.id)
    } finally {
      setOpeningId(null)
    }
  }
  const clearFailed = () => {
    const failed = transfers.filter((transfer) => isFailedClearable(transfer.status))
    if (failed.length === 0) return
    hapticImpact()
    for (const transfer of failed) onDismiss(transfer.id)
  }

  const dialog = (
    <div
      className="bg-[var(--anytty-app-bg)] text-[var(--anytty-app-text)] fixed inset-0 z-[100] flex"
      data-anytty-transfer-center-root=""
    >
      <ModalSurface
        aria-labelledby={titleId}
        aria-describedby={summaryId}
        className="flex h-full min-h-0 w-full flex-col bg-[var(--background)]"
        onRequestClose={onClose}
      >
        {selectionMode ? (
          <header className="border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex shrink-0 items-center justify-between gap-3 border-b px-4 pb-3 pt-[calc(env(safe-area-inset-top)+0.75rem)]">
            <Button variant="ghost"
              type="button"
              className="text-[15px] font-medium text-zinc-500 hover:text-zinc-700 active:text-zinc-800"
              onClick={exitSelectionMode}
            >
              {translate('common.cancel')}
            </Button>
            <div className="min-w-0 text-center">
              <h2 id={titleId} className="truncate text-[17px] font-semibold text-zinc-950">{translate('files.transfer.selected', { count: selectedIds.size })}</h2>
              <p id={summaryId} className="mt-0.5 text-[12px] font-medium text-zinc-500">{summary}</p>
            </div>
            <Button variant="ghost"
              type="button"
              aria-label={allSelected ? translate('files.transfer.clearSelection') : translate('files.transfer.selectAll')}
              className="text-[15px] font-medium text-blue-600 hover:text-blue-700 active:text-blue-800"
              onClick={() => { hapticSelection(); toggleSelectAll() }}
            >
              {allSelected ? translate('files.transfer.deselectAll') : translate('files.transfer.selectAll')}
            </Button>
          </header>
        ) : (
          <header className="border-[var(--anytty-app-line)] bg-[var(--anytty-app-bg)] flex shrink-0 items-center justify-between gap-3 border-b px-4 pb-3 pt-[calc(env(safe-area-inset-top)+0.75rem)]">
            <div className="min-w-0">
              <h2 id={titleId} className="text-[18px] font-semibold text-zinc-950">{translate('files.transfer.center')}</h2>
              <p id={summaryId} className="mt-0.5 text-[12px] font-medium text-zinc-500">{summary}</p>
            </div>
            <div className="flex shrink-0 items-center gap-2">
              {onResumeAll && resumableCount > 0 ? (
                <Button variant="default"
                  aria-label={translate('files.transfer.resumeAll')}
                  className="gap-1.5 px-3 text-[12px] font-semibold"
                  disabled={remoteActionsDisabled}
                  onClick={() => { hapticImpact(); onResumeAll() }}
                >
                  <RotateCw className="h-4 w-4" />
                  {translate('files.transfer.resumeAll')}
                </Button>
              ) : null}
              <Button variant="ghost" size="icon"
                aria-label={translate('files.transfer.closeCenter')}
                onClick={() => { hapticSelection(); onClose() }}
              >
                <X className="h-5 w-5" />
              </Button>
            </div>
          </header>
        )}
        <div className="min-h-0 flex-1 overflow-y-auto pb-[calc(env(safe-area-inset-bottom)+5rem)]" role="list">
          {transfers.length === 0 ? (
            <div className="flex h-48 items-center justify-center px-6 text-center text-[14px] font-medium text-zinc-500">
              {translate('files.transfer.empty')}
            </div>
          ) : null}
          {transfers.map((t) => {
            const progress = t.totalSize > 0 ? (t.transferredSize / t.totalSize) * 100 : 0
            const isFailed = t.status === 'failed'
            const isMissing = t.status === 'missing'
            const isActive = t.status === 'pending' || t.status === 'transferring'
            const isPaused = t.status === 'paused' || isMissing
            const isCompleted = t.status === 'completed'
            const canOpen = isCompleted && t.direction === 'download' && Boolean(t.savedUri) && Boolean(onOpenFile)
            const machineLabel = resolveMachineLabel?.(t.machineId) ?? t.machineId ?? translate('files.transfer.unknownDevice')
            const transferTarget = describeTransferTarget(t)
            const selected = selectedIds.has(t.id)
            return (
              <div key={t.id} role="listitem" className="border-b border-zinc-100 px-4 py-3 last:border-b-0">
                <div className="flex items-start gap-3">
                  {selectionMode ? (
                    <Button variant="ghost"
                      type="button"
                      aria-label={translate(selected ? 'files.transfer.deselectItem' : 'files.transfer.selectItem', { name: t.name })}
                      className="mt-1 flex h-11 w-11 shrink-0 items-center justify-center rounded-md text-zinc-400 hover:bg-zinc-50 active:bg-zinc-100 hover:text-zinc-700 active:text-zinc-800"
                      onClick={() => { hapticSelection(); toggleSelected(t.id) }}
                    >
                      {selected ? <CheckSquare className="h-4 w-4 text-zinc-900" /> : <Square className="h-4 w-4" />}
                    </Button>
                  ) : null}
                  <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-md border border-[var(--anytty-app-line)] ${t.direction === 'download' ? 'bg-blue-50 text-blue-600' : 'bg-violet-50 text-violet-600'}`}>
                    {t.direction === 'download' ? <ArrowDownToLine className="h-4 w-4" /> : <ArrowUpFromLine className="h-4 w-4" />}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        {canOpen ? (
                          <Button variant="ghost"
                            className="block max-w-full truncate text-left text-[13px] font-semibold text-[var(--anytty-app-accent)] focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)]"
                            type="button"
                            onClick={() => { void openCompletedDownload(t) }}
                          >
                            {t.name}
                          </Button>
                        ) : <p className="truncate text-[13px] font-semibold text-zinc-900">{t.name}</p>}
                        <p className="mt-1 text-[11px] font-semibold uppercase tracking-[0.14em] text-zinc-400">
                          {translate(t.direction === 'download' ? 'files.transfer.fromDevice' : 'files.transfer.toDevice', { device: machineLabel })}
                        </p>
                        {transferTarget ? (
                          <p className="mt-1 truncate text-[11px] font-medium text-zinc-500">{transferTarget}</p>
                        ) : null}
                        <p className="mt-0.5 text-[11px] font-medium text-zinc-500">
                          {isActive && t.totalSize > 0 ? `${formatSize(t.transferredSize)} / ${formatSize(t.totalSize)} · ${transferSpeed(t, now)}` : null}
                          {t.status === 'paused' ? `${formatSize(t.transferredSize)} / ${formatSize(t.totalSize)} · ${translate('files.transfer.status.paused')}` : null}
                          {isMissing ? `${formatSize(t.transferredSize)} / ${formatSize(t.totalSize)} · ${translate('files.transfer.status.missing')}` : null}
                          {isCompleted ? `${formatSize(t.totalSize)}${t.savedPath ? ` · ${t.savedPath}` : ` · ${translate('files.transfer.savedDownloads')}`}` : null}
                          {isFailed ? t.error || translate('files.transfer.status.failed') : null}
                          {!isActive && !isPaused && !isCompleted && !isFailed && !isMissing ? t.status : null}
                        </p>
                        {openErrorId === t.id ? (
                          <p className="mt-1 text-[11px] font-medium text-red-600" role="alert">{translate('files.transfer.openFailed')}</p>
                        ) : null}
                      </div>
                      <div className="flex shrink-0 items-center gap-1">
                        {!selectionMode && canOpen ? (
                          <Button variant="ghost"
                            type="button"
                            aria-label={translate('files.transfer.openItem', { name: t.name })}
                            className="flex h-11 w-11 items-center justify-center rounded-md text-[var(--anytty-app-accent)] hover:bg-zinc-50 active:bg-zinc-100 disabled:opacity-50"
                            disabled={openingId !== null}
                            onClick={() => { void openCompletedDownload(t) }}
                          >
                            {openingId === t.id ? <Spinner aria-hidden="true" /> : <ExternalLink className="h-4 w-4" />}
                          </Button>
                        ) : null}
                        {!selectionMode && isActive && onPause ? (
                          <Button variant="ghost"
                            type="button"
                            aria-label={translate('files.transfer.pauseItem', { name: t.name })}
                            className="flex h-11 w-11 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-50 active:bg-zinc-100 hover:text-zinc-700 active:text-zinc-800"
                            onClick={() => { hapticImpact(); onPause(t.id) }}
                          >
                            <Pause className="h-3.5 w-3.5" />
                          </Button>
                        ) : null}
                        {!selectionMode && isPaused && onResume ? (
                          <Button variant="ghost"
                            type="button"
                            aria-label={translate(isMissing ? 'files.transfer.retryItem' : 'files.transfer.resumeItem', { name: t.name })}
                            className="flex h-11 w-11 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-50 active:bg-zinc-100 hover:text-zinc-700 active:text-zinc-800"
                            disabled={remoteActionsDisabled}
                            onClick={() => { hapticImpact(); onResume(t.id) }}
                          >
                            <Play className="h-3.5 w-3.5" />
                          </Button>
                        ) : null}
                        {!selectionMode ? (
                          <Button variant="ghost"
                            aria-label={translate(isActive || t.status === 'paused' ? 'files.transfer.cancelItem' : 'files.transfer.clearItem', { name: t.name })}
                            className="flex h-11 w-11 shrink-0 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-50 active:bg-zinc-100 hover:text-zinc-700 active:text-zinc-800"
                            onClick={() => {
                              if (isActive || t.status === 'paused') hapticImpact()
                              else hapticSelection()
                              if (isActive || t.status === 'paused') onCancel(t.id)
                              else onDismiss(t.id)
                            }}
                          >
                            {isActive || t.status === 'paused' ? <X className="h-3.5 w-3.5" /> : <Trash2 className="h-3.5 w-3.5" />}
                          </Button>
                        ) : null}
                      </div>
                    </div>
                    <div className={`mt-2 h-1.5 overflow-hidden rounded-full ${isFailed || isMissing ? 'bg-red-100' : 'bg-zinc-100'}`}>
                      <div
                        className={`h-full rounded-full transition-all duration-500 ${isFailed || isMissing ? 'bg-red-500' : isCompleted ? 'bg-emerald-500' : 'bg-blue-500'}`}
                        style={{ width: `${Math.min(100, isCompleted ? 100 : progress)}%` }}
                      />
                    </div>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
        {transfers.length > 0 ? (
          <div className="absolute bottom-0 left-0 right-0 z-40 border-t border-zinc-200 bg-[var(--background)] pb-[env(safe-area-inset-bottom)] backdrop-blur-xl">
            {selectionMode ? (
              <div className="flex h-[52px] items-stretch justify-around px-2">
                <Button variant="ghost"
                  type="button"
                  aria-label={translate('files.transfer.pauseSelected')}
                  disabled={!onPause || selectedPausableCount === 0}
                  className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 px-3 text-zinc-600 disabled:opacity-40 hover:bg-zinc-50 hover:text-blue-600 active:bg-zinc-100"
                  onClick={pauseSelected}
                >
                  <Pause className="h-5 w-5" />
                  <span className="text-[11px] font-medium">{translate('files.transfer.pause')}</span>
                </Button>
                <Button variant="ghost"
                  type="button"
                  aria-label={translate('files.transfer.startSelected')}
                  disabled={remoteActionsDisabled || !onResume || selectedStartableCount === 0}
                  className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 px-3 text-zinc-600 disabled:opacity-40 hover:bg-zinc-50 hover:text-blue-600 active:bg-zinc-100"
                  onClick={startSelected}
                >
                  <Play className="h-5 w-5" />
                  <span className="text-[11px] font-medium">{translate('files.transfer.start')}</span>
                </Button>
              </div>
            ) : (
              <div className="flex h-[52px] items-center justify-around px-2">
                <Button variant="ghost"
                  type="button"
                  aria-label={translate('files.transfer.selectTransfers')}
                  className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 px-3 text-zinc-600 hover:bg-zinc-50 hover:text-blue-600 active:bg-zinc-100"
                  onClick={enterSelectionMode}
                >
                  <CheckSquare className="h-5 w-5" />
                  <span className="text-[11px] font-medium">{translate('files.transfer.select')}</span>
                </Button>
                <Button variant="ghost"
                  type="button"
                  aria-label={translate('files.transfer.clearCompleted')}
                  disabled={completedCount === 0}
                  className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 px-3 text-zinc-600 disabled:opacity-40 hover:bg-zinc-50 hover:text-blue-600 active:bg-zinc-100"
                  onClick={clearCompleted}
                >
                  <Trash2 className="h-5 w-5" />
                  <span className="text-[11px] font-medium">{translate('files.transfer.done')}</span>
                </Button>
                <Button variant="ghost"
                  type="button"
                  aria-label={translate('files.transfer.clearFailed')}
                  disabled={failedCount === 0}
                  className="flex min-h-11 min-w-11 flex-col items-center justify-center gap-1 px-3 text-zinc-600 disabled:opacity-40 hover:bg-zinc-50 hover:text-blue-600 active:bg-zinc-100"
                  onClick={clearFailed}
                >
                  <Trash2 className="h-5 w-5" />
                  <span className="text-[11px] font-medium">{translate('files.transfer.failed')}</span>
                </Button>
              </div>
            )}
          </div>
        ) : null}
      </ModalSurface>
    </div>
  )
  return typeof document === 'undefined' ? dialog : createPortal(dialog, document.body)
}

function sortTransfersByNewestFirst(transfers: TransferInfo[]): TransferInfo[] {
  return [...transfers].sort((left, right) => {
    const startedDelta = right.startedAt - left.startedAt
    if (startedDelta !== 0) return startedDelta
    return right.id.localeCompare(left.id)
  })
}

function isPausable(status: TransferInfo['status']): boolean {
  return status === 'pending' || status === 'transferring'
}

function isResumable(status: TransferInfo['status']): boolean {
  return status === 'paused' || status === 'failed' || status === 'missing' || status === 'pending'
}

function isFailedClearable(status: TransferInfo['status']): boolean {
  return status === 'failed' || status === 'missing'
}

function describeTransferTarget(transfer: TransferInfo): string | null {
  if (transfer.direction === 'download') {
    if (transfer.status === 'completed' && transfer.savedPath) {
      return `${transfer.filePath ?? transfer.name} -> ${transfer.savedPath}`
    }
    if (transfer.filePath) return transfer.filePath
    if (transfer.savedPath) return transfer.savedPath
    return null
  }
  if (transfer.targetDir) {
    return `${transfer.targetDir.replace(/\/+$/, '') || '/'} / ${transfer.name}`
  }
  if (transfer.localUri) return transfer.localUri
  return null
}
