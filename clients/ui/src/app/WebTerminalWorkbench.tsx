import { useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'
import { Folder, PanelBottom, PanelLeftClose, PanelLeftOpen, Plus, Rows2, Search, Settings2, SquareTerminal, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Terminal } from '../core/model'
import { Button } from '../ui/button'
import { ANYTTY_TERMINAL_DRAG_TYPE, terminalIdFromDrag } from './WebTerminalDropOverlay'

export { ANYTTY_TERMINAL_DRAG_TYPE, WebTerminalDropOverlay, type WebPaneDropTarget } from './WebTerminalDropOverlay'

export type WebSplitDirection = 'columns' | 'rows'

export interface WebTerminalWorkbenchProps {
  terminals: readonly Terminal[]
  tabTerminalIds: readonly string[]
  activeTabTerminalId: string | null
  splitTabTerminalIds: readonly string[]
  draggedTerminalId: string | null
  sidebarOpen: boolean
  canCreateTerminal: boolean
  canSplitTerminal: boolean
  disabled: boolean
  onActivateTab: (terminalId: string) => void
  onCloseTab: (terminalId: string) => void
  onCreateTerminal: () => void
  onOpenFiles: () => void
  onOpenTerminalPicker: () => void
  onOpenSettings: () => void
  onOpenSplit: () => void
  onReorderTabs: (terminalId: string, targetTerminalId: string, placement: 'before' | 'after') => void
  onToggleSidebar: () => void
  onTerminalDragChange: (terminalId: string | null) => void
}

export function WebTerminalWorkbench({
  terminals,
  tabTerminalIds,
  activeTabTerminalId,
  splitTabTerminalIds,
  draggedTerminalId,
  sidebarOpen,
  canCreateTerminal,
  canSplitTerminal,
  disabled,
  onActivateTab,
  onCloseTab,
  onCreateTerminal,
  onOpenFiles,
  onOpenTerminalPicker,
  onOpenSettings,
  onOpenSplit,
  onReorderTabs,
  onToggleSidebar,
  onTerminalDragChange,
}: WebTerminalWorkbenchProps) {
  const { t } = useTranslation()
  const terminalById = new Map(terminals.map((terminal) => [terminal.terminalId, terminal]))
  const tabs = tabTerminalIds.flatMap((terminalId) => {
    const terminal = terminalById.get(terminalId)
    return terminal ? [terminal] : []
  })
  const [dropMarker, setDropMarker] = useState<{ terminalId: string; placement: 'before' | 'after' } | null>(null)

  const moveTabFocus = (event: KeyboardEvent<HTMLButtonElement>, terminalId: string) => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key) || tabs.length < 2) return
    event.preventDefault()
    const currentIndex = tabs.findIndex((terminal) => terminal.terminalId === terminalId)
    const nextIndex = event.key === 'Home'
      ? 0
      : event.key === 'End'
        ? tabs.length - 1
        : event.key === 'ArrowLeft'
          ? (currentIndex - 1 + tabs.length) % tabs.length
          : (currentIndex + 1) % tabs.length
    const nextId = tabs[nextIndex]?.terminalId
    if (!nextId) return
    onActivateTab(nextId)
    window.setTimeout(() => document.getElementById(webTabId(nextId))?.focus(), 0)
  }

  return (
    <header className="hidden h-11 min-w-0 items-stretch border-b border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] md:flex" data-testid="anytty-web-workbench-bar">
      <div className="flex shrink-0 items-center border-r border-[var(--anytty-border-subtle)] px-1">
        <WorkbenchAction
          icon={sidebarOpen ? PanelLeftClose : PanelLeftOpen}
          label={t(sidebarOpen ? 'workspace.hideTerminalSidebar' : 'workspace.showTerminalSidebar')}
          onClick={onToggleSidebar}
        />
      </div>
      <div className="min-w-0 flex-1 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <div aria-label={t('terminal.list')} className="flex h-full min-w-max items-stretch" role="tablist">
          {tabs.map((terminal) => {
            const terminalId = terminal.terminalId
            const active = activeTabTerminalId === terminalId
            const inSplit = splitTabTerminalIds.includes(terminalId)
            const title = terminal.title || terminal.command || t('terminal.defaultTitle')
            const marker = dropMarker?.terminalId === terminalId ? dropMarker.placement : null
            return (
              <div
                className={`group relative flex h-full max-w-64 items-stretch border-r border-[var(--anytty-border-subtle)] ${active ? 'bg-[var(--anytty-terminal-bg)]' : 'bg-[var(--anytty-surface)] hover:bg-[var(--anytty-surface-raised)]'} ${marker === 'before' ? 'before:absolute before:inset-y-1 before:left-0 before:w-0.5 before:bg-[var(--anytty-accent)]' : ''} ${marker === 'after' ? 'after:absolute after:inset-y-1 after:right-0 after:w-0.5 after:bg-[var(--anytty-accent)]' : ''}`}
                draggable={!disabled}
                key={terminalId}
                onDragEnd={() => {
                  setDropMarker(null)
                  onTerminalDragChange(null)
                }}
                onDragOver={(event) => {
                  const draggedId = draggedTerminalId || terminalIdFromDrag(event)
                  if (!draggedId || draggedId === terminalId) return
                  event.preventDefault()
                  const bounds = event.currentTarget.getBoundingClientRect()
                  setDropMarker({ terminalId, placement: event.clientX < bounds.left + bounds.width / 2 ? 'before' : 'after' })
                }}
                onDragStart={(event) => {
                  event.dataTransfer.effectAllowed = 'move'
                  event.dataTransfer.setData(ANYTTY_TERMINAL_DRAG_TYPE, terminalId)
                  event.dataTransfer.setData('text/plain', terminalId)
                  onTerminalDragChange(terminalId)
                }}
                onDrop={(event) => {
                  const draggedId = terminalIdFromDrag(event)
                  if (!draggedId || draggedId === terminalId) return
                  event.preventDefault()
                  const bounds = event.currentTarget.getBoundingClientRect()
                  const placement = event.clientX < bounds.left + bounds.width / 2 ? 'before' : 'after'
                  onReorderTabs(draggedId, terminalId, placement)
                  setDropMarker(null)
                  onTerminalDragChange(null)
                }}
              >
                <button
                  aria-controls="anytty-web-terminal-viewport"
                  aria-selected={active}
                  className="flex min-w-0 flex-1 items-center gap-2 px-3 text-left text-xs font-medium text-[var(--anytty-muted)] outline-none transition-colors focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--anytty-accent)] aria-selected:text-[var(--anytty-text)]"
                  disabled={disabled}
                  id={webTabId(terminalId)}
                  onClick={() => onActivateTab(terminalId)}
                  onKeyDown={(event) => moveTabFocus(event, terminalId)}
                  role="tab"
                  tabIndex={active ? 0 : -1}
                  title={title}
                  type="button"
                >
                  <span className={`size-1.5 shrink-0 rounded-full ${terminal.state === 'running' ? 'bg-emerald-400' : 'border border-[var(--anytty-muted)]'}`} />
                  <span className="min-w-0 flex-1 truncate">{title}</span>
                  {inSplit ? <PanelBottom aria-hidden="true" className="size-3.5 shrink-0" /> : null}
                </button>
                <button
                  aria-label={t('common.closeNamed', { name: title })}
                  className="mr-1.5 flex size-7 shrink-0 self-center items-center justify-center rounded text-[var(--anytty-muted)] opacity-70 outline-none hover:bg-[var(--anytty-surface-raised)] hover:text-[var(--anytty-text)] focus-visible:ring-2 focus-visible:ring-[var(--anytty-accent)] group-hover:opacity-100"
                  onClick={() => onCloseTab(terminalId)}
                  title={t('common.closeNamed', { name: title })}
                  type="button"
                >
                  <X className="size-3.5" />
                </button>
              </div>
            )
          })}
        </div>
      </div>

      <div className="flex shrink-0 items-center border-l border-[var(--anytty-border-subtle)] px-1">
        <WorkbenchAction icon={Search} label={t('terminal.picker.title')} disabled={disabled} onClick={onOpenTerminalPicker} />
        <WorkbenchAction icon={Plus} label={t('workspace.createTerminal')} disabled={!canCreateTerminal || disabled} onClick={onCreateTerminal} />
        <WorkbenchAction icon={Folder} label={t('workspace.openFiles')} disabled={disabled} onClick={onOpenFiles} />
        <WorkbenchAction icon={Rows2} label={t('workspace.splitBelow')} disabled={disabled || !activeTabTerminalId || !canSplitTerminal} onClick={onOpenSplit} />
        <WorkbenchAction icon={Settings2} label={t('common.settings')} onClick={onOpenSettings} />
      </div>
    </header>
  )
}

