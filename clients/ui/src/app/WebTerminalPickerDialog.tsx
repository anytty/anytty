import { useEffect, useId, useMemo, useState, type KeyboardEvent, type ReactNode } from 'react'
import { Check, Plus, Search, SquareTerminal, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { Terminal } from '../core/model'
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '../ui/dialog'

export interface WebTerminalPickerDialogProps {
  open: boolean
  terminals: readonly Terminal[]
  activeTerminalId: string | null
  canCreateTerminal: boolean
  disabled: boolean
  onOpenChange: (open: boolean) => void
  onSelectTerminal: (terminalId: string) => void
  onCreateTerminal: () => void
}

type PickerItem =
  | { kind: 'create' }
  | { kind: 'terminal'; terminal: Terminal }

export function WebTerminalPickerDialog({
  open,
  terminals,
  activeTerminalId,
  canCreateTerminal,
  disabled,
  onOpenChange,
  onSelectTerminal,
  onCreateTerminal,
}: WebTerminalPickerDialogProps) {
  const { t } = useTranslation()
  const listId = useId()
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const terminalItems = useMemo(
    () => filterWebTerminalPickerTerminals(terminals, query),
    [query, terminals],
  )
  const showCreate = canCreateTerminal && pickerQueryMatchesAny(query, [
    t('workspace.newTerminal'),
    t('workspace.createTerminal'),
    'new terminal',
    'create terminal',
  ])
  const items = useMemo<PickerItem[]>(() => [
    ...(showCreate ? [{ kind: 'create' as const }] : []),
    ...terminalItems.map((terminal) => ({ kind: 'terminal' as const, terminal })),
  ], [showCreate, terminalItems])

  useEffect(() => {
    if (!open) return
    setQuery('')
    setSelectedIndex(0)
  }, [open])

  useEffect(() => {
    setSelectedIndex((current) => items.length === 0 ? 0 : Math.min(current, items.length - 1))
  }, [items.length])

  useEffect(() => {
    if (!open || items.length === 0) return
    document.getElementById(pickerOptionId(listId, selectedIndex))?.scrollIntoView?.({ block: 'nearest' })
  }, [items.length, listId, open, selectedIndex])

  const choose = (item: PickerItem | undefined) => {
    if (!item || disabled) return
    onOpenChange(false)
    if (item.kind === 'create') {
      onCreateTerminal()
      return
    }
    onSelectTerminal(item.terminal.terminalId)
  }

  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.nativeEvent.isComposing || items.length === 0) return
    if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
      event.preventDefault()
      const delta = event.key === 'ArrowDown' ? 1 : -1
      setSelectedIndex((current) => (current + delta + items.length) % items.length)
      return
    }
    if (event.key === 'Enter') {
      event.preventDefault()
      choose(items[selectedIndex])
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[min(38rem,calc(100dvh-3rem))] max-w-2xl grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden p-0" hideClose>
        <DialogHeader className="sr-only">
          <DialogTitle>{t('terminal.picker.title')}</DialogTitle>
          <DialogDescription>{t('terminal.picker.description')}</DialogDescription>
        </DialogHeader>

        <div className="relative flex h-14 items-center border-b border-[var(--anytty-app-line)] px-4 pr-14">
          <Search aria-hidden="true" className="mr-3 size-5 shrink-0 text-[var(--anytty-app-muted)]" />
          <input
            aria-activedescendant={items.length > 0 ? pickerOptionId(listId, selectedIndex) : undefined}
            aria-autocomplete="list"
            aria-controls={listId}
            aria-expanded="true"
            aria-label={t('terminal.picker.title')}
            autoComplete="off"
            autoFocus
            className="h-full min-w-0 flex-1 bg-transparent text-base text-[var(--anytty-app-text)] outline-none placeholder:text-[var(--anytty-app-muted)]"
            data-testid="anytty-web-terminal-picker-input"
            onChange={(event) => {
              setQuery(event.target.value)
              setSelectedIndex(0)
            }}
            onKeyDown={handleKeyDown}
            placeholder={t('terminal.picker.placeholder')}
            role="combobox"
            spellCheck={false}
            type="search"
            value={query}
          />
          <DialogClose
            aria-label={t('common.close')}
            className="absolute right-3 top-2.5 flex size-9 items-center justify-center rounded-md text-[var(--anytty-app-muted)] outline-none hover:bg-[var(--anytty-app-surface-soft)] hover:text-[var(--anytty-app-text)] focus-visible:ring-2 focus-visible:ring-[var(--anytty-app-accent)]"
            title={t('common.close')}
            type="button"
          >
            <X aria-hidden="true" className="size-4" />
          </DialogClose>
        </div>

        <div
          aria-label={t('terminal.list')}
          className="min-h-0 overflow-y-auto p-2"
          id={listId}
          role="listbox"
        >
          {items.length === 0 ? (
            <div className="flex min-h-28 items-center justify-center px-6 text-center text-sm text-[var(--anytty-app-muted)]" role="status">
              {t('terminal.picker.noResults')}
            </div>
          ) : items.map((item, index) => {
            const selected = index === selectedIndex
            if (item.kind === 'create') {
              return (
                <PickerOption
                  disabled={disabled}
                  id={pickerOptionId(listId, index)}
                  key="create"
                  selected={selected}
                  onChoose={() => choose(item)}
                  onSelect={() => setSelectedIndex(index)}
                >
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-md bg-[var(--anytty-app-surface-soft)] text-[var(--anytty-app-accent)]">
                    <Plus aria-hidden="true" className="size-5" />
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm font-semibold text-[var(--anytty-app-text)]">
                      <FuzzyText query={query} value={t('workspace.newTerminal')} />
                    </span>
                    <span className="block truncate text-xs text-[var(--anytty-app-muted)]">{t('terminal.picker.createDescription')}</span>
                  </span>
                </PickerOption>
              )
            }

            const terminal = item.terminal
            const title = terminal.title || terminal.command || t('terminal.defaultTitle')
            const detail = terminal.cwd || terminal.command || terminal.terminalId
            const current = terminal.terminalId === activeTerminalId
            const size = terminal.cols && terminal.rows ? `${terminal.cols}x${terminal.rows}` : null
            return (
              <PickerOption
                current={current}
                disabled={disabled}
                id={pickerOptionId(listId, index)}
                key={terminal.terminalId}
                selected={selected}
                onChoose={() => choose(item)}
                onSelect={() => setSelectedIndex(index)}
              >
                <span className="flex size-9 shrink-0 items-center justify-center text-[var(--anytty-app-muted)]">
                  <SquareTerminal aria-hidden="true" className="size-5" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="truncate text-sm font-semibold text-[var(--anytty-app-text)]">
                      <FuzzyText query={query} value={title} />
                    </span>
                    {current ? (
                      <span className="shrink-0 text-[var(--anytty-app-accent)]" title={t('terminal.picker.current')}>
                        <Check aria-hidden="true" className="size-4" />
                        <span className="sr-only">{t('terminal.picker.current')}</span>
                      </span>
                    ) : null}
                  </span>
                  <span className="block truncate font-mono text-[11px] text-[var(--anytty-app-muted)]">
                    <FuzzyText query={query} value={detail} />
                  </span>
                </span>
                <span className="hidden shrink-0 items-center gap-4 text-xs text-[var(--anytty-app-muted)] sm:flex">
                  {size ? <FuzzyText query={query} value={size} /> : null}
                  <span className="inline-flex min-w-20 items-center gap-1.5">
                    <span className={`size-2 rounded-full ${terminal.state === 'running' ? 'bg-emerald-500' : 'border border-[var(--anytty-app-muted)]'}`} />
                    {t(`terminal.state.${terminal.state}`)}
                  </span>
                </span>
              </PickerOption>
            )
          })}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function PickerOption({ children, current = false, disabled, id, selected, onChoose, onSelect }: {
  children: ReactNode
  current?: boolean
  disabled: boolean
  id: string
  selected: boolean
  onChoose: () => void
  onSelect: () => void
}) {
  return (
    <button
      aria-current={current ? 'true' : undefined}
      aria-selected={selected}
      className="flex min-h-12 w-full cursor-pointer items-center gap-3 rounded-md px-3 py-2 text-left outline-none transition-colors aria-selected:bg-[var(--anytty-app-surface-soft)] focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-[var(--anytty-app-accent)] disabled:cursor-not-allowed disabled:opacity-50"
      disabled={disabled}
      id={id}
      onClick={onChoose}
      onPointerMove={onSelect}
      role="option"
      tabIndex={-1}
      type="button"
    >
      {children}
    </button>
  )
}

function FuzzyText({ value, query }: { value: string; query: string }) {
  const matches = terminalPickerMatchIndexes(value, query)
  if (!matches || matches.length === 0) return value
  const matchSet = new Set(matches)
  return (
    <span aria-label={value}>
      <span aria-hidden="true">
        {Array.from(value).map((character, index) => matchSet.has(index)
          ? <span className="font-semibold text-[var(--anytty-app-accent)]" key={index}>{character}</span>
          : <span key={index}>{character}</span>)}
      </span>
    </span>
  )
}

export function filterWebTerminalPickerTerminals(terminals: readonly Terminal[], query: string): Terminal[] {
  return terminals
    .filter((terminal) => pickerQueryMatchesAny(query, terminalPickerSearchValues(terminal)))
    .sort((left, right) => {
      const leftTitle = left.title || left.command || left.terminalId
      const rightTitle = right.title || right.command || right.terminalId
      return leftTitle.localeCompare(rightTitle, undefined, { numeric: true, sensitivity: 'base' })
        || left.terminalId.localeCompare(right.terminalId, undefined, { numeric: true, sensitivity: 'base' })
    })
}

export function terminalPickerMatchIndexes(value: string, query: string): number[] | null {
  const queryRunes = Array.from(query.trim().toLocaleLowerCase())
  if (queryRunes.length === 0) return []
  const valueRunes = Array.from(value.toLocaleLowerCase())
  const matches: number[] = []
  let valueIndex = 0
  for (const queryRune of queryRunes) {
    while (valueIndex < valueRunes.length && valueRunes[valueIndex] !== queryRune) valueIndex += 1
    if (valueIndex >= valueRunes.length) return null
    matches.push(valueIndex)
    valueIndex += 1
  }
  return matches
}

function pickerQueryMatchesAny(query: string, values: readonly string[]): boolean {
  if (query.trim() === '') return true
  return values.some((value) => terminalPickerMatchIndexes(value, query) !== null)
}

function terminalPickerSearchValues(terminal: Terminal): string[] {
  return [
    terminal.title,
    terminal.terminalId,
    terminal.state,
    terminal.command ?? '',
    terminal.cwd ?? '',
    terminal.foregroundProcess ?? '',
    terminal.cols && terminal.rows ? `${terminal.cols}x${terminal.rows}` : '',
  ]
}

function pickerOptionId(listId: string, index: number): string {
  return `${listId}-option-${index}`
}
