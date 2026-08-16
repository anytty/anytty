import { Settings } from 'lucide-react'
import { useMemo, useState } from 'react'
import { hapticImpact, hapticSelection } from '../platform/haptics'
import { matchTerminalFnPreset, SYSTEM_FN_GROUPS, type TerminalFnGroup } from './terminalFnPresets'
import { Button } from '../ui/button'

export interface TerminalFnPanelProps {
  command?: string | undefined
  onSend: (data: string) => void
}

export function TerminalFnPanel({ command, onSend }: TerminalFnPanelProps) {
  const programPreset = useMemo(() => matchTerminalFnPreset(command), [command])
  const [activeTab, setActiveTab] = useState<'program' | 'system'>(programPreset ? 'program' : 'system')
  const selectedTab = programPreset && activeTab === 'program' ? 'program' : 'system'
  const groups = selectedTab === 'program' ? programPreset?.groups ?? SYSTEM_FN_GROUPS : SYSTEM_FN_GROUPS

  return (
    <div className="absolute inset-x-0 top-0 z-30 overflow-hidden rounded-b-lg border-b border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface)] text-[var(--anytty-text)] shadow-sm md:hidden" data-testid="anytty-fn-panel">
      <div className="max-h-[36vh] overflow-y-auto px-2 py-1.5">
        <div className="mb-1.5 flex items-center gap-1.5">
          {programPreset ? (
            <Button variant="ghost"
              type="button"
              className={`min-h-11 rounded-md px-3 text-[10px] font-semibold ${selectedTab === 'program' ? 'bg-[var(--anytty-accent)] text-[var(--anytty-accent-text)]' : 'bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)]'}`}
              onClick={() => { hapticSelection(); setActiveTab('program') }}
            >
              {programPreset.name}
            </Button>
          ) : null}
          <Button variant="ghost"
            type="button"
            className={`min-h-11 rounded-md px-3 text-[10px] font-semibold ${selectedTab === 'system' ? 'bg-[var(--anytty-accent)] text-[var(--anytty-accent-text)]' : 'bg-[var(--anytty-surface-raised)] text-[var(--anytty-text)]'}`}
            onClick={() => { hapticSelection(); setActiveTab('system') }}
          >
            System
          </Button>
          <div className="flex-1" />
          <Settings className="h-3.5 w-3.5 text-[var(--anytty-muted)]" />
        </div>
        <div className="space-y-2">
          {groups.map((group) => (
            <FnGroupView key={group.name} group={group} onSend={onSend} />
          ))}
        </div>
      </div>
    </div>
  )
}

function FnGroupView({ group, onSend }: { group: TerminalFnGroup; onSend: (data: string) => void }) {
  return (
    <section>
      <h3 className="mb-1 px-0.5 text-[9px] font-bold uppercase tracking-wider text-[var(--anytty-muted)]">{group.name}</h3>
      <div className="grid grid-cols-4 gap-1">
        {group.items.map((item) => (
          <Button variant="ghost"
            key={`${group.name}:${item.label}:${item.data}`}
            type="button"
            className="min-h-11 rounded-md border border-[var(--anytty-border-subtle)] bg-[var(--anytty-surface-raised)] px-1.5 py-1 text-left active:opacity-80"
            onPointerDown={(event) => event.preventDefault()}
            onClick={() => { hapticImpact(); onSend(item.data) }}
          >
            <span className="block truncate font-mono text-[11px] font-semibold text-[var(--anytty-accent)]">{item.label}</span>
            {item.description ? <span className="mt-0.5 block truncate text-[8px] font-medium text-[var(--anytty-muted)]">{item.description}</span> : null}
          </Button>
        ))}
      </div>
    </section>
  )
}