export function WebTerminalPaneHeader({ terminal, active, onClose }: {
  terminal: Terminal | undefined
  active: boolean
  onClose?: (() => void) | undefined
}) {
  const { t } = useTranslation()
  const title = terminal?.title || terminal?.command || t('terminal.defaultTitle')
  return (
    <div className={`absolute inset-x-0 top-0 z-10 flex h-7 items-center gap-2 border-b border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] px-2 text-[11px] ${active ? 'text-[var(--anytty-text)]' : 'text-[var(--anytty-muted)]'}`} data-testid="anytty-web-pane-header">
      <SquareTerminal className="size-3.5 shrink-0" />
      <span className="min-w-0 flex-1 truncate font-medium">{title}</span>
      {terminal?.cwd ? <span className="max-w-[45%] truncate font-mono text-[10px] text-[var(--anytty-faint)]">{terminal.cwd}</span> : null}
      {onClose ? (
        <button
          aria-label={t('workspace.closeSplit')}
          className="hidden size-6 shrink-0 items-center justify-center rounded outline-none hover:bg-[var(--anytty-surface-raised)] focus-visible:ring-2 focus-visible:ring-[var(--anytty-accent)] md:flex"
          onClick={onClose}
          title={t('workspace.closeSplit')}
          type="button"
        >
          <X className="size-3.5" />
        </button>
      ) : null}
    </div>
  )
}

