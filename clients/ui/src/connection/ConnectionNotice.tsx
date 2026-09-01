import { Info, type LucideIcon } from 'lucide-react'
import type { HTMLAttributes } from 'react'
import { Alert, AlertDescription, AlertTitle } from '../ui/alert'
import { Button, type ButtonVariant } from '../ui/button'
import { Spinner } from '../ui/spinner'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../ui/tooltip'
import { cn } from '../ui/utils'
import type { ConnectionPresentation } from './connectionPresentation'
import {
  connectionPresentationIcon,
  connectionPresentationIsBusy,
  connectionToneIconClass,
} from './ConnectionSummary'

export type ConnectionNoticeVariant = 'notice' | 'gate'

export interface ConnectionNoticeAction {
  label: string
  onClick: () => void
  disabled?: boolean | undefined
  pending?: boolean | undefined
  testId?: string | undefined
  variant?: ButtonVariant | undefined
}

export interface ConnectionNoticeDetailsAction {
  label: string
  onClick: () => void
  tooltip?: string | undefined
}

export interface ConnectionNoticeProps extends Omit<HTMLAttributes<HTMLDivElement>, 'children' | 'title'> {
  presentation: ConnectionPresentation
  title: string
  description?: string | undefined
  variant?: ConnectionNoticeVariant | undefined
  /** Existing terminal or file content keeps recovery feedback in the non-blocking notice density. */
  hasContent?: boolean | undefined
  primaryAction?: ConnectionNoticeAction | undefined
  secondaryAction?: ConnectionNoticeAction | undefined
  detailsAction?: ConnectionNoticeDetailsAction | undefined
}

/** Inline connection feedback. It never owns a viewport overlay or a modal focus boundary. */
export function ConnectionNotice({
  presentation,
  title,
  description,
  variant = 'notice',
  hasContent = false,
  primaryAction,
  secondaryAction,
  detailsAction,
  className,
  ...props
}: ConnectionNoticeProps) {
  const effectiveVariant: ConnectionNoticeVariant = variant === 'gate' && hasContent ? 'notice' : variant
  const Icon = connectionPresentationIcon(presentation)
  const busy = connectionPresentationIsBusy(presentation)
  const spinBusy = busy && presentation.state === 'connecting'
  const critical = presentation.tone === 'critical'

  return (
    <Alert
      aria-live={critical ? 'assertive' : 'polite'}
      className={cn(
        'text-[var(--foreground)]',
        connectionToneContainerClass[presentation.tone],
        effectiveVariant === 'notice'
          ? 'rounded-none border-x-0 px-3 py-2.5'
          : 'mx-auto max-w-md rounded-lg border bg-[var(--card)] px-5 py-8 text-center shadow-sm',
        className,
      )}
      data-action={presentation.action}
      data-connection-state={presentation.state}
      data-observed-path={presentation.observedPath}
      data-requested-variant={variant}
      data-route={presentation.route}
      data-tone={presentation.tone}
      data-variant={effectiveVariant}
      role={critical ? 'alert' : 'status'}
      variant={critical ? 'destructive' : 'default'}
      {...props}
    >
      <div className={cn(
        'flex min-w-0 gap-3',
        effectiveVariant === 'notice' ? 'items-start sm:items-center' : 'flex-col items-center',
      )}>
        <NoticeIcon Icon={Icon} busy={busy} spinBusy={spinBusy} presentation={presentation} variant={effectiveVariant} />

        <div className={cn('min-w-0 flex-1', effectiveVariant === 'gate' && 'w-full')}>
          <AlertTitle className={cn(
            'break-words text-sm leading-5',
            effectiveVariant === 'gate' && 'text-base leading-6',
          )}>
            {title}
          </AlertTitle>
          {description ? (
            <AlertDescription className={cn(
              'mt-0.5 break-words',
              effectiveVariant === 'notice' ? 'text-xs leading-4' : 'mx-auto mt-2 max-w-sm text-sm leading-6',
            )}>
              {description}
            </AlertDescription>
          ) : null}
        </div>

        <div className={cn(
          'flex shrink-0 items-center gap-2',
          effectiveVariant === 'notice'
            ? 'grid w-full grid-cols-1 pl-10 min-[360px]:w-auto min-[360px]:grid-cols-2 min-[360px]:pl-0'
            : 'mt-3 w-full max-w-sm flex-col sm:flex-row sm:justify-center',
        )}>
          {secondaryAction ? <NoticeAction action={secondaryAction} defaultVariant="secondary" /> : null}
          {primaryAction ? <NoticeAction action={primaryAction} defaultVariant="default" /> : null}
          {detailsAction ? <DetailsAction action={detailsAction} /> : null}
        </div>
      </div>
    </Alert>
  )
}

function NoticeIcon({
  Icon,
  busy,
  spinBusy,
  presentation,
  variant,
}: {
  Icon: LucideIcon
  busy: boolean
  spinBusy: boolean
  presentation: ConnectionPresentation
  variant: ConnectionNoticeVariant
}) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        'relative grid shrink-0 place-items-center rounded-md border border-[var(--border)] bg-[var(--muted)]',
        variant === 'notice' ? 'size-8' : 'size-11',
        connectionToneIconClass[presentation.tone],
        busy && !spinBusy && 'animate-pulse motion-reduce:animate-none',
      )}
    >
      <Icon className={cn(variant === 'notice' ? 'size-4' : 'size-5', spinBusy && 'animate-spin motion-reduce:animate-none')} />
    </span>
  )
}

function NoticeAction({
  action,
  defaultVariant,
}: {
  action: ConnectionNoticeAction
  defaultVariant: ButtonVariant
}) {
  return (
    <Button
      aria-busy={action.pending || undefined}
      className="min-h-11 min-w-11 px-3 text-xs sm:w-auto"
      data-testid={action.testId}
      disabled={action.disabled || action.pending}
      onClick={action.onClick}
      variant={action.variant ?? defaultVariant}
    >
      {action.pending ? <Spinner aria-hidden="true" /> : null}
      {action.label}
    </Button>
  )
}

function DetailsAction({ action }: { action: ConnectionNoticeDetailsAction }) {
  return (
    <TooltipProvider delayDuration={400}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button aria-label={action.label} className="min-h-11 min-w-11" onClick={action.onClick} size="icon" variant="ghost">
            <Info className="size-4" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>{action.tooltip ?? action.label}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

const connectionToneContainerClass: Record<ConnectionPresentation['tone'], string> = {
  neutral: 'border-[var(--border)] bg-[var(--card)]',
  info: 'border-[var(--anytty-app-line-strong)] bg-[var(--muted)]',
  positive: 'border-[var(--anytty-app-success)] bg-[var(--card)]',
  warning: 'border-[var(--anytty-app-line-strong)] bg-[var(--card)]',
  critical: 'border-[var(--destructive)] bg-[var(--card)]',
}
