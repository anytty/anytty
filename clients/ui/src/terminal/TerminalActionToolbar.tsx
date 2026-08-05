import { type ReactNode, useEffect, useRef } from 'react'
import { Clipboard, ClipboardList, Copy, Cpu, MousePointer2, PanelTopOpen, X, Minus, Plus } from 'lucide-react'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import type { TerminalRenderer } from './Terminal'
import type { TerminalResizeControl } from './terminalClient'
import { useTranslation } from 'react-i18next'
import '../i18n'

export type TerminalToolbarMode = 'default' | 'selection'

export interface TerminalActionToolbarProps {
  mode: TerminalToolbarMode
  hasSelection: boolean
  renderer?: TerminalRenderer | undefined
  fontSize?: number
  resizeControl?: TerminalResizeControl | undefined
  onModeChange: (mode: TerminalToolbarMode) => void
  onSelectAll: () => void
  onSelectVisible: () => void
  onCopy: () => void
  onPaste: () => void
  onOpenClipboardHistory?: (() => void) | undefined
  onOpenSnippets: () => void
  onRendererChange?: ((renderer: TerminalRenderer) => void) | undefined
  onFontSizeChange?: ((size: number) => void) | undefined
  onAcquireResizeOwner?: (() => void) | undefined
  onReleaseResizeOwner?: (() => void) | undefined
  onClose?: () => void
  onEscape?: () => void
  escapeEnabled?: boolean
}

const RENDERER_LABELS: Record<TerminalRenderer, string> = {
  auto: 'Auto',
  webgl: 'WebGL',
  canvas: 'Canvas',
  dom: 'DOM',
}
const RENDERER_CYCLE: TerminalRenderer[] = ['auto', 'webgl', 'canvas', 'dom']

