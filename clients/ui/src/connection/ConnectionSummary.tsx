import {
  CircleAlert,
  CircleCheck,
  CircleHelp,
  KeyRound,
  LoaderCircle,
  RadioTower,
  WifiOff,
  type LucideIcon,
} from 'lucide-react'
import type { HTMLAttributes } from 'react'
import { cn } from '../ui/utils'
import type { ConnectionPresentation } from './connectionPresentation'

export interface ConnectionSummaryProps extends Omit<HTMLAttributes<HTMLDivElement>, 'children'> {
  presentation: ConnectionPresentation
  label: string
  detail?: string | undefined
  icon?: LucideIcon | undefined
  variant?: 'compact' | undefined
  announce?: boolean | undefined
}

/** Compact, non-interactive connection state for device rows and workspace headers. */
export function ConnectionSummary({
  presentation,
  label,
  detail,
  icon,
  variant = 'compact',
  announce = false,
  className,
  ...props
}: ConnectionSummaryProps) {
  const Icon = icon ?? connectionPresentationIcon(presentation)
  const busy = connectionPresentationIsBusy(presentation)
  const spinBusy = busy && presentation.state === 'connecting'

  return (
    <div
      aria-live={announce ? 'polite' : undefined}
      aria-busy={busy || undefined}
      className={cn(
        'inline-flex min-w-0 items-center gap-2 text-sm text-[var(--foreground)]',
        className,
      )}
      data-action={presentation.action}
      data-connection-state={presentation.state}
      data-density={variant}
      data-observed-path={presentation.observedPath}
      data-route={presentation.route}
      data-tone={presentation.tone}
      role={announce ? 'status' : undefined}
      {...props}
    >
      <span
        aria-hidden="true"
        className={cn(
          'relative grid size-7 shrink-0 place-items-center rounded-md border border-[var(--border)] bg-[var(--muted)]',
          connectionToneIconClass[presentation.tone],
          busy && !spinBusy && 'animate-pulse motion-reduce:animate-none',
        )}
      >
        <Icon className={cn('size-4', spinBusy && 'animate-spin motion-reduce:animate-none')} />
      </span>
      <span className="min-w-0 truncate font-medium">{label}</span>
      {detail ? (
        <span className="min-w-0 truncate text-xs text-[var(--muted-foreground)]">{detail}</span>
      ) : null}
    </div>
  )
}

export function connectionPresentationIcon(presentation: ConnectionPresentation): LucideIcon {
  switch (presentation.state) {
    case 'phone_offline':
    case 'waiting_network':
      return WifiOff
    case 'auth_unavailable':
      return KeyRound
    case 'connecting':
      return LoaderCircle
    case 'ready':
      return CircleCheck
    case 'failed':
      return CircleAlert
    case 'idle':
      if (presentation.reachability === 'checking') return LoaderCircle
      if (presentation.reachability === 'reachable') return RadioTower
      if (presentation.reachability === 'unreachable') return WifiOff
      return CircleHelp
    default:
      return unreachablePresentationState(presentation.state)
  }
}

export function connectionPresentationIsBusy(presentation: ConnectionPresentation): boolean {
  return presentation.state === 'connecting' ||
    (presentation.state === 'idle' && presentation.reachability === 'checking')
}

export const connectionToneIconClass: Record<ConnectionPresentation['tone'], string> = {
  neutral: 'text-[var(--muted-foreground)]',
  info: 'text-[var(--anytty-app-accent)]',
  positive: 'text-[var(--anytty-app-success)]',
  warning: 'text-[var(--foreground)]',
  critical: 'text-[var(--destructive)]',
}

function unreachablePresentationState(state: never): never {
  throw new Error(`Unsupported connection presentation state: ${String(state)}`)
}
