import type { Terminal } from '../core/model'
import type { TFunction } from 'i18next'
import {
  Activity,
  AppWindow,
  Boxes,
  Braces,
  ChevronRight,
  CircleDot,
  Clock3,
  Container,
  Database,
  FileCode2,
  FileText,
  Fish,
  Gauge,
  GitBranch,
  HardDrive,
  ListTree,
  Logs,
  MoreVertical,
  Network,
  Package,
  PanelTop,
  Pin,
  Radar,
  ScrollText,
  Server,
  Shell,
  SquareTerminal,
  Table2,
  Terminal as TerminalIcon,
  type LucideIcon,
} from 'lucide-react'
import { hapticImpact } from '../platform/haptics'
import { useTranslation } from 'react-i18next'
import { useRef, useState } from 'react'
import '../i18n'
import { Button } from '../ui/button'
import { Spinner } from '../ui/spinner'
import claudeIcon from '../assets/terminal-programs/claude.png'
import clineIcon from '../assets/terminal-programs/cline.png'
import codexIcon from '../assets/terminal-programs/codex.png'
import cursorIcon from '../assets/terminal-programs/cursor.svg'
import geminiIcon from '../assets/terminal-programs/gemini.svg'
import githubCopilotIcon from '../assets/terminal-programs/github-copilot.svg'
import openCodeIcon from '../assets/terminal-programs/opencode.svg'
import qwenIcon from '../assets/terminal-programs/qwen.png'
import warpIcon from '../assets/terminal-programs/warp.png'
import zedIcon from '../assets/terminal-programs/zed.svg'

export interface OpenTerminalIntent {
  machineId: string
  terminalId: string
}

export interface TerminalListProps {
  machineId: string
  terminals: Terminal[]
  onOpenTerminal: (intent: OpenTerminalIntent) => void
  onManageTerminal?: ((intent: OpenTerminalIntent) => void) | undefined
  activeTerminalId?: string | undefined
  className?: string
  loading?: boolean
  loadingLabel?: string | undefined
  interactive?: boolean | undefined
  pinnedTerminalIds?: readonly string[] | undefined
  onReorderPinnedTerminal?: ((terminalId: string, targetTerminalId: string, placement: 'before' | 'after') => void) | undefined
}

