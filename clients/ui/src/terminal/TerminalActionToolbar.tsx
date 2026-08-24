import { type ReactNode, useEffect, useRef } from 'react'
import { Clipboard, ClipboardList, Copy, Cpu, Crown, Info, Link2, Link2Off, Lock, LockOpen, Minus, MousePointer2, MoveVertical, PanelBottomClose, PanelBottomOpen, PanelLeftOpen, PanelRightOpen, PanelTopOpen, Plus, Scaling, Search, Sparkles, X } from 'lucide-react'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import type { TerminalRenderer } from './Terminal'
import { terminalResizeControlOwnsResize, type TerminalResizeControl } from './terminalClient'
import type { TerminalKeyboardMode } from './terminalSettings'
import { useTranslation } from 'react-i18next'
import '../i18n'
import { Button } from '../ui/button'

export type TerminalToolbarMode = 'default' | 'selection'
export type TerminalSplitTarget = 'left' | 'right' | 'top' | 'bottom'

export interface TerminalActionToolbarProps {
  mode: TerminalToolbarMode
  hasSelection: boolean
  renderer?: TerminalRenderer | undefined
  fontSize?: number
  keyboardMode?: TerminalKeyboardMode | undefined
  resizeControl?: TerminalResizeControl | undefined
  sizeLocked?: boolean | undefined
  resizeOwnerPending?: boolean | undefined
  sizeLockPending?: boolean | undefined
  onModeChange: (mode: TerminalToolbarMode) => void
  onSelectAll: () => void
  onSelectVisible: () => void
  onCopy: () => void
  onPaste: () => void
  onOpenHistorySearch?: (() => void) | undefined
  onOpenClipboardHistory?: (() => void) | undefined
  onOpenSnippets: () => void
  onRendererChange?: ((renderer: TerminalRenderer) => void) | undefined
  onFontSizeChange?: ((size: number) => void) | undefined
  onKeyboardModeChange?: ((mode: TerminalKeyboardMode) => void) | undefined
  onAcquireResizeOwner?: (() => void) | undefined
  onReleaseResizeOwner?: (() => void) | undefined
  onToggleSizeLock?: (() => void) | undefined
  onOpenConnectionInfo?: (() => void) | undefined
  canSplitTerminal?: boolean | undefined
  onSplitTerminal?: ((target: TerminalSplitTarget) => void) | undefined
  splitTerminalOpen?: boolean | undefined
  syncSplitInput?: boolean | undefined
  onToggleSyncSplitInput?: (() => void) | undefined
  onCloseSplitTerminal?: (() => void) | undefined
  onClose?: () => void
  onEscape?: () => void
  escapeEnabled?: boolean
  remoteActionsDisabled?: boolean | undefined
  wideViewportVisible?: boolean | undefined
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
  keyboardMode = 'auto',
  resizeControl,
  sizeLocked = false,
  resizeOwnerPending = false,
  sizeLockPending = false,
  onModeChange,
  onSelectAll,
  onSelectVisible,
  onCopy,
  onPaste,
  onOpenHistorySearch,
  onOpenClipboardHistory,
  onOpenSnippets,
  onRendererChange,
  onFontSizeChange,
  onKeyboardModeChange,
  onAcquireResizeOwner,
  onReleaseResizeOwner,
  onToggleSizeLock,
  onOpenConnectionInfo,
  canSplitTerminal = false,
  onSplitTerminal,
  splitTerminalOpen = false,
  syncSplitInput = false,
  onToggleSyncSplitInput,
  onCloseSplitTerminal,
  onClose,
  onEscape,
  escapeEnabled = true,
  remoteActionsDisabled = false,
  wideViewportVisible = false,
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
      <div className={`absolute inset-x-2 bottom-2 z-40 rounded-lg border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] px-2 py-1.5 text-[var(--anytty-text)] shadow-[0_-4px_18px_rgba(0,0,0,0.22)] ${wideViewportVisible ? '' : 'md:hidden'}`} data-testid="anytty-terminal-action-toolbar">
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
  const ownsResize = terminalResizeControlOwnsResize(resizeControl)