function WorkbenchAction({ icon: Icon, label, disabled = false, onClick }: {
  icon: typeof Plus
  label: string
  disabled?: boolean
  onClick: () => void
}) {
  return (
    <Button
      aria-label={label}
      className="size-9 rounded border-0 bg-transparent p-0 text-[var(--anytty-muted)] shadow-none hover:bg-[var(--anytty-surface-raised)] hover:text-[var(--anytty-text)]"
      disabled={disabled}
      onClick={onClick}
      size="icon-sm"
      title={label}
      variant="ghost"
    >
      <Icon className="size-4" />
    </Button>
  )
}

export function WebSplitDivider({ direction, ratio, onRatioChange, onResizeEnd }: {
  direction: WebSplitDirection
  ratio: number
  onRatioChange: (ratio: number) => void
  onResizeEnd: () => void
}) {
  const { t } = useTranslation()
  const dragRef = useRef<{ start: number; ratio: number; extent: number } | null>(null)
  const coordinate = (event: PointerEvent<HTMLDivElement>) => direction === 'columns' ? event.clientX : event.clientY
  const extent = (element: HTMLElement) => {
    const bounds = element.parentElement?.getBoundingClientRect()
    return direction === 'columns' ? bounds?.width ?? 0 : bounds?.height ?? 0
  }
  return (
    <div
      aria-label={t('workspace.resizeSplit')}
      aria-orientation={direction === 'columns' ? 'vertical' : 'horizontal'}
      aria-valuemax={80}
      aria-valuemin={20}
      aria-valuenow={Math.round(ratio)}
      className={`group z-20 hidden shrink-0 touch-none items-center justify-center bg-[var(--anytty-border-subtle)] outline-none hover:bg-[var(--anytty-border)] focus-visible:bg-[var(--anytty-accent)] md:flex ${direction === 'columns' ? 'w-1.5 cursor-col-resize' : 'h-1.5 cursor-row-resize'}`}
      data-testid="anytty-web-split-divider"
      onKeyDown={(event) => {
        const decrease = event.key === 'ArrowLeft' || event.key === 'ArrowUp'
        const increase = event.key === 'ArrowRight' || event.key === 'ArrowDown'
        if (!decrease && !increase && event.key !== 'Home' && event.key !== 'End') return
        event.preventDefault()
        const next = event.key === 'Home' ? 20 : event.key === 'End' ? 80 : ratio + (decrease ? -5 : 5)
        onRatioChange(clampSplitRatio(next))
        onResizeEnd()
      }}
      onPointerCancel={(event) => {
        dragRef.current = null
        event.currentTarget.releasePointerCapture?.(event.pointerId)
        onResizeEnd()
      }}
      onPointerDown={(event) => {
        dragRef.current = { start: coordinate(event), ratio, extent: extent(event.currentTarget) }
        event.currentTarget.setPointerCapture?.(event.pointerId)
      }}
      onPointerMove={(event) => {
        const drag = dragRef.current
        if (!drag || drag.extent <= 0) return
        onRatioChange(clampSplitRatio(drag.ratio + ((coordinate(event) - drag.start) / drag.extent) * 100))
      }}
      onPointerUp={(event) => {
        if (!dragRef.current) return
        dragRef.current = null
        event.currentTarget.releasePointerCapture?.(event.pointerId)
        onResizeEnd()
      }}
      role="separator"
      tabIndex={0}
    >
      <span className={`rounded-full bg-[var(--anytty-muted)] opacity-0 transition-opacity group-hover:opacity-100 ${direction === 'columns' ? 'h-9 w-0.5' : 'h-0.5 w-9'}`} />
    </div>
  )
}

function webTabId(terminalId: string): string {
  return `anytty-web-tab-${encodeURIComponent(terminalId)}`
}

function clampSplitRatio(value: number): number {
  return Math.max(20, Math.min(80, value))
}