export function TerminalList({
  machineId,
  terminals,
  onOpenTerminal,
  onManageTerminal,
  activeTerminalId,
  className,
  loading,
  loadingLabel,
  interactive = true,
  pinnedTerminalIds = [],
  onReorderPinnedTerminal,
}: TerminalListProps) {
  const { t } = useTranslation()
  const terminalKeyCounts = new Map<string, number>()
  const longPressTimerRef = useRef<number | null>(null)
  const suppressOpenRef = useRef<string | null>(null)
  const dragRef = useRef<{ terminalId: string; targetTerminalId?: string; placement?: 'before' | 'after' } | null>(null)
  const [drag, setDrag] = useState<typeof dragRef.current>(null)
  const clearLongPress = () => {
    if (longPressTimerRef.current !== null) window.clearTimeout(longPressTimerRef.current)
    longPressTimerRef.current = null
  }
  const finishPointerGesture = (element: HTMLElement, pointerId: number) => {
    clearLongPress()
    const current = dragRef.current
    if (current?.targetTerminalId && current.placement) {
      onReorderPinnedTerminal?.(current.terminalId, current.targetTerminalId, current.placement)
      hapticImpact()
    }
    if (current) {
      suppressOpenRef.current = current.terminalId
      window.setTimeout(() => { suppressOpenRef.current = null }, 0)
    }
    dragRef.current = null
    setDrag(null)
    if (element.hasPointerCapture?.(pointerId)) element.releasePointerCapture(pointerId)
  }

  return (
    <div
      className={className}
      data-machine-id={machineId}
      data-testid="anytty-terminal-list"
    >
      {terminals.length === 0 ? (
        loading ? (
          <div
            className="flex min-h-24 items-center justify-center gap-2 rounded-lg border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface)] px-4 text-sm font-medium text-[var(--anytty-app-muted)]"
            role="status"
            aria-live="polite"
            aria-busy="true"
          >
            <Spinner className="h-4 w-4" aria-hidden="true" />
            <span>{loadingLabel ?? t('common.loading')}</span>
          </div>
        ) : (
          <div className="flex h-32 flex-col items-center justify-center gap-3 rounded-lg border border-dashed border-[var(--anytty-app-line-strong)] bg-[var(--anytty-app-surface-soft)] text-sm text-[var(--anytty-app-muted)] animate-in fade-in duration-300">
            <TerminalIcon className="h-8 w-8 text-zinc-300" />
            <p>{t('terminal.noActive')}</p>
          </div>
        )
      ) : (
        <ul
          aria-label={t('terminal.list')}
          aria-disabled={!interactive || undefined}
          className="flex flex-col gap-2"
        >
          {terminals.map((terminal) => {
            const isActive = activeTerminalId === terminal.terminalId
            const itemKey = uniqueTerminalListKey(terminalKeyCounts, machineId, terminal)
            const program = terminalProgramPresentation(terminal.foregroundProcess)
            const outputActivity = terminalOutputActivityLabel(terminal.lastOutputAt, Date.now(), t)
            const pinned = pinnedTerminalIds.includes(terminal.terminalId)
            const dragging = drag?.terminalId === terminal.terminalId
            const dragTarget = drag?.targetTerminalId === terminal.terminalId
            return (
              <li
                key={itemKey}
                data-terminal-id={terminal.terminalId}
                aria-grabbed={dragging || undefined}
                className={`overflow-hidden rounded-lg border bg-[var(--anytty-app-surface)] transition-[transform,box-shadow,border-color] duration-150 ${dragging ? 'relative z-20 rotate-1 scale-[1.02] border-[var(--anytty-app-accent)] shadow-xl' : dragTarget ? 'border-[var(--anytty-app-accent)] shadow-md' : 'border-[var(--anytty-app-line)]'}`}
              >
                <div
                  className={`group relative flex w-full items-center py-2 pl-2 pr-1 text-left transition-colors duration-200 has-[:focus-visible]:ring-2 has-[:focus-visible]:ring-inset has-[:focus-visible]:ring-[var(--anytty-app-accent)] ${pinned && onReorderPinnedTerminal ? 'touch-none' : ''} ${!interactive ? 'cursor-not-allowed' : ''} ${
                    isActive
                      ? 'bg-[var(--primary)] text-[var(--primary-foreground)]'
                      : 'bg-[var(--anytty-app-surface)] text-zinc-700 hover:bg-[var(--anytty-app-surface-soft)]'
                  }`}
                  onContextMenu={(event) => {
                    if (!interactive || !onManageTerminal) return
                    event.preventDefault()
                    onManageTerminal({ machineId, terminalId: terminal.terminalId })
                  }}
                  onPointerDown={(event) => {
                    if (!interactive || !onManageTerminal || event.pointerType === 'mouse') return
                    if ((event.target as HTMLElement).closest('[data-terminal-manage]')) return
                    const target = event.currentTarget
                    clearLongPress()
                    longPressTimerRef.current = window.setTimeout(() => {
                      longPressTimerRef.current = null
                      suppressOpenRef.current = terminal.terminalId
                      hapticImpact()
                      if (pinned && onReorderPinnedTerminal) {
                        target.setPointerCapture?.(event.pointerId)
                        const next = { terminalId: terminal.terminalId }
                        dragRef.current = next
                        setDrag(next)
                      } else {
                        onManageTerminal({ machineId, terminalId: terminal.terminalId })
                        window.setTimeout(() => {
                          if (suppressOpenRef.current === terminal.terminalId) suppressOpenRef.current = null
                        }, 700)
                      }
                    }, 450)
                  }}
                  onPointerMove={(event) => {
                    const current = dragRef.current
                    if (!current || current.terminalId !== terminal.terminalId) return
                    event.preventDefault()
                    const item = document.elementFromPoint(event.clientX, event.clientY)?.closest<HTMLElement>('[data-terminal-id]')
                    const targetTerminalId = item?.dataset.terminalId
                    if (!targetTerminalId || targetTerminalId === current.terminalId || !pinnedTerminalIds.includes(targetTerminalId)) return
                    const rect = item.getBoundingClientRect()
                    const next = { ...current, targetTerminalId, placement: event.clientY >= rect.top + rect.height / 2 ? 'after' as const : 'before' as const }
                    dragRef.current = next
                    setDrag(next)
                  }}
                  onPointerUp={(event) => finishPointerGesture(event.currentTarget, event.pointerId)}
                  onPointerCancel={(event) => finishPointerGesture(event.currentTarget, event.pointerId)}
                  onPointerLeave={() => {
                    if (!dragRef.current) clearLongPress()
                  }}
                >
                  <Button variant="ghost"
                    className="flex h-auto min-h-[72px] min-w-0 flex-1 items-center justify-start gap-2.5 whitespace-normal rounded-md px-1 py-1.5 text-left active:scale-[0.98] focus:outline-none"
                    type="button"
                    disabled={!interactive}
                    aria-label={t('terminal.open', { name: terminal.title || terminal.command || t('terminal.defaultTitle') })}
                    aria-current={isActive ? 'true' : 'false'}
                    onClick={() => {
                      if (!interactive) return
                      if (suppressOpenRef.current === terminal.terminalId) return
                      hapticImpact()
                      onOpenTerminal({ machineId, terminalId: terminal.terminalId })
                    }}
                  >
                    <ProgramGlyph active={isActive} presentation={program} />

                    <div className="flex min-w-0 flex-1 flex-col justify-center gap-1">
                      <div className="flex min-w-0 items-center justify-between gap-2">
                        <span className={`truncate text-[15px] font-semibold leading-5 ${isActive ? 'text-[var(--primary-foreground)]' : 'text-zinc-900'}`}>
                          {terminal.title || terminal.command || t('terminal.defaultTitle')}
                        </span>
                        {terminal.environment ? (
                          <span className={`shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-bold tracking-wider uppercase leading-none ${isActive ? 'bg-black/15 text-[var(--primary-foreground)]' : 'bg-zinc-100 text-zinc-500'}`}>
                            {terminal.environment}
                          </span>
                        ) : null}
                      </div>
                      {terminal.command || terminal.cwd ? (
                        <span className={`truncate text-[11px] font-medium leading-4 ${isActive ? 'text-[var(--primary-foreground)] opacity-75' : 'text-zinc-500'}`}>
                          {terminal.cwd ? terminal.cwd : terminal.command}
                        </span>
                      ) : null}

                      <div className={`flex flex-wrap items-center gap-x-2.5 gap-y-1 text-[10px] font-medium leading-4 ${isActive ? 'text-[var(--primary-foreground)] opacity-80' : 'text-zinc-500'}`}>
                        {program.label ? (
                          <span>{program.label}</span>
                        ) : null}
                        {outputActivity ? (
                          <span className="inline-flex items-center gap-1 tabular-nums">
                            <Clock3 className="size-3" />
                            {outputActivity}
                          </span>
                        ) : null}
                        <span className="inline-flex items-center gap-1">
                          <CircleDot className={`size-2.5 ${terminal.state === 'running' ? 'fill-emerald-500 text-emerald-500' : 'text-zinc-400'}`} />
                          {t(`terminal.state.${terminal.state === 'running' ? 'running' : terminal.state === 'exited' ? 'exited' : 'unknown'}`)}
                        </span>
                      </div>
                    </div>
                  </Button>

                  {onManageTerminal ? (
                    <Button variant="ghost"
                      type="button"
                      disabled={!interactive}
                      data-terminal-manage="true"
                      className={`flex h-11 w-11 shrink-0 items-center justify-center rounded-md focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] ${isActive ? 'text-[var(--primary-foreground)] hover:bg-black/15' : 'text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700'}`}
                      aria-label={t('terminal.manage', { name: terminal.title || terminal.command || t('terminal.defaultTitle') })}
                      onClick={() => {
                        if (!interactive) return
                        hapticImpact()
                        onManageTerminal({ machineId, terminalId: terminal.terminalId })
                      }}
                    >
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                  ) : null}
                  {pinned ? <Pin aria-label={t('terminal.order.pinned')} className={`mr-1 size-3.5 shrink-0 fill-current ${isActive ? 'text-[var(--primary-foreground)] opacity-75' : 'text-[var(--anytty-app-accent)]'}`} /> : null}
                  <ChevronRight
                    aria-hidden="true"
                    className={`mr-0.5 size-4 shrink-0 transition-transform group-active:translate-x-0.5 ${isActive ? 'text-[var(--primary-foreground)] opacity-70' : 'text-zinc-300 group-hover:text-zinc-400'}`}
                  />
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}

export interface TerminalProgramPresentation {
  icon: LucideIcon
  label?: string
  brandAsset?: string
  monogram?: string
}

const terminalProgramPresets: ReadonlyArray<{
  aliases: readonly string[]
  brandAsset: string
  label: string
}> = [
  { aliases: ['codex'], brandAsset: codexIcon, label: 'Codex' },
  { aliases: ['claude', 'claude-code'], brandAsset: claudeIcon, label: 'Claude Code' },
  { aliases: ['gemini', 'gemini-cli'], brandAsset: geminiIcon, label: 'Gemini CLI' },
  { aliases: ['copilot', 'copilot-cli', 'github-copilot', 'gh-copilot'], brandAsset: githubCopilotIcon, label: 'GitHub Copilot' },
  { aliases: ['cursor', 'cursor-agent'], brandAsset: cursorIcon, label: 'Cursor' },
  { aliases: ['cline'], brandAsset: clineIcon, label: 'Cline' },
  { aliases: ['opencode'], brandAsset: openCodeIcon, label: 'OpenCode' },
  { aliases: ['qwen', 'qwen-code'], brandAsset: qwenIcon, label: 'Qwen Code' },
  { aliases: ['zed'], brandAsset: zedIcon, label: 'Zed' },
  { aliases: ['warp', 'warp-cli'], brandAsset: warpIcon, label: 'Warp' },
]

const terminalCommandPresets: ReadonlyArray<{ aliases: readonly string[]; label: string; icon: LucideIcon }> = [
  { aliases: ['vim'], label: 'Vim', icon: FileCode2 },
  { aliases: ['nvim', 'neovim'], label: 'Neovim', icon: FileCode2 },
  { aliases: ['emacs'], label: 'Emacs', icon: FileText },
  { aliases: ['nano'], label: 'Nano', icon: FileText },
  { aliases: ['top'], label: 'top', icon: Gauge },
  { aliases: ['htop'], label: 'htop', icon: Activity },
  { aliases: ['btop', 'bpytop'], label: 'btop', icon: Activity },
  { aliases: ['glances'], label: 'Glances', icon: Radar },
  { aliases: ['tmux'], label: 'tmux', icon: PanelTop },
  { aliases: ['screen'], label: 'GNU Screen', icon: AppWindow },
  { aliases: ['ssh', 'mosh'], label: 'SSH', icon: Network },
  { aliases: ['lazygit'], label: 'lazygit', icon: GitBranch },
  { aliases: ['git'], label: 'Git', icon: GitBranch },
  { aliases: ['docker', 'dockerd'], label: 'Docker', icon: Container },
  { aliases: ['podman'], label: 'Podman', icon: Container },
  { aliases: ['kubectl', 'k9s'], label: 'Kubernetes', icon: Boxes },
  { aliases: ['node', 'nodejs'], label: 'Node.js', icon: Braces },
  { aliases: ['python', 'python3', 'ipython'], label: 'Python', icon: Braces },
  { aliases: ['ruby', 'irb'], label: 'Ruby', icon: Braces },
  { aliases: ['java', 'jshell'], label: 'Java', icon: Braces },
  { aliases: ['go'], label: 'Go', icon: Braces },
  { aliases: ['cargo', 'rustc'], label: 'Rust', icon: Package },
  { aliases: ['mysql', 'mariadb'], label: 'MySQL', icon: Database },
  { aliases: ['psql', 'postgres'], label: 'PostgreSQL', icon: Database },
  { aliases: ['redis-cli', 'redis-server'], label: 'Redis', icon: Database },
  { aliases: ['mongosh', 'mongo', 'mongod'], label: 'MongoDB', icon: Database },
  { aliases: ['sqlite3'], label: 'SQLite', icon: Table2 },
  { aliases: ['tail'], label: 'tail', icon: Logs },
  { aliases: ['less'], label: 'less', icon: FileText },
  { aliases: ['watch'], label: 'watch', icon: Clock3 },
  { aliases: ['journalctl'], label: 'journalctl', icon: ScrollText },
  { aliases: ['tcpdump'], label: 'tcpdump', icon: Network },
  { aliases: ['nethogs'], label: 'nethogs', icon: Activity },
  { aliases: ['iftop'], label: 'iftop', icon: Gauge },
  { aliases: ['nginx'], label: 'Nginx', icon: Server },
  { aliases: ['caddy'], label: 'Caddy', icon: Server },
  { aliases: ['supervisord', 'supervisorctl'], label: 'Supervisor', icon: ListTree },
  { aliases: ['pm2'], label: 'PM2', icon: ListTree },
  { aliases: ['ranger'], label: 'Ranger', icon: HardDrive },
  { aliases: ['yazi'], label: 'Yazi', icon: HardDrive },
  { aliases: ['mc'], label: 'Midnight Commander', icon: HardDrive },
  { aliases: ['zsh'], label: 'zsh', icon: SquareTerminal },
  { aliases: ['bash'], label: 'Bash', icon: Shell },
  { aliases: ['fish'], label: 'fish', icon: Fish },
  { aliases: ['pwsh', 'powershell'], label: 'PowerShell', icon: SquareTerminal },
]

export function terminalProgramPresentation(value?: string): TerminalProgramPresentation {
  const label = value?.trim()
  const process = terminalExecutableName(label)
  if (!process || !label) return { icon: TerminalIcon }
  const preset = terminalProgramPresets.find(({ aliases }) => aliases.includes(process))
  if (preset) return { icon: TerminalIcon, label: preset.label, brandAsset: preset.brandAsset }
  const commandPreset = terminalCommandPresets.find(({ aliases }) => aliases.includes(process))
  if (commandPreset) return { icon: commandPreset.icon, label: commandPreset.label }
  return { icon: TerminalIcon, label: process }
}

function terminalExecutableName(value?: string): string | undefined {
  const firstToken = value?.trim().toLowerCase().split(/\s+/, 1)[0]?.replaceAll('\\', '/')
  const executable = firstToken?.split('/').pop()?.replace(/\.exe$/, '')
  return executable || undefined
}

function ProgramGlyph({
  active,
  presentation,
}: {
  active: boolean
  presentation: TerminalProgramPresentation
}) {
  if (presentation.brandAsset) {
    return <img alt="" className="size-10 shrink-0 rounded-md object-contain" src={presentation.brandAsset} />
  }
  if (presentation.monogram) {
    return (
      <span aria-hidden="true" className={`grid size-10 shrink-0 place-items-center rounded-md border font-mono text-[12px] font-bold tracking-normal ${active ? 'border-white/25 bg-white/10 text-[var(--primary-foreground)]' : 'border-zinc-200 bg-zinc-50 text-zinc-700'}`}>
        {presentation.monogram}
      </span>
    )
  }
  const Icon = presentation.icon
  return <Icon className={`size-8 shrink-0 ${active ? 'text-[var(--primary-foreground)]' : 'text-zinc-500'}`} />
}

export function terminalOutputActivityLabel(
  lastOutputAt: string | undefined,
  now: number,
  t: TFunction,
): string | undefined {
  if (!lastOutputAt) return undefined
  const timestamp = Date.parse(lastOutputAt)
  if (!Number.isFinite(timestamp)) return undefined
  const quietSeconds = Math.max(0, Math.floor((now - timestamp) / 1_000))
  if (quietSeconds < 5) return t('terminal.outputActivity.now')
  if (quietSeconds < 60) return t('terminal.outputActivity.seconds', { count: quietSeconds })
  return t('terminal.outputActivity.minutes', { count: Math.floor(quietSeconds / 60) })
}

function uniqueTerminalListKey(counts: Map<string, number>, fallbackMachineId: string, terminal: Terminal): string {
  const baseKey = `${terminal.machineId || fallbackMachineId}:${terminal.terminalId}`
  const count = counts.get(baseKey) ?? 0
  counts.set(baseKey, count + 1)
  return count === 0 ? baseKey : `${baseKey}:${count}`
}