  return (
    <div ref={panelRef} className={`absolute inset-x-2 top-2 z-40 max-h-[calc(100%_-_1rem)] overflow-y-auto overscroll-contain rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] px-3 pb-3 text-[var(--anytty-text)] shadow-[0_12px_32px_rgba(0,0,0,0.28)] backdrop-blur-xl animate-in slide-in-from-top-2 ${wideViewportVisible ? '' : 'md:hidden'}`} data-testid="anytty-terminal-action-toolbar">
      <div className="flex h-11 items-center justify-between border-b border-[var(--anytty-border-subtle)]">
        <span className="text-sm font-semibold text-[var(--anytty-text)]">{t('workspace.terminalTools')}</span>
        {onClose ? (
          <Button
            variant="ghost"
            type="button"
            aria-label={t('common.close')}
            title={t('common.close')}
            className="flex h-11 w-11 items-center justify-center text-[var(--anytty-muted)] active:text-[var(--anytty-text)]"
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => { hapticSelection(); onClose() }}
          >
            <X className="h-4 w-4" />
          </Button>
        ) : null}
      </div>

      <div className="flex flex-col gap-3 pt-3">
        <ToolbarSection label={t('settings.appearance')}>
          <div className="grid grid-cols-[5rem_minmax(0,1fr)] items-center gap-3">
            <span className="text-xs font-medium text-[var(--anytty-muted)]">{t('settings.fontSize')}</span>
            <div className="ml-auto flex h-11 items-center overflow-hidden rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)]">
              <Button variant="ghost"
                type="button"
                aria-label={t('settings.decreaseFont')}
                title={t('settings.decreaseFont')}
                onPointerDown={(event) => event.preventDefault()}
                onClick={() => { hapticSelection(); onFontSizeChange?.(Math.max(6, fontSize - 1)) }}
                className="flex h-11 w-11 items-center justify-center rounded-none text-[var(--anytty-muted)] active:bg-[var(--anytty-surface)] active:text-[var(--anytty-text)]"
              >
                <Minus className="h-3.5 w-3.5" />
              </Button>
              <span className="w-12 border-x border-[var(--anytty-border-subtle)] text-center font-mono text-xs font-semibold tabular-nums text-[var(--anytty-text)]">{fontSize}px</span>
              <Button variant="ghost"
                type="button"
                aria-label={t('settings.increaseFont')}
                title={t('settings.increaseFont')}
                onPointerDown={(event) => event.preventDefault()}
                onClick={() => { hapticSelection(); onFontSizeChange?.(Math.min(32, fontSize + 1)) }}
                className="flex h-11 w-11 items-center justify-center rounded-none text-[var(--anytty-muted)] active:bg-[var(--anytty-surface)] active:text-[var(--anytty-text)]"
              >
                <Plus className="h-3.5 w-3.5" />
              </Button>
            </div>

            <span className="text-xs font-medium text-[var(--anytty-muted)]">{t('settings.renderer')}</span>
            <Button variant="ghost"
              type="button"
              aria-label={`${t('settings.renderer')}: ${RENDERER_LABELS[renderer]}`}
              title={t('settings.renderer')}
              onPointerDown={(event) => event.preventDefault()}
              onClick={() => { hapticSelection(); onRendererChange?.(nextRenderer) }}
              className="ml-auto flex h-11 min-w-28 items-center justify-center gap-2 rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] px-3 text-xs font-semibold text-[var(--anytty-text)] active:bg-[var(--anytty-surface)]"
            >
              <Cpu className="h-4 w-4 text-[var(--anytty-muted)]" />
              {RENDERER_LABELS[renderer]}
            </Button>
          </div>
        </ToolbarSection>