export function TerminalActionToolbar({
  mode,
  hasSelection,
  renderer = 'auto',
  fontSize = 14,
  resizeControl,
  onModeChange,
  onSelectAll,
  onSelectVisible,
  onCopy,
  onPaste,
  onOpenClipboardHistory,
  onOpenSnippets,
  onRendererChange,
  onFontSizeChange,
  onAcquireResizeOwner,
  onReleaseResizeOwner,
  onClose,
  onEscape,
  escapeEnabled = true,
}: TerminalActionToolbarProps) {
  const { t } = useTranslation()
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const escapeHandler = escapeEnabled ? (onEscape ?? onClose) : undefined
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape' || !escapeHandler) return
      event.preventDefault()
      event.stopPropagation()
      escapeHandler()
    }
    const handlePointerDown = (e: PointerEvent) => {
      const target = e.target as HTMLElement
      // Allow clicking buttons that open this menu (they usually stop propagation or we can check closest)
      if (target.closest('[data-testid="anytty-terminal-tools-button"]')) return
      if (panelRef.current && !panelRef.current.contains(target)) {
        onClose?.()
      }
    }
    if (escapeHandler) document.addEventListener('keydown', handleKeyDown, true)
    if (mode !== 'selection' && onClose) document.addEventListener('pointerdown', handlePointerDown, true)
    return () => {
      if (escapeHandler) document.removeEventListener('keydown', handleKeyDown, true)
      if (mode !== 'selection' && onClose) document.removeEventListener('pointerdown', handlePointerDown, true)
    }
  }, [escapeEnabled, mode, onClose, onEscape])

  if (mode === 'selection') {
    return (
      <div className="absolute inset-x-2 bottom-2 z-40 rounded-lg border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] px-2 py-1.5 text-[var(--anytty-text)] shadow-[0_-4px_18px_rgba(0,0,0,0.22)] md:hidden" data-testid="anytty-terminal-action-toolbar">
        <div className="flex items-center justify-between gap-1.5 overflow-x-auto">
          <div className="flex items-center gap-1.5">
            <ToolbarButton label={t('terminal.tools.selectAll')} onClick={onSelectAll} />
            <ToolbarButton label={t('terminal.tools.visible')} onClick={onSelectVisible} />
            <div className="mx-1 h-5 w-px shrink-0 bg-[var(--anytty-border-subtle)]" />
            <ToolbarButton
              label={t('files.actions.copy')}
              icon={<Copy className="h-3 w-3" />}
              onClick={onCopy}
              disabled={!hasSelection}
              primary={hasSelection}
            />
          </div>
          <ToolbarIconButton label={t('terminal.tools.cancelSelection')} onClick={() => onModeChange('default')}>
            <X className="h-3.5 w-3.5" />
          </ToolbarIconButton>
        </div>
      </div>
    )
  }

  const nextRenderer = RENDERER_CYCLE[(RENDERER_CYCLE.indexOf(renderer) + 1) % RENDERER_CYCLE.length]!
  const ownsResize = resizeControl?.canResize === true

  return (
    <div ref={panelRef} className="absolute inset-x-2 top-2 z-40 max-h-[calc(100%_-_1rem)] overflow-y-auto rounded-lg border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] px-3 py-3 text-[var(--anytty-text)] shadow-lg md:hidden animate-in slide-in-from-top-2" data-testid="anytty-terminal-action-toolbar">
      <div className="flex flex-col gap-3">
        {/* Settings Row: Font Size */}
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-[var(--anytty-muted)]">{t('settings.fontSize')}</span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              aria-label={t('settings.decreaseFont')}
              title={t('settings.decreaseFont')}
              onPointerDown={(event) => event.preventDefault()}
              onClick={() => { hapticSelection(); onFontSizeChange?.(Math.max(6, fontSize - 1)) }}
              className="flex h-11 w-11 items-center justify-center rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:opacity-75"
            >
              <Minus className="h-3.5 w-3.5" />
            </button>
            <span className="w-10 text-center font-mono text-xs font-semibold tabular-nums text-[var(--anytty-text)]">{fontSize}px</span>
            <button
              type="button"
              aria-label={t('settings.increaseFont')}
              title={t('settings.increaseFont')}
              onPointerDown={(event) => event.preventDefault()}
              onClick={() => { hapticSelection(); onFontSizeChange?.(Math.min(32, fontSize + 1)) }}
              className="flex h-11 w-11 items-center justify-center rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:opacity-75"
            >
              <Plus className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        {/* Settings Row: Renderer */}
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-[var(--anytty-muted)]">{t('settings.renderer')}</span>
          <button
            type="button"
            aria-label={`${t('settings.renderer')}: ${RENDERER_LABELS[renderer]}`}
            title={t('settings.renderer')}
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => { hapticSelection(); onRendererChange?.(nextRenderer) }}
            className="flex h-11 items-center justify-center gap-1.5 rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] px-3 text-xs font-semibold text-[var(--anytty-text)] active:opacity-75"
          >
            <Cpu className="h-3.5 w-3.5" />
            {RENDERER_LABELS[renderer]}
          </button>
        </div>

        <div className="my-1 h-px w-full bg-[var(--anytty-border-subtle)]" />

        <div className="flex items-center justify-between">
          <span className="text-xs font-medium text-[var(--anytty-muted)]">{t('terminal.tools.resizeControl')}</span>
          <button
            type="button"
            aria-label={t(ownsResize ? 'terminal.tools.releaseResize' : 'terminal.tools.acquireResize')}
            onPointerDown={(e) => {
              e.preventDefault()
              e.stopPropagation()
            }}
            onClick={(e) => {
              e.preventDefault()
              e.stopPropagation()
              hapticSelection()
              ownsResize ? onReleaseResizeOwner?.() : onAcquireResizeOwner?.()
            }}
            className="flex h-11 items-center justify-center gap-1.5 rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] px-3 text-xs font-semibold text-[var(--anytty-text)] active:opacity-75"
          >
            <span className="font-mono text-[11px] font-extrabold leading-none tracking-[-0.04em]">{ownsResize ? 'OW' : 'FL'}</span>
            {t(ownsResize ? 'terminal.tools.owner' : 'terminal.tools.follower')}
          </button>
        </div>

        {/* Tools Row */}
        <div className="grid grid-cols-4 gap-2">
          <ToolbarButton
            label={t('files.actions.select')}
            icon={<MousePointer2 className="h-3 w-3" />}
            onClick={() => onModeChange('selection')}
          />
          <ToolbarButton label={t('terminal.paste.confirm')} icon={<Clipboard className="h-3 w-3" />} onClick={onPaste} />
          <ToolbarButton label={t('workspace.clipboard')} icon={<ClipboardList className="h-3 w-3" />} onClick={() => onOpenClipboardHistory?.()} />
          <ToolbarButton label={t('terminal.tools.snippets')} icon={<PanelTopOpen className="h-3 w-3" />} onClick={onOpenSnippets} />
        </div>
      </div>
    </div>
  )
}

function ToolbarButton({
  disabled,
  icon,
  label,
  onClick,
  primary,
  title,
}: {
  disabled?: boolean
  icon?: ReactNode
  label: string
  onClick: () => void
  primary?: boolean
  title?: string
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={label}
      className={`flex min-h-11 min-w-0 items-center justify-center gap-1.5 rounded-md border border-[var(--anytty-border-subtle)] px-2 text-xs font-semibold transition-colors disabled:opacity-40 ${
        primary ? 'bg-[var(--anytty-accent)]/20 text-[var(--anytty-accent)]' : 'bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:opacity-75'
      }`}
      disabled={disabled}
      onPointerDown={(event) => event.preventDefault()}
      onClick={() => { hapticImpact(); onClick() }}
    >
      {icon}
      <span className="truncate">{label}</span>
    </button>
  )
}

function ToolbarIconButton({
  children,
  label,
  onClick,
}: {
  children: ReactNode
  label: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className="flex h-11 w-11 shrink-0 items-center justify-center rounded-md border border-red-500/20 bg-red-500/15 text-red-300 transition-colors hover:bg-red-50/80 active:bg-red-500/25"
      onPointerDown={(event) => event.preventDefault()}
      onClick={() => { hapticSelection(); onClick() }}
    >
      {children}
    </button>
  )
}
