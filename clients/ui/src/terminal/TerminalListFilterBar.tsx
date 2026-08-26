import { ChevronDown, History, List, Play, Tags, type LucideIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { hapticSelection } from '../platform/haptics'
import '../i18n'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../ui/dropdown-menu'
import type { TerminalStatusFilter, TerminalTagOption } from './terminalListFilters'

interface TerminalListFilterBarProps {
  status: TerminalStatusFilter
  statusCounts: Record<TerminalStatusFilter, number>
  tagOptions: readonly TerminalTagOption[]
  selectedTagIds: readonly string[]
  filteredCount: number
  mobileTagSheet: boolean
  tagSheetOpen: boolean
  onStatusChange: (status: TerminalStatusFilter) => void
  onTagToggle: (tagId: string) => void
  onClearTags: () => void
  onOpenTagSheet: () => void
}

const statusOptions: Array<{ status: TerminalStatusFilter; icon: LucideIcon }> = [
  { status: 'running', icon: Play },
  { status: 'exited', icon: History },
  { status: 'all', icon: List },
]

export function TerminalListFilterBar({
  status,
  statusCounts,
  tagOptions,
  selectedTagIds,
  filteredCount,
  mobileTagSheet,
  tagSheetOpen,
  onStatusChange,
  onTagToggle,
  onClearTags,
  onOpenTagSheet,
}: TerminalListFilterBarProps) {
  const { t } = useTranslation()
  const selectedTagCount = selectedTagIds.length
  const tagButton = (
    <button
      aria-expanded={mobileTagSheet ? tagSheetOpen : undefined}
      aria-controls={mobileTagSheet ? 'anytty-terminal-tag-filter-sheet' : undefined}
      aria-label={t('terminal.filters.tagLabel')}
      className={`relative flex min-w-0 items-center justify-center gap-1 rounded px-1 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] disabled:cursor-not-allowed disabled:opacity-45 ${selectedTagCount > 0 ? 'bg-[var(--anytty-app-accent)] text-white shadow-sm' : 'text-zinc-500 hover:bg-[var(--anytty-app-surface)] hover:text-zinc-800'}`}
      data-testid="anytty-terminal-tag-filter"
      disabled={tagOptions.length === 0}
      onClick={mobileTagSheet ? () => { hapticSelection(); onOpenTagSheet() } : undefined}
      type="button"
    >
      <Tags aria-hidden="true" className="hidden h-3.5 w-3.5 shrink-0 2xl:block" />
      <span className="truncate">{t('terminal.filters.tags')}</span>
      {selectedTagCount > 0 ? (
        <span className="min-w-4 rounded-sm bg-black/15 px-1 text-[10px] leading-4 tabular-nums">{selectedTagCount}</span>
      ) : (
        <ChevronDown aria-hidden="true" className="hidden h-3 w-3 shrink-0 2xl:block" />
      )}
    </button>
  )

  return (
    <div
      aria-label={t('terminal.filters.statusLabel')}
      className="mb-2 grid h-11 grid-cols-4 gap-1 rounded-md border border-[var(--anytty-app-line)] bg-[var(--anytty-app-surface-soft)] p-1"
      data-testid="anytty-terminal-filter-bar"
      role="group"
    >
      {statusOptions.map(({ status: option, icon: Icon }) => {
        const selected = status === option
        return (
          <button
            aria-pressed={selected}
            className={`flex min-w-0 items-center justify-center gap-1 rounded px-1 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)] ${selected ? 'bg-[var(--anytty-app-surface)] text-zinc-900 shadow-sm' : 'text-zinc-500 hover:text-zinc-800'}`}
            data-testid={`anytty-terminal-status-${option}`}
            key={option}
            onClick={() => { hapticSelection(); onStatusChange(option) }}
            type="button"
          >
            <Icon aria-hidden="true" className="hidden h-3.5 w-3.5 shrink-0 2xl:block" />
            <span className="truncate">{t(`terminal.filters.${option}`)}</span>
            <span className={`tabular-nums ${selected ? 'text-zinc-500' : 'text-zinc-400'}`}>{statusCounts[option]}</span>
          </button>
        )
      })}

      {tagOptions.length === 0 || mobileTagSheet ? tagButton : (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>{tagButton}</DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="max-h-[min(24rem,70dvh)] w-72 overflow-y-auto">
            <DropdownMenuLabel className="flex items-center justify-between gap-3">
              <span>{t('terminal.filters.tagLabel')}</span>
              <span className="text-xs font-normal tabular-nums text-[var(--anytty-app-muted)]">
                {t('terminal.filters.results', { shown: filteredCount, total: statusCounts.all })}
              </span>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            {tagOptions.map((option) => (
              <DropdownMenuCheckboxItem
                checked={selectedTagIds.includes(option.id)}
                key={option.id}
                onCheckedChange={() => { hapticSelection(); onTagToggle(option.id) }}
                onSelect={(event) => event.preventDefault()}
              >
                <span className="min-w-0 flex-1 truncate" title={option.label}>{option.label}</span>
                <span className="ml-3 shrink-0 text-xs tabular-nums text-[var(--anytty-app-muted)]">{option.count}</span>
              </DropdownMenuCheckboxItem>
            ))}
            {selectedTagCount > 0 ? (
              <>
                <DropdownMenuSeparator />
                <DropdownMenuItem onSelect={() => { hapticSelection(); onClearTags() }}>
                  {t('terminal.filters.resetTags')}
                </DropdownMenuItem>
              </>
            ) : null}
          </DropdownMenuContent>
        </DropdownMenu>
      )}

      <span className="sr-only" aria-atomic="true" aria-live="polite">
        {t('terminal.filters.results', { shown: filteredCount, total: statusCounts.all })}
      </span>
    </div>
  )
}