        <ToolbarSection label={t('settings.keyboard')}>
          <div className="grid min-w-0 grid-cols-3 gap-1 rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] p-1" role="group" aria-label={t('settings.keyboard')}>
            {([
              { mode: 'auto' as const, icon: <Sparkles className="h-3.5 w-3.5" />, label: t('settings.auto') },
              { mode: 'resize' as const, icon: <Scaling className="h-3.5 w-3.5" />, label: t('settings.resize') },
              { mode: 'shift' as const, icon: <MoveVertical className="h-3.5 w-3.5" />, label: t('settings.shift') },
            ]).map((option) => {
              const selected = keyboardMode === option.mode
              return (
                <Button
                  key={option.mode}
                  variant="ghost"
                  type="button"
                  aria-label={option.label}
                  aria-pressed={selected}
                  title={option.label}
                  className={`flex h-11 min-w-0 items-center justify-center gap-1.5 rounded-sm px-2 text-xs font-semibold transition-colors ${selected
                    ? 'bg-[var(--anytty-accent)]/18 text-[var(--anytty-text)] shadow-sm'
                    : 'text-[var(--anytty-muted)] active:bg-[var(--anytty-surface)] active:text-[var(--anytty-text)]'}`}
                  onPointerDown={(event) => event.preventDefault()}
                  onClick={() => { hapticSelection(); onKeyboardModeChange?.(option.mode) }}
                >
                  {option.icon}
                  <span className="truncate">{option.label}</span>
                </Button>
              )
            })}
          </div>
        </ToolbarSection>

        <ToolbarSection label={t('terminal.tools.resizeControl')}>
          <div className="grid grid-cols-2 gap-2" role="group" aria-label={t('terminal.tools.resizeControl')}>
            <Button variant="ghost"
              type="button"
              aria-label={t(ownsResize ? 'terminal.tools.releaseOwner' : 'terminal.tools.acquireOwner')}
              aria-pressed={ownsResize}
              title={t(ownsResize ? 'terminal.tools.releaseOwner' : 'terminal.tools.acquireOwner')}
              disabled={remoteActionsDisabled || resizeOwnerPending}
              onPointerDown={(event) => { event.preventDefault(); event.stopPropagation() }}
              onClick={(event) => {
                event.preventDefault()
                event.stopPropagation()
                hapticSelection()
                ownsResize ? onReleaseResizeOwner?.() : onAcquireResizeOwner?.()
              }}
              className={`flex h-11 min-w-0 items-center justify-center gap-2 rounded-md border px-2 text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${ownsResize
                ? 'border-[var(--anytty-accent)] bg-[var(--anytty-accent)]/15 text-[var(--anytty-text)]'
                : 'border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:bg-[var(--anytty-surface)]'}`}
            >
              <Crown className="h-4 w-4 shrink-0" />
              <span className="min-w-0 truncate">{t(ownsResize ? 'terminal.tools.releaseOwnerButton' : 'terminal.tools.acquireOwnerButton')}</span>
            </Button>
            <Button variant="ghost"
              type="button"
              aria-label={t(sizeLocked ? 'terminal.tools.unlockSize' : 'terminal.tools.lockSize')}
              aria-pressed={sizeLocked}
              title={t(sizeLocked ? 'terminal.tools.unlockSize' : 'terminal.tools.lockSize')}
              disabled={remoteActionsDisabled || sizeLockPending || !ownsResize}
              onPointerDown={(event) => { event.preventDefault(); event.stopPropagation() }}
              onClick={(event) => {
                event.preventDefault()
                event.stopPropagation()
                hapticSelection()
                onToggleSizeLock?.()
              }}
              className={`flex h-11 min-w-0 items-center justify-center gap-2 rounded-md border px-2 text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${sizeLocked
                ? 'border-[var(--anytty-accent)] bg-[var(--anytty-accent)]/15 text-[var(--anytty-text)]'
                : 'border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:bg-[var(--anytty-surface)]'}`}
            >
              {sizeLocked ? <Lock className="h-4 w-4 shrink-0" /> : <LockOpen className="h-4 w-4 shrink-0" />}
              <span className="min-w-0 truncate">{t(sizeLocked ? 'terminal.tools.unlockSizeButton' : 'terminal.tools.lockSizeButton')}</span>
            </Button>
          </div>
        </ToolbarSection>

        <ToolbarSection label={t('terminal.tools.actions')}>
          <div className="grid grid-cols-2 gap-2">
            <ToolbarActionTile label={t('files.actions.select')} icon={<MousePointer2 />} onClick={() => onModeChange('selection')} />
            <ToolbarActionTile disabled={remoteActionsDisabled} label={t('terminal.paste.confirm')} icon={<Clipboard />} onClick={onPaste} />
            <ToolbarActionTile disabled={remoteActionsDisabled} label={t('workspace.clipboard')} icon={<ClipboardList />} onClick={() => onOpenClipboardHistory?.()} />
            <ToolbarActionTile disabled={remoteActionsDisabled} label={t('terminal.tools.snippets')} icon={<PanelTopOpen />} onClick={onOpenSnippets} />
            {onOpenHistorySearch ? (
              <ToolbarActionTile disabled={remoteActionsDisabled} label={t('terminal.tools.searchHistory')} icon={<Search />} onClick={onOpenHistorySearch} />
            ) : null}
            {onOpenConnectionInfo ? (
              <ToolbarActionTile label={t('workspace.connection.label')} icon={<Info />} onClick={onOpenConnectionInfo} />
            ) : null}
            {splitTerminalOpen ? (
              <>
                <ToolbarActionTile
                  icon={syncSplitInput ? <Link2 /> : <Link2Off />}
                  label={t('workspace.syncInput')}
                  pressed={syncSplitInput}
                  onClick={() => onToggleSyncSplitInput?.()}
                />
                <ToolbarActionTile icon={<PanelBottomClose />} label={t('workspace.closeSplit')} onClick={() => onCloseSplitTerminal?.()} />
              </>
            ) : null}
          </div>
        </ToolbarSection>

        <ToolbarSection label={t('workspace.splitTerminal')}>
          <div className="grid grid-cols-2 gap-2" role="group" aria-label={t('workspace.splitTerminal')}>
            <ToolbarActionTile disabled={!canSplitTerminal || remoteActionsDisabled} label={t('workspace.splitLeft')} icon={<PanelLeftOpen />} onClick={() => onSplitTerminal?.('left')} />
            <ToolbarActionTile disabled={!canSplitTerminal || remoteActionsDisabled} label={t('workspace.splitRight')} icon={<PanelRightOpen />} onClick={() => onSplitTerminal?.('right')} />
            <ToolbarActionTile disabled={!canSplitTerminal || remoteActionsDisabled} label={t('workspace.splitAbove')} icon={<PanelTopOpen />} onClick={() => onSplitTerminal?.('top')} />
            <ToolbarActionTile disabled={!canSplitTerminal || remoteActionsDisabled} label={t('workspace.splitBelow')} icon={<PanelBottomOpen />} onClick={() => onSplitTerminal?.('bottom')} />
          </div>
        </ToolbarSection>
      </div>
    </div>
  )
}

function ToolbarSection({ children, label }: { children: ReactNode; label: string }) {
  return (
    <section className="border-b border-[var(--anytty-border-subtle)] pb-3 last:border-b-0 last:pb-0">
      <h3 className="mb-2 text-[11px] font-semibold text-[var(--anytty-muted)]">{label}</h3>
      {children}
    </section>
  )
}

function ToolbarActionTile({
  disabled,
  icon,
  label,
  onClick,
  pressed,
}: {
  disabled?: boolean
  icon: ReactNode
  label: string
  onClick: () => void
  pressed?: boolean
}) {
  return (
    <Button
      variant="ghost"
      type="button"
      aria-label={label}
      aria-pressed={pressed}
      title={label}
      disabled={disabled}
      className={`flex h-11 min-w-0 items-center justify-start gap-2 rounded-md border px-3 text-left text-xs font-semibold transition-colors disabled:cursor-not-allowed disabled:opacity-40 [&_svg]:h-4 [&_svg]:w-4 [&_svg]:shrink-0 ${
        pressed === true
          ? 'border-[var(--anytty-accent)] bg-[var(--anytty-accent)]/15 text-[var(--anytty-text)]'
          : 'border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)] active:bg-[var(--anytty-surface)]'
      }`}
      onPointerDown={(event) => event.preventDefault()}
      onClick={() => { hapticSelection(); onClick() }}
    >
      {icon}
      <span className="min-w-0 truncate">{label}</span>
    </Button>
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
    <Button variant="ghost"
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
    </Button>
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
    <Button variant="ghost"
      type="button"
      aria-label={label}
      title={label}
      className="flex h-11 w-11 shrink-0 items-center justify-center rounded-md border border-red-500/20 bg-red-500/15 text-red-300 transition-colors hover:bg-red-50/80 active:bg-red-500/25"
      onPointerDown={(event) => event.preventDefault()}
      onClick={() => { hapticSelection(); onClick() }}
    >
      {children}
    </Button>
  )
}
